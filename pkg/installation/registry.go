package installation

import (
	"database/sql"

	"github.com/guileen/metabase/pkg/interfaces"
)

// Registry implements the InstallerRegistry interface
type Registry struct {
	factories map[string]interfaces.ProjectFactory
}

// NewRegistry creates a new installer registry
func NewRegistry() interfaces.InstallerRegistry {
	return &Registry{
		factories: make(map[string]interfaces.ProjectFactory),
	}
}

// RegisterFactory registers a project factory
func (r *Registry) RegisterFactory(factory interfaces.ProjectFactory) {
	r.factories[factory.SupportedProjectType()] = factory
}

// GetFactory gets a factory for a project type
func (r *Registry) GetFactory(projectType string) (interfaces.ProjectFactory, bool) {
	factory, exists := r.factories[projectType]
	return factory, exists
}

// ListSupportedTypes returns all supported project types
func (r *Registry) ListSupportedTypes() []string {
	var types []string
	for projectType := range r.factories {
		types = append(types, projectType)
	}
	return types
}

// CreateInstaller creates an installer for the specified project type
func (r *Registry) CreateInstaller(projectType string, db *sql.DB, logger interface{}) (interfaces.ProjectInstaller, error) {
	factory, exists := r.GetFactory(projectType)
	if !exists {
		return nil, &interfaces.UnsupportedProjectTypeError{
			ProjectType: projectType,
			Supported:   r.ListSupportedTypes(),
		}
	}
	return factory.CreateInstaller(db, logger)
}
