package common

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// ErrorCode represents standardized error codes
type ErrorCode string

const (
	// System errors
	ErrCodeInternal      ErrorCode = "INTERNAL_ERROR"
	ErrCodeUnavailable   ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeTimeout       ErrorCode = "TIMEOUT"
	ErrCodeSystemError   ErrorCode = "SYSTEM_ERROR"
	ErrCodeNetworkError  ErrorCode = "NETWORK_ERROR"
	ErrCodeDatabaseError ErrorCode = "DATABASE_ERROR"
	ErrCodeStorageError  ErrorCode = "STORAGE_ERROR"
	ErrCodeSearchError   ErrorCode = "SEARCH_ERROR"

	// Validation errors
	ErrCodeInvalidInput  ErrorCode = "INVALID_INPUT"
	ErrCodeMissingField  ErrorCode = "MISSING_FIELD"
	ErrCodeInvalidFormat ErrorCode = "INVALID_FORMAT"
	ErrCodeValidationError ErrorCode = "VALIDATION_ERROR"

	// Storage errors
	ErrCodeNotFound    ErrorCode = "NOT_FOUND"
	ErrCodeDuplicate   ErrorCode = "DUPLICATE"
	ErrCodeConstraint  ErrorCode = "CONSTRAINT_VIOLATION"
	ErrCodeAlreadyExists ErrorCode = "ALREADY_EXISTS"
	ErrCodeConflict    ErrorCode = "CONFLICT"
	ErrCodeLimitExceeded ErrorCode = "LIMIT_EXCEEDED"

	// Authorization errors
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden    ErrorCode = "FORBIDDEN"
	ErrCodeInvalidToken ErrorCode = "INVALID_TOKEN"
	ErrCodeTokenExpired ErrorCode = "TOKEN_EXPIRED"

	// Rate limiting
	ErrCodeRateLimited   ErrorCode = "RATE_LIMITED"
	ErrCodeQuotaExceeded ErrorCode = "QUOTA_EXCEEDED"

	// External service errors
	ErrCodeServiceError ErrorCode = "SERVICE_ERROR"
)

// ErrorLevel represents the severity level of an error
type ErrorLevel string

const (
	ErrorLevelError   ErrorLevel = "ERROR"
	ErrorLevelWarning ErrorLevel = "WARNING"
	ErrorLevelInfo    ErrorLevel = "INFO"
	ErrorLevelDebug   ErrorLevel = "DEBUG"
)

// AppError represents a structured application error
type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Level      ErrorLevel             `json:"level"`
	Timestamp  int64                  `json:"timestamp"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Stack      []string               `json:"stack,omitempty"`
	HTTPStatus int                    `json:"-"`
	Cause      error                  `json:"-"`
	RequestId  string                 `json:"request_id,omitempty"`
	UserId     string                 `json:"user_id,omitempty"`
	TenantId   string                 `json:"tenant_id,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewAppError creates a new application error
func NewAppError(code ErrorCode, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Details:    make(map[string]interface{}),
	}
}

// WithCause adds underlying cause to the error
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// WithDetail adds additional detail to the error
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// Predefined error constructors
func NewInternalError(message string) *AppError {
	return NewAppError(ErrCodeInternal, message, http.StatusInternalServerError)
}

func NewNotFoundError(resource string) *AppError {
	return NewAppError(ErrCodeNotFound, fmt.Sprintf("%s not found", resource), http.StatusNotFound)
}

func NewInvalidInputError(message string) *AppError {
	return NewAppError(ErrCodeInvalidInput, message, http.StatusBadRequest)
}

func NewUnauthorizedError(message string) *AppError {
	return NewAppError(ErrCodeUnauthorized, message, http.StatusUnauthorized)
}

func NewForbiddenError(message string) *AppError {
	return NewAppError(ErrCodeForbidden, message, http.StatusForbidden)
}

func NewTimeoutError(message string) *AppError {
	return NewAppError(ErrCodeTimeout, message, http.StatusRequestTimeout)
}

func NewUnavailableError(message string) *AppError {
	return NewAppError(ErrCodeUnavailable, message, http.StatusServiceUnavailable)
}

// IsAppError checks if error is an AppError
func IsAppError(err error) (*AppError, bool) {
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}

