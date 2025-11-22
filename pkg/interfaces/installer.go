package interfaces

import (
	"context"
	"database/sql"
	"time"
)

// InstallResult represents the unified result of a project installation
type InstallResult struct {
	Success     bool                   `json:"success"`
	Message     string                 `json:"message"`
	ProjectID   string                 `json:"project_id"`
	ProjectType string                 `json:"project_type"` // "cms", "auth-gateway", etc.
	Version     string                 `json:"version"`
	InstalledAt time.Time              `json:"installed_at"`
	Endpoint    string                 `json:"endpoint,omitempty"`
	AdminURL    string                 `json:"admin_url,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// InstallRequest represents a unified installation request
type InstallRequest struct {
	ProjectType string                 `json:"project_type" validate:"required"`
	TenantID    string                 `json:"tenant_id" validate:"required"`
	Config      map[string]interface{} `json:"config"`
}

// ProjectInstaller interface defines the contract for project installers
type ProjectInstaller interface {
	// Basic project information
	Name() string
	Version() string
	Type() string // "cms", "auth-gateway", etc.

	// Installation lifecycle
	CheckInstallation(ctx context.Context, tenantID string) (bool, error)
	Install(ctx context.Context, req *InstallRequest) (*InstallResult, error)
	Uninstall(ctx context.Context, tenantID string) error

	// Configuration
	GetConfigurationSchema() map[string]interface{}
	ValidateConfiguration(config map[string]interface{}) error

	// Dependencies
	CheckDependencies(ctx context.Context, db *sql.DB) error
	GetDependencies() []string

	// Health and status
	GetInstallationStatus(ctx context.Context, tenantID string) (map[string]interface{}, error)
}

// ProjectHealthChecker interface defines health checking for projects
type ProjectHealthChecker interface {
	CheckHealth(ctx context.Context, tenantID string) (*HealthStatus, error)
	GetMetrics(ctx context.Context, tenantID string) (map[string]interface{}, error)
}

// HealthStatus represents the health status of a project
type HealthStatus struct {
	Status      string                 `json:"status"` // healthy, unhealthy, degraded
	Message     string                 `json:"message"`
	LastChecked time.Time              `json:"last_checked"`
	Metrics     map[string]interface{} `json:"metrics,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// ProjectFactory interface for creating project installers
type ProjectFactory interface {
	CreateInstaller(db *sql.DB, logger interface{}) (ProjectInstaller, error)
	SupportedProjectType() string
}

// InstallerRegistry manages registered project installers
type InstallerRegistry interface {
	RegisterFactory(factory ProjectFactory)
	GetFactory(projectType string) (ProjectFactory, bool)
	ListSupportedTypes() []string
	CreateInstaller(projectType string, db *sql.DB, logger interface{}) (ProjectInstaller, error)
}

// Constants for project types
const (
	ProjectTypeCMS         = "cms"
	ProjectTypeAuthGateway = "auth-gateway"
	ProjectTypeFileManager = "file-manager"
	ProjectTypeAnalytics   = "analytics"
)

// Constants for installation status
const (
	InstallationStatusPending    = "pending"
	InstallationStatusInstalling = "installing"
	InstallationStatusCompleted  = "completed"
	InstallationStatusFailed     = "failed"
)

// Constants for health status
const (
	HealthStatusHealthy   = "healthy"
	HealthStatusUnhealthy = "unhealthy"
	HealthStatusDegraded  = "degraded"
)

// UnsupportedProjectTypeError is returned when an unsupported project type is requested
type UnsupportedProjectTypeError struct {
	ProjectType string
	Supported   []string
}

// Error implements the error interface
func (e *UnsupportedProjectTypeError) Error() string {
	return "unsupported project type: " + e.ProjectType
}
