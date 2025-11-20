package errors

// Error codes for different types of errors
const (
	// Common error codes
	CodeUnknown       = "UNKNOWN"
	CodeInvalidInput  = "INVALID_INPUT"
	CodeUnauthorized  = "UNAUTHORIZED"
	CodeForbidden     = "FORBIDDEN"
	CodeNotFound      = "NOT_FOUND"
	CodeConflict      = "CONFLICT"
	CodeInternal      = "INTERNAL"
	CodeRateLimited   = "RATE_LIMITED"

	// Domain specific error codes
	CodeUserNotFound      = "USER_NOT_FOUND"
	CodeUserAlreadyExists  = "USER_ALREADY_EXISTS"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
	CodeAccountDisabled    = "ACCOUNT_DISABLED"
	CodeEmailNotVerified  = "EMAIL_NOT_VERIFIED"

	CodeTenantNotFound     = "TENANT_NOT_FOUND"
	CodeTenantDisabled     = "TENANT_DISABLED"

	CodeTableNotFound      = "TABLE_NOT_FOUND"
	CodeColumnNotFound     = "COLUMN_NOT_FOUND"
	CodeInvalidSchema      = "INVALID_SCHEMA"
	CodeDuplicateRecord    = "DUPLICATE_RECORD"

	CodeFileNotFound       = "FILE_NOT_FOUND"
	CodeUploadFailed       = "UPLOAD_FAILED"
	CodeDownloadFailed     = "DOWNLOAD_FAILED"
	CodeStorageQuotaExceeded = "STORAGE_QUOTA_EXCEEDED"

	CodeSearchFailed       = "SEARCH_FAILED"
	CodeIndexingFailed     = "INDEXING_FAILED"

	CodePermissionDenied   = "PERMISSION_DENIED"
	CodeRoleNotFound       = "ROLE_NOT_FOUND"

	CodeTokenExpired       = "TOKEN_EXPIRED"
	CodeInvalidToken       = "INVALID_TOKEN"
	CodeSessionExpired     = "SESSION_EXPIRED"

	CodeInfrastructure     = "INFRASTRUCTURE"
	CodeDatabaseError      = "DATABASE_ERROR"
	CodeNetworkError       = "NETWORK_ERROR"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"

	CodeValidationError   = "VALIDATION_ERROR"
	CodeConstraintViolation = "CONSTRAINT_VIOLATION"
	CodeTimeout            = "TIMEOUT"
)

// HTTP status code mappings
var ErrorHTTPStatus = map[string]int{
	CodeUnknown:           500,
	CodeInvalidInput:      400,
	CodeUnauthorized:      401,
	CodeForbidden:         403,
	CodeNotFound:          404,
	CodeConflict:          409,
	CodeInternal:          500,
	CodeRateLimited:       429,

	CodeUserNotFound:      404,
	CodeUserAlreadyExists:  409,
	CodeInvalidCredentials: 401,
	CodeAccountDisabled:    403,
	CodeEmailNotVerified:  403,

	CodeTenantNotFound:     404,
	CodeTenantDisabled:     403,

	CodeTableNotFound:      404,
	CodeColumnNotFound:     404,
	CodeInvalidSchema:      400,
	CodeDuplicateRecord:    409,

	CodeFileNotFound:       404,
	CodeUploadFailed:       400,
	CodeDownloadFailed:     500,
	CodeStorageQuotaExceeded: 413,

	CodeSearchFailed:       500,
	CodeIndexingFailed:     500,

	CodePermissionDenied:   403,
	CodeRoleNotFound:       404,

	CodeTokenExpired:       401,
	CodeInvalidToken:       401,
	CodeSessionExpired:     401,

	CodeInfrastructure:     503,
	CodeDatabaseError:      500,
	CodeNetworkError:       503,
	CodeServiceUnavailable: 503,

	CodeValidationError:   400,
	CodeConstraintViolation: 422,
	CodeTimeout:            408,
}

// GetHTTPStatus returns the appropriate HTTP status code for an error code
func GetHTTPStatus(code string) int {
	if status, exists := ErrorHTTPStatus[code]; exists {
		return status
	}
	return 500 // Default to Internal Server Error
}

// IsClientError returns true if the error is a client error (4xx)
func IsClientError(code string) bool {
	status := GetHTTPStatus(code)
	return status >= 400 && status < 500
}

// IsServerError returns true if the error is a server error (5xx)
func IsServerError(code string) bool {
	status := GetHTTPStatus(code)
	return status >= 500
}

// IsRetryable returns true if the error is retryable
func IsRetryable(code string) bool {
	retryableCodes := map[string]bool{
		CodeNetworkError:       true,
		CodeServiceUnavailable: true,
		CodeTimeout:            true,
		CodeRateLimited:       true,
		CodeInfrastructure:     true,
	}
	return retryableCodes[code]
}