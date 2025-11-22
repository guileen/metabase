package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/guileen/metabase/internal/app/admin"
	"github.com/guileen/metabase/internal/app/api"
	"github.com/guileen/metabase/internal/app/www"
	"github.com/guileen/metabase/internal/pkg/banner"
	"github.com/guileen/metabase/pkg/config"
	"github.com/guileen/metabase/pkg/log"
)

// Config represents the gateway server configuration
type Config struct {
	Host    string `json:"host"`
	Port    string `json:"port"`
	DevMode bool   `json:"dev_mode"`

	// Service ports
	APIPort   string `json:"api_port"`
	AdminPort string `json:"admin_port"`
	WebPort   string `json:"web_port"`

	// Service flags
	EnableAPI   bool `json:"enable_api"`
	EnableAdmin bool `json:"enable_admin"`
	EnableWeb   bool `json:"enable_web"`

	// Logging configuration
	LogConfig *config.LoggingConfig `json:"log_config,omitempty"`
}

// NewConfig creates a new gateway configuration with defaults and environment variables
func NewConfig() *Config {
	// Initialize global config to load environment variables
	appConfig := config.Get()

	cfg := &Config{
		Host:        appConfig.GetString("server.host"),
		DevMode:     appConfig.GetBool("server.dev_mode"),
		APIPort:     strconv.Itoa(appConfig.GetInt("server.api_port")),
		AdminPort:   strconv.Itoa(appConfig.GetInt("server.admin_port")),
		WebPort:     strconv.Itoa(appConfig.GetInt("server.web_port")),
		EnableAPI:   appConfig.GetBool("services.enable_api"),
		EnableAdmin: appConfig.GetBool("services.enable_admin"),
		EnableWeb:   appConfig.GetBool("services.enable_web"),
	}

	// Use gateway port if available, otherwise use server port
	if appConfig.GetInt("server.gateway_port") > 0 {
		cfg.Port = strconv.Itoa(appConfig.GetInt("server.gateway_port"))
	} else {
		cfg.Port = strconv.Itoa(appConfig.GetInt("server.port"))
	}

	// Override with environment variables if explicitly set for development
	if appConfig.GetBool("server.dev_mode") {
		if cfg.Host == "" || cfg.Host == "localhost" {
			cfg.Host = "0.0.0.0" // For development, allow external connections
		}
	}

	return cfg
}

// Server represents the main gateway server
type Server struct {
	config      *Config
	httpServer  *http.Server
	apiServer   *api.Server
	adminServer *admin.Server
	webServer   *www.Server

	// Logging
	loggerManager *log.Logger
	logStorage    *log.LogStorage
	logMiddleware *log.Middleware

	// Reverse proxies
	apiProxy   *httputil.ReverseProxy
	adminProxy *httputil.ReverseProxy
	webProxy   *httputil.ReverseProxy
}

// NewServer creates a new gateway server
func NewServer(cfg *Config) (*Server, error) {
	if cfg == nil {
		cfg = NewConfig()
	}

	// 初始化日志系统
	if cfg.LogConfig == nil {
		cfg.LogConfig = &config.LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			File:       "./logs/gateway.log",
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
		ServiceName:         "gateway",
		StoreLogs:           true,
		LogRequestBody:      false,
		LogResponseBody:     false,
		MaxBodySize:         1024 * 1024,
		GenerateTraceID:     true,
		LogStatus:           "all",
		SkipPaths:           []string{"/health"},
		DefaultFields:       map[string]interface{}{},
		MeasureResponseTime: true,
	}
	logMiddleware := log.NewMiddlewareWithConfigAndStorage(loggerManager, middlewareConfig, logStorage)

	server := &Server{
		config:        cfg,
		loggerManager: loggerManager,
		logStorage:    logStorage,
		logMiddleware: logMiddleware,
	}

	// Create reverse proxies
	if cfg.EnableAPI {
		apiURL, _ := url.Parse("http://localhost:" + cfg.APIPort)
		server.apiProxy = httputil.NewSingleHostReverseProxy(apiURL)
	}

	if cfg.EnableAdmin {
		adminURL, _ := url.Parse("http://localhost:" + cfg.AdminPort)
		server.adminProxy = httputil.NewSingleHostReverseProxy(adminURL)
	}

	if cfg.EnableWeb {
		webURL, _ := url.Parse("http://localhost:" + cfg.WebPort)
		server.webProxy = httputil.NewSingleHostReverseProxy(webURL)
	}

	return server, nil
}

