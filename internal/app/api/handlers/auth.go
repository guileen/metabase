package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(db *sql.DB, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		db:     db,
		logger: logger,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token        string   `json:"token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int      `json:"expires_in"`
	User         UserInfo `json:"user"`
}

// UserInfo represents user information
type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	TenantID string `json:"tenant_id"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	TenantID string `json:"tenant_id,omitempty"`
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		h.writeError(w, "Email and password required", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual user authentication
	// For now, return a mock response
	mockUser := UserInfo{
		ID:       "user_123",
		Email:    req.Email,
		Name:     "Test User",
		Role:     "user",
		TenantID: "tenant_123",
	}

	// Generate JWT token
	token, err := h.generateJWT(mockUser)
	if err != nil {
		h.logger.Error("Failed to generate JWT", zap.Error(err))
		h.writeError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Generate refresh token
	refreshToken, err := h.generateRefreshToken(mockUser)
	if err != nil {
		h.logger.Error("Failed to generate refresh token", zap.Error(err))
		h.writeError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresIn:    3600, // 1 hour
		User:         mockUser,
	}

	h.writeJSON(w, response)
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" || req.Name == "" {
		h.writeError(w, "Email, password, and name required", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual user registration
	// For now, return a mock response
	mockUser := UserInfo{
		ID:       "user_" + time.Now().Format("20060102150405"),
		Email:    req.Email,
		Name:     req.Name,
		Role:     "user",
		TenantID: req.TenantID,
	}

	if mockUser.TenantID == "" {
		mockUser.TenantID = "default_tenant"
	}

	// Hash password (in real implementation)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("Failed to hash password", zap.Error(err))
		h.writeError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("User registered (mock)",
		zap.String("email", req.Email),
		zap.String("name", req.Name),
		zap.String("hashed_password", string(hashedPassword)),
	)

	// Generate JWT token
	token, err := h.generateJWT(mockUser)
	if err != nil {
		h.logger.Error("Failed to generate JWT", zap.Error(err))
		h.writeError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"token": token,
		"user":  mockUser,
	}

	h.writeJSON(w, response)
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.RefreshToken == "" {
		h.writeError(w, "Refresh token required", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual refresh token validation
	// For now, generate a new token
	mockUser := UserInfo{
		ID:       "user_123",
		Email:    "test@example.com",
		Name:     "Test User",
		Role:     "user",
		TenantID: "tenant_123",
	}

	token, err := h.generateJWT(mockUser)
	if err != nil {
		h.logger.Error("Failed to generate JWT", zap.Error(err))
		h.writeError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"token":      token,
		"expires_in": 3600,
	}

	h.writeJSON(w, response)
}

// Helper methods

func (h *AuthHandler) generateJWT(user UserInfo) (string, error) {
	// This is a mock implementation
	// In production, use proper JWT secret and claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":   user.ID,
		"email":     user.Email,
		"name":      user.Name,
		"role":      user.Role,
		"tenant_id": user.TenantID,
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	})

	// TODO: Use proper secret from configuration
	secret := "your-secret-key"
	return token.SignedString([]byte(secret))
}

func (h *AuthHandler) generateRefreshToken(user UserInfo) (string, error) {
	// This is a mock implementation
	// In production, use proper token generation and storage
	return "refresh_" + user.ID + "_" + time.Now().Format("20060102150405"), nil
}

func (h *AuthHandler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *AuthHandler) writeError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
		"code":  code,
	})
}
