package gateway

import (
	"log"
	"net/http"
	"net/url"

	"github.com/metabase/metabase/internal/www"
	"github.com/metabase/metabase/internal/server"
)

// Config represents unified gateway configuration
type Config struct {
	Port    string
	Host    string
	DevMode bool
}

// Gateway represents the unified MetaBase gateway
type Gateway struct {
	config     *Config
	wwwServer  *www.Server
	coreServer *server.Server
}

// NewGateway creates a new unified gateway instance
func NewGateway(config *Config) *Gateway {
	// Create www server config
	wwwConfig := &www.Config{
		Port:        config.Port,
		Host:        config.Host,
		DevMode:     config.DevMode,
		RootDir:     "docs",
		TemplateDir: "templates",
		AssetDir:    "web/assets",
	}

	// Create core server config
	coreConfig := &server.Config{
		Port:    config.Port,
		Host:    config.Host,
		DevMode: config.DevMode,
	}

	return &Gateway{
		config:     config,
		wwwServer:  www.NewServer(wwwConfig),
		coreServer: server.NewServer(coreConfig),
	}
}

// Start starts the unified gateway
func Start(config *Config) error {
	gateway := NewGateway(config)
	return gateway.Start()
}

// Start starts the unified gateway instance
func (g *Gateway) Start() error {
	mux := http.NewServeMux()

	// Register routes with unique responsibility
	// 1. Admin routes - ONLY handles /admin/*
	mux.HandleFunc("/admin", g.handleAdmin)
	mux.HandleFunc("/admin/", g.handleAdmin)

	// 2. Documentation routes - ONLY handles /docs/*
	mux.HandleFunc("/docs/", g.wwwServer.HandleDocs)
	mux.HandleFunc("/search", g.wwwServer.HandleSearch)
	mux.HandleFunc("/api/docs", g.wwwServer.HandleAPI)
	mux.HandleFunc("/api/search", g.wwwServer.HandleAPISearch)

	// 3. API routes - ONLY handles /api/* (excluding /api/docs*)
	mux.HandleFunc("/api/", g.handleAPI)

	// 4. Static assets - ONLY handles /assets/*
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/assets"))))

	// 5. Root routes - ONLY handles /
	mux.HandleFunc("/", g.wwwServer.HandleIndex)

	addr := g.config.Host + ":" + g.config.Port

	// Print unified startup message
	log.Printf("🚀 MetaBase Unified Gateway listening on %s", addr)
	log.Printf("📖 Documentation: http://localhost:%s/docs/overview", g.config.Port)
	log.Printf("🔧 Admin Interface: http://localhost:%s/admin", g.config.Port)
	log.Printf("🌐 Access: \033[34;1mhttp://localhost:%s\033[0m", g.config.Port)
	log.Printf("💡 All services unified under single gateway")

	return http.ListenAndServe(addr, mux)
}

// handleAdmin serves the admin interface (unique responsibility)
func (g *Gateway) handleAdmin(w http.ResponseWriter, r *http.Request) {
	// Remove /admin prefix and create new request
	path := r.URL.Path[6:] // Remove "/admin"
	if path == "" {
		path = "index.html"
	}

	// Create a new request with the modified path
	newPath := "/" + path
	req := &http.Request{
		Method: r.Method,
		URL:    &url.URL{Path: newPath},
		Header: r.Header,
	}

	// Serve admin static files with the corrected path
	fs := http.FileServer(http.Dir("admin"))
	fs.ServeHTTP(w, req)
}

// handleAPI serves core API routes (unique responsibility)
func (g *Gateway) handleAPI(w http.ResponseWriter, r *http.Request) {
	// Exclude /api/docs routes which are handled by www server
	if r.URL.Path == "/api/docs" || r.URL.Path == "/api/search" {
		g.wwwServer.HandleAPI(w, r)
		return
	}

	// Handle core API routes here
	// For now, return a simple response
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "MetaBase API - Coming Soon", "version": "1.0.0"}`))
}