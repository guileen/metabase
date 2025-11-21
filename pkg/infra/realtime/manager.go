package realtime

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/guileen/metabase/pkg/client"
)

// EventType represents different types of real-time events
type EventType string

const (
	EventSystemMetrics EventType = "system_metrics"
	EventSearchUpdate  EventType = "search_update"
	EventStorageUpdate EventType = "storage_update"
	EventUserActivity  EventType = "user_activity"
	EventTenantUpdate  EventType = "tenant_update"
	EventAlert         EventType = "alert"
	EventNotification  EventType = "notification"
)

// Event represents a real-time event
type Event struct {
	Type      EventType   `json:"type"`
	Timestamp int64       `json:"timestamp"`
	Channel   string      `json:"channel,omitempty"`
	Data      interface{} `json:"data"`
	ID        string      `json:"id"`
	TenantID  string      `json:"tenant_id,omitempty"`
	UserID    string      `json:"user_id,omitempty"`
}

// Connection represents a WebSocket connection
type Connection struct {
	ID       string
	Conn     *websocket.Conn
	TenantID string
	UserID   string
	Channels map[string]bool
	LastPing time.Time
	mu       sync.RWMutex
	Send     chan Event
	Manager  *Manager
}

// Manager manages real-time connections and events
type Manager struct {
	connections map[string]*Connection
	channels    map[string]map[string]bool // channel -> connection IDs
	events      chan Event
	register    chan *Connection
	unregister  chan *Connection
	broadcast   chan Event
	apiClient   *client.Client
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	config      *Config
}

// Config represents manager configuration
type Config struct {
	PingInterval      time.Duration
	PongWait          time.Duration
	WriteWait         time.Duration
	MaxMessageSize    int64
	EnableCompression bool
	BufferSize        int
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		PingInterval:      30 * time.Second,
		PongWait:          60 * time.Second,
		WriteWait:         10 * time.Second,
		MaxMessageSize:    512,
		EnableCompression: true,
		BufferSize:        256,
	}
}

// NewManager creates a new real-time manager
func NewManager(apiClient *client.Client, config *Config) *Manager {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		connections: make(map[string]*Connection),
		channels:    make(map[string]map[string]bool),
		events:      make(chan Event, config.BufferSize),
		register:    make(chan *Connection, 64),
		unregister:  make(chan *Connection, 64),
		broadcast:   make(chan Event, config.BufferSize),
		apiClient:   apiClient,
		ctx:         ctx,
		cancel:      cancel,
		config:      config,
	}

	return manager
}

// Start starts the real-time manager
func (m *Manager) Start() {
	go m.run()
	go m.runEventLoop()
	go m.startMetricsTicker()
}

// Stop stops the real-time manager
func (m *Manager) Stop() {
	m.cancel()
	close(m.events)
	close(m.register)
	close(m.unregister)
	close(m.broadcast)

	// Close all connections
	m.mu.Lock()
	for _, conn := range m.connections {
		conn.Close()
	}
	m.mu.Unlock()
}

// run manages connection lifecycle
func (m *Manager) run() {
	ticker := time.NewTicker(m.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case conn := <-m.register:
			m.registerConnection(conn)
		case conn := <-m.unregister:
			m.unregisterConnection(conn)
		case event := <-m.broadcast:
			m.broadcastEvent(event)
		case <-ticker.C:
			m.pingConnections()
		}
	}
}

// runEventLoop processes events
func (m *Manager) runEventLoop() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case event := <-m.events:
			m.processEvent(event)
		}
	}
}

// registerConnection adds a new connection
func (m *Manager) registerConnection(conn *Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.connections[conn.ID] = conn
	log.Printf("Connection %s registered (total: %d)", conn.ID, len(m.connections))
}

