package embedded

import ("context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go")

// EmbeddedNATS represents an embedded NATS server
type EmbeddedNATS struct {
	server      *server.Server
	conn        *nats.Conn
	config      *Config
	shutdown    atomic.Bool
	shutdownCh  chan struct{}
	ready       atomic.Bool
	readyCh     chan struct{}
	mu          sync.RWMutex
	subscribers map[string][]*nats.Subscription
	startOnce   sync.Once
}

// Config represents embedded NATS configuration
type Config struct {
	ServerPort     int           `yaml:"server_port" json:"server_port"`
	ClientURL      string        `yaml:"client_url" json:"client_url"`
	StoreDir       string        `yaml:"store_dir" json:"store_dir"`
	MaxConn        int           `yaml:"max_conn" json:"max_conn"`
	ReconnectWait time.Duration `yaml:"reconnect_wait" json:"reconnect_wait"`
	JetStream      bool          `yaml:"jetstream" json:"jetstream"`
	MaxMemory      int64         `yaml:"max_memory" json:"max_memory"`
	MaxPayload     int32         `yaml:"max_payload" json:"max_payload"`
	Cluster        *ClusterConfig `yaml:"cluster" json:"cluster"`
	Monitoring     *MonitorConfig `yaml:"monitoring" json:"monitoring"`
	LogLevel       string        `yaml:"log_level" json:"log_level"`
}

// ClusterConfig for NATS clustering
type ClusterConfig struct {
	Name     string `yaml:"name" json:"name"`
	Routes   []string `yaml:"routes" json:"routes"`
	Seed     string `yaml:"seed" json:"seed"`
}

// MonitorConfig for monitoring
type MonitorConfig struct {
	HTTPPort   int    `yaml:"http_port" json:"http_port"`
	HTTPHost   string `yaml:"http_host" json:"http_host"`
	Prometheus bool   `yaml:"prometheus" json:"prometheus"`
}

// NewEmbeddedNATS creates a new embedded NATS server
func NewEmbeddedNATS(config *Config) *EmbeddedNATS {
	if config == nil {
		config = getDefaultConfig()
	}

	return &EmbeddedNATS{
		config:      config,
		shutdownCh:  make(chan struct{}),
		readyCh:     make(chan struct{}),
		subscribers: make(map[string][]*nats.Subscription),
	}
}

// getDefaultConfig returns default embedded NATS configuration
func getDefaultConfig() *Config {
	return &Config{
		ServerPort:     4222,
		ClientURL:      "nats://localhost:4222",
		StoreDir:       "./data/nats",
		MaxConn:        1000,
		ReconnectWait:  2 * time.Second,
		JetStream:      true,
		MaxMemory:      1024 * 1024 * 1024, // 1GB
		MaxPayload:     1024 * 1024,      // 1MB
		LogLevel:       "info",
		Monitoring: &MonitorConfig{
			HTTPPort:   8222,
			HTTPHost:   "0.0.0.0",
			Prometheus: true,
		},
	}
}

// Start starts the embedded NATS server
func (e *EmbeddedNATS) Start() error {
	var startErr error
	e.startOnce.Do(func() {
		startErr = e.startServer()
	})

	return startErr
}

// startServer actually starts the NATS server
func (e *EmbeddedNATS) startServer() error {
	// Ensure store directory exists
	if err := os.MkdirAll(e.config.StoreDir, 0755); err != nil {
		return fmt.Errorf("failed to create NATS store directory: %w", err)
	}

	// Configure NATS server
	opts := &server.Options{
		ServerName:    "metabase-embedded",
		Host:          "127.0.0.1",
		Port:          e.config.ServerPort,
		StoreDir:      e.config.StoreDir,
		NoLog:         e.config.LogLevel == "none",
		NoSigs:        true,
		MaxControlLine: 512,
		MaxPayload:    e.config.MaxPayload,
		MaxPending:    8192,
		MaxConn:       e.config.MaxConn,
		WriteDeadline:  10 * time.Second,
	}

	// Enable JetStream if configured
	if e.config.JetStream {
		// Configure JetStream store
		jsOpts := &server.JetStreamOptions{
			StoreDir:   filepath.Join(e.config.StoreDir, "jetstream"),
			MaxMemory:  e.config.MaxMemory,
		}
		opts.JetStream = jsOpts
	}

	// Create server
	s, err := server.NewServer(opts)
	if err != nil {
		return fmt.Errorf("failed to create NATS server: %w", err)
	}

	// Start server in goroutine
	go func() {
		if err := s.Start(); err != nil {
			fmt.Printf("NATS server error: %v\n", err)
			e.shutdown.Store(true)
			close(e.shutdownCh)
		} else {
			e.ready.Store(true)
			close(e.readyCh)
		}
	}()

	e.server = s

	// Wait for server to be ready
	select {
	case <-e.readyCh:
		fmt.Printf("Embedded NATS server started on port %d\n", e.config.ServerPort)
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("NATS server failed to start within 10 seconds")
	}
}

// Stop stops the embedded NATS server
func (e *EmbeddedNATS) Stop() error {
	e.shutdown.Store(true)

	if e.server != nil {
		// Shutdown server with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := e.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown NATS server: %w", err)
		}
	}

	if e.conn != nil {
		if err := e.conn.Drain(); err != nil {
			return fmt.Errorf("failed to drain NATS connection: %w", err)
		}
		e.conn.Close()
	}

	// Wait for shutdown
	select {
	case <-e.shutdownCh:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("NATS server shutdown timeout")
	}
}

