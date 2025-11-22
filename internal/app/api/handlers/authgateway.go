package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/guileen/metabase/internal/app/api/rest"
	"github.com/guileen/metabase/internal/biz/domain/authgateway"
	apperrors "github.com/guileen/metabase/pkg/common/errors"
	"go.uber.org/zap"
)

// AuthGatewayHandler handles authentication gateway API endpoints
type AuthGatewayHandler struct {
	authGatewayManager *authgateway.AuthGatewayManager
	logger             *zap.Logger
}

// NewAuthGatewayHandler creates a new authentication gateway handler
func NewAuthGatewayHandler(authGatewayManager *authgateway.AuthGatewayManager, logger *zap.Logger) *AuthGatewayHandler {
	return &AuthGatewayHandler{
		authGatewayManager: authGatewayManager,
		logger:             logger,
	}
}

// AuthRequest represents authentication request
type AuthRequest struct {
	Provider    string                 `json:"provider" validate:"required"`
	Method      string                 `json:"method" validate:"required"`
	Credentials map[string]interface{} `json:"credentials"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Success      bool                   `json:"success"`
	UserID       string                 `json:"user_id,omitempty"`
	Username     string                 `json:"username,omitempty"`
	Email        string                 `json:"email,omitempty"`
	TenantID     string                 `json:"tenant_id,omitempty"`
	Roles        []string               `json:"roles,omitempty"`
	AccessToken  string                 `json:"access_token,omitempty"`
	RefreshToken string                 `json:"refresh_token,omitempty"`
	ExpiresIn    int                    `json:"expires_in,omitempty"`
	Message      string                 `json:"message,omitempty"`
	UserInfo     *authgateway.UserInfo  `json:"user_info,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// RegisterRequest represents user registration request
type RegisterRequest struct {
	Username    string                 `json:"username" validate:"required,min=3"`
	Email       string                 `json:"email" validate:"required,email"`
	Password    string                 `json:"password" validate:"required,min=8"`
	DisplayName string                 `json:"display_name,omitempty"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	Roles       []string               `json:"roles,omitempty"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
}

// ChangePasswordRequest represents password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// ResetPasswordRequest represents password reset request
type ResetPasswordRequest struct {
	Email    string `json:"email" validate:"required,email"`
	TenantID string `json:"tenant_id,omitempty"`
}

// ConfirmResetPasswordRequest represents password reset confirmation request
type ConfirmResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// UpdateUserRequest represents user update request
type UpdateUserRequest struct {
	Username    string                 `json:"username,omitempty"`
	Email       string                 `json:"email,omitempty"`
	DisplayName string                 `json:"display_name,omitempty"`
	Avatar      string                 `json:"avatar,omitempty"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
}

// Handler routes

// Authenticate handles authentication requests
// @Summary Authenticate user
// @Description Authenticate a user with credentials
// @Tags auth-gateway
// @Accept json
// @Produce json
// @Param request body AuthRequest true "Authentication request"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} rest.ErrorResponse
// @Failure 401 {object} rest.ErrorResponse
// @Failure 500 {object} rest.ErrorResponse
// @Router /api/v1/auth/authenticate [post]
func (h *AuthGatewayHandler) Authenticate(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if err := rest.ValidateStruct(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// Extract client information
	ipAddress := rest.GetClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	// Build authentication request
	authReq := &authgateway.AuthRequest{
		Provider:    req.Provider,
		Method:      req.Method,
		Credentials: req.Credentials,
		TenantID:    req.TenantID,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Metadata:    req.Metadata,
	}

	// Authenticate
	result, err := h.authGatewayManager.Authenticate(r.Context(), authReq)
	if err != nil {
		h.logger.Error("Authentication failed", zap.Error(err))
		rest.WriteError(w, http.StatusInternalServerError, "Authentication failed", err)
		return
	}

	// Build response
	response := &AuthResponse{
		Success:      result.Success,
		UserID:       result.UserID,
		Username:     result.Username,
		Email:        result.Email,
		TenantID:     result.TenantID,
		Roles:        result.Roles,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		Message:      result.Message,
		UserInfo:     nil, // Will be populated if needed
		Metadata:     result.Metadata,
	}

	if result.Success {
		rest.WriteJSON(w, http.StatusOK, response)
	} else {
		rest.WriteError(w, http.StatusUnauthorized, result.Message, nil)
	}
}

// Register handles user registration
// @Summary Register new user
// @Description Register a new user account
// @Tags auth-gateway
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration request"
// @Success 201 {object} authgateway.UserInfo
// @Failure 400 {object} rest.ErrorResponse
// @Failure 409 {object} rest.ErrorResponse
// @Failure 500 {object} rest.ErrorResponse
// @Router /api/v1/auth/register [post]
func (h *AuthGatewayHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if err := rest.ValidateStruct(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// Build registration request
	registrationReq := &authgateway.UserRegistration{
		Username:    req.Username,
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
		TenantID:    req.TenantID,
		Roles:       req.Roles,
		Attributes:  req.Attributes,
	}

	// Register user
	userInfo, err := h.authGatewayManager.RegisterUser(r.Context(), registrationReq)
	if err != nil {
		if appErr, ok := apperrors.IsAppError(err); ok {
			if appErr.Code == apperrors.ErrCodeInvalidInput {
				rest.WriteError(w, http.StatusBadRequest, "Registration failed", err)
			} else if appErr.Code == apperrors.ErrCodeAlreadyExists {
				rest.WriteError(w, http.StatusConflict, "User already exists", err)
			} else {
				h.logger.Error("User registration failed", zap.Error(err))
				rest.WriteError(w, http.StatusInternalServerError, "Registration failed", err)
			}
		} else {
			h.logger.Error("User registration failed", zap.Error(err))
			rest.WriteError(w, http.StatusInternalServerError, "Registration failed", err)
		}
		return
	}

	rest.WriteJSON(w, http.StatusCreated, userInfo)
}

// ValidateToken handles token validation
// @Summary Validate access token
// @Description Validate an access token and return user information
// @Tags auth-gateway
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} authgateway.UserInfo
// @Failure 401 {object} rest.ErrorResponse
// @Failure 500 {object} rest.ErrorResponse
// @Router /api/v1/auth/validate [get]
func (h *AuthGatewayHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	token := rest.ExtractBearerToken(r)
	if token == "" {
		rest.WriteError(w, http.StatusUnauthorized, "Authorization token required", nil)
		return
	}

	userInfo, err := h.authGatewayManager.ValidateToken(r.Context(), token)
	if err != nil {
		rest.WriteError(w, http.StatusUnauthorized, "Invalid token", err)
		return
	}

	rest.WriteJSON(w, http.StatusOK, userInfo)
}

// RefreshToken handles token refresh
// @Summary Refresh access token
// @Description Refresh an access token using refresh token
// @Tags auth-gateway
// @Accept json
// @Produce json
// @Param request body map[string]string true "Refresh token request"
// @Success 200 {object} authgateway.TokenResult
// @Failure 400 {object} rest.ErrorResponse
// @Failure 401 {object} rest.ErrorResponse
// @Failure 500 {object} rest.ErrorResponse
// @Router /api/v1/auth/refresh [post]
func (h *AuthGatewayHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	refreshToken, ok := req["refresh_token"]
	if !ok || refreshToken == "" {
		rest.WriteError(w, http.StatusBadRequest, "Refresh token is required", nil)
		return
	}

	tokenResult, err := h.authGatewayManager.RefreshToken(r.Context(), refreshToken)
	if err != nil {
		rest.WriteError(w, http.StatusUnauthorized, "Failed to refresh token", err)
		return
	}

	rest.WriteJSON(w, http.StatusOK, tokenResult)
}

// Logout handles user logout
// @Summary Logout user
// @Description Logout user and invalidate session
// @Tags auth-gateway
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} map[string]string
// @Failure 401 {object} rest.ErrorResponse
// @Failure 500 {object} rest.ErrorResponse
// @Router /api/v1/auth/logout [post]
func (h *AuthGatewayHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := rest.ExtractBearerToken(r)
	if token == "" {
		rest.WriteError(w, http.StatusUnauthorized, "Authorization token required", nil)
		return
	}

	if err := h.authGatewayManager.Logout(r.Context(), token); err != nil {
		h.logger.Error("Logout failed", zap.Error(err))
		rest.WriteError(w, http.StatusInternalServerError, "Logout failed", err)
		return
	}

	rest.WriteJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// GetUserInfo handles get user info
// @Summary Get user information
// @Description Get current user information
// @Tags auth-gateway
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} authgateway.UserInfo
// @Failure 401 {object} rest.ErrorResponse
// @Failure 404 {object} rest.ErrorResponse
// @Failure 500 {object} rest.ErrorResponse
// @Router /api/v1/auth/user [get]
func (h *AuthGatewayHandler) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	userID := rest.GetUserIDFromContext(r)
	if userID == "" {
		rest.WriteError(w, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	userInfo, err := h.authGatewayManager.GetUserInfo(r.Context(), userID)
	if err != nil {
		if appErr, ok := apperrors.IsAppError(err); ok && appErr.Code == apperrors.ErrCodeNotFound {
			rest.WriteError(w, http.StatusNotFound, "User not found", err)
		} else {
			rest.WriteError(w, http.StatusInternalServerError, "Failed to get user info", err)
		}
		return
	}

	rest.WriteJSON(w, http.StatusOK, userInfo)
}

// UpdateUserInfo handles update user info
// @Summary Update user information
// @Description Update current user information
// @Tags auth-gateway
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body UpdateUserRequest true "Update user request"
// @Success 200 {object} authgateway.UserInfo
// @Failure 400 {object} rest.ErrorResponse
// @Failure 401 {object} rest.ErrorResponse
// @Failure 500 {object} rest.ErrorResponse
// @Router /api/v1/auth/user [put]
func (h *AuthGatewayHandler) UpdateUserInfo(w http.ResponseWriter, r *http.Request) {
	userID := rest.GetUserIDFromContext(r)
	if userID == "" {
		rest.WriteError(w, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Get current user info
	userInfo, err := h.authGatewayManager.GetUserInfo(r.Context(), userID)
	if err != nil {
		rest.WriteError(w, http.StatusInternalServerError, "Failed to get current user info", err)
		return
	}

	// Update fields
	if req.Username != "" {
		userInfo.Username = req.Username
	}
	if req.Email != "" {
		userInfo.Email = req.Email
	}
	if req.DisplayName != "" {
		userInfo.DisplayName = req.DisplayName
	}
	if req.Avatar != "" {
		userInfo.Avatar = req.Avatar
	}
	if req.Attributes != nil {
		userInfo.Attributes = req.Attributes
	}
	userInfo.UpdatedAt = time.Now()

	// Update user
	if err := h.authGatewayManager.UpdateUserInfo(r.Context(), userInfo); err != nil {
		rest.WriteError(w, http.StatusInternalServerError, "Failed to update user info", err)
		return
	}

	rest.WriteJSON(w, http.StatusOK, userInfo)
}

// ChangePassword handles password change
// @Summary Change password
// @Description Change user password
// @Tags auth-gateway
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body ChangePasswordRequest true "Change password request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} rest.ErrorResponse
// @Failure 401 {object} rest.ErrorResponse
// @Failure 500 {object} rest.ErrorResponse
// @Router /api/v1/auth/password/change [post]
func (h *AuthGatewayHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := rest.GetUserIDFromContext(r)
	if userID == "" {
		rest.WriteError(w, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if err := rest.ValidateStruct(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// Change password
	changeReq := &authgateway.PasswordChange{
		UserID:      userID,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	}

	if err := h.authGatewayManager.ChangePassword(r.Context(), changeReq); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Failed to change password", err)
		return
	}

	rest.WriteJSON(w, http.StatusOK, map[string]string{"message": "Password changed successfully"})
}

// ResetPassword handles password reset request
// @Summary Reset password
// @Description Request password reset
// @Tags auth-gateway
// @Accept json
// @Produce json
// @Param request body ResetPasswordRequest true "Reset password request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} rest.ErrorResponse
// @Failure 500 {object} rest.ErrorResponse
// @Router /api/v1/auth/password/reset [post]
func (h *AuthGatewayHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if err := rest.ValidateStruct(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Validation failed", err)
		return
	}

	resetReq := &authgateway.PasswordReset{
		Email:    req.Email,
		TenantID: req.TenantID,
	}

	if err := h.authGatewayManager.ResetPassword(r.Context(), resetReq); err != nil {
		h.logger.Error("Password reset failed", zap.Error(err))
		rest.WriteError(w, http.StatusInternalServerError, "Password reset failed", err)
		return
	}

	// Always return success to prevent email enumeration attacks
	rest.WriteJSON(w, http.StatusOK, map[string]string{"message": "Password reset email sent"})
}

// ConfirmResetPassword handles password reset confirmation
// @Summary Confirm password reset
// @Description Confirm password reset with token
// @Tags auth-gateway
// @Accept json
// @Produce json
// @Param request body ConfirmResetPasswordRequest true "Confirm reset password request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} rest.ErrorResponse
// @Failure 500 {object} rest.ErrorResponse
// @Router /api/v1/auth/password/reset/confirm [post]
func (h *AuthGatewayHandler) ConfirmResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ConfirmResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if err := rest.ValidateStruct(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// This would need to be implemented in the auth gateway manager
	// For now, return a placeholder response
	rest.WriteJSON(w, http.StatusOK, map[string]string{"message": "Password reset confirmed"})
}

// GetProviders handles get authentication providers
// @Summary Get authentication providers
// @Description Get list of available authentication providers
// @Tags auth-gateway
// @Accept json
// @Produce json
// @Success 200 {array} authgateway.ProviderInfo
// @Failure 500 {object} rest.ErrorResponse
// @Router /api/v1/auth/providers [get]
func (h *AuthGatewayHandler) GetProviders(w http.ResponseWriter, r *http.Request) {
	providers := h.authGatewayManager.GetProviders()
	rest.WriteJSON(w, http.StatusOK, providers)
}

// GetStats handles get authentication statistics
// @Summary Get authentication statistics
// @Description Get authentication statistics for monitoring
// @Tags auth-gateway
// @Accept json
// @Produce json
// @Param tenant_id query string false "Tenant ID"
// @Success 200 {object} authgateway.AuthStats
// @Failure 401 {object} rest.ErrorResponse
// @Failure 500 {object} rest.ErrorResponse
// @Router /api/v1/auth/stats [get]
func (h *AuthGatewayHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")

	stats, err := h.authGatewayManager.GetStats(r.Context(), tenantID)
	if err != nil {
		rest.WriteError(w, http.StatusInternalServerError, "Failed to get statistics", err)
		return
	}

	rest.WriteJSON(w, http.StatusOK, stats)
}

// Health handles health check
// @Summary Health check
// @Description Check if authentication gateway is healthy
// @Tags auth-gateway
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/auth/health [get]
func (h *AuthGatewayHandler) Health(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "auth-gateway",
	}

	// Check if auth gateway manager is initialized
	if h.authGatewayManager == nil {
		health["status"] = "unhealthy"
		health["error"] = "auth gateway manager not initialized"
		rest.WriteJSON(w, http.StatusServiceUnavailable, health)
		return
	}

	rest.WriteJSON(w, http.StatusOK, health)
}

// OAuth handlers

// OAuthLogin handles OAuth login initiation
// @Summary Initiate OAuth login
// @Description Initiate OAuth login with specified provider
// @Tags auth-gateway
// @Accept json
// @Produce json
// @Param provider path string true "OAuth provider"
// @Param tenant_id query string false "Tenant ID"
// @Success 307
// @Failure 400 {object} rest.ErrorResponse
// @Failure 500 {object} rest.ErrorResponse
// @Router /api/v1/auth/oauth/{provider}/login [get]
func (h *AuthGatewayHandler) OAuthLogin(w http.ResponseWriter, r *http.Request) {
	provider := rest.GetPathParam(r, "provider")
	tenantID := r.URL.Query().Get("tenant_id")
	_ = tenantID // TODO: Use tenantID in OAuth flow

	if provider == "" {
		rest.WriteError(w, http.StatusBadRequest, "Provider is required", nil)
		return
	}

	// TODO: Implement OAuth login initiation
	// This would redirect to the OAuth provider's authorization URL

	rest.WriteError(w, http.StatusNotImplemented, "OAuth login not implemented", nil)
}

// OAuthCallback handles OAuth callback
// @Summary Handle OAuth callback
// @Description Handle OAuth callback from provider
// @Tags auth-gateway
// @Accept json
// @Produce json
// @Param provider path string true "OAuth provider"
// @Param code query string true "Authorization code"
// @Param state query string true "State parameter"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} rest.ErrorResponse
// @Failure 500 {object} rest.ErrorResponse
// @Router /api/v1/auth/oauth/{provider}/callback [get]
func (h *AuthGatewayHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	provider := rest.GetPathParam(r, "provider")
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	_ = state // TODO: Use state in OAuth flow

	if provider == "" || code == "" {
		rest.WriteError(w, http.StatusBadRequest, "Provider and authorization code are required", nil)
		return
	}

	// TODO: Implement OAuth callback handling
	// This would exchange the authorization code for tokens and authenticate the user

	rest.WriteError(w, http.StatusNotImplemented, "OAuth callback not implemented", nil)
}
