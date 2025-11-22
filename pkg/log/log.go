package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/guileen/metabase/pkg/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Global logger instance
var (
	globalLogger *Logger
	once         sync.Once
)

// Logger provides a high-performance logging interface based on Go 1.21+ slog
type Logger struct {
	slogger     *slog.Logger
	config      *config.LoggingConfig
	mu          sync.RWMutex
	requestPool sync.Pool
	metrics     *Metrics
	tracing     *Tracing
}

// Metrics holds logging-related metrics
type Metrics struct {
	LogCounts   map[slog.Level]int64
	ErrorCounts map[string]int64
	TotalLogs   int64
	TotalBytes  int64
	LastReset   time.Time
	mu          sync.RWMutex
}

// Tracing holds request tracing information
type Tracing struct {
	ActiveRequests map[string]*RequestInfo
	RequestPool    sync.Pool
	mu             sync.RWMutex
}

// RequestInfo holds information about a request being traced
type RequestInfo struct {
	ID         string
	Method     string
	Path       string
	UserAgent  string
	RemoteAddr string
	UserID     string
	StartTime  time.Time
	Duration   time.Duration
	StatusCode int
	Tags       map[string]string
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Level     slog.Level
	Message   string
	Time      time.Time
	RequestID string
	UserID    string
	Component string
	Fields    map[string]interface{}
}

// Handler is a custom slog handler for enhanced functionality
type Handler struct {
	slog.Handler
	config      *config.LoggingConfig
	metrics     *Metrics
	writer      io.Writer
	requestPool sync.Pool
}

// NewLogger creates a new logger instance
func NewLogger(cfg *config.LoggingConfig) (*Logger, error) {
	if cfg == nil {
		cfg = &config.LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			File:       "./logs/metabase.log",
			MaxSize:    100,
			MaxAge:     7,
			MaxBackups: 3,
			Compress:   true,
			RequestID:  true,
			Caller:     false,
		}
	}

	logger := &Logger{
		config: cfg,
		metrics: &Metrics{
			LogCounts:   make(map[slog.Level]int64),
			ErrorCounts: make(map[string]int64),
			LastReset:   time.Now(),
		},
		tracing: &Tracing{
			ActiveRequests: make(map[string]*RequestInfo),
		},
	}

	// Setup request pool
	logger.requestPool = sync.Pool{
		New: func() interface{} {
			return &RequestInfo{
				Tags: make(map[string]string),
			}
		},
	}

	// Create writer
	var writer io.Writer
	switch cfg.Output {
	case "stdout":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	case "file":
		writer = &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSize,
			MaxAge:     cfg.MaxAge,
			MaxBackups: cfg.MaxBackups,
			Compress:   cfg.Compress,
		}
	default:
		return nil, fmt.Errorf("unsupported output: %s", cfg.Output)
	}

	// Parse log level
	level := parseLogLevel(cfg.Level)

	// Create handler options
	opts := &slog.HandlerOptions{
		Level: level,
	}

	if cfg.Caller {
		opts.AddSource = true
	}

	// Create base handler
	var baseHandler slog.Handler
	switch cfg.Format {
	case "json":
		baseHandler = slog.NewJSONHandler(writer, opts)
	case "text":
		baseHandler = slog.NewTextHandler(writer, opts)
	default:
		baseHandler = slog.NewJSONHandler(writer, opts)
	}

	// Wrap with custom handler
	customHandler := &Handler{
		Handler: baseHandler,
		config:  cfg,
		metrics: logger.metrics,
		writer:  writer,
	}

	logger.slogger = slog.New(customHandler)
	return logger, nil
}

