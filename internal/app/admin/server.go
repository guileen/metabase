package admin

import ("context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/guileen/metabase/internal/services/api"
	"github.com/guileen/metabase/pkg/common/nrpc/v2"
	"github.com/guileen/metabase/pkg/common/nrpc/embedded")

// Server represents the admin web server
type Server struct {
	nats        *embedded.EmbeddedNATS
	nrpcServer  *nrpc.Server
	metabase    *api.Client
	httpServer  *http.Server
	config      *Config
	staticFiles string
}

// Config represents admin server configuration
type Config struct {
	Host            string        `yaml:"host" json:"host"`
	Port            int           `yaml:"port" json:"port"`
	MetaBaseURL     string        `yaml:"metabase_url" json:"metabase_url"`
	MetaBaseAPIKey  string        `yaml:"metabase_api_key" json:"metabase_api_key"`
	StaticFiles     string        `yaml:"static_files" json:"static_files"`
	EnableRealtime  bool          `yaml:"enable_realtime" json:"enable_realtime"`
	SessionTimeout  time.Duration `yaml:"session_timeout" json:"session_timeout"`
	AuthRequired    bool          `yaml:"auth_required" json:"auth_required"`
}

// NewServer creates a new admin server
func NewServer(config *Config) (*Server, error) {
	// Set defaults
	if config == nil {
		config = &Config{
			Host:           "0.0.0.0",
			Port:           7680,
			StaticFiles:    "./admin/dist",
			EnableRealtime: true,
			SessionTimeout: time.Hour,
			AuthRequired:   true,
		}
	}

	// Create embedded NATS
	natsConfig := &embedded.Config{
		ServerPort: 4223, // Different port to avoid conflicts
		ClientURL:  "nats://localhost:4223",
		StoreDir:   "./data/admin_nats",
		JetStream:  true,
	}

	nats := embedded.NewEmbeddedNATS(natsConfig)

	// Create NRPC server
	nrpcConfig := &nrpc.Config{
		Name:            "metabase-admin",
		Version:         "1.0.0",
		Namespace:       "admin",
		EnableStreaming: true,
		EnableMetrics:   true,
	}

	nrpcServer := nrpc.NewServer(nats, nrpcConfig)

	// Create MetaBase client
	metabaseConfig := &api.Config{
		BaseURL:      config.MetaBaseURL,
		APIKey:       config.MetaBaseAPIKey,
		Timeout:      30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:   time.Second,
	}

	metabase := api.NewClient(metabaseConfig)

	server := &Server{
		nats:       nats,
		nrpcServer: nrpcServer,
		metabase:   metabase,
		config:     config,
		staticFiles: config.StaticFiles,
	}

	// Register NRPC services
	server.registerServices()

	return server, nil
}

// Start starts the admin server
func (s *Server) Start() error {
	log.Printf("Starting MetaBase Admin Server...")

	// Start embedded NATS
	if err := s.nats.Start(); err != nil {
		return fmt.Errorf("failed to start NATS: %w", err)
	}

	// Start NRPC server
	if err := s.nrpcServer.Start(); err != nil {
		return fmt.Errorf("failed to start NRPC server: %w", err)
	}

	// Setup HTTP routes
	mux := http.NewServeMux()
	s.setupRoutes(mux)

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
		Handler:      s.setupMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Admin server starting on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Stop stops the admin server
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var errors []error

	// Stop HTTP server
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown HTTP server: %w", err))
		}
	}

	// Stop NRPC server
	if s.nrpcServer != nil {
		if err := s.nrpcServer.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown NRPC server: %w", err))
		}
	}

	// Stop NATS
	if s.nats != nil {
		if err := s.nats.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop NATS: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("multiple errors occurred: %v", errors)
	}

	log.Printf("Admin server stopped successfully")
	return nil
}

