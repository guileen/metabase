package providers

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/guileen/metabase/internal/biz/domain/authgateway"
)

// WeChatOAuthProvider implements WeChat OAuth2 authentication
type WeChatOAuthProvider struct {
	*OAuth2Provider
}

// WeChatOAuthConfig represents WeChat OAuth2 specific configuration
type WeChatOAuthConfig struct {
	*OAuth2Config
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
	Scope     string `json:"scope"` // snsapi_base, snsapi_userinfo
}

// NewWeChatOAuthProvider creates a new WeChat OAuth2 provider
func NewWeChatOAuthProvider(db *sql.DB, config *WeChatOAuthConfig) *WeChatOAuthProvider {
	oauth2Config := &OAuth2Config{
		Name:         "wechat",
		DisplayName:  "WeChat",
		ClientID:     config.AppID,
		ClientSecret: config.AppSecret,
		AuthURL:      "https://open.weixin.qq.com/connect/qrconnect",
		TokenURL:     "https://api.weixin.qq.com/sns/oauth2/access_token",
		UserInfoURL:  "https://api.weixin.qq.com/sns/userinfo",
		Scopes:       []string{config.Scope},
		Enabled:      config.Enabled,
		Config:       map[string]interface{}{"app_id": config.AppID, "scope": config.Scope},
	}

	// Override default URLs for WeChat
	if config.Scope == "snsapi_base" {
		// For snsapi_base, we don't get user info directly
		oauth2Config.UserInfoURL = ""
	}

	return &WeChatOAuthProvider{
		OAuth2Provider: NewOAuth2Provider(db, oauth2Config),
	}
}

// parseUserInfo parses WeChat-specific user information
func (wop *WeChatOAuthProvider) parseUserInfo(rawData map[string]interface{}) *OAuth2UserInfo {
	userInfo := &OAuth2UserInfo{}

	// WeChat specific fields
	if openid, ok := rawData["openid"].(string); ok {
		userInfo.ID = openid
	}
	if unionid, ok := rawData["unionid"].(string); ok {
		userInfo.ID = unionid // Prefer unionid if available
	}
	if nickname, ok := rawData["nickname"].(string); ok {
		userInfo.Name = nickname
		userInfo.Username = nickname
	}
	if headImgURL, ok := rawData["headimgurl"].(string); ok {
		userInfo.Avatar = headImgURL
	}

	// WeChat doesn't provide email by default for privacy reasons
	// Email would need to be requested separately or entered by user

	return userInfo
}

// GetUserInfo retrieves user information with WeChat-specific handling
func (wop *WeChatOAuthProvider) GetUserInfo(ctx context.Context, token string) (*authgateway.UserInfo, error) {
	// WeChat token includes openid information
	// First, we might need to refresh user info using the access token
	// For WeChat, user info is typically obtained during the OAuth flow

	// Call the parent implementation
	return wop.OAuth2Provider.GetUserInfo(ctx, token)
}

// WeChat requires special handling for the authorization URL
func (wop *WeChatOAuthProvider) GetAuthURL(state string) string {
	// WeChat uses a different parameter name for appid
	baseURL := "https://open.weixin.qq.com/connect/qrconnect"
	params := fmt.Sprintf("?appid=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s#wechat_redirect",
		wop.config.ClientID,
		wop.config.RedirectURL,
		wop.config.Scopes[0],
		state,
	)
	return baseURL + params
}

// ValidateConfig validates WeChat-specific configuration
func (wop *WeChatOAuthProvider) ValidateConfig() error {
	if err := wop.OAuth2Provider.ValidateConfig(); err != nil {
		return err
	}

	// WeChat specific validation
	if wechatConfig, ok := wop.config.Config["app_id"].(string); !ok || wechatConfig == "" {
		return fmt.Errorf("WeChat app_id is required")
	}

	if wechatConfig, ok := wop.config.Config["app_secret"].(string); !ok || wechatConfig == "" {
		return fmt.Errorf("WeChat app_secret is required")
	}

	return nil
}
