package infra

import (
	"context"
	stderrors "errors"
	"fmt"
	"sort"

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
	ID                        uint   `gorm:"column:id;primaryKey"`
	OwnerUserID               uint   `gorm:"column:owner_user_id;not null"`
	Slug                      string `gorm:"column:slug;type:varchar(255);not null"`
	CurrentPublishedVersionID *uint  `gorm:"column:current_published_version_id"`
	CurrentDraftVersionID     *uint  `gorm:"column:current_draft_version_id"`
	CreatedAt                 int64  `gorm:"column:created_at;not null"`
	UpdatedAt                 int64  `gorm:"column:updated_at;not null"`
	DeletedAt                 *int64 `gorm:"column:deleted_at"`
}

func (courseRow) TableName() string { return constants.TableCourses }

type courseVersionRow struct {
	ID                 uint    `gorm:"column:id;primaryKey"`
	CourseID           uint    `gorm:"column:course_id;not null"`
	VersionNo          int     `gorm:"column:version_no;not null"`
	Status             string  `gorm:"column:status;type:varchar(32);not null"`
	BasedOnVersionID   *uint   `gorm:"column:based_on_version_id"`
	Title              string  `gorm:"column:title;type:varchar(255);not null"`
	ShortDescription   string  `gorm:"column:short_description;type:varchar(500);not null"`
	AboutCourse        string  `gorm:"column:about_course;type:text;not null"`
	ThumbnailFileID    *string `gorm:"column:thumbnail_file_id;type:uuid"`
	PreviewVideoFileID *string `gorm:"column:preview_video_file_id;type:uuid"`
	CourseLevelID      *uint   `gorm:"column:course_level_id"`
	CourseTopicID      *uint   `gorm:"column:course_topic_id"`
	RowVersion         int64   `gorm:"column:row_version;not null"`
	SubmittedByUserID  *uint   `gorm:"column:submitted_by_user_id"`
	SubmittedAt        *int64  `gorm:"column:submitted_at"`
	ApprovedByUserID   *uint   `gorm:"column:approved_by_user_id"`
	ApprovedAt         *int64  `gorm:"column:approved_at"`
	RejectedByUserID   *uint   `gorm:"column:rejected_by_user_id"`
	RejectedAt         *int64  `gorm:"column:rejected_at"`
	RejectionReason    string  `gorm:"column:rejection_reason;type:text;not null"`
	CreatedAt          int64   `gorm:"column:created_at;not null"`
	UpdatedAt          int64   `gorm:"column:updated_at;not null"`
	DeletedAt          *int64  `gorm:"column:deleted_at"`
}

func (courseVersionRow) TableName() string { return constants.TableCourseVersions }

type courseVersionRefRow struct {
	CourseVersionID uint `gorm:"column:course_version_id;primaryKey"`
	RefID           uint `gorm:"column:tag_id;primaryKey"`
}

type courseVersionSkillRefRow struct {
	CourseVersionID uint `gorm:"column:course_version_id;primaryKey"`
	RefID           uint `gorm:"column:skill_id;primaryKey"`
}

type courseVersionOutcomeRefRow struct {
	CourseVersionID uint `gorm:"column:course_version_id;primaryKey"`
	RefID           uint `gorm:"column:outcome_id;primaryKey"`
}

func (courseVersionRefRow) TableName() string        { return constants.TableCourseVersionTags }
func (courseVersionSkillRefRow) TableName() string   { return constants.TableCourseVersionSkills }
func (courseVersionOutcomeRefRow) TableName() string { return constants.TableCourseVersionOutcomes }

type collaboratorRow struct {
	ID        uint   `gorm:"column:id;primaryKey"`
	CourseID  uint   `gorm:"column:course_id;not null"`
	UserID    uint   `gorm:"column:user_id;not null"`
	Role      string `gorm:"column:role;type:varchar(16);not null"`
	CreatedAt int64  `gorm:"column:created_at;not null"`
	UpdatedAt int64  `gorm:"column:updated_at;not null"`
	DeletedAt *int64 `gorm:"column:deleted_at"`
}

func (collaboratorRow) TableName() string { return constants.TableCourseCollaborators }

