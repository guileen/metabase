package cms

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Installer handles CMS installation and initialization
type Installer struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewInstaller creates a new CMS installer
func NewInstaller(db *sql.DB, logger *zap.Logger) *Installer {
	return &Installer{
		db:     db,
		logger: logger,
	}
}

// InstallRequest represents the CMS installation request
type InstallRequest struct {
	// Basic site information
	SiteTitle        string `json:"site_title" validate:"required"`
	SiteDescription  string `json:"site_description"`
	SiteURL          string `json:"site_url" validate:"required,url"`
	AdminEmail       string `json:"admin_email" validate:"required,email"`
	AdminPassword    string `json:"admin_password" validate:"required,min=8"`
	Timezone         string `json:"timezone"`
	Language         string `json:"language"`

	// CMS Features
	EnableComments   bool `json:"enable_comments"`
	EnableRatings    bool `json:"enable_ratings"`
	EnableSearch     bool `json:"enable_search"`
	EnableCategories bool `json:"enable_categories"`
	EnableTags       bool `json:"enable_tags"`
	EnableMedia      bool `json:"enable_media"`
	EnableSEO        bool `json:"enable_seo"`

	// Theme settings
	PrimaryColor     string `json:"primary_color"`
	SecondaryColor   string `json:"secondary_color"`
	FontFamily       string `json:"font_family"`
	HeaderStyle      string `json:"header_style"`
}

// InstallResponse represents the CMS installation response
type InstallResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	SiteURL      string `json:"site_url"`
	AdminURL     string `json:"admin_url"`
	InstalledAt  string `json:"installed_at"`
	Version      string `json:"version"`
}

// CheckInstallation checks if CMS is already installed
func (i *Installer) CheckInstallation(ctx context.Context) (bool, error) {
	var count int
	err := i.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'cms_settings'").Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check installation: %w", err)
	}

	if count == 0 {
		return false, nil
	}

	// Check if installation is completed
	var settingCount int
	err = i.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM cms_settings WHERE key = 'site_installed'").Scan(&settingCount)

	if err != nil {
		return false, fmt.Errorf("failed to check installation status: %w", err)
	}

	return settingCount > 0, nil
}

