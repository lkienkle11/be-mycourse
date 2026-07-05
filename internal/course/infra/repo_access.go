package infra

import (
	"context"
	"strings"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/timex"
)

func (r *GormRepository) loadCourseDetail(ctx context.Context, db *gorm.DB, courseID string, userID string, includeDraft bool, includeOutline bool) (*domain.CourseDetail, error) {
	access, err := r.requireCourseAccess(ctx, db, courseID, userID)
	if err != nil {
		return nil, err
	}
	liveRow, draftRow, outlineVersionID, err := loadPublishedDraftVersionRows(ctx, r, db, access, includeDraft)
	if err != nil {
		return nil, err
	}
	live, draft, collabs, outline, lastRejectionReason, err := r.loadCourseDetailParts(ctx, db, courseID, liveRow, draftRow, outlineVersionID, includeOutline)
	if err != nil {
		return nil, err
	}
	return &domain.CourseDetail{
		Course:              toCourse(&access.courseRow),
		CollaboratorRole:    access.Role,
		LiveVersion:         live,
		DraftVersion:        draft,
		LastRejectionReason: lastRejectionReason,
		Collaborators:       collabs,
		Outline:             outline,
	}, nil
}

func loadPublishedDraftVersionRows(
	ctx context.Context,
	r *GormRepository,
	db *gorm.DB,
	access *courseAccess,
	includeDraft bool,
) (liveRow, draftRow *courseVersionRow, outlineVersionID string, err error) {
	group, gctx := errgroup.WithContext(ctx)
	if access.CurrentPublishedVersionID != nil {
		versionID := *access.CurrentPublishedVersionID
		group.Go(func() error {
			row, loadErr := r.loadVersionRow(gctx, parallelReadDB(db), versionID)
			if loadErr != nil {
				return loadErr
			}
			liveRow = row
			return nil
		})
	}
	if includeDraft && access.CurrentDraftVersionID != nil {
		versionID := *access.CurrentDraftVersionID
		group.Go(func() error {
			row, loadErr := r.loadVersionRow(gctx, parallelReadDB(db), versionID)
			if loadErr != nil {
				return loadErr
			}
			draftRow = row
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, nil, "", err
	}
	if draftRow != nil {
		outlineVersionID = draftRow.ID
	} else if liveRow != nil {
		outlineVersionID = liveRow.ID
	}
	return liveRow, draftRow, outlineVersionID, nil
}

func (r *GormRepository) loadCourseDetailParts(
	ctx context.Context,
	db *gorm.DB,
	courseID string,
	liveRow, draftRow *courseVersionRow,
	outlineVersionID string,
	includeOutline bool,
) (live, draft *domain.CourseVersion, collabs []domain.Collaborator, outline []domain.Section, lastRejectionReason string, err error) {
	versionRows := collectCourseVersionRows(liveRow, draftRow)

	var assetsByVersion map[string]*courseVersionAssets
	group, gctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		assets, loadErr := r.loadCourseVersionAssetsBatch(gctx, parallelReadDB(db), versionRows)
		if loadErr != nil {
			return loadErr
		}
		assetsByVersion = assets
		return nil
	})
	group.Go(func() error {
		rows, loadErr := r.loadCollaborators(gctx, parallelReadDB(db), courseID)
		if loadErr != nil {
			return loadErr
		}
		collabs = rows
		return nil
	})
	if includeOutline && outlineVersionID != "" {
		versionID := outlineVersionID
		group.Go(func() error {
			rows, loadErr := r.loadOutline(gctx, parallelReadDB(db), versionID)
			if loadErr != nil {
				return loadErr
			}
			outline = rows
			return nil
		})
	}
	r.addLastRejectionReasonTask(gctx, group, db, draftRow, &lastRejectionReason)
	if err := group.Wait(); err != nil {
		return nil, nil, nil, nil, "", err
	}
	live, draft = mapCourseVersionsFromAssets(liveRow, draftRow, assetsByVersion)
	if outline == nil {
		outline = []domain.Section{}
	}
	return live, draft, collabs, outline, lastRejectionReason, nil
}

