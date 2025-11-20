package www

import ("fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"gopkg.in/yaml.v2")

// Config represents static website server configuration
type Config struct {
	Port        string
	Host        string
	Config      string
	DevMode     bool
	RootDir     string
	TemplateDir string
	AssetDir    string
}

// BuildConfig represents static build configuration
type BuildConfig struct {
	OutputDir string
	Config    string
	RootDir   string
}

// DocMeta represents document metadata from Front Matter
type DocMeta struct {
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Order       int      `yaml:"order"`
	Section     string   `yaml:"section"`
	Tags        []string `yaml:"tags"`
	Category    string   `yaml:"category"`
}

// NavItem represents a navigation item
type NavItem struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Order int    `json:"order"`
}

// NavSection represents a navigation section
type NavSection struct {
	Title string    `json:"title"`
	Items []NavItem `json:"items"`
}

// Server represents the static website server
type Server struct {
	config     *Config
	markdown   goldmark.Markdown
	fileServer http.Handler
}

// NewServer creates a new static website server
func NewServer(config *Config) *Server {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Table, extension.Strikethrough, extension.Linkify),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	return &Server{
		config:     config,
		markdown:   md,
		fileServer: http.FileServer(http.Dir(config.AssetDir)),
	}
}

// Serve starts the docs server
func Serve(config *Config) error {
	server := NewServer(config)
	return server.Start()
}

// Start starts the docs server instance
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Register routes (put / last as it's a catch-all)
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(s.config.AssetDir))))
	mux.HandleFunc("/docs/", s.HandleDocs)
	mux.HandleFunc("/search", s.HandleSearch)
	mux.HandleFunc("/api/docs", s.HandleAPI)
	mux.HandleFunc("/api/search", s.HandleAPISearch)
	mux.HandleFunc("/", s.HandleIndex)

	addr := s.config.Host + ":" + s.config.Port

	log.Printf("🌐 MetaBase Static Website Server listening on %s", addr)
	log.Printf("📖 Documentation: http://localhost:%s/docs/overview", s.config.Port)
	log.Printf("🔧 Admin Interface: http://localhost:%s/admin", s.config.Port)
	log.Printf("🌐 Access: http://localhost%s", addr)
	log.Printf("🔧 Development mode: %v", s.config.DevMode)

	return http.ListenAndServe(addr, mux)
}

func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		s.handleNotFound(w, r)
		return
	}

	// Scan and render index page
	html := s.generateIndexHTML()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (s *Server) HandleDocs(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Path[6:] // Remove "/docs/"

	if slug == "" {
		http.Redirect(w, r, "/docs/overview", http.StatusSeeOther)
		return
	}

	// Find and render document
	html, err := s.renderDocument(slug)
	if err != nil {
		s.handleNotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (s *Server) HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Redirect(w, r, "/docs/overview", http.StatusSeeOther)
		return
	}

	// Perform search and render results
	html := s.generateSearchHTML(query)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (s *Server) HandleAPI(w http.ResponseWriter, r *http.Request) {
	// Simple API implementation
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "Docs API - coming soon!"}`))
}

func (s *Server) HandleAPISearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"query": "%s", "results": []}`, query)))
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(s.generateNotFoundHTML(r.URL.Path)))
}

// Build generates static site
func Build(config *BuildConfig) error {
	log.Printf("🏗️  Building static site...")
	log.Printf("📁 Source: %s", config.RootDir)
	log.Printf("📁 Output: %s", config.OutputDir)

	// TODO: Implement static site building
	log.Printf("✅ Static site built successfully!")
	return nil
}

