package infra

import (
	"context"
	stderrors "errors"
	"fmt"
	"sort"
	"strings"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/constants"
	sharedutils "mycourse-io-be/internal/shared/utils"
)

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

type courseRow struct {
	ID                        string  `gorm:"column:id;primaryKey"`
	OwnerUserID               string  `gorm:"column:owner_user_id;type:uuid;not null"`
	Slug                      string  `gorm:"column:slug;type:varchar(255);not null"`
	CurrentPublishedVersionID *string `gorm:"column:current_published_version_id"`
	CurrentDraftVersionID     *string `gorm:"column:current_draft_version_id"`
	CreatedAt                 int64   `gorm:"column:created_at;not null"`
	UpdatedAt                 int64   `gorm:"column:updated_at;not null"`
	DeletedAt                 *int64  `gorm:"column:deleted_at"`
	TrashedAt                 *int64  `gorm:"column:trashed_at"`
}

func (courseRow) TableName() string { return constants.TableCourses }

type courseVersionRow struct {
	ID                 string  `gorm:"column:id;primaryKey"`
	CourseID           string  `gorm:"column:course_id;not null"`
	VersionNo          int     `gorm:"column:version_no;not null"`
	Status             string  `gorm:"column:status;type:varchar(32);not null"`
	BasedOnVersionID   *string `gorm:"column:based_on_version_id"`
	Title              string  `gorm:"column:title;type:varchar(255);not null"`
	ShortDescription   string  `gorm:"column:short_description;type:varchar(500);not null"`
	AboutCourse        string  `gorm:"column:about_course;type:text;not null"`
	ThumbnailFileID    *string `gorm:"column:thumbnail_file_id;type:uuid"`
	PreviewVideoFileID *string `gorm:"column:preview_video_file_id;type:uuid"`
	CourseLevelID      *string `gorm:"column:course_level_id"`
	CourseTopicID      *string `gorm:"column:course_topic_id"`
	RowVersion         int64   `gorm:"column:row_version;not null"`
	SubmittedByUserID  *string `gorm:"column:submitted_by_user_id;type:uuid"`
	SubmittedAt        *int64  `gorm:"column:submitted_at"`
	ApprovedByUserID   *string `gorm:"column:approved_by_user_id;type:uuid"`
	ApprovedAt         *int64  `gorm:"column:approved_at"`
	RejectedByUserID   *string `gorm:"column:rejected_by_user_id;type:uuid"`
	RejectedAt         *int64  `gorm:"column:rejected_at"`
	RejectionReason    string  `gorm:"column:rejection_reason;type:text;not null"`
	CreatedAt          int64   `gorm:"column:created_at;not null"`
	UpdatedAt          int64   `gorm:"column:updated_at;not null"`
	DeletedAt          *int64  `gorm:"column:deleted_at"`
}

func (courseVersionRow) TableName() string { return constants.TableCourseVersions }

type courseVersionRefRow struct {
	CourseVersionID string `gorm:"column:course_version_id;primaryKey"`
	RefID           string `gorm:"column:tag_id;primaryKey"`
}

type courseVersionSkillRefRow struct {
	CourseVersionID string `gorm:"column:course_version_id;primaryKey"`
	RefID           string `gorm:"column:skill_id;primaryKey"`
}

type courseVersionOutcomeRefRow struct {
	CourseVersionID string `gorm:"column:course_version_id;primaryKey"`
	RefID           string `gorm:"column:outcome_id;primaryKey"`
}

func (courseVersionRefRow) TableName() string        { return constants.TableCourseVersionTags }
func (courseVersionSkillRefRow) TableName() string   { return constants.TableCourseVersionSkills }
func (courseVersionOutcomeRefRow) TableName() string { return constants.TableCourseVersionOutcomes }

type collaboratorRow struct {
	ID        string `gorm:"column:id;primaryKey"`
	CourseID  string `gorm:"column:course_id;not null"`
	UserID    string `gorm:"column:user_id;type:uuid;not null"`
	Role      string `gorm:"column:role;type:varchar(16);not null"`
	CreatedAt int64  `gorm:"column:created_at;not null"`
	UpdatedAt int64  `gorm:"column:updated_at;not null"`
	DeletedAt *int64 `gorm:"column:deleted_at"`
}

