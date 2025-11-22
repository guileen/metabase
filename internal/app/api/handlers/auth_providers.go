package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/guileen/metabase/internal/app/api/rest"
	"github.com/guileen/metabase/internal/biz/domain/authgateway"
	"github.com/guileen/metabase/internal/biz/domain/authgateway/providers"
	// "github.com/guileen/metabase/pkg/common/rest" // TODO: Fix this import
)

// AuthProvidersHandler handles authentication provider management
type AuthProvidersHandler struct {
	db          *sql.DB
	authGateway *authgateway.AuthGatewayManager
}

// AuthProviderConfigRequest represents a request to create/update auth provider
type AuthProviderConfigRequest struct {
	Name         string                 `json:"name"`
	DisplayName  string                 `json:"display_name"`
	Type         string                 `json:"type"` // oauth2, oidc, saml, ldap, local
	Enabled      bool                   `json:"enabled"`
	ClientID     string                 `json:"client_id,omitempty"`
	ClientSecret string                 `json:"client_secret,omitempty"`
	RedirectURL  string                 `json:"redirect_url,omitempty"`
	AuthURL      string                 `json:"auth_url,omitempty"`
	TokenURL     string                 `json:"token_url,omitempty"`
	UserInfoURL  string                 `json:"user_info_url,omitempty"`
	Scopes       []string               `json:"scopes,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Features     []string               `json:"features,omitempty"`
}

// AuthProviderResponse represents auth provider configuration response
type AuthProviderResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	DisplayName string                 `json:"display_name"`
	Type        string                 `json:"type"`
	Enabled     bool                   `json:"enabled"`
	ClientID    string                 `json:"client_id,omitempty"`
	AuthURL     string                 `json:"auth_url,omitempty"`
	TokenURL    string                 `json:"token_url,omitempty"`
	UserInfoURL string                 `json:"user_info_url,omitempty"`
	Scopes      []string               `json:"scopes,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Features    []string               `json:"features,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	// ClientSecret is omitted for security
}

// NewAuthProvidersHandler creates a new auth providers handler
func NewAuthProvidersHandler(db *sql.DB, authGateway *authgateway.AuthGatewayManager) *AuthProvidersHandler {
	return &AuthProvidersHandler{
		db:          db,
		authGateway: authGateway,
	}
}