type sectionRow struct {
	ID              uint   `gorm:"column:id;primaryKey"`
	StableID        string `gorm:"column:stable_id;type:uuid;not null"`
	CourseVersionID uint   `gorm:"column:course_version_id;not null"`
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
	ID              uint   `gorm:"column:id;primaryKey"`
	StableID        string `gorm:"column:stable_id;type:uuid;not null"`
	CourseVersionID uint   `gorm:"column:course_version_id;not null"`
	SectionID       uint   `gorm:"column:section_id;not null"`
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
	ID              uint   `gorm:"column:id;primaryKey"`
	StableID        string `gorm:"column:stable_id;type:uuid;not null"`
	CourseVersionID uint   `gorm:"column:course_version_id;not null"`
	LessonID        uint   `gorm:"column:lesson_id;not null"`
	Title           string `gorm:"column:title;type:varchar(255);not null"`
	Kind            string `gorm:"column:kind;type:varchar(16);not null"`
	IsPreview       bool   `gorm:"column:is_preview;not null"`
	OrderIndex      int    `gorm:"column:order_index;not null"`
	RowVersion      int64  `gorm:"column:row_version;not null"`
	CreatedAt       int64  `gorm:"column:created_at;not null"`
	UpdatedAt       int64  `gorm:"column:updated_at;not null"`
	DeletedAt       *int64 `gorm:"column:deleted_at"`
}

func (subLessonRow) TableName() string { return constants.TableCourseSubLessons }

type subLessonVideoRow struct {
	SubLessonID uint   `gorm:"column:sub_lesson_id;primaryKey"`
	MediaFileID string `gorm:"column:media_file_id;type:uuid;not null"`
	CreatedAt   int64  `gorm:"column:created_at;not null"`
	UpdatedAt   int64  `gorm:"column:updated_at;not null"`
}

func (subLessonVideoRow) TableName() string { return constants.TableCourseSubLessonVideos }

type subLessonTextRow struct {
	SubLessonID  uint   `gorm:"column:sub_lesson_id;primaryKey"`
	ContentDelta string `gorm:"column:content_delta;type:jsonb;not null"`
	CreatedAt    int64  `gorm:"column:created_at;not null"`
	UpdatedAt    int64  `gorm:"column:updated_at;not null"`
}

func (subLessonTextRow) TableName() string { return constants.TableCourseSubLessonTexts }

type subLessonQuizRow struct {
	SubLessonID   uint   `gorm:"column:sub_lesson_id;primaryKey"`
	Prompt        string `gorm:"column:prompt;type:text;not null"`
	AllowMultiple bool   `gorm:"column:allow_multiple;not null"`
	CreatedAt     int64  `gorm:"column:created_at;not null"`
	UpdatedAt     int64  `gorm:"column:updated_at;not null"`
}

func (subLessonQuizRow) TableName() string { return constants.TableCourseSubLessonQuizzes }

type subLessonQuizOptionRow struct {
	ID          uint   `gorm:"column:id;primaryKey"`
	SubLessonID uint   `gorm:"column:sub_lesson_id;not null"`
	OptionKey   string `gorm:"column:option_key;type:uuid;not null"`
	Body        string `gorm:"column:body;type:text;not null"`
	IsCorrect   bool   `gorm:"column:is_correct;not null"`
	OrderIndex  int    `gorm:"column:order_index;not null"`
	CreatedAt   int64  `gorm:"column:created_at;not null"`
	UpdatedAt   int64  `gorm:"column:updated_at;not null"`
}

func (subLessonQuizOptionRow) TableName() string { return constants.TableCourseSubLessonQuizOptions }

type leaseRow struct {
	ID               uint   `gorm:"column:id;primaryKey"`
	CourseID         uint   `gorm:"column:course_id;not null"`
	CourseVersionID  uint   `gorm:"column:course_version_id;not null"`
	ResourceType     string `gorm:"column:resource_type;type:varchar(32);not null"`
	ResourceStableID string `gorm:"column:resource_stable_id;type:uuid;not null"`
	HolderUserID     uint   `gorm:"column:holder_user_id;not null"`
	LeaseToken       string `gorm:"column:lease_token;type:uuid;not null"`
	ExpiresAt        int64  `gorm:"column:expires_at;not null"`
	CreatedAt        int64  `gorm:"column:created_at;not null"`
	UpdatedAt        int64  `gorm:"column:updated_at;not null"`
}

func (leaseRow) TableName() string { return constants.TableCourseEditLeases }