// Start starts the gateway server and its dependent services
func (s *Server) Start() error {
	var errors []error

	// Start API server
	if s.config.EnableAPI {
		apiConfig := &api.Config{
			Host:    "localhost",
			Port:    s.config.APIPort,
			DevMode: s.config.DevMode,
		}

		apiServer, err := api.NewServer(apiConfig)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to create API server: %w", err))
		} else {
			s.apiServer = apiServer
			go func() {
				if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
					s.loggerManager.Error("API server error", "error", err)
				}
			}()
		}
	}

	// Start admin server
	if s.config.EnableAdmin {
		adminConfig := &admin.Config{
			Host:        "localhost",
			Port:        s.config.AdminPort,
			DevMode:     s.config.DevMode,
			StaticFiles: "./web/admin",
		}

		adminServer, err := admin.NewServer(adminConfig)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to create admin server: %w", err))
		} else {
			s.adminServer = adminServer
			go func() {
				if err := adminServer.Start(); err != nil && err != http.ErrServerClosed {
					s.loggerManager.Error("Admin server error", "error", err)
				}
			}()
		}
	}

	// Start web server
	if s.config.EnableWeb {
		webConfig := &www.Config{
			Host:    "localhost",
			Port:    s.config.WebPort,
			DevMode: s.config.DevMode,
		}

		webServer := www.NewServer(webConfig)
		s.webServer = webServer
		go func() {
			if err := webServer.Start(); err != nil && err != http.ErrServerClosed {
				s.loggerManager.Error("Web server error", "error", err)
			}
		}()
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to start services: %v", errors)
	}

	// Give services a moment to start
	time.Sleep(100 * time.Millisecond)

	// Create main HTTP server
	mux := http.NewServeMux()
	s.setupRoutes(mux)

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

// Stop gracefully stops the gateway and its services
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var errors []error

	// Stop main HTTP server
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("gateway server shutdown: %w", err))
		}
	}

	// Stop API server
	if s.apiServer != nil {
		if err := s.apiServer.Stop(ctx); err != nil {
			errors = append(errors, fmt.Errorf("API server shutdown: %w", err))
		}
	}

	// Stop admin server
	if s.adminServer != nil {
		if err := s.adminServer.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("admin server shutdown: %w", err))
		}
	}

	// Close log storage
	if s.logStorage != nil {
		if err := s.logStorage.Close(); err != nil {
			errors = append(errors, fmt.Errorf("log storage close: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors occurred: %v", errors)
	}

	s.loggerManager.Info("MetaBase Gateway stopped gracefully")
	return nil
}

// setupRoutes configures the main routing
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// API routes - proxy to API server
	if s.config.EnableAPI {
		mux.HandleFunc("/api/", s.handleAPI)
	}

	// Admin routes - proxy to admin server
	if s.config.EnableAdmin {
		mux.HandleFunc("/admin/", s.handleAdmin)
	}

	// Documentation routes - proxy to web server
	if s.config.EnableWeb {
		mux.HandleFunc("/docs/", s.handleDocs)
		mux.HandleFunc("/search", s.handleSearch)
	}

	// Static assets - serve directly
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/assets"))))

	// Health check
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ping", s.handlePing)

	// Root route - proxy to web server for the main website
	if s.config.EnableWeb {
		mux.HandleFunc("/", s.handleWeb)
	} else {
		mux.HandleFunc("/", s.handleRoot)
	}
}

// Route handlers
func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	if s.apiProxy == nil {
		http.Error(w, "API service not available", http.StatusServiceUnavailable)
		return
	}

	// Remove /api prefix and proxy
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api")

	s.loggerManager.Info("API Proxy", "path", r.URL.Path, "target", "http://localhost:"+s.config.APIPort+r.URL.Path)
	s.apiProxy.ServeHTTP(w, r)
}

func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	if s.adminProxy == nil {
		http.Error(w, "Admin service not available", http.StatusServiceUnavailable)
		return
	}

	// Remove /admin prefix and proxy
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/admin")
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}

	s.loggerManager.Info("Admin Proxy", "path", r.URL.Path, "target", "http://localhost:"+s.config.AdminPort+r.URL.Path)
	s.adminProxy.ServeHTTP(w, r)
}

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	if s.webProxy == nil {
		http.Error(w, "Web service not available", http.StatusServiceUnavailable)
		return
	}

	// Keep the full path for docs
	s.loggerManager.Info("Docs Proxy", "path", r.URL.Path, "target", "http://localhost:"+s.config.WebPort+r.URL.Path)
	s.webProxy.ServeHTTP(w, r)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if s.webProxy == nil {
		http.Error(w, "Web service not available", http.StatusServiceUnavailable)
		return
	}

	s.loggerManager.Info("Search Proxy", "path", r.URL.Path, "target", "http://localhost:"+s.config.WebPort+r.URL.Path)
	s.webProxy.ServeHTTP(w, r)
}

