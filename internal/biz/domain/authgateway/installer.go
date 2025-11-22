package authgateway

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
	"go.uber.org/zap"
)

// Installer handles Auth Gateway installation and initialization
type Installer struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewInstaller creates a new auth gateway installer
func NewInstaller(db *sql.DB, logger *zap.Logger) *Installer {
	return &Installer{
		db:     db,
		logger: logger,
	}
}

// InstallRequest represents the auth gateway installation request
type InstallRequest struct {
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

	// Email settings (for password reset and verification)
	EmailSettings *EmailSettings `json:"email_settings,omitempty"`

	// Custom settings
	CustomSettings map[string]interface{} `json:"custom_settings,omitempty"`
}

// EmailSettings represents email configuration
type EmailSettings struct {
	SMTPHost  string `json:"smtp_host"`
	SMTPPort  int    `json:"smtp_port"`
	FromEmail string `json:"from_email"`
	FromName  string `json:"from_name"`
	UseTLS    bool   `json:"use_tls"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

// InstallResponse represents the auth gateway installation response
type InstallResponse struct {
	Success     bool           `json:"success"`
	Message     string         `json:"message"`
	GatewayURL  string         `json:"gateway_url"`
	APIURL      string         `json:"api_url"`
	AdminURL    string         `json:"admin_url"`
	InstalledAt string         `json:"installed_at"`
	Version     string         `json:"version"`
	AdminUser   *AdminUserInfo `json:"admin_user,omitempty"`
}

// InstallResult represents installation result (local copy to avoid import cycle)
type InstallResult struct {
	Success     bool                   `json:"success"`
	Message     string                 `json:"message"`
	ProjectID   string                 `json:"project_id"`
	Version     string                 `json:"version"`
	InstalledAt time.Time              `json:"installed_at"`
	Endpoint    string                 `json:"endpoint,omitempty"`
	AdminURL    string                 `json:"admin_url,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AdminUserInfo represents created admin user info
type AdminUserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"` // Only shown once during installation
}