// GetProviders returns a list of authentication providers
func (h *AuthProvidersHandler) GetProviders(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id, name, display_name, type, enabled, client_id, auth_url, token_url,
		user_info_url, scopes, config, features, created_at, updated_at
		FROM auth_providers ORDER BY name`

	rows, err := h.db.Query(query)
	if err != nil {
		rest.WriteError(w, http.StatusInternalServerError, "Failed to query providers", err)
		return
	}
	defer rows.Close()

	var providers []AuthProviderResponse
	for rows.Next() {
		var provider AuthProviderResponse
		var scopesJSON, configJSON, featuresJSON sql.NullString

		err := rows.Scan(
			&provider.ID, &provider.Name, &provider.DisplayName, &provider.Type,
			&provider.Enabled, &provider.ClientID, &provider.AuthURL, &provider.TokenURL,
			&provider.UserInfoURL, &scopesJSON, &configJSON, &featuresJSON,
			&provider.CreatedAt, &provider.UpdatedAt,
		)
		if err != nil {
			rest.WriteError(w, http.StatusInternalServerError, "Failed to scan provider", err)
			return
		}

		if scopesJSON.Valid {
			json.Unmarshal([]byte(scopesJSON.String), &provider.Scopes)
		}
		if configJSON.Valid {
			json.Unmarshal([]byte(configJSON.String), &provider.Config)
		}
		if featuresJSON.Valid {
			json.Unmarshal([]byte(featuresJSON.String), &provider.Features)
		}

		providers = append(providers, provider)
	}

	if err = rows.Err(); err != nil {
		rest.WriteError(w, http.StatusInternalServerError, "Error iterating providers", err)
		return
	}

	rest.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"providers": providers,
		"total":     len(providers),
	})
}

// GetProvider returns a specific authentication provider
func (h *AuthProvidersHandler) GetProvider(w http.ResponseWriter, r *http.Request) {
	providerID := r.PathValue("id")
	if providerID == "" {
		rest.WriteError(w, http.StatusBadRequest, "Provider ID is required", nil)
		return
	}

	query := `SELECT id, name, display_name, type, enabled, client_id, auth_url, token_url,
		user_info_url, scopes, config, features, created_at, updated_at
		FROM auth_providers WHERE id = ?`

	var provider AuthProviderResponse
	var scopesJSON, configJSON, featuresJSON sql.NullString

	err := h.db.QueryRow(query, providerID).Scan(
		&provider.ID, &provider.Name, &provider.DisplayName, &provider.Type,
		&provider.Enabled, &provider.ClientID, &provider.AuthURL, &provider.TokenURL,
		&provider.UserInfoURL, &scopesJSON, &configJSON, &featuresJSON,
		&provider.CreatedAt, &provider.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			rest.WriteError(w, http.StatusNotFound, "Provider not found", nil)
		} else {
			rest.WriteError(w, http.StatusInternalServerError, "Failed to get provider", err)
		}
		return
	}

	if scopesJSON.Valid {
		json.Unmarshal([]byte(scopesJSON.String), &provider.Scopes)
	}
	if configJSON.Valid {
		json.Unmarshal([]byte(configJSON.String), &provider.Config)
	}
	if featuresJSON.Valid {
		json.Unmarshal([]byte(featuresJSON.String), &provider.Features)
	}

	rest.WriteJSON(w, http.StatusOK, provider)
}

// CreateProvider creates a new authentication provider
func (h *AuthProvidersHandler) CreateProvider(w http.ResponseWriter, r *http.Request) {
	var req AuthProviderConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if err := h.validateProviderConfig(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Check if provider name already exists
	var exists bool
	err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM auth_providers WHERE name = ?)", req.Name).Scan(&exists)
	if err != nil {
		rest.WriteError(w, http.StatusInternalServerError, "Failed to check provider existence", err)
		return
	}
	if exists {
		rest.WriteError(w, http.StatusConflict, "Provider with this name already exists", nil)
		return
	}

	// Generate ID
	providerID := fmt.Sprintf("provider_%s_%d", req.Name, time.Now().UnixNano())

	// Prepare JSON fields
	scopesJSON, _ := json.Marshal(req.Scopes)
	configJSON, _ := json.Marshal(req.Config)
	featuresJSON, _ := json.Marshal(req.Features)

	query := `INSERT INTO auth_providers
		(id, name, display_name, type, enabled, client_id, client_secret, auth_url,
		token_url, user_info_url, scopes, config, features, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	_, err = h.db.Exec(query,
		providerID, req.Name, req.DisplayName, req.Type, req.Enabled,
		req.ClientID, req.ClientSecret, req.AuthURL, req.TokenURL, req.UserInfoURL,
		string(scopesJSON), string(configJSON), string(featuresJSON), now, now,
	)

	if err != nil {
		rest.WriteError(w, http.StatusInternalServerError, "Failed to create provider", err)
		return
	}

	// Return created provider
	h.GetProvider(w, r.WithContext(r.Context()))
}

// UpdateProvider updates an authentication provider
func (h *AuthProvidersHandler) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	providerID := r.PathValue("id")
	if providerID == "" {
		rest.WriteError(w, http.StatusBadRequest, "Provider ID is required", nil)
		return
	}

	var req AuthProviderConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if err := h.validateProviderConfig(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Prepare JSON fields
	scopesJSON, _ := json.Marshal(req.Scopes)
	configJSON, _ := json.Marshal(req.Config)
	featuresJSON, _ := json.Marshal(req.Features)

	query := `UPDATE auth_providers SET
		name = ?, display_name = ?, type = ?, enabled = ?, client_id = ?,
		client_secret = COALESCE(?, client_secret), auth_url = ?, token_url = ?,
		user_info_url = ?, scopes = ?, config = ?, features = ?, updated_at = ?
		WHERE id = ?`

	now := time.Now()
	result, err := h.db.Exec(query,
		req.Name, req.DisplayName, req.Type, req.Enabled, req.ClientID,
		req.ClientSecret, req.AuthURL, req.TokenURL, req.UserInfoURL,
		string(scopesJSON), string(configJSON), string(featuresJSON), now, providerID,
	)

	if err != nil {
		rest.WriteError(w, http.StatusInternalServerError, "Failed to update provider", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		rest.WriteError(w, http.StatusNotFound, "Provider not found", nil)
		return
	}

	// Return updated provider
	h.GetProvider(w, r.WithContext(r.Context()))
}

// DeleteProvider deletes an authentication provider
func (h *AuthProvidersHandler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	providerID := r.PathValue("id")
	if providerID == "" {
		rest.WriteError(w, http.StatusBadRequest, "Provider ID is required", nil)
		return
	}

	// Check if provider exists and is not 'local'
	var name, providerType string
	err := h.db.QueryRow("SELECT name, type FROM auth_providers WHERE id = ?", providerID).Scan(&name, &providerType)
	if err != nil {
		if err == sql.ErrNoRows {
			rest.WriteError(w, http.StatusNotFound, "Provider not found", nil)
		} else {
			rest.WriteError(w, http.StatusInternalServerError, "Failed to check provider", err)
		}
		return
	}

	// Don't allow deletion of local provider
	if name == "local" || providerType == "local" {
		rest.WriteError(w, http.StatusBadRequest, "Cannot delete local authentication provider", nil)
		return
	}

	// Delete provider
	result, err := h.db.Exec("DELETE FROM auth_providers WHERE id = ?", providerID)
	if err != nil {
		rest.WriteError(w, http.StatusInternalServerError, "Failed to delete provider", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		rest.WriteError(w, http.StatusNotFound, "Provider not found", nil)
		return
	}

	rest.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Provider deleted successfully",
	})
}

// TestProvider tests an authentication provider configuration
func (h *AuthProvidersHandler) TestProvider(w http.ResponseWriter, r *http.Request) {
	var req AuthProviderConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if err := h.validateProviderConfig(&req); err != nil {
		rest.WriteError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Create temporary provider instance for testing
	var provider authgateway.AuthProvider

	switch req.Type {
	case "oauth2":
		// Create generic OAuth2 provider for testing
		oauth2Config := &providers.OAuth2Config{
			Name:         req.Name,
			DisplayName:  req.DisplayName,
			ClientID:     req.ClientID,
			ClientSecret: req.ClientSecret,
			AuthURL:      req.AuthURL,
			TokenURL:     req.TokenURL,
			UserInfoURL:  req.UserInfoURL,
			Scopes:       req.Scopes,
			Enabled:      true,
			Config:       req.Config,
		}
		provider = providers.NewOAuth2Provider(h.db, oauth2Config)

	case "local":
		// Test local provider configuration
		localConfig := &providers.LocalAuthConfig{
			Enabled:           req.Enabled,
			MinPasswordLength: 8,
		}
		provider = providers.NewLocalAuthProvider(h.db, localConfig)

	default:
		rest.WriteError(w, http.StatusBadRequest, "Unsupported provider type for testing", nil)
		return
	}

	// Test provider configuration
	err := provider.ValidateConfig()
	if err != nil {
		rest.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"valid":  false,
			"error":  err.Error(),
			"result": "Configuration test failed",
		})
		return
	}

	rest.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"valid":  true,
		"result": "Configuration test passed",
	})
}

