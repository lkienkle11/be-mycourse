package infra

import (
	"context"

	"gorm.io/gorm"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/timex"
)

func (r *GormRepository) loadCourseDetail(ctx context.Context, db *gorm.DB, courseID, userID uint, includeDraft bool) (*domain.CourseDetail, error) {
	access, err := r.requireCourseAccess(ctx, db, courseID, userID)
	if err != nil {
		return nil, err
	}
	course := toCourse(&access.courseRow)
	var live *domain.CourseVersion
	var draft *domain.CourseVersion
	outlineVersionID := uint(0)
	if access.CurrentPublishedVersionID != nil {
		row, err := r.loadVersionRow(ctx, db, *access.CurrentPublishedVersionID)
		if err != nil {
			return nil, err
		}
		version, err := r.toCourseVersion(ctx, db, row)
		if err != nil {
			return nil, err
		}
		live = version
		outlineVersionID = row.ID
	}
	if includeDraft && access.CurrentDraftVersionID != nil {
		row, err := r.loadVersionRow(ctx, db, *access.CurrentDraftVersionID)
		if err != nil {
			return nil, err
		}
		version, err := r.toCourseVersion(ctx, db, row)
		if err != nil {
			return nil, err
		}
		draft = version
		outlineVersionID = row.ID
	}
	collabs, err := r.loadCollaborators(ctx, db, courseID)
	if err != nil {
		return nil, err
	}
	outline := []domain.Section{}
	if outlineVersionID > 0 {
		outline, err = r.loadOutline(ctx, db, outlineVersionID)
		if err != nil {
			return nil, err
		}
	}
	return &domain.CourseDetail{
		Course:           course,
		CollaboratorRole: access.Role,
		LiveVersion:      live,
		DraftVersion:     draft,
		Collaborators:    collabs,
		Outline:          outline,
	}, nil
}

func (r *GormRepository) loadLearnerCourseDetail(ctx context.Context, db *gorm.DB, courseID, userID uint) (*domain.CourseDetail, error) {
	course, err := r.loadCourse(ctx, db, courseID)
	if err != nil {
		return nil, err
	}
	if course.CurrentPublishedVersionID == nil {
		return nil, domain.ErrCoursePublishedRequired
	}
	versionID := *course.CurrentPublishedVersionID
	var enrollment *enrollmentRow
	var enrolled enrollmentRow
	if err := db.WithContext(ctx).Where("course_id = ? AND user_id = ? AND deleted_at IS NULL", courseID, userID).First(&enrolled).Error; err == nil {
		enrollment = &enrolled
		versionID = enrolled.CurrentVersionID
	}
	versionRow, err := r.loadVersionRow(ctx, db, versionID)
	if err != nil {
		return nil, err
	}
	version, err := r.toCourseVersion(ctx, db, versionRow)
	if err != nil {
		return nil, err
	}
	outline, err := r.loadOutline(ctx, db, versionID)
	if err != nil {
		return nil, err
	}
	if enrollment == nil {
		outline = filterPreviewOutline(outline)
	}
	return &domain.CourseDetail{
		Course:           toCourse(course),
		CollaboratorRole: "LEARNER",
		LiveVersion:      version,
		Collaborators:    []domain.Collaborator{},
		Outline:          outline,
	}, nil
}

func (r *GormRepository) loadProgress(ctx context.Context, db *gorm.DB, courseID, userID uint) (*domain.CourseProgress, error) {
	enrollment, err := r.requireEnrollment(ctx, db, courseID, userID)
	if err != nil {
		return nil, err
	}
	var rows []progressRow
	if err := db.WithContext(ctx).Where("enrollment_id = ? AND deleted_at IS NULL", enrollment.ID).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.ProgressItem, len(rows))
	for i := range rows {
		items[i] = toProgressItem(&rows[i])
	}
	progress := &domain.CourseProgress{
		Enrollment: toEnrollment(enrollment),
		Items:      items,
	}
	return progress, nil
}

func (r *GormRepository) requireEnrollment(ctx context.Context, db *gorm.DB, courseID, userID uint) (*enrollmentRow, error) {
	return loadActiveRow[enrollmentRow](ctx, db, domain.ErrCourseEnrollmentNotFound, "course_id = ? AND user_id = ? AND deleted_at IS NULL", courseID, userID)
}

