package domain

import (
	stderrors "errors"

	"mycourse-io-be/internal/shared/constants"
)

// ErrDuplicateCertificate is returned when two certificate rows in one payload
// share the same credential_url, certificate_file_id, or title|issuer|issued_year
// composite. Declared standalone (not in the var block below) to keep the
// instructor sentinel-error block structurally distinct from auth/domain/errors.go.
// Sentinel text is shared with the errcode default message via constants.MsgDuplicateCertificate.
var ErrDuplicateCertificate = stderrors.New(constants.MsgDuplicateCertificate)

var (
	ErrRejectionReasonRequired      = stderrors.New("rejection_reason is required")
	ErrApplicationNotPending        = stderrors.New("application is not pending")
	ErrApplicationNotResubmittable  = stderrors.New("application cannot be resubmitted")
	ErrApplicationSubmitBlocked     = stderrors.New("instructor application submit is blocked")
	ErrApplicationRejectQuota       = stderrors.New("instructor application rejection quota reached")
	ErrApplicationAlreadyExists     = stderrors.New("instructor application already exists")
	ErrApplicationAlreadyInstructor = stderrors.New("user is already an instructor")
	ErrInvalidApplicationPayload    = stderrors.New("invalid application payload")
	ErrApplicationContactNotAllowed = stderrors.New("instructor application contact not allowed")
	ErrTicketClosed                 = stderrors.New("ticket is closed")
	ErrRosterPlatformStaffUser      = stderrors.New("platform staff cannot be added to instructor roster")
)
