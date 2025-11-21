package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/guileen/metabase/pkg/common/nrpc/embedded"
	"github.com/guileen/metabase/pkg/common/nrpc/middleware"
	"github.com/nats-io/nats.go"
)

// MessageType represents the type of NRPC message
type MessageType string

const (
	MessageTypeRequest  MessageType = "request"
	MessageTypeResponse MessageType = "response"
	MessageTypeError    MessageType = "error"
	MessageTypeEvent    MessageType = "event"
	MessageTypeStream   MessageType = "stream"
	MessageTypePing     MessageType = "ping"
	MessageTypePong     MessageType = "pong"
)

// Message represents an NRPC message
type Message struct {
	ID          string                 `json:"id"`
	Type        MessageType            `json:"type"`
	Service     string                 `json:"service"`
	Method      string                 `json:"method"`
	Data        map[string]interface{} `json:"data"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timestamp   time.Time              `json:"timestamp"`
	RequestID   string                 `json:"request_id,omitempty"`
	StreamIndex int64                  `json:"stream_index,omitempty"`
	StreamEnd   bool                   `json:"stream_end,omitempty"`
	Error       *ErrorInfo             `json:"error,omitempty"`
}

// ErrorInfo represents error information in a message
type ErrorInfo struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ServiceHandler represents a service handler interface
type ServiceHandler interface {
	Name() string
	Methods() map[string]MethodInfo
	Handle(ctx context.Context, req *Request) (*Response, error)
}

// MethodInfo represents method information
type MethodInfo struct {
	Name        string
	Description string
	InputType   reflect.Type
	OutputType  reflect.Type
	Streaming   bool
	Metadata    map[string]interface{}
}

// Request represents a service request
type Request struct {
	ID       string                 `json:"id"`
	Service  string                 `json:"service"`
	Method   string                 `json:"method"`
	Data     map[string]interface{} `json:"data"`
	Metadata map[string]interface{} `json:"metadata"`
	Context  context.Context        `json:"-"`
}

// Response represents a service response
type Response struct {
	ID       string                 `json:"id"`
	Data     map[string]interface{} `json:"data"`
	Metadata map[string]interface{} `json:"metadata"`
	Error    *ErrorInfo             `json:"error,omitempty"`
}

// Server represents an NRPC v2 server
type Server struct {
	nats          *embedded.EmbeddedNATS
	config        *Config
	handlers      map[string]ServiceHandler
	middleware    []middleware.Middleware
	subscriptions map[string]*nats.Subscription
	mu            sync.RWMutex
	started       bool
	startOnce     sync.Once
	ctx           context.Context
	cancel        context.CancelFunc
}

// Config represents server configuration
type Config struct {
	Name            string        `yaml:"name" json:"name"`
	Version         string        `yaml:"version" json:"version"`
	Namespace       string        `yaml:"namespace" json:"namespace"`
	EnableStreaming bool          `yaml:"enable_streaming" json:"enable_streaming"`
	EnableMetrics   bool          `yaml:"enable_metrics" json:"enable_metrics"`
	EnableTracing   bool          `yaml:"enable_tracing" json:"enable_tracing"`
	Timeout         time.Duration `yaml:"timeout" json:"timeout"`
	MaxConcurrency  int           `yaml:"max_concurrency" json:"max_concurrency"`
}

// NewServer creates a new NRPC v2 server
func NewServer(nats *embedded.EmbeddedNATS, config *Config) *Server {
	if config == nil {
		config = &Config{
			Name:            "metabase-nrpc-server",
			Version:         "2.0.0",
			Namespace:       "metabase",
			EnableStreaming: true,
			EnableMetrics:   true,
			EnableTracing:   false,
			Timeout:         30 * time.Second,
			MaxConcurrency:  1000,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		nats:          nats,
		config:        config,
		handlers:      make(map[string]ServiceHandler),
		middleware:    make([]middleware.Middleware, 0),
		subscriptions: make(map[string]*nats.Subscription),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// RegisterHandler registers a service handler
func (s *Server) RegisterHandler(handler ServiceHandler) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := handler.Name()
	if _, exists := s.handlers[name]; exists {
		return fmt.Errorf("handler %s already registered", name)
	}

	s.handlers[name] = handler
	log.Printf("Registered handler: %s", name)
	return nil
}

// Use adds middleware to the server
func (s *Server) Use(mw middleware.Middleware) {
	s.middleware = append(s.middleware, mw)
}

// Start starts the NRPC server
func (s *Server) Start() error {
	var startErr error
	s.startOnce.Do(func() {
		startErr = s.startServer()
	})
	return startErr
}

// startServer actually starts the server
func (s *Server) startServer() error {
	// Wait for NATS to be ready
	if err := s.nats.WaitReady(10 * time.Second); err != nil {
		return fmt.Errorf("NATS not ready: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Subscribe to all registered services
	for serviceName, handler := range s.handlers {
		if err := s.subscribeToService(serviceName, handler); err != nil {
			log.Printf("Failed to subscribe to service %s: %v", serviceName, err)
			return err
		}
	}

	// Subscribe to control subjects
	if err := s.subscribeToControl(); err != nil {
		return fmt.Errorf("failed to subscribe to control subjects: %w", err)
	}

	s.started = true
	log.Printf("NRPC v2 server started: %s v%s", s.config.Name, s.config.Version)
	return nil
}

// subscribeToService subscribes to a service
func (s *Server) subscribeToService(serviceName string, handler ServiceHandler) error {
	subject := s.getSubject(serviceName, ">")

	sub, err := s.nats.Subscribe(subject, func(msg *nats.Msg) {
		s.handleMessage(handler, msg)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}

	s.subscriptions[subject] = sub
	log.Printf("Subscribed to service: %s", subject)
	return nil
}

// subscribeToControl subscribes to control subjects
func (s *Server) subscribeToControl() error {
	controlSubjects := []string{
		s.getSubject("control", "ping"),
		s.getSubject("control", "info"),
		s.getSubject("control", "health"),
	}

	for _, subject := range controlSubjects {
		sub, err := s.nats.Subscribe(subject, func(msg *nats.Msg) {
			s.handleControlMessage(subject, msg)
		})
		if err != nil {
			return fmt.Errorf("failed to subscribe to control %s: %w", subject, err)
		}

		s.subscriptions[subject] = sub
	}

	return nil
}

// handleMessage handles incoming messages
func (s *Server) handleMessage(handler ServiceHandler, msg *nats.Msg) {
	// Parse message
	var nrpcMsg Message
	if err := json.Unmarshal(msg.Data, &nrpcMsg); err != nil {
		s.sendError(msg.Reply, "parse_error", "Failed to parse message", nil)
		return
	}

	// Create request
	req := &Request{
		ID:       nrpcMsg.ID,
		Service:  nrpcMsg.Service,
		Method:   nrpcMsg.Method,
		Data:     nrpcMsg.Data,
		Metadata: nrpcMsg.Metadata,
		Context:  s.createContext(&nrpcMsg, msg),
	}

	// Apply middleware chain
	handlerFunc := func(ctx context.Context, r *Request) (*Response, error) {
		return handler.Handle(ctx, r)
	}

	// Apply middleware in reverse order
	for i := len(s.middleware) - 1; i >= 0; i-- {
		mw := s.middleware[i]
		next := handlerFunc
		handlerFunc = func(ctx context.Context, r *Request) (*Response, error) {
			return mw.Handle(ctx, r, next)
		}
	}

	// Execute handler chain
	resp, err := handlerFunc(req.Context, req)
	if err != nil {
		s.sendErrorFromError(msg.Reply, nrpcMsg.ID, err)
		return
	}

	// Send response
	if msg.Reply != "" {
		responseMsg := &Message{
			ID:        resp.ID,
			Type:      MessageTypeResponse,
			Service:   resp.ID,
			Data:      resp.Data,
			Metadata:  resp.Metadata,
			Timestamp: time.Now(),
			RequestID: nrpcMsg.ID,
		}

		if err := s.sendMessage(msg.Reply, responseMsg); err != nil {
			log.Printf("Failed to send response: %v", err)
		}
	}
}

// handleControlMessage handles control messages
func (s *Server) handleControlMessage(subject string, msg *nats.Msg) {
	parts := strings.Split(subject, ".")
	if len(parts) < 4 {
		return
	}

	command := parts[3]
	var responseMsg *Message

	switch command {
	case "ping":
		responseMsg = &Message{
			ID:        generateMessageID(),
			Type:      MessageTypePong,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"pong": time.Now().Unix(),
			},
		}
	case "info":
		responseMsg = &Message{
			ID:        generateMessageID(),
			Type:      MessageTypeResponse,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"name":      s.config.Name,
				"version":   s.config.Version,
				"namespace": s.config.Namespace,
				"services":  s.getServiceInfo(),
				"started":   s.started,
				"timestamp": time.Now(),
			},
		}
	case "health":
		responseMsg = &Message{
			ID:        generateMessageID(),
			Type:      MessageTypeResponse,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"status":     "healthy",
				"nats_ready": s.nats.IsReady(),
				"timestamp":  time.Now(),
			},
		}
	}

	if responseMsg != nil && msg.Reply != "" {
		if err := s.sendMessage(msg.Reply, responseMsg); err != nil {
			log.Printf("Failed to send control response: %v", err)
		}
	}
}

// createContext creates a context with metadata
func (s *Server) createContext(msg *Message, natsMsg *nats.Msg) context.Context {
	ctx := s.ctx

	// Add timeout
	if s.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()
	}

	// Add metadata to context
	if msg.Metadata != nil {
		for key, value := range msg.Metadata {
			ctx = context.WithValue(ctx, contextKey(key), value)
		}
	}

	// Add NATS message info
	ctx = context.WithValue(ctx, contextKey("nats_subject"), natsMsg.Subject)
	ctx = context.WithValue(ctx, contextKey("nats_reply"), natsMsg.Reply)

	return ctx
}

// sendMessage sends a message
func (s *Server) sendMessage(subject string, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return s.nats.Publish(subject, data)
}

// sendError sends an error message
func (s *Server) sendError(reply, code, message string, details map[string]interface{}) {
	errorMsg := &Message{
		ID:        generateMessageID(),
		Type:      MessageTypeError,
		Timestamp: time.Now(),
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	}

	if reply != "" {
		s.sendMessage(reply, errorMsg)
	}
}

// sendErrorFromError sends an error message from an error
func (s *Server) sendErrorFromError(reply, requestID string, err error) {
	errorInfo := &ErrorInfo{
		Code:    "internal_error",
		Message: err.Error(),
	}

	errorMsg := &Message{
		ID:        generateMessageID(),
		Type:      MessageTypeError,
		Timestamp: time.Now(),
		RequestID: requestID,
		Error:     errorInfo,
	}

	if reply != "" {
		s.sendMessage(reply, errorMsg)
	}
}

// getSubject returns a namespaced subject
func (s *Server) getSubject(parts ...string) string {
	allParts := []string{s.config.Namespace}
	allParts = append(allParts, parts...)
	return strings.Join(allParts, ".")
}

// getServiceInfo returns information about registered services
func (s *Server) getServiceInfo() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	services := make(map[string]interface{})
	for name, handler := range s.handlers {
		methods := make(map[string]interface{})
		for methodName, methodInfo := range handler.Methods() {
			methods[methodName] = map[string]interface{}{
				"name":        methodInfo.Name,
				"description": methodInfo.Description,
				"streaming":   methodInfo.Streaming,
				"metadata":    methodInfo.Metadata,
			}
		}

		services[name] = map[string]interface{}{
			"name":    name,
			"methods": methods,
		}
	}

	return services
}

// Stop stops the server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil
	}

	// Cancel context
	s.cancel()

	// Unsubscribe from all subjects
	for subject, sub := range s.subscriptions {
		if err := sub.Unsubscribe(); err != nil {
			log.Printf("Failed to unsubscribe from %s: %v", subject, err)
		}
	}

	s.subscriptions = make(map[string]*nats.Subscription)
	s.started = false

	log.Printf("NRPC v2 server stopped")
	return nil
}

// IsStarted checks if the server is started
func (s *Server) IsStarted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.started
}

// GetHandlers returns registered handlers
func (s *Server) GetHandlers() map[string]ServiceHandler {
	s.mu.RLock()
	defer s.mu.RUnlock()

	handlers := make(map[string]ServiceHandler)
	for name, handler := range s.handlers {
		handlers[name] = handler
	}
	return handlers
}

// Context key type for context values
type contextKey string

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}
