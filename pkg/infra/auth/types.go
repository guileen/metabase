package auth

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user account
type User struct {
	ID         string                 `json:"id" yaml:"id"`
	Email      string                 `json:"email" yaml:"email"`
	Username   string                 `json:"username,omitempty" yaml:"username,omitempty"`
	Phone      string                 `json:"phone,omitempty" yaml:"phone,omitempty"`
	Password   string                 `json:"-" yaml:"password"` // Hidden in JSON
	FirstName  string                 `json:"first_name,omitempty" yaml:"first_name,omitempty"`
	LastName   string                 `json:"last_name,omitempty" yaml:"last_name,omitempty"`
	Avatar     string                 `json:"avatar,omitempty" yaml:"avatar,omitempty"`
	Provider   string                 `json:"provider,omitempty" yaml:"provider,omitempty"` // "email", "google", "github", etc.
	ProviderID string                 `json:"provider_id,omitempty" yaml:"provider_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Roles      []string               `json:"roles,omitempty" yaml:"roles,omitempty"`

	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty" yaml:"email_verified_at,omitempty"`
	PhoneVerifiedAt *time.Time `json:"phone_verified_at,omitempty" yaml:"phone_verified_at,omitempty"`
	LastSignInAt    *time.Time `json:"last_sign_in_at,omitempty" yaml:"last_sign_in_at,omitempty"`
	PasswordResetAt *time.Time `json:"password_reset_at,omitempty" yaml:"password_reset_at,omitempty"`

	IsAnonymous     bool `json:"is_anonymous,omitempty" yaml:"is_anonymous,omitempty"`
	IsActive        bool `json:"is_active" yaml:"is_active"`
	IsEmailVerified bool `json:"is_email_verified" yaml:"is_email_verified"`
	IsPhoneVerified bool `json:"is_phone_verified,omitempty" yaml:"is_phone_verified,omitempty"`

	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
}

// ProviderConfig represents an auth provider configuration
type ProviderConfig struct {
	Name      string                 `json:"name" yaml:"name"`
	Type      string                 `json:"type" yaml:"type"` // "email", "oauth2", "saml", "phone"
	Enabled   bool                   `json:"enabled" yaml:"enabled"`
	Config    map[string]interface{} `json:"config" yaml:"config"`
	Scopes    []string               `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	CreatedAt time.Time              `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" yaml:"updated_at"`
}

