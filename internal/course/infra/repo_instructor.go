package infra

import (
	"context"
	stderrors "errors"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/internal/course/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/gormx"
	"mycourse-io-be/internal/shared/timex"
)

func (r *GormRepository) ListEditableCourses(ctx context.Context, userID uint) ([]domain.CourseListItem, error) {
	q := `
SELECT
    ` + courseListBaseColumns + `,
    CASE WHEN c.owner_user_id = @user_id THEN 'OWNER' ELSE cc.role END AS role,
    COALESCE(dv.title, pv.title, '') AS title,
    COALESCE(dv.status, pv.status, '') AS review_status,
    COALESCE(dv.version_no, pv.version_no, 0) AS version_no,
    (c.current_published_version_id IS NOT NULL) AS has_published,
    (c.current_draft_version_id IS NOT NULL) AS has_draft,
    COALESCE(dv.thumbnail_file_id::text, pv.thumbnail_file_id::text, '') AS thumbnail_file_id,
    COALESCE(dm.url, pm.url, '') AS thumbnail_url,
    COALESCE(dv.preview_video_file_id::text, pv.preview_video_file_id::text, '') AS preview_video_file_id
FROM courses c
LEFT JOIN course_collaborators cc
    ON cc.course_id = c.id AND cc.user_id = @user_id AND cc.deleted_at IS NULL
LEFT JOIN course_versions dv
    ON dv.id = c.current_draft_version_id AND dv.deleted_at IS NULL
LEFT JOIN course_versions pv
    ON pv.id = c.current_published_version_id AND pv.deleted_at IS NULL
LEFT JOIN media_files dm
    ON dm.id = dv.thumbnail_file_id AND dm.deleted_at IS NULL
LEFT JOIN media_files pm
    ON pm.id = pv.thumbnail_file_id AND pm.deleted_at IS NULL
WHERE c.deleted_at IS NULL
  AND (c.owner_user_id = @user_id OR cc.id IS NOT NULL)
ORDER BY c.id DESC`

	var rows []courseListScanRow
	if err := r.db.WithContext(ctx).Raw(q, map[string]any{"user_id": userID}).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CourseListItem, len(rows))
	for i, row := range rows {
		out[i] = toCourseListItem(&row)
	}
	return out, nil
}

func (r *GormRepository) CreateCourse(ctx context.Context, in domain.CreateCourseInput) (*domain.CourseDetail, error) {
	var detail *domain.CourseDetail
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		course := &courseRow{OwnerUserID: in.ActorUserID, Slug: strings.TrimSpace(in.Slug)}
		gormx.TouchCreatedUpdated(&course.CreatedAt, &course.UpdatedAt)
		if err := tx.Create(course).Error; err != nil {
			return err
		}

		version := &courseVersionRow{
			CourseID: course.ID, VersionNo: 1, Status: domain.VersionStatusDraft,
			Title: strings.TrimSpace(in.Title), RowVersion: 1,
		}
		gormx.TouchCreatedUpdated(&version.CreatedAt, &version.UpdatedAt)
		if err := tx.Create(version).Error; err != nil {
			return err
		}
		if err := tx.Model(&courseRow{}).Where("id = ?", course.ID).
			Updates(map[string]any{"current_draft_version_id": version.ID, "updated_at": timex.NowUnix()}).Error; err != nil {
			return err
		}

		collab := &collaboratorRow{CourseID: course.ID, UserID: in.ActorUserID, Role: domain.CollaboratorRoleOwner}
		gormx.TouchCreatedUpdated(&collab.CreatedAt, &collab.UpdatedAt)
		if err := tx.Create(collab).Error; err != nil {
			return err
		}
		var err error
		detail, err = r.loadCourseDetail(ctx, tx, course.ID, in.ActorUserID, true)
		return err
	})
	return detail, err
}

func (r *GormRepository) GetCourseDetail(ctx context.Context, courseID, userID uint, includeDraft bool) (*domain.CourseDetail, error) {
	return r.loadCourseDetail(ctx, r.db.WithContext(ctx), courseID, userID, includeDraft)
}

func (r *GormRepository) PrepareDraft(ctx context.Context, courseID, actorUserID uint) (*domain.CourseDetail, error) {
	var detail *domain.CourseDetail
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.requireEditorAccess(ctx, tx, courseID, actorUserID)
		if err != nil {
			return err
		}
		if access.CurrentDraftVersionID == nil {
			draftID, err := r.createDraftVersion(ctx, tx, &access.courseRow)
			if err != nil {
				return err
			}
			if err := tx.Model(&courseRow{}).Where("id = ?", courseID).
				Updates(map[string]any{"current_draft_version_id": draftID, "updated_at": timex.NowUnix()}).Error; err != nil {
				return err
			}
		}
		detail, err = r.loadCourseDetail(ctx, tx, courseID, actorUserID, true)
		return err
	})
	return detail, err
}

