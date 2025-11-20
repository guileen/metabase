package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
)

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorInfo represents error information
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Meta represents response metadata
type Meta struct {
	Timestamp int64  `json:"timestamp"`
	RequestID string `json:"request_id"`
	Version   string `json:"version"`
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Channel   string      `json:"channel,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp int64       `json:"timestamp"`
	ID        string      `json:"id,omitempty"`
}

// Config represents API client configuration
type Config struct {
	BaseURL            string
	WebSocketURL       string
	APIKey            string
	Timeout           time.Duration
	RetryAttempts     int
	RetryDelay        time.Duration
	EnableCompression bool
}

// Client represents the unified API client
type Client struct {
	config     *Config
	httpClient *http.Client
	wsConn     *websocket.Conn
	natsConn   *nats.Conn
	mu         sync.RWMutex
	subs       map[string]func(WebSocketMessage)
	subMu      sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewClient creates a new API client
func NewClient(config *Config) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	// Set default values
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}

	client := &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		subs:   make(map[string]func(WebSocketMessage)),
		ctx:    ctx,
		cancel: cancel,
	}

	return client
}

// ConnectWebSocket establishes WebSocket connection
func (c *Client) ConnectWebSocket() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dialer := websocket.DefaultDialer
	if c.config.EnableCompression {
		dialer.EnableCompression = true
	}

	wsURL := c.config.WebSocketURL
	if wsURL == "" {
		wsURL = fmt.Sprintf("ws://%s/api/ws", c.config.BaseURL)
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect WebSocket: %w", err)
	}

	c.wsConn = conn

	// Start message handler
	go c.handleWebSocketMessages()

	return nil
}

// ConnectNATS establishes NATS connection
func (c *Client) ConnectNATS() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		return fmt.Errorf("failed to connect NATS: %w", err)
	}

	c.natsConn = nc
	return nil
}

// Request makes HTTP request with retry logic
func (c *Client) Request(ctx context.Context, method, endpoint string, data interface{}) (*Response, error) {
	var lastErr error

	for attempt := 0; attempt < c.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.config.RetryDelay):
			}
		}

		resp, err := c.doRequest(ctx, method, endpoint, data)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		log.Printf("Request attempt %d failed: %v", attempt+1, err)
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", c.config.RetryAttempts, lastErr)
}

// doRequest performs a single HTTP request
func (c *Client) doRequest(ctx context.Context, method, endpoint string, data interface{}) (*Response, error) {
	url := fmt.Sprintf("%s%s", c.config.BaseURL, endpoint)

	var body []byte
	var err error
	if data != nil {
		body, err = json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Body = http.NoBody
		req.ContentLength = int64(len(body))
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	// Add body if present
	if len(body) > 0 {
		req.Body = http.NoBody
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(body)), nil
		}
		req.ContentLength = int64(len(body))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	var apiResp Response
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success && apiResp.Error != nil {
		return &apiResp, fmt.Errorf("API error: %s - %s", apiResp.Error.Code, apiResp.Error.Message)
	}

	return &apiResp, nil
}

// Get performs GET request
func (c *Client) Get(ctx context.Context, endpoint string) (*Response, error) {
	return c.Request(ctx, http.MethodGet, endpoint, nil)
}

// Post performs POST request
func (c *Client) Post(ctx context.Context, endpoint string, data interface{}) (*Response, error) {
	return c.Request(ctx, http.MethodPost, endpoint, data)
}

// Put performs PUT request
func (c *Client) Put(ctx context.Context, endpoint string, data interface{}) (*Response, error) {
	return c.Request(ctx, http.MethodPut, endpoint, data)
}

// Delete performs DELETE request
func (c *Client) Delete(ctx context.Context, endpoint string) (*Response, error) {
	return c.Request(ctx, http.MethodDelete, endpoint, nil)
}

// Subscribe subscribes to WebSocket channel
func (c *Client) Subscribe(channel string, handler func(WebSocketMessage)) error {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	c.subs[channel] = handler
	return nil
}

// Unsubscribe unsubscribes from WebSocket channel
func (c *Client) Unsubscribe(channel string) {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	delete(c.subs, channel)
}

// Publish publishes message to WebSocket channel
func (c *Client) Publish(channel string, data interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.wsConn == nil {
		return fmt.Errorf("WebSocket connection not established")
	}

	msg := WebSocketMessage{
		Type:      "publish",
		Channel:   channel,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	return c.wsConn.WriteJSON(msg)
}

// PublishNATS publishes message to NATS subject
func (c *Client) PublishNATS(subject string, data []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.natsConn == nil {
		return fmt.Errorf("NATS connection not established")
	}

	return c.natsConn.Publish(subject, data)
}

// SubscribeNATS subscribes to NATS subject
func (c *Client) SubscribeNATS(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.natsConn == nil {
		return nil, fmt.Errorf("NATS connection not established")
	}

	return c.natsConn.Subscribe(subject, handler)
}

// handleWebSocketMessages handles incoming WebSocket messages
func (c *Client) handleWebSocketMessages() {
	defer func() {
		c.mu.Lock()
		if c.wsConn != nil {
			c.wsConn.Close()
			c.wsConn = nil
		}
		c.mu.Unlock()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		c.mu.RLock()
		conn := c.wsConn
		c.mu.RUnlock()

		if conn == nil {
			return
		}

		var msg WebSocketMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			return
		}

		// Handle message based on type
		switch msg.Type {
		case "message", "publish":
			c.subMu.RLock()
			handler, exists := c.subs[msg.Channel]
			c.subMu.RUnlock()
			if exists {
				go handler(msg)
			}
		}
	}
}

// Close closes all connections
func (c *Client) Close() error {
	c.cancel()

	c.mu.Lock()
	defer c.mu.Unlock()

	var errs []error

	// Close WebSocket
	if c.wsConn != nil {
		if err := c.wsConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close WebSocket: %w", err))
		}
		c.wsConn = nil
	}

	// Close NATS
	if c.natsConn != nil {
		c.natsConn.Close()
		c.natsConn = nil
	}

	if len(errs) > 0 {
		return fmt.Errorf("multiple errors occurred: %v", errs)
	}

	return nil
}

// Health checks API health
func (c *Client) Health(ctx context.Context) error {
	resp, err := c.Get(ctx, "/api/health")
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("health check failed")
	}

	return nil
}

// GetMetrics retrieves system metrics
func (c *Client) GetMetrics(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "/api/metrics")
}

// GetSearchStatus retrieves search engine status
func (c *Client) GetSearchStatus(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "/api/search/status")
}

// GetStorageStats retrieves storage statistics
func (c *Client) GetStorageStats(ctx context.Context) (*Response, error) {
	return c.Get(ctx, "/api/storage/stats")
}