package auth

import ("context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/guileen/metabase/pkg/common/errors"
	"golang.org/x/crypto/bcrypt")

// Manager manages user sessions and API keys
type Manager struct {
	db       *sql.DB
	jwt      *auth.JWTManager
	rbac     *auth.RBACManager
	config   *SessionConfig
	cache    *SessionCache
	mu       sync.RWMutex
}

// SessionConfig represents session configuration
type SessionConfig struct {
	SessionTimeout     time.Duration `json:"session_timeout"`
	RefreshTokenExpiry time.Duration `json:"refresh_token_expiry"`
	MaxSessionsPerUser int          `json:"max_sessions_per_user"`
	EnableAPIKeys      bool          `json:"enable_api_keys"`
	APIKeyExpiry       time.Duration `json:"api_key_expiry"`
	CleanupInterval    time.Duration `json:"cleanup_interval"`
}

// Session represents a user session
type Session struct {
	ID            string                 `json:"id"`
	UserID        string                 `json:"user_id"`
	TenantID      string                 `json:"tenant_id"`
	ProjectID     string                 `json:"project_id,omitempty"`
	TokenHash     string                 `json:"token_hash"`
	RefreshToken  string                 `json:"refresh_token,omitempty"`
	IPAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent"`
	ExpiresAt     time.Time              `json:"expires_at"`
	LastActiveAt  time.Time              `json:"last_active_at"`
	IsActive      bool                   `json:"is_active"`
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
}