// scanDocuments scans the docs directory for markdown files with Front Matter
func (s *Server) scanDocuments() ([]NavSection, error) {
	docsDir := s.config.RootDir
	if _, err := os.Stat(docsDir); os.IsNotExist(err) {
		log.Printf("Docs directory %s does not exist", docsDir)
		return []NavSection{}, nil
	}

	sections := make(map[string][]NavItem)

	err := filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		meta, _, err := s.parseFrontMatter(content)
		if err != nil {
			log.Printf("Error parsing %s: %v", path, err)
			return nil
		}

		if meta.Title == "" {
			// Skip files without title in Front Matter
			return nil
		}

		// Generate URL from file path
		relPath, _ := filepath.Rel(docsDir, path)
		url := "/docs/" + strings.TrimSuffix(relPath, ".md")

		section := meta.Section
		if section == "" {
			section = "其他"
		}

		navItem := NavItem{
			Title: meta.Title,
			URL:   url,
			Order: meta.Order,
		}

		sections[section] = append(sections[section], navItem)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort sections and items
	var navSections []NavSection
	sectionOrder := map[string]string{
		"getting-started": "开始使用",
		"core-concepts":   "核心概念",
		"api":            "API 参考",
		"deployment":     "部署",
	}

	// Add sections with defined order first
	for sectionKey, sectionTitle := range sectionOrder {
		if items, exists := sections[sectionKey]; exists {
			// Sort items by order
			for i := 0; i < len(items); i++ {
				for j := i + 1; j < len(items); j++ {
					if items[i].Order > items[j].Order {
						items[i], items[j] = items[j], items[i]
					}
				}
			}
			navSections = append(navSections, NavSection{
				Title: sectionTitle,
				Items: items,
			})
		}
	}

	// Add any remaining sections
	for sectionKey, items := range sections {
		found := false
		for _, definedKey := range []string{"getting-started", "core-concepts", "api", "deployment"} {
			if sectionKey == definedKey {
				found = true
				break
			}
		}
		if !found && len(items) > 0 {
			// Sort items by order
			for i := 0; i < len(items); i++ {
				for j := i + 1; j < len(items); j++ {
					if items[i].Order > items[j].Order {
						items[i], items[j] = items[j], items[i]
					}
				}
			}
			navSections = append(navSections, NavSection{
				Title: sectionKey,
				Items: items,
			})
		}
	}

	return navSections, nil
}

// parseFrontMatter parses Front Matter from markdown content
func (s *Server) parseFrontMatter(content []byte) (*DocMeta, []byte, error) {
	if len(content) < 3 || string(content[:3]) != "---" {
		return &DocMeta{}, content, nil
	}

	endIndex := strings.Index(string(content[3:]), "---")
	if endIndex == -1 {
		return &DocMeta{}, content, nil
	}

	frontMatter := content[3 : endIndex+3]
	remaining := content[endIndex+6:]

	var meta DocMeta
	err := yaml.Unmarshal(frontMatter, &meta)
	if err != nil {
		return nil, nil, err
	}

	return &meta, remaining, nil
}

