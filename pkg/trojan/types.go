package trojan

import (
	"crypto/tls"
	"time"
)

// TrojanConfig represents Trojan proxy configuration
type TrojanConfig struct {
	// Server configuration
	Enabled    bool   `json:"enabled"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Password   string `json:"password"`
	ServerName string `json:"server_name"`
	CertPath   string `json:"cert_path"`
	KeyPath    string `json:"key_path"`
	AutoCert   bool   `json:"auto_cert"`

	// Client management
	MaxClients    int           `json:"max_clients"`
	ClientTimeout time.Duration `json:"client_timeout"`

	// Logging
	LogLevel   string `json:"log_level"`
	LogPath    string `json:"log_path"`
	EnableAuth bool   `json:"enable_auth"`

	// Performance
	BufferSize    int `json:"buffer_size"`
	MaxPacketSize int `json:"max_packet_size"`

	// Security features
	EnableTLS    bool     `json:"enable_tls"`
	ALPN         []string `json:"alpn"`
	CipherSuites []uint16 `json:"cipher_suites"`
}

// ClientInfo represents Trojan client information
type ClientInfo struct {
	ID          string     `json:"id"`
	Password    string     `json:"password"`
	Name        string     `json:"name"`
	Status      string     `json:"status"` // "active", "disabled", "expired"
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	LastSeen    *time.Time `json:"last_seen"`
	DataLimit   int64      `json:"data_limit"` // bytes
	DataUsed    int64      `json:"data_used"`  // bytes
	IPWhitelist []string   `json:"ip_whitelist"`
	Tags        []string   `json:"tags"`
}

// ClientStats represents client statistics
type ClientStats struct {
	ClientID      string    `json:"client_id"`
	UploadBytes   int64     `json:"upload_bytes"`
	DownloadBytes int64     `json:"download_bytes"`
	TotalBytes    int64     `json:"total_bytes"`
	Connections   int       `json:"connections"`
	LastActivity  time.Time `json:"last_activity"`
}

// ServerStats represents server statistics
type ServerStats struct {
	ActiveConnections int           `json:"active_connections"`
	TotalConnections  int           `json:"total_connections"`
	UploadBytes       int64         `json:"upload_bytes"`
	DownloadBytes     int64         `json:"download_bytes"`
	TotalBytes        int64         `json:"total_bytes"`
	Uptime            time.Duration `json:"uptime"`
	CPUUsage          float64       `json:"cpu_usage"`
	MemoryUsage       int64         `json:"memory_usage"`
}

// Connection represents an active Trojan connection
type Connection struct {
	ID         string    `json:"id"`
	ClientID   string    `json:"client_id"`
	RemoteAddr string    `json:"remote_addr"`
	TargetAddr string    `json:"target_addr"`
	CreatedAt  time.Time `json:"created_at"`
	BytesSent  int64     `json:"bytes_sent"`
	BytesRecv  int64     `json:"bytes_recv"`
	Status     string    `json:"status"` // "active", "closing", "closed"
}

// DefaultConfig returns a default Trojan configuration
func DefaultConfig() *TrojanConfig {
	return &TrojanConfig{
		Enabled:       false,
		Host:          "0.0.0.0",
		Port:          8443,
		Password:      "default-password-change-me",
		ServerName:    "localhost",
		AutoCert:      true,
		MaxClients:    1000,
		ClientTimeout: 30 * time.Minute,
		LogLevel:      "info",
		LogPath:       "./logs/trojan.log",
		EnableAuth:    true,
		BufferSize:    32 * 1024,
		MaxPacketSize: 16384,
		EnableTLS:     true,
		ALPN:          []string{"h2", "http/1.1"},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
	}
}

// Validate validates the Trojan configuration
func (c *TrojanConfig) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return &TrojanError{Type: "invalid_port", Message: "port must be between 1 and 65535"}
	}

	if c.Password == "" {
		return &TrojanError{Type: "empty_password", Message: "password cannot be empty"}
	}

	if c.MaxClients <= 0 {
		return &TrojanError{Type: "invalid_max_clients", Message: "max_clients must be positive"}
	}

	if c.BufferSize <= 0 {
		return &TrojanError{Type: "invalid_buffer_size", Message: "buffer_size must be positive"}
	}

	if c.MaxPacketSize <= 0 {
		c.MaxPacketSize = 8 * 1024 // Set default to 8KB
	}

	return nil
}

// Clone creates a copy of the configuration
func (c *TrojanConfig) Clone() *TrojanConfig {
	clone := *c
	if c.ALPN != nil {
		clone.ALPN = make([]string, len(c.ALPN))
		copy(clone.ALPN, c.ALPN)
	}
	if c.CipherSuites != nil {
		clone.CipherSuites = make([]uint16, len(c.CipherSuites))
		copy(clone.CipherSuites, c.CipherSuites)
	}
	return &clone
}
