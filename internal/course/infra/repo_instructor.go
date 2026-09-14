package infra

import (
	"context"
	stderrors "errors"
	"strings"

	"gorm.io/gorm"

	authzdomain "mycourse-io-be/internal/authorization/domain"
	courseapp "mycourse-io-be/internal/course/application"
	"mycourse-io-be/internal/course/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/gormx"
	"mycourse-io-be/internal/shared/timex"
)

// ListEditableCourses reads the principal's own active course-scoped role bindings through
// RoleBindingService.List (the one sanctioned read path for authorization_role_bindings — never
// a direct query against that table), then filters/labels courses by that small, already-known
// id set instead of joining the table live.
func (r *GormRepository) ListEditableCourses(ctx context.Context, userID string) ([]domain.CourseListItem, error) {
	bindings, err := r.roleBindings.List(ctx, authzdomain.RoleBindingQuery{
		PrincipalUserIDs: []string{userID},
		Resource:         authzdomain.Resource{Type: courseapp.ResourceTypeCourse},
	})
	if err != nil {
		return nil, err
	}
	editorCourseIDs := make([]string, len(bindings))
	for i, b := range bindings {
		editorCourseIDs[i] = b.Resource.ID
	}

	membershipClause := "c.owner_user_id = @user_id"
	roleClause := "CASE WHEN c.owner_user_id = @user_id THEN 'OWNER' ELSE '' END"
	args := map[string]any{"user_id": userID}
	if len(editorCourseIDs) > 0 {
		membershipClause += " OR c.id::text IN @editor_course_ids"
		roleClause = "CASE WHEN c.owner_user_id = @user_id THEN 'OWNER' ELSE 'EDITOR' END"
		args["editor_course_ids"] = editorCourseIDs
	}

	q := `
SELECT
    ` + courseListBaseColumns + `,
    ` + roleClause + ` AS role,
    COALESCE(dv.title, pv.title, '') AS title,
    COALESCE(dv.status, pv.status, '') AS review_status,
    COALESCE(dv.version_no, pv.version_no, 0) AS version_no,
    (c.current_published_version_id IS NOT NULL) AS has_published,
    (c.current_draft_version_id IS NOT NULL) AS has_draft,
    COALESCE(dv.thumbnail_file_id::text, pv.thumbnail_file_id::text, '') AS thumbnail_file_id,
    COALESCE(dm.url, pm.url, '') AS thumbnail_url,
    COALESCE(dv.preview_video_file_id::text, pv.preview_video_file_id::text, '') AS preview_video_file_id
FROM courses c
LEFT JOIN course_versions dv
    ON dv.id = c.current_draft_version_id AND dv.deleted_at IS NULL
LEFT JOIN course_versions pv
    ON pv.id = c.current_published_version_id AND pv.deleted_at IS NULL
LEFT JOIN media_files dm
    ON dm.id = dv.thumbnail_file_id AND dm.deleted_at IS NULL
LEFT JOIN media_files pm
    ON pm.id = pv.thumbnail_file_id AND pm.deleted_at IS NULL
WHERE c.deleted_at IS NULL
  AND c.trashed_at IS NULL
  AND (` + membershipClause + `)
ORDER BY c.id DESC`

	var rows []courseListScanRow
	if err := r.db.WithContext(ctx).Raw(q, args).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CourseListItem, len(rows))
	for i, row := range rows {
		out[i] = toCourseListItem(&row)
	}
	return out, nil
}

func (r *GormRepository) CreateCourse(ctx context.Context, in domain.CreateCourseInput) (*domain.CourseDetail, error) {
	var (
		detail *domain.CourseDetail
		err    error
	)
	for range courseSlugCreateRetry {
		detail, err = r.createCourseOnce(ctx, in)
		if err == nil || !isCourseSlugDuplicateKey(err) {
			return detail, err
		}
	}
	return nil, err
}

func (r *GormRepository) createCourseOnce(ctx context.Context, in domain.CreateCourseInput) (*domain.CourseDetail, error) {
	var courseID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		slug, err := ensureUniqueCourseSlug(ctx, tx, in.Slug, nil)
		if err != nil {
			return err
		}
		course := &courseRow{OwnerUserID: in.ActorUserID, Slug: slug}
		if err := touchCreateCourseEntity(ctx, tx, &course.CreatedAt, &course.UpdatedAt, course); err != nil {
			return err
		}
		courseID = course.ID

		version := &courseVersionRow{
			CourseID: course.ID, VersionNo: 1, Status: domain.VersionStatusDraft,
			Title: strings.TrimSpace(in.Title), RowVersion: 1,
		}
		if err := touchCreateCourseEntity(ctx, tx, &version.CreatedAt, &version.UpdatedAt, version); err != nil {
			return err
		}
		// No course_collaborators row for the owner: ownership is courses.owner_user_id
		// itself, synthesized by CoursePolicyProvider and read directly by
		// collaboratorsSelectSQL/instructorCandidatesBaseSQL/ListEditableCourses — never a
		// stored role binding or collaborator row.
		return tx.Model(&courseRow{}).Where("id = ?", course.ID).
			Updates(map[string]any{"current_draft_version_id": version.ID, "updated_at": timex.NowUnix()}).Error
	})
	if err != nil {
		return nil, err
	}
	return r.loadCourseDetail(ctx, r.db, courseID, in.ActorUserID, true, true)
}

