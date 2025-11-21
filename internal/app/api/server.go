package api

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/guileen/metabase/internal/app/api/handlers"
	"github.com/guileen/metabase/internal/app/api/keys"
	"go.uber.org/zap"
	_ "github.com/mattn/go-sqlite3"
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
	config       *Config
	httpServer   *http.Server
	logger       *zap.Logger
	db           *sql.DB
	keysManager  *keys.Manager
	restHandler  *handlers.RestHandler
	authHandler  *handlers.AuthHandler
	systemHandler *handlers.SystemHandler
	keyHandler   *keys.Handler
}

// NewServer creates a new API server
func NewServer(config *Config) (*Server, error) {
	if config == nil {
		config = NewConfig()
	}

	logger, _ := zap.NewDevelopment()

	// 初始化数据库
	db, err := sql.Open("sqlite3", config.DatabasePath)
	if err != nil {
		return nil, err
	}

	// 初始化API密钥管理器
	keysManager := keys.NewManager(db, logger)
	if err := keysManager.Initialize(context.Background()); err != nil {
		logger.Error("Failed to initialize keys manager", zap.Error(err))
		// 继续运行，可能是表已存在
	}

	server := &Server{
		config:       config,
		logger:       logger,
		db:           db,
		keysManager:  keysManager,
		restHandler:  handlers.NewRestHandler(db, logger),
		authHandler:  handlers.NewAuthHandler(db, logger),
		systemHandler: handlers.NewSystemHandler(logger),
		keyHandler:   keys.NewHandler(keysManager, logger),
	}

	return server, nil
}

// Start starts the API server
func (s *Server) Start() error {
	// 使用 chi 路由器
	r := chi.NewRouter()

	// Setup routes
	s.setupRoutes(r)

	s.httpServer = &http.Server{
		Addr:         s.config.Host + ":" + s.config.Port,
		Handler:      s.withMiddleware(r),
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

	if s.db != nil {
		s.db.Close()
	}

	if s.logger != nil {
		s.logger.Sync()
	}

	return nil
}

// setupRoutes configures API routes
func (s *Server) setupRoutes(r chi.Router) {
	// Health and system routes (no auth required)
	r.Get("/health", s.systemHandler.Health)
	r.Get("/ping", s.systemHandler.Ping)
	r.Get("/version", s.systemHandler.Version)

	// Authentication routes
	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", s.authHandler.Login)
		r.Post("/register", s.authHandler.Register)
		r.Post("/refresh", s.authHandler.RefreshToken)
	})

	// API Key management routes (requires auth)
	r.Route("/keys", func(r chi.Router) {
		r.Use(s.authMiddleware)
		s.keyHandler.RegisterRoutes(r)
	})

	// Supabase-like REST API routes (requires API key)
	r.Route("/", func(r chi.Router) {
		r.Use(s.apiKeyMiddleware)
		s.restHandler.RegisterRoutes(r)
	})
}

// Middleware

// withMiddleware applies global middleware
func (s *Server) withMiddleware(handler http.Handler) http.Handler {
	return s.corsMiddleware(s.loggingMiddleware(handler))
}

// corsMiddleware handles CORS
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, apikey")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create response writer wrapper to capture status
		ww := &responseWriter{ResponseWriter: w, statusCode: 200}

		next.ServeHTTP(ww, r)

		duration := time.Since(start)

		// Log request details
		s.logger.Info("API request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", ww.statusCode),
			zap.Duration("duration", duration),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
		)
	})
}

// authMiddleware handles authentication using JWT
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Simple validation - in production, validate JWT token
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		token := authHeader[7:] // Remove "Bearer " prefix

		// TODO: Implement proper JWT validation
		// For now, just check if token exists
		if token == "" {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// apiKeyMiddleware handles API key authentication
func (s *Server) apiKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for API key in header or query parameter
		apiKey := r.Header.Get("apikey")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("apikey")
		}

		if apiKey == "" {
			http.Error(w, "API key required", http.StatusUnauthorized)
			return
		}

		// Validate API key
		validKey, err := s.keysManager.Validate(r.Context(), apiKey)
		if err != nil {
			s.logger.Error("Invalid API key", zap.Error(err))
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		// Add API key to context
		ctx := context.WithValue(r.Context(), "apiKey", validKey.ToRestAPIKey())
		next.ServeHTTP(w, r.WithContext(ctx))
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