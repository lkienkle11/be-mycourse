package domain

import stderrors "errors"

var (
	ErrRejectionReasonRequired = stderrors.New("rejection_reason is required")
	ErrApplicationNotPending   = stderrors.New("application is not pending")
	ErrTicketClosed            = stderrors.New("ticket is closed")
)
