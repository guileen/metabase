package trojan

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// Server represents the Trojan proxy server
type Server struct {
	config       *TrojanConfig
	listener     net.Listener
	auth         *Authenticator
	connections  map[string]*Connection
	connMutex    sync.RWMutex
	stats        *ServerStats
	statsMutex   sync.RWMutex
	logger       *log.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	running      bool
	runningMutex sync.RWMutex

	// TLS certificate management
	tlsConfig *tls.Config
	certMgr   *CertificateManager
}

// NewServer creates a new Trojan server
func NewServer(config *TrojanConfig) (*Server, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	server := &Server{
		config:      config.Clone(),
		auth:        NewAuthenticator(config),
		connections: make(map[string]*Connection),
		stats: &ServerStats{
			ActiveConnections: 0,
			TotalConnections:  0,
			UploadBytes:       0,
			DownloadBytes:     0,
			TotalBytes:        0,
			Uptime:            0,
			CPUUsage:          0,
			MemoryUsage:       0,
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize logger
	if config.LogPath != "" {
		// In a real implementation, set up proper file logging
		server.logger = log.New(io.Discard, "[trojan] ", log.LstdFlags)
	} else {
		server.logger = log.New(io.Discard, "[trojan] ", log.LstdFlags)
	}

	// Setup TLS configuration
	if err := server.setupTLS(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to setup TLS: %w", err)
	}

	return server, nil
}

// setupTLS configures TLS for the server
func (s *Server) setupTLS() error {
	if !s.config.EnableTLS {
		return nil
	}

	// Initialize certificate manager
	var err error
	s.certMgr, err = NewCertificateManager(s.config)
	if err != nil {
		return fmt.Errorf("failed to create certificate manager: %w", err)
	}

	s.tlsConfig = &tls.Config{
		GetCertificate: s.certMgr.GetCertificate,
		NextProtos:     s.config.ALPN,
		CipherSuites:   s.config.CipherSuites,
		MinVersion:     tls.VersionTLS12,
		MaxVersion:     tls.VersionTLS13,
	}

	return nil
}

// Start starts the Trojan server
func (s *Server) Start() error {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if s.running {
		return ErrServerAlreadyRunning
	}

	// Start statistics collector
	go s.collectStats()

	// Start certificate manager if auto-cert is enabled
	if s.config.AutoCert && s.certMgr != nil {
		go s.certMgr.Start()
	}

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	var listener net.Listener
	var err error

	if s.config.EnableTLS && s.tlsConfig != nil {
		listener, err = tls.Listen("tcp", addr, s.tlsConfig)
	} else {
		listener, err = net.Listen("tcp", addr)
	}

	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.listener = listener
	s.running = true

	s.logger.Printf("Trojan server started on %s", addr)

	// Start accepting connections
	go s.acceptConnections()

	return nil
}

// Stop stops the Trojan server
func (s *Server) Stop() error {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if !s.running {
		return ErrServerNotStarted
	}

	s.cancel()
	s.running = false

	// Close listener
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			s.logger.Printf("Error closing listener: %v", err)
		}
	}

	// Close all connections
	s.connMutex.Lock()
	for _, conn := range s.connections {
		if conn.Status == "active" {
			// Close connection logic would go here
			conn.Status = "closed"
		}
	}
	s.connMutex.Unlock()

	// Stop certificate manager
	if s.certMgr != nil {
		s.certMgr.Stop()
	}

	s.logger.Println("Trojan server stopped")
	return nil
}

// acceptConnections accepts incoming connections
func (s *Server) acceptConnections() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.ctx.Done():
					return
				default:
					s.logger.Printf("Error accepting connection: %v", err)
					continue
				}
			}

			go s.handleConnection(conn)
		}
	}
}

// handleConnection handles a new connection
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Check client limit
	if s.getActiveConnectionCount() >= s.config.MaxClients {
		s.logger.Printf("Connection rejected: maximum client limit reached")
		return
	}

	// Set timeout
	if err := conn.SetDeadline(time.Now().Add(s.config.ClientTimeout)); err != nil {
		s.logger.Printf("Error setting connection deadline: %v", err)
		return
	}

	// Create TLS connection if needed
	var tlsConn *tls.Conn
	if s.config.EnableTLS {
		if tc, ok := conn.(*tls.Conn); ok {
			tlsConn = tc
			if err := tlsConn.Handshake(); err != nil {
				s.logger.Printf("TLS handshake failed: %v", err)
				return
			}
		} else {
			s.logger.Printf("Expected TLS connection but got plain TCP")
			return
		}
	} else {
		tlsConn = conn.(*tls.Conn) // This won't work, but for now...
	}

	// Perform Trojan protocol handshake
	client, targetAddr, err := s.handshake(tlsConn)
	if err != nil {
		s.logger.Printf("Handshake failed: %v", err)
		return
	}

	// Add connection to tracking
	connID := s.generateConnectionID()
	s.addConnection(connID, &Connection{
		ID:         connID,
		ClientID:   client.ID,
		RemoteAddr: conn.RemoteAddr().String(),
		TargetAddr: targetAddr,
		CreatedAt:  time.Now(),
		Status:     "active",
	})
	defer s.removeConnection(connID)

	s.logger.Printf("New connection from client %s to %s", client.ID, targetAddr)

	// Connect to target
	targetConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		s.logger.Printf("Failed to connect to target %s: %v", targetAddr, err)
		return
	}
	defer targetConn.Close()

	// Start data relay
	s.relayData(tlsConn, targetConn, client)
}