func (r *GormRepository) requireCourseAccess(ctx context.Context, db *gorm.DB, courseID, userID uint) (*courseAccess, error) {
	course, err := r.loadCourse(ctx, db, courseID)
	if err != nil {
		return nil, err
	}
	if course.OwnerUserID == userID {
		return &courseAccess{courseRow: *course, Role: domain.CollaboratorRoleOwner}, nil
	}
	collab, err := loadActiveRow[collaboratorRow](ctx, db, domain.ErrCourseCollaboratorAccess, "course_id = ? AND user_id = ? AND deleted_at IS NULL", courseID, userID)
	if err != nil {
		return nil, err
	}
	return &courseAccess{courseRow: *course, Role: collab.Role}, nil
}

func (r *GormRepository) requireEditorAccess(ctx context.Context, db *gorm.DB, courseID, userID uint) (*courseAccess, error) {
	return r.requireCourseAccess(ctx, db, courseID, userID)
}

func (r *GormRepository) requireOwnerAccess(ctx context.Context, db *gorm.DB, courseID, userID uint) (*courseAccess, error) {
	access, err := r.requireCourseAccess(ctx, db, courseID, userID)
	if err != nil {
		return nil, err
	}
	if access.Role != domain.CollaboratorRoleOwner {
		return nil, domain.ErrCourseOwnerOnly
	}
	return access, nil
}

func (r *GormRepository) ensureEditableDraft(ctx context.Context, tx *gorm.DB, courseID, userID uint) (*courseAccess, error) {
	access, err := r.requireEditorAccess(ctx, tx, courseID, userID)
	if err != nil {
		return nil, err
	}
	if access.CurrentDraftVersionID == nil {
		draftID, err := r.createDraftVersion(ctx, tx, &access.courseRow)
		if err != nil {
			return nil, err
		}
		if err := tx.Model(&courseRow{}).Where("id = ?", access.ID).Updates(map[string]any{
			"current_draft_version_id": draftID,
			"updated_at":               timex.NowUnix(),
		}).Error; err != nil {
			return nil, err
		}
		access.CurrentDraftVersionID = &draftID
	}
	version, err := r.loadVersionRow(ctx, tx, *access.CurrentDraftVersionID)
	if err != nil {
		return nil, err
	}
	if version.Status == domain.VersionStatusInReview {
		return nil, domain.ErrCourseDraftInReview
	}
	return access, nil
}

func (r *GormRepository) requireDraftVersion(ctx context.Context, tx *gorm.DB, courseID uint) (*courseRow, *courseVersionRow, error) {
	course, err := r.loadCourse(ctx, tx, courseID)
	if err != nil {
		return nil, nil, err
	}
	if course.CurrentDraftVersionID == nil {
		return nil, nil, domain.ErrCourseDraftRequired
	}
	version, err := r.loadVersionRow(ctx, tx, *course.CurrentDraftVersionID)
	if err != nil {
		return nil, nil, err
	}
	return course, version, nil
}

func (r *GormRepository) loadCourse(ctx context.Context, db *gorm.DB, courseID uint) (*courseRow, error) {
	return loadActiveRow[courseRow](ctx, db, domain.ErrCourseNotFound, "id = ? AND deleted_at IS NULL", courseID)
}

func (r *GormRepository) loadVersionRow(ctx context.Context, db *gorm.DB, versionID uint) (*courseVersionRow, error) {
	return loadActiveRow[courseVersionRow](ctx, db, domain.ErrCourseVersionNotFound, "id = ? AND deleted_at IS NULL", versionID)
}