func (r *GormRepository) GetCourseDetail(ctx context.Context, courseID string, userID string, includeDraft bool, includeOutline bool) (*domain.CourseDetail, error) {
	return r.loadCourseDetail(ctx, r.db.WithContext(ctx), courseID, userID, includeDraft, includeOutline)
}

func (r *GormRepository) PrepareDraft(ctx context.Context, courseID string, actorUserID string) (*domain.CourseDetail, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.requireOwnerAccess(ctx, tx, courseID, actorUserID, courseapp.ActionCourseDraftPrepare)
		if err != nil {
			return err
		}
		if access.CurrentDraftVersionID != nil {
			return nil
		}
		draftID, err := r.createDraftVersion(ctx, tx, &access.courseRow)
		if err != nil {
			return err
		}
		return tx.Model(&courseRow{}).Where("id = ?", courseID).
			Updates(map[string]any{"current_draft_version_id": draftID, "updated_at": timex.NowUnix()}).Error
	})
	if err != nil {
		return nil, err
	}
	return r.loadCourseDetail(ctx, r.db, courseID, actorUserID, true, true)
}

func (r *GormRepository) UpdateBasicInfo(ctx context.Context, courseID string, actorUserID string, in domain.UpdateBasicInfoInput) (*domain.CourseDetail, error) {
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
		if in.Slug != nil {
			slug, err := ensureUniqueCourseSlug(ctx, tx, *in.Slug, &access.ID)
			if err != nil {
				return err
			}
			if err := tx.Model(&courseRow{}).Where("id = ? AND deleted_at IS NULL", access.ID).
				Updates(map[string]any{"slug": slug, "updated_at": timex.NowUnix()}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if stderrors.Is(err, apperrors.ErrMediaOptimisticLock) {
		return nil, domain.ErrCourseOptimisticLock
	}
	if err != nil {
		return nil, err
	}
	return r.loadCourseDetail(ctx, r.db, courseID, actorUserID, true, true)
}

// DeleteCourse revokes every role binding on the course (via RevokeResource, as part of the same
// course-side transaction via gormx.WithTx) only when softDeleteCourseTree actually ran (a hard
// delete, not a move-to-trash — a trashed-but-not-yet-permanently-deleted course keeps its
// collaborators).
func (r *GormRepository) DeleteCourse(ctx context.Context, courseID string, actorUserID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.requireOwnerAccess(ctx, tx, courseID, actorUserID, courseapp.ActionCourseLifecycleDelete)
		if err != nil {
			return err
		}
		published, draft, err := r.loadPublishedAndDraftVersions(ctx, tx, &access.courseRow)
		if err != nil {
			return err
		}
		if courseEligibleForTrash(published, draft) {
			now := timex.NowUnix()
			return tx.Model(&courseRow{}).Where("id = ? AND deleted_at IS NULL", access.ID).
				Updates(map[string]any{"trashed_at": now, "updated_at": now}).Error
		}
		if err := r.softDeleteCourseTree(ctx, tx, access.ID); err != nil {
			return err
		}
		return r.roleBindings.RevokeResource(gormx.WithTx(ctx, tx), authzdomain.Resource{Type: courseapp.ResourceTypeCourse, ID: access.ID})
	})
}

// RemoveCollaborator's access check and the role-binding revoke run in one course-side
// transaction: gormx.WithTx makes internal/authorization's GormGrantRepository join tx instead
// of running a separately committed write. The revoke leaves RoleName empty so it ends every
// active role binding this user holds on the course, not only EDITOR — matching this method's
// pre-role-gate behavior of deleting the collaborator row by (course_id, user_id) regardless of
// its stored role.
func (r *GormRepository) RemoveCollaborator(ctx context.Context, courseID string, actorUserID, userID string) ([]domain.Collaborator, error) {
	var resolvedCourseID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.requireOwnerAccess(ctx, tx, courseID, actorUserID, courseapp.ActionCourseCollaboratorsManage)
		if err != nil {
			return err
		}
		if access.OwnerUserID == userID {
			return domain.ErrCourseOwnerCannotBeRemoved
		}
		resolvedCourseID = access.ID
		return r.roleBindings.Revoke(gormx.WithTx(ctx, tx), authzdomain.RoleBindingRevocation{
			Issuer:           courseAuthorizationPrincipal(ctx, actorUserID),
			PrincipalUserIDs: []string{userID},
			Resource:         authzdomain.Resource{Type: courseapp.ResourceTypeCourse, ID: resolvedCourseID},
			Context:          authzdomain.EvaluationContext{DomainFacts: courseapp.CourseAuthorizationFacts{IsOwner: true}},
		})
	})
	if err != nil {
		return nil, err
	}
	return r.loadCollaborators(ctx, r.db, resolvedCourseID)
}
