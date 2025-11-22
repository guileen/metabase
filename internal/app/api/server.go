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
	"github.com/guileen/metabase/internal/app/api/middleware"
	"github.com/guileen/metabase/pkg/config"
	"github.com/guileen/metabase/pkg/infra/auth"
	"github.com/guileen/metabase/pkg/log"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

// Config represents the API server configuration
type Config struct {
	Host         string                `json:"host"`
	Port         string                `json:"port"`
	DevMode      bool                  `json:"dev_mode"`
	DatabasePath string                `json:"database_path"`
	LogConfig    *config.LoggingConfig `json:"log_config,omitempty"`
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
	config            *Config
	httpServer        *http.Server
	logger            *zap.Logger
	loggerManager     *log.Logger
	logStorage        *log.LogStorage
	logMiddleware     *log.Middleware
	db                *sql.DB
	keysManager       *keys.Manager
	rbacManager       *auth.RBACManager
	tenantManager     *auth.TenantManager
	restHandler       *handlers.RestHandler
	authHandler       *handlers.AuthHandler
	systemHandler     *handlers.SystemHandler
	keyHandler        *keys.Handler
	tenantHandler     *handlers.TenantHandler
	adminHandler      *handlers.AdminHandler
	projectMiddleware *middleware.ProjectMiddleware
}

