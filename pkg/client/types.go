package client

import (
	"time"
)

// UploadOptions represents file upload options
type UploadOptions struct {
	TenantID     string                 `json:"tenant_id,omitempty"`
	ProjectID    string                 `json:"project_id,omitempty"`
	CreatedBy    string                 `json:"created_by,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	IsPublic     bool                   `json:"is_public"`
	MaxSize      int64                  `json:"max_size"`
	AllowedTypes []string               `json:"allowed_types"`
}

// FileMetadata represents file metadata
type FileMetadata struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	OriginalName string                 `json:"original_name"`
	Path         string                 `json:"path"`
	Size         int64                  `json:"size"`
	MimeType     string                 `json:"mime_type"`
	FileType     string                 `json:"file_type"`
	MD5Hash      string                 `json:"md5_hash"`
	SHA256Hash   string                 `json:"sha256_hash"`
	StorageType  string                 `json:"storage_type"`
	TenantID     string                 `json:"tenant_id,omitempty"`
	ProjectID    string                 `json:"project_id,omitempty"`
	CreatedBy    string                 `json:"created_by,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	AccessCount  int64                  `json:"access_count"`
	LastAccessed *time.Time             `json:"last_accessed,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	IsPublic     bool                   `json:"is_public"`
	ExpiresAt    *time.Time             `json:"expires_at,omitempty"`
}

// AnalyticsEvent represents an analytics event
type AnalyticsEvent struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	TenantID   string                 `json:"tenant_id,omitempty"`
	ProjectID  string                 `json:"project_id,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	SessionID  string                 `json:"session_id,omitempty"`
	EventType  string                 `json:"event_type"`
	EventName  string                 `json:"event_name"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Duration   *int64                 `json:"duration,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	IP         string                 `json:"ip,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
}

// Metric represents a metric definition
type Metric struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	EventType    string                 `json:"event_type"`
	FieldName    string                 `json:"field_name,omitempty"`
	Filter       map[string]interface{} `json:"filter,omitempty"`
	Formula      string                 `json:"formula,omitempty"`
	Format       string                 `json:"format,omitempty"`
	Description  string                 `json:"description,omitempty"`
}

// FilterOptions represents analytics query filters
type FilterOptions struct {
	TenantID    string            `json:"tenant_id,omitempty"`
	ProjectID   string            `json:"project_id,omitempty"`
	UserID      string            `json:"user_id,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	EventTypes  []string          `json:"event_types,omitempty"`
	EventNames  []string          `json:"event_names,omitempty"`
	DateRange   *DateRange        `json:"date_range,omitempty"`
	Countries   []string          `json:"countries,omitempty"`
	Devices     []string          `json:"devices,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Search      string            `json:"search,omitempty"`
	Properties  map[string]string `json:"properties,omitempty"`
}

// DateRange represents a date range filter
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}