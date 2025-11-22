package installer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/guileen/metabase/pkg/interfaces"
	"go.uber.org/zap"
)

// AuthGatewayInstaller handles Auth Gateway installation and initialization
type AuthGatewayInstaller struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewAuthGatewayInstaller creates a new auth gateway installer
func NewAuthGatewayInstaller(db *sql.DB, logger *zap.Logger) interfaces.ProjectInstaller {
	return &AuthGatewayInstaller{
		db:     db,
		logger: logger,
	}
}

// AuthGatewayInstallRequest represents the auth gateway installation request
type AuthGatewayInstallRequest struct {
	// Basic configuration
	EnableLocalAuth    bool   `json:"enable_local_auth"`
	DefaultProvider    string `json:"default_provider"`
	SessionTimeout     int    `json:"session_timeout_minutes"`
	RefreshTokenExpiry int    `json:"refresh_token_expiry_hours"`
	MaxLoginAttempts   int    `json:"max_login_attempts"`
	LockoutDuration    int    `json:"lockout_duration_minutes"`

	// Password policy
	PasswordPolicy *PasswordPolicy `json:"password_policy"`

	// Multi-tenant settings
	EnableTenantIsolation bool              `json:"enable_tenant_isolation"`
	SharedUserDatabase    bool              `json:"shared_user_database"`
	CrossTenantAuth       bool              `json:"cross_tenant_auth"`
	DefaultTenantRoles    map[string]string `json:"default_tenant_roles"`

	// OAuth providers
	EnableOAuth2    bool          `json:"enable_oauth2"`
	GoogleOAuth2    *OAuth2Config `json:"google_oauth2,omitempty"`
	GitHubOAuth2    *OAuth2Config `json:"github_oauth2,omitempty"`
	MicrosoftOAuth2 *OAuth2Config `json:"microsoft_oauth2,omitempty"`

	// Features
	EnableMFA               bool `json:"enable_mfa"`
	EnablePasswordReset     bool `json:"enable_password_reset"`
	EnableEmailVerification bool `json:"enable_email_verification"`
	EnableAuditLog          bool `json:"enable_audit_log"`

	// Admin user setup
	CreateAdminUser  bool   `json:"create_admin_user"`
	AdminUsername    string `json:"admin_username"`
	AdminEmail       string `json:"admin_email"`
	AdminPassword    string `json:"admin_password"`
	AdminDisplayName string `json:"admin_display_name"`
}

// PasswordPolicy represents password security policy
type PasswordPolicy struct {
	MinLength        int  `json:"min_length"`
	RequireUppercase bool `json:"require_uppercase"`
	RequireLowercase bool `json:"require_lowercase"`
	RequireNumbers   bool `json:"require_numbers"`
	RequireSymbols   bool `json:"require_symbols"`
	HistoryCount     int  `json:"history_count"`
	MaxAge           int  `json:"max_age_days"`
}

// OAuth2Config represents OAuth2 provider configuration
type OAuth2Config struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURL  string   `json:"redirect_url"`
	Scopes       []string `json:"scopes"`
	AuthURL      string   `json:"auth_url"`
	TokenURL     string   `json:"token_url"`
	UserInfoURL  string   `json:"user_info_url"`
	Enabled      bool     `json:"enabled"`
}

// AuthGatewayInstallResponse represents the auth gateway installation response
type AuthGatewayInstallResponse struct {
	Success     bool           `json:"success"`
	Message     string         `json:"message"`
	GatewayURL  string         `json:"gateway_url"`
	APIURL      string         `json:"api_url"`
	AdminURL    string         `json:"admin_url"`
	InstalledAt string         `json:"installed_at"`
	Version     string         `json:"version"`
	AdminUser   *AdminUserInfo `json:"admin_user,omitempty"`
}

// AdminUserInfo represents created admin user info
type AdminUserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"` // Only shown once during installation
}

// Name returns the project name
func (i *AuthGatewayInstaller) Name() string {
	return "Auth Gateway"
}

// Version returns the project version
func (i *AuthGatewayInstaller) Version() string {
	return "1.0.0"
}

// Type returns the project type
func (i *AuthGatewayInstaller) Type() string {
	return interfaces.ProjectTypeAuthGateway
}

