package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/guileen/metabase/internal/app/api/rest"
	"github.com/guileen/metabase/internal/biz/domain/authgateway"

	// "github.com/guileen/metabase/pkg/common/rest" // TODO: Fix this import
	"github.com/guileen/metabase/pkg/infra/auth"
)

// UserProfilesHandler handles user profile management
type UserProfilesHandler struct {
	db          *sql.DB
	authGateway *authgateway.AuthGatewayManager
	authMgr     *auth.Manager
}

// UserProfile represents user profile information
type UserProfile struct {
	ID            string                 `json:"id"`
	Username      string                 `json:"username"`
	Email         string                 `json:"email"`
	Phone         string                 `json:"phone,omitempty"`
	FirstName     string                 `json:"first_name,omitempty"`
	LastName      string                 `json:"last_name,omitempty"`
	Avatar        string                 `json:"avatar,omitempty"`
	DisplayName   string                 `json:"display_name,omitempty"`
	Provider      string                 `json:"provider"`
	ProviderID    string                 `json:"provider_id,omitempty"`
	Roles         []string               `json:"roles"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
	IsActive      bool                   `json:"is_active"`
	EmailVerified bool                   `json:"is_email_verified"`
	PhoneVerified bool                   `json:"is_phone_verified"`
	LastSignInAt  *time.Time             `json:"last_sign_in_at,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// UserProfileUpdateRequest represents user profile update request
type UserProfileUpdateRequest struct {
	Username    string                 `json:"username,omitempty"`
	Email       string                 `json:"email,omitempty"`
	Phone       string                 `json:"phone,omitempty"`
	FirstName   string                 `json:"first_name,omitempty"`
	LastName    string                 `json:"last_name,omitempty"`
	Avatar      string                 `json:"avatar,omitempty"`
	DisplayName string                 `json:"display_name,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
}

// PasswordChangeRequest represents password change request
type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

// ConnectedAccount represents a connected OAuth account
type ConnectedAccount struct {
	ID             string                 `json:"id"`
	Provider       string                 `json:"provider"`
	ProviderUserID string                 `json:"provider_user_id"`
	ProviderData   map[string]interface{} `json:"provider_data"`
	IsActive       bool                   `json:"is_active"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// NewUserProfilesHandler creates a new user profiles handler
func NewUserProfilesHandler(db *sql.DB, authGateway *authgateway.AuthGatewayManager, authMgr *auth.Manager) *UserProfilesHandler {
	return &UserProfilesHandler{
		db:          db,
		authGateway: authGateway,
		authMgr:     authMgr,
	}
}

// GetProfile returns the current user's profile
func (h *UserProfilesHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Get user from JWT token
	userID, err := h.getUserIDFromToken(r)
	if err != nil {
		rest.WriteError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	profile, err := h.getUserProfile(r.Context(), userID)
	if err != nil {
		if err == sql.ErrNoRows {
			rest.WriteError(w, http.StatusNotFound, "User profile not found", nil)
		} else {
			rest.WriteError(w, http.StatusInternalServerError, "Failed to get user profile", err)
		}
		return
	}

	rest.WriteJSON(w, http.StatusOK, profile)
}

// UpdateProfile updates the current user's profile
func (h *UserProfilesHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserIDFromToken(r)
	if err != nil {
		rest.WriteError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	var req UserProfileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if err := h.validateProfileUpdate(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Update profile
	if err := h.updateUserProfile(r.Context(), userID, &req); err != nil {
		rest.WriteError(w, http.StatusInternalServerError, "Failed to update profile", err)
		return
	}

	// Return updated profile
	h.GetProfile(w, r)
}

// ChangePassword changes the user's password
func (h *UserProfilesHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserIDFromToken(r)
	if err != nil {
		rest.WriteError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	var req PasswordChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if err := h.validatePasswordChange(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Get user to check provider
	user, err := h.getUser(r.Context(), userID)
	if err != nil {
		rest.WriteError(w, http.StatusInternalServerError, "Failed to get user", err)
		return
	}

	// Only allow password change for local auth users
	if user.Provider != "local" && user.Provider != "" {
		rest.WriteError(w, http.StatusBadRequest, "Password change is only available for local authentication users", nil)
		return
	}

	// Change password using auth gateway
	passwordChange := &authgateway.PasswordChange{
		UserID:      userID,
		OldPassword: req.CurrentPassword,
		NewPassword: req.NewPassword,
	}

	if err := h.authGateway.ChangePassword(r.Context(), passwordChange); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Failed to change password", err)
		return
	}

	rest.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Password changed successfully",
	})
}

// GetConnectedAccounts returns the user's connected OAuth accounts
func (h *UserProfilesHandler) GetConnectedAccounts(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserIDFromToken(r)
	if err != nil {
		rest.WriteError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	accounts, err := h.getConnectedAccounts(r.Context(), userID)
	if err != nil {
		rest.WriteError(w, http.StatusInternalServerError, "Failed to get connected accounts", err)
		return
	}

	rest.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"connected_accounts": accounts,
		"total":              len(accounts),
	})
}

// ConnectAccount links a new OAuth account to the user
func (h *UserProfilesHandler) ConnectAccount(w http.ResponseWriter, r *http.Request) {
	_, err := h.getUserIDFromToken(r)
	if err != nil {
		rest.WriteError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	var req struct {
		Provider string `json:"provider"`
		Code     string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// This would typically involve OAuth flow completion
	// For now, return success message
	rest.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "Account connection initiated",
		"provider": req.Provider,
	})
}

// DisconnectAccount removes a connected OAuth account
func (h *UserProfilesHandler) DisconnectAccount(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserIDFromToken(r)
	if err != nil {
		rest.WriteError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	accountID := r.PathValue("account_id")
	if accountID == "" {
		rest.WriteError(w, http.StatusBadRequest, "Account ID is required", nil)
		return
	}

	// Check if account belongs to user and get provider info
	var provider string
	err = h.db.QueryRow("SELECT provider FROM user_connected_accounts WHERE id = ? AND user_id = ?",
		accountID, userID).Scan(&provider)
	if err != nil {
		if err == sql.ErrNoRows {
			rest.WriteError(w, http.StatusNotFound, "Connected account not found", nil)
		} else {
			rest.WriteError(w, http.StatusInternalServerError, "Failed to check account", err)
		}
		return
	}

	// Don't allow disconnecting if it's the only authentication method
	var authMethods int
	h.db.QueryRow("SELECT COUNT(DISTINCT provider) FROM auth_users WHERE id = ?", userID).Scan(&authMethods)
	if authMethods <= 1 {
		rest.WriteError(w, http.StatusBadRequest, "Cannot disconnect the only authentication method", nil)
		return
	}

	// Disconnect account
	_, err = h.db.Exec("DELETE FROM user_connected_accounts WHERE id = ? AND user_id = ?",
		accountID, userID)
	if err != nil {
		rest.WriteError(w, http.StatusInternalServerError, "Failed to disconnect account", err)
		return
	}

	rest.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("Successfully disconnected %s account", provider),
	})
}

// SyncProfile synchronizes user profile from OAuth providers
func (h *UserProfilesHandler) SyncProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserIDFromToken(r)
	if err != nil {
		rest.WriteError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	// Get connected accounts
	accounts, err := h.getConnectedAccounts(r.Context(), userID)
	if err != nil {
		rest.WriteError(w, http.StatusInternalServerError, "Failed to get connected accounts", err)
		return
	}

	var syncedProviders []string
	var syncErrors []string

	// Sync from each connected account
	for _, account := range accounts {
		if account.IsActive {
			if err := h.syncFromProvider(r.Context(), userID, account.Provider); err != nil {
				syncErrors = append(syncErrors, fmt.Sprintf("%s: %v", account.Provider, err))
			} else {
				syncedProviders = append(syncedProviders, account.Provider)
			}
		}
	}

	rest.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message":          "Profile synchronization completed",
		"synced_providers": syncedProviders,
		"sync_errors":      syncErrors,
		"sync_count":       len(syncedProviders),
	})
}