func (s *Server) handleWeb(w http.ResponseWriter, r *http.Request) {
	if s.webProxy == nil {
		http.Error(w, "Web service not available", http.StatusServiceUnavailable)
		return
	}

	s.loggerManager.Info("Web Proxy", "path", r.URL.Path, "target", "http://localhost:"+s.config.WebPort+r.URL.Path)
	s.webProxy.ServeHTTP(w, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
		"services": map[string]interface{}{
			"api":   s.config.EnableAPI,
			"admin": s.config.EnableAdmin,
			"web":   s.config.EnableWeb,
		},
		"endpoints": map[string]string{
			"health": "/health",
			"api":    "/api/v1/health",
			"admin":  "/admin/",
			"docs":   "/docs/",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := writeJSON(w, health); err != nil {
		s.loggerManager.Error("Failed to write health response", "error", err)
	}
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("pong"))
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Serve API info
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"name":        "MetaBase Gateway",
		"description": "下一代后端核心 - 统一网关",
		"version":     "1.0.0",
		"services": map[string]string{
			"api":   fmt.Sprintf("http://localhost:%s/api", s.config.Port),
			"admin": fmt.Sprintf("http://localhost:%s/admin", s.config.Port),
		},
		"endpoints": map[string]string{
			"health":   "/health",
			"api_base": "/api/v1",
			"admin":    "/admin",
		},
	}

	if err := writeJSON(w, response); err != nil {
		s.loggerManager.Error("Failed to write root response", "error", err)
	}
}

// Helper methods
func (s *Server) printStartupInfo() {
	// 构建服务信息
	services := []banner.ServiceInfo{
		{Name: "Gateway", Status: "running", Port: s.config.Port, Color: banner.BrightCyan},
	}

	if s.config.EnableAPI {
		services = append(services, banner.ServiceInfo{
			Name: "API", Status: "running", Port: s.config.APIPort, Color: banner.BrightGreen,
		})
	}

	if s.config.EnableAdmin {
		services = append(services, banner.ServiceInfo{
			Name: "Admin", Status: "running", Port: s.config.AdminPort, Color: banner.BrightYellow,
		})
	}

	if s.config.EnableWeb {
		services = append(services, banner.ServiceInfo{
			Name: "Web", Status: "running", Port: s.config.WebPort, Color: banner.BrightMagenta,
		})
	}

	// 构建访问链接
	accessLinks := []banner.AccessLink{
		{Name: "Health Check", URL: fmt.Sprintf("http://localhost:%s/health", s.config.Port), Desc: "服务健康状态", Color: banner.BrightGreen},
	}

	if s.config.EnableAPI {
		accessLinks = append(accessLinks, banner.AccessLink{
			Name: "API Service", URL: fmt.Sprintf("http://localhost:%s/api", s.config.Port), Desc: "REST API 接口", Color: banner.BrightBlue,
		})
	}

	if s.config.EnableAdmin {
		accessLinks = append(accessLinks, banner.AccessLink{
			Name: "Admin Panel", URL: fmt.Sprintf("http://localhost:%s/admin", s.config.Port), Desc: "管理控制台", Color: banner.BrightYellow,
		})
	}

	if s.config.EnableWeb {
		accessLinks = append(accessLinks, banner.AccessLink{
			Name: "Documentation", URL: fmt.Sprintf("http://localhost:%s/docs/overview", s.config.Port), Desc: "产品文档", Color: banner.BrightMagenta,
		})
		accessLinks = append(accessLinks, banner.AccessLink{
			Name: "Website", URL: fmt.Sprintf("http://localhost:%s/", s.config.Port), Desc: "主站首页", Color: banner.BrightCyan,
		})
	}

	// 打印启动信息
	startupInfo := &banner.StartupInfo{
		Services:    services,
		AccessLinks: accessLinks,
		DevMode:     s.config.DevMode,
		StartTime:   time.Now(),
	}

	banner.PrintStartupInfo(startupInfo)
}

// withMiddleware applies global middleware to the HTTP handler
func (s *Server) withMiddleware(handler http.Handler) http.Handler {
	return s.logMiddleware.Middleware(s.logMiddleware.ComponentMiddleware("gateway")(handler))
}

// writeJSON writes JSON data to the HTTP response
func writeJSON(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}