// CheckInstallation checks if auth gateway is already installed
func (i *AuthGatewayInstaller) CheckInstallation(ctx context.Context, tenantID string) (bool, error) {
	var count int
	err := i.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'auth_users'").Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check installation: %w", err)
	}

	if count == 0 {
		return false, nil
	}

	// Check if installation is completed
	var settingCount int
	err = i.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM auth_gateway_settings WHERE key = 'auth_gateway_installed'").Scan(&settingCount)

	if err != nil {
		return false, fmt.Errorf("failed to check installation status: %w", err)
	}

	return settingCount > 0, nil
}

// Install performs the auth gateway installation
func (i *AuthGatewayInstaller) Install(ctx context.Context, req *interfaces.InstallRequest) (*interfaces.InstallResult, error) {
	// Create install request from config
	authReq := &AuthGatewayInstallRequest{
		EnableLocalAuth:         true,
		DefaultProvider:         "local",
		SessionTimeout:          1440, // 24 hours in minutes
		RefreshTokenExpiry:      168,  // 7 days in hours
		MaxLoginAttempts:        5,
		LockoutDuration:         15, // 15 minutes
		EnableTenantIsolation:   true,
		SharedUserDatabase:      false,
		CrossTenantAuth:         false,
		EnableOAuth2:            false,
		EnableMFA:               false,
		EnablePasswordReset:     true,
		EnableEmailVerification: false,
		EnableAuditLog:          true,
		CreateAdminUser:         false,
		PasswordPolicy: &PasswordPolicy{
			MinLength:        8,
			RequireUppercase: true,
			RequireLowercase: true,
			RequireNumbers:   true,
			RequireSymbols:   false,
			HistoryCount:     3,
			MaxAge:           90,
		},
	}

	if req.Config != nil {
		if enableLocal, ok := req.Config["enable_local_auth"].(bool); ok {
			authReq.EnableLocalAuth = enableLocal
		}
		if defaultProvider, ok := req.Config["default_provider"].(string); ok {
			authReq.DefaultProvider = defaultProvider
		}
		if sessionTimeout, ok := req.Config["session_timeout_minutes"].(int); ok {
			authReq.SessionTimeout = sessionTimeout
		}
		if enableTenantIsolation, ok := req.Config["enable_tenant_isolation"].(bool); ok {
			authReq.EnableTenantIsolation = enableTenantIsolation
		}
		if enableOAuth2, ok := req.Config["enable_oauth2"].(bool); ok {
			authReq.EnableOAuth2 = enableOAuth2
		}
		if enableMFA, ok := req.Config["enable_mfa"].(bool); ok {
			authReq.EnableMFA = enableMFA
		}
		if enablePasswordReset, ok := req.Config["enable_password_reset"].(bool); ok {
			authReq.EnablePasswordReset = enablePasswordReset
		}
		if enableEmailVerification, ok := req.Config["enable_email_verification"].(bool); ok {
			authReq.EnableEmailVerification = enableEmailVerification
		}
		if enableAuditLog, ok := req.Config["enable_audit_log"].(bool); ok {
			authReq.EnableAuditLog = enableAuditLog
		}

		// Admin user setup
		if createAdmin, ok := req.Config["create_admin_user"].(bool); ok && createAdmin {
			authReq.CreateAdminUser = true
			if username, ok := req.Config["admin_username"].(string); ok {
				authReq.AdminUsername = username
			}
			if email, ok := req.Config["admin_email"].(string); ok {
				authReq.AdminEmail = email
			}
			if password, ok := req.Config["admin_password"].(string); ok {
				authReq.AdminPassword = password
			}
			if displayName, ok := req.Config["admin_display_name"].(string); ok {
				authReq.AdminDisplayName = displayName
			}
		}
	}

	// Call the existing install method
	response, err := i.installWithRequest(ctx, authReq, req.TenantID)
	if err != nil {
		return &interfaces.InstallResult{
			Success: false,
			Message: err.Error(),
		}, err
	}

	// Convert response to InstallResult
	projectID := fmt.Sprintf("%s-%s", req.ProjectType, req.TenantID)
	return &interfaces.InstallResult{
		Success:     response.Success,
		Message:     response.Message,
		ProjectID:   projectID,
		ProjectType: req.ProjectType,
		Version:     response.Version,
		InstalledAt: time.Now(),
		Endpoint:    response.APIURL,
		AdminURL:    response.AdminURL,
		Config:      req.Config,
		Metadata: map[string]interface{}{
			"admin_user": response.AdminUser,
		},
	}, nil
}