// registerServices registers NRPC services
func (s *Server) registerServices() {
	// Register admin services
	healthService := nrpc.NewHealthService()
	echoService := nrpc.NewEchoService()
	adminService := s.createAdminService()

	s.nrpcServer.RegisterHandler(healthService)
	s.nrpcServer.RegisterHandler(echoService)
	s.nrpcServer.RegisterHandler(adminService)

	// Register real-time services if enabled
	if s.config.EnableRealtime {
		realtimeService := s.createRealtimeService()
		s.nrpcServer.RegisterHandler(realtimeService)
	}
}

// createAdminService creates the admin NRPC service
func (s *Server) createAdminService() *nrpc.Service {
	builder := nrpc.NewServiceBuilder("admin")

	// Get server status
	builder.Method("status", "Get admin server status", func(ctx context.Context, req *nrpc.Request) (*nrpc.Response, error) {
		status := map[string]interface{}{
			"server":      "metabase-admin",
			"version":     "1.0.0",
			"uptime":      time.Since(time.Now().Add(-time.Hour)).String(),
			"nats_ready":  s.nats.IsReady(),
			"nrpc_started": s.nrpcServer.IsStarted(),
		}

		return &nrpc.Response{
			ID:   req.ID,
			Data: status,
		}, nil
	})

	// Get database info
	builder.Method("database_info", "Get database information", func(ctx context.Context, req *nrpc.Request) (*nrpc.Response, error) {
		response, err := s.metabase.Get(ctx, "/admin/database/info")
		if err != nil {
			return nil, fmt.Errorf("failed to get database info: %w", err)
		}

		return &nrpc.Response{
			ID:   req.ID,
			Data: response.Data,
		}, nil
	})

	// Get tenant info
	builder.Method("tenant_info", "Get tenant information", func(ctx context.Context, req *nrpc.Request) (*nrpc.Response, error) {
		response, err := s.metabase.Get(ctx, "/admin/tenants")
		if err != nil {
			return nil, fmt.Errorf("failed to get tenant info: %w", err)
		}

		return &nrpc.Response{
			ID:   req.ID,
			Data: response.Data,
		}, nil
	})

	// Create tenant
	builder.Method("create_tenant", "Create a new tenant", func(ctx context.Context, req *nrpc.Request) (*nrpc.Response, error) {
		response, err := s.metabase.Post(ctx, "/admin/tenants", req.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to create tenant: %w", err)
		}

		return &nrpc.Response{
			ID:   req.ID,
			Data: response.Data,
		}, nil
	})

	// Get user info
	builder.Method("user_info", "Get user information", func(ctx context.Context, req *nrpc.Request) (*nrpc.Response, error) {
		userID, ok := req.Data["user_id"].(string)
		if !ok {
			return nil, fmt.Errorf("user_id required")
		}

		response, err := s.metabase.Get(ctx, fmt.Sprintf("/admin/users/%s", userID))
		if err != nil {
			return nil, fmt.Errorf("failed to get user info: %w", err)
		}

		return &nrpc.Response{
			ID:   req.ID,
			Data: response.Data,
		}, nil
	})

	// Get metrics
	builder.Method("metrics", "Get server metrics", func(ctx context.Context, req *nrpc.Request) (*nrpc.Response, error) {
		metrics := map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"uptime":    time.Since(time.Now().Add(-time.Hour)).Seconds(),
			"nats":      s.getNATSMetrics(),
			"nrpc":      s.getNRPCMetrics(),
		}

		return &nrpc.Response{
			ID:   req.ID,
			Data: metrics,
		}, nil
	})

	return builder.Build()
}

// createRealtimeService creates the real-time NRPC service
func (s *Server) createRealtimeService() *nrpc.Service {
	builder := nrpc.NewServiceBuilder("realtime")

	// Subscribe to events
	builder.Method("subscribe", "Subscribe to real-time events", func(ctx context.Context, req *nrpc.Request) (*nrpc.Response, error) {
		channel, ok := req.Data["channel"].(string)
		if !ok {
			return nil, fmt.Errorf("channel required")
		}

		// Implementation would handle real-time subscriptions
		return &nrpc.Response{
			ID:   req.ID,
			Data: map[string]interface{}{"subscribed": true, "channel": channel},
		}, nil
	})

	// Unsubscribe from events
	builder.Method("unsubscribe", "Unsubscribe from real-time events", func(ctx context.Context, req *nrpc.Request) (*nrpc.Response, error) {
		channel, ok := req.Data["channel"].(string)
		if !ok {
			return nil, fmt.Errorf("channel required")
		}

		// Implementation would handle real-time unsubscriptions
		return &nrpc.Response{
			ID:   req.ID,
			Data: map[string]interface{}{"unsubscribed": true, "channel": channel},
		}, nil
	})

	return builder.Build()
}

