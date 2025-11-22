package providers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/guileen/metabase/internal/biz/domain/authgateway"
	"golang.org/x/oauth2"
)

// GoogleOAuthProvider implements Google OAuth2 authentication
type GoogleOAuthProvider struct {
	*OAuth2Provider
}

// GoogleOAuthConfig represents Google OAuth2 specific configuration
type GoogleOAuthConfig struct {
	*OAuth2Config
	HostedDomain string `json:"hosted_domain,omitempty"` // For Google Workspace accounts
}

// NewGoogleOAuthProvider creates a new Google OAuth2 provider
func NewGoogleOAuthProvider(db *sql.DB, config *GoogleOAuthConfig) *GoogleOAuthProvider {
	oauth2Config := &OAuth2Config{
		Name:         "google",
		DisplayName:  "Google",
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
		Scopes:       []string{"openid", "profile", "email"},
		Enabled:      config.Enabled,
		Config:       make(map[string]interface{}),
	}

	// Add Google-specific config
	if config.HostedDomain != "" {
		oauth2Config.Config["hosted_domain"] = config.HostedDomain
	}

	return &GoogleOAuthProvider{
		OAuth2Provider: NewOAuth2Provider(db, oauth2Config),
	}
}

// parseUserInfo parses Google-specific user information
func (gop *GoogleOAuthProvider) parseUserInfo(rawData map[string]interface{}) *OAuth2UserInfo {
	userInfo := &OAuth2UserInfo{}

	// Google specific fields
	if id, ok := rawData["id"].(string); ok {
		userInfo.ID = id
	}
	if email, ok := rawData["email"].(string); ok {
		userInfo.Email = email
	}
	if name, ok := rawData["name"].(string); ok {
		userInfo.Name = name
	}
	if givenName, ok := rawData["given_name"].(string); ok {
		// Store given name in metadata for potential use
		if userInfo.RawData == nil {
			userInfo.RawData = make(map[string]interface{})
		}
		userInfo.RawData["first_name"] = givenName
	}
	if familyName, ok := rawData["family_name"].(string); ok {
		// Store family name in metadata for potential use
		if userInfo.RawData == nil {
			userInfo.RawData = make(map[string]interface{})
		}
		userInfo.RawData["last_name"] = familyName
	}
	if picture, ok := rawData["picture"].(string); ok {
		userInfo.Avatar = picture
	}
	if verified, ok := rawData["verified_email"].(bool); ok {
		userInfo.Verified = verified
	}

	// Generate username from email if name is not available
	if userInfo.Name == "" && userInfo.Email != "" {
		parts := strings.Split(userInfo.Email, "@")
		if len(parts) > 0 {
			userInfo.Username = parts[0]
		}
	}

	return userInfo
}

// GetAuthURL returns the Google OAuth2 authorization URL with optional parameters
func (gop *GoogleOAuthProvider) GetAuthURL(state string) string {
	opts := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOffline, // Request refresh token
		oauth2.ApprovalForce,     // Force consent screen for better UX
	}

	// Add hosted domain if specified (for Google Workspace)
	if hostedDomain, ok := gop.config.Config["hosted_domain"].(string); ok && hostedDomain != "" {
		opts = append(opts, oauth2.SetAuthURLParam("hd", hostedDomain))
	}

	return gop.oauth2.AuthCodeURL(state, opts...)
}

// GetUserInfo retrieves user information with Google-specific handling
func (gop *GoogleOAuthProvider) GetUserInfo(ctx context.Context, token string) (*authgateway.UserInfo, error) {
	// Get primary user info
	userInfo, err := gop.getUserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Find user in database
	user, err := gop.findUserByProviderID(ctx, userInfo.ID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
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

// createUserFromOAuth creates a new user from Google OAuth2 information
func (gop *GoogleOAuthProvider) createUserFromOAuth(ctx context.Context, userInfo *OAuth2UserInfo) (*authgateway.UserInfo, error) {
	userID := gop.generateUserID()

	// Generate username from email or name
	username := userInfo.Username
	if username == "" {
		if userInfo.Email != "" {
			parts := strings.Split(userInfo.Email, "@")
			if len(parts) > 0 {
				username = parts[0]
			}
		}
		if username == "" {
			username = fmt.Sprintf("google_user_%s", userInfo.ID)
		}
	}

	// Extract first and last name from raw data if available
	var firstName, lastName string
	if userInfo.RawData != nil {
		if fn, ok := userInfo.RawData["first_name"].(string); ok {
			firstName = fn
		}
		if ln, ok := userInfo.RawData["last_name"].(string); ok {
			lastName = ln
		}
	}

	// Fallback to full name if first/last not available
	if firstName == "" && lastName == "" && userInfo.Name != "" {
		parts := strings.Split(userInfo.Name, " ")
		if len(parts) > 0 {
			firstName = parts[0]
		}
		if len(parts) > 1 {
			lastName = strings.Join(parts[1:], " ")
		}
	}

	// Default roles
	roles := []string{"user"}

	query := `INSERT INTO auth_users
		(id, username, email, first_name, last_name, avatar, provider, provider_id,
		is_active, is_email_verified, roles, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, TRUE, ?, ?, ?, ?)`

	rolesJSON, _ := json.Marshal(roles)
	now := time.Now()

	_, err := gop.db.ExecContext(ctx, query,
		userID, username, userInfo.Email, firstName, lastName, userInfo.Avatar,
		gop.config.Name, userInfo.ID, userInfo.Verified, string(rolesJSON), now, now,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	user := &authgateway.UserInfo{
		ID:          userID,
		Username:    username,
		Email:       userInfo.Email,
		DisplayName: userInfo.Name,
		Avatar:      userInfo.Avatar,
		Roles:       roles,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return user, nil
}

// validateHostedDomain checks if user's email domain matches the configured hosted domain
func (gop *GoogleOAuthProvider) validateHostedDomain(email string) error {
	hostedDomain, ok := gop.config.Config["hosted_domain"].(string)
	if !ok || hostedDomain == "" {
		return nil // No domain restriction
	}

	if email == "" {
		return fmt.Errorf("email is required for hosted domain validation")
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid email format for domain validation")
	}

	userDomain := parts[1]
	if userDomain != hostedDomain {
		return fmt.Errorf("email domain %s is not allowed. Only %s accounts are permitted", userDomain, hostedDomain)
	}

	return nil
}

// Authenticate with Google-specific validation
func (gop *GoogleOAuthProvider) Authenticate(ctx context.Context, req *authgateway.AuthRequest) (*authgateway.AuthResult, error) {
	// Call parent authentication
	result, err := gop.OAuth2Provider.Authenticate(ctx, req)
	if err != nil {
		return result, err
	}

	if result.Success && result.Metadata != nil {
		if userInfo, ok := result.Metadata["user_info"].(*OAuth2UserInfo); ok && userInfo.Email != "" {
			// Validate hosted domain if configured
			if err := gop.validateHostedDomain(userInfo.Email); err != nil {
				return &authgateway.AuthResult{
					Success: false,
					Message: err.Error(),
				}, nil
			}
		}
	}

	return result, nil
}

// Helper functions
func (gop *GoogleOAuthProvider) generateUserID() string {
	return fmt.Sprintf("google_user_%d", time.Now().UnixNano())
}