func (r *GormRepository) UpdateBasicInfo(ctx context.Context, courseID, actorUserID uint, in domain.UpdateBasicInfoInput) (*domain.CourseDetail, error) {
	var detail *domain.CourseDetail
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.ensureEditableDraft(ctx, tx, courseID, actorUserID)
		if err != nil {
			return err
		}
		version, err := r.loadVersionRow(ctx, tx, *access.CurrentDraftVersionID)
		if err != nil {
			return err
		}
		if version.RowVersion != in.ExpectedRowVersion {
			return apperrors.ErrMediaOptimisticLock
		}
		if err := r.validateVersionRefs(ctx, tx, versionRefValidationInput{
			ThumbnailFileID:    in.ThumbnailFileID,
			PreviewVideoFileID: in.PreviewVideoFileID,
			CourseLevelID:      in.CourseLevelID,
			CourseTopicID:      in.CourseTopicID,
			TagIDs:             in.TagIDs,
			SkillIDs:           in.SkillIDs,
			OutcomeIDs:         in.OutcomeIDs,
		}); err != nil {
			return err
		}
		updates := buildBasicInfoUpdates(in)
		if err := tx.Model(&courseVersionRow{}).
			Where("id = ? AND row_version = ? AND deleted_at IS NULL", version.ID, in.ExpectedRowVersion).
			Updates(updates).Error; err != nil {
			return err
		}
		if err := r.replaceVersionRefs(ctx, tx, version.ID, in.TagIDs, in.SkillIDs, in.OutcomeIDs); err != nil {
			return err
		}
		detail, err = r.loadCourseDetail(ctx, tx, courseID, actorUserID, true)
		return err
	})
	if stderrors.Is(err, apperrors.ErrMediaOptimisticLock) {
		return nil, domain.ErrCourseOptimisticLock
	}
	return detail, err
}

func (r *GormRepository) DeleteCourse(ctx context.Context, courseID, actorUserID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.requireOwnerAccess(ctx, tx, courseID, actorUserID)
		if err != nil {
			return err
		}
		now := timex.NowUnix()
		for _, model := range []any{&courseRow{}, &courseVersionRow{}, &collaboratorRow{}, &enrollmentRow{}, &progressRow{}} {
			switch model.(type) {
			case *courseRow:
				if err := tx.Model(model).Where("id = ? AND deleted_at IS NULL", access.ID).Updates(map[string]any{"deleted_at": now, "updated_at": now}).Error; err != nil {
					return err
				}
			case *courseVersionRow:
				if err := tx.Model(model).Where("course_id = ? AND deleted_at IS NULL", access.ID).Updates(map[string]any{"deleted_at": now, "updated_at": now}).Error; err != nil {
					return err
				}
			case *collaboratorRow:
				if err := tx.Model(model).Where("course_id = ? AND deleted_at IS NULL", access.ID).Updates(map[string]any{"deleted_at": now, "updated_at": now}).Error; err != nil {
					return err
				}
			case *enrollmentRow:
				if err := tx.Model(model).Where("course_id = ? AND deleted_at IS NULL", access.ID).Updates(map[string]any{"deleted_at": now, "updated_at": now}).Error; err != nil {
					return err
				}
			case *progressRow:
				if err := tx.Exec(`
UPDATE course_progress_items
SET deleted_at = ?, updated_at = ?
WHERE deleted_at IS NULL
  AND enrollment_id IN (
      SELECT id FROM course_enrollments
      WHERE course_id = ? AND deleted_at IS NULL
  )`, now, now, access.ID).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *GormRepository) ListCollaborators(ctx context.Context, courseID, actorUserID uint) ([]domain.Collaborator, error) {
	if _, err := r.requireCourseAccess(ctx, r.db.WithContext(ctx), courseID, actorUserID); err != nil {
		return nil, err
	}
	return r.loadCollaborators(ctx, r.db.WithContext(ctx), courseID)
}

func (r *GormRepository) AddCollaborator(ctx context.Context, courseID, actorUserID, userID uint, role string) ([]domain.Collaborator, error) {
	var out []domain.Collaborator
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.requireOwnerAccess(ctx, tx, courseID, actorUserID)
		if err != nil {
			return err
		}
		if !r.userIsInstructor(ctx, tx, userID) {
			return domain.ErrCourseInstructorRequired
		}
		var existing collaboratorRow
		err = tx.Where("course_id = ? AND user_id = ? AND deleted_at IS NULL", access.ID, userID).First(&existing).Error
		if err == nil {
			if err := tx.Model(&collaboratorRow{}).Where("id = ?", existing.ID).Updates(map[string]any{
				"role":       strings.ToUpper(strings.TrimSpace(role)),
				"updated_at": timex.NowUnix(),
			}).Error; err != nil {
				return err
			}
		} else if !stderrors.Is(err, gorm.ErrRecordNotFound) {
			return err
		} else {
			row := &collaboratorRow{CourseID: access.ID, UserID: userID, Role: strings.ToUpper(strings.TrimSpace(role))}
			gormx.TouchCreatedUpdated(&row.CreatedAt, &row.UpdatedAt)
			if row.Role == "" {
				row.Role = domain.CollaboratorRoleEditor
			}
			if err := tx.Create(row).Error; err != nil {
				return err
			}
		}
		out, err = r.loadCollaborators(ctx, tx, courseID)
		return err
	})
	return out, err
}

func (r *GormRepository) RemoveCollaborator(ctx context.Context, courseID, actorUserID, userID uint) ([]domain.Collaborator, error) {
	var out []domain.Collaborator
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.requireOwnerAccess(ctx, tx, courseID, actorUserID)
		if err != nil {
			return err
		}
		if access.OwnerUserID == userID {
			return domain.ErrCourseOwnerCannotBeRemoved
		}
		if err := gormx.SoftDeleteWithAudit(ctx, tx, &collaboratorRow{}, "course_id = ? AND user_id = ? AND deleted_at IS NULL", access.ID, userID); err != nil {
			return err
		}
		out, err = r.loadCollaborators(ctx, tx, courseID)
		return err
	})
	return out, err
}