// unregisterConnection removes a connection
func (m *Manager) unregisterConnection(conn *Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.connections[conn.ID]; exists {
		delete(m.connections, conn.ID)

		// Remove from all channels
		for channel := range conn.Channels {
			if channelConns, exists := m.channels[channel]; exists {
				delete(channelConns, conn.ID)
				if len(channelConns) == 0 {
					delete(m.channels, channel)
				}
			}
		}

		conn.Close()
		log.Printf("Connection %s unregistered (total: %d)", conn.ID, len(m.connections))
	}
}

// subscribe adds connection to a channel
func (m *Manager) subscribe(conn *Connection, channel string) {
	conn.mu.Lock()
	if conn.Channels == nil {
		conn.Channels = make(map[string]bool)
	}
	conn.Channels[channel] = true
	conn.mu.Unlock()

	m.mu.Lock()
	if m.channels[channel] == nil {
		m.channels[channel] = make(map[string]bool)
	}
	m.channels[channel][conn.ID] = true
	m.mu.Unlock()

	log.Printf("Connection %s subscribed to channel %s", conn.ID, channel)
}

// unsubscribe removes connection from a channel
func (m *Manager) unsubscribe(conn *Connection, channel string) {
	conn.mu.Lock()
	delete(conn.Channels, channel)
	conn.mu.Unlock()

	m.mu.Lock()
	if channelConns, exists := m.channels[channel]; exists {
		delete(channelConns, conn.ID)
		if len(channelConns) == 0 {
			delete(m.channels, channel)
		}
	}
	m.mu.Unlock()

	log.Printf("Connection %s unsubscribed from channel %s", conn.ID, channel)
}

// broadcastEvent sends event to all subscribers of a channel
func (m *Manager) broadcastEvent(event Event) {
	m.mu.RLock()
	channelConns, exists := m.channels[event.Channel]
	m.mu.RUnlock()

	if !exists {
		return
	}

	for connID := range channelConns {
		m.mu.RLock()
		conn, exists := m.connections[connID]
		m.mu.RUnlock()

		if exists && m.shouldSendEvent(conn, event) {
			select {
			case conn.Send <- event:
			default:
				// Channel is full, drop the event
				log.Printf("Dropping event for connection %s: channel buffer full", connID)
			}
		}
	}
}

// processEvent handles incoming events
func (m *Manager) processEvent(event Event) {
	switch event.Type {
	case EventSystemMetrics:
		m.broadcastEvent(event)
	case EventSearchUpdate:
		m.broadcastEvent(event)
	case EventStorageUpdate:
		m.broadcastEvent(event)
	case EventUserActivity:
		m.broadcastEvent(event)
	case EventTenantUpdate:
		m.broadcastEvent(event)
	case EventAlert:
		m.broadcastEvent(event)
	case EventNotification:
		m.broadcastEvent(event)
	default:
		log.Printf("Unknown event type: %s", event.Type)
	}
}

// shouldSendEvent checks if connection should receive the event
func (m *Manager) shouldSendEvent(conn *Connection, event Event) bool {
	// Check tenant filter
	if event.TenantID != "" && conn.TenantID != "" && event.TenantID != conn.TenantID {
		return false
	}

	// Check user filter
	if event.UserID != "" && conn.UserID != "" && event.UserID != conn.UserID {
		return false
	}

	return true
}

// pingConnections sends ping to all connections
func (m *Manager) pingConnections() {
	m.mu.RLock()
	connections := make([]*Connection, 0, len(m.connections))
	for _, conn := range m.connections {
		connections = append(connections, conn)
	}
	m.mu.RUnlock()

	for _, conn := range connections {
		if err := conn.Ping(); err != nil {
			log.Printf("Failed to ping connection %s: %v", conn.ID, err)
			go func() { m.unregister <- conn }()
		}
	}
}

// startMetricsTicker starts sending system metrics periodically
func (m *Manager) startMetricsTicker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			if err := m.sendSystemMetrics(); err != nil {
				log.Printf("Failed to send system metrics: %v", err)
			}
		}
	}
}