// installWithRequest performs the auth gateway installation with specific request
func (i *AuthGatewayInstaller) installWithRequest(ctx context.Context, req *AuthGatewayInstallRequest, tenantID string) (*AuthGatewayInstallResponse, error) {
	i.logger.Info("Starting Auth Gateway installation",
		zap.String("tenant_id", tenantID),
		zap.Bool("enable_local_auth", req.EnableLocalAuth),
		zap.String("default_provider", req.DefaultProvider))

	// Validate request
	if err := i.validateInstallRequest(req); err != nil {
		return &AuthGatewayInstallResponse{
			Success: false,
			Message: fmt.Sprintf("Validation failed: %v", err),
		}, nil
	}

	// Check if already installed
	installed, err := i.CheckInstallation(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check installation status: %w", err)
	}

	if installed {
		return &AuthGatewayInstallResponse{
			Success: false,
			Message: "Auth Gateway is already installed",
		}, nil
	}

	// Begin transaction
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			i.logger.Warn("Failed to rollback transaction", zap.Error(err))
		}
	}()

	// Run installation steps
	if err := i.runMigrations(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	if err := i.insertDefaultSettings(ctx, tx, tenantID, req); err != nil {
		return nil, fmt.Errorf("failed to insert default settings: %w", err)
	}

	var adminUser *AdminUserInfo
	if req.CreateAdminUser {
		adminUser, err = i.createAdminUser(ctx, tx, tenantID, req)
		if err != nil {
			return nil, fmt.Errorf("failed to create admin user: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit installation: %w", err)
	}

	i.logger.Info("Auth Gateway installation completed successfully",
		zap.String("tenant_id", tenantID),
		zap.Bool("create_admin_user", req.CreateAdminUser))

	return &AuthGatewayInstallResponse{
		Success:     true,
		Message:     "Auth Gateway installed successfully",
		GatewayURL:  "/api/v1/auth",
		APIURL:      "/api/v1/auth",
		AdminURL:    "/admin/auth",
		InstalledAt: time.Now().Format(time.RFC3339),
		Version:     "1.0.0",
		AdminUser:   adminUser,
	}, nil
}

// validateInstallRequest validates the installation request
func (i *AuthGatewayInstaller) validateInstallRequest(req *AuthGatewayInstallRequest) error {
	if req == nil {
		return fmt.Errorf("installation request is required")
	}

	// Set defaults
	if req.DefaultProvider == "" {
		req.DefaultProvider = "local"
	}
	if req.SessionTimeout == 0 {
		req.SessionTimeout = 1440 // 24 hours in minutes
	}
	if req.RefreshTokenExpiry == 0 {
		req.RefreshTokenExpiry = 168 // 7 days in hours
	}
	if req.MaxLoginAttempts == 0 {
		req.MaxLoginAttempts = 5
	}
	if req.LockoutDuration == 0 {
		req.LockoutDuration = 15 // 15 minutes
	}

	// Validate password policy
	if req.PasswordPolicy == nil {
		req.PasswordPolicy = &PasswordPolicy{
			MinLength:        8,
			RequireUppercase: true,
			RequireLowercase: true,
			RequireNumbers:   true,
			RequireSymbols:   false,
			HistoryCount:     3,
			MaxAge:           90,
		}
	}

	// Validate admin user setup if enabled
	if req.CreateAdminUser {
		if req.AdminUsername == "" {
			return fmt.Errorf("admin username is required when creating admin user")
		}
		if req.AdminEmail == "" {
			return fmt.Errorf("admin email is required when creating admin user")
		}
		if req.AdminPassword == "" {
			return fmt.Errorf("admin password is required when creating admin user")
		}
		if len(req.AdminPassword) < req.PasswordPolicy.MinLength {
			return fmt.Errorf("admin password must be at least %d characters", req.PasswordPolicy.MinLength)
		}
	}

	return nil
}

// runMigrations runs the auth gateway database migrations
func (i *AuthGatewayInstaller) runMigrations(ctx context.Context, tx *sql.Tx) error {
	// Read migration file
	migrationPath := filepath.Join("internal", "biz", "domain", "authgateway", "migrations.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Execute migration
	_, err = tx.ExecContext(ctx, string(migrationSQL))
	if err != nil {
		return fmt.Errorf("failed to execute migrations: %w", err)
	}

	return nil
}

// insertDefaultSettings inserts default auth gateway settings
func (i *AuthGatewayInstaller) insertDefaultSettings(ctx context.Context, tx *sql.Tx, tenantID string, req *AuthGatewayInstallRequest) error {
	now := time.Now()

	settings := []struct {
		key         string
		value       interface{}
		description string
		category    string
		isPublic    bool
	}{
		// Installation status
		{
			key:         "auth_gateway_installed",
			value:       true,
			description: "Auth Gateway installation status",
			category:    "system",
			isPublic:    false,
		},
		{
			key:         "installed_at",
			value:       now.Format(time.RFC3339),
			description: "Auth Gateway installation timestamp",
			category:    "system",
			isPublic:    false,
		},
		{
			key:         "installed_version",
			value:       "1.0.0",
			description: "Auth Gateway version",
			category:    "system",
			isPublic:    false,
		},

		// Authentication settings
		{
			key:         "enable_local_auth",
			value:       req.EnableLocalAuth,
			description: "Enable local username/password authentication",
			category:    "authentication",
			isPublic:    true,
		},
		{
			key:         "default_provider",
			value:       req.DefaultProvider,
			description: "Default authentication provider",
			category:    "authentication",
			isPublic:    true,
		},
		{
			key:         "session_timeout_minutes",
			value:       req.SessionTimeout,
			description: "Session timeout in minutes",
			category:    "session",
			isPublic:    false,
		},

		// Multi-tenant settings
		{
			key:         "enable_tenant_isolation",
			value:       req.EnableTenantIsolation,
			description: "Enable tenant isolation for user authentication",
			category:    "multitenancy",
			isPublic:    false,
		},
		{
			key:         "shared_user_database",
			value:       req.SharedUserDatabase,
			description: "Share user database across tenants",
			category:    "multitenancy",
			isPublic:    false,
		},

		// Feature settings
		{
			key:         "enable_oauth2",
			value:       req.EnableOAuth2,
			description: "Enable OAuth2 authentication providers",
			category:    "oauth",
			isPublic:    true,
		},
		{
			key:         "enable_mfa",
			value:       req.EnableMFA,
			description: "Enable multi-factor authentication",
			category:    "features",
			isPublic:    false,
		},
		{
			key:         "enable_password_reset",
			value:       req.EnablePasswordReset,
			description: "Enable password reset functionality",
			category:    "features",
			isPublic:    false,
		},
		{
			key:         "enable_email_verification",
			value:       req.EnableEmailVerification,
			description: "Require email verification for new users",
			category:    "features",
			isPublic:    false,
		},
		{
			key:         "enable_audit_log",
			value:       req.EnableAuditLog,
			description: "Enable authentication audit logging",
			category:    "features",
			isPublic:    false,
		},
	}

	for _, setting := range settings {
		id := uuid.New().String()
		var valueJSON string

		switch v := setting.value.(type) {
		case string:
			valueJSON = fmt.Sprintf(`"%s"`, v)
		case bool:
			valueJSON = fmt.Sprintf(`%t`, v)
		case int:
			valueJSON = fmt.Sprintf(`%d`, v)
		default:
			jsonBytes, _ := json.Marshal(v)
			valueJSON = string(jsonBytes)
		}

		_, err := tx.ExecContext(ctx, `
			INSERT INTO auth_gateway_settings (
				id, tenant_id, key, value, description, category, is_public, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		`, id, tenantID, setting.key, valueJSON, setting.description, setting.category, setting.isPublic, now)

		if err != nil {
			return fmt.Errorf("failed to insert setting %s: %w", setting.key, err)
		}
	}

	return nil
}

// createAdminUser creates the initial admin user
func (i *AuthGatewayInstaller) createAdminUser(ctx context.Context, tx *sql.Tx, tenantID string, req *AuthGatewayInstallRequest) (*AdminUserInfo, error) {
	// Generate password hash (this would use the actual password hashing function)
	passwordHash := "hashed_password_placeholder" // TODO: Use actual password hashing

	userID := uuid.New().String()
	now := time.Now()

	// Insert admin user
	_, err := tx.ExecContext(ctx, `
		INSERT INTO auth_users (
			id, username, email, password_hash, display_name, tenant_id, provider,
			is_active, is_verified, roles, attributes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12)
	`,
		userID,
		req.AdminUsername,
		req.AdminEmail,
		passwordHash,
		req.AdminDisplayName,
		tenantID,
		"local",
		true,
		!req.EnableEmailVerification, // Auto-verify if email verification is disabled
		`["admin", "super_admin"]`,
		`{"is_system_admin": true}`,
		now,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}

	// Log admin user creation
	i.logger.Info("Created admin user",
		zap.String("user_id", userID),
		zap.String("username", req.AdminUsername),
		zap.String("email", req.AdminEmail))

	return &AdminUserInfo{
		ID:       userID,
		Username: req.AdminUsername,
		Email:    req.AdminEmail,
		Password: req.AdminPassword, // Only shown once during installation
	}, nil
}

// GetInstallationStatus returns the current installation status
func (i *AuthGatewayInstaller) GetInstallationStatus(ctx context.Context, tenantID string) (map[string]interface{}, error) {
	installed, err := i.CheckInstallation(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	status := map[string]interface{}{
		"installed": installed,
	}

	if installed {
		// Get additional status information
		var settingValue sql.NullString
		var installedAt time.Time

		err := i.db.QueryRowContext(ctx, `
			SELECT COALESCE(
				(SELECT value::text FROM auth_gateway_settings WHERE key = 'installed_at'),
				'1970-01-01T00:00:00Z'
			)::timestamp as installed_at,
			COALESCE(
				(SELECT value::text FROM auth_gateway_settings WHERE key = 'enable_local_auth'),
				'true'
			)::boolean as enable_local_auth
		`).Scan(&installedAt, &settingValue)

		if err == nil {
			status["installed_at"] = installedAt.Format(time.RFC3339)
			if settingValue.Valid {
				status["enable_local_auth"] = strings.Trim(settingValue.String, `"`) == "true"
			}
		}
	}

	return status, nil
}

// GetConfigurationSchema returns the configuration schema
func (i *AuthGatewayInstaller) GetConfigurationSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"enable_local_auth": map[string]interface{}{
				"type":        "boolean",
				"default":     true,
				"description": "Enable local authentication",
			},
			"enable_oauth2": map[string]interface{}{
				"type":        "boolean",
				"default":     false,
				"description": "Enable OAuth2 authentication",
			},
			"session_timeout": map[string]interface{}{
				"type":        "integer",
				"default":     3600,
				"description": "Session timeout in seconds",
			},
		},
	}
}