// AuthRequest represents an authentication request
type AuthRequest struct {
	Provider     string                 `json:"provider"`
	Method       string                 `json:"method"` // "login", "signup", "refresh", "logout"
	Email        string                 `json:"email,omitempty"`
	Password     string                 `json:"password,omitempty"`
	Phone        string                 `json:"phone,omitempty"`
	Code         string                 `json:"code,omitempty"`          // Verification code
	OAuthToken   string                 `json:"oauth_token,omitempty"`   // OAuth token
	RefreshToken string                 `json:"refresh_token,omitempty"` // Refresh token
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	Success   bool                   `json:"success"`
	User      *User                  `json:"user,omitempty"`
	Session   *Session               `json:"session,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Error     string                 `json:"error,omitempty"`
	ErrorCode string                 `json:"error_code,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// VerificationRequest represents a verification request
type VerificationRequest struct {
	Type     string `json:"type"` // "email", "phone", "password_reset"
	Email    string `json:"email,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Language string `json:"language,omitempty"`
}

// MagicLink represents a magic link for passwordless login
type MagicLink struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Email     string     `json:"email"`
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
}

// PasswordReset represents a password reset request
type PasswordReset struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Email     string     `json:"email"`
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
}

// AuditLog represents an authentication audit log entry
type AuditLog struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id,omitempty"`
	Action    string                 `json:"action"` // "login", "signup", "logout", "password_change", etc.
	Provider  string                 `json:"provider,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// IdentityProvider represents an OAuth identity provider
type IdentityProvider struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	DisplayName  string                 `json:"display_name"`
	Type         string                 `json:"type"` // "oauth2", "oidc", "saml"
	ClientID     string                 `json:"client_id"`
	ClientSecret string                 `json:"client_secret"`
	AuthURL      string                 `json:"auth_url"`
	TokenURL     string                 `json:"token_url"`
	UserInfoURL  string                 `json:"user_info_url"`
	Scopes       []string               `json:"scopes"`
	Enabled      bool                   `json:"enabled"`
	Config       map[string]interface{} `json:"config,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// Constants
const (
	// Providers
	ProviderEmail    = "email"
	ProviderGoogle   = "google"
	ProviderGitHub   = "github"
	ProviderFacebook = "facebook"
	ProviderApple    = "apple"
	ProviderPhone    = "phone"
	ProviderSAML     = "saml"

	// Chinese domestic providers
	ProviderWeChat   = "wechat"
	ProviderDingTalk = "dingtalk"
	ProviderFeishu   = "feishu"
	ProviderQQ       = "qq"
	ProviderWeibo    = "weibo"
	ProviderAlipay   = "alipay"

	// Auth actions
	ActionLogin         = "login"
	ActionSignup        = "signup"
	ActionLogout        = "logout"
	ActionRefresh       = "refresh"
	ActionPasswordReset = "password_reset"
	ActionEmailVerify   = "email_verify"
	ActionPhoneVerify   = "phone_verify"

	// User roles
	RoleAnonymous  = "anonymous"
	RoleUser       = "user"
	RoleAdmin      = "admin"
	RoleSuperAdmin = "super_admin"

	// Default permissions
	PermissionReadAll   = "read:all"
	PermissionWriteAll  = "write:all"
	PermissionDeleteAll = "delete:all"
	PermissionAdminAll  = "admin:all"
)

// Error types
var (
	ErrInvalidCredentials   = fmt.Errorf("invalid credentials")
	ErrUserNotFound         = fmt.Errorf("user not found")
	ErrUserAlreadyExists    = fmt.Errorf("user already exists")
	ErrEmailAlreadyExists   = fmt.Errorf("email already exists")
	ErrPhoneAlreadyExists   = fmt.Errorf("phone already exists")
	ErrInvalidToken         = fmt.Errorf("invalid token")
	ErrTokenExpired         = fmt.Errorf("token expired")
	ErrSessionExpired       = fmt.Errorf("session expired")
	ErrProviderNotFound     = fmt.Errorf("provider not found")
	ErrProviderDisabled     = fmt.Errorf("provider disabled")
	ErrEmailNotVerified     = fmt.Errorf("email not verified")
	ErrPhoneNotVerified     = fmt.Errorf("phone not verified")
	ErrAccountDisabled      = fmt.Errorf("account disabled")
	ErrPermissionDenied     = fmt.Errorf("permission denied")
	ErrTooManyAttempts      = fmt.Errorf("too many authentication attempts")
	ErrInvalidOTP           = fmt.Errorf("invalid OTP")
	ErrOTPExpired           = fmt.Errorf("OTP expired")
	ErrMagicLinkExpired     = fmt.Errorf("magic link expired")
	ErrPasswordResetExpired = fmt.Errorf("password reset expired")
)

// Helper functions

// IsEmailValid checks if an email address is valid
func IsEmailValid(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// IsPhoneValid checks if a phone number is valid
func IsPhoneValid(phone string) bool {
	// Basic phone validation - in production, use a proper phone validation library
	phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	return phoneRegex.MatchString(phone)
}

// HashPassword creates a hash of the password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword verifies a password against its hash
func CheckPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// GenerateToken creates a JWT token
func GenerateToken(userID string, expiresAt time.Time, secret string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     expiresAt.Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken validates a JWT token
func ValidateToken(tokenString, secret string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, ok := claims["user_id"].(string)
		if !ok {
			return "", fmt.Errorf("invalid user_id in token")
		}
		return userID, nil
	}

	return "", fmt.Errorf("invalid token")
}

// GenerateRandomToken creates a random token
func GenerateRandomToken(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		var n uint16
		binary.Read(rand.Reader, binary.LittleEndian, &n)
		b[i] = charset[int(n)%len(charset)]
	}
	return string(b)
}

// GenerateMagicLinkToken creates a magic link token
func GenerateMagicLinkToken() string {
	return GenerateRandomToken(32)
}

// GenerateOTPToken creates an OTP token
func GenerateOTPToken() string {
	var n uint32
	binary.Read(rand.Reader, binary.LittleEndian, &n)
	return fmt.Sprintf("%06d", n%1000000)
}
