package auth

import ("crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5")

// JWTConfig represents JWT configuration
type JWTConfig struct {
	SecretKey    string        `json:"secret_key"`
	Issuer       string        `json:"issuer"`
	Expiry       time.Duration `json:"expiry"`
	RefreshExpiry time.Duration `json:"refresh_expiry"`
}

// Claims represents JWT claims
type Claims struct {
	UserID      string                 `json:"user_id"`
	TenantID    string                 `json:"tenant_id"`
	ProjectID   string                 `json:"project_id"`
	Roles       []string               `json:"roles"`
	Permissions []string               `json:"permissions"`
	Metadata    map[string]interface{} `json:"metadata"`
	jwt.RegisteredClaims
}

// JWTManager manages JWT tokens
type JWTManager struct {
	config *JWTConfig
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(config *JWTConfig) *JWTManager {
	if config.SecretKey == "" {
		// Generate random secret key if not provided
		config.SecretKey = generateSecretKey()
	}
	if config.Issuer == "" {
		config.Issuer = "metabase"
	}
	if config.Expiry == 0 {
		config.Expiry = time.Hour
	}
	if config.RefreshExpiry == 0 {
		config.RefreshExpiry = 24 * time.Hour
	}

	return &JWTManager{
		config: config,
	}
}

// GenerateToken generates a new JWT token
func (j *JWTManager) GenerateToken(userID, tenantID, projectID string, roles, permissions []string, metadata map[string]interface{}) (string, error) {
	now := time.Now()

	claims := &Claims{
		UserID:      userID,
		TenantID:    tenantID,
		ProjectID:   projectID,
		Roles:       roles,
		Permissions: permissions,
		Metadata:    metadata,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        generateTokenID(),
			Issuer:    j.config.Issuer,
			Subject:   userID,
			Audience:  []string{"metabase-api"},
			ExpiresAt: jwt.NewNumericDate(now.Add(j.config.Expiry)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.config.SecretKey))
}

// GenerateRefreshToken generates a refresh token
func (j *JWTManager) GenerateRefreshToken(userID string) (string, error) {
	now := time.Now()

	claims := &jwt.RegisteredClaims{
		ID:        generateTokenID(),
		Issuer:    j.config.Issuer,
		Subject:   userID,
		Audience:  []string{"metabase-refresh"},
		ExpiresAt: jwt.NewNumericDate(now.Add(j.config.RefreshExpiry)),
		NotBefore: jwt.NewNumericDate(now),
		IssuedAt:  jwt.NewNumericDate(now),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.config.SecretKey))
}

// ValidateToken validates a JWT token and returns claims
func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.config.SecretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token
func (j *JWTManager) ValidateRefreshToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.config.SecretKey), nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse refresh token: %w", err)
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid refresh token")
	}

	// Check audience
	for _, audience := range claims.Audience {
		if audience == "metabase-refresh" {
			return claims.Subject, nil
		}
	}

	return "", fmt.Errorf("invalid refresh token audience")
}

// RefreshToken generates a new access token from refresh token
func (j *JWTManager) RefreshToken(refreshToken string, userID, tenantID, projectID string, roles, permissions []string, metadata map[string]interface{}) (string, error) {
	// Validate refresh token
	refreshUserID, err := j.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Verify user ID matches
	if refreshUserID != userID {
		return "", fmt.Errorf("user ID mismatch")
	}

	// Generate new access token
	return j.GenerateToken(userID, tenantID, projectID, roles, permissions, metadata)
}

// generateSecretKey generates a random secret key
func generateSecretKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a less secure key if random generation fails
		return "metabase-secret-key-fallback-please-change-in-production"
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// generateTokenID generates a unique token ID
func generateTokenID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("tok_%d", time.Now().UnixNano())
	}
	return base64.URLEncoding.EncodeToString(bytes)
}