// GetProviderStats returns authentication provider statistics
func (h *AuthProvidersHandler) GetProviderStats(w http.ResponseWriter, r *http.Request) {
	query := `SELECT provider, COUNT(*) as user_count
		FROM auth_users WHERE provider IS NOT NULL AND provider != 'local'
		GROUP BY provider ORDER BY user_count DESC`

	rows, err := h.db.Query(query)
	if err != nil {
		rest.WriteError(w, http.StatusInternalServerError, "Failed to query provider stats", err)
		return
	}
	defer rows.Close()

	var stats []map[string]interface{}
	for rows.Next() {
		var provider string
		var userCount int
		err := rows.Scan(&provider, &userCount)
		if err != nil {
			rest.WriteError(w, http.StatusInternalServerError, "Failed to scan stats", err)
			return
		}

		stats = append(stats, map[string]interface{}{
			"provider":   provider,
			"user_count": userCount,
		})
	}

	// Get overall auth stats
	var totalUsers, oauthUsers int
	h.db.QueryRow("SELECT COUNT(*) FROM auth_users").Scan(&totalUsers)
	h.db.QueryRow("SELECT COUNT(*) FROM auth_users WHERE provider != 'local'").Scan(&oauthUsers)

	rest.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"total_users":    totalUsers,
		"oauth_users":    oauthUsers,
		"local_users":    totalUsers - oauthUsers,
		"provider_stats": stats,
		"last_updated":   time.Now(),
	})
}

// validateProviderConfig validates provider configuration
func (h *AuthProvidersHandler) validateProviderConfig(req *AuthProviderConfigRequest) error {
	if req.Name == "" {
		return fmt.Errorf("provider name is required")
	}
	if req.DisplayName == "" {
		return fmt.Errorf("provider display name is required")
	}
	if req.Type == "" {
		return fmt.Errorf("provider type is required")
	}

	// Validate based on type
	switch req.Type {
	case "oauth2":
		if req.ClientID == "" {
			return fmt.Errorf("client_id is required for OAuth2 providers")
		}
		if req.ClientSecret == "" {
			return fmt.Errorf("client_secret is required for OAuth2 providers")
		}
		if req.AuthURL == "" {
			return fmt.Errorf("auth_url is required for OAuth2 providers")
		}
		if req.TokenURL == "" {
			return fmt.Errorf("token_url is required for OAuth2 providers")
		}
		if req.UserInfoURL == "" {
			return fmt.Errorf("user_info_url is required for OAuth2 providers")
		}
		if len(req.Scopes) == 0 {
			return fmt.Errorf("at least one scope is required for OAuth2 providers")
		}
	case "local":
		// Local provider specific validation
		if len(req.Features) == 0 {
			req.Features = []string{"login", "register"}
		}
	default:
		return fmt.Errorf("unsupported provider type: %s", req.Type)
	}

	return nil
}

// Helper function to parse pagination parameters
func parsePaginationParams(r *http.Request) (page, limit int, err error) {
	page = 1
	limit = 20

	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	return page, limit, nil
}