func (r *GormRepository) loadCollaborators(ctx context.Context, db *gorm.DB, courseID uint) ([]domain.Collaborator, error) {
	q := `
SELECT
    cc.user_id,
    cc.role,
    COALESCE(u.display_name, '') AS display_name,
    COALESCE(u.email, '') AS email,
    COALESCE(u.avatar_file_id::text, '') AS avatar_file_id,
    COALESCE(m.url, '') AS avatar_url
FROM course_collaborators cc
INNER JOIN users u
    ON u.id = cc.user_id AND u.deleted_at IS NULL
LEFT JOIN media_files m
    ON m.id = u.avatar_file_id AND m.deleted_at IS NULL
WHERE cc.course_id = @course_id AND cc.deleted_at IS NULL
ORDER BY CASE WHEN cc.role = 'OWNER' THEN 0 ELSE 1 END, cc.id ASC`
	type row struct {
		UserID       uint   `gorm:"column:user_id"`
		Role         string `gorm:"column:role"`
		DisplayName  string `gorm:"column:display_name"`
		Email        string `gorm:"column:email"`
		AvatarFileID string `gorm:"column:avatar_file_id"`
		AvatarURL    string `gorm:"column:avatar_url"`
	}
	var rows []row
	if err := db.WithContext(ctx).Raw(q, map[string]any{"course_id": courseID}).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Collaborator, len(rows))
	for i, row := range rows {
		out[i] = domain.Collaborator{
			UserID: row.UserID, Role: row.Role, DisplayName: row.DisplayName, Email: row.Email,
			AvatarFileID: row.AvatarFileID, AvatarURL: row.AvatarURL,
		}
	}
	return out, nil
}

func (r *GormRepository) loadSectionsByVersion(ctx context.Context, db *gorm.DB, versionID uint) ([]sectionRow, error) {
	return loadActiveRows[sectionRow](ctx, db, "course_version_id = ? AND deleted_at IS NULL", versionID)
}

func (r *GormRepository) loadLessonsBySection(ctx context.Context, db *gorm.DB, sectionID uint) ([]lessonRow, error) {
	return loadActiveRows[lessonRow](ctx, db, "section_id = ? AND deleted_at IS NULL", sectionID)
}

func (r *GormRepository) loadSubLessonsByLesson(ctx context.Context, db *gorm.DB, lessonID uint) ([]subLessonRow, error) {
	return loadActiveRows[subLessonRow](ctx, db, "lesson_id = ? AND deleted_at IS NULL", lessonID)
}

func (r *GormRepository) loadOutline(ctx context.Context, db *gorm.DB, versionID uint) ([]domain.Section, error) {
	sections, err := r.loadSectionsByVersion(ctx, db, versionID)
	if err != nil {
		return nil, err
	}
	lessons := map[uint][]lessonRow{}
	subLessons := map[uint][]domain.SubLesson{}
	for _, section := range sections {
		rows, err := r.loadLessonsBySection(ctx, db, section.ID)
		if err != nil {
			return nil, err
		}
		lessons[section.ID] = rows
		for _, lesson := range rows {
			subRows, err := r.loadSubLessonsByLesson(ctx, db, lesson.ID)
			if err != nil {
				return nil, err
			}
			list := make([]domain.SubLesson, len(subRows))
			for i := range subRows {
				sub, err := r.loadSubLessonDomain(ctx, db, subRows[i].ID)
				if err != nil {
					return nil, err
				}
				list[i] = *sub
			}
			subLessons[lesson.ID] = list
		}
	}
	out := make([]domain.Section, len(sections))
	for i, sec := range sections {
		out[i] = toSection(&sec)
		lessonRows := lessons[sec.ID]
		out[i].Lessons = make([]domain.Lesson, len(lessonRows))
		for j, lesson := range lessonRows {
			out[i].Lessons[j] = toLesson(&lesson)
			out[i].Lessons[j].SubLessons = subLessons[lesson.ID]
		}
	}
	return out, nil
}

func (r *GormRepository) loadSection(ctx context.Context, db *gorm.DB, sectionID, versionID uint) (*sectionRow, error) {
	return loadActiveRow[sectionRow](ctx, db, domain.ErrCourseNotFound, "id = ? AND course_version_id = ? AND deleted_at IS NULL", sectionID, versionID)
}

func (r *GormRepository) loadLesson(ctx context.Context, db *gorm.DB, lessonID, versionID uint) (*lessonRow, error) {
	return loadActiveRow[lessonRow](ctx, db, domain.ErrCourseNotFound, "id = ? AND course_version_id = ? AND deleted_at IS NULL", lessonID, versionID)
}

func (r *GormRepository) loadSubLesson(ctx context.Context, db *gorm.DB, subLessonID, versionID uint) (*subLessonRow, error) {
	return loadActiveRow[subLessonRow](ctx, db, domain.ErrCourseNotFound, "id = ? AND course_version_id = ? AND deleted_at IS NULL", subLessonID, versionID)
}