// Helper methods

func (h *UserProfilesHandler) getUserIDFromToken(r *http.Request) (string, error) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is required")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("invalid authorization header format")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	// Validate session and get user ID
	session, err := h.authMgr.ValidateSession(r.Context(), token)
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	return session.UserID, nil
}

func (h *UserProfilesHandler) getUserProfile(ctx context.Context, userID string) (*UserProfile, error) {
	query := `SELECT id, username, email, phone, first_name, last_name, avatar, provider,
		provider_id, is_active, is_email_verified, is_phone_verified, last_sign_in_at,
		roles, metadata, created_at, updated_at
		FROM auth_users WHERE id = ?`

	var profile UserProfile
	var rolesJSON, metadataJSON sql.NullString
	var lastSignIn sql.NullTime

	err := h.db.QueryRowContext(ctx, query, userID).Scan(
		&profile.ID, &profile.Username, &profile.Email, &profile.Phone,
		&profile.FirstName, &profile.LastName, &profile.Avatar,
		&profile.Provider, &profile.ProviderID, &profile.IsActive,
		&profile.EmailVerified, &profile.PhoneVerified, &lastSignIn,
		&rolesJSON, &metadataJSON, &profile.CreatedAt, &profile.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if lastSignIn.Valid {
		profile.LastSignInAt = &lastSignIn.Time
	}

	if rolesJSON.Valid {
		json.Unmarshal([]byte(rolesJSON.String), &profile.Roles)
	}

	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &profile.Metadata)
	}

	// Build display name
	if profile.FirstName != "" && profile.LastName != "" {
		profile.DisplayName = profile.FirstName + " " + profile.LastName
	} else if profile.FirstName != "" {
		profile.DisplayName = profile.FirstName
	} else if profile.LastName != "" {
		profile.DisplayName = profile.LastName
	} else {
		profile.DisplayName = profile.Username
	}

	return &profile, nil
}