func (collaboratorRow) TableName() string { return constants.TableCourseCollaborators }

type sectionRow struct {
	ID              string `gorm:"column:id;primaryKey"`
	StableID        string `gorm:"column:stable_id;type:uuid;not null"`
	CourseVersionID string `gorm:"column:course_version_id;not null"`
	Title           string `gorm:"column:title;type:varchar(255);not null"`
	Description     string `gorm:"column:description;type:text;not null"`
	OrderIndex      int    `gorm:"column:order_index;not null"`
	RowVersion      int64  `gorm:"column:row_version;not null"`
	CreatedAt       int64  `gorm:"column:created_at;not null"`
	UpdatedAt       int64  `gorm:"column:updated_at;not null"`
	DeletedAt       *int64 `gorm:"column:deleted_at"`
}

func (sectionRow) TableName() string { return constants.TableCourseSections }

type lessonRow struct {
	ID              string `gorm:"column:id;primaryKey"`
	StableID        string `gorm:"column:stable_id;type:uuid;not null"`
	CourseVersionID string `gorm:"column:course_version_id;not null"`
	SectionID       string `gorm:"column:section_id;not null"`
	Title           string `gorm:"column:title;type:varchar(255);not null"`
	Summary         string `gorm:"column:summary;type:text;not null"`
	OrderIndex      int    `gorm:"column:order_index;not null"`
	RowVersion      int64  `gorm:"column:row_version;not null"`
	CreatedAt       int64  `gorm:"column:created_at;not null"`
	UpdatedAt       int64  `gorm:"column:updated_at;not null"`
	DeletedAt       *int64 `gorm:"column:deleted_at"`
}

func (lessonRow) TableName() string { return constants.TableCourseLessons }

type subLessonRow struct {
	ID                  string `gorm:"column:id;primaryKey"`
	StableID            string `gorm:"column:stable_id;type:uuid;not null"`
	CourseVersionID     string `gorm:"column:course_version_id;not null"`
	LessonID            string `gorm:"column:lesson_id;not null"`
	Title               string `gorm:"column:title;type:varchar(255);not null"`
	Kind                string `gorm:"column:kind;type:varchar(16);not null"`
	IsPreview           bool   `gorm:"column:is_preview;not null"`
	OrderIndex          int    `gorm:"column:order_index;not null"`
	EstimatedDurationMs int64  `gorm:"column:estimated_duration_ms;not null"`
	RowVersion          int64  `gorm:"column:row_version;not null"`
	CreatedAt           int64  `gorm:"column:created_at;not null"`
	UpdatedAt           int64  `gorm:"column:updated_at;not null"`
	DeletedAt           *int64 `gorm:"column:deleted_at"`
}

func (subLessonRow) TableName() string { return constants.TableCourseSubLessons }

type subLessonVideoRow struct {
	SubLessonID string `gorm:"column:sub_lesson_id;primaryKey"`
	MediaFileID string `gorm:"column:media_file_id;type:uuid;not null"`
	CreatedAt   int64  `gorm:"column:created_at;not null"`
	UpdatedAt   int64  `gorm:"column:updated_at;not null"`
}

func (subLessonVideoRow) TableName() string { return constants.TableCourseSubLessonVideos }

type subLessonTextRow struct {
	SubLessonID  string `gorm:"column:sub_lesson_id;primaryKey"`
	ContentDelta string `gorm:"column:content_delta;type:jsonb;not null"`
	CreatedAt    int64  `gorm:"column:created_at;not null"`
	UpdatedAt    int64  `gorm:"column:updated_at;not null"`
}

func (subLessonTextRow) TableName() string { return constants.TableCourseSubLessonTexts }

type subLessonQuizRow struct {
	SubLessonID   string `gorm:"column:sub_lesson_id;primaryKey"`
	Prompt        string `gorm:"column:prompt;type:text;not null"`
	AllowMultiple bool   `gorm:"column:allow_multiple;not null"`
	CreatedAt     int64  `gorm:"column:created_at;not null"`
	UpdatedAt     int64  `gorm:"column:updated_at;not null"`
}

func (subLessonQuizRow) TableName() string { return constants.TableCourseSubLessonQuizzes }

