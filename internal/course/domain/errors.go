package domain

import "errors"

var (
	ErrCourseNotFound              = errors.New("course not found")
	ErrCourseVersionNotFound       = errors.New("course version not found")
	ErrCourseDraftRequired         = errors.New("course draft is required")
	ErrCourseDraftInReview         = errors.New("course draft is in review")
	ErrCourseDraftRejectedOnly     = errors.New("only a rejected draft can be reopened")
	ErrCoursePublishedRequired     = errors.New("published course version is required")
	ErrCourseOwnerOnly             = errors.New("only the course owner can perform this action")
	ErrCourseCollaboratorAccess    = errors.New("course collaborator access is required")
	ErrCourseOptimisticLock        = errors.New("course resource was modified by another request; refresh and retry")
	ErrCourseLeaseHeldByOtherUser  = errors.New("course resource is currently locked by another user")
	ErrCourseLeaseTokenInvalid     = errors.New("course lease token is invalid")
	ErrCourseInvalidSubLessonKind  = errors.New("invalid course sub-lesson kind")
	ErrCourseInvalidReviewState    = errors.New("course draft is not in a valid state for this review action")
	ErrCourseInvalidOrdering       = errors.New("ordered stable ids do not match the existing resource set")
	ErrCourseInstructorRequired    = errors.New("course collaborator must be an instructor")
	ErrCourseOwnerCannotBeRemoved  = errors.New("course owner cannot be removed from collaborators")
	ErrCourseEnrollmentNotFound    = errors.New("course enrollment not found")
	ErrCourseProgressVersionAbsent = errors.New("course version for learner progress is not available")
	ErrCourseInvalidSlug              = errors.New("course title must produce a non-empty slug")
	ErrCourseTitleTooShort            = errors.New("course title must contain at least 5 non-whitespace characters")
	ErrCoursePreviewNotAllowedForQuiz = errors.New("quiz lesson items cannot be marked as preview")
)
