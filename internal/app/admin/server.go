package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/guileen/metabase/pkg/client"
	"github.com/guileen/metabase/pkg/config"
	"github.com/guileen/metabase/pkg/log"
)

// Config represents the admin server configuration
type Config struct {
	Host           string                `json:"host"`
	Port           string                `json:"port"`
	DevMode        bool                  `json:"dev_mode"`
	StaticFiles    string                `json:"static_files"`
	EnableRealtime bool                  `json:"enable_realtime"`
	SessionTimeout time.Duration         `json:"session_timeout"`
	AuthRequired   bool                  `json:"auth_required"`
	LogConfig      *config.LoggingConfig `json:"log_config,omitempty"`
}

// NewConfig creates a new admin server configuration with defaults
func NewConfig() *Config {
	return &Config{
		Host:           "localhost",
		Port:           "7680",
		DevMode:        true,
		StaticFiles:    "./web/admin",
		EnableRealtime: true,
		SessionTimeout: time.Hour,
		AuthRequired:   true,
	}
}

// Server represents the admin web server (refactored version)
type Server struct {
	config        *Config
	httpServer    *http.Server
	metabase      *client.Client
	loggerManager *log.Logger
	logStorage    *log.LogStorage
	logMiddleware *log.Middleware
}

// NewServer creates a new admin server
func NewServer(cfg *Config) (*Server, error) {
	if cfg == nil {
		cfg = NewConfig()
	}

	// Create MetaBase client (mock implementation for now)
	metabaseConfig := &client.Config{
		URL:    "http://localhost:7610", // API server port
		APIKey: "admin-api-key",         // Should come from config
	}

	// 初始化日志系统
	if cfg.LogConfig == nil {
		cfg.LogConfig = &config.LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			File:       "./logs/admin.log",
			MaxSize:    100,
			MaxAge:     7,
			MaxBackups: 3,
			Compress:   true,
			RequestID:  true,
			Caller:     false,
		}
	}

	loggerManager, err := log.NewLogger(cfg.LogConfig)
	if err != nil {
		return nil, err
	}

	// 初始化日志存储
	logStorage, err := log.NewLogStorage("./data/logs.db", 100000, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	// 初始化日志middleware
	middlewareConfig := &log.MiddlewareConfig{
		ServiceName:         "admin",
		StoreLogs:           true,
		LogRequestBody:      false,
		LogResponseBody:     false,
		MaxBodySize:         1024 * 1024,
		GenerateTraceID:     true,
		LogStatus:           "all",
		SkipPaths:           []string{"/health", "/api/admin/health"},
		DefaultFields:       map[string]interface{}{},
		MeasureResponseTime: true,
	}
	logMiddleware := log.NewMiddlewareWithConfigAndStorage(loggerManager, middlewareConfig, logStorage)

	return &Server{
		config:        cfg,
		metabase:      client.New(metabaseConfig),
		loggerManager: loggerManager,
		logStorage:    logStorage,
		logMiddleware: logMiddleware,
	}, nil
}

// Start starts the admin server
func (s *Server) Start() error {
	s.loggerManager.Info("Starting MetaBase Admin Server...")

	// Setup HTTP routes
	mux := http.NewServeMux()
	s.setupRoutes(mux)

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         s.config.Host + ":" + s.config.Port,
		Handler:      s.withMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.loggerManager.Info("Admin server starting", "address", s.httpServer.Addr)
	s.loggerManager.Info("🔧 Admin Interface", "url", fmt.Sprintf("http://localhost:%s", s.config.Port))

	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the admin server
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop HTTP server
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("HTTP server shutdown: %w", err)
		}
	}

	// Close log storage
	if s.logStorage != nil {
		if err := s.logStorage.Close(); err != nil {
			s.loggerManager.Error("Failed to close log storage", "error", err)
		}
	}

	s.loggerManager.Info("Admin server stopped successfully")
	return nil
}

// setupRoutes configures HTTP routes
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// API routes
	mux.HandleFunc("/api/admin/status", s.handleStatus)
	mux.HandleFunc("/api/admin/health", s.handleHealth)
	mux.HandleFunc("/api/admin/metrics", s.handleMetrics)

	// Log API routes - integrate log storage API
	logAPI := log.NewAPI(s.logStorage)

	// Wrap log API routes to work with net/http
	mux.HandleFunc("/api/admin/logs/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/stats"):
			logAPI.GetStats(w, r)
		case strings.HasSuffix(r.URL.Path, "/services"):
			logAPI.GetServices(w, r)
		case strings.HasSuffix(r.URL.Path, "/components"):
			logAPI.GetComponents(w, r)
		case strings.HasSuffix(r.URL.Path, "/levels"):
			logAPI.GetLevels(w, r)
		default:
			logAPI.QueryLogs(w, r)
		}
	})

	mux.HandleFunc("/api/admin/", s.handleAdminAPI)

	// Static files - serve admin interface
	mux.HandleFunc("/", s.handleStatic)
}

// HTTP handlers
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := map[string]interface{}{
		"server":    "metabase-admin",
		"version":   "1.0.0",
		"status":    "running",
		"timestamp": time.Now().Unix(),
	}

	s.writeJSON(w, status)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status":    "healthy",
		"nats":      "disabled", // Simplified since we removed NRPC
		"nrpc":      "disabled", // Simplified since we removed NRPC
		"timestamp": time.Now().Unix(),
	}

	s.writeJSON(w, health)
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"uptime":    time.Since(time.Now().Add(-time.Hour)).Seconds(),
		"services": map[string]interface{}{
			"admin": "running",
		},
	}

	s.writeJSON(w, metrics)
}

func (s *Server) handleAdminAPI(w http.ResponseWriter, r *http.Request) {
	// For now, return a simple admin API response
	path := strings.TrimPrefix(r.URL.Path, "/api/admin")
	// TODO: Implement proper admin API proxying using available client methods
	response := map[string]interface{}{
		"message": "Admin API endpoint",
		"path":    path,
		"method":  r.Method,
		"status":  "ok",
		"time":    time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Serve admin interface static files
	filePath := filepath.Join(s.config.StaticFiles, r.URL.Path)

	// Default to index.html for root path
	if r.URL.Path == "/" {
		filePath = filepath.Join(s.config.StaticFiles, "index.html")
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Serve index.html for SPA routing
		filePath = filepath.Join(s.config.StaticFiles, "index.html")
	}

	http.ServeFile(w, r, filePath)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Implementation would handle WebSocket connections for real-time updates
	http.Error(w, "WebSocket not implemented", http.StatusNotImplemented)
}

// Helper methods
func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.loggerManager.Error("Failed to encode JSON", "error", err)
	}
}

// Middleware
func (s *Server) withMiddleware(handler http.Handler) http.Handler {
	return s.logMiddleware.Middleware(s.logMiddleware.ComponentMiddleware("admin")(s.corsMiddleware(handler)))
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// responseWriter wrapper to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
