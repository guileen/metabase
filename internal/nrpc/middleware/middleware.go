package middleware

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Request represents an NRPC request (copy to avoid circular imports)
type Request struct {
	ID        string                 `json:"id"`
	Service   string                 `json:"service"`
	Method    string                 `json:"method"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// Response represents an NRPC response (copy to avoid circular imports)
type Response struct {
	ID        string                 `json:"id"`
	Service   string                 `json:"service"`
	Method    string                 `json:"method"`
	Data      interface{}            `json:"data,omitempty"`
	Error     *Error                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// Error represents an NRPC error (copy to avoid circular imports)
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Middleware represents a middleware function
type Middleware interface {
	Handle(ctx context.Context, req *Request, next NextFunc) (*Response, error)
}

// NextFunc represents the next function in the middleware chain
type NextFunc func(ctx context.Context, req *Request) (*Response, error)

// Chain creates a middleware chain
func Chain(middlewares ...Middleware) Middleware {
	return &chainMiddleware{middlewares: middlewares}
}

type chainMiddleware struct {
	middlewares []Middleware
}

func (c *chainMiddleware) Handle(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
	// Build the chain
	var handler NextFunc = next
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		mw := c.middlewares[i]
		nextHandler := handler
		handler = func(ctx context.Context, req *Request) (*Response, error) {
			return mw.Handle(ctx, req, nextHandler)
		}
	}

	return handler(ctx, req)
}

// LoggingMiddleware logs requests and responses
type LoggingMiddleware struct {
	Logger *log.Logger
}

func NewLoggingMiddleware(logger *log.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{Logger: logger}
}

func (l *LoggingMiddleware) Handle(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
	start := time.Now()

	if l.Logger != nil {
		l.Logger.Printf("Request: %s.%s (ID: %s)", req.Service, req.Method, req.ID)
	}

	resp, err := next(ctx, req)
	duration := time.Since(start)

	if l.Logger != nil {
		if err != nil {
			l.Logger.Printf("Request failed: %s.%s (ID: %s) - %v (%v)", req.Service, req.Method, req.ID, err, duration)
		} else {
			l.Logger.Printf("Request completed: %s.%s (ID: %s) (%v)", req.Service, req.Method, req.ID, duration)
		}
	}

	return resp, err
}

// MetricsMiddleware collects request metrics
type MetricsMiddleware struct {
	RequestCount    map[string]int64
	RequestDuration map[string]time.Duration
	ErrorCount      map[string]int64
}

func NewMetricsMiddleware() *MetricsMiddleware {
	return &MetricsMiddleware{
		RequestCount:    make(map[string]int64),
		RequestDuration: make(map[string]time.Duration),
		ErrorCount:      make(map[string]int64),
	}
}

func (m *MetricsMiddleware) Handle(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
	key := fmt.Sprintf("%s.%s", req.Service, req.Method)
	start := time.Now()

	resp, err := next(ctx, req)
	duration := time.Since(start)

	m.RequestCount[key]++
	m.RequestDuration[key] += duration

	if err != nil {
		m.ErrorCount[key]++
	}

	return resp, err
}

// TimeoutMiddleware adds timeout to requests
type TimeoutMiddleware struct {
	Timeout time.Duration
}

func NewTimeoutMiddleware(timeout time.Duration) *TimeoutMiddleware {
	return &TimeoutMiddleware{Timeout: timeout}
}

func (t *TimeoutMiddleware) Handle(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
	if t.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.Timeout)
		defer cancel()
	}

	return next(ctx, req)
}

// AuthMiddleware handles authentication
type AuthMiddleware struct {
	TokenValidator func(token string) (map[string]interface{}, error)
}

func NewAuthMiddleware(tokenValidator func(string) (map[string]interface{}, error)) *AuthMiddleware {
	return &AuthMiddleware{TokenValidator: tokenValidator}
}

func (a *AuthMiddleware) Handle(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
	// Get token from metadata
	token, ok := req.Metadata["authorization"].(string)
	if !ok {
		return nil, fmt.Errorf("authorization token required")
	}

	// Validate token
	claims, err := a.TokenValidator(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Add claims to context
	for key, value := range claims {
		ctx = context.WithValue(ctx, contextKey(key), value)
	}

	return next(ctx, req)
}

// RateLimitMiddleware implements rate limiting
type RateLimitMiddleware struct {
	Requests map[string]int
	Limit    int
	Window   time.Duration
}

func NewRateLimitMiddleware(limit int, window time.Duration) *RateLimitMiddleware {
	// Start cleanup goroutine
	rl := &RateLimitMiddleware{
		Requests: make(map[string]int),
		Limit:    limit,
		Window:   window,
	}
	go rl.cleanup()
	return rl
}

func (r *RateLimitMiddleware) Handle(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
	clientID := getClientID(req)

	r.Requests[clientID]++
	if r.Requests[clientID] > r.Limit {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	return next(ctx, req)
}

func (r *RateLimitMiddleware) cleanup() {
	ticker := time.NewTicker(r.Window)
	defer ticker.Stop()

	for range ticker.C {
		r.Requests = make(map[string]int)
	}
}

// ValidationMiddleware validates requests
type ValidationMiddleware struct {
	Validator func(req *Request) error
}

func NewValidationMiddleware(validator func(*Request) error) *ValidationMiddleware {
	return &ValidationMiddleware{Validator: validator}
}

func (v *ValidationMiddleware) Handle(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
	if v.Validator != nil {
		if err := v.Validator(req); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	return next(ctx, req)
}

// CircuitBreakerMiddleware implements circuit breaker pattern
type CircuitBreakerMiddleware struct {
	MaxFailures  int
	ResetTimeout time.Duration
	state        circuitState
	failures     int
	lastFailTime time.Time
}

type circuitState int

const (
	StateClosed circuitState = iota
	StateOpen
	StateHalfOpen
)

func NewCircuitBreakerMiddleware(maxFailures int, resetTimeout time.Duration) *CircuitBreakerMiddleware {
	return &CircuitBreakerMiddleware{
		MaxFailures:  maxFailures,
		ResetTimeout: resetTimeout,
		state:        StateClosed,
	}
}

func (cb *CircuitBreakerMiddleware) Handle(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
	if cb.state == StateOpen {
		if time.Since(cb.lastFailTime) > cb.ResetTimeout {
			cb.state = StateHalfOpen
		} else {
			return nil, fmt.Errorf("circuit breaker is open")
		}
	}

	resp, err := next(ctx, req)

	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		if cb.failures >= cb.MaxFailures {
			cb.state = StateOpen
		}
	} else {
		if cb.state == StateHalfOpen {
			cb.state = StateClosed
		}
		cb.failures = 0
	}

	return resp, err
}

// RecoveryMiddleware recovers from panics
type RecoveryMiddleware struct {
	Logger *log.Logger
}

func NewRecoveryMiddleware(logger *log.Logger) *RecoveryMiddleware {
	return &RecoveryMiddleware{Logger: logger}
}

func (r *RecoveryMiddleware) Handle(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
	defer func() {
		if err := recover(); err != nil {
			if r.Logger != nil {
				r.Logger.Printf("Panic recovered in %s.%s: %v", req.Service, req.Method, err)
			}
		}
	}()

	return next(ctx, req)
}

// Helper functions

type contextKey string

func getClientID(req *Request) string {
	if clientID, ok := req.Metadata["client_id"].(string); ok {
		return clientID
	}
	return "anonymous"
}