// APIKey represents an API key
type APIKey struct {
	ID           string                 `json:"id"`
	KeyHash      string                 `json:"key_hash"`
	Name         string                 `json:"name"`
	UserID       string                 `json:"user_id"`
	TenantID     string                 `json:"tenant_id"`
	ProjectID    string                 `json:"project_id,omitempty"`
	Permissions  []string               `json:"permissions"`
	RateLimit    int                    `json:"rate_limit"`
	ExpiresAt    *time.Time             `json:"expires_at,omitempty"`
	LastUsedAt   *time.Time             `json:"last_used_at,omitempty"`
	IsActive     bool                   `json:"is_active"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// SessionCache provides caching for session data
type SessionCache struct {
	sessions map[string]*Session
	apiKeys  map[string]*APIKey
	mu       sync.RWMutex
	ttl      time.Duration
	maxSize  int
}

// CacheEntry represents a cached entry
type CacheEntry struct {
	Data      interface{}
	ExpiresAt time.Time
}

// NewManager creates a new session manager
func NewManager(db *sql.DB, jwt *auth.JWTManager, rbac *auth.RBACManager, config *SessionConfig) *Manager {
	if config == nil {
		config = &SessionConfig{
			SessionTimeout:     time.Hour,
			RefreshTokenExpiry: 24 * time.Hour,
			MaxSessionsPerUser: 5,
			EnableAPIKeys:      true,
			APIKeyExpiry:       time.Hour * 24 * 30, // 30 days
			CleanupInterval:    time.Hour,
		}
	}

	manager := &Manager{
		db:     db,
		jwt:    jwt,
		rbac:   rbac,
		config: config,
		cache:  NewSessionCache(config.SessionTimeout/4, 10000),
	}

	// Start cleanup routine
	go manager.startCleanupRoutine()

	return manager
}

// NewSessionCache creates a new session cache
func NewSessionCache(ttl time.Duration, maxSize int) *SessionCache {
	return &SessionCache{
		sessions: make(map[string]*Session),
		apiKeys:  make(map[string]*APIKey),
		ttl:      ttl,
		maxSize:  maxSize,
	}
}

// CreateSession creates a new user session
func (m *Manager) CreateSession(ctx context.Context, userID, tenantID, projectID, ipAddress, userAgent string, metadata map[string]interface{}) (*Session, error) {
	// Check session limit
	if err := m.checkSessionLimit(ctx, userID); err != nil {
		return nil, err
	}

	// Generate tokens
	accessToken, err := m.jwt.GenerateToken(userID, tenantID, projectID, []string{}, []string{}, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := m.jwt.GenerateRefreshToken(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create session record
	session := &Session{
		ID:           generateSessionID(),
		UserID:       userID,
		TenantID:     tenantID,
		ProjectID:    projectID,
		TokenHash:    hashToken(accessToken),
		RefreshToken: refreshToken,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		ExpiresAt:    time.Now().Add(m.config.SessionTimeout),
		LastActiveAt: time.Now(),
		IsActive:     true,
		Metadata:     metadata,
		CreatedAt:    time.Now(),
	}

	// Save to database
	if err := m.saveSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// Update cache
	m.cache.SetSession(session)

	return session, nil
}

// ValidateSession validates a session token
func (m *Manager) ValidateSession(ctx context.Context, token string) (*Session, error) {
	// Check cache first
	if cached := m.cache.GetSessionByToken(token); cached != nil {
		return cached, nil
	}

	// Validate JWT
	claims, err := m.jwt.ValidateToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Load session from database
	session, err := m.getSessionByTokenHash(ctx, hashToken(token))
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Check if session is still valid
	if !session.IsActive || time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired or inactive")
	}

	// Update last active time
	session.LastActiveAt = time.Now()
	if err := m.updateSession(ctx, session); err != nil {
		// Log error but don't fail validation
		fmt.Printf("Failed to update session last active time: %v\n", err)
	}

	// Update cache
	m.cache.SetSession(session)

	return session, nil
}

// RefreshToken refreshes an access token using refresh token
func (m *Manager) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	// Validate refresh token
	userID, err := m.jwt.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Get user sessions
	sessions, err := m.getUserSessions(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user sessions: %w", err)
	}

	// Find session with matching refresh token
	var session *Session
	for _, s := range sessions {
		if s.RefreshToken == refreshToken && s.IsActive {
			session = s
			break
		}
	}

	if session == nil {
		return "", fmt.Errorf("refresh token not found or expired")
	}

	// Generate new access token
	newAccessToken, err := m.jwt.GenerateToken(
		session.UserID,
		session.TenantID,
		session.ProjectID,
		[]string{},
		[]string{},
		session.Metadata,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate new access token: %w", err)
	}

	// Update session with new token hash
	session.TokenHash = hashToken(newAccessToken)
	session.LastActiveAt = time.Now()
	session.ExpiresAt = time.Now().Add(m.config.SessionTimeout)

	if err := m.updateSession(ctx, session); err != nil {
		return "", fmt.Errorf("failed to update session: %w", err)
	}

	// Update cache
	m.cache.SetSession(session)

	return newAccessToken, nil
}

// InvalidateSession invalidates a session
func (m *Manager) InvalidateSession(ctx context.Context, sessionID string) error {
	session, err := m.getSession(ctx, sessionID)
	if err != nil {
		return err
	}

	session.IsActive = false
	session.ExpiresAt = time.Now() // Expire immediately

	if err := m.updateSession(ctx, session); err != nil {
		return fmt.Errorf("failed to invalidate session: %w", err)
	}

	// Remove from cache
	m.cache.DeleteSession(sessionID)

	return nil
}

// InvalidateUserSessions invalidates all sessions for a user
func (m *Manager) InvalidateUserSessions(ctx context.Context, userID string) error {
	sessions, err := m.getUserSessions(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user sessions: %w", err)
	}

	for _, session := range sessions {
		session.IsActive = false
		session.ExpiresAt = time.Now()

		if err := m.updateSession(ctx, session); err != nil {
			return fmt.Errorf("failed to invalidate session %s: %w", session.ID, err)
		}

		// Remove from cache
		m.cache.DeleteSession(session.ID)
	}

	return nil
}

// CreateAPIKey creates a new API key
func (m *Manager) CreateAPIKey(ctx context.Context, userID, tenantID, projectID, name string, permissions []string, rateLimit int, expiresAt *time.Time) (*APIKey, string, error) {
	if !m.config.EnableAPIKeys {
		return nil, fmt.Errorf("API keys are disabled")
	}

	// Generate API key
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	keyRecord := &APIKey{
		ID:          generateAPIKeyID(),
		KeyHash:     hashToken(apiKey),
		Name:        name,
		UserID:      userID,
		TenantID:    tenantID,
		ProjectID:   projectID,
		Permissions: permissions,
		RateLimit:   rateLimit,
		ExpiresAt:   expiresAt,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save to database
	if err := m.saveAPIKey(ctx, keyRecord); err != nil {
		return "", fmt.Errorf("failed to save API key: %w", err)
	}

	// Return the actual key (only shown once)
	return keyRecord, apiKey, nil
}

// ValidateAPIKey validates an API key
func (m *Manager) ValidateAPIKey(ctx context.Context, apiKey string) (*APIKey, error) {
	if !m.config.EnableAPIKeys {
		return nil, fmt.Errorf("API keys are disabled")
	}

	// Check cache first
	if cached := m.cache.GetAPIKeyByKey(apiKey); cached != nil {
		return cached, nil
	}

	// Validate key hash
	keyHash := hashToken(apiKey)

	// Load from database
	keyRecord, err := m.getAPIKeyByHash(ctx, keyHash)
	if err != nil {
		return nil, fmt.Errorf("API key not found: %w", err)
	}

	// Check if key is still valid
	if !keyRecord.IsActive {
		return nil, fmt.Errorf("API key is inactive")
	}

	if keyRecord.ExpiresAt != nil && time.Now().After(*keyRecord.ExpiresAt) {
		return nil, fmt.Errorf("API key has expired")
	}

	// Update last used time
	now := time.Now()
	keyRecord.LastUsedAt = &now
	keyRecord.UpdatedAt = now

	if err := m.updateAPIKey(ctx, keyRecord); err != nil {
		// Log error but don't fail validation
		fmt.Printf("Failed to update API key last used time: %v\n", err)
	}

	// Update cache
	m.cache.SetAPIKey(keyRecord)

	return keyRecord, nil
}

// RevokeAPIKey revokes an API key
func (m *Manager) RevokeAPIKey(ctx context.Context, keyID string) error {
	keyRecord, err := m.getAPIKey(ctx, keyID)
	if err != nil {
		return err
	}

	keyRecord.IsActive = false
	keyRecord.UpdatedAt = time.Now()

	if err := m.updateAPIKey(ctx, keyRecord); err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	// Remove from cache
	m.cache.DeleteAPIKey(keyID)

	return nil
}

// GetUserSessions gets all active sessions for a user
func (m *Manager) GetUserSessions(ctx context.Context, userID string) ([]*Session, error) {
	return m.getUserSessions(ctx, userID)
}

// GetUserAPIKeys gets all active API keys for a user
func (m *Manager) GetUserAPIKeys(ctx context.Context, userID string) ([]*APIKey, error) {
	query := `SELECT id, key_hash, name, user_id, tenant_id, project_id,
			  permissions, rate_limit, expires_at, last_used_at, is_active,
			  metadata, created_at, updated_at
			  FROM api_keys
			  WHERE user_id = ? AND is_active = TRUE
			  ORDER BY created_at DESC`

	rows, err := m.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query API keys: %w", err)
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		var key APIKey
		var permissionsJSON, metadataJSON sql.NullString
		var lastUsedAt, expiresAt sql.NullTime

		err := rows.Scan(
			&key.ID, &key.KeyHash, &key.Name, &key.UserID, &key.TenantID, &key.ProjectID,
			&permissionsJSON, &key.RateLimit, &expiresAt, &lastUsedAt, &key.IsActive,
			&metadataJSON, &key.CreatedAt, &key.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}

		if permissionsJSON.Valid {
			json.Unmarshal([]byte(permissionsJSON.String), &key.Permissions)
		}
		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &key.Metadata)
		}
		if lastUsedAt.Valid {
			key.LastUsedAt = &lastUsedAt.Time
		}
		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Time
		}

		keys = append(keys, &key)
	}

	return keys, nil
}

// Database helper methods

func (m *Manager) saveSession(ctx context.Context, session *Session) error {
	query := `INSERT INTO user_sessions
		(id, user_id, tenant_id, project_id, token_hash, refresh_token_hash,
		 ip_address, user_agent, expires_at, last_active_at, is_active, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	metadataJSON, _ := json.Marshal(session.Metadata)

	_, err := m.db.ExecContext(ctx, query,
		session.ID, session.UserID, session.TenantID, session.ProjectID,
		session.TokenHash, hashToken(session.RefreshToken),
		session.IPAddress, session.UserAgent, session.ExpiresAt,
		session.LastActiveAt, session.IsActive, string(metadataJSON),
		session.CreatedAt)

	return err
}

func (m *Manager) getSession(ctx context.Context, sessionID string) (*Session, error) {
	query := `SELECT id, user_id, tenant_id, project_id, token_hash, refresh_token_hash,
			  ip_address, user_agent, expires_at, last_active_at, is_active, metadata, created_at
			  FROM user_sessions WHERE id = ?`

	var session Session
	var metadataJSON sql.NullString

	err := m.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.ID, &session.UserID, &session.TenantID, &session.ProjectID,
		&session.TokenHash, &session.RefreshToken, &session.IPAddress, &session.UserAgent,
		&session.ExpiresAt, &session.LastActiveAt, &session.IsActive, &metadataJSON,
		&session.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, common.NewNotFoundError("session")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &session.Metadata)
	}

	return &session, nil
}