// NewServer creates a new API server
func NewServer(cfg *Config) (*Server, error) {
	if cfg == nil {
		cfg = NewConfig()
	}

	logger, _ := zap.NewDevelopment()

	// 初始化数据库
	db, err := sql.Open("sqlite3", cfg.DatabasePath)
	if err != nil {
		return nil, err
	}

	// 初始化日志系统
	if cfg.LogConfig == nil {
		cfg.LogConfig = &config.LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			File:       "./logs/api.log",
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
		ServiceName:         "api",
		StoreLogs:           true,
		LogRequestBody:      false,
		LogResponseBody:     false,
		MaxBodySize:         1024 * 1024,
		GenerateTraceID:     true,
		LogStatus:           "all",
		SkipPaths:           []string{"/health", "/ping", "/version"},
		DefaultFields:       map[string]interface{}{},
		MeasureResponseTime: true,
	}
	logMiddleware := log.NewMiddlewareWithConfigAndStorage(loggerManager, middlewareConfig, logStorage)

	// 初始化API密钥管理器
	keysManager := keys.NewManager(db, logger)
	if err := keysManager.Initialize(context.Background()); err != nil {
		logger.Error("Failed to initialize keys manager", zap.Error(err))
		// 继续运行，可能是表已存在
	}

	// 运行数据库迁移，创建租户和项目表
	migrationRunner := auth.NewMigrationRunner(db)
	if err := migrationRunner.RunMigrations(context.Background()); err != nil {
		logger.Error("Failed to run database migrations", zap.Error(err))
		// 继续运行，可能是表已存在
	}

	// 初始化RBAC和租户管理器
	rbacManager := auth.NewRBACManager()
	if err := rbacManager.InitializeDefaults(); err != nil {
		logger.Error("Failed to initialize RBAC manager", zap.Error(err))
	}

	tenantManager := auth.NewTenantManager()

	// 初始化项目权限中间件
	projectMiddleware := middleware.NewProjectMiddleware(db, rbacManager, tenantManager, logger)

	server := &Server{
		config:            cfg,
		logger:            logger,
		loggerManager:     loggerManager,
		logStorage:        logStorage,
		logMiddleware:     logMiddleware,
		db:                db,
		keysManager:       keysManager,
		rbacManager:       rbacManager,
		tenantManager:     tenantManager,
		restHandler:       handlers.NewRestHandler(db, logger),
		authHandler:       handlers.NewAuthHandler(db, logger),
		systemHandler:     handlers.NewSystemHandler(logger),
		keyHandler:        keys.NewHandler(keysManager, logger),
		tenantHandler:     handlers.NewTenantHandler(db, logger),
		adminHandler:      handlers.NewAdminHandler(db, logger),
		projectMiddleware: projectMiddleware,
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
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return err
		}
	}

	if s.logStorage != nil {
		if err := s.logStorage.Close(); err != nil {
			s.logger.Error("Failed to close log storage", zap.Error(err))
		}
	}

	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.Error("Failed to close database", zap.Error(err))
		}
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

	// Tenant management routes (system admin only)
	r.Route("/admin/v1/tenants", func(r chi.Router) {
		// Only system admins can manage tenants
		r.Use(s.authMiddleware)
		r.Use(s.projectMiddleware.SystemAdminMiddleware)

		r.Get("/", s.tenantHandler.ListTenants)
		r.Post("/", s.tenantHandler.CreateTenant)
		r.Get("/{id}", s.tenantHandler.GetTenant)
		r.Put("/{id}", s.tenantHandler.UpdateTenant)
		r.Delete("/{id}", s.tenantHandler.DeleteTenant)
	})

	// Project management routes (project-centric)
	r.Route("/admin/v1/projects", func(r chi.Router) {
		// List projects for current user
		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)
			r.Get("/", s.tenantHandler.ListUserProjects) // User's projects across all tenants
		})

		// Individual project operations
		r.Route("/{projectId}", func(r chi.Router) {
			// View project requires viewer access
			r.Group(func(r chi.Router) {
				r.Use(s.authMiddleware)
				r.Use(s.projectMiddleware.ProjectViewerMiddleware)
				r.Get("/", s.tenantHandler.GetProject)
			})

			// Update project requires owner access
			r.Group(func(r chi.Router) {
				r.Use(s.authMiddleware)
				r.Use(s.projectMiddleware.ProjectOwnerMiddleware)
				r.Put("/", s.tenantHandler.UpdateProject)
			})

			// Delete project requires owner access
			r.Group(func(r chi.Router) {
				r.Use(s.authMiddleware)
				r.Use(s.projectMiddleware.ProjectOwnerMiddleware)
				r.Delete("/", s.tenantHandler.DeleteProject)
			})

			// Project member management requires owner or collaborator with management permissions
			r.Group(func(r chi.Router) {
				r.Use(s.authMiddleware)
				r.Use(s.projectMiddleware.CanManageProjectMiddleware)

				// Invite user to project (supports cross-tenant collaboration)
				r.Post("/invite", s.tenantHandler.InviteUserToProject)

				// List project members
				r.Get("/members", s.tenantHandler.ListProjectMembers)

				// Remove user from project
				r.Delete("/members/{userId}", s.tenantHandler.RemoveUserFromProject)

				// Transfer ownership
				r.Post("/transfer-ownership", s.tenantHandler.TransferOwnership)
			})
		})
	})

	// Project creation (tenant-based)
	r.Route("/admin/v1/tenants/{tenantId}/projects", func(r chi.Router) {
		r.Use(s.authMiddleware)
		// User must have access to the tenant to create projects
		r.Use(s.projectMiddleware.TenantAccessMiddleware)
		r.Post("/", s.tenantHandler.CreateProject)
	})

	// API Key management routes (requires auth)
	r.Route("/keys", func(r chi.Router) {
		r.Use(s.authMiddleware)
		s.keyHandler.RegisterRoutes(r)
	})

	// Log management routes (requires auth)
	r.Route("/admin/logs", func(r chi.Router) {
		r.Use(s.authMiddleware)
		logAPI := log.NewAPI(s.logStorage)
		logAPI.RegisterRoutes(r)
	})

	// General admin routes (legacy compatibility)
	r.Route("/admin", func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Get("/system/info", s.adminHandler.SystemInfo)
		r.Get("/system/stats", s.adminHandler.SystemStats)
		r.Get("/migrations/run", s.adminHandler.RunMigrations)
		r.Get("/database/backup", s.adminHandler.DatabaseBackup)

		// User management (legacy)
		r.Get("/users", s.adminHandler.ListUsers)
		r.Post("/users", s.adminHandler.CreateUser)
		r.Get("/users/{id}", s.adminHandler.GetUser)
		r.Put("/users/{id}", s.adminHandler.UpdateUser)
		r.Delete("/users/{id}", s.adminHandler.DeleteUser)

		// Tenant management (legacy - redirect to new API)
		r.Get("/tenants", s.tenantHandler.ListTenants)
		r.Post("/tenants", s.tenantHandler.CreateTenant)
		r.Get("/tenants/{id}", s.tenantHandler.GetTenant)
		r.Put("/tenants/{id}", s.tenantHandler.UpdateTenant)
		r.Delete("/tenants/{id}", s.tenantHandler.DeleteTenant)
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
	return s.corsMiddleware(s.logMiddleware.Middleware(s.logMiddleware.ComponentMiddleware("api")(handler)))
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