// GetConnection returns a NATS connection to the embedded server
func (e *EmbeddedNATS) GetConnection() (*nats.Conn, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.conn != nil {
		return e.conn, nil
	}

	// Wait for server to be ready
	select {
	case <-e.readyCh:
		// Server is ready, create connection
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("NATS server not ready")
	}

	// Create connection
	conn, err := nats.Connect(e.config.ClientURL,
		nats.ReconnectWait(e.config.ReconnectWait),
		nats.MaxReconnects(10),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			if sub != nil {
				e.removeSubscriber(sub.Subject)
			}
			fmt.Printf("NATS connection error: %v\n", err)
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			fmt.Printf("NATS disconnected: %v\n", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Printf("NATS reconnected to %s\n", nc.ConnectedUrl())
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to embedded NATS: %w", err)
	}

	e.conn = conn
	return conn, nil
}

// Subscribe subscribes to a subject with the embedded NATS server
func (e *EmbeddedNATS) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	conn, err := e.GetConnection()
	if err != nil {
		return nil, err
	}

	sub, err := conn.Subscribe(subject, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}

	e.addSubscriber(subject, sub)
	return sub, nil
}

// Publish publishes a message to the embedded NATS server
func (e *EmbeddedNATS) Publish(subject string, data []byte) error {
	conn, err := e.GetConnection()
	if err != nil {
		return err
	}

	return conn.Publish(subject, data)
}

// addSubscriber tracks a subscription
func (e *EmbeddedNATS) addSubscriber(subject string, sub *nats.Subscription) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.subscribers[subject] == nil {
		e.subscribers[subject] = make([]*nats.Subscription, 0)
	}
	e.subscribers[subject] = append(e.subscribers[subject], sub)
}

// removeSubscriber removes a subscription from tracking
func (e *EmbeddedNATS) removeSubscriber(subject string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	subs, exists := e.subscribers[subject]
	if !exists {
		return
	}

	var updated []*nats.Subscription
	for _, sub := range subs {
		if sub.Subject == subject {
			sub.Unsubscribe()
			continue
		}
		updated = append(updated, sub)
	}

	if len(updated) == 0 {
		delete(e.subscribers, subject)
	} else {
		e.subscribers[subject] = updated
	}
}

// GetStats returns embedded NATS server statistics
func (e *EmbeddedNATS) GetStats() *Stats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := &Stats{
		Ready:       e.ready.Load(),
		Shutdown:    e.shutdown.Load(),
		ServerPort:  e.config.ServerPort,
		ClientURL:   e.config.ClientURL,
		StoreDir:    e.config.StoreDir,
		MaxConn:     e.config.MaxConn,
		JetStream:   e.config.JetStream,
		Subscribers: make(map[string]int),
	}

	// Count subscribers by subject
	for subject, subs := range e.subscribers {
		stats.Subscribers[subject] = len(subs)
	}

	if e.conn != nil {
		natsStats := e.conn.Stats()
		stats.Connections = natatsStats.Connections
		stats.InMsgs = natatsStats.InMsgs
		stats.OutMsgs = natStats.OutMsgs
		stats.InBytes = natatsStats.InBytes
		stats.OutBytes = natatsStats.OutBytes
		stats.Reconnects = natatsStats.Reconnects
	}

	return stats
}

// Stats represents embedded NATS statistics
type Stats struct {
	Ready       bool              `json:"ready"`
	Shutdown    bool              `json:"shutdown"`
	ServerPort  int               `json:"server_port"`
	ClientURL   string            `json:"client_url"`
	StoreDir    string            `json:"store_dir"`
	MaxConn     int               `json:"max_conn"`
	JetStream   bool              `json:"jetstream"`
	Subscribers map[string]int    `json:"subscribers"`
	Connections int64             `json:"connections"`
	InMsgs      uint64            `json:"in_msgs"`
	OutMsgs     uint64            `json:"out_msgs"`
	InBytes     uint64            `json:"in_bytes"`
	OutBytes    uint64            `json:"out_bytes"`
	Reconnects  uint64            `json:"reconnects"`
}

// IsReady checks if the embedded NATS server is ready
func (e *EmbeddedNATS) IsReady() bool {
	return e.ready.Load()
}

// IsShutdown checks if the embedded NATS server is shutdown
func (e *EmbeddedNATS) IsShutdown() bool {
	return e.shutdown.Load()
}

// WaitReady waits for the server to be ready
func (e *EmbeddedNATS) WaitReady(timeout time.Duration) error {
	select {
	case <-e.readyCh:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("NATS server not ready within %v", timeout)
	}
}

// CreateJetStream creates a JetStream if configured
func (e *EmbeddedNATS) CreateJetStream(streamName string, config *JetStreamConfig) error {
	if !e.config.JetStream {
		return fmt.Errorf("JetStream is not enabled")
	}

	conn, err := e.GetConnection()
	if err != nil {
		return err
	}

	js := conn.JetStream()
	if js == nil {
		return fmt.Errorf("JetStream not available")
	}

	// Convert config to NATS JetStream config
	natsConfig := &nats.StreamConfig{
		Name:      streamName,
		Retention: "workqueue",
		Subjects:  []string{streamName + ".>"},
		MaxAge:    config.MaxAge,
		Storage:   nats.FileStorageType,
		Replicas:  config.Replicas,
	}

	if config.MaxBytes != 0 {
		natsConfig.MaxBytes = config.MaxBytes
	}
	if config.MaxMsgSize != 0 {
		natsConfig.MaxMsgSize = config.MaxMsgSize
	}

	return js.AddStream(natsConfig)
}

// JetStreamConfig represents JetStream configuration
type JetStreamConfig struct {
	MaxAge    time.Duration `json:"max_age"`
	MaxBytes  int64        `json:"max_bytes"`
	MaxMsgSize int32        `json:"max_msg_size"`
	Replicas  int           `json:"replicas"`
}