// NewLoggerWithWriter creates a logger with a custom writer
func NewLoggerWithWriter(cfg *config.LoggingConfig, writer io.Writer) (*Logger, error) {
	if cfg == nil {
		cfg = &config.LoggingConfig{
			Level:     "info",
			Format:    "json",
			RequestID: true,
			Caller:    false,
		}
	}

	logger := &Logger{
		config: cfg,
		metrics: &Metrics{
			LogCounts:   make(map[slog.Level]int64),
			ErrorCounts: make(map[string]int64),
			LastReset:   time.Now(),
		},
		tracing: &Tracing{
			ActiveRequests: make(map[string]*RequestInfo),
		},
	}

	// Parse log level
	level := parseLogLevel(cfg.Level)

	// Create handler options
	opts := &slog.HandlerOptions{
		Level: level,
	}

	if cfg.Caller {
		opts.AddSource = true
	}

	// Create base handler
	var baseHandler slog.Handler
	switch cfg.Format {
	case "json":
		baseHandler = slog.NewJSONHandler(writer, opts)
	case "text":
		baseHandler = slog.NewTextHandler(writer, opts)
	default:
		baseHandler = slog.NewJSONHandler(writer, opts)
	}

	logger.slogger = slog.New(baseHandler)
	return logger, nil
}

// Initialize initializes the global logger
func Initialize(cfg *config.LoggingConfig) error {
	var err error
	once.Do(func() {
		globalLogger, err = NewLogger(cfg)
	})
	return err
}

// Get returns the global logger
func Get() *Logger {
	once.Do(func() {
		if globalLogger == nil {
			globalLogger, _ = NewLogger(nil)
		}
	})
	return globalLogger
}

// MustGet returns the global logger and panics if not initialized
func MustGet() *Logger {
	logger := Get()
	if logger == nil {
		panic("Logger not initialized")
	}
	return logger
}

// GetSlogger returns the underlying slog logger
func (l *Logger) GetSlogger() *slog.Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.slogger
}

// WithContext creates a new logger with context values
func (l *Logger) WithContext(ctx context.Context) *slog.Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	attrs := make([]slog.Attr, 0)

	// Add request ID if available
	if l.config.RequestID {
		if requestID := ctx.Value("request_id"); requestID != nil {
			attrs = append(attrs, slog.String("request_id", requestID.(string)))
		}
	}

	// Add user ID if available
	if userID := ctx.Value("user_id"); userID != nil {
		attrs = append(attrs, slog.String("user_id", userID.(string)))
	}

	// Add component if available
	if component := ctx.Value("component"); component != nil {
		attrs = append(attrs, slog.String("component", component.(string)))
	}

	// Add trace ID if available
	if traceID := ctx.Value("trace_id"); traceID != nil {
		attrs = append(attrs, slog.String("trace_id", traceID.(string)))
	}

	// Add span ID if available
	if spanID := ctx.Value("span_id"); spanID != nil {
		attrs = append(attrs, slog.String("span_id", spanID.(string)))
	}

	args := make([]any, len(attrs)*2)
	for i, attr := range attrs {
		args[i*2] = attr.Key
		args[i*2+1] = attr.Value.Any()
	}
	return l.slogger.With(args...)
}

// WithComponent adds a component attribute
func (l *Logger) WithComponent(component string) *slog.Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.slogger.With(slog.String("component", component))
}

// WithRequestID adds a request ID attribute
func (l *Logger) WithRequestID(requestID string) *slog.Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.slogger.With(slog.String("request_id", requestID))
}

// WithUserID adds a user ID attribute
func (l *Logger) WithUserID(userID string) *slog.Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.slogger.With(slog.String("user_id", userID))
}

// WithFields adds multiple fields
func (l *Logger) WithFields(fields map[string]interface{}) *slog.Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	attrs := make([]slog.Attr, 0, len(fields))
	for key, value := range fields {
		attrs = append(attrs, slog.Any(key, value))
	}

	args := make([]any, len(attrs)*2)
	for i, attr := range attrs {
		args[i*2] = attr.Key
		args[i*2+1] = attr.Value.Any()
	}
	return l.slogger.With(args...)
}