type subLessonQuizOptionRow struct {
	ID          string `gorm:"column:id;primaryKey"`
	SubLessonID string `gorm:"column:sub_lesson_id;not null"`
	OptionKey   string `gorm:"column:option_key;type:uuid;not null"`
	Body        string `gorm:"column:body;type:text;not null"`
	IsCorrect   bool   `gorm:"column:is_correct;not null"`
	OrderIndex  int    `gorm:"column:order_index;not null"`
	CreatedAt   int64  `gorm:"column:created_at;not null"`
	UpdatedAt   int64  `gorm:"column:updated_at;not null"`
}

func (subLessonQuizOptionRow) TableName() string { return constants.TableCourseSubLessonQuizOptions }

type leaseRow struct {
	ID               string `gorm:"column:id;primaryKey"`
	CourseID         string `gorm:"column:course_id;not null"`
	CourseVersionID  string `gorm:"column:course_version_id;not null"`
	ResourceType     string `gorm:"column:resource_type;type:varchar(32);not null"`
	ResourceStableID string `gorm:"column:resource_stable_id;type:uuid;not null"`
	HolderUserID     string `gorm:"column:holder_user_id;type:uuid;not null"`
	LeaseToken       string `gorm:"column:lease_token;type:uuid;not null"`
	ExpiresAt        int64  `gorm:"column:expires_at;not null"`
	CreatedAt        int64  `gorm:"column:created_at;not null"`
	UpdatedAt        int64  `gorm:"column:updated_at;not null"`
}

func (leaseRow) TableName() string { return constants.TableCourseEditLeases }

type enrollmentRow struct {
	ID               string `gorm:"column:id;primaryKey"`
	CourseID         string `gorm:"column:course_id;not null"`
	UserID           string `gorm:"column:user_id;type:uuid;not null"`
	CurrentVersionID string `gorm:"column:current_version_id;not null"`
	CreatedAt        int64  `gorm:"column:created_at;not null"`
	UpdatedAt        int64  `gorm:"column:updated_at;not null"`
	DeletedAt        *int64 `gorm:"column:deleted_at"`
}

func (enrollmentRow) TableName() string { return constants.TableCourseEnrollments }

type progressRow struct {
	ID               string  `gorm:"column:id;primaryKey"`
	EnrollmentID     string  `gorm:"column:enrollment_id;not null"`
	StableContentID  string  `gorm:"column:stable_content_id;type:uuid;not null"`
	ContentType      string  `gorm:"column:content_type;type:varchar(24);not null"`
	Status           string  `gorm:"column:status;type:varchar(24);not null"`
	Score            float64 `gorm:"column:score;not null"`
	QuizAttempt      string  `gorm:"column:quiz_attempt;type:jsonb;not null"`
	LastInteractedAt *int64  `gorm:"column:last_interacted_at"`
	CreatedAt        int64   `gorm:"column:created_at;not null"`
	UpdatedAt        int64   `gorm:"column:updated_at;not null"`
	DeletedAt        *int64  `gorm:"column:deleted_at"`
}

func (progressRow) TableName() string { return constants.TableCourseProgressItems }

type mediaInfoRow struct {
	ID       string `gorm:"column:id"`
	Kind     string `gorm:"column:kind"`
	Status   string `gorm:"column:status"`
	MimeType string `gorm:"column:mime_type"`
	URL      string `gorm:"column:url"`
}

const courseListBaseColumns = `c.id, c.owner_user_id, c.slug, c.current_published_version_id, c.current_draft_version_id, c.created_at, c.updated_at, c.deleted_at, c.trashed_at`

