package errorx

import "net/http"

// AppError is the standard API error with code, message, details.
type AppError struct {
	Code    string
	Message string
	Details map[string]any
}

func (e *AppError) Error() string { return e.Message }

// New creates an AppError.
func New(code, message string) *AppError {
	return &AppError{Code: code, Message: message, Details: nil}
}

// WithDetails sets details and returns self for chaining.
func (e *AppError) WithDetails(d map[string]any) *AppError {
	e.Details = d
	return e
}

// HTTPStatus returns suggested HTTP status for the code (see docs/error-codes.md).
func (e *AppError) HTTPStatus() int {
	switch {
	case e.Code == REQBadRequest || e.Code == REQValidationFailed || e.Code == REQMissingField ||
		e.Code == REQInvalidField || e.Code == REQUnsupportedOperation || e.Code == SUBInvalidURL ||
		e.Code == SUBParseFailed || e.Code == SUBFormatUnsupported || e.Code == SUBEmptyOutbounds ||
		e.Code == NODEInvalidOutbound || e.Code == RTModeUnsupported:
		return http.StatusBadRequest
	case e.Code == REQTooLarge || e.Code == SUBResponseTooLarge:
		return http.StatusRequestEntityTooLarge
	case e.Code == DBNotFound || e.Code == SUBNotFound || e.Code == NODENotFound:
		return http.StatusNotFound
	case e.Code == DBConstraintViolation || e.Code == SUBDisabled || e.Code == NODETagConflict ||
		e.Code == CFGNoEnabledNodes || e.Code == JOBReloadInProgress || e.Code == JOBRefreshInProgress:
		return http.StatusConflict
	case e.Code == JOBRateLimited:
		return http.StatusTooManyRequests
	case e.Code == SUBFetchFailed || e.Code == SUBFetchTimeout || e.Code == SUBHTTPStatusError:
		return http.StatusBadGateway
	case e.Code == RTDockerSockUnavailable || e.Code == RTSingboxContainerNotFound:
		return http.StatusServiceUnavailable
	case e.Code == NotImplemented:
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}