// Logging methods
func (l *Logger) Debug(msg string, args ...any) {
	l.mu.RLock()
	slogger := l.slogger
	l.mu.RUnlock()
	slogger.Debug(msg, args...)
}

func (l *Logger) Info(msg string, args ...any) {
	l.mu.RLock()
	slogger := l.slogger
	l.mu.RUnlock()
	slogger.Info(msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.mu.RLock()
	slogger := l.slogger
	l.mu.RUnlock()
	slogger.Warn(msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.mu.RLock()
	slogger := l.slogger
	l.mu.RUnlock()
	slogger.Error(msg, args...)
}

func (l *Logger) Fatal(msg string, args ...any) {
	l.mu.RLock()
	slogger := l.slogger
	l.mu.RUnlock()
	slogger.Error(msg, args...)
	os.Exit(1)
}

// Structured logging methods
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).Debug(msg, args...)
}

func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).Info(msg, args...)
}

func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).Warn(msg, args...)
}

func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).Error(msg, args...)
}

// Request tracing methods
func (l *Logger) StartRequest(ctx context.Context, requestID, method, path, userAgent, remoteAddr, userID string) context.Context {
	infoInterface := l.tracing.RequestPool.Get()
	info, ok := infoInterface.(*RequestInfo)
	if !ok {
		// Create new RequestInfo if pool is empty or contains wrong type
		info = &RequestInfo{
			Tags: make(map[string]string),
		}
	}
	info.ID = requestID
	info.Method = method
	info.Path = path
	info.UserAgent = userAgent
	info.RemoteAddr = remoteAddr
	info.UserID = userID
	info.StartTime = time.Now()
	info.Tags = make(map[string]string)

	l.tracing.mu.Lock()
	l.tracing.ActiveRequests[requestID] = info
	l.tracing.mu.Unlock()

	return context.WithValue(ctx, "request_id", requestID)
}

func (l *Logger) EndRequest(ctx context.Context, requestID string, statusCode int) {
	l.tracing.mu.Lock()
	defer l.tracing.mu.Unlock()

	info, exists := l.tracing.ActiveRequests[requestID]
	if !exists {
		return
	}

	info.Duration = time.Since(info.StartTime)
	info.StatusCode = statusCode

	// Log the request completion
	l.WithContext(ctx).Info("Request completed",
		slog.String("method", info.Method),
		slog.String("path", info.Path),
		slog.Int("status", statusCode),
		slog.Duration("duration", info.Duration),
		slog.String("remote_addr", info.RemoteAddr),
		slog.String("user_agent", info.UserAgent),
	)

	// Return request info to pool
	for k := range info.Tags {
		delete(info.Tags, k)
	}
	l.tracing.RequestPool.Put(info)
	delete(l.tracing.ActiveRequests, requestID)
}

func (l *Logger) AddRequestTag(ctx context.Context, key, value string) {
	requestID, ok := ctx.Value("request_id").(string)
	if !ok {
		return
	}

	l.tracing.mu.Lock()
	defer l.tracing.mu.Unlock()

	if info, exists := l.tracing.ActiveRequests[requestID]; exists {
		info.Tags[key] = value
	}
}

// Metrics methods
func (l *Logger) GetMetrics() *Metrics {
	l.metrics.mu.RLock()
	defer l.metrics.mu.RUnlock()

	// Return a copy
	metrics := &Metrics{
		LogCounts:   make(map[slog.Level]int64),
		ErrorCounts: make(map[string]int64),
		TotalLogs:   l.metrics.TotalLogs,
		TotalBytes:  l.metrics.TotalBytes,
		LastReset:   l.metrics.LastReset,
	}

	for k, v := range l.metrics.LogCounts {
		metrics.LogCounts[k] = v
	}

	for k, v := range l.metrics.ErrorCounts {
		metrics.ErrorCounts[k] = v
	}

	return metrics
}

