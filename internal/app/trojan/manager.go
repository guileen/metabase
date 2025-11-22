package trojan

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/guileen/metabase/pkg/trojan"
	"go.uber.org/zap"
)

// Manager manages Trojan VPN service
type Manager struct {
	server *trojan.Server
	config *trojan.TrojanConfig
	db     *sql.DB
	logger *zap.Logger
	mutex  sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewManager creates a new Trojan manager
func NewManager(db *sql.DB, logger *zap.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		config: trojan.DefaultConfig(),
		db:     db,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Initialize initializes the Trojan manager
func (m *Manager) Initialize() error {
	// Load configuration from database
	if err := m.loadConfig(); err != nil {
		m.logger.Error("Failed to load Trojan config", zap.Error(err))
		// Use default config if load fails
	}

	// Initialize database tables
	if err := m.initTables(); err != nil {
		return fmt.Errorf("failed to initialize database tables: %w", err)
	}

	// Load clients from database
	if err := m.loadClients(); err != nil {
		m.logger.Error("Failed to load Trojan clients", zap.Error(err))
	}

	return nil
}

// Start starts the Trojan service
func (m *Manager) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.config.Enabled {
		m.logger.Info("Trojan service is disabled")
		return nil
	}

	if m.server != nil && m.server.IsRunning() {
		return trojan.ErrServerAlreadyRunning
	}

	server, err := trojan.NewServer(m.config)
	if err != nil {
		return fmt.Errorf("failed to create Trojan server: %w", err)
	}

	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start Trojan server: %w", err)
	}

	m.server = server
	m.logger.Info("Trojan service started",
		zap.String("host", m.config.Host),
		zap.Int("port", m.config.Port),
		zap.Bool("tls", m.config.EnableTLS))

	return nil
}

// Stop stops the Trojan service
func (m *Manager) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.server == nil {
		return trojan.ErrServerNotStarted
	}

	if err := m.server.Stop(); err != nil {
		return fmt.Errorf("failed to stop Trojan server: %w", err)
	}

	m.server = nil
	m.logger.Info("Trojan service stopped")

	return nil
}

// Restart restarts the Trojan service
func (m *Manager) Restart() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	wasRunning := m.server != nil && m.server.IsRunning()

	if wasRunning {
		if err := m.server.Stop(); err != nil {
			return fmt.Errorf("failed to stop Trojan server: %w", err)
		}
	}

	if !m.config.Enabled {
		m.logger.Info("Trojan service is disabled, not starting")
		return nil
	}

	server, err := trojan.NewServer(m.config)
	if err != nil {
		return fmt.Errorf("failed to create Trojan server: %w", err)
	}

	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start Trojan server: %w", err)
	}

	m.server = server
	m.logger.Info("Trojan service restarted")

	return nil
}

// GetStatus returns the current status of the Trojan service
func (m *Manager) GetStatus() *Status {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	status := &Status{
		Enabled: m.config.Enabled,
		Running: m.server != nil && m.server.IsRunning(),
		Config:  m.config.Clone(),
	}

	if status.Running && m.server != nil {
		status.Stats = m.server.GetStats()
		status.Connections = m.server.GetConnections()
		status.CertificateInfo = m.server.GetAuth()
	}

	return status
}

// UpdateConfig updates the Trojan configuration
func (m *Manager) UpdateConfig(newConfig *trojan.TrojanConfig) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Save configuration to database
	if err := m.saveConfig(newConfig); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	wasRunning := m.server != nil && m.server.IsRunning()

	// Stop current server if running
	if wasRunning {
		if err := m.server.Stop(); err != nil {
			m.logger.Error("Failed to stop server for config update", zap.Error(err))
		}
	}

	m.config = newConfig.Clone()

	// Start server if enabled and was running
	if m.config.Enabled && wasRunning {
		server, err := trojan.NewServer(m.config)
		if err != nil {
			return fmt.Errorf("failed to create new server: %w", err)
		}

		if err := server.Start(); err != nil {
			return fmt.Errorf("failed to start new server: %w", err)
		}

		m.server = server
		m.logger.Info("Trojan configuration updated and service restarted")
	} else {
		m.logger.Info("Trojan configuration updated")
	}

	return nil
}

// AddClient adds a new Trojan client
func (m *Manager) AddClient(client *trojan.ClientInfo) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.server == nil {
		return trojan.ErrServerNotStarted
	}

	auth := m.server.GetAuth()
	if err := auth.AddClient(client); err != nil {
		return fmt.Errorf("failed to add client: %w", err)
	}

	// Save client to database
	if err := m.saveClient(client); err != nil {
		// Rollback
		auth.RemoveClient(client.ID)
		return fmt.Errorf("failed to save client: %w", err)
	}

	m.logger.Info("Trojan client added", zap.String("id", client.ID), zap.String("name", client.Name))
	return nil
}

