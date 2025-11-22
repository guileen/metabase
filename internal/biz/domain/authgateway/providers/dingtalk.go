package providers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/guileen/metabase/internal/biz/domain/authgateway"
)

// DingTalkOAuthProvider implements DingTalk OAuth2 authentication
type DingTalkOAuthProvider struct {
	*OAuth2Provider
}

// DingTalkOAuthConfig represents DingTalk OAuth2 specific configuration
type DingTalkOAuthConfig struct {
	*OAuth2Config
	AppKey    string `json:"app_key"`
	AppSecret string `json:"app_secret"`
}

// NewDingTalkOAuthProvider creates a new DingTalk OAuth2 provider
func NewDingTalkOAuthProvider(db *sql.DB, config *DingTalkOAuthConfig) *DingTalkOAuthProvider {
	oauth2Config := &OAuth2Config{
		Name:         "dingtalk",
		DisplayName:  "DingTalk",
		ClientID:     config.AppKey,
		ClientSecret: config.AppSecret,
		AuthURL:      "https://login.dingtalk.com/oauth2/auth",
		TokenURL:     "https://api.dingtalk.com/v1.0/oauth2/userAccessToken",
		UserInfoURL:  "https://api.dingtalk.com/v1.0/contact/users/me",
		Scopes:       []string{"openid", "contact:read"},
		Enabled:      config.Enabled,
		Config: map[string]interface{}{
			"app_key":    config.AppKey,
			"app_secret": config.AppSecret,
		},
	}

	return &DingTalkOAuthProvider{
		OAuth2Provider: NewOAuth2Provider(db, oauth2Config),
	}
}

// parseUserInfo parses DingTalk-specific user information
func (dop *DingTalkOAuthProvider) parseUserInfo(rawData map[string]interface{}) *OAuth2UserInfo {
	userInfo := &OAuth2UserInfo{}

	// DingTalk specific fields
	if unionId, ok := rawData["unionId"].(string); ok {
		userInfo.ID = unionId
	} else if userId, ok := rawData["userId"].(string); ok {
		userInfo.ID = userId
	}
	if nick, ok := rawData["nick"].(string); ok {
		userInfo.Name = nick
		userInfo.Username = nick
	}
	if mobile, ok := rawData["mobile"].(string); ok {
		userInfo.Phone = mobile
	}
	if avatarUrl, ok := rawData["avatarUrl"].(string); ok {
		userInfo.Avatar = avatarUrl
	}
	if email, ok := rawData["email"].(string); ok {
		userInfo.Email = email
	}

	// DingTalk typically doesn't provide email by default for privacy
	// Email would need to be entered by user or obtained through other means

	return userInfo
}

// GetAuthURL returns the DingTalk OAuth2 authorization URL
func (dop *DingTalkOAuthProvider) GetAuthURL(state string) string {
	// DingTalk uses a different parameter structure
	params := map[string]string{
		"response_type": "code",
		"client_id":     dop.config.ClientID,
		"redirect_uri":  dop.config.RedirectURL,
		"scope":         strings.Join(dop.config.Scopes, " "),
		"state":         state,
	}

	var paramStrs []string
	for key, value := range params {
		paramStrs = append(paramStrs, fmt.Sprintf("%s=%s", key, value))
	}

	return dop.config.AuthURL + "?" + strings.Join(paramStrs, "&")
}

// exchangeCodeForToken exchanges authorization code for access token (DingTalk specific)
func (dop *DingTalkOAuthProvider) exchangeCodeForToken(ctx context.Context, code string) (*OAuth2Token, error) {
	// DingTalk requires a different token exchange process
	// First get authorization code, then exchange for user access token
	// This is a simplified version - actual implementation would follow DingTalk's OAuth2 flow

	// For now, use the standard OAuth2 flow
	return dop.OAuth2Provider.exchangeCodeForToken(ctx, code)
}

// GetUserInfo retrieves user information with DingTalk-specific handling
func (dop *DingTalkOAuthProvider) GetUserInfo(ctx context.Context, token string) (*authgateway.UserInfo, error) {
	userInfo, err := dop.getUserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	user, err := dop.findUserByProviderID(ctx, userInfo.ID)
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

// createUserFromOAuth creates a new user from DingTalk OAuth2 information
func (dop *DingTalkOAuthProvider) createUserFromOAuth(ctx context.Context, userInfo *OAuth2UserInfo) (*authgateway.UserInfo, error) {
	userID := dop.generateUserID()

	// Generate username from nickname
	username := userInfo.Username
	if username == "" {
		if userInfo.Name != "" {
			username = strings.ToLower(strings.ReplaceAll(userInfo.Name, " ", "_"))
		} else {
			username = fmt.Sprintf("dingtalk_user_%s", userInfo.ID)
		}
	}

	// Default roles
	roles := []string{"user"}

	query := `INSERT INTO auth_users
		(id, username, email, phone, first_name, avatar, provider, provider_id,
		is_active, is_phone_verified, roles, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, TRUE, ?, ?, ?, ?)`

	rolesJSON, _ := json.Marshal(roles)
	now := time.Now()

	// DingTalk users typically have verified phone numbers
	var phoneVerified bool = userInfo.Phone != ""

	_, err := dop.db.ExecContext(ctx, query,
		userID, username, userInfo.Email, userInfo.Phone, userInfo.Name,
		userInfo.Avatar, dop.config.Name, userInfo.ID, phoneVerified,
		string(rolesJSON), now, now,
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

// ValidateConfig validates DingTalk-specific configuration
func (dop *DingTalkOAuthProvider) ValidateConfig() error {
	if err := dop.OAuth2Provider.ValidateConfig(); err != nil {
		return err
	}

	// DingTalk specific validation
	if appKey, ok := dop.config.Config["app_key"].(string); !ok || appKey == "" {
		return fmt.Errorf("DingTalk app_key is required")
	}

	if appSecret, ok := dop.config.Config["app_secret"].(string); !ok || appSecret == "" {
		return fmt.Errorf("DingTalk app_secret is required")
	}

	return nil
}

// Helper functions
func (dop *DingTalkOAuthProvider) generateUserID() string {
	return fmt.Sprintf("dingtalk_user_%d", time.Now().UnixNano())
}