func collectCourseVersionRows(liveRow, draftRow *courseVersionRow) []*courseVersionRow {
	versionRows := make([]*courseVersionRow, 0, 2)
	if liveRow != nil {
		versionRows = append(versionRows, liveRow)
	}
	if draftRow != nil {
		versionRows = append(versionRows, draftRow)
	}
	return versionRows
}

func mapCourseVersionsFromAssets(
	liveRow, draftRow *courseVersionRow,
	assetsByVersion map[string]*courseVersionAssets,
) (live, draft *domain.CourseVersion) {
	if liveRow != nil {
		live = mapCourseVersionRow(liveRow, assetsByVersion[liveRow.ID])
	}
	if draftRow != nil {
		draft = mapCourseVersionRow(draftRow, assetsByVersion[draftRow.ID])
	}
	return live, draft
}

func (r *GormRepository) addLastRejectionReasonTask(
	ctx context.Context,
	group *errgroup.Group,
	db *gorm.DB,
	draftRow *courseVersionRow,
	lastRejectionReason *string,
) {
	if draftRow == nil || draftRow.BasedOnVersionID == nil {
		return
	}
	basedOnVersionID := *draftRow.BasedOnVersionID
	group.Go(func() error {
		basedOn, loadErr := r.loadVersionRow(ctx, parallelReadDB(db), basedOnVersionID)
		if loadErr != nil {
			return loadErr
		}
		if basedOn.Status == domain.VersionStatusRejected {
			*lastRejectionReason = strings.TrimSpace(basedOn.RejectionReason)
		}
		return nil
	})
}