func (l *Logger) ResetMetrics() {
	l.metrics.mu.Lock()
	defer l.metrics.mu.Unlock()

	l.metrics.LogCounts = make(map[slog.Level]int64)
	l.metrics.ErrorCounts = make(map[string]int64)
	l.metrics.TotalLogs = 0
	l.metrics.TotalBytes = 0
	l.metrics.LastReset = time.Now()
}

// Performance monitoring
func (l *Logger) GetActiveRequests() map[string]*RequestInfo {
	l.tracing.mu.RLock()
	defer l.tracing.mu.RUnlock()

	result := make(map[string]*RequestInfo)
	for k, v := range l.tracing.ActiveRequests {
		// Create a copy
		info := &RequestInfo{
			ID:         v.ID,
			Method:     v.Method,
			Path:       v.Path,
			UserAgent:  v.UserAgent,
			RemoteAddr: v.RemoteAddr,
			UserID:     v.UserID,
			StartTime:  v.StartTime,
			Tags:       make(map[string]string),
		}
		for tk, tv := range v.Tags {
			info.Tags[tk] = tv
		}
		result[k] = info
	}

	return result
}

// Custom handler implementation
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	// Update metrics
	h.metrics.mu.Lock()
	h.metrics.LogCounts[r.Level]++
	h.metrics.TotalLogs++
	h.metrics.TotalBytes += int64(len(r.Message))
	h.metrics.mu.Unlock()

	// Add error tracking for error levels
	if r.Level >= slog.LevelError {
		// Extract error type from attributes
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == "error" || a.Key == "err" {
				if err, ok := a.Value.Any().(error); ok {
					errorType := fmt.Sprintf("%T", err)
					h.metrics.mu.Lock()
					h.metrics.ErrorCounts[errorType]++
					h.metrics.mu.Unlock()
				}
			}
			return true
		})
	}

	// Add performance information
	if h.config.Caller {
		// Add goroutine ID for performance tracking
		buf := make([]byte, 64)
		buf = buf[:runtime.Stack(buf, false)]
		stackInfo := strings.Split(string(buf), "\n")
		if len(stackInfo) > 1 {
			r.AddAttrs(slog.String("goroutine", stackInfo[1]))
		}
	}

	// Call underlying handler
	return h.Handler.Handle(ctx, r)
}

// WithAttrs returns a new handler with the given attributes
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		Handler: h.Handler.WithAttrs(attrs),
		config:  h.config,
		metrics: h.metrics,
		writer:  h.writer,
	}
}

// WithGroup returns a new handler with the given group name
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		Handler: h.Handler.WithGroup(name),
		config:  h.config,
		metrics: h.metrics,
		writer:  h.writer,
	}
}

// Utility functions
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info", "information":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Global convenience functions
func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}

func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}

func Error(msg string, args ...any) {
	Get().Error(msg, args...)
}

func Fatal(msg string, args ...any) {
	Get().Fatal(msg, args...)
}

func DebugContext(ctx context.Context, msg string, args ...any) {
	Get().DebugContext(ctx, msg, args...)
}

func InfoContext(ctx context.Context, msg string, args ...any) {
	Get().InfoContext(ctx, msg, args...)
}

func WarnContext(ctx context.Context, msg string, args ...any) {
	Get().WarnContext(ctx, msg, args...)
}

func ErrorContext(ctx context.Context, msg string, args ...any) {
	Get().ErrorContext(ctx, msg, args...)
}

func WithComponent(component string) *slog.Logger {
	return Get().WithComponent(component)
}

func WithRequestID(requestID string) *slog.Logger {
	return Get().WithRequestID(requestID)
}

func WithUserID(userID string) *slog.Logger {
	return Get().WithUserID(userID)
}

func WithFields(fields map[string]interface{}) *slog.Logger {
	return Get().WithFields(fields)
}

func WithContext(ctx context.Context) *slog.Logger {
	return Get().WithContext(ctx)
}
