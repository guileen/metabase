package builtin

import (
	"context"
	"database/sql"

	"github.com/guileen/metabase/internal/biz/domain/tenant"
	"github.com/guileen/metabase/pkg/builtin"
	"go.uber.org/zap"
)

// Manager is a wrapper around the new builtin manager for backward compatibility
type Manager struct {
	builtinManager *builtin.Manager
	tenantManager  *tenant.TenantManager
	logger         *zap.Logger
}

// NewManager creates a new built-in project manager
func NewManager(db *sql.DB, tenantManager *tenant.TenantManager, logger *zap.Logger) *Manager {
	builtinManager := builtin.NewManager(db, logger)

	return &Manager{
		builtinManager: builtinManager,
		tenantManager:  tenantManager,
		logger:         logger,
	}
}

// ListAvailableProjects returns list of available built-in projects
func (m *Manager) ListAvailableProjects() []*BuiltinProject {
	projects := m.builtinManager.ListAvailableProjects()

	// Convert to legacy format for backward compatibility
	var legacyProjects []*BuiltinProject
	for _, project := range projects {
		legacyProject := &BuiltinProject{
			ID:           project.ID,
			Name:         project.Name,
			Description:  project.Description,
			Version:      project.Version,
			Category:     project.Category,
			Tags:         project.Tags,
			Dependencies: project.Dependencies,
			Config:       project.Config,
			IsInstalled:  project.IsInstalled,
			IsEnabled:    project.IsEnabled,
			InstalledAt:  project.InstalledAt,
		}
		legacyProjects = append(legacyProjects, legacyProject)
	}

	return legacyProjects
}

// GetProject returns a specific built-in project
func (m *Manager) GetProject(projectID string) (*BuiltinProject, error) {
	project, err := m.builtinManager.GetProject(projectID)
	if err != nil {
		return nil, err
	}

	// Convert to legacy format for backward compatibility
	return &BuiltinProject{
		ID:           project.ID,
		Name:         project.Name,
		Description:  project.Description,
		Version:      project.Version,
		Category:     project.Category,
		Tags:         project.Tags,
		Dependencies: project.Dependencies,
		Config:       project.Config,
		IsInstalled:  project.IsInstalled,
		IsEnabled:    project.IsEnabled,
		InstalledAt:  project.InstalledAt,
	}, nil
}

// DeployProject deploys a built-in project
func (m *Manager) DeployProject(ctx context.Context, req *DeploymentRequest) (*InstallResult, error) {
	// Convert to new deployment request format
	deploymentReq := &builtin.DeploymentRequest{
		ProjectID: req.ProjectID,
		Config:    req.Config,
		TenantID:  req.TenantID,
		AutoStart: req.AutoStart,
	}

	result, err := m.builtinManager.DeployProject(ctx, deploymentReq)
	if err != nil {
		return nil, err
	}

	// Convert to legacy result format for backward compatibility
	return &InstallResult{
		Success:     result.Success,
		Message:     result.Message,
		ProjectID:   result.ProjectID,
		ProjectType: result.ProjectType,
		Version:     result.Version,
		InstalledAt: result.InstalledAt,
		Endpoint:    result.Endpoint,
		AdminURL:    result.AdminURL,
		Config:      result.Config,
		Metadata:    result.Metadata,
	}, nil
}

// UndeployProject undeploys a built-in project
func (m *Manager) UndeployProject(ctx context.Context, projectID, tenantID string) error {
	return m.builtinManager.UndeployProject(ctx, projectID, tenantID)
}

// GetInstalledProjects returns list of installed projects for a tenant
func (m *Manager) GetInstalledProjects(ctx context.Context, tenantID string) ([]*BuiltinProject, error) {
	projects, err := m.builtinManager.GetInstalledProjects(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Convert to legacy format for backward compatibility
	var legacyProjects []*BuiltinProject
	for _, project := range projects {
		legacyProject := &BuiltinProject{
			ID:           project.ID,
			Name:         project.Name,
			Description:  project.Description,
			Version:      project.Version,
			Category:     project.Category,
			Tags:         project.Tags,
			Dependencies: project.Dependencies,
			Config:       project.Config,
			IsInstalled:  project.IsInstalled,
			IsEnabled:    project.IsEnabled,
			InstalledAt:  project.InstalledAt,
		}
		legacyProjects = append(legacyProjects, legacyProject)
	}

	return legacyProjects, nil
}

// CheckProjectHealth checks the health of installed projects
func (m *Manager) CheckProjectHealth(ctx context.Context, projectID, tenantID string) (*HealthStatus, error) {
	return m.builtinManager.CheckProjectHealth(ctx, projectID, tenantID)
}

// Legacy types for backward compatibility
type (
	BuiltinProject    = builtin.Project
	DeploymentRequest = builtin.DeploymentRequest
	InstallResult     = builtin.InstallResult
	HealthStatus      = builtin.HealthStatus
)