func (m *Manager) getSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, error) {
	query := `SELECT id, user_id, tenant_id, project_id, token_hash, refresh_token_hash,
			  ip_address, user_agent, expires_at, last_active_at, is_active, metadata, created_at
			  FROM user_sessions WHERE token_hash = ? AND is_active = TRUE`

	var session Session
	var metadataJSON sql.NullString

	err := m.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&session.ID, &session.UserID, &session.TenantID, &session.ProjectID,
		&session.TokenHash, &session.RefreshToken, &session.IPAddress, &session.UserAgent,
		&session.ExpiresAt, &session.LastActiveAt, &session.IsActive, &metadataJSON,
		&session.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, common.NewNotFoundError("session")
		}
		return nil, fmt.Errorf("failed to get session by token: %w", err)
	}

	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &session.Metadata)
	}

	return &session, nil
}

func (m *Manager) updateSession(ctx context.Context, session *Session) error {
	query := `UPDATE user_sessions SET
		last_active_at = ?, expires_at = ?, is_active = ?, metadata = ?
		WHERE id = ?`

	metadataJSON, _ := json.Marshal(session.Metadata)

	_, err := m.db.ExecContext(ctx, query,
		session.LastActiveAt, session.ExpiresAt, session.IsActive,
		string(metadataJSON), session.ID)

	return err
}

