package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// Client represents an NRPC v2 client
type Client struct {
	nats          *nats.Conn
	config        *ClientConfig
	subscriptions map[string]*nats.Subscription
	pending       map[string]*pendingRequest
	mu            sync.RWMutex
	closed        bool
}

// ClientConfig represents client configuration
type ClientConfig struct {
	Namespace     string        `yaml:"namespace" json:"namespace"`
	Timeout       time.Duration `yaml:"timeout" json:"timeout"`
	EnableMetrics bool          `yaml:"enable_metrics" json:"enable_metrics"`
	EnableTracing bool          `yaml:"enable_tracing" json:"enable_tracing"`
	MaxRetries    int           `yaml:"max_retries" json:"max_retries"`
	RetryDelay    time.Duration `yaml:"retry_delay" json:"retry_delay"`
}

// pendingRequest represents a pending request
type pendingRequest struct {
	ID       string
	Response chan *Message
	Timeout  time.Time
}

// NewClient creates a new NRPC v2 client
func NewClient(conn *nats.Conn, config *ClientConfig) *Client {
	if config == nil {
		config = &ClientConfig{
			Namespace:     "metabase",
			Timeout:       30 * time.Second,
			EnableMetrics: false,
			EnableTracing: false,
			MaxRetries:    3,
			RetryDelay:    100 * time.Millisecond,
		}
	}

	client := &Client{
		nats:          conn,
		config:        config,
		subscriptions: make(map[string]*nats.Subscription),
		pending:       make(map[string]*pendingRequest),
	}

	// Start cleanup goroutine
	go client.cleanupPending()

	return client
}

// Call calls a service method
func (c *Client) Call(ctx context.Context, service, method string, data map[string]interface{}, metadata map[string]interface{}) (*Response, error) {
	requestID := generateMessageID()
	subject := c.getSubject(service, method)

	// Create request message
	msg := &Message{
		ID:        requestID,
		Type:      MessageTypeRequest,
		Service:   service,
		Method:    method,
		Data:      data,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}

	// Create pending request
	respChan := make(chan *Message, 1)
	timeout := time.Now().Add(c.config.Timeout)

	c.mu.Lock()
	c.pending[requestID] = &pendingRequest{
		ID:       requestID,
		Response: respChan,
		Timeout:  timeout,
	}
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, requestID)
		c.mu.Unlock()
	}()

	// Subscribe to reply subject
	replySubject := c.getReplySubject(requestID)
	sub, err := c.nats.Subscribe(replySubject, func(msg *nats.Msg) {
		c.handleResponse(msg.Data)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to reply subject: %w", err)
	}
	defer sub.Unsubscribe()

	// Send request with retries
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.config.RetryDelay * time.Duration(attempt))
		}

		if err := c.sendMessage(subject, msg); err != nil {
			lastErr = err
			continue
		}

		// Wait for response
		select {
		case responseMsg := <-respChan:
			if responseMsg.Error != nil {
				return nil, fmt.Errorf("service error: %s", responseMsg.Error.Message)
			}

			return &Response{
				ID:       responseMsg.ID,
				Data:     responseMsg.Data,
				Metadata: responseMsg.Metadata,
			}, nil

		case <-time.After(c.config.Timeout):
			lastErr = fmt.Errorf("request timeout")
			continue

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", c.config.MaxRetries+1, lastErr)
}

// Publish publishes an event message
func (c *Client) Publish(ctx context.Context, subject string, data map[string]interface{}, metadata map[string]interface{}) error {
	msg := &Message{
		ID:        generateMessageID(),
		Type:      MessageTypeEvent,
		Data:      data,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}

	return c.sendMessage(c.getSubject(subject), msg)
}

