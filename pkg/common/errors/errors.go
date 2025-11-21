package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode represents different types of errors
type ErrorCode string

const (
	// Validation errors
	ErrCodeValidation   ErrorCode = "validation"
	ErrCodeInvalidInput ErrorCode = "invalid_input"
	ErrCodeMissingField ErrorCode = "missing_field"

	// Resource errors
	ErrCodeNotFound      ErrorCode = "not_found"
	ErrCodeAlreadyExists ErrorCode = "already_exists"
	ErrCodeConflict      ErrorCode = "conflict"

	// Permission errors
	ErrCodeUnauthorized ErrorCode = "unauthorized"
	ErrCodeForbidden    ErrorCode = "forbidden"
	ErrCodePermission   ErrorCode = "permission_denied"

	// System errors
	ErrCodeInternal ErrorCode = "internal"
	ErrCodeDatabase ErrorCode = "database"
	ErrCodeNetwork  ErrorCode = "network"
	ErrCodeTimeout  ErrorCode = "timeout"

	// Business logic errors
	ErrCodeBusiness      ErrorCode = "business"
	ErrCodeLimitExceeded ErrorCode = "limit_exceeded"
	ErrCodeQuotaExceeded ErrorCode = "quota_exceeded"
)

// AppError represents a structured application error
type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	HTTPStatus int                    `json:"-"`
	Cause      error                  `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// New creates a new application error
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: defaultHTTPStatus(code),
	}
}

// WithCause adds a cause to the error
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// WithDetail adds a detail to the error
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithDetails adds multiple details to the error
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// WithHTTPStatus sets a custom HTTP status code
func (e *AppError) WithHTTPStatus(status int) *AppError {
	e.HTTPStatus = status
	return e
}

// Predefined error constructors
func NotFound(resource string) *AppError {
	return New(ErrCodeNotFound, fmt.Sprintf("%s not found", resource))
}

func InvalidInput(message string) *AppError {
	return New(ErrCodeInvalidInput, message)
}

func Unauthorized(message string) *AppError {
	return New(ErrCodeUnauthorized, message)
}

func Forbidden(message string) *AppError {
	return New(ErrCodeForbidden, message)
}

func Internal(message string) *AppError {
	return New(ErrCodeInternal, message)
}

func Database(message string) *AppError {
	return New(ErrCodeDatabase, message)
}

func Network(message string) *AppError {
	return New(ErrCodeNetwork, message)
}

func Timeout(message string) *AppError {
	return New(ErrCodeTimeout, message)
}

func Conflict(message string) *AppError {
	return New(ErrCodeConflict, message)
}

func AlreadyExists(resource string) *AppError {
	return New(ErrCodeAlreadyExists, fmt.Sprintf("%s already exists", resource))
}

func Validation(message string) *AppError {
	return New(ErrCodeValidation, message)
}

func MissingField(field string) *AppError {
	return New(ErrCodeMissingField, fmt.Sprintf("Missing required field: %s", field))
}

func Business(message string) *AppError {
	return New(ErrCodeBusiness, message)
}

func LimitExceeded(message string) *AppError {
	return New(ErrCodeLimitExceeded, message)
}

func QuotaExceeded(message string) *AppError {
	return New(ErrCodeQuotaExceeded, message)
}

// WrapError wraps an existing error with additional context
func WrapError(err error, code ErrorCode, message string) *AppError {
	return New(code, message).WithCause(err)
}

// defaultHTTPStatus returns the default HTTP status code for an error code
func defaultHTTPStatus(code ErrorCode) int {
	switch code {
	case ErrCodeValidation, ErrCodeInvalidInput, ErrCodeMissingField:
		return http.StatusBadRequest
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeForbidden, ErrCodePermission:
		return http.StatusForbidden
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeConflict, ErrCodeAlreadyExists:
		return http.StatusConflict
	case ErrCodeLimitExceeded, ErrCodeQuotaExceeded:
		return http.StatusTooManyRequests
	case ErrCodeTimeout:
		return http.StatusRequestTimeout
	case ErrCodeNetwork:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) (*AppError, bool) {
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}

// GetCode extracts the error code from an error
func GetCode(err error) ErrorCode {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return ErrCodeInternal
}

// GetHTTPStatus extracts the HTTP status code from an error
func GetHTTPStatus(err error) int {
	if appErr, ok := err.(*AppError); ok {
		return appErr.HTTPStatus
	}
	return http.StatusInternalServerError
}