// setupRoutes sets up HTTP routes
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// API routes
	mux.HandleFunc("/api/admin/status", s.handleStatus)
	mux.HandleFunc("/api/admin/health", s.handleHealth)
	mux.HandleFunc("/api/admin/metrics", s.handleMetrics)
	mux.HandleFunc("/api/admin/", s.handleAdminAPI)

	// WebSocket for real-time updates
	if s.config.EnableRealtime {
		mux.HandleFunc("/ws/realtime", s.handleWebSocket)
	}

	// Static files
	mux.HandleFunc("/", s.handleStatic)
}

// setupMiddleware sets up HTTP middleware
func (s *Server) setupMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS middleware
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Logging middleware
		start := time.Now()
		handler.ServeHTTP(w, r)
		duration := time.Since(start)

		log.Printf("%s %s %s %v", r.Method, r.URL.Path, r.RemoteAddr, duration)
	})
}

// HTTP handlers
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := map[string]interface{}{
		"server":       "metabase-admin",
		"version":      "1.0.0",
		"status":       "running",
		"nats_ready":   s.nats.IsReady(),
		"nrpc_started": s.nrpcServer.IsStarted(),
		"timestamp":    time.Now().Unix(),
	}

	s.writeJSON(w, status)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status":     "healthy",
		"nats":       "healthy" if s.nats.IsReady() else "unhealthy",
		"nrpc":       "healthy" if s.nrpcServer.IsStarted() else "unhealthy",
		"timestamp":  time.Now().Unix(),
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
		"nats":      s.getNATSMetrics(),
		"nrpc":      s.getNRPCMetrics(),
	}

	s.writeJSON(w, metrics)
}

func (s *Server) handleAdminAPI(w http.ResponseWriter, r *http.Request) {
	// Proxy admin API requests to MetaBase using the client library
	path := strings.TrimPrefix(r.URL.Path, "/api/admin")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response *api.Response
	var err error

	switch r.Method {
	case http.MethodGet:
		response, err = s.metabase.Get(ctx, path)
	case http.MethodPost:
		var data interface{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		response, err = s.metabase.Post(ctx, path, data)
	case http.MethodPut:
		var data interface{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		response, err = s.metabase.Put(ctx, path, data)
	case http.MethodDelete:
		response, err = s.metabase.Delete(ctx, path)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if response.Data != nil {
		json.NewEncoder(w).Encode(response.Data)
	}
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Serve static files from the admin directory
	filePath := filepath.Join(s.staticFiles, r.URL.Path)

	// Default to index.html for root path
	if r.URL.Path == "/" {
		filePath = filepath.Join(s.staticFiles, "index.html")
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Serve index.html for SPA routing
		filePath = filepath.Join(s.staticFiles, "index.html")
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

func (s *Server) getNATSMetrics() map[string]interface{} {
	return map[string]interface{}{
		"ready":    s.nats.IsReady(),
		"shutdown": s.nats.IsShutdown(),
		"stats":    s.nats.GetStats(),
	}
}

func (s *Server) getNRPCMetrics() map[string]interface{} {
	handlers := s.nrpcServer.GetHandlers()
	handlerCount := make(map[string]int)
	for name, handler := range handlers {
		handlerCount[name] = len(handler.Methods())
	}

	return map[string]interface{}{
		"started":  s.nrpcServer.IsStarted(),
		"handlers": len(handlers),
		"methods":  handlerCount,
	}
}