// GetHTTPStatus extracts HTTP status from error
func GetHTTPStatus(err error) int {
	if appErr, ok := IsAppError(err); ok {
		return appErr.HTTPStatus
	}
	return http.StatusInternalServerError
}

// Enhanced error constructors with more context
func NewAppErrorWithLevel(code ErrorCode, message string, level ErrorLevel, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Level:      level,
		Timestamp:  time.Now().Unix(),
		HTTPStatus: httpStatus,
		Details:    make(map[string]interface{}),
		Stack:     getStack(),
	}
}

// WithContext adds additional context to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithRequest sets request information
func (e *AppError) WithRequest(requestId, userId, tenantId string) *AppError {
	e.RequestId = requestId
	e.UserId = userId
	e.TenantId = tenantId
	return e
}

// Predefined enhanced error constructors
func NewSystemError(message string, err error) *AppError {
	if err != nil {
		return NewAppErrorWithLevel(ErrCodeSystemError, message, ErrorLevelError, http.StatusInternalServerError).WithCause(err)
	}
	return NewAppErrorWithLevel(ErrCodeSystemError, message, ErrorLevelError, http.StatusInternalServerError)
}

func NewNetworkError(message string, err error) *AppError {
	if err != nil {
		return NewAppErrorWithLevel(ErrCodeNetworkError, message, ErrorLevelError, http.StatusInternalServerError).WithCause(err)
	}
	return NewAppErrorWithLevel(ErrCodeNetworkError, message, ErrorLevelError, http.StatusInternalServerError)
}

func NewDatabaseError(message string, err error) *AppError {
	if err != nil {
		return NewAppErrorWithLevel(ErrCodeDatabaseError, message, ErrorLevelError, http.StatusInternalServerError).WithCause(err)
	}
	return NewAppErrorWithLevel(ErrCodeDatabaseError, message, ErrorLevelError, http.StatusInternalServerError)
}

func NewValidationError(message string, field string) *AppError {
	err := NewAppErrorWithLevel(ErrCodeValidationError, message, ErrorLevelWarning, http.StatusBadRequest)
	if field != "" {
		err.WithContext("field", field)
	}
	return err
}

func NewRateLimitError(message string, retryAfter int) *AppError {
	if message == "" {
		message = "Rate limit exceeded"
	}
	err := NewAppErrorWithLevel(ErrCodeRateLimited, message, ErrorLevelWarning, http.StatusTooManyRequests)
	if retryAfter > 0 {
		err.WithContext("retry_after", retryAfter)
	}
	return err
}

// getStack returns the current stack trace
func getStack() []string {
	var stack []string
	for i := 2; i < 10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		// Only include our own packages in the stack
		if strings.Contains(fn.Name(), "github.com/metabase/metabase") {
			stack = append(stack, fmt.Sprintf("%s:%d - %s", file, line, fn.Name()))
		}
	}
	return stack
}

// ErrorHandler handles error processing and logging
type ErrorHandler struct {
	logger ErrorLogger
	config *ErrorHandlerConfig
}

// ErrorLogger interface for different logging implementations
type ErrorLogger interface {
	LogError(error)
	LogWarning(error)
	LogInfo(error)
	LogDebug(error)
}

// DefaultLogger implements basic file/stdout logging
type DefaultLogger struct{}

// LogError logs error level messages
func (l *DefaultLogger) LogError(err error) {
	log.Printf("[ERROR] %v", err)
}

// LogWarning logs warning level messages
func (l *DefaultLogger) LogWarning(err error) {
	log.Printf("[WARN] %v", err)
}

// LogInfo logs info level messages
func (l *DefaultLogger) LogInfo(err error) {
	log.Printf("[INFO] %v", err)
}

// LogDebug logs debug level messages
func (l *DefaultLogger) LogDebug(err error) {
	log.Printf("[DEBUG] %v", err)
}

// ErrorHandlerConfig configures error handler behavior
type ErrorHandlerConfig struct {
	EnableStackTrace bool
	LogContext       bool
	LogLevel         ErrorLevel
	ReportCritical   bool
	MaxRetries       int
	RetryDelay       time.Duration
}

