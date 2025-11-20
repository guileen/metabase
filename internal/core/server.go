package core

import ("context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/guileen/metabase/internal/app/console"
	"github.com/guileen/metabase/pkg/common/nrpc"
	"github.com/guileen/metabase/pkg/infra/storage")

// Config represents the unified server configuration
type Config struct {
	Host    string `json:"host"`
	Port    string `json:"port"`
	DevMode bool   `json:"dev_mode"`

	// Component configurations
	NRPC     *nrpc.Config     `json:"nrpc"`
	Storage  *storage.Config  `json:"storage"`
	Console  *console.Config  `json:"console"`

	// Feature flags
	EnableNRPC    bool `json:"enable_nrpc"`
	EnableStorage bool `json:"enable_storage"`
	EnableConsole bool `json:"enable_console"`
}

// Server represents the unified MetaBase server
type Server struct {
	config   *Config
	ctx      context.Context
	cancel   context.CancelFunc

	// Core components
	nrpcServer  *nrpc.Server
	nrpcClient  *nrpc.Client
	storage     *storage.Engine
	console     *console.Server

	// HTTP server
	httpServer *http.Server
}

// NewConfig creates a new server configuration with defaults
func NewConfig() *Config {
	return &Config{
		Host:    "localhost",
		Port:    "7609",
		DevMode: true,

		NRPC:     nrpc.NewConfig(),
		Storage:  storage.NewConfig(),
		Console:  console.NewConfig(),

		EnableNRPC:    true,
		EnableStorage: true,
		EnableConsole: true,
	}
}

// NewServer creates a new unified server
func NewServer(config *Config) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	server := &Server{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize components
	if config.EnableStorage {
		storageEngine, err := storage.NewEngine(config.Storage)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to initialize storage engine: %w", err)
		}
		server.storage = storageEngine
		log.Printf("✅ Storage engine initialized")
	}

	if config.EnableConsole {
		// Create console config with storage reference
		consoleConfig := &console.Config{
			Port:       config.Console.Port,
			Host:       config.Console.Host,
			DevMode:    config.Console.DevMode,
			LogLevel:   config.Console.LogLevel,
			MaxLogs:    config.Console.MaxLogs,
			MetricsTTL: config.Console.MetricsTTL,
			Storage:    server.storage,
		}
		consoleServer := console.NewServer(consoleConfig)
		server.console = consoleServer
		log.Printf("✅ Console server initialized")
	}

	if config.EnableNRPC {
		nrpcServer, err := nrpc.NewServer(config.NRPC)
		if err != nil {
			cancel()
			if server.storage != nil {
				server.storage.Close()
			}
			return nil, fmt.Errorf("failed to initialize NRPC server: %w", err)
		}
		server.nrpcServer = nrpcServer

		nrpcClient, err := nrpc.NewClient(config.NRPC)
		if err != nil {
			cancel()
			if server.storage != nil {
				server.storage.Close()
			}
			if server.nrpcServer != nil {
				server.nrpcServer.Close()
			}
			return nil, fmt.Errorf("failed to initialize NRPC client: %w", err)
		}
		server.nrpcClient = nrpcClient

		// Register core services
		if err := server.registerServices(); err != nil {
			cancel()
			server.cleanup()
			return nil, fmt.Errorf("failed to register services: %w", err)
		}

		log.Printf("✅ NRPC server initialized")
	}

	return server, nil
}

// Start starts the unified server
func (s *Server) Start() error {
	// Start component servers
	if s.console != nil {
		// Start console in background
		go func() {
			if err := s.console.Start(); err != nil {
				log.Printf("Console server error: %v", err)
			}
		}()
	}

	// Create HTTP server with unified routing
	mux := s.createRouter()

	s.httpServer = &http.Server{
		Addr:         s.config.Host + ":" + s.config.Port,
		Handler:      s.withMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Print startup information
	s.printStartupInfo()

	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}

	var errors [][]error

	// Stop HTTP server
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(ctx); err != nil {
			errors = append(errors, []error{fmt.Errorf("HTTP server shutdown: %w", err)})
		}
	}

	// Stop console server
	if s.console != nil {
		if err := s.console.Stop(); err != nil {
			errors = append(errors, []error{fmt.Errorf("console server shutdown: %w", err)})
		}
	}

	// Cleanup components
	s.cleanup()

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors occurred: %v", errors)
	}

	log.Printf("🛑 MetaBase server stopped gracefully")
	return nil
}

