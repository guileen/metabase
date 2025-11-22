package trojan

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"strings"
	"time"
)

// Authenticator handles Trojan client authentication
type Authenticator struct {
	clients map[string]*ClientInfo
	config  *TrojanConfig
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator(config *TrojanConfig) *Authenticator {
	return &Authenticator{
		clients: make(map[string]*ClientInfo),
		config:  config,
	}
}

// AddClient adds a new client
func (a *Authenticator) AddClient(client *ClientInfo) error {
	if _, exists := a.clients[client.ID]; exists {
		return ErrClientExists
	}

	a.clients[client.ID] = client
	return nil
}

// RemoveClient removes a client
func (a *Authenticator) RemoveClient(clientID string) error {
	if _, exists := a.clients[clientID]; !exists {
		return ErrClientNotFound
	}

	delete(a.clients, clientID)
	return nil
}

// GetClient retrieves a client by ID
func (a *Authenticator) GetClient(clientID string) (*ClientInfo, error) {
	client, exists := a.clients[clientID]
	if !exists {
		return nil, ErrClientNotFound
	}

	return client, nil
}

// ListClients returns all clients
func (a *Authenticator) ListClients() []*ClientInfo {
	clients := make([]*ClientInfo, 0, len(a.clients))
	for _, client := range a.clients {
		clients = append(clients, client)
	}
	return clients
}

// Authenticate authenticates a client using password and remote address
func (a *Authenticator) Authenticate(password string, remoteAddr string) (*ClientInfo, error) {
	if !a.config.EnableAuth {
		// If authentication is disabled, allow any password
		return &ClientInfo{
			ID:       "anonymous",
			Password: password,
			Name:     "Anonymous User",
			Status:   "active",
		}, nil
	}

	// Find client by password hash
	for _, client := range a.clients {
		if a.hashPassword(client.Password) == password {
			// Check client status
			if client.Status != "active" {
				return nil, ErrClientNotFound
			}

			// Check expiration
			if client.ExpiresAt != nil && time.Now().After(*client.ExpiresAt) {
				return nil, ErrClientExpired
			}

			// Check data limit
			if client.DataLimit > 0 && client.DataUsed >= client.DataLimit {
				return nil, ErrDataLimitExceeded
			}

			// Check IP whitelist
			if len(client.IPWhitelist) > 0 {
				clientIP := a.getClientIP(remoteAddr)
				allowed := false
				for _, allowedIP := range client.IPWhitelist {
					if a.isIPAllowed(clientIP, allowedIP) {
						allowed = true
						break
					}
				}
				if !allowed {
					return nil, ErrIPNotAllowed
				}
			}

			// Update last seen
			now := time.Now()
			client.LastSeen = &now

			return client, nil
		}
	}

	return nil, ErrInvalidPassword
}

// hashPassword creates SHA256 hash of the password
func (a *Authenticator) hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// getClientIP extracts client IP from remote address
func (a *Authenticator) getClientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

// isIPAllowed checks if client IP is allowed by whitelist entry
func (a *Authenticator) isIPAllowed(clientIP, allowedEntry string) bool {
	// Support CIDR notation
	if strings.Contains(allowedEntry, "/") {
		_, ipNet, err := net.ParseCIDR(allowedEntry)
		if err != nil {
			return clientIP == allowedEntry
		}
		ip := net.ParseIP(clientIP)
		if ip == nil {
			return false
		}
		return ipNet.Contains(ip)
	}

	// Simple IP match
	return clientIP == allowedEntry
}

// UpdateClientStats updates client data usage
func (a *Authenticator) UpdateClientStats(clientID string, uploadBytes, downloadBytes int64) error {
	client, exists := a.clients[clientID]
	if !exists {
		return ErrClientNotFound
	}

	client.DataUsed += uploadBytes + downloadBytes
	return nil
}

// GetClientStats returns statistics for all clients
func (a *Authenticator) GetClientStats() []*ClientStats {
	stats := make([]*ClientStats, 0, len(a.clients))
	for _, client := range a.clients {
		stats = append(stats, &ClientStats{
			ClientID:      client.ID,
			UploadBytes:   0, // These would be tracked in the connection manager
			DownloadBytes: 0,
			TotalBytes:    client.DataUsed,
			Connections:   0, // This would be tracked in the connection manager
			LastActivity:  time.Now(),
		})
	}
	return stats
}

// ValidatePassword validates password format
func (a *Authenticator) ValidatePassword(password string) bool {
	return len(password) >= 8 && len(password) <= 128
}

// GeneratePassword generates a secure password (simplified version)
func (a *Authenticator) GeneratePassword() string {
	// This is a simplified version - in production, use crypto/rand
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	password := make([]byte, 16)
	for i := range password {
		password[i] = charset[i%len(charset)]
	}
	return string(password)
}