// Install performs the CMS installation
func (i *Installer) Install(ctx context.Context, req *InstallRequest, tenantID string) (*InstallResponse, error) {
	i.logger.Info("Starting CMS installation",
		zap.String("tenant_id", tenantID),
		zap.String("site_title", req.SiteTitle))

	// Validate request
	if err := i.validateInstallRequest(req); err != nil {
		return &InstallResponse{
			Success: false,
			Message: fmt.Sprintf("Validation failed: %v", err),
		}, nil
	}

	// Check if already installed
	installed, err := i.CheckInstallation(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check installation status: %w", err)
	}

	if installed {
		return &InstallResponse{
			Success: false,
			Message: "CMS is already installed",
		}, nil
	}

	// Begin transaction
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Run installation steps
	if err := i.runMigrations(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	if err := i.createDefaultContentTypes(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("failed to create default content types: %w", err)
	}

	if err := i.insertDefaultSettings(ctx, tx, tenantID, req); err != nil {
		return nil, fmt.Errorf("failed to insert default settings: %w", err)
	}

	if err := i.createDefaultCategories(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("failed to create default categories: %w", err)
	}

	if err := i.createSampleContent(ctx, tx, tenantID, req); err != nil {
		return nil, fmt.Errorf("failed to create sample content: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit installation: %w", err)
	}

	i.logger.Info("CMS installation completed successfully",
		zap.String("tenant_id", tenantID),
		zap.String("site_title", req.SiteTitle))

	return &InstallResponse{
		Success:     true,
		Message:     "CMS installed successfully",
		SiteURL:     req.SiteURL,
		AdminURL:    fmt.Sprintf("%s/admin", req.SiteURL),
		InstalledAt: time.Now().Format(time.RFC3339),
		Version:     "1.0.0",
	}, nil
}

// validateInstallRequest validates the installation request
func (i *Installer) validateInstallRequest(req *InstallRequest) error {
	if req.SiteTitle == "" {
		return fmt.Errorf("site title is required")
	}
	if req.SiteURL == "" {
		return fmt.Errorf("site URL is required")
	}
	if req.AdminEmail == "" {
		return fmt.Errorf("admin email is required")
	}
	if req.AdminPassword == "" {
		return fmt.Errorf("admin password is required")
	}
	if len(req.AdminPassword) < 8 {
		return fmt.Errorf("admin password must be at least 8 characters")
	}

	// Set defaults
	if req.Timezone == "" {
		req.Timezone = "UTC"
	}
	if req.Language == "" {
		req.Language = "en"
	}
	if req.PrimaryColor == "" {
		req.PrimaryColor = "#3b82f6"
	}
	if req.SecondaryColor == "" {
		req.SecondaryColor = "#64748b"
	}
	if req.FontFamily == "" {
		req.FontFamily = "system-ui, -apple-system, sans-serif"
	}
	if req.HeaderStyle == "" {
		req.HeaderStyle = "default"
	}

	return nil
}

// runMigrations runs the CMS database migrations
func (i *Installer) runMigrations(ctx context.Context, tx *sql.Tx) error {
	// Read migration file
	migrationPath := filepath.Join("internal", "cms", "migrations.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Execute migration
	_, err = tx.ExecContext(ctx, string(migrationSQL))
	if err != nil {
		return fmt.Errorf("failed to execute migrations: %w", err)
	}

	return nil
}

// createDefaultContentTypes creates default content types
func (i *Installer) createDefaultContentTypes(ctx context.Context, tx *sql.Tx, tenantID string) error {
	now := time.Now()
	contentTypes := []struct {
		name        string
		slug        string
		description string
		icon        string
		color       string
		hierarchical bool
		hasCategories bool
		hasTags       bool
		hasComments   bool
		hasMedia      bool
		hasRatings    bool
		hasWorkflow   bool
		autoPublish   bool
		hasSEO        bool
	}{
		{
			name:        "Blog Posts",
			slug:        "blog-posts",
			description: "Blog posts and articles",
			icon:        "article",
			color:       "#3b82f6",
			hierarchical: false,
			hasCategories: true,
			hasTags:       true,
			hasComments:   true,
			hasMedia:      true,
			hasRatings:    true,
			hasWorkflow:   true,
			autoPublish:   false,
			hasSEO:        true,
		},
		{
			name:        "Pages",
			slug:        "pages",
			description: "Static pages like About, Contact, etc.",
			icon:        "file-text",
			color:       "#10b981",
			hierarchical: true,
			hasCategories: false,
			hasTags:       false,
			hasComments:   false,
			hasMedia:      false,
			hasRatings:    false,
			hasWorkflow:   true,
			autoPublish:   false,
			hasSEO:        true,
		},
		{
			name:        "Forum Topics",
			slug:        "forum-topics",
			description: "Discussion forum topics and posts",
			icon:        "message-square",
			color:       "#f59e0b",
			hierarchical: false,
			hasCategories: true,
			hasTags:       true,
			hasComments:   true,
			hasMedia:      true,
			hasRatings:    false,
			hasWorkflow:   false,
			autoPublish:   true,
			hasSEO:        true,
		},
	}

	for _, ct := range contentTypes {
		id := uuid.New().String()
		_, err := tx.ExecContext(ctx, `
			INSERT INTO cms_content_types (
				id, tenant_id, name, slug, description, icon, color,
				is_hierarchical, has_categories, has_tags, has_comments,
				has_media, has_ratings, has_workflow, auto_publish,
				has_seo, status, created_at, updated_at, created_by
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, 'active', $17, $17, 'system')
		`, id, tenantID, ct.name, ct.slug, ct.description, ct.icon, ct.color,
			ct.hierarchical, ct.hasCategories, ct.hasTags, ct.hasComments,
			ct.hasMedia, ct.hasRatings, ct.hasWorkflow, ct.autoPublish,
			ct.hasSEO, now)

		if err != nil {
			return fmt.Errorf("failed to create content type %s: %w", ct.slug, err)
		}

		// Create default fields for blog posts
		if ct.slug == "blog-posts" {
			if err := i.createBlogPostFields(ctx, tx, id, tenantID); err != nil {
				return fmt.Errorf("failed to create blog post fields: %w", err)
			}
		}
	}

	return nil
}

// createBlogPostFields creates default fields for blog posts
func (i *Installer) createBlogPostFields(ctx context.Context, tx *sql.Tx, contentTypeId, tenantID string) error {
	now := time.Now()
	fields := []struct {
		name        string
		slug        string
		fieldType   string
		required    bool
		orderIndex  int
		searchable  bool
		filterable  bool
	}{
		{
			name:       "Featured Image",
			slug:       "featured_image",
			fieldType:  "image",
			required:   false,
			orderIndex: 1,
			searchable: false,
			filterable: false,
		},
		{
			name:       "Image Alt Text",
			slug:       "featured_image_alt",
			fieldType:  "text",
			required:   false,
			orderIndex: 2,
			searchable: true,
			filterable: false,
		},
		{
			name:       "Excerpt",
			slug:       "excerpt",
			fieldType:  "textarea",
			required:   false,
			orderIndex: 3,
			searchable: true,
			filterable: false,
		},
		{
			name:       "Reading Time",
			slug:       "reading_time",
			fieldType:  "number",
			required:   false,
			orderIndex: 4,
			searchable: false,
			filterable: true,
		},
	}

	for _, field := range fields {
		id := uuid.New().String()
		_, err := tx.ExecContext(ctx, `
			INSERT INTO cms_content_fields (
				id, content_type_id, name, slug, type, required,
				order_index, is_searchable, is_filterable,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		`, id, contentTypeId, field.name, field.slug, field.fieldType, field.required,
			field.orderIndex, field.searchable, field.filterable, now)

		if err != nil {
			return fmt.Errorf("failed to create field %s: %w", field.slug, err)
		}
	}

	return nil
}

// insertDefaultSettings inserts default CMS settings
func (i *Installer) insertDefaultSettings(ctx context.Context, tx *sql.Tx, tenantID string, req *InstallRequest) error {
	now := time.Now()
	systemID := uuid.New().String()

	// Convert settings to JSON
	features := map[string]interface{}{
		"comments":      req.EnableComments,
		"ratings":       req.EnableRatings,
		"search":        req.EnableSearch,
		"categories":    req.EnableCategories,
		"tags":          req.EnableTags,
		"media_library": req.EnableMedia,
		"seo":           req.EnableSEO,
	}

	theme := map[string]interface{}{
		"primary_color": req.PrimaryColor,
		"secondary_color": req.SecondaryColor,
		"font_family":   req.FontFamily,
		"header_style":  req.HeaderStyle,
	}

	settings := []struct {
		key         string
		value       interface{}
		description string
		type_       string
		category    string
	}{
		// Installation status
		{
			key:         "site_installed",
			value:       true,
			description: "Site installation status",
			type_:       "boolean",
			category:    "system",
		},
		{
			key:         "installed_at",
			value:       now.Format(time.RFC3339),
			description: "Site installation timestamp",
			type_:       "string",
			category:    "system",
		},
		{
			key:         "installed_version",
			value:       "1.0.0",
			description: "CMS version",
			type_:       "string",
			category:    "system",
		},

		// Site settings
		{
			key:         "site_title",
			value:       req.SiteTitle,
			description: "Website title",
			type_:       "string",
			category:    "general",
		},
		{
			key:         "site_description",
			value:       req.SiteDescription,
			description: "Website description",
			type_:       "string",
			category:    "general",
		},
		{
			key:         "site_url",
			value:       req.SiteURL,
			description: "Website URL",
			type_:       "string",
			category:    "general",
		},
		{
			key:         "admin_email",
			value:       req.AdminEmail,
			description: "Administrator email",
			type_:       "string",
			category:    "general",
		},
		{
			key:         "timezone",
			value:       req.Timezone,
			description: "Default timezone",
			type_:       "string",
			category:    "general",
		},
		{
			key:         "language",
			value:       req.Language,
			description: "Default language",
			type_:       "string",
			category:    "general",
		},

		// Features
		{
			key:         "features",
			value:       features,
			description: "Enabled CMS features",
			type_:       "json",
			category:    "features",
		},

		// Theme
		{
			key:         "theme",
			value:       theme,
			description: "Theme configuration",
			type_:       "json",
			category:    "theme",
		},

		// Blog settings
		{
			key:         "blog_posts_per_page",
			value:       10,
			description: "Number of blog posts per page",
			type_:       "number",
			category:    "blog",
		},
		{
			key:         "blog_excerpt_length",
			value:       200,
			description: "Blog post excerpt length",
			type_:       "number",
			category:    "blog",
		},
		{
			key:         "blog_show_author",
			value:       true,
			description: "Show author on blog posts",
			type_:       "boolean",
			category:    "blog",
		},

		// Comment settings
		{
			key:         "comments_require_approval",
			value:       false,
			description: "Require comment approval",
			type_:       "boolean",
			category:    "comments",
		},
		{
			key:         "comments_allow_guest",
			value:       true,
			description: "Allow guest comments",
			type_:       "boolean",
			category:    "comments",
		},
	}

	for _, setting := range settings {
		id := uuid.New().String()
		var valueJSON string

		switch v := setting.value.(type) {
		case string:
			valueJSON = fmt.Sprintf(`"%s"`, v)
		case bool:
			valueJSON = fmt.Sprintf(`%t`, v)
		case int:
			valueJSON = fmt.Sprintf(`%d`, v)
		default:
			jsonBytes, _ := json.Marshal(v)
			valueJSON = string(jsonBytes)
		}

		_, err := tx.ExecContext(ctx, `
			INSERT INTO cms_settings (
				id, tenant_id, key, value, description, type, category,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		`, id, tenantID, setting.key, valueJSON, setting.description, setting.type_, setting.category, now)

		if err != nil {
			return fmt.Errorf("failed to insert setting %s: %w", setting.key, err)
		}
	}

	return nil
}

// createDefaultCategories creates default categories
func (i *Installer) createDefaultCategories(ctx context.Context, tx *sql.Tx, tenantID string) error {
	now := time.Now()

	// Get blog posts content type ID
	var contentTypeId string
	err := tx.QueryRowContext(ctx,
		"SELECT id FROM cms_content_types WHERE tenant_id = $1 AND slug = 'blog-posts'",
		tenantID).Scan(&contentTypeId)

	if err != nil {
		// Content type might not exist, which is okay
		return nil
	}

	categories := []struct {
		name  string
		slug  string
		color string
	}{
		{"Technology", "technology", "#3b82f6"},
		{"Business", "business", "#10b981"},
		{"Design", "design", "#f59e0b"},
		{"Marketing", "marketing", "#ef4444"},
		{"Lifestyle", "lifestyle", "#8b5cf6"},
		{"Tutorial", "tutorial", "#06b6d4"},
	}

	for _, cat := range categories {
		id := uuid.New().String()
		_, err := tx.ExecContext(ctx, `
			INSERT INTO cms_categories (
				id, tenant_id, content_type_id, name, slug, color,
				status, created_at, updated_at, created_by
			) VALUES ($1, $2, $3, $4, $5, $6, 'active', $7, $7, 'system')
		`, id, tenantID, contentTypeId, cat.name, cat.slug, cat.color, now)

		if err != nil {
			return fmt.Errorf("failed to create category %s: %w", cat.slug, err)
		}
	}

	return nil
}

// createSampleContent creates sample content for demonstration
func (i *Installer) createSampleContent(ctx context.Context, tx *sql.Tx, tenantID string, req *InstallRequest) error {
	now := time.Now()

	// Get blog posts content type ID
	var contentTypeId string
	err := tx.QueryRowContext(ctx,
		"SELECT id FROM cms_content_types WHERE tenant_id = $1 AND slug = 'blog-posts'",
		tenantID).Scan(&contentTypeId)

	if err != nil {
		// Content type might not exist, which is okay
		return nil
	}

	// Get first category ID
	var categoryId string
	err = tx.QueryRowContext(ctx,
		"SELECT id FROM cms_categories WHERE tenant_id = $1 AND content_type_id = $2 LIMIT 1",
		tenantID, contentTypeId).Scan(&categoryId)

	if err != nil {
		// No categories found, which is okay
		categoryId = ""
	}

	// Create welcome post
	postID := uuid.New().String()
	publishedAt := now.Add(-1 * time.Hour) // Published 1 hour ago

	customFields := map[string]interface{}{
		"reading_time": 5,
	}

	customFieldsJSON, _ := json.Marshal(customFields)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO cms_content (
			id, tenant_id, content_type_id, title, slug, content, status,
			author_id, created_by, published_at, custom_fields,
			view_count, like_count, comment_count,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, 'published', 'system', 'system', $7, $8, 0, 0, 0, $9, $9)
	`, postID, tenantID, contentTypeId,
		"Welcome to Your New CMS Site",
		"welcome-to-your-new-cms-site",
		fmt.Sprintf(`
<h1>Welcome to Your New CMS Site!</h1>
<p>Congratulations! Your MetaBase CMS has been successfully installed and is ready to use. This powerful content management system provides everything you need to create and manage your website content.</p>

<h2>What's Included?</h2>
<ul>
<li><strong>Blog Posts</strong> - Share your thoughts and updates with the world</li>
<li><strong>Pages</strong> - Create static pages like About, Contact, and more</li>
<li><strong>Categories & Tags</strong> - Organize your content effectively</li>
<li><strong>Comments</strong> - Engage with your audience</li>
<li><strong>Media Library</strong> - Manage images and files</li>
<li><strong>SEO Features</strong> - Optimize your content for search engines</li>
</ul>

<h2>Next Steps</h2>
<ol>
<li>Customize your site settings in the admin panel</li>
<li>Create your first blog post</li>
<li>Set up your navigation menu</li>
<li>Choose a theme that matches your brand</li>
<li>Start creating amazing content!</li>
</ol>

<p>Thank you for choosing MetaBase CMS. If you need any help, check out our documentation or contact our support team.</p>
		`, req.SiteTitle),
		publishedAt,
		string(customFieldsJSON),
		now)

	if err != nil {
		return fmt.Errorf("failed to create welcome post: %w", err)
	}

	// Link post to category if category exists
	if categoryId != "" {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO cms_content_categories (content_id, category_id, created_at)
			VALUES ($1, $2, $3)
		`, postID, categoryId, now)

		if err != nil {
			return fmt.Errorf("failed to link post to category: %w", err)
		}
	}

	return nil
}

