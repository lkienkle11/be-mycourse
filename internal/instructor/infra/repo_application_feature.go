package infra

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"mycourse-io-be/internal/instructor/domain"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/gormx"
	"mycourse-io-be/internal/shared/timex"
)

const applicationSLASeconds = domain.ApplicationSLADays * 24 * 60 * 60

func (r *GormRepository) CreateFirstApplication(ctx context.Context, userID string, in domain.SubmitApplicationInput) (*domain.Application, error) {
	var count int64
	if err := activeScope(r.db.WithContext(ctx)).Model(&applicationRow{}).
		Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, domain.ErrApplicationAlreadyExists
	}
	return r.saveApplication(ctx, userID, in, true)
}

func (r *GormRepository) ResubmitApplication(ctx context.Context, userID string, in domain.SubmitApplicationInput) (*domain.Application, error) {
	var existing applicationRow
	err := activeScope(r.db.WithContext(ctx)).Where("user_id = ?", userID).First(&existing).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	if existing.ReviewStatus != domain.ReviewStatusReturned && existing.ReviewStatus != domain.ReviewStatusRejected {
		return nil, domain.ErrApplicationNotResubmittable
	}
	if existing.ReviewStatus == domain.ReviewStatusRejected && existing.RejectionCount >= domain.MaxApplicationRejections {
		return nil, domain.ErrApplicationRejectQuota
	}
	return r.saveApplication(ctx, userID, in, false)
}

