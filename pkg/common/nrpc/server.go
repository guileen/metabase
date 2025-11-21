package nrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// MessageType represents the type of NRPC message
type MessageType string

const (
	MessageTypeRequest  MessageType = "request"
	MessageTypeResponse MessageType = "response"
	MessageTypeTask     MessageType = "task"
	MessageTypeResult   MessageType = "result"
)

// Message represents an NRPC message
type Message struct {
	ID         string                 `json:"id"`
	Type       MessageType            `json:"type"`
	Service    string                 `json:"service"`
	Method     string                 `json:"method"`
	Data       map[string]interface{} `json:"data"`
	Timestamp  time.Time              `json:"timestamp"`
	ReplyTo    string                 `json:"reply_to,omitempty"`
	RetryCount int                    `json:"retry_count"`
	MaxRetries int                    `json:"max_retries"`
	Delay      time.Duration          `json:"delay,omitempty"`
	Timeout    time.Duration          `json:"timeout,omitempty"`
}

// Handler represents an NRPC service handler
type Handler func(ctx context.Context, msg *Message) (*Message, error)

// Service represents an NRPC service
type Service struct {
	name    string
	handler map[string]Handler
}

// Config represents NRPC configuration
type Config struct {
	NATSURL        string        `json:"nats_url"`
	MaxReconnects  int           `json:"max_reconnects"`
	ReconnectWait  time.Duration `json:"reconnect_wait"`
	DefaultTimeout time.Duration `json:"default_timeout"`
	MaxRetries     int           `json:"max_retries"`
	RequestQueue   string        `json:"request_queue"`
	TaskQueue      string        `json:"task_queue"`
	ResultQueue    string        `json:"result_queue"`
}

// Client represents an NRPC client
type Client struct {
	conn   *nats.Conn
	config *Config
}

// Server represents an NRPC server
type Server struct {
	conn     *nats.Conn
	config   *Config
	services map[string]*Service
}

// NewConfig creates a new NRPC configuration with defaults
func NewConfig() *Config {
	return &Config{
		NATSURL:        nats.DefaultURL,
		MaxReconnects:  5,
		ReconnectWait:  2 * time.Second,
		DefaultTimeout: 30 * time.Second,
		MaxRetries:     3,
		RequestQueue:   "nrpc.requests",
		TaskQueue:      "nrpc.tasks",
		ResultQueue:    "nrpc.results",
	}
}

