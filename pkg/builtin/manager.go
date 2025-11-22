package builtin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/guileen/metabase/internal/installer"
	"github.com/guileen/metabase/pkg/installation"
	"github.com/guileen/metabase/pkg/interfaces"
	"go.uber.org/zap"
)

// Manager manages built-in project deployment using the new architecture
type Manager struct {
	db                *sql.DB
	installManager    *installation.Manager
	logger            *zap.Logger
	availableProjects map[string]*Project
}

// Project represents a built-in project
type Project struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Version      string                 `json:"version"`
	Category     string                 `json:"category"`
	Tags         []string               `json:"tags"`
	Dependencies []string               `json:"dependencies"`
	Config       map[string]interface{} `json:"config"`
	IsInstalled  bool                   `json:"is_installed"`
	IsEnabled    bool                   `json:"is_enabled"`
	InstalledAt  *time.Time             `json:"installed_at,omitempty"`
}

// NewManager creates a new built-in project manager
func NewManager(db *sql.DB, logger *zap.Logger) *Manager {
	// Create installer registry and register factories
	registry := installation.NewRegistry()
	registry.RegisterFactory(installer.NewCMSFactory())
	registry.RegisterFactory(installer.NewAuthGatewayFactory())

	installManager := installation.NewManager(db, registry, logger)

	manager := &Manager{
		db:                db,
		installManager:    installManager,
		logger:            logger,
		availableProjects: make(map[string]*Project),
	}

	// Register built-in projects
	manager.registerBuiltinProjects()

	return manager
}

// registerBuiltinProjects registers all available built-in projects
func (m *Manager) registerBuiltinProjects() {
	// Register Auth Gateway project
	authGatewayProject := &Project{
		ID:           interfaces.ProjectTypeAuthGateway,
		Name:         "统一认证网关",
		Description:  "提供统一的用户认证、授权和身份管理服务，支持多种认证方式和多租户隔离",
		Version:      "1.0.0",
		Category:     "security",
		Tags:         []string{"authentication", "authorization", "security", "multi-tenant"},
		Dependencies: []string{},
		Config: map[string]interface{}{
			"enable_local_auth":       true,
			"default_provider":        "local",
			"session_timeout_minutes": 1440,
			"enable_tenant_isolation": true,
			"enable_oauth2":           true,
			"enable_mfa":              false,
		},
		IsInstalled: false,
		IsEnabled:   false,
	}

	m.availableProjects[authGatewayProject.ID] = authGatewayProject

	// Register CMS project
	cmsProject := &Project{
		ID:           interfaces.ProjectTypeCMS,
		Name:         "内容管理系统",
		Description:  "功能完整的内容管理系统，支持文章、页面、分类、标签和媒体管理",
		Version:      "1.0.0",
		Category:     "content",
		Tags:         []string{"cms", "content", "blog", "pages"},
		Dependencies: []string{interfaces.ProjectTypeAuthGateway}, // CMS requires auth gateway
		Config: map[string]interface{}{
			"enable_comments":   true,
			"enable_search":     true,
			"enable_categories": true,
			"enable_tags":       true,
			"enable_media":      true,
		},
		IsInstalled: false,
		IsEnabled:   false,
	}

	m.availableProjects[cmsProject.ID] = cmsProject
}

// ListAvailableProjects returns list of available built-in projects
func (m *Manager) ListAvailableProjects() []*Project {
	var projects []*Project
	for _, project := range m.availableProjects {
		projects = append(projects, project)
	}
	return projects
}

// GetProject returns a specific built-in project
func (m *Manager) GetProject(projectID string) (*Project, error) {
	project, exists := m.availableProjects[projectID]
	if !exists {
		return nil, fmt.Errorf("project '%s' not found", projectID)
	}
	return project, nil
}

