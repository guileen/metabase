package api

import ("context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/zap"

	"github.com/guileen/metabase/internal/services/api/keys"
	"github.com/guileen/metabase/internal/services/api/rest")

// Server API服务器
type Server struct {
	router      *chi.Router
	keyManager  *keys.Manager
	restHandler *rest.Handler
	logger      *zap.Logger
	server      *http.Server
}

// Config API服务器配置
type Config struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

// NewServer 创建API服务器
func NewServer(db *sql.DB, logger *zap.Logger, config *Config) *Server {
	if config == nil {
		config = &Config{
			Host:         "0.0.0.0",
			Port:         7609,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		}
	}

	// 创建组件
	keyManager := keys.NewManager(db, logger)
	restHandler := rest.NewHandler(db, logger)

	// 创建路由器
	router := chi.NewRouter()

	server := &Server{
		router:      router,
		keyManager:  keyManager,
		restHandler: restHandler,
		logger:      logger,
		server: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
			IdleTimeout:  config.IdleTimeout,
			Handler:      router,
		},
	}

	server.setupMiddleware()
	server.setupRoutes()

	return server
}

// setupMiddleware 设置中间件
func (s *Server) setupMiddleware() {
	// 基础中间件
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(60 * time.Second))

	// CORS中间件
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // 在生产环境中应该限制具体域名
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// 压缩中间件
	s.router.Use(middleware.Compress(5))

	// 自定义中间件
	s.router.Use(s.securityHeadersMiddleware)
	s.router.Use(s.loggingMiddleware)
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 健康检查（不需要认证）
	s.router.Get("/health", s.handleHealth)
	s.router.Get("/ping", s.handlePing)

	// API密钥管理路由
	keyHandlers := keys.NewHandlers(s.keyManager, s.logger)
	keyHandlers.RegisterRoutes(s.router)

	// REST API路由
	s.restHandler.RegisterRoutes(s.router)

	// 管理API路由
	s.setupAdminRoutes()

	// WebSocket路由（实时功能）
	s.setupWebsocketRoutes()

	// 文件服务路由
	s.setupFileRoutes()
}

// setupAdminRoutes 设置管理API路由
func (s *Server) setupAdminRoutes() {
	s.router.Route("/admin/v1", func(r chi.Router) {
		// 需要系统管理员权限
		r.Use(s.requireSystemAdmin)

		// 系统信息
		r.Get("/system/info", s.handleSystemInfo)
		r.Get("/system/stats", s.handleSystemStats)

		// 用户管理
		r.Route("/users", func(r chi.Router) {
			r.Get("/", s.handleListUsers)
			r.Post("/", s.handleCreateUser)
			r.Get("/{id}", s.handleGetUser)
			r.Put("/{id}", s.handleUpdateUser)
			r.Delete("/{id}", s.handleDeleteUser)
		})

		// 租户管理
		r.Route("/tenants", func(r chi.Router) {
			r.Get("/", s.handleListTenants)
			r.Post("/", s.handleCreateTenant)
			r.Get("/{id}", s.handleGetTenant)
			r.Put("/{id}", s.handleUpdateTenant)
			r.Delete("/{id}", s.handleDeleteTenant)
		})

		// 数据库管理
		r.Route("/database", func(r chi.Router) {
			r.Get("/tables", s.handleListTables)
			r.Post("/migrations", s.handleRunMigrations)
			r.Get("/backup", s.handleDatabaseBackup)
		})
	})
}

// setupWebsocketRoutes 设置WebSocket路由
func (s *Server) setupWebsocketRoutes() {
	s.router.Route("/ws/v1", func(r chi.Router) {
		r.Use(s.authenticateMiddleware)
		r.Get("/realtime", s.handleWebsocketRealtime)
	})
}

// setupFileRoutes 设置文件服务路由
func (s *Server) setupFileRoutes() {
	s.router.Route("/files/v1", func(r chi.Router) {
		r.Use(s.authenticateMiddleware)
		r.Post("/upload", s.handleFileUpload)
		r.Get("/{id}", s.handleFileDownload)
		r.Delete("/{id}", s.handleFileDelete)
		r.Get("/", s.handleListFiles)
	})
}

// 中间件实现

func (s *Server) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 安全头部
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		next.ServeHTTP(w, r)
	})
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 创建响应记录器
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		// 记录请求
		s.logger.Info("HTTP request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", ww.Status()),
			zap.Int("size", ww.BytesWritten()),
			zap.Duration("duration", time.Since(start)),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
		)
	})
}

func (s *Server) authenticateMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// 解析Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		// 验证API密钥
		apiKey, err := s.keyManager.GetKeyByKey(r.Context(), parts[1])
		if err != nil {
			s.logger.Warn("Invalid API key", zap.String("key", parts[1]), zap.Error(err))
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		if !apiKey.IsValid() {
			http.Error(w, "API key is invalid or expired", http.StatusUnauthorized)
			return
		}

		// 将API密钥信息添加到上下文
		ctx := context.WithValue(r.Context(), "apiKey", apiKey)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) requireSystemAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Context().Value("apiKey").(*keys.APIKey)

		// 检查是否是系统管理员密钥
		if apiKey.Type != keys.KeyTypeSystem || !apiKey.HasScope("system:write") {
			http.Error(w, "System admin access required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// 处理器实现

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现详细的健康检查
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok", "version": "1.0.0"}`))
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}

func (s *Server) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现系统信息获取
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"version": "1.0.0", "build": "dev", "uptime": "0h"}`))
}

func (s *Server) handleSystemStats(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现系统统计信息
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"requests": 0, "active_connections": 0, "memory_usage": 0}`))
}

// 占位符方法 - 需要进一步实现
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request)         {}
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request)        {}
func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request)           {}
func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request)        {}
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request)        {}

func (s *Server) handleListTenants(w http.ResponseWriter, r *http.Request)       {}
func (s *Server) handleCreateTenant(w http.ResponseWriter, r *http.Request)      {}
func (s *Server) handleGetTenant(w http.ResponseWriter, r *http.Request)          {}
func (s *Server) handleUpdateTenant(w http.ResponseWriter, r *http.Request)       {}
func (s *Server) handleDeleteTenant(w http.ResponseWriter, r *http.Request)       {}

func (s *Server) handleListTables(w http.ResponseWriter, r *http.Request)         {}
func (s *Server) handleRunMigrations(w http.ResponseWriter, r *http.Request)     {}
func (s *Server) handleDatabaseBackup(w http.ResponseWriter, r *http.Request)    {}

func (s *Server) handleWebsocketRealtime(w http.ResponseWriter, r *http.Request)  {}
func (s *Server) handleFileUpload(w http.ResponseWriter, r *http.Request)         {}
func (s *Server) handleFileDownload(w http.ResponseWriter, r *http.Request)       {}
func (s *Server) handleFileDelete(w http.ResponseWriter, r *http.Request)         {}
func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request)          {}

// Start 启动API服务器
func (s *Server) Start() error {
	s.logger.Info("Starting API server",
		zap.String("address", s.server.Addr),
	)

	return s.server.ListenAndServe()
}

// Stop 停止API服务器
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping API server")
	return s.server.Shutdown(ctx)
}

// GetRouter 获取路由器（用于集成到主服务器）
func (s *Server) GetRouter() *chi.Router {
	return s.router
}