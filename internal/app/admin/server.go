package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/guileen/metabase/pkg/client"
)

// Config represents the admin server configuration
type Config struct {
	Host           string        `json:"host"`
	Port           string        `json:"port"`
	DevMode        bool          `json:"dev_mode"`
	StaticFiles    string        `json:"static_files"`
	EnableRealtime bool          `json:"enable_realtime"`
	SessionTimeout time.Duration `json:"session_timeout"`
	AuthRequired   bool          `json:"auth_required"`
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
	config     *Config
	httpServer *http.Server
	metabase   *client.Client
}

// NewServer creates a new admin server
func NewServer(config *Config) (*Server, error) {
	if config == nil {
		config = NewConfig()
	}

	// Create MetaBase client (mock implementation for now)
	metabaseConfig := &client.Config{
		URL:    fmt.Sprintf("http://localhost:7610"), // API server port
		APIKey: "admin-api-key",                      // Should come from config
	}

	return &Server{
		config:   config,
		metabase: client.New(metabaseConfig),
	}, nil
}

// Start starts the admin server
func (s *Server) Start() error {
	log.Printf("Starting MetaBase Admin Server...")

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

	log.Printf("Admin server starting on %s", s.httpServer.Addr)
	log.Printf("🔧 Admin Interface: http://localhost:%s", s.config.Port)

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

	log.Printf("Admin server stopped successfully")
	return nil
}

// setupRoutes configures HTTP routes
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// API routes
	mux.HandleFunc("/api/admin/status", s.handleStatus)
	mux.HandleFunc("/api/admin/health", s.handleHealth)
	mux.HandleFunc("/api/admin/metrics", s.handleMetrics)
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
	json.NewEncoder(w).Encode(response)
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
		log.Printf("Failed to encode JSON: %v", err)
	}
}

// Middleware
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
		wrapper := &responseWriter{ResponseWriter: w, statusCode: 200}

		next.ServeHTTP(wrapper, r)

		duration := time.Since(start)

		// Log request
		log.Printf("[Admin] %s %s %d %v", r.Method, r.URL.Path, wrapper.statusCode, duration)
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
