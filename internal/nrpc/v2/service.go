package v2

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

// Service represents a service that can be registered with the NRPC server
type Service struct {
	name    string
	methods map[string]*Method
	handler ServiceHandler
}

// Method represents a service method
type Method struct {
	Name        string
	Description string
	InputType   reflect.Type
	OutputType  reflect.Type
	Streaming   bool
	Handler     MethodHandler
}

// MethodHandler represents a method handler function
type MethodHandler func(ctx context.Context, req *Request) (*Response, error)

// NewService creates a new service
func NewService(name string) *Service {
	return &Service{
		name:    name,
		methods: make(map[string]*Method),
	}
}

// Name returns the service name
func (s *Service) Name() string {
	return s.name
}

// Methods returns all methods
func (s *Service) Methods() map[string]MethodInfo {
	methods := make(map[string]MethodInfo)
	for name, method := range s.methods {
		methods[name] = MethodInfo{
			Name:        method.Name,
			Description: method.Description,
			InputType:   method.InputType,
			OutputType:  method.OutputType,
			Streaming:   method.Streaming,
			Metadata:    make(map[string]interface{}),
		}
	}
	return methods
}

// Handle handles a request
func (s *Service) Handle(ctx context.Context, req *Request) (*Response, error) {
	method, exists := s.methods[req.Method]
	if !exists {
		return nil, fmt.Errorf("method %s not found in service %s", req.Method, req.Service)
	}

	return method.Handler(ctx, req)
}

// RegisterMethod registers a method
func (s *Service) RegisterMethod(method *Method) error {
	if _, exists := s.methods[method.Name]; exists {
		return fmt.Errorf("method %s already registered", method.Name)
	}

	s.methods[method.Name] = method
	return nil
}

// RegisterMethodFunc registers a method from a function
func (s *Service) RegisterMethodFunc(name, description string, handler MethodHandler) error {
	method := &Method{
		Name:        name,
		Description: description,
		Handler:     handler,
	}

	return s.RegisterMethod(method)
}

// RegisterHandler registers a service handler
func (s *Service) RegisterHandler(handler ServiceHandler) {
	s.handler = handler
}

// GetMethod returns a method by name
func (s *Service) GetMethod(name string) (*Method, bool) {
	method, exists := s.methods[name]
	return method, exists
}

// ListMethods returns a list of method names
func (s *Service) ListMethods() []string {
	methods := make([]string, 0, len(s.methods))
	for name := range s.methods {
		methods = append(methods, name)
	}
	return methods
}

// ServiceRegistry represents a registry of services
type ServiceRegistry struct {
	services map[string]*Service
	mu       sync.RWMutex
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]*Service),
	}
}

// Register registers a service
func (r *ServiceRegistry) Register(service *Service) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.services[service.name]; exists {
		return fmt.Errorf("service %s already registered", service.name)
	}

	r.services[service.name] = service
	return nil
}

// Get returns a service by name
func (r *ServiceRegistry) Get(name string) (*Service, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	service, exists := r.services[name]
	return service, exists
}

// List returns a list of service names
func (r *ServiceRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]string, 0, len(r.services))
	for name := range r.services {
		services = append(services, name)
	}
	return services
}

// Unregister unregisters a service
func (r *ServiceRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.services, name)
}

// Size returns the number of registered services
func (r *ServiceRegistry) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.services)
}

// ServiceBuilder helps build services
type ServiceBuilder struct {
	service *Service
}

// NewServiceBuilder creates a new service builder
func NewServiceBuilder(name string) *ServiceBuilder {
	service := NewService(name)
	return &ServiceBuilder{service: service}
}

// Method adds a method to the service
func (b *ServiceBuilder) Method(name, description string, handler MethodHandler) *ServiceBuilder {
	method := &Method{
		Name:        name,
		Description: description,
		Handler:     handler,
	}
	b.service.RegisterMethod(method)
	return b
}

// StreamingMethod adds a streaming method to the service
func (b *ServiceBuilder) StreamingMethod(name, description string, handler MethodHandler) *ServiceBuilder {
	method := &Method{
		Name:        name,
		Description: description,
		Streaming:   true,
		Handler:     handler,
	}
	b.service.RegisterMethod(method)
	return b
}