func (m *Manager) getUserSessions(ctx context.Context, userID string) ([]*Session, error) {
	query := `SELECT id, user_id, tenant_id, project_id, token_hash, refresh_token_hash,
			  ip_address, user_agent, expires_at, last_active_at, is_active, metadata, created_at
			  FROM user_sessions
			  WHERE user_id = ? AND is_active = TRUE
			  ORDER BY last_active_at DESC`

	rows, err := m.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var session Session
		var metadataJSON sql.NullString

		err := rows.Scan(
			&session.ID, &session.UserID, &session.TenantID, &session.ProjectID,
			&session.TokenHash, &session.RefreshToken, &session.IPAddress, &session.UserAgent,
			&session.ExpiresAt, &session.LastActiveAt, &session.IsActive, &metadataJSON,
			&session.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &session.Metadata)
		}

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

func (m *Manager) saveAPIKey(ctx context.Context, key *APIKey) error {
	query := `INSERT INTO api_keys
		(id, key_hash, name, user_id, tenant_id, project_id, permissions,
		 rate_limit, expires_at, last_used_at, is_active, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	permissionsJSON, _ := json.Marshal(key.Permissions)
	metadataJSON, _ := json.Marshal(key.Metadata)

	_, err := m.db.ExecContext(ctx, query,
		key.ID, key.KeyHash, key.Name, key.UserID, key.TenantID, key.ProjectID,
		string(permissionsJSON), key.RateLimit, key.ExpiresAt, key.LastUsedAt,
		key.IsActive, string(metadataJSON), key.CreatedAt, key.UpdatedAt)

	return err
}

func (m *Manager) getAPIKey(ctx context.Context, keyID string) (*APIKey, error) {
	query := `SELECT id, key_hash, name, user_id, tenant_id, project_id,
			  permissions, rate_limit, expires_at, last_used_at, is_active,
			  metadata, created_at, updated_at
			  FROM api_keys WHERE id = ?`

	var key APIKey
	var permissionsJSON, metadataJSON sql.NullString
	var lastUsedAt, expiresAt sql.NullTime

	err := m.db.QueryRowContext(ctx, query, keyID).Scan(
		&key.ID, &key.KeyHash, &key.Name, &key.UserID, &key.TenantID, &key.ProjectID,
		&permissionsJSON, &key.RateLimit, &expiresAt, &lastUsedAt, &key.IsActive,
		&metadataJSON, &key.CreatedAt, &key.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, common.NewNotFoundError("API key")
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	if permissionsJSON.Valid {
		json.Unmarshal([]byte(permissionsJSON.String), &key.Permissions)
	}
	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &key.Metadata)
	}
	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	}
	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}

	return &key, nil
}

func (m *Manager) getAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	query := `SELECT id, key_hash, name, user_id, tenant_id, project_id,
			  permissions, rate_limit, expires_at, last_used_at, is_active,
			  metadata, created_at, updated_at
			  FROM api_keys WHERE key_hash = ? AND is_active = TRUE`

	var key APIKey
	var permissionsJSON, metadataJSON sql.NullString
	var lastUsedAt, expiresAt sql.NullTime

	err := m.db.QueryRowContext(ctx, query, keyHash).Scan(
		&key.ID, &key.KeyHash, &key.Name, &key.UserID, &key.TenantID, &key.ProjectID,
		&permissionsJSON, &key.RateLimit, &expiresAt, &lastUsedAt, &key.IsActive,
		&metadataJSON, &key.CreatedAt, &key.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, common.NewNotFoundError("API key")
		}
		return nil, fmt.Errorf("failed to get API key by hash: %w", err)
	}

	if permissionsJSON.Valid {
		json.Unmarshal([]byte(permissionsJSON.String), &key.Permissions)
	}
	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &key.Metadata)
	}
	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	}
	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}

	return &key, nil
}

func (m *Manager) updateAPIKey(ctx context.Context, key *APIKey) error {
	query := `UPDATE api_keys SET
		last_used_at = ?, expires_at = ?, is_active = ?, metadata = ?, updated_at = ?
		WHERE id = ?`

	metadataJSON, _ := json.Marshal(key.Metadata)

	_, err := m.db.ExecContext(ctx, query,
		key.LastUsedAt, key.ExpiresAt, key.IsActive,
		string(metadataJSON), key.UpdatedAt, key.ID)

	return err
}

// Utility methods

func (m *Manager) checkSessionLimit(ctx context.Context, userID string) error {
	sessions, err := m.getUserSessions(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to check session limit: %w", err)
	}

	if len(sessions) >= m.config.MaxSessionsPerUser {
		// Revoke oldest session
		oldestSession := sessions[len(sessions)-1]
		if err := m.InvalidateSession(ctx, oldestSession.ID); err != nil {
			return fmt.Errorf("failed to revoke oldest session: %w", err)
		}
	}

	return nil
}

func (m *Manager) startCleanupRoutine() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanupExpiredSessions()
		}
	}
}

func (m *Manager) cleanupExpiredSessions() {
	ctx := context.Background()

	// Clean expired sessions
	query := `UPDATE user_sessions SET is_active = FALSE WHERE expires_at < ? AND is_active = TRUE`
	result, err := m.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		fmt.Printf("Failed to cleanup expired sessions: %v\n", err)
	} else {
		if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
			fmt.Printf("Cleaned up %d expired sessions\n", rowsAffected)
		}
	}

	// Clean expired API keys
	query = `UPDATE api_keys SET is_active = FALSE WHERE expires_at < ? AND is_active = TRUE`
	result, err = m.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		fmt.Printf("Failed to cleanup expired API keys: %v\n", err)
	} else {
		if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
			fmt.Printf("Cleaned up %d expired API keys\n", rowsAffected)
		}
	}

	// Clean cache
	m.cache.Cleanup()
}

// Cache methods

func (sc *SessionCache) GetSession(sessionID string) *Session {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if session, exists := sc.sessions[sessionID]; exists {
		if time.Now().After(session.ExpiresAt) {
			delete(sc.sessions, sessionID)
			return nil
		}
		return session
	}
	return nil
}

func (sc *SessionCache) GetSessionByToken(token string) *Session {
	tokenHash := hashToken(token)
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	for _, session := range sc.sessions {
		if session.TokenHash == tokenHash && time.Now().Before(session.ExpiresAt) {
			return session
		}
	}
	return nil
}

func (sc *SessionCache) GetAPIKeyByKey(apiKey string) *APIKey {
	keyHash := hashToken(apiKey)
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	for _, key := range sc.apiKeys {
		if key.KeyHash == keyHash && key.IsActive &&
			(key.ExpiresAt == nil || time.Now().Before(*key.ExpiresAt)) {
			return key
		}
	}
	return nil
}

func (sc *SessionCache) SetSession(session *Session) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Remove oldest entries if cache is full
	if len(sc.sessions) >= sc.maxSize {
		sc.evictOldestSession()
	}

	sc.sessions[session.ID] = session
}

func (sc *SessionCache) SetAPIKey(key *APIKey) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Remove oldest entries if cache is full
	if len(sc.apiKeys) >= sc.maxSize {
		sc.evictOldestAPIKey()
	}

	sc.apiKeys[key.ID] = key
}

func (sc *SessionCache) DeleteSession(sessionID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.sessions, sessionID)
}

func (sc *SessionCache) DeleteAPIKey(keyID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.apiKeys, keyID)
}

func (sc *SessionCache) Cleanup() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Clean expired sessions
	now := time.Now()
	for id, session := range sc.sessions {
		if now.After(session.ExpiresAt) {
			delete(sc.sessions, id)
		}
	}

	// Clean expired API keys
	for id, key := range sc.apiKeys {
		if key.ExpiresAt != nil && now.After(*key.ExpiresAt) {
			delete(sc.apiKeys, id)
		}
	}
}

func (sc *SessionCache) evictOldestSession() {
	var oldestID string
	var oldestTime time.Time = time.Now()

	for id, session := range sc.sessions {
		if session.CreatedAt.Before(oldestTime) {
			oldestTime = session.CreatedAt
			oldestID = id
		}
	}

	if oldestID != "" {
		delete(sc.sessions, oldestID)
	}
}

func (sc *SessionCache) evictOldestAPIKey() {
	var oldestID string
	var oldestTime time.Time = time.Now()

	for id, key := range sc.apiKeys {
		if key.CreatedAt.Before(oldestTime) {
			oldestTime = key.CreatedAt
			oldestID = id
		}
	}

	if oldestID != "" {
		delete(sc.apiKeys, oldestID)
	}
}

// Helper functions

func generateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("sess_%s", base64.URLEncoding.EncodeToString(bytes))
}

func generateAPIKeyID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("key_%s", base64.URLEncoding.EncodeToString(bytes))
}

func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func hashToken(token string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	return string(hash)
}

// verifyHash checks if a token matches a hash (for validation)
func verifyHash(token, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(token))
	return err == nil
}