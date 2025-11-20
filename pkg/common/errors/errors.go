package errors

import ("fmt"
	"runtime/debug"
	"time")

// Error represents a structured error with context
type Error struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Cause      error                  `json:"cause,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	StackTrace []string               `json:"stack_trace,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

// Wrap creates a new error with context
func Wrap(err error, code, message string) *Error {
	if err == nil {
		return nil
	}

	// Prevent double-wrapping
	if typedErr, ok := err.(*Error); ok {
		return typedErr
	}

	stack := make([]string, 0)
	for _, pc := range debug.Stack() {
		stack = append(stack, pc)
	}

	return &Error{
		Code:       code,
		Message:    message,
		Cause:      err,
		Timestamp:  time.Now(),
		StackTrace: stack,
	}
}

// New creates a new error without wrapping
func New(code, message string) *Error {
	return &Error{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// NotFound creates a not found error
func NotFound(resource string) *Error {
	return New(CodeNotFound, fmt.Sprintf("%s not found", resource))
}

// NotFoundWithID creates a not found error with ID
func NotFoundWithID(resource, id string) *Error {
	return New(CodeNotFound, fmt.Sprintf("%s with id '%s' not found", resource, id))
}

// AlreadyExists creates an already exists error
func AlreadyExists(resource string) *Error {
	return New(CodeConflict, fmt.Sprintf("%s already exists", resource))
}

// AlreadyExistsWithID creates an already exists error with ID
func AlreadyExistsWithID(resource, id string) *Error {
	return New(CodeConflict, fmt.Sprintf("%s with id '%s' already exists", resource, id))
}

// Unauthorized creates an unauthorized error
func Unauthorized(message string) *Error {
	if message == "" {
		message = "Unauthorized access"
	}
	return New(CodeUnauthorized, message)
}

// Forbidden creates a forbidden error
func Forbidden(message string) *Error {
	if message == "" {
		message = "Access forbidden"
	}
	return New(CodeForbidden, message)
}

// InvalidInput creates an invalid input error
func InvalidInput(message string) *Error {
	if message == "" {
		message = "Invalid input"
	}
	return New(CodeInvalidInput, message)
}

// ValidationFailed creates a validation failed error
func ValidationFailed(message string, details map[string]interface{}) *Error {
	err := New(CodeValidationError, message)
	if details != nil {
		err.Details = details
	}
	return err
}

// Internal creates an internal server error
func Internal(message string) *Error {
	if message == "" {
		message = "Internal server error"
	}
	return New(CodeInternal, message)
}

// InternalWrap wraps an error as internal server error
func InternalWrap(err error, message string) *Error {
	if message == "" {
		message = "Internal server error"
	}
	return Wrap(err, CodeInternal, message)
}

// DatabaseError creates a database error
func DatabaseError(message string) *Error {
	return New(CodeDatabaseError, message)
}

// DatabaseErrorWrap wraps an error as database error
func DatabaseErrorWrap(err error, message string) *Error {
	if message == "" {
		message = "Database error"
	}
	return Wrap(err, CodeDatabaseError, message)
}

// NetworkError creates a network error
func NetworkError(message string) *Error {
	return New(CodeNetworkError, message)
}

// NetworkErrorWrap wraps an error as network error
func NetworkErrorWrap(err error, message string) *Error {
	if message == "" {
		message = "Network error"
	}
	return Wrap(err, CodeNetworkError, message)
}

// TimeoutError creates a timeout error
func TimeoutError(message string) *Error {
	if message == "" {
		message = "Operation timed out"
	}
	return New(CodeTimeout, message)
}

// TimeoutErrorWrap wraps an error as timeout error
func TimeoutErrorWrap(err error, message string) *Error {
	if message == "" {
		message = "Operation timed out"
	}
	return Wrap(err, CodeTimeout, message)
}

// RateLimited creates a rate limited error
func RateLimited(message string) *Error {
	if message == "" {
		message = "Rate limit exceeded"
	}
	return New(CodeRateLimited, message)
}

// ServiceUnavailable creates a service unavailable error
func ServiceUnavailable(service string) *Error {
	if service == "" {
		service = "Service"
	}
	return New(CodeServiceUnavailable, fmt.Sprintf("%s is unavailable", service))
}

// ServiceUnavailableWrap wraps an error as service unavailable error
func ServiceUnavailableWrap(err error, service string) *Error {
	if service == "" {
		service = "Service"
	}
	message := fmt.Sprintf("%s is unavailable", service)
	return Wrap(err, CodeServiceUnavailable, message)
}

// User specific errors
func UserNotFound(id string) *Error {
	return NotFoundWithID("User", id)
}

func UserAlreadyExists(email string) *Error {
	return New(CodeUserAlreadyExists, fmt.Sprintf("User with email '%s' already exists", email))
}

func InvalidCredentials() *Error {
	return New(CodeInvalidCredentials, "Invalid credentials")
}

func AccountDisabled() *Error {
	return New(CodeAccountDisabled, "Account is disabled")
}

func EmailNotVerified() *Error {
	return New(CodeEmailNotVerified, "Email not verified")
}

// Tenant specific errors
func TenantNotFound(id string) *Error {
	return NotFoundWithID("Tenant", id)
}

func TenantDisabled() *Error {
	return New(CodeTenantDisabled, "Tenant is disabled")
}

// Table specific errors
func TableNotFound(name string) *Error {
	return NotFoundWithID("Table", name)
}

func ColumnNotFound(table, column string) *Error {
	return New(CodeColumnNotFound, fmt.Sprintf("Column '%s' not found in table '%s'", column, table))
}

func InvalidSchema(message string) *Error {
	if message == "" {
		message = "Invalid schema"
	}
	return New(CodeInvalidSchema, message)
}

// File specific errors
func FileNotFound(path string) *Error {
	return New(CodeFileNotFound, fmt.Sprintf("File not found: %s", path))
}

func UploadFailed(message string, details map[string]interface{}) *Error {
	err := New(CodeUploadFailed, message)
	if details != nil {
		err.Details = details
	}
	return err
}

func DownloadFailed(message string) *Error {
	return New(CodeDownloadFailed, message)
}

func StorageQuotaExceeded() *Error {
	return New(CodeStorageQuotaExceeded, "Storage quota exceeded")
}

// Search specific errors
func SearchFailed(message string) *Error {
	return New(CodeSearchFailed, message)
}

func IndexingFailed(message string) *Error {
	return New(CodeIndexingFailed, message)
}

// Permission specific errors
func PermissionDenied(resource, action string) *Error {
	return New(CodePermissionDenied, fmt.Sprintf("Permission denied: %s %s", action, resource))
}

func RoleNotFound(name string) *Error {
	return NotFoundWithID("Role", name)
}

// Token specific errors
func TokenExpired() *Error {
	return New(CodeTokenExpired, "Token has expired")
}

func InvalidToken() *Error {
	return New(CodeInvalidToken, "Invalid token")
}

func SessionExpired() *Error {
	return New(CodeSessionExpired, "Session has expired")
}

// ConstraintViolation creates a constraint violation error
func ConstraintViolation(constraint string) *Error {
	return New(CodeConstraintViolation, fmt.Sprintf("Constraint violation: %s", constraint))
}

// IsErrorType checks if an error matches a specific error code
func IsErrorType(err error, code string) bool {
	if typedErr, ok := err.(*Error); ok {
		return typedErr.Code == code
	}
	return false
}

// GetErrorCode returns the error code from an error
func GetErrorCode(err error) string {
	if typedErr, ok := err.(*Error); ok {
		return typedErr.Code
	}
	return CodeUnknown
}

// GetErrorMessage returns the error message from an error
func GetErrorMessage(err error) string {
	if typedErr, ok := err.(*Error); ok {
		return typedErr.Message
	}
	if err != nil {
		return err.Error()
	}
	return "Unknown error"
}