type courseListScanRow struct {
	ID                        string  `gorm:"column:id"`
	OwnerUserID               string  `gorm:"column:owner_user_id"`
	Slug                      string  `gorm:"column:slug"`
	CurrentPublishedVersionID *string `gorm:"column:current_published_version_id"`
	CurrentDraftVersionID     *string `gorm:"column:current_draft_version_id"`
	CreatedAt                 int64   `gorm:"column:created_at"`
	UpdatedAt                 int64   `gorm:"column:updated_at"`
	DeletedAt                 *int64  `gorm:"column:deleted_at"`
	TrashedAt                 *int64  `gorm:"column:trashed_at"`
	Role                      string  `gorm:"column:role"`
	Title                     string  `gorm:"column:title"`
	ReviewStatus              string  `gorm:"column:review_status"`
	VersionID                 string  `gorm:"column:version_id"`
	VersionNo                 int     `gorm:"column:version_no"`
	HasPublished              bool    `gorm:"column:has_published"`
	HasDraft                  bool    `gorm:"column:has_draft"`
	ThumbnailFileID           string  `gorm:"column:thumbnail_file_id"`
	ThumbnailURL              string  `gorm:"column:thumbnail_url"`
	PreviewVideoFileID        string  `gorm:"column:preview_video_file_id"`
	DraftReviewStatus         string  `gorm:"column:draft_review_status"`
}

func (row *courseListScanRow) asCourseRow() courseRow {
	return courseRow{
		ID:                        row.ID,
		OwnerUserID:               row.OwnerUserID,
		Slug:                      row.Slug,
		CurrentPublishedVersionID: row.CurrentPublishedVersionID,
		CurrentDraftVersionID:     row.CurrentDraftVersionID,
		CreatedAt:                 row.CreatedAt,
		UpdatedAt:                 row.UpdatedAt,
		DeletedAt:                 row.DeletedAt,
		TrashedAt:                 row.TrashedAt,
	}
}

type courseAccess struct {
	courseRow
	Role string
}

const courseMediaKindImage = "IMAGE"

func (r *GormRepository) loadSubLessonDomain(ctx context.Context, db *gorm.DB, subLessonID string) (*domain.SubLesson, error) {
	var row subLessonRow
	if err := db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", subLessonID).First(&row).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCourseNotFound
		}
		return nil, err
	}
	subs, err := r.batchHydrateSubLessons(ctx, db, []subLessonRow{row}, nil)
	if err != nil {
		return nil, err
	}
	sub, ok := subs[row.ID]
	if !ok {
		return nil, domain.ErrCourseNotFound
	}
	return &sub, nil
}

func (r *GormRepository) toCourseVersion(ctx context.Context, db *gorm.DB, row *courseVersionRow) (*domain.CourseVersion, error) {
	assets, err := r.loadCourseVersionAssets(ctx, db, row)
	if err != nil {
		return nil, err
	}
	return mapCourseVersionRow(row, assets), nil
}

type courseVersionAssets struct {
	tagIDs     []string
	skillIDs   []string
	outcomeIDs []string
	thumbURL   string
	videoURL   string
}

func (r *GormRepository) loadCourseVersionAssets(ctx context.Context, db *gorm.DB, row *courseVersionRow) (*courseVersionAssets, error) {
	var assets courseVersionAssets
	group, gctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		ids, err := r.loadVersionRefIDs(gctx, parallelReadDB(db), constants.TableCourseVersionTags, "tag_id", row.ID)
		if err != nil {
			return err
		}
		assets.tagIDs = ids
		return nil
	})
	group.Go(func() error {
		ids, err := r.loadVersionRefIDs(gctx, parallelReadDB(db), constants.TableCourseVersionSkills, "skill_id", row.ID)
		if err != nil {
			return err
		}
		assets.skillIDs = ids
		return nil
	})
	group.Go(func() error {
		ids, err := r.loadVersionRefIDs(gctx, parallelReadDB(db), constants.TableCourseVersionOutcomes, "outcome_id", row.ID)
		if err != nil {
			return err
		}
		assets.outcomeIDs = ids
		return nil
	})
	group.Go(func() error {
		mediaIDs := make([]string, 0, 2)
		if row.ThumbnailFileID != nil {
			mediaIDs = append(mediaIDs, *row.ThumbnailFileID)
		}
		if row.PreviewVideoFileID != nil {
			mediaIDs = append(mediaIDs, *row.PreviewVideoFileID)
		}
		if len(mediaIDs) == 0 {
			return nil
		}
		urls, err := r.batchMediaURLMap(gctx, parallelReadDB(db), mediaIDs)
		if err != nil {
			return err
		}
		if row.ThumbnailFileID != nil {
			assets.thumbURL = urls[*row.ThumbnailFileID]
		}
		if row.PreviewVideoFileID != nil {
			assets.videoURL = urls[*row.PreviewVideoFileID]
		}
		return nil
	})
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return &assets, nil
}