// createRouter creates the main HTTP router
func (s *Server) createRouter() *http.ServeMux {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/version", s.handleVersion)
	mux.HandleFunc("/api/stats", s.handleStats)

	// Storage API routes
	if s.storage != nil {
		mux.HandleFunc("/api/storage/", s.handleStorageAPI)
	}

	// NRPC API routes
	if s.nrpcClient != nil {
		mux.HandleFunc("/api/rpc/", s.handleRPCAPI)
		mux.HandleFunc("/api/tasks/", s.handleTasksAPI)
	}

	// Console routes (if enabled separately)
	if !s.config.EnableConsole {
		// Fallback console routes
		mux.HandleFunc("/api/console/", s.handleConsoleAPI)
	}

	// Admin routes
	mux.HandleFunc("/admin/", s.handleAdmin)

	// Static assets
	mux.Handle("/assets/", http.FileServer(http.Dir("web/assets")))

	// Root route
	mux.HandleFunc("/", s.handleRoot)

	return mux
}

// withMiddleware adds middleware to the handler
func (s *Server) withMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log request
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			userID = "anonymous"
		}

		// Create response writer wrapper to capture status
		wrapper := &responseWriter{ResponseWriter: w, statusCode: 200}

		// Call next handler
		handler.ServeHTTP(wrapper, r)

		duration := time.Since(start)

		// Log to console
		if s.console != nil {
			s.console.LogRequest(r.Method, r.URL.Path, wrapper.statusCode, duration, userID, getRemoteIP(r), r.UserAgent())
		} else {
			log.Printf("[%s] %s %s %d %v", userID, r.Method, r.URL.Path, wrapper.statusCode, duration)
		}
	})
}

// registerServices registers NRPC services
func (s *Server) registerServices() error {
	// Storage service
	if s.storage != nil {
		storageService := nrpc.NewService("storage")
		storageService.RegisterHandler("create", s.handleStorageCreate)
		storageService.RegisterHandler("get", s.handleStorageGet)
		storageService.RegisterHandler("update", s.handleStorageUpdate)
		storageService.RegisterHandler("delete", s.handleStorageDelete)
		storageService.RegisterHandler("query", s.handleStorageQuery)

		if err := s.nrpcServer.RegisterService(storageService); err != nil {
			return fmt.Errorf("failed to register storage service: %w", err)
		}

		if err := s.nrpcServer.SubscribeTasks(storageService); err != nil {
			return fmt.Errorf("failed to subscribe storage tasks: %w", err)
		}
	}

	// System service
	systemService := nrpc.NewService("system")
	systemService.RegisterHandler("ping", s.handleSystemPing)
	systemService.RegisterHandler("stats", s.handleSystemStats)

	if err := s.nrpcServer.RegisterService(systemService); err != nil {
		return fmt.Errorf("failed to register system service: %w", err)
	}

	return nil
}

