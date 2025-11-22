package log

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// Key types for context values
type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	UserIDKey    contextKey = "user_id"
	TraceIDKey   contextKey = "trace_id"
	SpanIDKey    contextKey = "span_id"
	ComponentKey contextKey = "component"
)

// Middleware provides HTTP logging middleware
type Middleware struct {
	logger  *Logger
	storage *LogStorage
	config  *MiddlewareConfig
}

// MiddlewareConfig holds configuration for the logging middleware
type MiddlewareConfig struct {
	// Whether to log request body
	LogRequestBody bool

	// Whether to log response body
	LogResponseBody bool

	// Max request body size to log (in bytes)
	MaxBodySize int64

	// Whether to generate trace IDs
	GenerateTraceID bool

	// Which status codes to log (all, success, error, client_error, server_error)
	LogStatus string

	// Skip logging for these paths
	SkipPaths []string

	// Additional fields to include in every log entry
	DefaultFields map[string]interface{}

	// Whether to measure response time
	MeasureResponseTime bool

	// Custom response time handler
	ResponseTimeHandler func(duration time.Duration)

	// Service name for identification
	ServiceName string

	// Whether to store logs in database
	StoreLogs bool
}

// DefaultMiddlewareConfig returns the default middleware configuration
func DefaultMiddlewareConfig() *MiddlewareConfig {
	return &MiddlewareConfig{
		LogRequestBody:      false,
		LogResponseBody:     false,
		MaxBodySize:         1024 * 1024, // 1MB
		GenerateTraceID:     true,
		LogStatus:           "all",
		SkipPaths:           []string{"/health", "/metrics", "/favicon.ico"},
		DefaultFields:       map[string]interface{}{},
		MeasureResponseTime: true,
		ServiceName:         "unknown",
		StoreLogs:           true,
	}
}

// NewMiddleware creates a new logging middleware
func NewMiddleware(logger *Logger) *Middleware {
	return NewMiddlewareWithStorage(logger, nil)
}

// NewMiddlewareWithStorage creates a new logging middleware with storage
func NewMiddlewareWithStorage(logger *Logger, storage *LogStorage) *Middleware {
	if logger == nil {
		logger = Get()
	}

	return &Middleware{
		logger:  logger,
		storage: storage,
		config:  DefaultMiddlewareConfig(),
	}
}

// NewMiddlewareWithConfig creates a new logging middleware with custom configuration
func NewMiddlewareWithConfig(logger *Logger, config *MiddlewareConfig) *Middleware {
	return NewMiddlewareWithConfigAndStorage(logger, config, nil)
}

// NewMiddlewareWithConfigAndStorage creates a new logging middleware with custom configuration and storage
func NewMiddlewareWithConfigAndStorage(logger *Logger, config *MiddlewareConfig, storage *LogStorage) *Middleware {
	if logger == nil {
		logger = Get()
	}

	if config == nil {
		config = DefaultMiddlewareConfig()
	}

	return &Middleware{
		logger:  logger,
		storage: storage,
		config:  config,
	}
}

