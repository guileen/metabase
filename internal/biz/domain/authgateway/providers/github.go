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

// GitHubOAuthProvider implements GitHub OAuth2 authentication
type GitHubOAuthProvider struct {
	*OAuth2Provider
}

// GitHubOAuthConfig represents GitHub OAuth2 specific configuration
type GitHubOAuthConfig struct {
	*OAuth2Config
}

// NewGitHubOAuthProvider creates a new GitHub OAuth2 provider
func NewGitHubOAuthProvider(db *sql.DB, config *GitHubOAuthConfig) *GitHubOAuthProvider {
	oauth2Config := &OAuth2Config{
		Name:         "github",
		DisplayName:  "GitHub",
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		AuthURL:      "https://github.com/login/oauth/authorize",
		TokenURL:     "https://github.com/login/oauth/access_token",
		UserInfoURL:  "https://api.github.com/user",
		Scopes:       []string{"user:email"},
		Enabled:      config.Enabled,
		Config:       config.Config,
	}

	return &GitHubOAuthProvider{
		OAuth2Provider: NewOAuth2Provider(db, oauth2Config),
	}
}

// parseUserInfo parses GitHub-specific user information
func (gop *GitHubOAuthProvider) parseUserInfo(rawData map[string]interface{}) *OAuth2UserInfo {
	userInfo := &OAuth2UserInfo{}

	// GitHub specific fields
	if id, ok := rawData["id"].(float64); ok {
		userInfo.ID = fmt.Sprintf("%.0f", id)
	}
	if login, ok := rawData["login"].(string); ok {
		userInfo.Username = login
	}
	if name, ok := rawData["name"].(string); ok {
		userInfo.Name = name
	}
	if email, ok := rawData["email"].(string); ok {
		userInfo.Email = email
	}
	if avatarURL, ok := rawData["avatar_url"].(string); ok {
		userInfo.Avatar = avatarURL
	}

	// GitHub doesn't provide email by default in the user endpoint
	// We need to make an additional call to get emails
	if userInfo.Email == "" {
		userInfo.Email = gop.getUserEmail(rawData)
	}

	return userInfo
}

// getUserEmail extracts email from GitHub user data or makes additional call
func (gop *GitHubOAuthProvider) getUserEmail(rawData map[string]interface{}) string {
	// Check if email is publicly available
	if email, ok := rawData["email"].(string); ok && email != "" {
		return email
	}

	// For private emails, we would need to call the GitHub emails API
	// This requires the 'user:email' scope
	// For now, return empty string and handle email collection separately
	return ""
}

// GetUserInfo retrieves user information with GitHub-specific handling
func (gop *GitHubOAuthProvider) GetUserInfo(ctx context.Context, token string) (*authgateway.UserInfo, error) {
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

// getUserEmailFromAPI makes an additional API call to get user's email addresses
func (gop *GitHubOAuthProvider) getUserEmailFromAPI(ctx context.Context, token string) (string, error) {
	// This would make a call to https://api.github.com/user/emails
	// For now, return empty string
	// In a real implementation, you would want to:
	// 1. Make HTTP request to GitHub emails API
	// 2. Parse the response
	// 3. Return the primary verified email
	return "", nil
}

// GitHub requires additional scope information
func (gop *GitHubOAuthProvider) GetAuthURL(state string) string {
	// GitHub supports additional parameters like login and scope
	baseURL := gop.oauth2.AuthCodeURL(state, oauth2.AccessTypeOnline)

	// Add custom parameters for GitHub
	if strings.Contains(baseURL, "?") {
		return baseURL + "&scope=user:email"
	}
	return baseURL + "?scope=user:email"
}

// Create user with GitHub-specific handling
func (gop *GitHubOAuthProvider) createUserFromOAuth(ctx context.Context, userInfo *OAuth2UserInfo) (*authgateway.UserInfo, error) {
	userID := gop.generateUserID()

	// For GitHub, use the login name as username
	username := userInfo.Username
	if username == "" {
		username = fmt.Sprintf("github_user_%s", userInfo.ID)
	}

	// Extract first and last name
	var firstName, lastName string
	if userInfo.Name != "" {
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

	// GitHub users with verified emails are considered email verified
	var emailVerified bool = false
	if userInfo.Email != "" {
		// In a real implementation, you'd check if the email is verified
		// For now, assume GitHub emails are verified
		emailVerified = true
	}

	_, err := gop.db.ExecContext(ctx, query,
		userID, username, userInfo.Email, firstName, lastName, userInfo.Avatar,
		gop.config.Name, userInfo.ID, emailVerified, string(rolesJSON), now, now,
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

// Helper functions to access OAuth2Provider private methods
func (gop *GitHubOAuthProvider) generateUserID() string {
	return fmt.Sprintf("github_user_%d", time.Now().UnixNano())
}
