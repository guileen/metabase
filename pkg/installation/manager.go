package installation

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/guileen/metabase/pkg/interfaces"
	"go.uber.org/zap"
)

// Manager manages project installations
type Manager struct {
	db       *sql.DB
	registry interfaces.InstallerRegistry
	logger   *zap.Logger
}

// NewManager creates a new installation manager
func NewManager(db *sql.DB, registry interfaces.InstallerRegistry, logger *zap.Logger) *Manager {
	return &Manager{
		db:       db,
		registry: registry,
		logger:   logger,
	}
}

// InstallProject installs a project
func (m *Manager) InstallProject(ctx context.Context, req *interfaces.InstallRequest) (*interfaces.InstallResult, error) {
	m.logger.Info("Installing project",
		zap.String("project_type", req.ProjectType),
		zap.String("tenant_id", req.TenantID))

	// Get installer
	installer, err := m.registry.CreateInstaller(req.ProjectType, m.db, m.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create installer: %w", err)
	}

	// Check if already installed
	alreadyInstalled, err := installer.CheckInstallation(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check installation status: %w", err)
	}

	if alreadyInstalled {
		return &interfaces.InstallResult{
			Success: false,
			Message: fmt.Sprintf("Project '%s' is already installed for tenant '%s'", req.ProjectType, req.TenantID),
		}, nil
	}

	// Validate configuration
	if err := installer.ValidateConfiguration(req.Config); err != nil {
		return &interfaces.InstallResult{
			Success: false,
			Message: fmt.Sprintf("Configuration validation failed: %v", err),
		}, nil
	}

	// Check dependencies
	dependencies := installer.GetDependencies()
	for _, dep := range dependencies {
		depInstaller, err := m.registry.CreateInstaller(dep, m.db, m.logger)
		if err != nil {
			return &interfaces.InstallResult{
				Success: false,
				Message: fmt.Sprintf("Failed to get dependency installer for '%s': %v", dep, err),
			}, nil
		}

		installed, err := depInstaller.CheckInstallation(ctx, req.TenantID)
		if err != nil {
			return &interfaces.InstallResult{
				Success: false,
				Message: fmt.Sprintf("Failed to check dependency '%s' installation: %v", dep, err),
			}, nil
		}

		if !installed {
			return &interfaces.InstallResult{
				Success: false,
				Message: fmt.Sprintf("Dependency '%s' is not installed", dep),
			}, nil
		}
	}

	// Perform installation
	result, err := installer.Install(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("installation failed: %w", err)
	}

	if result.Success {
		m.logger.Info("Project installed successfully",
			zap.String("project_type", req.ProjectType),
			zap.String("tenant_id", req.TenantID),
			zap.String("project_id", result.ProjectID))
	}

	return result, nil
}

// UninstallProject uninstalls a project
func (m *Manager) UninstallProject(ctx context.Context, projectType, tenantID string) error {
	m.logger.Info("Uninstalling project",
		zap.String("project_type", projectType),
		zap.String("tenant_id", tenantID))

	// Get installer
	installer, err := m.registry.CreateInstaller(projectType, m.db, m.logger)
	if err != nil {
		return fmt.Errorf("failed to create installer: %w", err)
	}

	// Check if installed
	installed, err := installer.CheckInstallation(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to check installation status: %w", err)
	}

	if !installed {
		return fmt.Errorf("project '%s' is not installed for tenant '%s'", projectType, tenantID)
	}

	// Perform uninstallation
	if err := installer.Uninstall(ctx, tenantID); err != nil {
		return fmt.Errorf("uninstallation failed: %w", err)
	}

	m.logger.Info("Project uninstalled successfully",
		zap.String("project_type", projectType),
		zap.String("tenant_id", tenantID))

	return nil
}

// GetInstallationStatus returns the installation status for a project
func (m *Manager) GetInstallationStatus(ctx context.Context, projectType, tenantID string) (map[string]interface{}, error) {
	installer, err := m.registry.CreateInstaller(projectType, m.db, m.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create installer: %w", err)
	}

	return installer.GetInstallationStatus(ctx, tenantID)
}

// ListSupportedProjects returns all supported project types
func (m *Manager) ListSupportedProjects() []string {
	return m.registry.ListSupportedTypes()
}

// CheckProjectHealth checks the health of an installed project
func (m *Manager) CheckProjectHealth(ctx context.Context, projectType, tenantID string) (*interfaces.HealthStatus, error) {
	installer, err := m.registry.CreateInstaller(projectType, m.db, m.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create installer: %w", err)
	}

	// Check if project supports health checking
	if healthChecker, ok := installer.(interfaces.ProjectHealthChecker); ok {
		return healthChecker.CheckHealth(ctx, tenantID)
	}

	// Default health check
	installed, err := installer.CheckInstallation(ctx, tenantID)
	if err != nil {
		return &interfaces.HealthStatus{
			Status:      interfaces.HealthStatusUnhealthy,
			Message:     fmt.Sprintf("Failed to check installation: %v", err),
			LastChecked: getCurrentTime(),
		}, nil
	}

	if !installed {
		return &interfaces.HealthStatus{
			Status:      interfaces.HealthStatusUnhealthy,
			Message:     fmt.Sprintf("Project '%s' is not installed", projectType),
			LastChecked: getCurrentTime(),
		}, nil
	}

	return &interfaces.HealthStatus{
		Status:      interfaces.HealthStatusHealthy,
		Message:     "Project is running normally",
		LastChecked: getCurrentTime(),
	}, nil
}

// getCurrentTime returns the current time (extracted for easier testing)
func getCurrentTime() time.Time {
	return time.Now()
}