// ValidateConfiguration validates the provided configuration
func (i *AuthGatewayInstaller) ValidateConfiguration(config map[string]interface{}) error {
	if config == nil {
		return nil
	}

	// Validate session timeout
	if timeout, ok := config["session_timeout"]; ok {
		if timeoutInt, ok := timeout.(int); ok {
			if timeoutInt < 60 || timeoutInt > 86400 {
				return fmt.Errorf("session_timeout must be between 60 and 86400 seconds")
			}
		} else {
			return fmt.Errorf("session_timeout must be an integer")
		}
	}

	return nil
}

// CheckDependencies checks if required dependencies are met
func (i *AuthGatewayInstaller) CheckDependencies(ctx context.Context, db *sql.DB) error {
	// Check if database is accessible
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	return nil
}

// GetDependencies returns the list of project dependencies
func (i *AuthGatewayInstaller) GetDependencies() []string {
	// Auth Gateway doesn't have any hard dependencies currently
	return []string{}
}

// Uninstall removes the Auth Gateway from the specified tenant
func (i *AuthGatewayInstaller) Uninstall(ctx context.Context, tenantID string) error {
	// For safety, we don't actually delete data on uninstall
	i.logger.Info("Auth Gateway uninstalled", zap.String("tenant_id", tenantID))
	return nil
}