// Subscribe subscribes to a subject for streaming messages
func (c *Client) Subscribe(subject string, handler func(*Message)) (*nats.Subscription, error) {
	sub, err := c.nats.Subscribe(c.getSubject(subject, ">"), func(msg *nats.Msg) {
		var nrpcMsg Message
		if err := json.Unmarshal(msg.Data, &nrpcMsg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			return
		}
		handler(&nrpcMsg)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	c.mu.Lock()
	c.subscriptions[subject] = sub
	c.mu.Unlock()

	return sub, nil
}

// Stream creates a stream client for server-side streaming
func (c *Client) Stream(ctx context.Context, service, method string, data map[string]interface{}, metadata map[string]interface{}) (<-chan *Message, error) {
	requestID := generateMessageID()
	subject := c.getSubject(service, method)

	// Create stream request message
	msg := &Message{
		ID:       requestID,
		Type:     MessageTypeRequest,
		Service:  service,
		Method:   method,
		Data:     data,
		Metadata: metadata,
	}

	// Create response channel
	respChan := make(chan *Message, 100)

	// Subscribe to stream subject
	streamSubject := c.getStreamSubject(requestID)
	sub, err := c.nats.Subscribe(streamSubject, func(msg *nats.Msg) {
		var nrpcMsg Message
		if err := json.Unmarshal(msg.Data, &nrpcMsg); err != nil {
			log.Printf("Failed to unmarshal stream message: %v", err)
			return
		}

		select {
		case respChan <- &nrpcMsg:
		case <-ctx.Done():
		}

		// Close channel if stream ended
		if nrpcMsg.StreamEnd {
			close(respChan)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to stream: %w", err)
	}

	// Send stream request
	if err := c.sendMessage(subject, msg); err != nil {
		sub.Unsubscribe()
		return nil, fmt.Errorf("failed to send stream request: %w", err)
	}

	// Handle cleanup
	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
		close(respChan)
	}()

	return respChan, nil
}

// Ping sends a ping to the server
func (c *Client) Ping(ctx context.Context) error {
	response, err := c.Call(ctx, "control", "ping", nil, nil)
	if err != nil {
		return err
	}

	if response.Data == nil {
		return fmt.Errorf("invalid ping response")
	}

	return nil
}

// GetInfo gets server information
func (c *Client) GetInfo(ctx context.Context) (map[string]interface{}, error) {
	response, err := c.Call(ctx, "control", "info", nil, nil)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

// HealthCheck performs a health check
func (c *Client) HealthCheck(ctx context.Context) (map[string]interface{}, error) {
	response, err := c.Call(ctx, "control", "health", nil, nil)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

// handleResponse handles incoming response messages
func (c *Client) handleResponse(data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Failed to unmarshal response: %v", err)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if pending, exists := c.pending[msg.RequestID]; exists {
		select {
		case pending.Response <- &msg:
		default:
			log.Printf("Response channel full for request %s", msg.RequestID)
		}
	}
}

// sendMessage sends a message
func (c *Client) sendMessage(subject string, msg *Message) error {
	if c.closed {
		return fmt.Errorf("client is closed")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return c.nats.Publish(subject, data)
}

// getSubject returns a namespaced subject
func (c *Client) getSubject(parts ...string) string {
	allParts := []string{c.config.Namespace}
	allParts = append(allParts, parts...)
	return strings.Join(allParts, ".")
}

// getReplySubject returns a reply subject for a request
func (c *Client) getReplySubject(requestID string) string {
	return c.getSubject("reply", requestID)
}

// getStreamSubject returns a stream subject for a request
func (c *Client) getStreamSubject(requestID string) string {
	return c.getSubject("stream", requestID)
}

// cleanupPending cleans up expired pending requests
func (c *Client) cleanupPending() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for id, pending := range c.pending {
				if now.After(pending.Timeout) {
					close(pending.Response)
					delete(c.pending, id)
				}
			}
			c.mu.Unlock()
		}
	}
}

// Close closes the client
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	// Unsubscribe from all subscriptions
	for subject, sub := range c.subscriptions {
		if err := sub.Unsubscribe(); err != nil {
			log.Printf("Failed to unsubscribe from %s: %v", subject, err)
		}
	}

	// Close pending requests
	for _, pending := range c.pending {
		close(pending.Response)
	}

	c.subscriptions = make(map[string]*nats.Subscription)
	c.pending = make(map[string]*pendingRequest)

	return nil
}

// IsClosed checks if the client is closed
func (c *Client) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}