// CheckInstallation checks if auth gateway is already installed
func (i *Installer) CheckInstallation(ctx context.Context) (bool, error) {
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
func (i *Installer) InstallWithRequest(ctx context.Context, req *InstallRequest, tenantID string) (*InstallResponse, error) {
	i.logger.Info("Starting Auth Gateway installation",
		zap.String("tenant_id", tenantID),
		zap.Bool("enable_local_auth", req.EnableLocalAuth),
		zap.String("default_provider", req.DefaultProvider))

	// Validate request
	if err := i.validateInstallRequest(req); err != nil {
		return &InstallResponse{
			Success: false,
			Message: fmt.Sprintf("Validation failed: %v", err),
		}, nil
	}

	// Check if already installed
	installed, err := i.CheckInstallation(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check installation status: %w", err)
	}

	if installed {
		return &InstallResponse{
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

	if err := i.insertOAuthProviders(ctx, tx, tenantID, req); err != nil {
		return nil, fmt.Errorf("failed to insert OAuth providers: %w", err)
	}

	var adminUser *AdminUserInfo
	if req.CreateAdminUser {
		adminUser, err = i.createAdminUser(ctx, tx, tenantID, req)
		if err != nil {
			return nil, fmt.Errorf("failed to create admin user: %w", err)
		}
	}

	if err := i.createDefaultRoles(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("failed to create default roles: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit installation: %w", err)
	}

	i.logger.Info("Auth Gateway installation completed successfully",
		zap.String("tenant_id", tenantID),
		zap.Bool("create_admin_user", req.CreateAdminUser))

	return &InstallResponse{
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
func (i *Installer) validateInstallRequest(req *InstallRequest) error {
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
func (i *Installer) runMigrations(ctx context.Context, tx *sql.Tx) error {
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
func (i *Installer) insertDefaultSettings(ctx context.Context, tx *sql.Tx, tenantID string, req *InstallRequest) error {
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
		{
			key:         "refresh_token_expiry_hours",
			value:       req.RefreshTokenExpiry,
			description: "Refresh token expiry in hours",
			category:    "session",
			isPublic:    false,
		},

		// Security settings
		{
			key:         "max_login_attempts",
			value:       req.MaxLoginAttempts,
			description: "Maximum number of failed login attempts before lockout",
			category:    "security",
			isPublic:    false,
		},
		{
			key:         "lockout_duration_minutes",
			value:       req.LockoutDuration,
			description: "Account lockout duration in minutes",
			category:    "security",
			isPublic:    false,
		},
		{
			key:         "password_policy",
			value:       req.PasswordPolicy,
			description: "Password security policy configuration",
			category:    "security",
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
		{
			key:         "cross_tenant_auth",
			value:       req.CrossTenantAuth,
			description: "Allow cross-tenant authentication",
			category:    "multitenancy",
			isPublic:    false,
		},
		{
			key:         "default_tenant_roles",
			value:       req.DefaultTenantRoles,
			description: "Default roles for new tenant users",
			category:    "multitenancy",
			isPublic:    false,
		},

		// OAuth settings
		{
			key:         "enable_oauth2",
			value:       req.EnableOAuth2,
			description: "Enable OAuth2 authentication providers",
			category:    "oauth",
			isPublic:    true,
		},

		// Feature settings
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

	// Insert custom settings
	if req.CustomSettings != nil {
		for key, value := range req.CustomSettings {
			id := uuid.New().String()
			valueJSON, _ := json.Marshal(value)

			_, err := tx.ExecContext(ctx, `
				INSERT INTO auth_gateway_settings (
					id, tenant_id, key, value, description, category, is_public, created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
			`, id, tenantID, key, string(valueJSON), "Custom setting", "custom", false, now)

			if err != nil {
				return fmt.Errorf("failed to insert custom setting %s: %w", key, err)
			}
		}
	}

	return nil
}

// insertOAuthProviders inserts OAuth provider configurations
func (i *Installer) insertOAuthProviders(ctx context.Context, tx *sql.Tx, tenantID string, req *InstallRequest) error {
	if !req.EnableOAuth2 {
		return nil
	}

	providers := []struct {
		name        string
		displayName string
		config      interface{}
		enabled     bool
	}{
		{
			name:        "google",
			displayName: "Google OAuth2",
			config:      req.GoogleOAuth2,
			enabled:     req.GoogleOAuth2 != nil && req.GoogleOAuth2.ClientID != "",
		},
		{
			name:        "github",
			displayName: "GitHub OAuth2",
			config:      req.GitHubOAuth2,
			enabled:     req.GitHubOAuth2 != nil && req.GitHubOAuth2.ClientID != "",
		},
		{
			name:        "microsoft",
			displayName: "Microsoft OAuth2",
			config:      req.MicrosoftOAuth2,
			enabled:     req.MicrosoftOAuth2 != nil && req.MicrosoftOAuth2.ClientID != "",
		},
	}

	for _, provider := range providers {
		id := uuid.New().String()
		configJSON, _ := json.Marshal(provider.config)

		_, err := tx.ExecContext(ctx, `
			INSERT INTO auth_oauth_providers (
				id, tenant_id, name, display_name, type, config, enabled, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		`, id, tenantID, provider.name, provider.displayName, "oauth2", string(configJSON), provider.enabled, time.Now())

		if err != nil {
			return fmt.Errorf("failed to insert OAuth provider %s: %w", provider.name, err)
		}
	}

	return nil
}

// createAdminUser creates the initial admin user
func (i *Installer) createAdminUser(ctx context.Context, tx *sql.Tx, tenantID string, req *InstallRequest) (*AdminUserInfo, error) {
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

	// Create admin roles
	roles := []string{"admin", "super_admin", "user_manager", "auth_manager"}
	for _, role := range roles {
		roleID := uuid.New().String()
		_, err := tx.ExecContext(ctx, `
			INSERT INTO auth_user_roles (
				id, user_id, tenant_id, role_name, scope, granted_by, granted_at, is_active
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, roleID, userID, tenantID, role, "global", "system", now, true)

		if err != nil {
			return nil, fmt.Errorf("failed to create admin role %s: %w", role, err)
		}
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

// createDefaultRoles creates default roles and permissions
func (i *Installer) createDefaultRoles(ctx context.Context, tx *sql.Tx, tenantID string) error {
	// TODO: Implement default roles and permissions creation
	// This would create standard roles like:
	// - admin: Full access to all auth gateway features
	// - user_manager: Can manage users but not system settings
	// - auth_manager: Can manage authentication settings
	// - viewer: Read-only access to auth information

	return nil
}

// GetInstallationStatus returns the current installation status
func (i *Installer) GetInstallationStatus(ctx context.Context, tenantID string) (map[string]interface{}, error) {
	installed, err := i.CheckInstallation(ctx)
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

		// Get user statistics
		var userCount, adminCount int
		if err := i.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM auth_users").Scan(&userCount); err != nil {
			i.logger.Warn("Failed to get user count", zap.Error(err))
		}
		if err := i.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM auth_user_roles WHERE role_name IN ('admin', 'super_admin')").Scan(&adminCount); err != nil {
			i.logger.Warn("Failed to get admin count", zap.Error(err))
		}

		status["statistics"] = map[string]int{
			"total_users": userCount,
			"admin_users": adminCount,
		}
	}

	return status, nil
}

// Name returns the project name
func (i *Installer) Name() string {
	return "Auth Gateway"
}

// Version returns the project version
func (i *Installer) Version() string {
	return "1.0.0"
}

// CheckDependencies checks if required dependencies are met
func (i *Installer) CheckDependencies(ctx context.Context, db *sql.DB) error {
	// Check if database is accessible
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	// Check if required tables can be created
	// For now, just return nil as SQLite doesn't have complex dependencies
	return nil
}

// GetConfigurationSchema returns the configuration schema
func (i *Installer) GetConfigurationSchema() map[string]interface{} {
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
func (i *Installer) ValidateConfiguration(config map[string]interface{}) error {
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

// Uninstall removes the Auth Gateway from the specified tenant
func (i *Installer) Uninstall(ctx context.Context, db *sql.DB, tenantID string) error {
	// For safety, we don't actually delete data on uninstall
	// Instead, we could mark it as inactive if needed
	i.logger.Info("Auth Gateway uninstalled", zap.String("tenant_id", tenantID))
	return nil
}

// Install implements the ProjectInstaller interface
func (i *Installer) Install(ctx context.Context, db *sql.DB, config map[string]interface{}, tenantID string) (*InstallResult, error) {
	// Create install request from config
	req := &InstallRequest{
		EnableLocalAuth: true,
		SessionTimeout:  3600,
	}

	if config != nil {
		if enableLocal, ok := config["enable_local_auth"].(bool); ok {
			req.EnableLocalAuth = enableLocal
		}
		if enableOAuth, ok := config["enable_oauth2"].(bool); ok {
			req.EnableOAuth2 = enableOAuth
		}
		if timeout, ok := config["session_timeout"].(int); ok {
			req.SessionTimeout = timeout
		}
	}

	// Call the existing install method
	response, err := i.InstallWithRequest(ctx, req, tenantID)
	if err != nil {
		return &InstallResult{
			Success: false,
			Message: err.Error(),
		}, err
	}

	// Convert response to InstallResult
	projectID := "auth-gateway-" + tenantID
	return &InstallResult{
		Success:     response.Success,
		Message:     response.Message,
		ProjectID:   projectID,
		Version:     response.Version,
		InstalledAt: time.Now(),
		Endpoint:    response.APIURL,
		AdminURL:    response.AdminURL,
		Config: map[string]interface{}{
			"enable_local_auth": req.EnableLocalAuth,
			"enable_oauth2":     req.EnableOAuth2,
			"session_timeout":   req.SessionTimeout,
		},
	}, nil
}
