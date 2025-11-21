package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Config represents the API server configuration
type Config struct {
	Host         string `json:"host"`
	Port         string `json:"port"`
	DevMode      bool   `json:"dev_mode"`
	DatabasePath string `json:"database_path"`
}

// NewConfig creates a new API server configuration with defaults
func NewConfig() *Config {
	return &Config{
		Host:         "localhost",
		Port:         "7610",
		DevMode:      true,
		DatabasePath: "./data/metabase.db",
	}
}

// Server represents the API server
type Server struct {
	config     *Config
	httpServer *http.Server
	logger     *zap.Logger
}

// NewServer creates a new API server
func NewServer(config *Config) (*Server, error) {
	if config == nil {
		config = NewConfig()
	}

	logger, _ := zap.NewDevelopment()

	server := &Server{
		config: config,
		logger: logger,
	}

	return server, nil
}

// Start starts the API server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Setup routes
	s.setupRoutes(mux)

	s.httpServer = &http.Server{
		Addr:         s.config.Host + ":" + s.config.Port,
		Handler:      s.withMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	s.logger.Info("Starting API server",
		zap.String("address", s.httpServer.Addr),
	)

	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the API server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping API server")

	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}

	if s.logger != nil {
		s.logger.Sync()
	}

	return nil
}

// setupRoutes configures API routes
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// Health and system routes (no auth required)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ping", s.handlePing)
	mux.HandleFunc("/version", s.handleVersion)

	// API v1 routes
	mux.HandleFunc("/v1/", s.handleV1)
}

// handleV1 handles all v1 API routes
func (s *Server) handleV1(w http.ResponseWriter, r *http.Request) {
	// Simple v1 API handler
	response := map[string]interface{}{
		"message": "MetaBase API v1.0",
		"version": "1.0.0",
		"endpoints": []string{
			"/v1/health",
			"/v1/ping",
			"/v1/version",
		},
	}

	s.writeJSON(w, response)
}

// HTTP handlers
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
		"service":   "metabase-api",
		"uptime":    "0h", // TODO: calculate actual uptime
	}

	s.writeJSON(w, health)
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	version := map[string]interface{}{
		"version":    "1.0.0",
		"build_time":  time.Now().Format(time.RFC3339),
		"go_version":  "go1.25.3",
		"service":     "metabase-api",
		"environment": "development",
	}

	s.writeJSON(w, version)
}

// Helper methods
func (s *Server) withMiddleware(handler http.Handler) http.Handler {
	return s.loggingMiddleware(s.corsMiddleware(handler))
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

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create response writer wrapper to capture status
		ww := &responseWriter{ResponseWriter: w, statusCode: 200}

		next.ServeHTTP(ww, r)

		duration := time.Since(start)

		// Log request details
		fields := []zap.Field{
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", ww.statusCode),
			zap.Duration("duration", duration),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
		}

		s.logger.Info("API request", fields...)
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

func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}