// NewServer creates a new NRPC server
func NewServer(config *Config) (*Server, error) {
	conn, err := nats.Connect(config.NATSURL,
		nats.MaxReconnects(config.MaxReconnects),
		nats.ReconnectWait(config.ReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("NRPC server disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NRPC server reconnected to %s", nc.ConnectedUrl())
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &Server{
		conn:     conn,
		config:   config,
		services: make(map[string]*Service),
	}, nil
}

// NewClient creates a new NRPC client
func NewClient(config *Config) (*Client, error) {
	conn, err := nats.Connect(config.NATSURL,
		nats.MaxReconnects(config.MaxReconnects),
		nats.ReconnectWait(config.ReconnectWait),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &Client{
		conn:   conn,
		config: config,
	}, nil
}

// RegisterService registers a service with the server
func (s *Server) RegisterService(service *Service) error {
	if _, exists := s.services[service.name]; exists {
		return fmt.Errorf("service %s already registered", service.name)
	}

	s.services[service.name] = service

	// Subscribe to service queue
	subject := fmt.Sprintf("nrpc.%s.*", service.name)
	_, err := s.conn.Subscribe(subject, func(msg *nats.Msg) {
		s.handleMessage(msg, service)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}

	// Queue subscription is set when using QueueSubscribe

	log.Printf("NRPC: Registered service %s on subject %s", service.name, subject)
	return nil
}

// handleMessage processes incoming messages
func (s *Server) handleMessage(natsMsg *nats.Msg, service *Service) {
	var msg Message
	if err := json.Unmarshal(natsMsg.Data, &msg); err != nil {
		log.Printf("NRPC: Failed to unmarshal message: %v", err)
		return
	}

	// Extract method from subject
	subjectParts := natsMsg.Subject
	if len(subjectParts) < 3 {
		log.Printf("NRPC: Invalid subject format: %s", subjectParts)
		return
	}

	// Handle message in goroutine for non-blocking processing
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), s.config.DefaultTimeout)
		defer cancel()

		handler, exists := service.handler[msg.Method]
		if !exists {
			log.Printf("NRPC: Method %s not found in service %s", msg.Method, service.name)
			return
		}

		response, err := handler(ctx, &msg)
		if err != nil {
			log.Printf("NRPC: Handler error: %v", err)
			response = &Message{
				ID:        msg.ID,
				Type:      MessageTypeResponse,
				Service:   service.name,
				Method:    msg.Method,
				Data:      map[string]interface{}{"error": err.Error()},
				Timestamp: time.Now(),
			}
		}

		// Send response if there's a reply-to subject
		if msg.ReplyTo != "" {
			responseData, _ := json.Marshal(response)
			s.conn.Publish(msg.ReplyTo, responseData)
		}
	}()
}

// Call makes an RPC call to a service
func (c *Client) Call(ctx context.Context, service, method string, data map[string]interface{}) (*Message, error) {
	msg := &Message{
		ID:        generateMessageID(),
		Type:      MessageTypeRequest,
		Service:   service,
		Method:    method,
		Data:      data,
		Timestamp: time.Now(),
		Timeout:   c.config.DefaultTimeout,
	}

	subject := fmt.Sprintf("nrpc.%s.%s", service, method)

	// Create inbox for response
	inbox := nats.NewInbox()
	msg.ReplyTo = inbox

	// Subscribe to response
	responseChan := make(chan *Message, 1)
	sub, err := c.conn.Subscribe(inbox, func(m *nats.Msg) {
		var response Message
		if err := json.Unmarshal(m.Data, &response); err != nil {
			log.Printf("NRPC: Failed to unmarshal response: %v", err)
			return
		}
		responseChan <- &response
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to response: %w", err)
	}
	defer sub.Unsubscribe()

	// Send request
	requestData, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if err := c.conn.Publish(subject, requestData); err != nil {
		return nil, fmt.Errorf("failed to publish request: %w", err)
	}

	// Wait for response with timeout
	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(msg.Timeout):
		return nil, fmt.Errorf("request timeout after %v", msg.Timeout)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// PublishTask publishes a task to the task queue
func (c *Client) PublishTask(ctx context.Context, service, method string, data map[string]interface{}, delay time.Duration) error {
	msg := &Message{
		ID:         generateMessageID(),
		Type:       MessageTypeTask,
		Service:    service,
		Method:     method,
		Data:       data,
		Timestamp:  time.Now(),
		MaxRetries: c.config.MaxRetries,
		Delay:      delay,
	}

	taskData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// For delayed messaging, we would need to use NATS JetStream
	// For now, just publish immediately
	return c.conn.Publish(c.config.TaskQueue, taskData)
}

// SubscribeTasks subscribes to tasks from the task queue
func (s *Server) SubscribeTasks(service *Service) error {
	_, err := s.conn.QueueSubscribe(s.config.TaskQueue, "workers", func(msg *nats.Msg) {
		var taskMsg Message
		if err := json.Unmarshal(msg.Data, &taskMsg); err != nil {
			log.Printf("NRPC: Failed to unmarshal task: %v", err)
			return
		}

		// Check if task is for this service
		if taskMsg.Service != service.name {
			return
		}

		// Handle task with retry logic
		go s.handleTask(&taskMsg, service)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to tasks: %w", err)
	}

	log.Printf("NRPC: Service %s subscribed to task queue", service.name)
	return nil
}

// handleTask processes a task with retry logic
func (s *Server) handleTask(task *Message, service *Service) {
	// Apply delay if specified
	if task.Delay > 0 {
		time.Sleep(task.Delay)
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.config.DefaultTimeout)
	defer cancel()

	handler, exists := service.handler[task.Method]
	if !exists {
		log.Printf("NRPC: Task method %s not found in service %s", task.Method, service.name)
		return
	}

	_, err := handler(ctx, task)
	if err != nil {
		log.Printf("NRPC: Task handler error: %v", err)

		// Retry logic
		if task.RetryCount < task.MaxRetries {
			task.RetryCount++
			task.Timestamp = time.Now()

			// Exponential backoff
			delay := time.Duration(task.RetryCount) * time.Second

			log.Printf("NRPC: Retrying task %s (attempt %d/%d) in %v",
				task.ID, task.RetryCount, task.MaxRetries, delay)

			time.Sleep(delay)

			// Republish task
			taskData, _ := json.Marshal(task)
			s.conn.Publish(s.config.TaskQueue, taskData)
		} else {
			log.Printf("NRPC: Task %s failed after %d retries", task.ID, task.MaxRetries)
		}
	} else {
		log.Printf("NRPC: Task %s completed successfully", task.ID)
	}
}

// NewService creates a new service
func NewService(name string) *Service {
	return &Service{
		name:    name,
		handler: make(map[string]Handler),
	}
}

// RegisterHandler registers a handler for a method
func (s *Service) RegisterHandler(method string, handler Handler) {
	s.handler[method] = handler
	log.Printf("NRPC: Registered handler %s for service %s", method, s.name)
}

// Close closes the NRPC connection
func (s *Server) Close() {
	if s.conn != nil {
		s.conn.Close()
	}
}

// Close closes the NRPC client connection
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// GetStats returns connection statistics
func (s *Server) GetStats() map[string]interface{} {
	stats := s.conn.Stats()
	return map[string]interface{}{
		"in_msgs":    stats.InMsgs,
		"out_msgs":   stats.OutMsgs,
		"in_bytes":   stats.InBytes,
		"out_bytes":  stats.OutBytes,
		"reconnects": stats.Reconnects,
	}
}

// GetStats returns connection statistics for client
func (c *Client) GetStats() map[string]interface{} {
	stats := c.conn.Stats()
	return map[string]interface{}{
		"in_msgs":    stats.InMsgs,
		"out_msgs":   stats.OutMsgs,
		"in_bytes":   stats.InBytes,
		"out_bytes":  stats.OutBytes,
		"reconnects": stats.Reconnects,
	}
}