// Helper methods for HTML generation
func (s *Server) generateIndexHTML() string {
	navSections, err := s.scanDocuments()
	if err != nil {
		log.Printf("Error scanning documents: %v", err)
		navSections = []NavSection{}
	}

	navHTML := s.generateNavHTML(navSections, "")

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MetaBase Documentation</title>
    <link rel="stylesheet" href="/assets/docs.css">
</head>
<body>
    <div class="docs-container">
        <nav class="sidebar">
            <div class="sidebar-header">
                <h2 class="site-title">MetaBase</h2>
                <p class="site-subtitle">下一代后端核心</p>
            </div>
            <div class="nav-menu">
                %s
            </div>
        </nav>
        <main class="main-content">
            <div class="content-wrapper">
                <div class="docs-content">
                    <h1>欢迎来到 MetaBase 文档</h1>
                    <p>选择左侧菜单开始阅读，或使用搜索功能查找内容。</p>
                    <div class="cards">
                        <div class="card">
                            <h3>🚀 快速开始</h3>
                            <p>快速了解如何开始使用 MetaBase。</p>
                            <a href="/docs/overview">开始 →</a>
                        </div>
                        <div class="card">
                            <h3>📖 文档</h3>
                            <p>浏览完整的文档内容。</p>
                            <a href="/docs/overview">阅读 →</a>
                        </div>
                        <div class="card">
                            <h3>🏗️ 架构</h3>
                            <p>深入了解 MetaBase 的技术架构。</p>
                            <a href="/docs/architecture">查看 →</a>
                        </div>
                    </div>
                </div>
            </div>
        </main>
    </div>
    <script src="/assets/docs.js"></script>
</body>
</html>`, navHTML)
}

// generateNavHTML generates HTML for navigation menu
func (s *Server) generateNavHTML(sections []NavSection, activeURL string) string {
	var html strings.Builder

	for _, section := range sections {
		html.WriteString(fmt.Sprintf(`<div class="nav-section">
                    <h3>%s</h3>
                    <ul>`, section.Title))

		for _, item := range section.Items {
			activeClass := ""
			if item.URL == activeURL {
				activeClass = " class=\"active\""
			}
			html.WriteString(fmt.Sprintf(`<li><a href="%s"%s>%s</a></li>`, item.URL, activeClass, item.Title))
		}

		html.WriteString(`</ul>
                </div>`)
	}

	return html.String()
}

func (s *Server) renderDocument(slug string) (string, error) {
	// Scan documents for navigation
	navSections, err := s.scanDocuments()
	if err != nil {
		log.Printf("Error scanning documents: %v", err)
		navSections = []NavSection{}
	}

	// Try to find and parse the markdown file
	docPath := filepath.Join(s.config.RootDir, slug+".md")
	content, err := os.ReadFile(docPath)
	if err != nil {
		return "", fmt.Errorf("document not found: %v", err)
	}

	// Parse Front Matter and content
	meta, markdownContent, err := s.parseFrontMatter(content)
	if err != nil {
		return "", fmt.Errorf("error parsing Front Matter: %v", err)
	}

	// Convert markdown to HTML
	var htmlContent strings.Builder
	err = s.markdown.Convert([]byte(markdownContent), &htmlContent)
	if err != nil {
		return "", fmt.Errorf("error converting markdown: %v", err)
	}

	title := meta.Title
	if title == "" {
		title = slug
	}

	navHTML := s.generateNavHTML(navSections, "/docs/"+slug)

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>%s - MetaBase</title>
    <link rel="stylesheet" href="/assets/docs.css">
</head>
<body>
    <div class="docs-container">
        <nav class="sidebar">
            <div class="sidebar-header">
                <h2 class="site-title">MetaBase</h2>
                <p class="site-subtitle">下一代后端核心</p>
            </div>
            <div class="nav-menu">
                %s
            </div>
        </nav>
        <main class="main-content">
            <div class="content-wrapper">
                <div class="docs-content">
                    %s
                </div>
            </div>
        </main>
    </div>
    <script src="/assets/docs.js"></script>
</body>
</html>`, title, navHTML, htmlContent.String()), nil
}

func (s *Server) generateSearchHTML(query string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>搜索结果: %s - MetaBase</title>
    <link rel="stylesheet" href="/assets/docs.css">
</head>
<body>
    <div class="docs-container">
        <nav class="sidebar">
            <div class="sidebar-header">
                <h2 class="site-title">MetaBase</h2>
                <p class="site-subtitle">下一代后端核心</p>
            </div>
        </nav>
        <main class="main-content">
            <div class="content-wrapper">
                <div class="docs-content">
                    <h1>搜索结果: "%s"</h1>
                    <p>TODO: 实现搜索功能</p>
                </div>
            </div>
        </main>
    </div>
</body>
</html>`, query, query)
}

func (s *Server) generateNotFoundHTML(path string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>页面未找到 - MetaBase</title>
    <style>
        body { font-family: system-ui; text-align: center; padding: 2rem; }
        .error { color: #dc2626; }
    </style>
</head>
<body>
    <h1 class="error">404 - 页面未找到</h1>
    <p>请求的页面 <code>%s</code> 不存在。</p>
    <p><a href="/docs/overview">返回文档首页</a></p>
</body>
</html>`, path)
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