func mapCourseVersionRow(row *courseVersionRow, assets *courseVersionAssets) *domain.CourseVersion {
	return &domain.CourseVersion{
		ID: row.ID, CourseID: row.CourseID, VersionNo: row.VersionNo, Status: row.Status,
		BasedOnVersionID: row.BasedOnVersionID,
		Title:            row.Title, ShortDescription: row.ShortDescription, AboutCourse: row.AboutCourse,
		ThumbnailFileID: row.ThumbnailFileID, ThumbnailURL: assets.thumbURL,
		PreviewVideoFileID: row.PreviewVideoFileID, PreviewVideoURL: assets.videoURL,
		CourseLevelID: row.CourseLevelID, CourseTopicID: row.CourseTopicID,
		TagIDs: assets.tagIDs, SkillIDs: assets.skillIDs, OutcomeIDs: assets.outcomeIDs,
		RowVersion:        row.RowVersion,
		SubmittedByUserID: row.SubmittedByUserID, SubmittedAt: row.SubmittedAt,
		ApprovedByUserID: row.ApprovedByUserID, ApprovedAt: row.ApprovedAt,
		RejectedByUserID: row.RejectedByUserID, RejectedAt: row.RejectedAt,
		RejectionReason: row.RejectionReason,
		CreatedAt:       row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func (r *GormRepository) loadVersionRefIDs(ctx context.Context, db *gorm.DB, table, col string, versionID string) ([]string, error) {
	var ids []string
	query := fmt.Sprintf("SELECT %s AS id FROM %s WHERE course_version_id = ?", col, table)
	if err := db.WithContext(ctx).Raw(query, versionID).Scan(&ids).Error; err != nil {
		return nil, err
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

func (r *GormRepository) batchMediaDurationMs(ctx context.Context, db *gorm.DB, fileIDs []string) (map[string]int64, error) {
	_, durations, err := r.batchMediaURLAndDurationMsMaps(ctx, db, fileIDs)
	return durations, err
}

func (r *GormRepository) resolveSubLessonDomainDuration(ctx context.Context, db *gorm.DB, sub *domain.SubLesson) error {
	if sub == nil {
		return nil
	}
	var mediaMs int64
	if sub.Kind == domain.SubLessonKindVideo && sub.Video != nil {
		fileID := strings.TrimSpace(sub.Video.MediaFileID)
		if fileID != "" {
			durations, err := r.batchMediaDurationMs(ctx, db, []string{fileID})
			if err != nil {
				return err
			}
			mediaMs = durations[fileID]
		}
	}
	sub.EstimatedDurationMs = resolveSubLessonEstimatedDurationMs(*sub, mediaMs)
	return nil
}

func (r *GormRepository) userIsInstructor(ctx context.Context, db *gorm.DB, userID string) bool {
	var count int64
	_ = db.WithContext(ctx).Raw(`
SELECT COUNT(*)
FROM user_roles ur
INNER JOIN roles ro ON ro.id = ur.role_id
WHERE ur.user_id = ? AND ro.name IN ('instructor', 'sysadmin', 'admin')`, userID).Scan(&count).Error
	return count > 0
}

func (r *GormRepository) nextOrderIndex(ctx context.Context, db *gorm.DB, table, predicate string, args ...any) (int, error) {
	var next int
	query := fmt.Sprintf("SELECT COALESCE(MAX(order_index), -1) + 1 FROM %s WHERE %s", table, predicate)
	if err := db.WithContext(ctx).Raw(query, args...).Scan(&next).Error; err != nil {
		return 0, err
	}
	return next, nil
}

func toCourse(row *courseRow) domain.Course {
	return domain.Course{
		ID: row.ID, OwnerUserID: row.OwnerUserID, Slug: row.Slug,
		CurrentPublishedVersionID: row.CurrentPublishedVersionID,
		CurrentDraftVersionID:     row.CurrentDraftVersionID,
		TrashedAt:                 row.TrashedAt,
		CreatedAt:                 row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func toCourseListItem(row *courseListScanRow) domain.CourseListItem {
	cr := row.asCourseRow()
	return domain.CourseListItem{
		Course:             toCourse(&cr),
		Title:              row.Title,
		ReviewStatus:       row.ReviewStatus,
		VersionID:          row.VersionID,
		VersionNo:          row.VersionNo,
		CollaboratorRole:   row.Role,
		HasPublished:       row.HasPublished,
		HasDraft:           row.HasDraft,
		ThumbnailFileID:    row.ThumbnailFileID,
		ThumbnailURL:       row.ThumbnailURL,
		PreviewVideoFileID: row.PreviewVideoFileID,
		DraftReviewStatus:  row.DraftReviewStatus,
	}
}

func toSection(row *sectionRow) domain.Section {
	return domain.Section{
		ID: row.ID, StableID: row.StableID, Title: row.Title, Description: row.Description,
		OrderIndex: row.OrderIndex, RowVersion: row.RowVersion, Lessons: []domain.Lesson{},
	}
}

func toLesson(row *lessonRow) domain.Lesson {
	return domain.Lesson{
		ID: row.ID, StableID: row.StableID, Title: row.Title, Summary: row.Summary,
		OrderIndex: row.OrderIndex, RowVersion: row.RowVersion, SubLessons: []domain.SubLesson{},
	}
}

func toSubLesson(row *subLessonRow) domain.SubLesson {
	return domain.SubLesson{
		ID: row.ID, StableID: row.StableID, Title: row.Title, Kind: row.Kind,
		IsPreview: row.IsPreview, OrderIndex: row.OrderIndex, RowVersion: row.RowVersion,
		EstimatedDurationMs: row.EstimatedDurationMs,
	}
}

func toLease(row *leaseRow) domain.Lease {
	return domain.Lease{
		ID: row.ID, CourseID: row.CourseID, CourseVersionID: row.CourseVersionID, ResourceType: row.ResourceType,
		ResourceStableID: row.ResourceStableID, HolderUserID: row.HolderUserID, LeaseToken: row.LeaseToken,
		ExpiresAt: row.ExpiresAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func toEnrollment(row *enrollmentRow) domain.Enrollment {
	return domain.Enrollment{
		ID: row.ID, CourseID: row.CourseID, UserID: row.UserID, CurrentVersionID: row.CurrentVersionID,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func toProgressItem(row *progressRow) domain.ProgressItem {
	return domain.ProgressItem{
		ID: row.ID, StableContentID: row.StableContentID, ContentType: row.ContentType,
		Status: row.Status, Score: row.Score, QuizAttempt: row.QuizAttempt, LastInteractedAt: row.LastInteractedAt,
	}
}

func mapQuizOptions(rows []subLessonQuizOptionRow) []domain.QuizOption {
	out := make([]domain.QuizOption, len(rows))
	for i := range rows {
		out[i] = domain.QuizOption{
			ID: rows[i].ID, OptionKey: rows[i].OptionKey, Body: rows[i].Body,
			IsCorrect: rows[i].IsCorrect, OrderIndex: rows[i].OrderIndex,
		}
	}
	return out
}

func sameStableIDs[TRow any](rows []TRow, ordered []string, stableIDOf func(TRow) string) bool {
	if len(rows) != len(ordered) {
		return false
	}
	current := make([]string, len(rows))
	for i, row := range rows {
		current[i] = stableIDOf(row)
	}
	return sharedutils.SameStringSet(current, ordered)
}

func filterPreviewOutline(rows []domain.Section) []domain.Section {
	out := make([]domain.Section, 0, len(rows))
	for _, section := range rows {
		lessons := make([]domain.Lesson, 0, len(section.Lessons))
		for _, lesson := range section.Lessons {
			subs := make([]domain.SubLesson, 0, len(lesson.SubLessons))
			for _, sub := range lesson.SubLessons {
				if sub.IsPreview && sub.Kind != domain.SubLessonKindQuiz {
					subs = append(subs, sub)
				}
			}
			if len(subs) == 0 {
				continue
			}
			lesson.SubLessons = subs
			lessons = append(lessons, lesson)
		}
		if len(lessons) == 0 {
			continue
		}
		section.Lessons = lessons
		out = append(out, section)
	}
	return out
}