// HTTP Handlers
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
		"components": map[string]interface{}{
			"storage": s.storage != nil,
			"nrpc":    s.nrpcServer != nil,
			"console": s.console != nil,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	version := map[string]interface{}{
		"version":     "1.0.0",
		"build_time":  time.Now().Format(time.RFC3339),
		"go_version":  "go1.25.3",
		"components": map[string]string{
			"nats":    "v2.12.2",
			"sqlite":  "builtin",
			"pebble":  "builtin",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(version)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"uptime": time.Since(time.Now()), // This should be calculated from server start time
		"components": map[string]interface{}{},
	}

	// Storage stats
	if s.storage != nil {
		stats["components"].(map[string]interface{})["storage"] = s.storage.Stats()
	}

	// NRPC stats
	if s.nrpcServer != nil {
		stats["components"].(map[string]interface{})["nrpc"] = s.nrpcServer.GetStats()
	}

	// Console stats
	if s.console != nil {
		stats["components"].(map[string]interface{})["console"] = s.console.GetStats()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleStorageAPI(w http.ResponseWriter, r *http.Request) {
	if s.storage == nil {
		http.Error(w, "Storage not available", http.StatusServiceUnavailable)
		return
	}

	// Extract operation from path
	path := strings.TrimPrefix(r.URL.Path, "/api/storage/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 {
		http.Error(w, "Invalid storage API path", http.StatusBadRequest)
		return
	}

	operation := parts[0]

	switch operation {
	case "create":
		s.handleStorageCreateHTTP(w, r)
	case "get":
		s.handleStorageGetHTTP(w, r)
	case "update":
		s.handleStorageUpdateHTTP(w, r)
	case "delete":
		s.handleStorageDeleteHTTP(w, r)
	case "query":
		s.handleStorageQueryHTTP(w, r)
	default:
		http.Error(w, "Unknown storage operation", http.StatusBadRequest)
	}
}

func (s *Server) handleRPCAPI(w http.ResponseWriter, r *http.Request) {
	if s.nrpcClient == nil {
		http.Error(w, "RPC not available", http.StatusServiceUnavailable)
		return
	}

	// Extract service and method from path
	path := strings.TrimPrefix(r.URL.Path, "/api/rpc/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid RPC API path", http.StatusBadRequest)
		return
	}

	service := parts[0]
	method := parts[1]

	// Parse request data
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Make RPC call
	ctx := r.Context()
	response, err := s.nrpcClient.Call(ctx, service, method, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleTasksAPI(w http.ResponseWriter, r *http.Request) {
	if s.nrpcClient == nil {
		http.Error(w, "Task queue not available", http.StatusServiceUnavailable)
		return
	}

	if r.Method == "POST" {
		// Extract service and method from path
		path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			http.Error(w, "Invalid task API path", http.StatusBadRequest)
			return
		}

		service := parts[0]
		method := parts[1]

		// Parse request data
		var data map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Get delay from query
		delayStr := r.URL.Query().Get("delay")
		var delay time.Duration
		if delayStr != "" {
			if d, err := time.ParseDuration(delayStr); err == nil {
				delay = d
			}
		}

		// Publish task
		ctx := r.Context()
		if err := s.nrpcClient.PublishTask(ctx, service, method, data, delay); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Task published",
		})
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleConsoleAPI(w http.ResponseWriter, r *http.Request) {
	// Fallback console API implementation
	// This would proxy to the console server if it's not running on a separate port
	http.Error(w, "Console not available", http.StatusServiceUnavailable)
}

func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	// Serve admin interface from static files
	http.ServeFile(w, r, "admin"+r.URL.Path[len("/admin/"):])
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		// Serve dashboard or redirect to docs
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":        "MetaBase",
			"description": "下一代后端核心",
			"version":     "1.0.0",
			"endpoints": map[string]string{
				"health":  "/api/health",
				"version": "/api/version",
				"stats":   "/api/stats",
			},
		})
		return
	}

	http.NotFound(w, r)
}

