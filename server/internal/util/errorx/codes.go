package errorx

// Error codes (see docs/error-codes.md). Stable identifiers.
const (
	// REQ_*
	REQBadRequest           = "REQ_BAD_REQUEST"
	REQValidationFailed     = "REQ_VALIDATION_FAILED"
	REQMissingField         = "REQ_MISSING_FIELD"
	REQInvalidField         = "REQ_INVALID_FIELD"
	REQUnsupportedOperation = "REQ_UNSUPPORTED_OPERATION"
	REQTooLarge             = "REQ_TOO_LARGE"

	// DB_*
	DBError               = "DB_ERROR"
	DBMigrationFailed     = "DB_MIGRATION_FAILED"
	DBConstraintViolation = "DB_CONSTRAINT_VIOLATION"
	DBNotFound            = "DB_NOT_FOUND"
	DBTxFailed            = "DB_TX_FAILED"

	// SUB_*
	SUBNotFound           = "SUB_NOT_FOUND"
	SUBDisabled           = "SUB_DISABLED"
	SUBInvalidURL         = "SUB_INVALID_URL"
	SUBFetchFailed        = "SUB_FETCH_FAILED"
	SUBFetchTimeout       = "SUB_FETCH_TIMEOUT"
	SUBHTTPStatusError    = "SUB_HTTP_STATUS_ERROR"
	SUBResponseTooLarge   = "SUB_RESPONSE_TOO_LARGE"
	SUBParseFailed        = "SUB_PARSE_FAILED"
	SUBFormatUnsupported  = "SUB_FORMAT_UNSUPPORTED"
	SUBEmptyOutbounds     = "SUB_EMPTY_OUTBOUNDS"
	SUBReplaceNodesFailed = "SUB_REPLACE_NODES_FAILED"

	// NODE_*
	NODENotFound        = "NODE_NOT_FOUND"
	NODETagConflict     = "NODE_TAG_CONFLICT"
	NODEInvalidOutbound = "NODE_INVALID_OUTBOUND"
	NODEUpdateFailed    = "NODE_UPDATE_FAILED"
	NODEListFailed      = "NODE_LIST_FAILED"

	// CFG_*
	CFGBuildFailed    = "CFG_BUILD_FAILED"
	CFGNoEnabledNodes = "CFG_NO_ENABLED_NODES"
	CFGJSONInvalid    = "CFG_JSON_INVALID"
	CFGWriteFailed    = "CFG_WRITE_FAILED"
	CFGBackupFailed   = "CFG_BACKUP_FAILED"
	CFGRollbackFailed = "CFG_ROLLBACK_FAILED"
	CFGCheckFailed    = "CFG_CHECK_FAILED"

	// RT_*
	RTRestartFailed = "RT_RESTART_FAILED"
	RTStartFailed   = "RT_START_FAILED"
	RTStopFailed    = "RT_STOP_FAILED"
	RTStatusFailed  = "RT_STATUS_FAILED"

	// JOB_*
	JOBReloadInProgress  = "JOB_RELOAD_IN_PROGRESS"
	JOBRefreshInProgress = "JOB_REFRESH_IN_PROGRESS"
	JOBSchedulerFailed   = "JOB_SCHEDULER_FAILED"
	JOBRateLimited       = "JOB_RATE_LIMITED"

	// Fallback
	InternalError  = "INTERNAL_ERROR"
	NotImplemented = "NOT_IMPLEMENTED"
)