// DefaultErrorHandlerConfig returns default configuration
func DefaultErrorHandlerConfig() *ErrorHandlerConfig {
	return &ErrorHandlerConfig{
		EnableStackTrace: true,
		LogContext:       true,
		LogLevel:         ErrorLevelWarning,
		ReportCritical:   true,
		MaxRetries:       3,
		RetryDelay:       time.Second,
	}
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger ErrorLogger, config *ErrorHandlerConfig) *ErrorHandler {
	if config == nil {
		config = DefaultErrorHandlerConfig()
	}
	if logger == nil {
		logger = &DefaultLogger{}
	}

	return &ErrorHandler{
		logger: logger,
		config: config,
	}
}

// Handle processes an error according to configuration
func (h *ErrorHandler) Handle(err error) {
	if err == nil {
		return
	}

	if appErr, ok := IsAppError(err); ok {
		h.handleAppError(appErr)
	} else {
		h.handleGenericError(err)
	}
}

// handleAppError processes structured application errors
func (h *ErrorHandler) handleAppError(err *AppError) {
	shouldLog := h.shouldLog(err.Level)

	if !shouldLog {
		return
	}

	switch err.Level {
	case ErrorLevelError:
		h.logger.LogError(err)
	case ErrorLevelWarning:
		h.logger.LogWarning(err)
	case ErrorLevelInfo:
		h.logger.LogInfo(err)
	case ErrorLevelDebug:
		h.logger.LogDebug(err)
	}

	// Report critical errors
	if h.config.ReportCritical && err.Level == ErrorLevelError {
		h.reportError(err)
	}
}

// handleGenericError processes generic errors
func (h *ErrorHandler) handleGenericError(err error) {
	// Convert to structured error for consistent handling
	appErr := NewSystemError(err.Error(), err)
	h.handleAppError(appErr)
}

// shouldLog determines if error should be logged based on level
func (h *ErrorHandler) shouldLog(level ErrorLevel) bool {
	levels := map[ErrorLevel]int{
		ErrorLevelError:   4,
		ErrorLevelWarning: 3,
		ErrorLevelInfo:    2,
		ErrorLevelDebug:   1,
	}

	threshold := levels[h.config.LogLevel]
	current := levels[level]

	return current >= threshold
}

// reportError reports critical errors (could integrate with external services)
func (h *ErrorHandler) reportError(err *AppError) {
	// This could integrate with services like Sentry, DataDog, etc.
	log.Printf("[CRITICAL] Error reported: %s - %s", err.Code, err.Message)
}

// RetryableOperation represents an operation that can be retried
type RetryableOperation func() error

// Retry executes an operation with retry logic
func (h *ErrorHandler) Retry(op RetryableOperation) error {
	var lastErr error

	for attempt := 0; attempt <= h.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(h.config.RetryDelay)
		}

		if err := op(); err != nil {
			lastErr = err
			h.Handle(err)

			// Don't retry on certain error types
			if appErr, ok := IsAppError(err); ok {
				if !h.shouldRetry(appErr) {
					break
				}
			}
			continue
		}

		return nil
	}

	return lastErr
}

// shouldRetry determines if error should be retried
func (h *ErrorHandler) shouldRetry(err *AppError) bool {
	switch err.Code {
	case ErrCodeNetworkError, ErrCodeTimeout, ErrCodeUnavailable, ErrCodeServiceError:
		return true
	case ErrCodeValidationError, ErrCodeUnauthorized, ErrCodeForbidden, ErrCodeNotFound:
		return false
	default:
		return true
	}
}

// Global error handler instance
var globalErrorHandler *ErrorHandler

// InitErrorHandler initializes the global error handler
func InitErrorHandler(logger ErrorLogger, config *ErrorHandlerConfig) {
	globalErrorHandler = NewErrorHandler(logger, config)
}

// HandleError handles an error using the global handler
func HandleError(err error) {
	if globalErrorHandler != nil {
		globalErrorHandler.Handle(err)
	} else {
		// Fallback to basic logging
		if err != nil {
			log.Printf("[ERROR] %v", err)
		}
	}
}

// RetryOperation executes an operation with retry using global handler
func RetryOperation(op RetryableOperation) error {
	if globalErrorHandler != nil {
		return globalErrorHandler.Retry(op)
	}
	return op()
}