type enrollmentRow struct {
	ID               uint   `gorm:"column:id;primaryKey"`
	CourseID         uint   `gorm:"column:course_id;not null"`
	UserID           uint   `gorm:"column:user_id;not null"`
	CurrentVersionID uint   `gorm:"column:current_version_id;not null"`
	CreatedAt        int64  `gorm:"column:created_at;not null"`
	UpdatedAt        int64  `gorm:"column:updated_at;not null"`
	DeletedAt        *int64 `gorm:"column:deleted_at"`
}

func (enrollmentRow) TableName() string { return constants.TableCourseEnrollments }

type progressRow struct {
	ID               uint    `gorm:"column:id;primaryKey"`
	EnrollmentID     uint    `gorm:"column:enrollment_id;not null"`
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

type userInfoRow struct {
	ID           uint    `gorm:"column:id"`
	DisplayName  string  `gorm:"column:display_name"`
	Email        string  `gorm:"column:email"`
	AvatarFileID *string `gorm:"column:avatar_file_id"`
}

type mediaInfoRow struct {
	ID       string `gorm:"column:id"`
	Kind     string `gorm:"column:kind"`
	Status   string `gorm:"column:status"`
	MimeType string `gorm:"column:mime_type"`
	URL      string `gorm:"column:url"`
}

const courseListBaseColumns = `c.id, c.owner_user_id, c.slug, c.current_published_version_id, c.current_draft_version_id, c.created_at, c.updated_at, c.deleted_at`

type courseListScanRow struct {
	ID                        uint   `gorm:"column:id"`
	OwnerUserID               uint   `gorm:"column:owner_user_id"`
	Slug                      string `gorm:"column:slug"`
	CurrentPublishedVersionID *uint  `gorm:"column:current_published_version_id"`
	CurrentDraftVersionID     *uint  `gorm:"column:current_draft_version_id"`
	CreatedAt                 int64  `gorm:"column:created_at"`
	UpdatedAt                 int64  `gorm:"column:updated_at"`
	DeletedAt                 *int64 `gorm:"column:deleted_at"`
	Role                      string `gorm:"column:role"`
	Title                     string `gorm:"column:title"`
	ReviewStatus              string `gorm:"column:review_status"`
	VersionNo                 int    `gorm:"column:version_no"`
	HasPublished              bool   `gorm:"column:has_published"`
	HasDraft                  bool   `gorm:"column:has_draft"`
	ThumbnailFileID           string `gorm:"column:thumbnail_file_id"`
	ThumbnailURL              string `gorm:"column:thumbnail_url"`
	PreviewVideoFileID        string `gorm:"column:preview_video_file_id"`
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
	}
}

type courseAccess struct {
	courseRow
	Role string
}

const courseMediaKindImage = "IMAGE"

func (r *GormRepository) loadSubLessonDomain(ctx context.Context, db *gorm.DB, subLessonID uint) (*domain.SubLesson, error) {
	var row subLessonRow
	if err := db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", subLessonID).First(&row).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCourseNotFound
		}
		return nil, err
	}
	sub := toSubLesson(&row)
	switch row.Kind {
	case domain.SubLessonKindVideo:
		var video subLessonVideoRow
		if err := db.WithContext(ctx).Where("sub_lesson_id = ?", row.ID).First(&video).Error; err == nil {
			url, _ := r.mediaURL(ctx, db, video.MediaFileID)
			sub.Video = &domain.VideoContent{MediaFileID: video.MediaFileID, MediaURL: url}
		}
	case domain.SubLessonKindText:
		var text subLessonTextRow
		if err := db.WithContext(ctx).Where("sub_lesson_id = ?", row.ID).First(&text).Error; err == nil {
			sub.Text = &domain.TextContent{ContentDelta: text.ContentDelta}
		}
	case domain.SubLessonKindQuiz:
		var quiz subLessonQuizRow
		if err := db.WithContext(ctx).Where("sub_lesson_id = ?", row.ID).First(&quiz).Error; err == nil {
			var options []subLessonQuizOptionRow
			if err := db.WithContext(ctx).Where("sub_lesson_id = ?", row.ID).Order("order_index ASC").Find(&options).Error; err != nil {
				return nil, err
			}
			sub.Quiz = &domain.QuizContent{Prompt: quiz.Prompt, AllowMultiple: quiz.AllowMultiple, Options: mapQuizOptions(options)}
		}
	}
	return &sub, nil
}