// RemoveClient removes a Trojan client
func (m *Manager) RemoveClient(clientID string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.server == nil {
		return trojan.ErrServerNotStarted
	}

	auth := m.server.GetAuth()
	if err := auth.RemoveClient(clientID); err != nil {
		return fmt.Errorf("failed to remove client: %w", err)
	}

	// Remove client from database
	if err := m.deleteClient(clientID); err != nil {
		m.logger.Error("Failed to delete client from database", zap.String("id", clientID), zap.Error(err))
	}

	m.logger.Info("Trojan client removed", zap.String("id", clientID))
	return nil
}

// ListClients returns all Trojan clients
func (m *Manager) ListClients() ([]*trojan.ClientInfo, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.server == nil {
		return []*trojan.ClientInfo{}, nil
	}

	auth := m.server.GetAuth()
	return auth.ListClients(), nil
}

// GetClient returns a specific Trojan client
func (m *Manager) GetClient(clientID string) (*trojan.ClientInfo, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.server == nil {
		return nil, trojan.ErrServerNotStarted
	}

	auth := m.server.GetAuth()
	return auth.GetClient(clientID)
}

// GetClientStats returns statistics for all clients
func (m *Manager) GetClientStats() ([]*trojan.ClientStats, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.server == nil {
		return []*trojan.ClientStats{}, nil
	}

	auth := m.server.GetAuth()
	return auth.GetClientStats(), nil
}

// GenerateClientConfig generates a client configuration file
func (m *Manager) GenerateClientConfig(clientID string) (map[string]interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.server == nil {
		return nil, trojan.ErrServerNotStarted
	}

	client, err := m.server.GetAuth().GetClient(clientID)
	if err != nil {
		return nil, fmt.Errorf("client not found: %w", err)
	}

	config := map[string]interface{}{
		"run_type":    "client",
		"local_addr":  "127.0.0.1",
		"local_port":  1080,
		"remote_addr": m.config.ServerName,
		"remote_port": m.config.Port,
		"password":    []string{client.Password},
		"log_level":   m.config.LogLevel,
		"ssl": map[string]interface{}{
			"verify":          true,
			"verify_hostname": true,
			"sni":             m.config.ServerName,
			"fingerprint":     "chrome",
			"cipher":          m.config.CipherSuites,
			"cipher_tls13":    "TLS_AES_128_GCM_SHA256:TLS_CHACHA20_POLY1305_SHA256:TLS_AES_256_GCM_SHA384",
			"curves":          "",
		},
		"mux": map[string]interface{}{
			"enabled":      true,
			"concurrency":  -1,
			"idle_timeout": 60,
		},
		"router": map[string]interface{}{
			"enabled": false,
		},
		"websocket": map[string]interface{}{
			"enabled": false,
		},
	}

	return config, nil
}

// Cleanup cleans up the Trojan manager
func (m *Manager) Cleanup() error {
	m.cancel()

	if m.server != nil {
		if err := m.server.Stop(); err != nil {
			m.logger.Error("Failed to stop Trojan server during cleanup", zap.Error(err))
		}
	}

	return nil
}

// Database operations

