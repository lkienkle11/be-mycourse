package constants

import "time"

// MaxRegisterConfirmationEmailsLifetime caps successful Brevo confirmation emails per pending user.
// When already reached, the next register/resend attempt deletes the row and returns errcode 4009.
const MaxRegisterConfirmationEmailsLifetime = 15

// MaxRegisterConfirmationEmailsPerWindow is the Redis sliding-window cap per users.id.
const MaxRegisterConfirmationEmailsPerWindow = 5

// RegisterConfirmationEmailWindow is the sliding window length for per-window cap.
const RegisterConfirmationEmailWindow = 4 * time.Hour