// Storage HTTP handlers
func (s *Server) handleStorageCreateHTTP(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Table string                 `json:"table"`
		Data  map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	record, err := s.storage.Create(r.Context(), request.Table, request.Data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

func (s *Server) handleStorageGetHTTP(w http.ResponseWriter, r *http.Request) {
	table := r.URL.Query().Get("table")
	id := r.URL.Query().Get("id")

	if table == "" || id == "" {
		http.Error(w, "Table and ID required", http.StatusBadRequest)
		return
	}

	record, err := s.storage.Get(r.Context(), table, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

func (s *Server) handleStorageUpdateHTTP(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Table string                 `json:"table"`
		ID    string                 `json:"id"`
		Data  map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	record, err := s.storage.Update(r.Context(), request.Table, request.ID, request.Data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

func (s *Server) handleStorageDeleteHTTP(w http.ResponseWriter, r *http.Request) {
	table := r.URL.Query().Get("table")
	id := r.URL.Query().Get("id")

	if table == "" || id == "" {
		http.Error(w, "Table and ID required", http.StatusBadRequest)
		return
	}

	err := s.storage.Delete(r.Context(), table, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) handleStorageQueryHTTP(w http.ResponseWriter, r *http.Request) {
	table := r.URL.Query().Get("table")
	if table == "" {
		http.Error(w, "Table required", http.StatusBadRequest)
		return
	}

	options := &storage.QueryOptions{
		Limit: 100,
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			options.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			options.Offset = offset
		}
	}

	result, err := s.storage.Query(r.Context(), table, options)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// NRPC Service handlers
func (s *Server) handleStorageCreate(ctx context.Context, msg *nrpc.Message) (*nrpc.Message, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("storage not available")
	}

	table := msg.Data["table"].(string)
	data := msg.Data["data"].(map[string]interface{})

	record, err := s.storage.Create(ctx, table, data)
	if err != nil {
		return nil, err
	}

	return &nrpc.Message{
		ID:      msg.ID,
		Type:    nrpc.MessageTypeResult,
		Service: msg.Service,
		Method:  msg.Method,
		Data: map[string]interface{}{
			"record": record,
		},
		Timestamp: time.Now(),
	}, nil
}

func (s *Server) handleStorageGet(ctx context.Context, msg *nrpc.Message) (*nrpc.Message, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("storage not available")
	}

	table := msg.Data["table"].(string)
	id := msg.Data["id"].(string)

	record, err := s.storage.Get(ctx, table, id)
	if err != nil {
		return nil, err
	}

	return &nrpc.Message{
		ID:      msg.ID,
		Type:    nrpc.MessageTypeResult,
		Service: msg.Service,
		Method:  msg.Method,
		Data: map[string]interface{}{
			"record": record,
		},
		Timestamp: time.Now(),
	}, nil
}

func (s *Server) handleStorageUpdate(ctx context.Context, msg *nrpc.Message) (*nrpc.Message, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("storage not available")
	}

	table := msg.Data["table"].(string)
	id := msg.Data["id"].(string)
	data := msg.Data["data"].(map[string]interface{})

	record, err := s.storage.Update(ctx, table, id, data)
	if err != nil {
		return nil, err
	}

	return &nrpc.Message{
		ID:      msg.ID,
		Type:    nrpc.MessageTypeResult,
		Service: msg.Service,
		Method:  msg.Method,
		Data: map[string]interface{}{
			"record": record,
		},
		Timestamp: time.Now(),
	}, nil
}

func (s *Server) handleStorageDelete(ctx context.Context, msg *nrpc.Message) (*nrpc.Message, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("storage not available")
	}

	table := msg.Data["table"].(string)
	id := msg.Data["id"].(string)

	err := s.storage.Delete(ctx, table, id)
	if err != nil {
		return nil, err
	}

	return &nrpc.Message{
		ID:      msg.ID,
		Type:    nrpc.MessageTypeResult,
		Service: msg.Service,
		Method:  msg.Method,
		Data: map[string]interface{}{
			"status": "success",
		},
		Timestamp: time.Now(),
	}, nil
}

func (s *Server) handleStorageQuery(ctx context.Context, msg *nrpc.Message) (*nrpc.Message, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("storage not available")
	}

	table := msg.Data["table"].(string)
	options := &storage.QueryOptions{Limit: 100}

	if limit, ok := msg.Data["limit"].(float64); ok {
		options.Limit = int(limit)
	}
	if offset, ok := msg.Data["offset"].(float64); ok {
		options.Offset = int(offset)
	}

	result, err := s.storage.Query(ctx, table, options)
	if err != nil {
		return nil, err
	}

	return &nrpc.Message{
		ID:      msg.ID,
		Type:    nrpc.MessageTypeResult,
		Service: msg.Service,
		Method:  msg.Method,
		Data: map[string]interface{}{
			"result": result,
		},
		Timestamp: time.Now(),
	}, nil
}

func (s *Server) handleSystemPing(ctx context.Context, msg *nrpc.Message) (*nrpc.Message, error) {
	return &nrpc.Message{
		ID:      msg.ID,
		Type:    nrpc.MessageTypeResult,
		Service: msg.Service,
		Method:  msg.Method,
		Data: map[string]interface{}{
			"pong":    true,
			"time":    time.Now(),
			"version": "1.0.0",
		},
		Timestamp: time.Now(),
	}, nil
}

func (s *Server) handleSystemStats(ctx context.Context, msg *nrpc.Message) (*nrpc.Message, error) {
	stats := map[string]interface{}{}

	if s.storage != nil {
		stats["storage"] = s.storage.Stats()
	}
	if s.nrpcServer != nil {
		stats["nrpc"] = s.nrpcServer.GetStats()
	}

	return &nrpc.Message{
		ID:      msg.ID,
		Type:    nrpc.MessageTypeResult,
		Service: msg.Service,
		Method:  msg.Method,
		Data: map[string]interface{}{
			"stats": stats,
		},
		Timestamp: time.Now(),
	}, nil
}

// Helper methods
func (s *Server) printStartupInfo() {
	log.Printf("🚀 MetaBase Core Server started on http://%s", s.httpServer.Addr)
	log.Printf("📊 Health Check: http://%s/api/health", s.httpServer.Addr)
	log.Printf("📈 Statistics: http://%s/api/stats", s.httpServer.Addr)

	if s.config.EnableConsole && s.console != nil {
		log.Printf("🔧 Console Dashboard: http://localhost:%s", s.config.Console.Port)
	}

	if s.config.EnableStorage {
		log.Printf("💾 Storage Engine: SQLite + Pebble")
	}

	if s.config.EnableNRPC {
		log.Printf("🔄 Message Queue: NATS (%s)", s.config.NRPC.NATSURL)
	}

	if s.config.DevMode {
		log.Printf("🔧 Development Mode: Enabled")
	}
}

func (s *Server) cleanup() {
	if s.nrpcServer != nil {
		s.nrpcServer.Close()
	}
	if s.nrpcClient != nil {
		s.nrpcClient.Close()
	}
	if s.storage != nil {
		s.storage.Close()
	}
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

// Helper functions
func getRemoteIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}