// Middleware returns an HTTP middleware handler
func (m *Middleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate request ID
		requestID := generateRequestID()

		// Generate trace ID and span ID if enabled
		traceID := ""
		spanID := ""
		if m.config.GenerateTraceID {
			traceID = generateTraceID()
			spanID = generateSpanID()
		}

		// Create context with IDs
		ctx := r.Context()
		ctx = context.WithValue(ctx, RequestIDKey, requestID)
		if traceID != "" {
			ctx = context.WithValue(ctx, TraceIDKey, traceID)
			ctx = context.WithValue(ctx, SpanIDKey, spanID)
		}

		// Check if we should skip logging this path
		if m.shouldSkipPath(r.URL.Path) {
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Start request tracing
		startTime := time.Now()
		userAgent := r.UserAgent()
		remoteAddr := r.RemoteAddr
		userID := getUserIDFromRequest(r)

		ctx = m.logger.StartRequest(ctx, requestID, r.Method, r.URL.Path, userAgent, remoteAddr, userID)
		if userID != "" {
			ctx = context.WithValue(ctx, UserIDKey, userID)
		}

		// Create response writer wrapper to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default status code
		}

		// Log the request
		args := []any{}
		for k, v := range m.config.DefaultFields {
			args = append(args, k, v)
		}

		args = append(args,
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"remote_addr", remoteAddr,
			"user_agent", userAgent,
			"content_length", r.ContentLength,
			"content_type", r.Header.Get("Content-Type"),
		)

		// Add request body if configured
		if m.config.LogRequestBody && r.ContentLength > 0 && r.ContentLength <= m.config.MaxBodySize {
			body := readRequestBody(r)
			if body != "" {
				args = append(args, "request_body", body)
			}
		}

		// Log request start
		logMessage := "Request started"
		if m.config.StoreLogs && m.storage != nil {
			m.storeLogEntry(ctx, logMessage, "INFO", r.Method, r.URL.Path, 0, 0, remoteAddr, userAgent, args)
		}
		m.logger.WithContext(ctx).Info(logMessage, args...)

		// Call next handler
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		// Calculate duration
		duration := time.Since(startTime)

		// End request tracing
		m.logger.EndRequest(ctx, requestID, wrapped.statusCode)

		// Log the response if configured
		if m.shouldLogStatus(wrapped.statusCode) {
			args := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"duration_ms", duration.Milliseconds(),
			}

			// Add response body if configured
			if m.config.LogResponseBody && wrapped.responseBody != nil {
				args = append(args, "response_body", string(wrapped.responseBody))
			}

			// Log based on status code
			var logLevel string
			var logMessage string
			if wrapped.statusCode >= 400 {
				logLevel = "ERROR"
				logMessage = "Request failed"
				if m.config.StoreLogs && m.storage != nil {
					m.storeLogEntry(ctx, logMessage, logLevel, r.Method, r.URL.Path, int64(wrapped.statusCode), duration.Milliseconds(), remoteAddr, userAgent, args)
				}
				m.logger.WithContext(ctx).Error(logMessage, args...)
			} else {
				logLevel = "INFO"
				logMessage = "Request completed"
				if m.config.StoreLogs && m.storage != nil {
					m.storeLogEntry(ctx, logMessage, logLevel, r.Method, r.URL.Path, int64(wrapped.statusCode), duration.Milliseconds(), remoteAddr, userAgent, args)
				}
				m.logger.WithContext(ctx).Info(logMessage, args...)
			}
		}

		// Handle response time measurement
		if m.config.MeasureResponseTime {
			if m.config.ResponseTimeHandler != nil {
				m.config.ResponseTimeHandler(duration)
			} else {
				// Default response time logging for slow requests (>1s)
				if duration > time.Second {
					m.logger.WithContext(ctx).Warn("Slow request detected",
						"method", r.Method,
						"path", r.URL.Path,
						"duration_ms", duration.Milliseconds(),
					)
				}
			}
		}
	})
}

// RecoveryMiddleware creates a middleware that recovers from panics and logs them
func (m *Middleware) RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Get request context
				ctx := r.Context()

				// Log the panic
				m.logger.WithContext(ctx).Error("Request panic recovered",
					"panic", err,
					"method", r.Method,
					"path", r.URL.Path,
					"remote_addr", r.RemoteAddr,
				)

				// Send internal server error response
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// RequestIDMiddleware adds a request ID to the context and response headers
func (m *Middleware) RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := generateRequestID()

		// Add to context
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

		// Add to response header
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ComponentMiddleware adds a component identifier to the context
func (m *Middleware) ComponentMiddleware(component string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ComponentKey, component)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SetConfig updates the middleware configuration
func (m *Middleware) SetConfig(config *MiddlewareConfig) {
	if config != nil {
		m.config = config
	}
}

// GetConfig returns the current middleware configuration
func (m *Middleware) GetConfig() *MiddlewareConfig {
	return m.config
}

// Helper functions

func (m *Middleware) shouldSkipPath(path string) bool {
	for _, skipPath := range m.config.SkipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

func (m *Middleware) shouldLogStatus(statusCode int) bool {
	switch m.config.LogStatus {
	case "all":
		return true
	case "success":
		return statusCode >= 200 && statusCode < 300
	case "error":
		return statusCode >= 400
	case "client_error":
		return statusCode >= 400 && statusCode < 500
	case "server_error":
		return statusCode >= 500
	default:
		return true
	}
}

func generateRequestID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func generateTraceID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func generateSpanID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func getUserIDFromRequest(r *http.Request) string {
	// Try to get user ID from Authorization header or other means
	// This is a placeholder implementation
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		return userID
	}
	return ""
}

func readRequestBody(r *http.Request) string {
	// This is a placeholder - in a real implementation, you'd need to
	// use a request body wrapper to avoid consuming the body
	return ""
}

// responseWriter wraps http.ResponseWriter to capture status code and response body
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseBody []byte
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	// Store response body if needed
	if rw.responseBody == nil {
		rw.responseBody = make([]byte, 0)
	}
	rw.responseBody = append(rw.responseBody, data...)
	return rw.ResponseWriter.Write(data)
}

