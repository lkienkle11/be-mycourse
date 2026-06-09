package constants

// APPCLI fixed-window rate limits (shared across CLI_SYSTEM_LOGIN and CLI_REGISTER_NEW_SYSTEM_USER).
const (
	CLIAttempts = 5
	CLIMinutes  = 3

	CLIOpSystemLogin  = "CLI_SYSTEM_LOGIN"
	CLIOpRegister     = "CLI_REGISTER_NEW_SYSTEM_USER"
	CLIOpLegacyImport = "CLI_IMPORT_LEGACY_DATA"
)