func (r *GormRepository) loadLearnerCourseDetail(ctx context.Context, db *gorm.DB, courseID string, userID string) (*domain.CourseDetail, error) {
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

func (r *GormRepository) loadProgress(ctx context.Context, db *gorm.DB, courseID string, userID string) (*domain.CourseProgress, error) {
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

func (r *GormRepository) requireEnrollment(ctx context.Context, db *gorm.DB, courseID string, userID string) (*enrollmentRow, error) {
	return loadActiveRow[enrollmentRow](ctx, db, domain.ErrCourseEnrollmentNotFound, "course_id = ? AND user_id = ? AND deleted_at IS NULL", courseID, userID)
}

func (r *GormRepository) requireCourseAccess(ctx context.Context, db *gorm.DB, courseID string, userID string) (*courseAccess, error) {
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

func (r *GormRepository) requireEditorAccess(ctx context.Context, db *gorm.DB, courseID string, userID string) (*courseAccess, error) {
	return r.requireCourseAccess(ctx, db, courseID, userID)
}

func (r *GormRepository) requireOwnerAccess(ctx context.Context, db *gorm.DB, courseID string, userID string) (*courseAccess, error) {
	access, err := r.requireCourseAccess(ctx, db, courseID, userID)
	if err != nil {
		return nil, err
	}
	if access.Role != domain.CollaboratorRoleOwner {
		return nil, domain.ErrCourseOwnerOnly
	}
	return access, nil
}

func (r *GormRepository) ensureEditableDraft(ctx context.Context, tx *gorm.DB, courseID string, userID string) (*courseAccess, error) {
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
	if version.Status == domain.VersionStatusRejected {
		return nil, domain.ErrCourseDraftRejectedOnly
	}
	return access, nil
}

func (r *GormRepository) requireDraftVersion(ctx context.Context, tx *gorm.DB, courseID string) (*courseRow, *courseVersionRow, error) {
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

func (r *GormRepository) loadCourse(ctx context.Context, db *gorm.DB, courseID string) (*courseRow, error) {
	return loadActiveRow[courseRow](ctx, db, domain.ErrCourseNotFound, "id = ? AND deleted_at IS NULL AND trashed_at IS NULL", courseID)
}

func (r *GormRepository) loadTrashedCourse(ctx context.Context, db *gorm.DB, courseID string) (*courseRow, error) {
	return loadActiveRow[courseRow](ctx, db, domain.ErrCourseNotTrashed, "id = ? AND deleted_at IS NULL AND trashed_at IS NOT NULL", courseID)
}

func (r *GormRepository) loadCourseForAdmin(ctx context.Context, db *gorm.DB, courseID string) (*courseRow, error) {
	return loadActiveRow[courseRow](ctx, db, domain.ErrCourseNotFound, "id = ? AND deleted_at IS NULL", courseID)
}

func (r *GormRepository) loadVersionRow(ctx context.Context, db *gorm.DB, versionID string) (*courseVersionRow, error) {
	return loadActiveRow[courseVersionRow](ctx, db, domain.ErrCourseVersionNotFound, "id = ? AND deleted_at IS NULL", versionID)
}

func (r *GormRepository) loadCollaborators(ctx context.Context, db *gorm.DB, courseID string) ([]domain.Collaborator, error) {
	q := collaboratorsFilteredSelectSQL() + collaboratorOrderSQL()
	var rows []collaboratorScanRow
	if err := db.WithContext(ctx).Raw(q, map[string]any{
		"course_id": courseID,
		"now":       timex.NowUnix(),
	}).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return scanRowsToCollaborators(rows), nil
}

func (r *GormRepository) loadSectionsByVersion(ctx context.Context, db *gorm.DB, versionID string) ([]sectionRow, error) {
	return loadActiveRows[sectionRow](ctx, db, "course_version_id = ? AND deleted_at IS NULL", versionID)
}

func (r *GormRepository) loadLessonsBySection(ctx context.Context, db *gorm.DB, sectionID string) ([]lessonRow, error) {
	return loadActiveRows[lessonRow](ctx, db, "section_id = ? AND deleted_at IS NULL", sectionID)
}

func (r *GormRepository) loadSubLessonsByLesson(ctx context.Context, db *gorm.DB, lessonID string) ([]subLessonRow, error) {
	return loadActiveRows[subLessonRow](ctx, db, "lesson_id = ? AND deleted_at IS NULL", lessonID)
}

func (r *GormRepository) loadOutline(ctx context.Context, db *gorm.DB, versionID string) ([]domain.Section, error) {
	sections, lessonRows, subRows, err := r.loadOutlineTreeRows(ctx, db, versionID)
	if err != nil {
		return nil, err
	}
	subMap, err := r.batchHydrateSubLessons(ctx, db, subRows, nil)
	if err != nil {
		return nil, err
	}
	out, err := assembleOutlineSections(sections, lessonRows, subRows, subMap)
	if err != nil {
		return nil, err
	}
	videoIDs := collectVideoMediaFileIDs(out)
	videoMediaMs, err := r.batchMediaDurationMs(ctx, db, videoIDs)
	if err != nil {
		return nil, err
	}
	applyOutlineEstimatedDurations(out, videoMediaMs)
	return out, nil
}

// loadOutlineSequential loads the full outline without parallel DB reads.
// Use inside an open transaction — errgroup + parallelReadDB causes conn busy on pgx.
func (r *GormRepository) loadOutlineSequential(ctx context.Context, db *gorm.DB, versionID string) ([]domain.Section, error) {
	sections, err := r.loadSectionsByVersion(ctx, db, versionID)
	if err != nil {
		return nil, err
	}
	lessonRows, err := loadActiveRows[lessonRow](ctx, db, "course_version_id = ? AND deleted_at IS NULL", versionID)
	if err != nil {
		return nil, err
	}
	subRows, err := loadActiveRows[subLessonRow](ctx, db, "course_version_id = ? AND deleted_at IS NULL", versionID)
	if err != nil {
		return nil, err
	}
	videoMediaMs := make(map[string]int64)
	subMap, err := r.batchHydrateSubLessons(ctx, db, subRows, videoMediaMs)
	if err != nil {
		return nil, err
	}
	out, err := assembleOutlineSections(sections, lessonRows, subRows, subMap)
	if err != nil {
		return nil, err
	}
	videoIDs := collectVideoMediaFileIDs(out)
	if len(videoIDs) > 0 {
		loaded, err := r.batchMediaDurationMs(ctx, db, videoIDs)
		if err != nil {
			return nil, err
		}
		for id, ms := range loaded {
			videoMediaMs[id] = ms
		}
	}
	applyOutlineEstimatedDurations(out, videoMediaMs)
	return out, nil
}

func (r *GormRepository) loadOutlineTreeRows(
	ctx context.Context,
	db *gorm.DB,
	versionID string,
) (sections []sectionRow, lessonRows []lessonRow, subRows []subLessonRow, err error) {
	group, gctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		rows, loadErr := r.loadSectionsByVersion(gctx, parallelReadDB(db), versionID)
		if loadErr != nil {
			return loadErr
		}
		sections = rows
		return nil
	})
	group.Go(func() error {
		rows, loadErr := loadActiveRows[lessonRow](gctx, parallelReadDB(db), "course_version_id = ? AND deleted_at IS NULL", versionID)
		if loadErr != nil {
			return loadErr
		}
		lessonRows = rows
		return nil
	})
	group.Go(func() error {
		rows, loadErr := loadActiveRows[subLessonRow](gctx, parallelReadDB(db), "course_version_id = ? AND deleted_at IS NULL", versionID)
		if loadErr != nil {
			return loadErr
		}
		subRows = rows
		return nil
	})
	if err := group.Wait(); err != nil {
		return nil, nil, nil, err
	}
	return sections, lessonRows, subRows, nil
}

func assembleOutlineSections(
	sections []sectionRow,
	lessonRows []lessonRow,
	subRows []subLessonRow,
	subMap map[string]domain.SubLesson,
) ([]domain.Section, error) {
	lessonsBySection := make(map[string][]lessonRow, len(sections))
	for _, lesson := range lessonRows {
		lessonsBySection[lesson.SectionID] = append(lessonsBySection[lesson.SectionID], lesson)
	}
	subLessonsByLesson := make(map[string][]domain.SubLesson, len(lessonRows))
	for _, sub := range subRows {
		hydrated, ok := subMap[sub.ID]
		if !ok {
			return nil, domain.ErrCourseNotFound
		}
		subLessonsByLesson[sub.LessonID] = append(subLessonsByLesson[sub.LessonID], hydrated)
	}
	out := make([]domain.Section, len(sections))
	for i, sec := range sections {
		out[i] = toSection(&sec)
		sectionLessons := lessonsBySection[sec.ID]
		out[i].Lessons = make([]domain.Lesson, len(sectionLessons))
		for j, lesson := range sectionLessons {
			out[i].Lessons[j] = toLesson(&lesson)
			out[i].Lessons[j].SubLessons = subLessonsByLesson[lesson.ID]
			if out[i].Lessons[j].SubLessons == nil {
				out[i].Lessons[j].SubLessons = []domain.SubLesson{}
			}
		}
	}
	return out, nil
}

func (r *GormRepository) loadSection(ctx context.Context, db *gorm.DB, sectionID, versionID string) (*sectionRow, error) {
	return loadActiveRow[sectionRow](ctx, db, domain.ErrCourseNotFound, "id = ? AND course_version_id = ? AND deleted_at IS NULL", sectionID, versionID)
}

func (r *GormRepository) loadLesson(ctx context.Context, db *gorm.DB, lessonID, versionID string) (*lessonRow, error) {
	return loadActiveRow[lessonRow](ctx, db, domain.ErrCourseNotFound, "id = ? AND course_version_id = ? AND deleted_at IS NULL", lessonID, versionID)
}

func (r *GormRepository) loadSubLesson(ctx context.Context, db *gorm.DB, subLessonID, versionID string) (*subLessonRow, error) {
	return loadActiveRow[subLessonRow](ctx, db, domain.ErrCourseNotFound, "id = ? AND course_version_id = ? AND deleted_at IS NULL", subLessonID, versionID)
}
