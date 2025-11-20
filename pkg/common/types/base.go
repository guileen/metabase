package types

import ("context"
	"fmt"
	"math/rand"
	"time")

// Base ID type for all entities
type ID string

func (id ID) String() string {
	return string(id)
}

func (id ID) IsEmpty() bool {
	return string(id) == ""
}

// Base entity interface
type Entity interface {
	ID() ID
	CreatedAt() time.Time
	UpdatedAt() time.Time
}

// BaseAuditFields contains common audit fields for entities
type BaseAuditFields struct {
	ID        ID       `json:"id" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at,omitempty"`
	Version  int      `json:"version" db:"version"`
}

// Base entity with audit fields
type Base struct {
	BaseAuditFields
}

func (b *Base) ID() ID {
	return b.BaseAuditFields.ID
}

func (b *Base) CreatedAt() time.Time {
	return b.BaseAuditFields.CreatedAt
}

func (b *Base) UpdatedAt() time.Time {
	return b.BaseAuditFields.UpdatedAt
}

// Pagination represents pagination parameters
type Pagination struct {
	Page     int    `json:"page" form:"page" query:"page"`
	PageSize int    `json:"page_size" form:"page_size" query:"page_size"`
	Offset   int    `json:"offset" form:"offset" query:"offset"`
	Limit    int    `json:"limit" form:"limit" query:"limit"`
}

// Validate pagination parameters
func (p *Pagination) Validate() error {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 10
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	return nil
}

// GetOffset calculates offset from page and page_size
func (p *Pagination) GetOffset() int {
	if p.Offset > 0 {
		return p.Offset
	}
	return (p.Page - 1) * p.PageSize
}

// GetLimit returns the limit with proper bounds
func (p *Pagination) GetLimit() int {
	if p.Limit > 0 {
		return p.Limit
	}
	return p.PageSize
}

// QueryOptions contains common query options
type QueryOptions struct {
	IncludeDeleted bool                    `json:"include_deleted,omitempty"`
	OrderBy        []string               `json:"order_by,omitempty"`
	Filters        map[string]interface{} `json:"filters,omitempty"`
	Search         string                 `json:"search,omitempty"`
	Fields         []string               `json:"fields,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// TenantID represents a tenant identifier
type TenantID string

func (t TenantID) String() string {
	return string(t)
}

func (t TenantID) IsEmpty() bool {
	return string(t) == ""
}

// UserID represents a user identifier
type UserID string

func (u UserID) String() string {
	return string(u)
}

func (u UserID) IsEmpty() bool {
	return string(u) == ""
}

// RequestContext contains common request context information
type RequestContext struct {
	RequestID string    `json:"request_id"`
	TenantID  TenantID `json:"tenant_id"`
	UserID    UserID   `json:"user_id,omitempty"`
	IP        string   `json:"ip,omitempty"`
	UserAgent string   `json:"user_agent,omitempty"`
	Path      string   `json:"path,omitempty"`
	Method    string   `json:"method,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewRequestContext creates a new request context
func NewRequestContext(ctx context.Context) *RequestContext {
	return &RequestContext{
		RequestID: GenerateRequestID(),
		Timestamp: time.Now(),
	}
}

// WithTenant adds tenant ID to context
func (rc *RequestContext) WithTenant(tenantID TenantID) *RequestContext {
	rc.TenantID = tenantID
	return rc
}

// WithUser adds user ID to context
func (rc *RequestContext) WithUser(userID UserID) *RequestContext {
	rc.UserID = userID
	return rc
}

// WithIP adds IP address to context
func (rc *RequestContext) WithIP(ip string) *RequestContext {
	rc.IP = ip
	return rc
}

// WithUserAgent adds user agent to context
func (rc *RequestContext) WithUserAgent(userAgent string) *RequestContext {
	rc.UserAgent = userAgent
	return rc
}

// Metadata represents generic metadata
type Metadata map[string]interface{}

// NewMetadata creates new metadata
func NewMetadata() Metadata {
	return make(Metadata)
}

// Set sets a value in metadata
func (m Metadata) Set(key string, value interface{}) {
	m[key] = value
}

// Get gets a value from metadata
func (m Metadata) Get(key string) (interface{}, bool) {
	value, exists := m[key]
	return value, exists
}

// GetString gets a string value from metadata
func (m Metadata) GetString(key string) string {
	if value, exists := m[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// GetInt gets an int value from metadata
func (m Metadata) GetInt(key string) int {
	if value, exists := m[key]; exists {
		if num, ok := value.(int); ok {
			return num
		}
	}
	return 0
}

// GetBool gets a bool value from metadata
func (m Metadata) GetBool(key string) bool {
	if value, exists := m[key]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

// Error represents a structured error
type Error struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Cause      error                  `json:"-"`
	Timestamp  time.Time              `json:"timestamp"`
	RequestID  string                 `json:"request_id,omitempty"`
	StackTrace []string               `json:"stack_trace,omitempty"`
}

// Error implements the error interface
func (e *Error) Error() string {
	return e.Message
}

// Unwrap returns the cause of the error
func (e *Error) Unwrap() error {
	return e.Cause
}

// NewError creates a new structured error
func NewError(code, message string) *Error {
	return &Error{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// WithCause adds a cause to the error
func (e *Error) WithCause(cause error) *Error {
	e.Cause = cause
	return e
}

// WithDetails adds details to the error
func (e *Error) WithDetails(details map[string]interface{}) *Error {
	e.Details = details
	return e
}

// WithRequestID adds request ID to the error
func (e *Error) WithRequestID(requestID string) *Error {
	e.RequestID = requestID
	return e
}

// WithStackTrace adds stack trace to the error
func (e *Error) WithStackTrace(stack []string) *Error {
	e.StackTrace = stack
	return e
}

// Result represents a result type for operations
type Result[T any] struct {
	Value    T      `json:"value,omitempty"`
	Error    *Error `json:"error,omitempty"`
	Metadata Metadata `json:"metadata,omitempty"`
}

// NewSuccess creates a successful result
func NewSuccess[T any](value T) *Result[T] {
	return &Result[T]{
		Value:    value,
		Metadata: NewMetadata(),
	}
}

// NewError creates an error result
func NewError[T any](err error) *Result[T] {
	var structuredErr *Error
	if se, ok := err.(*Error); ok {
		structuredErr = se
	} else {
		structuredErr = NewError("UNKNOWN", err.Error()).WithCause(err)
	}
	return &Result[T]{
		Error:    structuredErr,
		Metadata: NewMetadata(),
	}
}

// IsSuccess returns true if the result is successful
func (r *Result[T]) IsSuccess() bool {
	return r.Error == nil
}

// IsError returns true if the result has an error
func (r *Result[T]) IsError() bool {
	return r.Error != nil
}

// Helper function to generate request IDs
func GenerateRequestID() string {
	return fmt.Sprintf("%d_%x", time.Now().UnixNano(), rand.Uint64())
}

// Type aliases for common types
type (
	String  = string
	Boolean = bool
	Integer = int
	Float   = float64
)