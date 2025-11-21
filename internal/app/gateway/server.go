package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/guileen/metabase/internal/app/admin"
	"github.com/guileen/metabase/internal/app/api"
	"github.com/guileen/metabase/internal/app/www"
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
}

// NewConfig creates a new gateway configuration with defaults
func NewConfig() *Config {
	return &Config{
		Host:        "0.0.0.0",
		Port:        "7609",
		DevMode:     true,
		APIPort:     "7610",
		AdminPort:   "7680",
		WebPort:     "8080",
		EnableAPI:   true,
		EnableAdmin: true,
		EnableWeb:   true,
	}
}

// Server represents the main gateway server
type Server struct {
	config      *Config
	httpServer  *http.Server
	apiServer   *api.Server
	adminServer *admin.Server
	webServer   *www.Server

	// Reverse proxies
	apiProxy   *httputil.ReverseProxy
	adminProxy *httputil.ReverseProxy
	webProxy   *httputil.ReverseProxy
}

// NewServer creates a new gateway server
func NewServer(config *Config) (*Server, error) {
	if config == nil {
		config = NewConfig()
	}

	server := &Server{
		config: config,
	}

	// Create reverse proxies
	if config.EnableAPI {
		apiURL, _ := url.Parse("http://localhost:" + config.APIPort)
		server.apiProxy = httputil.NewSingleHostReverseProxy(apiURL)
	}

	if config.EnableAdmin {
		adminURL, _ := url.Parse("http://localhost:" + config.AdminPort)
		server.adminProxy = httputil.NewSingleHostReverseProxy(adminURL)
	}

	if config.EnableWeb {
		webURL, _ := url.Parse("http://localhost:" + config.WebPort)
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
				log.Printf("🚀 Starting API server on port %s", s.config.APIPort)
				if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
					log.Printf("API server error: %v", err)
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
				log.Printf("🔧 Starting admin server on port %s", s.config.AdminPort)
				if err := adminServer.Start(); err != nil && err != http.ErrServerClosed {
					log.Printf("Admin server error: %v", err)
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
			log.Printf("🌐 Starting web server on port %s", s.config.WebPort)
			if err := webServer.Start(); err != nil && err != http.ErrServerClosed {
				log.Printf("Web server error: %v", err)
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

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors occurred: %v", errors)
	}

	log.Println("🛑 MetaBase Gateway stopped gracefully")
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

	log.Printf("🔗 API Proxy: %s -> http://localhost:%s%s", r.URL.Path, s.config.APIPort, r.URL.Path)
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

	log.Printf("🔗 Admin Proxy: %s -> http://localhost:%s%s", r.URL.Path, s.config.AdminPort, r.URL.Path)
	s.adminProxy.ServeHTTP(w, r)
}

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	if s.webProxy == nil {
		http.Error(w, "Web service not available", http.StatusServiceUnavailable)
		return
	}

	// Keep the full path for docs
	log.Printf("🔗 Docs Proxy: %s -> http://localhost:%s%s", r.URL.Path, s.config.WebPort, r.URL.Path)
	s.webProxy.ServeHTTP(w, r)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if s.webProxy == nil {
		http.Error(w, "Web service not available", http.StatusServiceUnavailable)
		return
	}

	log.Printf("🔗 Search Proxy: %s -> http://localhost:%s%s", r.URL.Path, s.config.WebPort, r.URL.Path)
	s.webProxy.ServeHTTP(w, r)
}

func (s *Server) handleWeb(w http.ResponseWriter, r *http.Request) {
	if s.webProxy == nil {
		http.Error(w, "Web service not available", http.StatusServiceUnavailable)
		return
	}

	log.Printf("🔗 Web Proxy: %s -> http://localhost:%s%s", r.URL.Path, s.config.WebPort, r.URL.Path)
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
		log.Printf("Failed to write health response: %v", err)
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
		log.Printf("Failed to write root response: %v", err)
	}
}

// Helper methods
func (s *Server) printStartupInfo() {
	log.Printf("🚀 MetaBase Gateway listening on http://%s", s.httpServer.Addr)
	log.Printf("📊 Health Check: http://localhost:%s/health", s.config.Port)

	if s.config.EnableAPI {
		log.Printf("🚀 API Service: http://localhost:%s/api -> http://localhost:%s", s.config.Port, s.config.APIPort)
	}

	if s.config.EnableAdmin {
		log.Printf("🔧 Admin Interface: http://localhost:%s/admin", s.config.Port)
	}

	if s.config.EnableWeb {
		log.Printf("🌐 Documentation: http://localhost:%s/docs/overview", s.config.Port)
		log.Printf("🌐 Website: http://localhost:%s/", s.config.Port)
	}

	if s.config.DevMode {
		log.Printf("🔧 Development Mode: Enabled")
	}
}
