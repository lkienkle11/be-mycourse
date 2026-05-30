// Package infra provides GORM persistence for the instructor bounded context.
package infra

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/internal/instructor/domain"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/gormx"
	"mycourse-io-be/internal/shared/timex"
)

// GormRepository implements instructor domain repositories.
type GormRepository struct{ db *gorm.DB }

func NewGormRepository(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

func instrPageParams(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	return page, pageSize
}

func activeScope(db *gorm.DB) *gorm.DB {
	return db.Where("deleted_at IS NULL")
}

// --- Applications -----------------------------------------------------------

func (r *GormRepository) ListApplications(ctx context.Context, f domain.ApplicationFilter) ([]domain.Application, int64, error) {
	q := activeScope(r.db.WithContext(ctx).Table(constants.TableInstructorApplications + " ia")).
		Joins("LEFT JOIN " + constants.TableAppUsers + " u ON u.id = ia.user_id AND u.deleted_at IS NULL")
	if s := strings.TrimSpace(f.ReviewStatus); s != "" {
		q = q.Where("ia.review_status = ?", s)
	}
	if f.HasProfile != nil {
		if *f.HasProfile {
			q = q.Where("ia.headline <> '' AND ia.cv_file_id <> ''")
		} else {
			q = q.Where("ia.headline = '' OR ia.cv_file_id = ''")
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := instrPageParams(f.Page, f.PageSize)
	type applicationWithUserRow struct {
		applicationRow
		FullName     string `gorm:"column:full_name"`
		AvatarFileID string `gorm:"column:avatar_file_id"`
	}
	var rows []applicationWithUserRow
	if err := q.
		Select("ia.*, COALESCE(u.display_name, '') AS full_name, COALESCE(u.avatar_file_id::text, '') AS avatar_file_id").
		Order("ia.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.Application, len(rows))
	for i := range rows {
		out[i] = appRowToDomain(&rows[i].applicationRow)
		out[i].FullName = rows[i].FullName
		out[i].AvatarFileID = rows[i].AvatarFileID
	}
	return out, total, nil
}

func (r *GormRepository) GetApplicationByID(ctx context.Context, id uint) (*domain.Application, error) {
	return loadApplicationRow(ctx, r.db, "ia.id = ?", id)
}

func (r *GormRepository) GetActiveApplicationByUserID(ctx context.Context, userID uint) (*domain.Application, error) {
	return loadApplicationRow(ctx, r.db, "ia.user_id = ?", userID)
}

func (r *GormRepository) UpsertPendingApplication(ctx context.Context, userID uint, p domain.ProfilePayload) (*domain.Application, error) {
	var existing applicationRow
	err := activeScope(r.db.WithContext(ctx)).Where("user_id = ?", userID).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	row := &applicationRow{UserID: userID, ReviewStatus: domain.ReviewStatusPending}
	if err == nil {
		if existing.ReviewStatus != domain.ReviewStatusPending {
			return nil, fmt.Errorf("application not pending")
		}
		row = &existing
	}
	if e := applyPayloadToAppRow(row, p); e != nil {
		return nil, e
	}
	row.RejectionReason = ""
	gormx.TouchUpdated(&row.UpdatedAt)
	if row.ID == 0 {
		gormx.TouchCreatedUpdated(&row.CreatedAt, &row.UpdatedAt)
		if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
			return nil, err
		}
	} else if err := r.db.WithContext(ctx).Save(row).Error; err != nil {
		return nil, err
	}
	a := appRowToDomain(row)
	return &a, nil
}

func (r *GormRepository) SetApplicationReview(ctx context.Context, id uint, status, rejectionReason string) error {
	now := timex.NowUnix()
	return r.db.WithContext(ctx).Model(&applicationRow{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{
			"review_status": status, "rejection_reason": rejectionReason, "updated_at": now,
		}).Error
}

func (r *GormRepository) DeleteApplicationsByUserID(ctx context.Context, userID uint) error {
	return gormx.SoftDeleteWithAudit(ctx, r.db, &applicationRow{}, "user_id = ?", userID)
}

// --- Profiles ---------------------------------------------------------------

func (r *GormRepository) ListProfiles(ctx context.Context, f domain.ProfileFilter) ([]domain.Profile, int64, error) {
	q := activeScope(r.db.WithContext(ctx).Table(constants.TableInstructorProfiles + " ip")).
		Joins("LEFT JOIN " + constants.TableAppUsers + " u ON u.id = ip.user_id AND u.deleted_at IS NULL")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := instrPageParams(f.Page, f.PageSize)
	type profileWithUserRow struct {
		profileRow
		FullName     string `gorm:"column:full_name"`
		AvatarFileID string `gorm:"column:avatar_file_id"`
	}
	var rows []profileWithUserRow
	if err := q.
		Select("ip.*, COALESCE(u.display_name, '') AS full_name, COALESCE(u.avatar_file_id::text, '') AS avatar_file_id").
		Order("ip.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.Profile, len(rows))
	for i := range rows {
		out[i] = profileRowToDomain(&rows[i].profileRow)
		out[i].FullName = rows[i].FullName
		out[i].AvatarFileID = rows[i].AvatarFileID
	}
	return out, total, nil
}

func (r *GormRepository) GetProfileByUserID(ctx context.Context, userID uint) (*domain.Profile, error) {
	return loadProfileRow(ctx, r.db, "ip.user_id = ?", userID)
}

func (r *GormRepository) UpsertProfile(ctx context.Context, in domain.UpsertProfileInput) (*domain.Profile, error) {
	var row profileRow
	err := activeScope(r.db.WithContext(ctx)).Where("user_id = ?", in.UserID).First(&row).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		row = profileRow{UserID: in.UserID}
	}
	if e := applyPayloadToProfileRow(&row, in.ProfilePayload); e != nil {
		return nil, e
	}
	gormx.TouchUpdated(&row.UpdatedAt)
	if row.ID == 0 {
		gormx.TouchCreatedUpdated(&row.CreatedAt, &row.UpdatedAt)
		if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
	} else if err := r.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	p := profileRowToDomain(&row)
	return &p, nil
}

func (r *GormRepository) DeleteProfileByUserID(ctx context.Context, userID uint) error {
	return gormx.SoftDeleteWithAudit(ctx, r.db, &profileRow{}, "user_id = ?", userID)
}

// --- Roster -----------------------------------------------------------------

func (r *GormRepository) ListRoster(ctx context.Context, f domain.RosterFilter) ([]domain.RosterMember, int64, error) {
	base := fmt.Sprintf(`
SELECT u.id, u.display_name, u.email, COALESCE(u.phone, '') AS phone,
       COALESCE(u.avatar_file_id::text, '') AS avatar_file_id
FROM %s u
INNER JOIN %s ur ON ur.user_id = u.id
INNER JOIN %s ro ON ro.id = ur.role_id AND ro.name = ?
WHERE u.deleted_at IS NULL`, constants.TableAppUsers, constants.TableRBACUserRoles, constants.TableRBACRoles)
	q := r.db.WithContext(ctx).Table("(?) AS roster", r.db.Raw(base, domain.RoleNameInstructor))
	if s := strings.TrimSpace(f.Search); s != "" {
		like := "%" + s + "%"
		q = q.Where("display_name ILIKE ? OR email ILIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := instrPageParams(f.Page, f.PageSize)
	type scanRow struct {
		ID           uint
		DisplayName  string
		Email        string
		Phone        string
		AvatarFileID string
	}
	var rows []scanRow
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.RosterMember, len(rows))
	for i, row := range rows {
		out[i] = domain.RosterMember{
			UserID: row.ID, FullName: row.DisplayName, Email: row.Email,
			Phone: row.Phone, AvatarFileID: row.AvatarFileID,
		}
	}
	return out, total, nil
}

func (r *GormRepository) UserHasInstructorRole(ctx context.Context, userID uint) (bool, error) {
	var n int64
	q := fmt.Sprintf(`
SELECT COUNT(*) FROM %s ur
INNER JOIN %s ro ON ro.id = ur.role_id AND ro.name = ?
WHERE ur.user_id = ?`, constants.TableRBACUserRoles, constants.TableRBACRoles)
	if err := r.db.WithContext(ctx).Raw(q, domain.RoleNameInstructor, userID).Scan(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// --- Expertise --------------------------------------------------------------

func (r *GormRepository) ListExpertise(ctx context.Context, userID uint, isTopic bool) (any, error) {
	if isTopic {
		var rows []expertiseTopicRow
		if err := activeScope(r.db.WithContext(ctx)).Where("user_id = ?", userID).Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.ExpertiseTopic, len(rows))
		for i, row := range rows {
			out[i] = domain.ExpertiseTopic{ID: row.ID, UserID: row.UserID, TopicID: row.TopicID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
		}
		return out, nil
	}
	var rows []expertiseSkillRow
	if err := activeScope(r.db.WithContext(ctx)).Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ExpertiseSkill, len(rows))
	for i, row := range rows {
		out[i] = domain.ExpertiseSkill{ID: row.ID, UserID: row.UserID, SkillID: row.SkillID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	}
	return out, nil
}

func (r *GormRepository) InsertExpertise(ctx context.Context, userID, refID uint, isTopic bool) (any, error) {
	return r.addExpertise(ctx, userID, refID, isTopic)
}

func (r *GormRepository) DeleteTopic(ctx context.Context, id uint) error {
	return gormx.SoftDeleteWithAudit(ctx, r.db, &expertiseTopicRow{}, "id = ?", id)
}

func (r *GormRepository) DeleteAllTopicsForUser(ctx context.Context, userID uint) error {
	return gormx.SoftDeleteWithAudit(ctx, r.db, &expertiseTopicRow{}, "user_id = ?", userID)
}

func (r *GormRepository) ListSkills(ctx context.Context, userID uint) ([]domain.ExpertiseSkill, error) {
	v, err := r.ListExpertise(ctx, userID, false)
	if err != nil {
		return nil, err
	}
	return v.([]domain.ExpertiseSkill), nil
}

func (r *GormRepository) DeleteSkill(ctx context.Context, id uint) error {
	return gormx.SoftDeleteWithAudit(ctx, r.db, &expertiseSkillRow{}, "id = ?", id)
}

func (r *GormRepository) DeleteAllSkillsForUser(ctx context.Context, userID uint) error {
	return gormx.SoftDeleteWithAudit(ctx, r.db, &expertiseSkillRow{}, "user_id = ?", userID)
}

func (r *GormRepository) addExpertise(ctx context.Context, userID, refID uint, isTopic bool) (any, error) {
	if isTopic {
		row := &expertiseTopicRow{UserID: userID, TopicID: refID}
		if err := touchAndCreate(ctx, r.db, &row.CreatedAt, &row.UpdatedAt, row); err != nil {
			return nil, err
		}
		return &domain.ExpertiseTopic{ID: row.ID, UserID: row.UserID, TopicID: row.TopicID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
	}
	row := &expertiseSkillRow{UserID: userID, SkillID: refID}
	if err := touchAndCreate(ctx, r.db, &row.CreatedAt, &row.UpdatedAt, row); err != nil {
		return nil, err
	}
	return &domain.ExpertiseSkill{ID: row.ID, UserID: row.UserID, SkillID: row.SkillID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func touchAndCreate(ctx context.Context, db *gorm.DB, created, updated *int64, row any) error {
	gormx.TouchCreatedUpdated(created, updated)
	return db.WithContext(ctx).Create(row).Error
}

// --- Tickets ----------------------------------------------------------------

func (r *GormRepository) ListTickets(ctx context.Context, f domain.TicketFilter) ([]domain.Ticket, int64, error) {
	q := activeScope(r.db.WithContext(ctx).Model(&ticketRow{}))
	if f.UserID > 0 {
		q = q.Where("user_id = ?", f.UserID)
	}
	if s := strings.TrimSpace(f.Status); s != "" {
		q = q.Where("status = ?", s)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := instrPageParams(f.Page, f.PageSize)
	var rows []ticketRow
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.Ticket, len(rows))
	for i, row := range rows {
		out[i] = domain.Ticket{ID: row.ID, UserID: row.UserID, Subject: row.Subject, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	}
	return out, total, nil
}

func (r *GormRepository) GetTicketByID(ctx context.Context, id uint) (*domain.Ticket, error) {
	var row ticketRow
	if err := activeScope(r.db.WithContext(ctx)).First(&row, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	t := domain.Ticket{ID: row.ID, UserID: row.UserID, Subject: row.Subject, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	return &t, nil
}

func (r *GormRepository) CreateTicket(ctx context.Context, userID uint, subject string) (*domain.Ticket, error) {
	row := &ticketRow{UserID: userID, Subject: subject, Status: domain.TicketStatusOpen}
	gormx.TouchCreatedUpdated(&row.CreatedAt, &row.UpdatedAt)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	t := domain.Ticket{ID: row.ID, UserID: row.UserID, Subject: row.Subject, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	return &t, nil
}

func (r *GormRepository) CloseTicket(ctx context.Context, id uint) error {
	now := timex.NowUnix()
	return r.db.WithContext(ctx).Model(&ticketRow{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{"status": domain.TicketStatusClosed, "updated_at": now}).Error
}

func (r *GormRepository) DeleteTicketsByUserID(ctx context.Context, userID uint) error {
	return gormx.SoftDeleteWithAudit(ctx, r.db, &ticketRow{}, "user_id = ?", userID)
}

func (r *GormRepository) ListMessages(ctx context.Context, ticketID uint) ([]domain.TicketMessage, error) {
	var rows []ticketMessageRow
	if err := activeScope(r.db.WithContext(ctx)).Where("ticket_id = ?", ticketID).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.TicketMessage, len(rows))
	for i, row := range rows {
		out[i] = domain.TicketMessage{
			ID: row.ID, TicketID: row.TicketID, AuthorUserID: row.AuthorUserID,
			Body: row.Body, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		}
	}
	return out, nil
}

func (r *GormRepository) AddMessage(ctx context.Context, ticketID, authorUserID uint, body string) (*domain.TicketMessage, error) {
	row := &ticketMessageRow{TicketID: ticketID, AuthorUserID: authorUserID, Body: body}
	gormx.TouchCreatedUpdated(&row.CreatedAt, &row.UpdatedAt)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	m := domain.TicketMessage{
		ID: row.ID, TicketID: row.TicketID, AuthorUserID: row.AuthorUserID,
		Body: row.Body, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	return &m, nil
}

func (r *GormRepository) DeleteMessagesByUserTickets(ctx context.Context, userID uint) error {
	sub := activeScope(r.db.WithContext(ctx).Model(&ticketRow{})).Select("id").Where("user_id = ?", userID)
	return gormx.SoftDeleteWithAudit(ctx, r.db, &ticketMessageRow{}, "ticket_id IN (?)", sub)
}

// --- Wipe -------------------------------------------------------------------

func (r *GormRepository) WipeInstructorScopedData(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := &GormRepository{db: tx}
		if err := repo.DeleteMessagesByUserTickets(ctx, userID); err != nil {
			return err
		}
		if err := repo.DeleteTicketsByUserID(ctx, userID); err != nil {
			return err
		}
		if err := repo.DeleteAllTopicsForUser(ctx, userID); err != nil {
			return err
		}
		if err := repo.DeleteAllSkillsForUser(ctx, userID); err != nil {
			return err
		}
		if err := repo.DeleteProfileByUserID(ctx, userID); err != nil {
			return err
		}
		return repo.DeleteApplicationsByUserID(ctx, userID)
	})
}
