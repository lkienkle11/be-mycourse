package domain

import stderrors "errors"

var (
	ErrRejectionReasonRequired      = stderrors.New("rejection_reason is required")
	ErrApplicationNotPending        = stderrors.New("application is not pending")
	ErrApplicationNotResubmittable  = stderrors.New("application cannot be resubmitted")
	ErrApplicationSubmitBlocked     = stderrors.New("instructor application submit is blocked")
	ErrApplicationRejectQuota       = stderrors.New("instructor application rejection quota reached")
	ErrApplicationAlreadyExists     = stderrors.New("instructor application already exists")
	ErrApplicationAlreadyInstructor = stderrors.New("user is already an instructor")
	ErrInvalidApplicationPayload    = stderrors.New("invalid application payload")
	ErrTicketClosed                 = stderrors.New("ticket is closed")
	ErrRosterPlatformStaffUser      = stderrors.New("platform staff cannot be added to instructor roster")
)
