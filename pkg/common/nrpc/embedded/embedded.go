package embedded

import (
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

var (
	// Global NATS connection
	globalConn *nats.Conn
	connMutex  sync.RWMutex
)

// Config holds the embedded NATS configuration
type Config struct {
	URL             string        `json:"url"`
	MaxReconnects   int           `json:"max_reconnects"`
	ReconnectWait   time.Duration `json:"reconnect_wait"`
	ReconnectJitter time.Duration `json:"reconnect_jitter"`
	ClientName      string        `json:"client_name"`
}

// DefaultConfig returns the default embedded NATS configuration
func DefaultConfig() *Config {
	return &Config{
		URL:             nats.DefaultURL,
		MaxReconnects:   5,
		ReconnectWait:   2 * time.Second,
		ReconnectJitter: 100 * time.Millisecond,
		ClientName:      "metabase-embedded",
	}
}

// Connect establishes a connection to the embedded NATS server
func Connect() (*nats.Conn, error) {
	connMutex.Lock()
	defer connMutex.Unlock()

	if globalConn != nil && globalConn.IsConnected() {
		return globalConn, nil
	}

	config := DefaultConfig()

	opts := []nats.Option{
		nats.ReconnectWait(config.ReconnectWait),
		nats.ReconnectJitter(config.ReconnectJitter, 2*time.Millisecond),
		nats.MaxReconnects(config.MaxReconnects),
		nats.Name(config.ClientName),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				fmt.Printf("NATS disconnected: %v\n", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Printf("NATS reconnected to %s\n", nc.ConnectedUrl())
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			if err != nil {
				fmt.Printf("NATS error: %v\n", err)
			}
		}),
	}

	conn, err := nats.Connect(config.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	globalConn = conn
	return globalConn, nil
}

// ConnectWithConfig establishes a connection with custom configuration
func ConnectWithConfig(config *Config) (*nats.Conn, error) {
	connMutex.Lock()
	defer connMutex.Unlock()

	if config == nil {
		config = DefaultConfig()
	}

	if globalConn != nil && globalConn.IsConnected() {
		globalConn.Close()
	}

	opts := []nats.Option{
		nats.ReconnectWait(config.ReconnectWait),
		nats.ReconnectJitter(config.ReconnectJitter, 2*time.Millisecond),
		nats.MaxReconnects(config.MaxReconnects),
		nats.Name(config.ClientName),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				fmt.Printf("NATS disconnected: %v\n", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Printf("NATS reconnected to %s\n", nc.ConnectedUrl())
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			if err != nil {
				fmt.Printf("NATS error: %v\n", err)
			}
		}),
	}

	conn, err := nats.Connect(config.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	globalConn = conn
	return globalConn, nil
}

// GetConnection returns the global NATS connection
func GetConnection() *nats.Conn {
	connMutex.RLock()
	defer connMutex.RUnlock()
	return globalConn
}

// IsConnected returns true if the global NATS connection is active
func IsConnected() bool {
	connMutex.RLock()
	defer connMutex.RUnlock()
	return globalConn != nil && globalConn.IsConnected()
}

// Disconnect closes the global NATS connection
func Disconnect() error {
	connMutex.Lock()
	defer connMutex.Unlock()

	if globalConn != nil {
		conn := globalConn
		globalConn = nil
		conn.Close()
	}
	return nil
}

// GetStats returns connection statistics
func GetStats() map[string]interface{} {
	connMutex.RLock()
	defer connMutex.RUnlock()

	if globalConn == nil {
		return map[string]interface{}{
			"connected": false,
		}
	}

	stats := globalConn.Stats()
	return map[string]interface{}{
		"connected":         globalConn.IsConnected(),
		"connected_url":     globalConn.ConnectedUrl(),
		"reconnects":        stats.Reconnects,
		"messages_sent":     stats.OutMsgs,
		"messages_received": stats.InMsgs,
		"bytes_sent":        stats.OutBytes,
		"bytes_received":    stats.InBytes,
	}
}
