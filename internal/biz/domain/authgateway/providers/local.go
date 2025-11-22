package providers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/guileen/metabase/internal/biz/domain/authgateway"
	"golang.org/x/crypto/bcrypt"
)

// LocalAuthProvider implements local username/password authentication
type LocalAuthProvider struct {
	db     *sql.DB
	config *LocalAuthConfig
}

// LocalAuthConfig represents local authentication provider configuration
type LocalAuthConfig struct {
	Enabled               bool          `json:"enabled"`
	PasswordHashing       string        `json:"password_hashing"` // bcrypt, argon2
	MinPasswordLength     int           `json:"min_password_length"`
	RequireUppercase      bool          `json:"require_uppercase"`
	RequireLowercase      bool          `json:"require_lowercase"`
	RequireNumbers        bool          `json:"require_numbers"`
	RequireSymbols        bool          `json:"require_symbols"`
	MaxLoginAttempts      int           `json:"max_login_attempts"`
	LockoutDuration       time.Duration `json:"lockout_duration"`
	PasswordResetEnabled  bool          `json:"password_reset_enabled"`
	EmailVerification     bool          `json:"email_verification"`
	AccountAutoActivation bool          `json:"account_auto_activation"`
}

// LocalUser represents a local user
type LocalUser struct {
	ID            string                 `json:"id"`
	Username      string                 `json:"username"`
	Email         string                 `json:"email"`
	PasswordHash  string                 `json:"password_hash"`
	DisplayName   string                 `json:"display_name"`
	Avatar        string                 `json:"avatar"`
	TenantID      string                 `json:"tenant_id"`
	IsActive      bool                   `json:"is_active"`
	IsVerified    bool                   `json:"is_verified"`
	Roles         []string               `json:"roles"`
	Attributes    map[string]interface{} `json:"attributes"`
	LastLogin     *time.Time             `json:"last_login"`
	LoginAttempts int                    `json:"login_attempts"`
	LockedUntil   *time.Time             `json:"locked_until"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// LoginAttempt represents a login attempt record
type LoginAttempt struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Success   bool      `json:"success"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

// PasswordReset represents a password reset request
type PasswordReset struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Token     string     `json:"token"`
	Email     string     `json:"email"`
	IPAddress string     `json:"ip_address"`
	UserAgent string     `json:"user_agent"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// NewLocalAuthProvider creates a new local authentication provider
func NewLocalAuthProvider(db *sql.DB, config *LocalAuthConfig) *LocalAuthProvider {
	if config == nil {
		config = &LocalAuthConfig{
			Enabled:               true,
			PasswordHashing:       "bcrypt",
			MinPasswordLength:     8,
			RequireUppercase:      true,
			RequireLowercase:      true,
			RequireNumbers:        true,
			RequireSymbols:        false,
			MaxLoginAttempts:      5,
			LockoutDuration:       15 * time.Minute,
			PasswordResetEnabled:  true,
			EmailVerification:     false,
			AccountAutoActivation: true,
		}
	}

	return &LocalAuthProvider{
		db:     db,
		config: config,
	}
}

// Name returns the provider name
func (lap *LocalAuthProvider) Name() string {
	return "local"
}

// Authenticate handles local username/password authentication
func (lap *LocalAuthProvider) Authenticate(ctx context.Context, req *authgateway.AuthRequest) (*authgateway.AuthResult, error) {
	// Extract credentials
	username, ok := req.Credentials["username"].(string)
	if !ok {
		return &authgateway.AuthResult{
			Success: false,
			Message: "Username is required",
		}, nil
	}

	password, ok := req.Credentials["password"].(string)
	if !ok {
		return &authgateway.AuthResult{
			Success: false,
			Message: "Password is required",
		}, nil
	}

	// Get user by username
	user, err := lap.getUserByUsername(ctx, username, req.TenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			lap.logLoginAttempt(ctx, "", username, req.IPAddress, req.UserAgent, false, "User not found")
			return &authgateway.AuthResult{
				Success: false,
				Message: "Invalid username or password",
			}, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if account is locked
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		lap.logLoginAttempt(ctx, user.ID, username, req.IPAddress, req.UserAgent, false, "Account locked")
		return &authgateway.AuthResult{
			Success: false,
			Message: "Account is temporarily locked due to too many failed login attempts",
		}, nil
	}

	// Check if account is active
	if !user.IsActive {
		lap.logLoginAttempt(ctx, user.ID, username, req.IPAddress, req.UserAgent, false, "Account inactive")
		return &authgateway.AuthResult{
			Success: false,
			Message: "Account is not active",
		}, nil
	}

	// Check if email verification is required
	if lap.config.EmailVerification && !user.IsVerified {
		lap.logLoginAttempt(ctx, user.ID, username, req.IPAddress, req.UserAgent, false, "Email not verified")
		return &authgateway.AuthResult{
			Success: false,
			Message: "Email verification required",
		}, nil
	}

	// Verify password
	if !lap.verifyPassword(password, user.PasswordHash) {
		// Increment login attempts
		if err := lap.incrementLoginAttempts(ctx, user.ID); err != nil {
			return nil, fmt.Errorf("failed to increment login attempts: %w", err)
		}

		lap.logLoginAttempt(ctx, user.ID, username, req.IPAddress, req.UserAgent, false, "Invalid password")
		return &authgateway.AuthResult{
			Success: false,
			Message: "Invalid username or password",
		}, nil
	}

	// Reset login attempts on successful login
	if err := lap.resetLoginAttempts(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("failed to reset login attempts: %w", err)
	}

	// Update last login
	if err := lap.updateLastLogin(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("failed to update last login: %w", err)
	}

	// Log successful login
	lap.logLoginAttempt(ctx, user.ID, username, req.IPAddress, req.UserAgent, true, "Successful login")

	// Build user info
	userInfo := &authgateway.UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Avatar:      user.Avatar,
		TenantID:    user.TenantID,
		Roles:       user.Roles,
		Attributes:  user.Attributes,
		IsActive:    user.IsActive,
		LastLogin:   &time.Time{},
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}

	return &authgateway.AuthResult{
		Success:  true,
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		TenantID: user.TenantID,
		Roles:    user.Roles,
		Message:  "Authentication successful",
		Metadata: map[string]interface{}{
			"provider":   "local",
			"user_info":  userInfo,
			"last_login": time.Now(),
		},
	}, nil
}

// GetUserInfo retrieves user information for a token
func (lap *LocalAuthProvider) GetUserInfo(ctx context.Context, token string) (*authgateway.UserInfo, error) {
	// Extract user ID from token (this would depend on token format)
	// For now, assume token contains user ID directly
	userID := token // This is a simplification

	user, err := lap.getUserByID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &authgateway.UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Avatar:      user.Avatar,
		TenantID:    user.TenantID,
		Roles:       user.Roles,
		Attributes:  user.Attributes,
		IsActive:    user.IsActive,
		LastLogin:   user.LastLogin,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}, nil
}

// RefreshToken refreshes an access token
func (lap *LocalAuthProvider) RefreshToken(ctx context.Context, refreshToken string) (*authgateway.TokenResult, error) {
	// Local auth doesn't typically support token refresh through the provider
	// This is usually handled at the session management level
	return nil, fmt.Errorf("token refresh not supported by local provider")
}

// ValidateConfig validates the provider configuration
func (lap *LocalAuthProvider) ValidateConfig() error {
	if lap.config == nil {
		return fmt.Errorf("local auth config is required")
	}

	if !lap.config.Enabled {
		return nil // Provider is disabled
	}

	if lap.config.MinPasswordLength < 6 {
		return fmt.Errorf("minimum password length must be at least 6")
	}

	if lap.config.MaxLoginAttempts < 1 {
		return fmt.Errorf("max login attempts must be at least 1")
	}

	if lap.config.LockoutDuration < time.Minute {
		return fmt.Errorf("lockout duration must be at least 1 minute")
	}

	return nil
}

// RegisterUser registers a new local user
func (lap *LocalAuthProvider) RegisterUser(ctx context.Context, req *authgateway.UserRegistration) (*authgateway.UserInfo, error) {
	// Validate input
	if err := lap.validateRegistration(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if user already exists
	exists, err := lap.userExists(ctx, req.Username, req.Email, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("user with this username or email already exists")
	}

	// Hash password
	passwordHash, err := lap.hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &LocalUser{
		ID:           lap.generateUserID(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		DisplayName:  req.DisplayName,
		TenantID:     req.TenantID,
		IsActive:     lap.config.AccountAutoActivation,
		IsVerified:   !lap.config.EmailVerification,
		Roles:        req.Roles,
		Attributes:   req.Attributes,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Save user to database
	if err := lap.saveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	// TODO: Send verification email if required
	if lap.config.EmailVerification && !lap.config.AccountAutoActivation {
		// TODO: Implement email verification
	}

	userInfo := &authgateway.UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		TenantID:    user.TenantID,
		Roles:       user.Roles,
		Attributes:  user.Attributes,
		IsActive:    user.IsActive,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}

	return userInfo, nil
}

// ChangePassword changes user password
func (lap *LocalAuthProvider) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	// Get user
	user, err := lap.getUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Verify old password
	if !lap.verifyPassword(oldPassword, user.PasswordHash) {
		return fmt.Errorf("old password is incorrect")
	}

	// Validate new password
	if err := lap.validatePassword(newPassword); err != nil {
		return fmt.Errorf("new password validation failed: %w", err)
	}

	// Hash new password
	newPasswordHash, err := lap.hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password
	if err := lap.updateUserPassword(ctx, userID, newPasswordHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// InitiatePasswordReset initiates password reset process
func (lap *LocalAuthProvider) InitiatePasswordReset(ctx context.Context, email, tenantID, ipAddress, userAgent string) (string, error) {
	if !lap.config.PasswordResetEnabled {
		return "", fmt.Errorf("password reset is disabled")
	}

	// Get user by email
	user, err := lap.getUserByEmail(ctx, email, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Don't reveal if email exists for security
			return "", nil
		}
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	// Generate reset token
	token := lap.generateResetToken()

	// Create password reset record
	passwordReset := &PasswordReset{
		ID:        lap.generatePasswordResetID(),
		UserID:    user.ID,
		Token:     token,
		Email:     email,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hours expiry
		CreatedAt: time.Now(),
	}

	// Save password reset
	if err := lap.savePasswordReset(ctx, passwordReset); err != nil {
		return "", fmt.Errorf("failed to save password reset: %w", err)
	}

	// TODO: Send reset email

	return token, nil
}

// ResetPassword resets password using reset token
func (lap *LocalAuthProvider) ResetPassword(ctx context.Context, token, newPassword string) error {
	// Get password reset record
	passwordReset, err := lap.getPasswordReset(ctx, token)
	if err != nil {
		return fmt.Errorf("invalid or expired reset token")
	}

	// Check if token is used or expired
	if passwordReset.UsedAt != nil {
		return fmt.Errorf("reset token has already been used")
	}
	if time.Now().After(passwordReset.ExpiresAt) {
		return fmt.Errorf("reset token has expired")
	}

	// Validate new password
	if err := lap.validatePassword(newPassword); err != nil {
		return fmt.Errorf("password validation failed: %w", err)
	}

	// Hash new password
	passwordHash, err := lap.hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update user password
	if err := lap.updateUserPassword(ctx, passwordReset.UserID, passwordHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Mark reset token as used
	now := time.Now()
	if err := lap.markPasswordResetUsed(ctx, passwordReset.ID, &now); err != nil {
		return fmt.Errorf("failed to mark reset token as used: %w", err)
	}

	return nil
}

// VerifyEmail verifies user email using verification token
func (lap *LocalAuthProvider) VerifyEmail(ctx context.Context, token string) error {
	// TODO: Implement email verification
	return fmt.Errorf("email verification not implemented")
}

// Helper methods

func (lap *LocalAuthProvider) validateRegistration(req *authgateway.UserRegistration) error {
	if req.Username == "" {
		return fmt.Errorf("username is required")
	}
	if len(req.Username) < 3 {
		return fmt.Errorf("username must be at least 3 characters")
	}
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}
	if req.Password == "" {
		return fmt.Errorf("password is required")
	}

	return lap.validatePassword(req.Password)
}

func (lap *LocalAuthProvider) validatePassword(password string) error {
	if len(password) < lap.config.MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", lap.config.MinPasswordLength)
	}

	if lap.config.RequireUppercase {
		hasUpper := false
		for _, r := range password {
			if r >= 'A' && r <= 'Z' {
				hasUpper = true
				break
			}
		}
		if !hasUpper {
			return fmt.Errorf("password must contain at least one uppercase letter")
		}
	}

	if lap.config.RequireLowercase {
		hasLower := false
		for _, r := range password {
			if r >= 'a' && r <= 'z' {
				hasLower = true
				break
			}
		}
		if !hasLower {
			return fmt.Errorf("password must contain at least one lowercase letter")
		}
	}

	if lap.config.RequireNumbers {
		hasNumber := false
		for _, r := range password {
			if r >= '0' && r <= '9' {
				hasNumber = true
				break
			}
		}
		if !hasNumber {
			return fmt.Errorf("password must contain at least one number")
		}
	}

	if lap.config.RequireSymbols {
		hasSymbol := false
		symbols := "!@#$%^&*()_+-=[]{}|;:,.<>?"
		for _, r := range password {
			for _, s := range symbols {
				if r == s {
					hasSymbol = true
					break
				}
			}
			if hasSymbol {
				break
			}
		}
		if !hasSymbol {
			return fmt.Errorf("password must contain at least one symbol")
		}
	}

	return nil
}

func (lap *LocalAuthProvider) hashPassword(password string) (string, error) {
	switch lap.config.PasswordHashing {
	case "bcrypt":
		hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return "", err
		}
		return string(hashedBytes), nil
	case "argon2":
		// TODO: Implement Argon2
		return "", fmt.Errorf("Argon2 not implemented")
	default:
		return "", fmt.Errorf("unsupported password hashing algorithm: %s", lap.config.PasswordHashing)
	}
}

func (lap *LocalAuthProvider) verifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (lap *LocalAuthProvider) userExists(ctx context.Context, username, email, tenantID string) (bool, error) {
	query := `SELECT COUNT(*) FROM auth_users
		WHERE (username = ? OR email = ?)
		AND (tenant_id = ? OR ? = '')`
	var count int
	err := lap.db.QueryRowContext(ctx, query, username, email, tenantID, tenantID).Scan(&count)
	return count > 0, err
}

func (lap *LocalAuthProvider) generateUserID() string {
	return fmt.Sprintf("local_user_%d", time.Now().UnixNano())
}

func (lap *LocalAuthProvider) generatePasswordResetID() string {
	return fmt.Sprintf("reset_%d", time.Now().UnixNano())
}

func (lap *LocalAuthProvider) generateResetToken() string {
	return fmt.Sprintf("reset_token_%d", time.Now().UnixNano())
}

// Database operations (these would be implemented based on your schema)
func (lap *LocalAuthProvider) getUserByUsername(ctx context.Context, username, tenantID string) (*LocalUser, error) {
	query := `SELECT id, username, email, password_hash, display_name, avatar,
		tenant_id, is_active, is_verified, roles, attributes, last_login,
		login_attempts, locked_until, created_at, updated_at
		FROM auth_users
		WHERE username = ? AND (tenant_id = ? OR ? = '')`

	var user LocalUser
	var rolesJSON, attributesJSON sql.NullString
	var lastLogin, lockedUntil sql.NullTime

	err := lap.db.QueryRowContext(ctx, query, username, tenantID, tenantID).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.DisplayName, &user.Avatar,
		&user.TenantID, &user.IsActive, &user.IsVerified, &rolesJSON, &attributesJSON, &lastLogin,
		&user.LoginAttempts, &lockedUntil, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if rolesJSON.Valid {
		json.Unmarshal([]byte(rolesJSON.String), &user.Roles)
	}
	if attributesJSON.Valid {
		json.Unmarshal([]byte(attributesJSON.String), &user.Attributes)
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}
	if lockedUntil.Valid {
		user.LockedUntil = &lockedUntil.Time
	}

	return &user, nil
}

func (lap *LocalAuthProvider) getUserByID(ctx context.Context, userID string) (*LocalUser, error) {
	// Similar implementation to getUserByUsername but using ID
	return nil, fmt.Errorf("not implemented")
}

func (lap *LocalAuthProvider) getUserByEmail(ctx context.Context, email, tenantID string) (*LocalUser, error) {
	// Similar implementation to getUserByUsername but using email
	return nil, fmt.Errorf("not implemented")
}

func (lap *LocalAuthProvider) saveUser(ctx context.Context, user *LocalUser) error {
	query := `INSERT INTO auth_users
		(id, username, email, password_hash, display_name, avatar, tenant_id,
		is_active, is_verified, roles, attributes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	rolesJSON, _ := json.Marshal(user.Roles)
	attributesJSON, _ := json.Marshal(user.Attributes)

	_, err := lap.db.ExecContext(ctx, query,
		user.ID, user.Username, user.Email, user.PasswordHash, user.DisplayName, user.Avatar,
		user.TenantID, user.IsActive, user.IsVerified, string(rolesJSON), string(attributesJSON),
		user.CreatedAt, user.UpdatedAt,
	)

	return err
}

func (lap *LocalAuthProvider) updateUserPassword(ctx context.Context, userID, passwordHash string) error {
	query := `UPDATE auth_users SET password_hash = ?, updated_at = ? WHERE id = ?`
	_, err := lap.db.ExecContext(ctx, query, passwordHash, time.Now(), userID)
	return err
}

func (lap *LocalAuthProvider) updateLastLogin(ctx context.Context, userID string) error {
	query := `UPDATE auth_users SET last_login = ?, updated_at = ? WHERE id = ?`
	_, err := lap.db.ExecContext(ctx, query, time.Now(), time.Now(), userID)
	return err
}

func (lap *LocalAuthProvider) incrementLoginAttempts(ctx context.Context, userID string) error {
	query := `UPDATE auth_users
		SET login_attempts = login_attempts + 1,
			locked_until = CASE
				WHEN login_attempts + 1 >= ? THEN ?
				ELSE locked_until
			END,
			updated_at = ?
		WHERE id = ?`

	_, err := lap.db.ExecContext(ctx, query,
		lap.config.MaxLoginAttempts,
		time.Now().Add(lap.config.LockoutDuration),
		time.Now(),
		userID,
	)
	return err
}

func (lap *LocalAuthProvider) resetLoginAttempts(ctx context.Context, userID string) error {
	query := `UPDATE auth_users SET login_attempts = 0, locked_until = NULL, updated_at = ? WHERE id = ?`
	_, err := lap.db.ExecContext(ctx, query, time.Now(), userID)
	return err
}

func (lap *LocalAuthProvider) logLoginAttempt(ctx context.Context, userID, username, ipAddress, userAgent string, success bool, reason string) {
	query := `INSERT INTO auth_login_attempts
		(id, user_id, username, ip_address, user_agent, success, reason, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := lap.db.ExecContext(ctx, query,
		lap.generateLoginAttemptID(),
		userID,
		username,
		ipAddress,
		userAgent,
		success,
		reason,
		time.Now(),
	)

	if err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log login attempt: %v\n", err)
	}
}

func (lap *LocalAuthProvider) generateLoginAttemptID() string {
	return fmt.Sprintf("login_attempt_%d", time.Now().UnixNano())
}

func (lap *LocalAuthProvider) savePasswordReset(ctx context.Context, passwordReset *PasswordReset) error {
	query := `INSERT INTO auth_password_resets
		(id, user_id, token, email, ip_address, user_agent, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := lap.db.ExecContext(ctx, query,
		passwordReset.ID,
		passwordReset.UserID,
		passwordReset.Token,
		passwordReset.Email,
		passwordReset.IPAddress,
		passwordReset.UserAgent,
		passwordReset.ExpiresAt,
		passwordReset.CreatedAt,
	)

	return err
}

func (lap *LocalAuthProvider) getPasswordReset(ctx context.Context, token string) (*PasswordReset, error) {
	query := `SELECT id, user_id, token, email, ip_address, user_agent, expires_at, used_at, created_at
		FROM auth_password_resets
		WHERE token = ?`

	var passwordReset PasswordReset
	var usedAt sql.NullTime

	err := lap.db.QueryRowContext(ctx, query, token).Scan(
		&passwordReset.ID,
		&passwordReset.UserID,
		&passwordReset.Token,
		&passwordReset.Email,
		&passwordReset.IPAddress,
		&passwordReset.UserAgent,
		&passwordReset.ExpiresAt,
		&usedAt,
		&passwordReset.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	if usedAt.Valid {
		passwordReset.UsedAt = &usedAt.Time
	}

	return &passwordReset, nil
}

func (lap *LocalAuthProvider) markPasswordResetUsed(ctx context.Context, resetID string, usedAt *time.Time) error {
	query := `UPDATE auth_password_resets SET used_at = ? WHERE id = ?`
	_, err := lap.db.ExecContext(ctx, query, usedAt, resetID)
	return err
}