func (h *UserProfilesHandler) updateUserProfile(ctx context.Context, userID string, req *UserProfileUpdateRequest) error {
	query := `UPDATE auth_users SET
		username = COALESCE(?, username),
		email = COALESCE(?, email),
		phone = COALESCE(?, phone),
		first_name = COALESCE(?, first_name),
		last_name = COALESCE(?, last_name),
		avatar = COALESCE(?, avatar),
		metadata = COALESCE(?, metadata),
		updated_at = ?
		WHERE id = ?`

	var metadataJSON []byte
	if req.Metadata != nil {
		metadataJSON, _ = json.Marshal(req.Metadata)
	}

	now := time.Now()
	_, err := h.db.ExecContext(ctx, query,
		req.Username, req.Email, req.Phone, req.FirstName, req.LastName,
		req.Avatar, metadataJSON, now, userID,
	)

	return err
}

func (h *UserProfilesHandler) validateProfileUpdate(req *UserProfileUpdateRequest) error {
	if req.Username != "" && len(req.Username) < 3 {
		return fmt.Errorf("username must be at least 3 characters")
	}

	if req.Email != "" && !auth.IsEmailValid(req.Email) {
		return fmt.Errorf("invalid email format")
	}

	if req.Phone != "" && !auth.IsPhoneValid(req.Phone) {
		return fmt.Errorf("invalid phone format")
	}

	return nil
}

func (h *UserProfilesHandler) validatePasswordChange(req *PasswordChangeRequest) error {
	if req.CurrentPassword == "" {
		return fmt.Errorf("current password is required")
	}

	if req.NewPassword == "" {
		return fmt.Errorf("new password is required")
	}

	if req.NewPassword != req.ConfirmPassword {
		return fmt.Errorf("password confirmation does not match")
	}

	if len(req.NewPassword) < 8 {
		return fmt.Errorf("new password must be at least 8 characters")
	}

	return nil
}

func (h *UserProfilesHandler) getUser(ctx context.Context, userID string) (*auth.User, error) {
	query := `SELECT id, username, email, provider, provider_id, is_active,
		roles, created_at, updated_at FROM auth_users WHERE id = ?`

	var user auth.User
	var rolesJSON sql.NullString

	err := h.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.Provider, &user.ProviderID,
		&user.IsActive, &rolesJSON, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if rolesJSON.Valid {
		json.Unmarshal([]byte(rolesJSON.String), &user.Roles)
	}

	return &user, nil
}

func (h *UserProfilesHandler) getConnectedAccounts(ctx context.Context, userID string) ([]ConnectedAccount, error) {
	query := `SELECT id, provider, provider_user_id, provider_data, is_active,
		created_at, updated_at FROM user_connected_accounts
		WHERE user_id = ? ORDER BY created_at DESC`

	rows, err := h.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []ConnectedAccount
	for rows.Next() {
		var account ConnectedAccount
		var providerDataJSON sql.NullString

		err := rows.Scan(
			&account.ID, &account.Provider, &account.ProviderUserID,
			&providerDataJSON, &account.IsActive, &account.CreatedAt, &account.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		if providerDataJSON.Valid {
			json.Unmarshal([]byte(providerDataJSON.String), &account.ProviderData)
		}

		accounts = append(accounts, account)
	}

	return accounts, nil
}

func (h *UserProfilesHandler) syncFromProvider(ctx context.Context, userID, provider string) error {
	// This would sync user profile information from the OAuth provider
	// Implementation would depend on the specific provider
	// For now, just update the last_sign_in_at timestamp

	query := `UPDATE auth_users SET last_sign_in_at = ?, updated_at = ? WHERE id = ?`
	now := time.Now()
	_, err := h.db.ExecContext(ctx, query, now, now, userID)

	return err
}
