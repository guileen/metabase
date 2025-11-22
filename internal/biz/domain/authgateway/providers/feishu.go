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

// FeishuOAuthProvider implements Feishu (Lark) OAuth2 authentication
type FeishuOAuthProvider struct {
	*OAuth2Provider
}

// FeishuOAuthConfig represents Feishu OAuth2 specific configuration
type FeishuOAuthConfig struct {
	*OAuth2Config
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

// NewFeishuOAuthProvider creates a new Feishu OAuth2 provider
func NewFeishuOAuthProvider(db *sql.DB, config *FeishuOAuthConfig) *FeishuOAuthProvider {
	oauth2Config := &OAuth2Config{
		Name:         "feishu",
		DisplayName:  "Feishu",
		ClientID:     config.AppID,
		ClientSecret: config.AppSecret,
		AuthURL:      "https://open.feishu.cn/open-apis/authen/v1/authorize",
		TokenURL:     "https://open.feishu.cn/open-apis/authen/v1/access_token",
		UserInfoURL:  "https://open.feishu.cn/open-apis/authen/v1/user_info",
		Scopes:       []string{"contact:user.base:readonly", "contact:user.email:readonly"},
		Enabled:      config.Enabled,
		Config: map[string]interface{}{
			"app_id":     config.AppID,
			"app_secret": config.AppSecret,
		},
	}

	return &FeishuOAuthProvider{
		OAuth2Provider: NewOAuth2Provider(db, oauth2Config),
	}
}

// parseUserInfo parses Feishu-specific user information
func (fop *FeishuOAuthProvider) parseUserInfo(rawData map[string]interface{}) *OAuth2UserInfo {
	userInfo := &OAuth2UserInfo{}

	// Feishu specific fields
	if userId, ok := rawData["user_id"].(string); ok {
		userInfo.ID = userId
	}
	if unionId, ok := rawData["union_id"].(string); ok {
		// Store union_id in metadata as it's more stable across apps
		if userInfo.RawData == nil {
			userInfo.RawData = make(map[string]interface{})
		}
		userInfo.RawData["union_id"] = unionId
	}
	if name, ok := rawData["name"].(string); ok {
		userInfo.Name = name
		userInfo.Username = strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	}
	if enName, ok := rawData["en_name"].(string); ok {
		// Store English name in metadata
		if userInfo.RawData == nil {
			userInfo.RawData = make(map[string]interface{})
		}
		userInfo.RawData["en_name"] = enName
	}
	if email, ok := rawData["email"].(string); ok {
		userInfo.Email = email
	}
	if mobile, ok := rawData["mobile"].(string); ok {
		userInfo.Phone = mobile
	}
	if avatarBig, ok := rawData["avatar_big"].(string); ok {
		userInfo.Avatar = avatarBig
	} else if avatarMiddle, ok := rawData["avatar_middle"].(string); ok {
		userInfo.Avatar = avatarMiddle
	} else if avatarThumb, ok := rawData["avatar_thumb"].(string); ok {
		userInfo.Avatar = avatarThumb
	}

	// Feishu typically provides verified emails and phone numbers
	userInfo.Verified = true

	return userInfo
}

// GetAuthURL returns the Feishu OAuth2 authorization URL
func (fop *FeishuOAuthProvider) GetAuthURL(state string) string {
	// Feishu requires app_id parameter in addition to standard OAuth2 params
	params := map[string]string{
		"response_type": "code",
		"client_id":     fop.config.ClientID,
		"redirect_uri":  fop.config.RedirectURL,
		"scope":         strings.Join(fop.config.Scopes, " "),
		"state":         state,
	}

	var paramStrs []string
	for key, value := range params {
		paramStrs = append(paramStrs, fmt.Sprintf("%s=%s", key, value))
	}

	return fop.config.AuthURL + "?" + strings.Join(paramStrs, "&")
}

// GetUserInfo retrieves user information with Feishu-specific handling
func (fop *FeishuOAuthProvider) GetUserInfo(ctx context.Context, token string) (*authgateway.UserInfo, error) {
	userInfo, err := fop.getUserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	user, err := fop.findUserByProviderID(ctx, userInfo.ID)
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

// createUserFromOAuth creates a new user from Feishu OAuth2 information
func (fop *FeishuOAuthProvider) createUserFromOAuth(ctx context.Context, userInfo *OAuth2UserInfo) (*authgateway.UserInfo, error) {
	userID := fop.generateUserID()

	// Generate username from name
	username := userInfo.Username
	if username == "" {
		if userInfo.Name != "" {
			username = strings.ToLower(strings.ReplaceAll(userInfo.Name, " ", "_"))
		} else {
			username = fmt.Sprintf("feishu_user_%s", userInfo.ID)
		}
	}

	// Default roles
	roles := []string{"user"}

	query := `INSERT INTO auth_users
		(id, username, email, phone, first_name, avatar, provider, provider_id,
		is_active, is_email_verified, is_phone_verified, roles, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, TRUE, TRUE, ?, ?, ?, ?, ?)`

	rolesJSON, _ := json.Marshal(roles)
	metadataJSON, _ := json.Marshal(userInfo.RawData)
	now := time.Now()

	_, err := fop.db.ExecContext(ctx, query,
		userID, username, userInfo.Email, userInfo.Phone, userInfo.Name,
		userInfo.Avatar, fop.config.Name, userInfo.ID,
		userInfo.Email != "", // email verified if present
		userInfo.Phone != "", // phone verified if present
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

// ValidateConfig validates Feishu-specific configuration
func (fop *FeishuOAuthProvider) ValidateConfig() error {
	if err := fop.OAuth2Provider.ValidateConfig(); err != nil {
		return err
	}

	// Feishu specific validation
	if appID, ok := fop.config.Config["app_id"].(string); !ok || appID == "" {
		return fmt.Errorf("Feishu app_id is required")
	}

	if appSecret, ok := fop.config.Config["app_secret"].(string); !ok || appSecret == "" {
		return fmt.Errorf("Feishu app_secret is required")
	}

	return nil
}

// Helper functions
func (fop *FeishuOAuthProvider) generateUserID() string {
	return fmt.Sprintf("feishu_user_%d", time.Now().UnixNano())
}
