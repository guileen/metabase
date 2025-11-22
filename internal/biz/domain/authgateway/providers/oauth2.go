package providers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/guileen/metabase/internal/biz/domain/authgateway"
	"golang.org/x/oauth2"
)

// OAuth2Provider implements generic OAuth2 authentication
type OAuth2Provider struct {
	db     *sql.DB
	config *OAuth2Config
	oauth2 *oauth2.Config
}

// OAuth2Config represents OAuth2 provider configuration
type OAuth2Config struct {
	Name         string                 `json:"name"`
	DisplayName  string                 `json:"display_name"`
	ClientID     string                 `json:"client_id"`
	ClientSecret string                 `json:"client_secret"`
	RedirectURL  string                 `json:"redirect_url"`
	AuthURL      string                 `json:"auth_url"`
	TokenURL     string                 `json:"token_url"`
	UserInfoURL  string                 `json:"user_info_url"`
	Scopes       []string               `json:"scopes"`
	Enabled      bool                   `json:"enabled"`
	Config       map[string]interface{} `json:"config,omitempty"`
}

// OAuth2UserInfo represents user information from OAuth2 provider
type OAuth2UserInfo struct {
	ID       string                 `json:"id"`
	Username string                 `json:"username,omitempty"`
	Email    string                 `json:"email,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Avatar   string                 `json:"avatar,omitempty"`
	Phone    string                 `json:"phone,omitempty"`
	Verified bool                   `json:"verified,omitempty"`
	Provider string                 `json:"provider"`
	RawData  map[string]interface{} `json:"raw_data,omitempty"`
}

// OAuth2Token represents OAuth2 token response
type OAuth2Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// NewOAuth2Provider creates a new OAuth2 provider
func NewOAuth2Provider(db *sql.DB, config *OAuth2Config) *OAuth2Provider {
	if config == nil {
		return nil
	}

	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       config.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  config.AuthURL,
			TokenURL: config.TokenURL,
		},
	}

	return &OAuth2Provider{
		db:     db,
		config: config,
		oauth2: oauth2Config,
	}
}

// Name returns the provider name
func (op *OAuth2Provider) Name() string {
	return op.config.Name
}

// Authenticate handles OAuth2 authentication
func (op *OAuth2Provider) Authenticate(ctx context.Context, req *authgateway.AuthRequest) (*authgateway.AuthResult, error) {
	// OAuth2 authentication requires authorization code or access token
	code, ok := req.Credentials["code"].(string)
	if !ok {
		return &authgateway.AuthResult{
			Success: false,
			Message: "Authorization code is required",
		}, nil
	}

	// Exchange authorization code for access token
	token, err := op.exchangeCodeForToken(ctx, code)
	if err != nil {
		return &authgateway.AuthResult{
			Success: false,
			Message: fmt.Sprintf("Failed to exchange code for token: %v", err),
		}, nil
	}

	// Get user information
	userInfo, err := op.getUserInfo(ctx, token.AccessToken)
	if err != nil {
		return &authgateway.AuthResult{
			Success: false,
			Message: fmt.Sprintf("Failed to get user info: %v", err),
		}, nil
	}

	// Find or create user
	user, isNewUser, err := op.findOrCreateUser(ctx, userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create user: %w", err)
	}

	// Store connected account
	if err := op.storeConnectedAccount(ctx, user.ID, userInfo, token); err != nil {
		// Log error but don't fail authentication
		fmt.Printf("Failed to store connected account: %v\n", err)
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
			"provider":    op.config.Name,
			"user_info":   userInfo,
			"is_new_user": isNewUser,
		},
	}, nil
}

// GetUserInfo retrieves user information for an access token
func (op *OAuth2Provider) GetUserInfo(ctx context.Context, token string) (*authgateway.UserInfo, error) {
	// Get user information from OAuth2 provider
	userInfo, err := op.getUserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Find user in database
	user, err := op.findUserByProviderID(ctx, userInfo.ID)
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

// RefreshToken refreshes an access token
func (op *OAuth2Provider) RefreshToken(ctx context.Context, refreshToken string) (*authgateway.TokenResult, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token is required")
	}

	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	// Use OAuth2 library to refresh token
	newToken, err := op.oauth2.TokenSource(ctx, token).Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	var expiresIn int64 = 3600 // Default 1 hour
	if newToken.Expiry.Unix() > 0 {
		expiresIn = int64(newToken.Expiry.Sub(time.Now()).Seconds())
	}

	return &authgateway.TokenResult{
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken,
		ExpiresIn:    int(expiresIn),
		TokenType:    newToken.TokenType,
	}, nil
}

// ValidateConfig validates the provider configuration
func (op *OAuth2Provider) ValidateConfig() error {
	if op.config == nil {
		return fmt.Errorf("oauth2 config is required")
	}

	if !op.config.Enabled {
		return nil // Provider is disabled
	}

	if op.config.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}

	if op.config.ClientSecret == "" {
		return fmt.Errorf("client_secret is required")
	}

	if op.config.AuthURL == "" {
		return fmt.Errorf("auth_url is required")
	}

	if op.config.TokenURL == "" {
		return fmt.Errorf("token_url is required")
	}

	if op.config.UserInfoURL == "" {
		return fmt.Errorf("user_info_url is required")
	}

	if len(op.config.Scopes) == 0 {
		return fmt.Errorf("at least one scope is required")
	}

	return nil
}

// GetAuthURL returns the OAuth2 authorization URL
func (op *OAuth2Provider) GetAuthURL(state string) string {
	return op.oauth2.AuthCodeURL(state)
}

// exchangeCodeForToken exchanges authorization code for access token
func (op *OAuth2Provider) exchangeCodeForToken(ctx context.Context, code string) (*OAuth2Token, error) {
	token, err := op.oauth2.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	var expiresIn int64 = 3600 // Default 1 hour
	if token.Expiry.Unix() > 0 {
		expiresIn = int64(token.Expiry.Sub(time.Now()).Seconds())
	}

	return &OAuth2Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

// getUserInfo gets user information from OAuth2 provider
func (op *OAuth2Provider) getUserInfo(ctx context.Context, accessToken string) (*OAuth2UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", op.config.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var rawData map[string]interface{}
	if err := json.Unmarshal(body, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse user info response: %w", err)
	}

	// Parse user info based on provider
	userInfo := op.parseUserInfo(rawData)
	userInfo.Provider = op.config.Name
	userInfo.RawData = rawData

	return userInfo, nil
}

// parseUserInfo parses user information from provider-specific response
func (op *OAuth2Provider) parseUserInfo(rawData map[string]interface{}) *OAuth2UserInfo {
	userInfo := &OAuth2UserInfo{}

	// Default parsing - can be overridden by specific providers
	if id, ok := rawData["id"].(string); ok {
		userInfo.ID = id
	}
	if username, ok := rawData["login"].(string); ok {
		userInfo.Username = username
	} else if username, ok := rawData["username"].(string); ok {
		userInfo.Username = username
	}
	if email, ok := rawData["email"].(string); ok {
		userInfo.Email = email
	}
	if name, ok := rawData["name"].(string); ok {
		userInfo.Name = name
	}
	if avatar, ok := rawData["avatar_url"].(string); ok {
		userInfo.Avatar = avatar
	} else if avatar, ok := rawData["picture"].(string); ok {
		userInfo.Avatar = avatar
	}

	return userInfo
}

// findOrCreateUser finds existing user or creates a new one
func (op *OAuth2Provider) findOrCreateUser(ctx context.Context, userInfo *OAuth2UserInfo) (*authgateway.UserInfo, bool, error) {
	// First try to find user by provider ID
	user, err := op.findUserByProviderID(ctx, userInfo.ID)
	if err == nil {
		// User exists, update information
		op.updateUserInfo(ctx, user, userInfo)
		return user, false, nil
	}

	// If email exists, link this provider to existing user
	if userInfo.Email != "" {
		user, err := op.findUserByEmail(ctx, userInfo.Email)
		if err == nil {
			// Link provider to existing user
			op.linkProviderToUser(ctx, user.ID, userInfo)
			return user, false, nil
		}
	}

	// Create new user
	user, err = op.createUserFromOAuth(ctx, userInfo)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create user: %w", err)
	}

	return user, true, nil
}

// findUserByProviderID finds user by OAuth2 provider ID
func (op *OAuth2Provider) findUserByProviderID(ctx context.Context, providerID string) (*authgateway.UserInfo, error) {
	query := `SELECT u.id, u.username, u.email, u.first_name, u.last_name, u.avatar,
		u.tenant_id, u.is_active, u.roles, u.attributes, u.last_sign_in_at, u.created_at, u.updated_at
		FROM auth_users u
		JOIN user_connected_accounts ca ON u.id = ca.user_id
		WHERE ca.provider = ? AND ca.provider_user_id = ? AND ca.is_active = TRUE`

	var user authgateway.UserInfo
	var rolesJSON, attributesJSON sql.NullString
	var lastSignIn, tenantID sql.NullString
	var firstName, lastName sql.NullString

	err := op.db.QueryRowContext(ctx, query, op.config.Name, providerID).Scan(
		&user.ID, &user.Username, &user.Email, &firstName, &lastName, &user.Avatar,
		&tenantID, &user.IsActive, &rolesJSON, &attributesJSON, &lastSignIn,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if firstName.Valid {
		user.DisplayName = firstName.String
		if lastName.Valid {
			user.DisplayName += " " + lastName.String
		}
	}
	if tenantID.Valid {
		user.TenantID = tenantID.String
	}
	if lastSignIn.Valid {
		if parsedTime, err := time.Parse(time.RFC3339, lastSignIn.String); err == nil {
			user.LastLogin = &parsedTime
		}
	}

	if rolesJSON.Valid {
		json.Unmarshal([]byte(rolesJSON.String), &user.Roles)
	}
	if attributesJSON.Valid {
		json.Unmarshal([]byte(attributesJSON.String), &user.Attributes)
	}

	return &user, nil
}

// findUserByEmail finds user by email address
func (op *OAuth2Provider) findUserByEmail(ctx context.Context, email string) (*authgateway.UserInfo, error) {
	query := `SELECT id, username, first_name, last_name, avatar, tenant_id, is_active,
		roles, attributes, last_sign_in_at, created_at, updated_at
		FROM auth_users WHERE email = ? AND is_active = TRUE`

	var user authgateway.UserInfo
	var rolesJSON, attributesJSON sql.NullString
	var lastSignIn, tenantID sql.NullString
	var firstName, lastName sql.NullString

	err := op.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Username, &firstName, &lastName, &user.Avatar,
		&tenantID, &user.IsActive, &rolesJSON, &attributesJSON, &lastSignIn,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if firstName.Valid {
		user.DisplayName = firstName.String
		if lastName.Valid {
			user.DisplayName += " " + lastName.String
		}
	}
	if tenantID.Valid {
		user.TenantID = tenantID.String
	}
	if lastSignIn.Valid {
		if parsedTime, err := time.Parse(time.RFC3339, lastSignIn.String); err == nil {
			user.LastLogin = &parsedTime
		}
	}

	user.Email = email

	if rolesJSON.Valid {
		json.Unmarshal([]byte(rolesJSON.String), &user.Roles)
	}
	if attributesJSON.Valid {
		json.Unmarshal([]byte(attributesJSON.String), &user.Attributes)
	}

	return &user, nil
}

// createUserFromOAuth creates a new user from OAuth2 information
func (op *OAuth2Provider) createUserFromOAuth(ctx context.Context, userInfo *OAuth2UserInfo) (*authgateway.UserInfo, error) {
	userID := op.generateUserID()

	// Generate username from provider info if not provided
	username := userInfo.Username
	if username == "" {
		if userInfo.Name != "" {
			username = strings.ToLower(strings.ReplaceAll(userInfo.Name, " ", "_"))
		} else {
			username = fmt.Sprintf("%s_user", op.config.Name)
		}
		username = fmt.Sprintf("%s_%d", username, time.Now().Unix())
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
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, TRUE, TRUE, ?, ?, ?)`

	rolesJSON, _ := json.Marshal(roles)
	now := time.Now()

	_, err := op.db.ExecContext(ctx, query,
		userID, username, userInfo.Email, firstName, lastName, userInfo.Avatar,
		op.config.Name, userInfo.ID, string(rolesJSON), now, now,
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

// updateUserInfo updates existing user information from OAuth2 provider
func (op *OAuth2Provider) updateUserInfo(ctx context.Context, user *authgateway.UserInfo, userInfo *OAuth2UserInfo) {
	query := `UPDATE auth_users SET
		email = COALESCE(?, email),
		first_name = COALESCE(?, first_name),
		last_name = COALESCE(?, last_name),
		avatar = COALESCE(?, avatar),
		last_sign_in_at = ?,
		updated_at = ?
		WHERE id = ?`

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

	now := time.Now()
	_, err := op.db.ExecContext(ctx, query,
		userInfo.Email, firstName, lastName, userInfo.Avatar, now, now, user.ID,
	)

	if err != nil {
		fmt.Printf("Failed to update user info: %v\n", err)
	}
}

// linkProviderToUser links OAuth2 provider to existing user
func (op *OAuth2Provider) linkProviderToUser(ctx context.Context, userID string, userInfo *OAuth2UserInfo) {
	query := `INSERT OR IGNORE INTO user_connected_accounts
		(id, user_id, provider, provider_user_id, provider_data, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	providerData, _ := json.Marshal(userInfo)
	now := time.Now()

	_, err := op.db.ExecContext(ctx, query,
		op.generateConnectedAccountID(), userID, op.config.Name, userInfo.ID,
		string(providerData), now, now,
	)

	if err != nil {
		fmt.Printf("Failed to link provider to user: %v\n", err)
	}
}

// storeConnectedAccount stores connected account information
func (op *OAuth2Provider) storeConnectedAccount(ctx context.Context, userID string, userInfo *OAuth2UserInfo, token *OAuth2Token) error {
	query := `INSERT OR REPLACE INTO user_connected_accounts
		(id, user_id, provider, provider_user_id, provider_data, access_token, refresh_token, token_expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	providerData, _ := json.Marshal(userInfo)
	now := time.Now()
	var tokenExpiresAt *time.Time
	if token.ExpiresIn > 0 {
		expiresAt := now.Add(time.Duration(token.ExpiresIn) * time.Second)
		tokenExpiresAt = &expiresAt
	}

	_, err := op.db.ExecContext(ctx, query,
		op.generateConnectedAccountID(), userID, op.config.Name, userInfo.ID,
		string(providerData), token.AccessToken, token.RefreshToken, tokenExpiresAt,
		now, now,
	)

	return err
}

// Helper functions
func (op *OAuth2Provider) generateUserID() string {
	return fmt.Sprintf("oauth2_user_%s_%d", op.config.Name, time.Now().UnixNano())
}

func (op *OAuth2Provider) generateConnectedAccountID() string {
	return fmt.Sprintf("connected_%s_%d", op.config.Name, time.Now().UnixNano())
}