// GetInstallationStatus returns the current installation status
func (i *Installer) GetInstallationStatus(ctx context.Context, tenantID string) (map[string]interface{}, error) {
	installed, err := i.CheckInstallation(ctx)
	if err != nil {
		return nil, err
	}

	status := map[string]interface{}{
		"installed": installed,
	}

	if installed {
		// Get additional status information
		var siteTitle, siteDescription string
		var installedAt time.Time

		err := i.db.QueryRowContext(ctx, `
			SELECT
				COALESCE(
					(SELECT value::text FROM cms_settings WHERE tenant_id = $1 AND key = 'site_title'),
					'Untitled Site'
				) as site_title,
				COALESCE(
					(SELECT value::text FROM cms_settings WHERE tenant_id = $1 AND key = 'site_description'),
					'No description'
				) as site_description,
				COALESCE(
					(SELECT value::text FROM cms_settings WHERE tenant_id = $1 AND key = 'installed_at'),
					'1970-01-01T00:00:00Z'
				)::timestamp as installed_at
		`, tenantID).Scan(&siteTitle, &siteDescription, &installedAt)

		if err == nil {
			status["site_title"] = strings.Trim(siteTitle, `"`)
			status["site_description"] = strings.Trim(siteDescription, `"`)
			status["installed_at"] = installedAt.Format(time.RFC3339)
		}

		// Get content statistics
		var contentCount, categoryCount, tagCount int
		i.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM cms_content WHERE tenant_id = $1",
			tenantID).Scan(&contentCount)
		i.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM cms_categories WHERE tenant_id = $1",
			tenantID).Scan(&categoryCount)
		i.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM cms_tags WHERE tenant_id = $1",
			tenantID).Scan(&tagCount)

		status["statistics"] = map[string]int{
			"content_items": contentCount,
			"categories":    categoryCount,
			"tags":          tagCount,
		}
	}

	return status, nil
}