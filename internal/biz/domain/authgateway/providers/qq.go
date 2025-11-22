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

// QQOAuthProvider implements QQ OAuth2 authentication
type QQOAuthProvider struct {
	*OAuth2Provider
}

// QQOAuthConfig represents QQ OAuth2 specific configuration
type QQOAuthConfig struct {
	*OAuth2Config
	AppID  string `json:"app_id"`
	AppKey string `json:"app_key"`
}

// NewQQOAuthProvider creates a new QQ OAuth2 provider
func NewQQOAuthProvider(db *sql.DB, config *QQOAuthConfig) *QQOAuthProvider {
	oauth2Config := &OAuth2Config{
		Name:         "qq",
		DisplayName:  "QQ",
		ClientID:     config.AppID,
		ClientSecret: config.AppKey,
		AuthURL:      "https://graph.qq.com/oauth2.0/authorize",
		TokenURL:     "https://graph.qq.com/oauth2.0/token",
		UserInfoURL:  "https://graph.qq.com/user/get_user_info",
		Scopes:       []string{"get_user_info"},
		Enabled:      config.Enabled,
		Config: map[string]interface{}{
			"app_id":  config.AppID,
			"app_key": config.AppKey,
		},
	}

	return &QQOAuthProvider{
		OAuth2Provider: NewOAuth2Provider(db, oauth2Config),
	}
}

// parseUserInfo parses QQ-specific user information
func (qop *QQOAuthProvider) parseUserInfo(rawData map[string]interface{}) *OAuth2UserInfo {
	userInfo := &OAuth2UserInfo{}

	// QQ specific fields
	if nickname, ok := rawData["nickname"].(string); ok {
		userInfo.Name = nickname
		userInfo.Username = strings.ToLower(strings.ReplaceAll(nickname, " ", "_"))
	}
	if gender, ok := rawData["gender"].(string); ok {
		// Store gender in metadata
		if userInfo.RawData == nil {
			userInfo.RawData = make(map[string]interface{})
		}
		userInfo.RawData["gender"] = gender
	}
	if figureurl, ok := rawData["figureurl_qq_2"].(string); ok {
		// Use QQ avatar (figureurl_qq_2 is 100x100)
		userInfo.Avatar = figureurl
	} else if figureurl, ok := rawData["figureurl_qq_1"].(string); ok {
		// Fallback to 40x40 avatar
		userInfo.Avatar = figureurl
	}

	// QQ doesn't provide email by default for privacy
	// We would need to get OpenID first, then user info

	return userInfo
}

// GetAuthURL returns the QQ OAuth2 authorization URL
func (qop *QQOAuthProvider) GetAuthURL(state string) string {
	// QQ uses a different parameter structure
	params := map[string]string{
		"response_type": "code",
		"client_id":     qop.config.ClientID,
		"redirect_uri":  qop.config.RedirectURL,
		"scope":         strings.Join(qop.config.Scopes, " "),
		"state":         state,
	}

	var paramStrs []string
	for key, value := range params {
		paramStrs = append(paramStrs, fmt.Sprintf("%s=%s", key, value))
	}

	return qop.config.AuthURL + "?" + strings.Join(paramStrs, "&")
}

// exchangeCodeForToken exchanges authorization code for access token (QQ specific)
func (qop *QQOAuthProvider) exchangeCodeForToken(ctx context.Context, code string) (*OAuth2Token, error) {
	// QQ requires a different token exchange process
	// Need to get access token, then get OpenID, then get user info

	// For now, use the standard OAuth2 flow
	// In a real implementation, you would:
	// 1. Exchange code for access token
	// 2. Get OpenID using the access token
	// 3. Get user info using OpenID and access token

	return qop.OAuth2Provider.exchangeCodeForToken(ctx, code)
}

// getUserInfo gets user information from QQ provider (overridden for QQ-specific flow)
func (qop *QQOAuthProvider) getUserInfo(ctx context.Context, accessToken string) (*OAuth2UserInfo, error) {
	// QQ requires a specific flow:
	// 1. Get OpenID first using access token
	// 2. Then get user info using OpenID and access token

	// For now, use the parent implementation
	// In a real implementation, you would:
	// 1. Make request to https://graph.qq.com/oauth2.0/me?access_token=xxx to get OpenID
	// 2. Parse the callback( {"client_id":"...","openid":"..."} ) response
	// 3. Make request to user info endpoint with OpenID and access token

	userInfo, err := qop.OAuth2Provider.getUserInfo(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	// Set provider
	userInfo.Provider = "qq"
	return userInfo, nil
}

// GetUserInfo retrieves user information with QQ-specific handling
func (qop *QQOAuthProvider) GetUserInfo(ctx context.Context, token string) (*authgateway.UserInfo, error) {
	userInfo, err := qop.getUserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	user, err := qop.findUserByProviderID(ctx, userInfo.ID)
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

// createUserFromOAuth creates a new user from QQ OAuth2 information
func (qop *QQOAuthProvider) createUserFromOAuth(ctx context.Context, userInfo *OAuth2UserInfo) (*authgateway.UserInfo, error) {
	userID := qop.generateUserID()

	// Generate username from nickname
	username := userInfo.Username
	if username == "" {
		if userInfo.Name != "" {
			username = strings.ToLower(strings.ReplaceAll(userInfo.Name, " ", "_"))
		} else {
			username = fmt.Sprintf("qq_user_%s", userInfo.ID)
		}
	}

	// Default roles
	roles := []string{"user"}

	query := `INSERT INTO auth_users
		(id, username, email, first_name, avatar, provider, provider_id,
		is_active, roles, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, TRUE, ?, ?, ?, ?)`

	rolesJSON, _ := json.Marshal(roles)
	metadataJSON, _ := json.Marshal(userInfo.RawData)
	now := time.Now()

	_, err := qop.db.ExecContext(ctx, query,
		userID, username, userInfo.Email, userInfo.Name,
		userInfo.Avatar, qop.config.Name, userInfo.ID,
		string(rolesJSON), string(metadataJSON), now, now,
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
		Attributes:  userInfo.RawData,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return user, nil
}

// ValidateConfig validates QQ-specific configuration
func (qop *QQOAuthProvider) ValidateConfig() error {
	if err := qop.OAuth2Provider.ValidateConfig(); err != nil {
		return err
	}

	// QQ specific validation
	if appID, ok := qop.config.Config["app_id"].(string); !ok || appID == "" {
		return fmt.Errorf("QQ app_id is required")
	}

	if appKey, ok := qop.config.Config["app_key"].(string); !ok || appKey == "" {
		return fmt.Errorf("QQ app_key is required")
	}

	return nil
}

// Helper functions
func (qop *QQOAuthProvider) generateUserID() string {
	return fmt.Sprintf("qq_user_%d", time.Now().UnixNano())
}