// sendSystemMetrics sends system metrics to all subscribers
func (m *Manager) sendSystemMetrics() error {
	if m.apiClient == nil {
		return fmt.Errorf("API client not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := m.apiClient.Health(ctx)
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}

	event := Event{
		Type:      EventSystemMetrics,
		Timestamp: time.Now().Unix(),
		Channel:   "system.metrics",
		Data:      response,
		ID:        generateEventID(),
	}

	select {
	case m.events <- event:
		return nil
	default:
		return fmt.Errorf("event channel is full")
	}
}

// HandleConnection handles a new WebSocket connection
func (m *Manager) HandleConnection(wsConn *websocket.Conn, tenantID, userID string) *Connection {
	connID := generateConnectionID()

	conn := &Connection{
		ID:       connID,
		Conn:     wsConn,
		TenantID: tenantID,
		UserID:   userID,
		Channels: make(map[string]bool),
		LastPing: time.Now(),
		Send:     make(chan Event, m.config.BufferSize),
		Manager:  m,
	}

	// Register connection
	m.register <- conn

	// Start connection goroutines
	go conn.writePump()
	go conn.readPump()

	return conn
}

// PublishEvent publishes an event to the manager
func (m *Manager) PublishEvent(eventType EventType, channel string, data interface{}, tenantID, userID string) error {
	event := Event{
		Type:      eventType,
		Timestamp: time.Now().Unix(),
		Channel:   channel,
		Data:      data,
		ID:        generateEventID(),
		TenantID:  tenantID,
		UserID:    userID,
	}

	select {
	case m.events <- event:
		return nil
	default:
		return fmt.Errorf("event channel is full")
	}
}

// PublishSearchUpdate publishes a search update event
func (m *Manager) PublishSearchUpdate(update interface{}) error {
	return m.PublishEvent(EventSearchUpdate, "search.updates", update, "", "")
}

// PublishStorageUpdate publishes a storage update event
func (m *Manager) PublishStorageUpdate(update interface{}) error {
	return m.PublishEvent(EventStorageUpdate, "storage.updates", update, "", "")
}

// GetStats returns manager statistics
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"connections": len(m.connections),
		"channels":    len(m.channels),
		"timestamp":   time.Now().Unix(),
	}

	// Channel statistics
	channelStats := make(map[string]int)
	for channel, conns := range m.channels {
		channelStats[channel] = len(conns)
	}
	stats["channel_stats"] = channelStats

	return stats
}

// Connection methods

// Ping sends a ping to the connection
func (c *Connection) Ping() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.LastPing = time.Now()

	if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		return err
	}

	return nil
}

// Close closes the connection
func (c *Connection) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Conn != nil {
		c.Conn.Close()
	}
	close(c.Send)
}

// writePump handles writing messages to WebSocket
func (c *Connection) writePump() {
	ticker := time.NewTicker(c.Manager.config.PingInterval)
	defer func() {
		ticker.Stop()
		c.Manager.unregister <- c
	}()

	for {
		select {
		case <-c.Manager.ctx.Done():
			return
		case event, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Manager.config.WriteWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(event); err != nil {
				log.Printf("Failed to write JSON to connection %s: %v", c.ID, err)
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Manager.config.WriteWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump handles reading messages from WebSocket
func (c *Connection) readPump() {
	defer func() {
		c.Manager.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(c.Manager.config.MaxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(c.Manager.config.PongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(c.Manager.config.PongWait))
		return nil
	})

	for {
		select {
		case <-c.Manager.ctx.Done():
			return
		default:
		}

		var msg struct {
			Type    string      `json:"type"`
			Channel string      `json:"channel"`
			Data    interface{} `json:"data"`
		}

		if err := c.Conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for connection %s: %v", c.ID, err)
			}
			break
		}

		// Handle subscription messages
		switch msg.Type {
		case "subscribe":
			c.Manager.subscribe(c, msg.Channel)
		case "unsubscribe":
			c.Manager.unsubscribe(c, msg.Channel)
		}
	}
}

// Utility functions

func generateConnectionID() string {
	return fmt.Sprintf("conn_%d", time.Now().UnixNano())
}

func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}
