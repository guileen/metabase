package builtin

// This package now delegates to the new interfaces and installation packages
// The core interfaces have been moved to pkg/interfaces/
// The installation logic has been moved to pkg/installation/
//
// This file is kept for backward compatibility during the migration.

import (
	"github.com/guileen/metabase/pkg/interfaces"
)

// Type aliases for backward compatibility
type (
	// Re-export core interfaces for backward compatibility
	InstallResult        = interfaces.InstallResult
	InstallRequest       = interfaces.InstallRequest
	ProjectInstaller     = interfaces.ProjectInstaller
	ProjectFactory       = interfaces.ProjectFactory
	ProjectHealthChecker = interfaces.ProjectHealthChecker
	HealthStatus         = interfaces.HealthStatus
	InstallerRegistry    = interfaces.InstallerRegistry
	ProjectConfig        = interfaces.ProjectConfig
	DeploymentRequest    = interfaces.DeploymentRequest
	InstallationState    = interfaces.InstallationState

	// Legacy BuiltinProjectManager type alias
	BuiltinProjectManager = Manager
)

// Constants for backward compatibility
const (
	ProjectTypeCMS         = interfaces.ProjectTypeCMS
	ProjectTypeAuthGateway = interfaces.ProjectTypeAuthGateway
	ProjectTypeFileManager = interfaces.ProjectTypeFileManager
	ProjectTypeAnalytics   = interfaces.ProjectTypeAnalytics

	InstallationStatusPending    = interfaces.InstallationStatusPending
	InstallationStatusInstalling = interfaces.InstallationStatusInstalling
	InstallationStatusCompleted  = interfaces.InstallationStatusCompleted
	InstallationStatusFailed     = interfaces.InstallationStatusFailed

	HealthStatusHealthy   = interfaces.HealthStatusHealthy
	HealthStatusUnhealthy = interfaces.HealthStatusUnhealthy
	HealthStatusDegraded  = interfaces.HealthStatusDegraded
)

// Deprecated: Use installation.NewRegistry() instead
func NewInstallerRegistry() interfaces.InstallerRegistry {
	// Import the installation package to create registry
	// This is a compatibility wrapper
	return nil // TODO: Properly implement this after cleaning up circular dependencies
}