// handshake performs the Trojan protocol handshake
func (s *Server) handshake(conn *tls.Conn) (*ClientInfo, string, error) {
	reader := bufio.NewReader(conn)

	// Read authentication data
	// Trojan protocol: HASH(password) + CRLF + COMMAND + CRLF + TARGET + CRLF
	authLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, "", fmt.Errorf("failed to read auth line: %w", err)
	}
	authLine = authLine[:len(authLine)-2] // Remove CRLF

	// Authenticate client
	client, err := s.auth.Authenticate(authLine, conn.RemoteAddr().String())
	if err != nil {
		return nil, "", fmt.Errorf("authentication failed: %w", err)
	}

	// Read command
	command, err := reader.ReadString('\n')
	if err != nil {
		return nil, "", fmt.Errorf("failed to read command: %w", err)
	}
	command = command[:len(command)-2] // Remove CRLF

	if command != "CONNECT" {
		return nil, "", fmt.Errorf("unsupported command: %s", command)
	}

	// Read target
	target, err := reader.ReadString('\n')
	if err != nil {
		return nil, "", fmt.Errorf("failed to read target: %w", err)
	}
	target = target[:len(target)-2] // Remove CRLF

	return client, target, nil
}

// relayData relays data between client and target
func (s *Server) relayData(clientConn, targetConn net.Conn, client *ClientInfo) {
	var wg sync.WaitGroup
	var uploadBytes, downloadBytes int64

	// Client to target
	wg.Add(1)
	go func() {
		defer wg.Done()
		buffer := make([]byte, s.config.BufferSize)
		for {
			n, err := clientConn.Read(buffer)
			if err != nil {
				break
			}
			_, err = targetConn.Write(buffer[:n])
			if err != nil {
				break
			}
			uploadBytes += int64(n)
		}
	}()

	// Target to client
	wg.Add(1)
	go func() {
		defer wg.Done()
		buffer := make([]byte, s.config.BufferSize)
		for {
			n, err := targetConn.Read(buffer)
			if err != nil {
				break
			}
			_, err = clientConn.Write(buffer[:n])
			if err != nil {
				break
			}
			downloadBytes += int64(n)
		}
	}()

	wg.Wait()

	// Update client stats
	s.auth.UpdateClientStats(client.ID, uploadBytes, downloadBytes)

	// Update server stats
	s.updateStats(uploadBytes, downloadBytes)
}

// Helper methods

func (s *Server) generateConnectionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

func (s *Server) getActiveConnectionCount() int {
	s.connMutex.RLock()
	defer s.connMutex.RUnlock()

	count := 0
	for _, conn := range s.connections {
		if conn.Status == "active" {
			count++
		}
	}
	return count
}

func (s *Server) addConnection(id string, conn *Connection) {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()
	s.connections[id] = conn
	s.stats.ActiveConnections++
	s.stats.TotalConnections++
}

func (s *Server) removeConnection(id string) {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()
	if conn, exists := s.connections[id]; exists {
		conn.Status = "closed"
		delete(s.connections, id)
		s.stats.ActiveConnections--
	}
}

func (s *Server) updateStats(uploadBytes, downloadBytes int64) {
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()
	s.stats.UploadBytes += uploadBytes
	s.stats.DownloadBytes += downloadBytes
	s.stats.TotalBytes += uploadBytes + downloadBytes
}

func (s *Server) collectStats() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.statsMutex.Lock()
			s.stats.Uptime = time.Since(startTime)
			// In a real implementation, collect actual CPU and memory usage
			s.stats.CPUUsage = 0
			s.stats.MemoryUsage = 0
			s.statsMutex.Unlock()
		}
	}
}

// Public API methods

// IsRunning returns whether the server is running
func (s *Server) IsRunning() bool {
	s.runningMutex.RLock()
	defer s.runningMutex.RUnlock()
	return s.running
}

// GetConfig returns the current configuration
func (s *Server) GetConfig() *TrojanConfig {
	return s.config.Clone()
}

// UpdateConfig updates the server configuration
func (s *Server) UpdateConfig(newConfig *TrojanConfig) error {
	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	wasRunning := s.IsRunning()
	if wasRunning {
		if err := s.Stop(); err != nil {
			return fmt.Errorf("failed to stop server for config update: %w", err)
		}
	}

	s.config = newConfig.Clone()
	if err := s.setupTLS(); err != nil {
		return fmt.Errorf("failed to setup TLS with new config: %w", err)
	}

	if wasRunning {
		if err := s.Start(); err != nil {
			return fmt.Errorf("failed to restart server with new config: %w", err)
		}
	}

	return nil
}

// GetStats returns server statistics
func (s *Server) GetStats() *ServerStats {
	s.statsMutex.RLock()
	defer s.statsMutex.RUnlock()

	stats := *s.stats
	return &stats
}

// GetConnections returns active connections
func (s *Server) GetConnections() []*Connection {
	s.connMutex.RLock()
	defer s.connMutex.RUnlock()

	connections := make([]*Connection, 0, len(s.connections))
	for _, conn := range s.connections {
		connections = append(connections, conn)
	}
	return connections
}

// GetAuth returns the authenticator
func (s *Server) GetAuth() *Authenticator {
	return s.auth
}