func (m *Manager) initTables() error {
	// Create trojan_config table
	configTable := `
	CREATE TABLE IF NOT EXISTS trojan_config (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT UNIQUE NOT NULL,
		value TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := m.db.Exec(configTable); err != nil {
		return fmt.Errorf("failed to create trojan_config table: %w", err)
	}

	// Create trojan_clients table
	clientsTable := `
	CREATE TABLE IF NOT EXISTS trojan_clients (
		id TEXT PRIMARY KEY,
		password TEXT NOT NULL,
		name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME,
		last_seen DATETIME,
		data_limit INTEGER DEFAULT 0,
		data_used INTEGER DEFAULT 0,
		ip_whitelist TEXT, -- JSON array
		tags TEXT -- JSON array
	)`

	if _, err := m.db.Exec(clientsTable); err != nil {
		return fmt.Errorf("failed to create trojan_clients table: %w", err)
	}

	return nil
}

func (m *Manager) loadConfig() error {
	query := "SELECT key, value FROM trojan_config"
	rows, err := m.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query config: %w", err)
	}
	defer rows.Close()

	configMap := make(map[string]interface{})
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return fmt.Errorf("failed to scan config row: %w", err)
		}

		// Try to parse as JSON first, then as string
		var jsonValue interface{}
		if err := json.Unmarshal([]byte(value), &jsonValue); err != nil {
			configMap[key] = value
		} else {
			configMap[key] = jsonValue
		}
	}

	// Convert config map to TrojanConfig
	if len(configMap) > 0 {
		configJSON, err := json.Marshal(configMap)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		var config trojan.TrojanConfig
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}

		m.config = &config
	}

	return nil
}

func (m *Manager) saveConfig(config *trojan.TrojanConfig) error {
	// Clear existing config
	if _, err := m.db.Exec("DELETE FROM trojan_config"); err != nil {
		return fmt.Errorf("failed to clear existing config: %w", err)
	}

	// Save new config
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(configJSON, &configMap); err != nil {
		return fmt.Errorf("failed to unmarshal config for saving: %w", err)
	}

	for key, value := range configMap {
		valueJSON, err := json.Marshal(value)
		if err != nil {
			continue // Skip problematic values
		}

		query := "INSERT INTO trojan_config (key, value) VALUES (?, ?)"
		if _, err := m.db.Exec(query, key, string(valueJSON)); err != nil {
			m.logger.Error("Failed to save config key", zap.String("key", key), zap.Error(err))
		}
	}

	return nil
}

func (m *Manager) loadClients() error {
	if m.server == nil {
		return nil // Server not started yet
	}

	query := `
	SELECT id, password, name, status, created_at, expires_at, last_seen,
	       data_limit, data_used, ip_whitelist, tags
	FROM trojan_clients`

	rows, err := m.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query clients: %w", err)
	}
	defer rows.Close()

	auth := m.server.GetAuth()

	for rows.Next() {
		client := &trojan.ClientInfo{}
		var ipWhitelistJSON, tagsJSON sql.NullString
		var createdStr string

		err := rows.Scan(
			&client.ID,
			&client.Password,
			&client.Name,
			&client.Status,
			&createdStr,
			&client.ExpiresAt,
			&client.LastSeen,
			&client.DataLimit,
			&client.DataUsed,
			&ipWhitelistJSON,
			&tagsJSON,
		)
		if err != nil {
			m.logger.Error("Failed to scan client row", zap.Error(err))
			continue
		}

		// Parse created_at
		if createdAt, err := time.Parse(time.RFC3339, createdStr); err == nil {
			client.CreatedAt = createdAt
		}

		// Parse IP whitelist
		if ipWhitelistJSON.Valid && ipWhitelistJSON.String != "" {
			var ipWhitelist []string
			if err := json.Unmarshal([]byte(ipWhitelistJSON.String), &ipWhitelist); err == nil {
				client.IPWhitelist = ipWhitelist
			}
		}

		// Parse tags
		if tagsJSON.Valid && tagsJSON.String != "" {
			var tags []string
			if err := json.Unmarshal([]byte(tagsJSON.String), &tags); err == nil {
				client.Tags = tags
			}
		}

		if err := auth.AddClient(client); err != nil {
			m.logger.Error("Failed to add loaded client to auth", zap.String("id", client.ID), zap.Error(err))
		}
	}

	return nil
}

func (m *Manager) saveClient(client *trojan.ClientInfo) error {
	ipWhitelistJSON, _ := json.Marshal(client.IPWhitelist)
	tagsJSON, _ := json.Marshal(client.Tags)

	query := `
	INSERT OR REPLACE INTO trojan_clients
	(id, password, name, status, created_at, expires_at, last_seen,
	 data_limit, data_used, ip_whitelist, tags)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := m.db.Exec(query,
		client.ID,
		client.Password,
		client.Name,
		client.Status,
		client.CreatedAt.Format(time.RFC3339),
		client.ExpiresAt,
		client.LastSeen,
		client.DataLimit,
		client.DataUsed,
		string(ipWhitelistJSON),
		string(tagsJSON),
	)

	return err
}

func (m *Manager) deleteClient(clientID string) error {
	query := "DELETE FROM trojan_clients WHERE id = ?"
	_, err := m.db.Exec(query, clientID)
	return err
}

// Status represents the Trojan service status
type Status struct {
	Enabled         bool                 `json:"enabled"`
	Running         bool                 `json:"running"`
	Config          *trojan.TrojanConfig `json:"config"`
	Stats           *trojan.ServerStats  `json:"stats,omitempty"`
	Connections     []*trojan.Connection `json:"connections,omitempty"`
	CertificateInfo interface{}          `json:"certificate_info,omitempty"`
}
