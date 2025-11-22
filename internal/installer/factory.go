package installer

import (
	"database/sql"
	"fmt"

	"github.com/guileen/metabase/pkg/interfaces"
	"go.uber.org/zap"
)

// CMSFactory implements the ProjectFactory interface for CMS projects
type CMSFactory struct{}

// NewCMSFactory creates a new CMS factory
func NewCMSFactory() interfaces.ProjectFactory {
	return &CMSFactory{}
}

// SupportedProjectType returns the project type this factory supports
func (f *CMSFactory) SupportedProjectType() string {
	return interfaces.ProjectTypeCMS
}

// CreateInstaller creates a new CMS installer
func (f *CMSFactory) CreateInstaller(db *sql.DB, logger interface{}) (interfaces.ProjectInstaller, error) {
	zapLogger, ok := logger.(*zap.Logger)
	if !ok && logger != nil {
		return nil, fmt.Errorf("logger must be *zap.Logger, got %T", logger)
	}
	return NewCMSInstaller(db, zapLogger), nil
}

// AuthGatewayFactory implements the ProjectFactory interface for Auth Gateway projects
type AuthGatewayFactory struct{}

// NewAuthGatewayFactory creates a new Auth Gateway factory
func NewAuthGatewayFactory() interfaces.ProjectFactory {
	return &AuthGatewayFactory{}
}

// SupportedProjectType returns the project type this factory supports
func (f *AuthGatewayFactory) SupportedProjectType() string {
	return interfaces.ProjectTypeAuthGateway
}

// CreateInstaller creates a new Auth Gateway installer
func (f *AuthGatewayFactory) CreateInstaller(db *sql.DB, logger interface{}) (interfaces.ProjectInstaller, error) {
	zapLogger, ok := logger.(*zap.Logger)
	if !ok && logger != nil {
		return nil, fmt.Errorf("logger must be *zap.Logger, got %T", logger)
	}
	return NewAuthGatewayInstaller(db, zapLogger), nil
}
