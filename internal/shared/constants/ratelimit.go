package constants

// APPCLI fixed-window rate limits (shared across the CLI_* operations below).
const (
	CLIAttempts = 5
	CLIMinutes  = 3

	CLIOpSystemLogin  = "CLI_SYSTEM_LOGIN"
	CLIOpRegister     = "CLI_REGISTER_NEW_SYSTEM_USER"
	CLIOpLegacyImport = "CLI_IMPORT_LEGACY_DATA"

	CLIOpBackfillUserSlugs = "CLI_BACKFILL_USER_SLUGS"
)
