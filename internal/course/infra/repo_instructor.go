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
	sharedslug "mycourse-io-be/internal/shared/slug"
	"mycourse-io-be/internal/shared/timex"
)

// courseSlugCreateRetry bounds the outer retry loop CreateCourse uses when a
// concurrent writer wins the race on a candidate slug (raised from 3 to 5:
// random-suffix collisions can need more attempts under contention than the
// old deterministic numeric scheme did).
const courseSlugCreateRetry = 5

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
		finalSlug, err := resolveCreateSlug(ctx, tx, in)
		if err != nil {
			return err // may be *domain.SlugConflictError — propagated as-is, not wrapped
		}
		course := &courseRow{OwnerUserID: in.ActorUserID, Slug: finalSlug}
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

// resolveCreateSlug implements the create-time slug branch: a non-empty
// in.Slug (manual, already format-validated by the service layer) is checked
// for availability and used as-is, or the whole operation fails with
// *domain.SlugConflictError (no silent auto-suffix on a caller's own
// explicit choice — see specs/course/slug-management/spec.md). An empty
// in.Slug (auto-generate) derives a base from the title via gosimple/slug and
// resolves any collision with a random suffix via sharedslug.RetryWithSuffix.
// Lives in repo_instructor.go (not slug.go) because it orchestrates courseRow
// writes/reads directly; slug.go holds only the smaller, more reusable
// primitives (generateAutoSlugBase, courseSlugAvailable). Every
// RetryWithSuffix call builds its candidate via buildSuffixedCandidate
// (slug.go), which truncates the base to leave room for the suffix within
// domain.MaxSlugLen and errors out instead of producing an invalid
// leading-hyphen candidate if no room remains at all.
func resolveCreateSlug(ctx context.Context, tx *gorm.DB, in domain.CreateCourseInput) (string, error) {
	if in.Slug != "" {
		available, err := courseSlugAvailable(ctx, tx, in.Slug, nil)
		if err != nil {
			return "", err
		}
		if available {
			return in.Slug, nil
		}
		// Treated exactly like any other manual-slug conflict, including on the
		// round trip where the caller resubmits create with this exact
		// recommended value: if a concurrent writer took it in the meantime
		// (astronomically rare given the suffix's keyspace), the resubmission is
		// itself just another manual slug and gets its own fresh
		// SlugConflictError + a new recommendation — no special-cased "never
		// conflict twice" bypass.
		recommended, err := sharedslug.RetryWithSuffix(func(suffix string) (string, bool, error) {
			candidate, err := buildSuffixedCandidate(in.Slug, suffix)
			if err != nil {
				return "", false, err
			}
			ok, err := courseSlugAvailable(ctx, tx, candidate, nil)
			return candidate, ok, err
		})
		if err != nil {
			return "", err
		}
		return "", &domain.SlugConflictError{RecommendedSlug: recommended}
	}
	base, mustSuffix := generateAutoSlugBase(in.Title) // already truncated to domain.MaxSlugLen internally
	if !mustSuffix {
		available, err := courseSlugAvailable(ctx, tx, base, nil)
		if err != nil {
			return "", err
		}
		if available {
			return base, nil
		}
	}
	// write-attempt-based collision resolution below relies on the OUTER
	// CreateCourse retry loop + isCourseSlugDuplicateKey to catch a race this
	// SELECT-based accept callback missed — this accept is only the
	// UX-fast-path, not the safety net (see the "Slug uniqueness is enforced
	// at the database" spec requirement).
	return sharedslug.RetryWithSuffix(func(suffix string) (string, bool, error) {
		candidate, err := buildSuffixedCandidate(base, suffix)
		if err != nil {
			return "", false, err
		}
		ok, err := courseSlugAvailable(ctx, tx, candidate, nil)
		return candidate, ok, err
	})
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
		return r.applyUpdateSlugWithRetry(ctx, tx, access, in.Slug)
	})
	if stderrors.Is(err, apperrors.ErrMediaOptimisticLock) {
		return nil, domain.ErrCourseOptimisticLock
	}
	if err != nil {
		return nil, err
	}
	return r.loadCourseDetail(ctx, r.db, courseID, actorUserID, true, true)
}

// applyUpdateSlugWithRetry closes a pre-existing race-safety gap:
// UpdateBasicInfo previously resolved a new slug and wrote it exactly once
// with no retry-on-duplicate-key, unlike CreateCourse. Scoped to only the
// slug write sub-step (not the whole surrounding transaction) so a slug-only
// collision retry does not re-validate/re-write the unrelated metadata fields
// already committed earlier in the same transaction.
func (r *GormRepository) applyUpdateSlugWithRetry(ctx context.Context, tx *gorm.DB, access *courseAccess, newSlugInput *string) error {
	if newSlugInput == nil {
		return nil // omitted — do not touch courses.slug at all
	}
	newSlug := *newSlugInput
	if newSlug == access.Slug {
		return nil // no-op: identical to current slug, never a self-conflict
	}
	const updateSlugRetry = 5 // mirrors courseSlugCreateRetry; same tuning rationale
	var lastErr error
	for range updateSlugRetry {
		finalSlug, err := resolveUpdateSlug(ctx, tx, newSlug, access.ID)
		if err != nil {
			return err
		}
		writeErr := tx.Model(&courseRow{}).Where("id = ? AND deleted_at IS NULL", access.ID).
			Updates(map[string]any{"slug": finalSlug, "updated_at": timex.NowUnix()}).Error
		if writeErr == nil {
			return nil
		}
		if !isCourseSlugDuplicateKey(writeErr) {
			return writeErr
		}
		lastErr = writeErr // a concurrent writer won the race on finalSlug — retry with a fresh suffix
	}
	return lastErr
}

// resolveUpdateSlug implements the update-time slug branch — no
// *domain.SlugConflictError path exists here (update auto-resolves directly,
// unlike create's manual-conflict-returns-a-recommendation behavior).
// Builds via buildSuffixedCandidate for the same VARCHAR(255) overflow
// guard as resolveCreateSlug.
func resolveUpdateSlug(ctx context.Context, tx *gorm.DB, newSlug string, excludeCourseID string) (string, error) {
	available, err := courseSlugAvailable(ctx, tx, newSlug, &excludeCourseID)
	if err != nil {
		return "", err
	}
	if available {
		return newSlug, nil
	}
	return sharedslug.RetryWithSuffix(func(suffix string) (string, bool, error) {
		candidate, err := buildSuffixedCandidate(newSlug, suffix)
		if err != nil {
			return "", false, err
		}
		ok, err := courseSlugAvailable(ctx, tx, candidate, &excludeCourseID)
		return candidate, ok, err
	})
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