func (r *GormRepository) toCourseVersion(ctx context.Context, db *gorm.DB, row *courseVersionRow) (*domain.CourseVersion, error) {
	tagIDs, err := r.loadVersionRefIDs(ctx, db, constants.TableCourseVersionTags, "tag_id", row.ID)
	if err != nil {
		return nil, err
	}
	skillIDs, err := r.loadVersionRefIDs(ctx, db, constants.TableCourseVersionSkills, "skill_id", row.ID)
	if err != nil {
		return nil, err
	}
	outcomeIDs, err := r.loadVersionRefIDs(ctx, db, constants.TableCourseVersionOutcomes, "outcome_id", row.ID)
	if err != nil {
		return nil, err
	}
	thumbURL := ""
	if row.ThumbnailFileID != nil {
		thumbURL, _ = r.mediaURL(ctx, db, *row.ThumbnailFileID)
	}
	videoURL := ""
	if row.PreviewVideoFileID != nil {
		videoURL, _ = r.mediaURL(ctx, db, *row.PreviewVideoFileID)
	}
	return &domain.CourseVersion{
		ID: row.ID, CourseID: row.CourseID, VersionNo: row.VersionNo, Status: row.Status,
		BasedOnVersionID: row.BasedOnVersionID,
		Title:            row.Title, ShortDescription: row.ShortDescription, AboutCourse: row.AboutCourse,
		ThumbnailFileID: row.ThumbnailFileID, ThumbnailURL: thumbURL,
		PreviewVideoFileID: row.PreviewVideoFileID, PreviewVideoURL: videoURL,
		CourseLevelID: row.CourseLevelID, CourseTopicID: row.CourseTopicID,
		TagIDs: tagIDs, SkillIDs: skillIDs, OutcomeIDs: outcomeIDs,
		RowVersion:        row.RowVersion,
		SubmittedByUserID: row.SubmittedByUserID, SubmittedAt: row.SubmittedAt,
		ApprovedByUserID: row.ApprovedByUserID, ApprovedAt: row.ApprovedAt,
		RejectedByUserID: row.RejectedByUserID, RejectedAt: row.RejectedAt,
		RejectionReason: row.RejectionReason,
		CreatedAt:       row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *GormRepository) loadVersionRefIDs(ctx context.Context, db *gorm.DB, table, col string, versionID uint) ([]uint, error) {
	type row struct {
		ID uint `gorm:"column:id"`
	}
	var ids []uint
	query := fmt.Sprintf("SELECT %s AS id FROM %s WHERE course_version_id = ?", col, table)
	if err := db.WithContext(ctx).Raw(query, versionID).Scan(&ids).Error; err != nil {
		return nil, err
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

func (r *GormRepository) mediaURL(ctx context.Context, db *gorm.DB, fileID string) (string, error) {
	var row mediaInfoRow
	if err := db.WithContext(ctx).Table(constants.TableMediaFiles).Select("id, url").Where("id = ? AND deleted_at IS NULL", fileID).First(&row).Error; err != nil {
		return "", err
	}
	return row.URL, nil
}

func (r *GormRepository) userIsInstructor(ctx context.Context, db *gorm.DB, userID uint) bool {
	var count int64
	_ = db.WithContext(ctx).Raw(`
SELECT COUNT(*)
FROM user_roles ur
INNER JOIN roles ro ON ro.id = ur.role_id
WHERE ur.user_id = ? AND ro.name = 'instructor'`, userID).Scan(&count).Error
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
		CreatedAt:                 row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func toCourseListItem(row *courseListScanRow) domain.CourseListItem {
	cr := row.asCourseRow()
	return domain.CourseListItem{
		Course:             toCourse(&cr),
		Title:              row.Title,
		ReviewStatus:       row.ReviewStatus,
		VersionNo:          row.VersionNo,
		CollaboratorRole:   row.Role,
		HasPublished:       row.HasPublished,
		HasDraft:           row.HasDraft,
		ThumbnailFileID:    row.ThumbnailFileID,
		ThumbnailURL:       row.ThumbnailURL,
		PreviewVideoFileID: row.PreviewVideoFileID,
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
				if sub.IsPreview {
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