func (r *GormRepository) saveApplication(ctx context.Context, userID string, in domain.SubmitApplicationInput, isCreate bool) (*domain.Application, error) {
	var result *domain.Application
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row applicationRow
		if !isCreate {
			if err := activeScope(tx).Where("user_id = ?", userID).First(&row).Error; err != nil {
				return mapNotFound(err)
			}
		} else {
			row = applicationRow{UserID: userID}
		}
		if err := applyPayloadToAppRow(&row, in.ProfilePayload); err != nil {
			return err
		}
		now := timex.NowUnix()
		row.ReviewStatus = domain.ReviewStatusPending
		row.RejectionReason = ""
		row.SubmittedAt = now
		row.ReviewDueAt = now + applicationSLASeconds
		row.ReturnedAt = nil
		ensureApplicationRowDefaults(&row)
		gormx.TouchUpdated(&row.UpdatedAt)
		if isCreate {
			gormx.TouchCreatedUpdated(&row.CreatedAt, &row.UpdatedAt)
			if err := touchAndCreate(ctx, tx, &row.CreatedAt, &row.UpdatedAt, &row); err != nil {
				return err
			}
		} else if err := tx.Save(&row).Error; err != nil {
			return err
		}
		if err := replaceApplicationJunctionDB(tx, row.ID, in.TopicIDs, true); err != nil {
			return err
		}
		if err := replaceApplicationJunctionDB(tx, row.ID, in.SkillIDs, false); err != nil {
			return err
		}
		loaded, err := loadApplicationRow(ctx, tx, "ia.user_id = ?", userID)
		if err != nil {
			return err
		}
		result = loaded
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func ensureApplicationRowDefaults(row *applicationRow) {
	if row.RejectionHistory == nil {
		row.RejectionHistory = emptyRejectionHistoryPtr()
	}
}

func (r *GormRepository) MarkReturnedIfDue(ctx context.Context, userID string) error {
	now := timex.NowUnix()
	return r.db.WithContext(ctx).Model(&applicationRow{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Where("review_status = ?", domain.ReviewStatusPending).
		Where("review_due_at > 0 AND review_due_at <= ?", now).
		Updates(map[string]any{
			"review_status": domain.ReviewStatusReturned,
			"returned_at":   now,
			"updated_at":    now,
		}).Error
}

func (r *GormRepository) RejectApplicationWithHistory(ctx context.Context, in domain.RejectApplicationInput) error {
	app, err := r.GetApplicationByID(ctx, in.ApplicationID)
	if err != nil {
		return err
	}
	if app.ReviewStatus != domain.ReviewStatusPending {
		return domain.ErrApplicationNotPending
	}
	now := timex.NowUnix()
	record := rejectionHistoryJSON{
		RejectedAt: now, RejectedByUserID: in.ReviewerUserID,
		ReviewerDisplayName: in.ReviewerDisplayName, Reason: in.RejectionReason,
	}
	var row applicationRow
	if err := activeScope(r.db.WithContext(ctx)).Where("id = ?", in.ApplicationID).First(&row).Error; err != nil {
		return mapNotFound(err)
	}
	history := rejectionHistoryFromJSON(row.RejectionHistory)
	history = append(history, domain.RejectionRecord{
		RejectedAt: record.RejectedAt, RejectedByUserID: record.RejectedByUserID,
		ReviewerDisplayName: record.ReviewerDisplayName, Reason: record.Reason,
	})
	hJSON := rejectionHistoryToJSON(history)
	return r.db.WithContext(ctx).Model(&applicationRow{}).
		Where("id = ? AND deleted_at IS NULL", in.ApplicationID).
		Updates(map[string]any{
			"review_status":     domain.ReviewStatusRejected,
			"rejection_reason":  in.RejectionReason,
			"rejection_count":   gorm.Expr("rejection_count + 1"),
			"rejection_history": hJSON,
			"updated_at":        now,
		}).Error
}

func (r *GormRepository) ApproveApplicationCopySnapshot(ctx context.Context, appID, userID string) error {
	app, err := r.GetApplicationByID(ctx, appID)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := upsertProfileFromApplication(ctx, tx, userID, app.ProfilePayload); err != nil {
			return err
		}
		topicIDs, err := pluckApplicationRefIDsDB(tx, appID, true)
		if err != nil {
			return err
		}
		skillIDs, err := pluckApplicationRefIDsDB(tx, appID, false)
		if err != nil {
			return err
		}
		if err := copyExpertiseFromApplication(tx, userID, topicIDs, skillIDs); err != nil {
			return err
		}
		now := timex.NowUnix()
		return tx.Model(&applicationRow{}).Where("id = ? AND deleted_at IS NULL", appID).
			Updates(map[string]any{
				"review_status":    domain.ReviewStatusApproved,
				"rejection_reason": "",
				"updated_at":       now,
			}).Error
	})
}

func upsertProfileFromApplication(ctx context.Context, tx *gorm.DB, userID string, payload domain.ProfilePayload) error {
	var profile profileRow
	err := activeScope(tx).Where("user_id = ?", userID).First(&profile).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == gorm.ErrRecordNotFound {
		profile = profileRow{UserID: userID}
	}
	if e := applyPayloadToProfileRow(&profile, payload); e != nil {
		return e
	}
	gormx.TouchUpdated(&profile.UpdatedAt)
	if profile.ID == "" {
		gormx.TouchCreatedUpdated(&profile.CreatedAt, &profile.UpdatedAt)
		return touchAndCreate(ctx, tx, &profile.CreatedAt, &profile.UpdatedAt, &profile)
	}
	return tx.Save(&profile).Error
}

func copyExpertiseFromApplication(tx *gorm.DB, userID string, topicIDs, skillIDs []string) error {
	if err := softDeleteAllExpertise(tx, userID); err != nil {
		return err
	}
	now := timex.NowUnix()
	for _, topicID := range topicIDs {
		row := expertiseTopicRow{ID: uuid.NewString(), UserID: userID, TopicID: topicID, CreatedAt: now, UpdatedAt: now}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	for _, skillID := range skillIDs {
		row := expertiseSkillRow{ID: uuid.NewString(), UserID: userID, SkillID: skillID, CreatedAt: now, UpdatedAt: now}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func softDeleteAllExpertise(tx *gorm.DB, userID string) error {
	now := timex.NowUnix()
	if err := tx.Model(&expertiseTopicRow{}).Where("user_id = ? AND deleted_at IS NULL", userID).
		Update("deleted_at", now).Error; err != nil {
		return err
	}
	return tx.Model(&expertiseSkillRow{}).Where("user_id = ? AND deleted_at IS NULL", userID).
		Update("deleted_at", now).Error
}

func (r *GormRepository) ListApplicationTopicIDs(ctx context.Context, appID string) ([]string, error) {
	return r.pluckApplicationRefIDs(ctx, appID, true)
}

func (r *GormRepository) ListApplicationSkillIDs(ctx context.Context, appID string) ([]string, error) {
	return r.pluckApplicationRefIDs(ctx, appID, false)
}

func (r *GormRepository) pluckApplicationRefIDs(ctx context.Context, appID string, isTopic bool) ([]string, error) {
	return pluckApplicationRefIDsDB(r.db.WithContext(ctx), appID, isTopic)
}

func pluckApplicationRefIDsDB(db *gorm.DB, appID string, isTopic bool) ([]string, error) {
	var ids []string
	q := activeScope(db)
	if isTopic {
		err := q.Model(&applicationTopicRow{}).Where("application_id = ?", appID).Pluck("topic_id", &ids).Error
		return ids, err
	}
	err := q.Model(&applicationSkillRow{}).Where("application_id = ?", appID).Pluck("skill_id", &ids).Error
	return ids, err
}

func (r *GormRepository) ListApplicationTopics(ctx context.Context, appID string, locale string) ([]domain.ApplicationTaxonomyChip, error) {
	return r.listApplicationTaxonomyChips(ctx, appID, true, locale)
}

func (r *GormRepository) ListApplicationSkills(ctx context.Context, appID string, locale string) ([]domain.ApplicationTaxonomyChip, error) {
	return r.listApplicationTaxonomyChips(ctx, appID, false, locale)
}

func (r *GormRepository) listApplicationTaxonomyChips(ctx context.Context, appID string, isTopic bool, locale string) ([]domain.ApplicationTaxonomyChip, error) {
	type row struct {
		RefID string `gorm:"column:ref_id"`
		Name  string `gorm:"column:name"`
		Slug  string `gorm:"column:slug"`
	}
	exact, base := taxonomyLocaleJoinArgs(locale)
	alias, table, joinTable, refCol := "ias", constants.TableInstructorApplicationSkills, constants.TableTaxonomyCourseSkills, "skill_id"
	rootAlias := "cs"
	if isTopic {
		alias, table, joinTable, refCol = "iat", constants.TableInstructorApplicationTopics, constants.TableTaxonomyCourseTopics, "topic_id"
		rootAlias = "ct"
	}
	baseJoin := "LEFT JOIN " + joinTable + " " + rootAlias + " ON " + rootAlias + ".id = " + alias + "." + refCol + " AND " + rootAlias + ".deleted_at IS NULL"
	var joins string
	if isTopic {
		joins = joinLocalizedTopicName(baseJoin)
	} else {
		joins = joinLocalizedSkillName(baseJoin)
	}
	var rows []row
	err := activeScopeAlias(r.db.WithContext(ctx), alias).Table(table+" "+alias).
		Select(alias+"."+refCol+" AS ref_id, "+localizedNameSelect(rootAlias)+", COALESCE("+rootAlias+".slug, '') AS slug").
		Joins(joins, exact, base).
		Where(alias+".application_id = ?", appID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.ApplicationTaxonomyChip, len(rows))
	for i, item := range rows {
		out[i] = domain.ApplicationTaxonomyChip{ID: item.RefID, Name: item.Name, Slug: item.Slug}
	}
	return out, nil
}

func replaceApplicationJunctionDB(tx *gorm.DB, appID string, refIDs []string, isTopic bool) error {
	table := constants.TableInstructorApplicationSkills
	col := "skill_id"
	if isTopic {
		table = constants.TableInstructorApplicationTopics
		col = "topic_id"
	}
	now := timex.NowUnix()
	if err := tx.Table(table).Where("application_id = ? AND deleted_at IS NULL", appID).
		Update("deleted_at", now).Error; err != nil {
		return err
	}
	for _, refID := range refIDs {
		if refID == "" {
			continue
		}
		id := uuid.NewString()
		row := map[string]any{
			"id": id, "application_id": appID, col: refID,
			"created_at": now, "updated_at": now,
		}
		if err := tx.Table(table).Create(row).Error; err != nil {
			return fmt.Errorf("insert %s: %w", table, err)
		}
	}
	return nil
}
