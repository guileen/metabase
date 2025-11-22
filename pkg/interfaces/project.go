package interfaces

import "time"

// ProjectConfig represents project configuration
type ProjectConfig struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Version       string                 `json:"version"`
	Category      string                 `json:"category"`
	Tags          []string               `json:"tags"`
	Dependencies  []string               `json:"dependencies"`
	DefaultConfig map[string]interface{} `json:"default_config"`
}

// DeploymentRequest represents a project deployment request
type DeploymentRequest struct {
	ProjectID string                 `json:"project_id" validate:"required"`
	Config    map[string]interface{} `json:"config"`
	TenantID  string                 `json:"tenant_id" validate:"required"`
	AutoStart bool                   `json:"auto_start"`
}

// InstallationState represents the state of a project installation
type InstallationState struct {
	ProjectID   string                 `json:"project_id"`
	ProjectType string                 `json:"project_type"`
	TenantID    string                 `json:"tenant_id"`
	Status      string                 `json:"status"` // "pending", "installing", "completed", "failed"
	Success     bool                   `json:"success"`
	Message     string                 `json:"message"`
	Version     string                 `json:"version"`
	InstalledAt *time.Time             `json:"installed_at,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}