// Build builds the service
func (b *ServiceBuilder) Build() *Service {
	return b.service
}

// EchoService example service
type EchoService struct{}

// NewEchoService creates a new echo service
func NewEchoService() *Service {
	builder := NewServiceBuilder("echo")

	builder.Method("echo", "Echoes back the input message", func(ctx context.Context, req *Request) (*Response, error) {
		return &Response{
			ID:   req.ID,
			Data: req.Data,
			Metadata: map[string]interface{}{
				"echoed_at": "now",
			},
		}, nil
	})

	builder.Method("ping", "Simple ping test", func(ctx context.Context, req *Request) (*Response, error) {
		return &Response{
			ID:   req.ID,
			Data: map[string]interface{}{"pong": true},
		}, nil
	})

	return builder.Build()
}

// HealthService example service
type HealthService struct{}

// NewHealthService creates a new health service
func NewHealthService() *Service {
	builder := NewServiceBuilder("health")

	builder.Method("check", "Health check endpoint", func(ctx context.Context, req *Request) (*Response, error) {
		return &Response{
			ID:   req.ID,
			Data: map[string]interface{}{
				"status":    "healthy",
				"timestamp": "now",
			},
		}, nil
	})

	builder.Method("version", "Returns version information", func(ctx context.Context, req *Request) (*Response, error) {
		return &Response{
			ID:   req.ID,
			Data: map[string]interface{}{
				"version":    "2.0.0",
				"build_time": "now",
			},
		}, nil
	})

	return builder.Build()
}

// AuthService example service
type AuthService struct{}

// NewAuthService creates a new auth service
func NewAuthService() *Service {
	builder := NewServiceBuilder("auth")

	builder.Method("login", "Authenticate user", func(ctx context.Context, req *Request) (*Response, error) {
		// Extract credentials from request data
		email, _ := req.Data["email"].(string)
		password, _ := req.Data["password"].(string)

		// Simple validation (in real implementation, verify against database)
		if email == "" || password == "" {
			return nil, fmt.Errorf("email and password required")
		}

		// Return mock token
		return &Response{
			ID:   req.ID,
			Data: map[string]interface{}{
				"token":     "mock_jwt_token",
				"user_id":   "user_123",
				"expires":   "2024-01-01T00:00:00Z",
				"tenant_id": "tenant_123",
			},
		}, nil
	})

	builder.Method("validate", "Validate token", func(ctx context.Context, req *Request) (*Response, error) {
		token, _ := req.Data["token"].(string)
		if token == "" {
			return nil, fmt.Errorf("token required")
		}

		// Mock validation
		return &Response{
			ID:   req.ID,
			Data: map[string]interface{}{
				"valid":   true,
				"user_id": "user_123",
			},
		}, nil
	})

	return builder.Build()
}

// DataService example service
type DataService struct{}

// NewDataService creates a new data service
func NewDataService() *Service {
	builder := NewServiceBuilder("data")

	builder.Method("create", "Create a record", func(ctx context.Context, req *Request) (*Response, error) {
		// Extract table and data from request
		table, _ := req.Data["table"].(string)
		data, _ := req.Data["data"].(map[string]interface{})

		if table == "" || data == nil {
			return nil, fmt.Errorf("table and data required")
		}

		// Mock record creation
		recordID := "rec_" + fmt.Sprintf("%d", len(data))

		return &Response{
			ID:   req.ID,
			Data: map[string]interface{}{
				"id":         recordID,
				"table":      table,
				"created_at": "now",
			},
		}, nil
	})

	builder.Method("query", "Query records", func(ctx context.Context, req *Request) (*Response, error) {
		table, _ := req.Data["table"].(string)
		limit, _ := req.Data["limit"].(int)

		if table == "" {
			return nil, fmt.Errorf("table required")
		}

		if limit <= 0 {
			limit = 10
		}

		// Mock query results
		records := make([]map[string]interface{}, limit)
		for i := 0; i < limit; i++ {
			records[i] = map[string]interface{}{
				"id":    fmt.Sprintf("rec_%d", i),
				"table": table,
				"data":  map[string]interface{}{"field": "value"},
			}
		}

		return &Response{
			ID: req.ID,
			Data: map[string]interface{}{
				"records": records,
				"total":   limit,
			},
		}, nil
	})

	return builder.Build()
}