// Convenience functions for using the default logger

// RequestMiddleware returns a request logging middleware using the default logger
func RequestMiddleware() func(http.Handler) http.Handler {
	return NewMiddleware(Get()).Middleware
}

// RecoveryMiddleware returns a panic recovery middleware using the default logger
func Recovery() func(http.Handler) http.Handler {
	return NewMiddleware(Get()).RecoveryMiddleware
}

// RequestIDMiddleware returns a request ID middleware using the default logger
func RequestID() func(http.Handler) http.Handler {
	return NewMiddleware(Get()).RequestIDMiddleware
}

// ComponentMiddleware returns a component middleware using the default logger
func Component(component string) func(http.Handler) http.Handler {
	return NewMiddleware(Get()).ComponentMiddleware(component)
}

// ChiMiddleware returns Chi-compatible middleware for request logging
func ChiMiddleware() func(next http.Handler) http.Handler {
	return middleware.RequestLogger(&chiLogger{})
}

// chiLogger implements middleware.LogFormatter for Chi
type chiLogger struct{}

func (l *chiLogger) NewLogEntry(r *http.Request) middleware.LogEntry {
	requestID := generateRequestID()
	ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

	return &chiLogEntry{
		requestID: requestID,
		ctx:       ctx,
		logger:    Get(),
		start:     time.Now(),
		r:         r,
	}
}

type chiLogEntry struct {
	requestID string
	ctx       context.Context
	logger    *Logger
	start     time.Time
	r         *http.Request
}

func (e *chiLogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	fields := map[string]interface{}{
		"request_id":  e.requestID,
		"method":      e.r.Method,
		"path":        e.r.URL.Path,
		"status":      status,
		"bytes":       bytes,
		"duration_ms": elapsed.Milliseconds(),
		"remote_addr": e.r.RemoteAddr,
		"user_agent":  e.r.UserAgent(),
	}

	args := []any{}
	for k, v := range fields {
		args = append(args, k, v)
	}

	logger := e.logger.GetSlogger().With(args...)
	if status >= 400 {
		logger.Error("Request completed with error")
	} else {
		logger.Info("Request completed")
	}
}

func (e *chiLogEntry) Panic(v interface{}, stack []byte) {
	args := []any{
		"request_id", e.requestID,
		"method", e.r.Method,
		"path", e.r.URL.Path,
		"panic", v,
		"stack_trace", string(stack),
		"remote_addr", e.r.RemoteAddr,
		"user_agent", e.r.UserAgent(),
	}

	e.logger.Error("Request panic", args...)
}

// Context helper functions

// GetRequestID returns the request ID from the context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// GetUserID returns the user ID from the context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// GetTraceID returns the trace ID from the context
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// GetSpanID returns the span ID from the context
func GetSpanID(ctx context.Context) string {
	if spanID, ok := ctx.Value(SpanIDKey).(string); ok {
		return spanID
	}
	return ""
}

// GetComponent returns the component from the context
func GetComponent(ctx context.Context) string {
	if component, ok := ctx.Value(ComponentKey).(string); ok {
		return component
	}
	return ""
}

// storeLogEntry stores a log entry in the database
func (m *Middleware) storeLogEntry(ctx context.Context, message, level, method, path string, status, durationMs int64, remoteAddr, userAgent string, args []any) {
	if m.storage == nil {
		return
	}

	// Convert args to fields map
	fields := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok {
				fields[key] = args[i+1]
			}
		}
	}

	// Serialize fields to JSON
	fieldsJSON := ""
	if len(fields) > 0 {
		if bytes, err := json.Marshal(fields); err == nil {
			fieldsJSON = string(bytes)
		}
	}

	entry := &StoredLogEntry{
		Timestamp:  time.Now(),
		Level:      level,
		Message:    message,
		RequestID:  GetRequestID(ctx),
		UserID:     GetUserID(ctx),
		Component:  GetComponent(ctx),
		Service:    m.config.ServiceName,
		Method:     method,
		Path:       path,
		Status:     int(status),
		DurationMs: durationMs,
		RemoteAddr: remoteAddr,
		UserAgent:  userAgent,
		TraceID:    GetTraceID(ctx),
		SpanID:     GetSpanID(ctx),
		Fields:     fieldsJSON,
	}

	// Store asynchronously to avoid blocking the request
	go func() {
		if err := m.storage.StoreLog(entry); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to store log entry: %v\n", err)
		}
	}()
}
