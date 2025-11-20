package server

import ("fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html")


// Server represents the MetaBase core server
type Server struct {
	config *Config
	markdown goldmark.Markdown
}

// NewServer creates a new server instance
func NewServer(config *Config) (*Server, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Table, extension.Strikethrough, extension.Linkify),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	return &Server{
		config:   config,
		markdown: md,
	}, nil
}

// Start starts the server
func Start(config *Config) error {
	server, err := NewServer(config)
	if err != nil {
		return err
	}
	return server.Start()
}

// Start starts the server instance
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Register routes - more specific routes first
	mux.HandleFunc("/admin", s.handleAdmin)
	mux.HandleFunc("/admin/", s.handleAdmin)
	mux.HandleFunc("/md/", s.handleMarkdown)
	mux.Handle("/assets/", http.FileServer(http.Dir("web")))
	mux.HandleFunc("/", s.handleRoot)

	addr := s.config.Host + ":" + s.config.Port

	log.Printf("🚀 MetaBase Core Server listening on %s", addr)
	log.Printf("📖 Documentation: http://localhost:8080/docs/overview")
	log.Printf("🔧 Admin Interface: http://localhost:%s/admin", s.config.Port)
	log.Printf("🌐 Access: \033[34;1mhttp://localhost:%s\033[0m", s.config.Port)
	log.Printf("💡 Tip: Start docs server with: metabase docs serve")

	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		// Serve home page or redirect to docs
		http.Redirect(w, r, "http://localhost:8080/docs/overview", http.StatusTemporaryRedirect)
		return
	}

	// Serve static files
	http.FileServer(http.Dir("web")).ServeHTTP(w, r)
}

func (s *Server) handleMarkdown(w http.ResponseWriter, r *http.Request) {
	// Simple markdown rendering for compatibility
	slug := r.URL.Path[4:] // Remove "/md/"

	if slug == "" {
		slug = "overview"
	}

	content := fmt.Sprintf(`# %s

这是 %s 的简单视图。

如需查看完整文档，请访问：
- [完整文档](http://localhost:8080/docs/%s)
- [返回首页](/)

## 功能特性

- ✅ 简单的 Markdown 渲染
- ✅ 基础的静态文件服务
- ✅ 轻量级设计
- ✅ 专注核心功能

如需完整功能（导航、搜索、主题等），请使用文档服务器：
<pre><code>metabase docs serve</code></pre>
`, slug, slug, slug)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(s.simpleHTMLWrapper(content, slug)))
}

func (s *Server) simpleHTMLWrapper(content, slug string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - MetaBase</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            line-height: 1.6;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background: #f9fafb;
        }
        .container {
            background: white;
            padding: 2rem;
            border-radius: 8px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        h1, h2, h3 { color: #1f2937; }
        h1 { border-bottom: 2px solid #e5e7eb; padding-bottom: 0.5rem; }
        code { background: #f3f4f6; padding: 2px 6px; border-radius: 3px; font-size: 0.875em; }
        pre {
            background: #1f2937;
            color: #f9fafb;
            padding: 1rem;
            border-radius: 6px;
            overflow-x: auto;
            font-size: 0.875em;
        }
        .nav {
            margin-bottom: 2rem;
            padding: 1rem;
            background: #f3f4f6;
            border-radius: 6px;
        }
        .nav a {
            margin-right: 1rem;
            color: #2563eb;
            text-decoration: none;
            font-weight: 500;
        }
        .nav a:hover { text-decoration: underline; }
        .footer {
            margin-top: 3rem;
            padding-top: 1rem;
            border-top: 1px solid #e5e7eb;
            color: #6b7280;
            font-size: 0.875em;
        }
        .tip {
            background: #fef3c7;
            border: 1px solid #f59e0b;
            padding: 1rem;
            border-radius: 6px;
            margin: 1rem 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <nav class="nav">
            <a href="/">🏠 首页</a>
            <a href="http://localhost:8080/docs/overview">📚 完整文档</a>
            <a href="/admin">🔧 管理后台</a>
            <a href="https://github.com/guileen/metabase">🔧 GitHub</a>
        </nav>

        %s

        <div class="tip">
            💡 <strong>提示:</strong> 这是简化视图。使用 <code>metabase docs serve</code> 启动完整文档服务器，获得导航、搜索、主题等专业功能。
        </div>

        <footer class="footer">
            <p>🚀 <strong>MetaBase</strong> - 下一代后端核心 |
               <a href="http://localhost:8080/docs/overview">完整文档</a> |
               <a href="/admin">管理后台</a> |
               <a href="https://github.com/guileen/metabase">GitHub</a>
            </p>
        </footer>
    </div>
</body>
</html>`, slug, content)
}

// handleAdmin serves the admin interface
func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
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

// Stop stops the server gracefully
func (s *Server) Stop() error {
	// In a real implementation, this would gracefully shutdown the HTTP server
	// For now, just log the shutdown
	log.Println("🛑 Server stopped")
	return nil
}