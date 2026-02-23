package dto

// ErrorEnvelope is the standard API error response (docs/error-codes.md).
type ErrorEnvelope struct {
	Error ErrorObject `json:"error"`
}

type ErrorObject struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}