// DeployProject deploys a built-in project
func (m *Manager) DeployProject(ctx context.Context, req *interfaces.DeploymentRequest) (*interfaces.InstallResult, error) {
	// Get project
	project, err := m.GetProject(req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Check if project is already installed for this tenant
	if m.isProjectInstalled(ctx, req.ProjectID, req.TenantID) {
		return &interfaces.InstallResult{
			Success: false,
			Message: fmt.Sprintf("Project '%s' is already installed for tenant '%s'", req.ProjectID, req.TenantID),
		}, nil
	}

	m.logger.Info("Deploying built-in project",
		zap.String("project_id", req.ProjectID),
		zap.String("project_name", project.Name),
		zap.String("tenant_id", req.TenantID))

	// Create installation request
	installReq := &interfaces.InstallRequest{
		ProjectType: req.ProjectID,
		TenantID:    req.TenantID,
		Config:      req.Config,
	}

	// Install project using the installation manager
	result, err := m.installManager.InstallProject(ctx, installReq)
	if err != nil {
		return nil, fmt.Errorf("installation failed: %w", err)
	}

	if result.Success {
		// Record installation in database
		if err := m.recordInstallation(ctx, req.ProjectID, req.TenantID, result); err != nil {
			m.logger.Error("Failed to record installation", zap.Error(err))
			// Don't fail the deployment, just log the error
		}

		m.logger.Info("Built-in project deployed successfully",
			zap.String("project_id", req.ProjectID),
			zap.String("tenant_id", req.TenantID),
			zap.String("version", project.Version))
	}

	return result, nil
}

// UndeployProject undeploys a built-in project
func (m *Manager) UndeployProject(ctx context.Context, projectID, tenantID string) error {
	// Get project
	project, err := m.GetProject(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Check if project is installed
	if !m.isProjectInstalled(ctx, projectID, tenantID) {
		return fmt.Errorf("project '%s' is not installed for tenant '%s'", projectID, tenantID)
	}

	m.logger.Info("Undeploying built-in project",
		zap.String("project_id", projectID),
		zap.String("project_name", project.Name),
		zap.String("tenant_id", tenantID))

	// Uninstall project using the installation manager
	if err := m.installManager.UninstallProject(ctx, projectID, tenantID); err != nil {
		return fmt.Errorf("uninstallation failed: %w", err)
	}

	// Remove installation record
	if err := m.removeInstallationRecord(ctx, projectID, tenantID); err != nil {
		m.logger.Error("Failed to remove installation record", zap.Error(err))
		// Don't fail the undeployment, just log the error
	}

	m.logger.Info("Built-in project undeployed successfully",
		zap.String("project_id", projectID),
		zap.String("tenant_id", tenantID))

	return nil
}

// GetInstalledProjects returns list of installed projects for a tenant
func (m *Manager) GetInstalledProjects(ctx context.Context, tenantID string) ([]*Project, error) {
	// Get installed project records from database
	query := `
		SELECT project_id, version, config, endpoint, admin_url, installed_at
		FROM builtin_project_installations
		WHERE tenant_id = ?
		ORDER BY installed_at DESC`

	rows, err := m.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query installed projects: %w", err)
	}
	defer rows.Close()

	var installedProjects []*Project
	for rows.Next() {
		var projectID, version, endpoint, adminURL string
		var configJSON sql.NullString
		var installedAt time.Time

		err := rows.Scan(&projectID, &version, &configJSON, &endpoint, &adminURL, &installedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan installation record: %w", err)
		}

		// Get project definition
		project, err := m.GetProject(projectID)
		if err != nil {
			m.logger.Warn("Project definition not found for installed project",
				zap.String("project_id", projectID))
			continue
		}

		// Update project status
		project.IsInstalled = true
		project.IsEnabled = true
		project.InstalledAt = &installedAt

		installedProjects = append(installedProjects, project)
	}

	return installedProjects, nil
}

// CheckProjectHealth checks the health of installed projects
func (m *Manager) CheckProjectHealth(ctx context.Context, projectID, tenantID string) (*interfaces.HealthStatus, error) {
	return m.installManager.CheckProjectHealth(ctx, projectID, tenantID)
}

// ListSupportedProjects returns all supported project types
func (m *Manager) ListSupportedProjects() []string {
	return m.installManager.ListSupportedProjects()
}

// Helper methods

func (m *Manager) isProjectInstalled(ctx context.Context, projectID, tenantID string) bool {
	var count int
	err := m.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM builtin_project_installations WHERE project_id = ? AND tenant_id = ?",
		projectID, tenantID).Scan(&count)
	return err == nil && count > 0
}

func (m *Manager) recordInstallation(ctx context.Context, projectID, tenantID string, result *interfaces.InstallResult) error {
	configJSON, _ := json.Marshal(result.Config)
	metadataJSON, _ := json.Marshal(result.Metadata)

	query := `
		INSERT INTO builtin_project_installations
		(id, project_id, tenant_id, version, config, endpoint, admin_url,
		 success, message, metadata, installed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := m.db.ExecContext(ctx, query,
		fmt.Sprintf("%s_%s", projectID, tenantID), // Generate unique ID
		projectID,
		tenantID,
		result.Version,
		string(configJSON),
		result.Endpoint,
		result.AdminURL,
		result.Success,
		result.Message,
		string(metadataJSON),
		result.InstalledAt)

	return err
}

func (m *Manager) removeInstallationRecord(ctx context.Context, projectID, tenantID string) error {
	query := "DELETE FROM builtin_project_installations WHERE project_id = ? AND tenant_id = ?"
	_, err := m.db.ExecContext(ctx, query, projectID, tenantID)
	return err
}
