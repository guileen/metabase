package installer

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
	"github.com/guileen/metabase/pkg/interfaces"
	"go.uber.org/zap"
)

// CMSInstaller handles CMS installation and initialization
type CMSInstaller struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewCMSInstaller creates a new CMS installer
func NewCMSInstaller(db *sql.DB, logger *zap.Logger) interfaces.ProjectInstaller {
	return &CMSInstaller{
		db:     db,
		logger: logger,
	}
}

// CMSInstallRequest represents the CMS installation request
type CMSInstallRequest struct {
	// Basic site information
	SiteTitle       string `json:"site_title" validate:"required"`
	SiteDescription string `json:"site_description"`
	SiteURL         string `json:"site_url" validate:"required,url"`
	AdminEmail      string `json:"admin_email" validate:"required,email"`
	AdminPassword   string `json:"admin_password" validate:"required,min=8"`
	Timezone        string `json:"timezone"`
	Language        string `json:"language"`

	// CMS Features
	EnableComments   bool `json:"enable_comments"`
	EnableRatings    bool `json:"enable_ratings"`
	EnableSearch     bool `json:"enable_search"`
	EnableCategories bool `json:"enable_categories"`
	EnableTags       bool `json:"enable_tags"`
	EnableMedia      bool `json:"enable_media"`
	EnableSEO        bool `json:"enable_seo"`

	// Theme settings
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	FontFamily     string `json:"font_family"`
	HeaderStyle    string `json:"header_style"`
}

// CMSInstallResponse represents the CMS installation response
type CMSInstallResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	SiteURL     string `json:"site_url"`
	AdminURL    string `json:"admin_url"`
	InstalledAt string `json:"installed_at"`
	Version     string `json:"version"`
}

// Name returns the project name
func (i *CMSInstaller) Name() string {
	return "CMS"
}

// Version returns the project version
func (i *CMSInstaller) Version() string {
	return "1.0.0"
}

// Type returns the project type
func (i *CMSInstaller) Type() string {
	return interfaces.ProjectTypeCMS
}

// CheckInstallation checks if CMS is already installed
func (i *CMSInstaller) CheckInstallation(ctx context.Context, tenantID string) (bool, error) {
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
func (i *CMSInstaller) Install(ctx context.Context, req *interfaces.InstallRequest) (*interfaces.InstallResult, error) {
	// Create install request from config
	cmsReq := &CMSInstallRequest{
		SiteTitle:        "Default CMS Site",
		SiteDescription:  "A CMS site created by MetaBase",
		SiteURL:          "https://localhost:8080",
		AdminEmail:       "admin@example.com",
		AdminPassword:    "changeme123",
		EnableComments:   true,
		EnableTags:       true,
		EnableSearch:     true,
		EnableCategories: true,
		EnableMedia:      true,
		EnableSEO:        true,
	}

	if req.Config != nil {
		if siteTitle, ok := req.Config["site_title"].(string); ok {
			cmsReq.SiteTitle = siteTitle
		}
		if siteDescription, ok := req.Config["site_description"].(string); ok {
			cmsReq.SiteDescription = siteDescription
		}
		if siteURL, ok := req.Config["site_url"].(string); ok {
			cmsReq.SiteURL = siteURL
		}
		if adminEmail, ok := req.Config["admin_email"].(string); ok {
			cmsReq.AdminEmail = adminEmail
		}
		if adminPassword, ok := req.Config["admin_password"].(string); ok {
			cmsReq.AdminPassword = adminPassword
		}
		if enableComments, ok := req.Config["enable_comments"].(bool); ok {
			cmsReq.EnableComments = enableComments
		}
		if enableTags, ok := req.Config["enable_tags"].(bool); ok {
			cmsReq.EnableTags = enableTags
		}
		if enableSearch, ok := req.Config["enable_search"].(bool); ok {
			cmsReq.EnableSearch = enableSearch
		}
		if enableCategories, ok := req.Config["enable_categories"].(bool); ok {
			cmsReq.EnableCategories = enableCategories
		}
		if enableMedia, ok := req.Config["enable_media"].(bool); ok {
			cmsReq.EnableMedia = enableMedia
		}
		if enableSEO, ok := req.Config["enable_seo"].(bool); ok {
			cmsReq.EnableSEO = enableSEO
		}
	}

	// Call the existing install method
	response, err := i.installWithRequest(ctx, cmsReq, req.TenantID)
	if err != nil {
		return &interfaces.InstallResult{
			Success: false,
			Message: err.Error(),
		}, err
	}

	// Convert response to InstallResult
	projectID := fmt.Sprintf("%s-%s", req.ProjectType, req.TenantID)
	return &interfaces.InstallResult{
		Success:     response.Success,
		Message:     response.Message,
		ProjectID:   projectID,
		ProjectType: req.ProjectType,
		Version:     i.Version(),
		InstalledAt: time.Now(),
		Endpoint:    "/api/v1/cms",
		AdminURL:    "/admin/cms",
		Config:      req.Config,
		Metadata: map[string]interface{}{
			"site_title": cmsReq.SiteTitle,
			"site_url":   cmsReq.SiteURL,
		},
	}, nil
}

// installWithRequest performs the CMS installation with specific request
func (i *CMSInstaller) installWithRequest(ctx context.Context, req *CMSInstallRequest, tenantID string) (*CMSInstallResponse, error) {
	i.logger.Info("Starting CMS installation",
		zap.String("tenant_id", tenantID),
		zap.String("site_title", req.SiteTitle))

	// Validate request
	if err := i.validateInstallRequest(req); err != nil {
		return &CMSInstallResponse{
			Success: false,
			Message: fmt.Sprintf("Validation failed: %v", err),
		}, nil
	}

	// Check if already installed
	installed, err := i.CheckInstallation(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check installation status: %w", err)
	}

	if installed {
		return &CMSInstallResponse{
			Success: false,
			Message: "CMS is already installed",
		}, nil
	}

	// Begin transaction
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			i.logger.Warn("Failed to rollback transaction", zap.Error(err))
		}
	}()

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

	return &CMSInstallResponse{
		Success:     true,
		Message:     "CMS installed successfully",
		SiteURL:     req.SiteURL,
		AdminURL:    fmt.Sprintf("%s/admin", req.SiteURL),
		InstalledAt: time.Now().Format(time.RFC3339),
		Version:     "1.0.0",
	}, nil
}

// validateInstallRequest validates the installation request
func (i *CMSInstaller) validateInstallRequest(req *CMSInstallRequest) error {
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
func (i *CMSInstaller) runMigrations(ctx context.Context, tx *sql.Tx) error {
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
func (i *CMSInstaller) createDefaultContentTypes(ctx context.Context, tx *sql.Tx, tenantID string) error {
	now := time.Now()
	contentTypes := []struct {
		name          string
		slug          string
		description   string
		icon          string
		color         string
		hierarchical  bool
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
			name:          "Blog Posts",
			slug:          "blog-posts",
			description:   "Blog posts and articles",
			icon:          "article",
			color:         "#3b82f6",
			hierarchical:  false,
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
			name:          "Pages",
			slug:          "pages",
			description:   "Static pages like About, Contact, etc.",
			icon:          "file-text",
			color:         "#10b981",
			hierarchical:  true,
			hasCategories: false,
			hasTags:       false,
			hasComments:   false,
			hasMedia:      false,
			hasRatings:    false,
			hasWorkflow:   true,
			autoPublish:   false,
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
	}

	return nil
}

// insertDefaultSettings inserts default CMS settings
func (i *CMSInstaller) insertDefaultSettings(ctx context.Context, tx *sql.Tx, tenantID string, req *CMSInstallRequest) error {
	now := time.Now()

	// Installation status
	settings := []struct {
		key         string
		value       interface{}
		description string
		type_       string
		category    string
	}{
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
	}

	for _, setting := range settings {
		id := uuid.New().String()
		var valueJSON string

		switch v := setting.value.(type) {
		case string:
			valueJSON = fmt.Sprintf(`"%s"`, v)
		case bool:
			valueJSON = fmt.Sprintf(`%t`, v)
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
func (i *CMSInstaller) createDefaultCategories(ctx context.Context, tx *sql.Tx, tenantID string) error {
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
func (i *CMSInstaller) createSampleContent(ctx context.Context, tx *sql.Tx, tenantID string, req *CMSInstallRequest) error {
	// Get blog posts content type ID
	var contentTypeId string
	err := tx.QueryRowContext(ctx,
		"SELECT id FROM cms_content_types WHERE tenant_id = $1 AND slug = 'blog-posts'",
		tenantID).Scan(&contentTypeId)

	if err != nil {
		// Content type might not exist, which is okay
		return nil
	}

	// Create welcome post
	postID := uuid.New().String()
	now := time.Now()
	publishedAt := now.Add(-1 * time.Hour) // Published 1 hour ago

	_, err = tx.ExecContext(ctx, `
		INSERT INTO cms_content (
			id, tenant_id, content_type_id, title, slug, content, status,
			author_id, created_by, published_at,
			view_count, like_count, comment_count,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, 'published', 'system', 'system', $7, 0, 0, 0, $8, $8)
	`, postID, tenantID, contentTypeId,
		"Welcome to Your New CMS Site",
		"welcome-to-your-new-cms-site",
		fmt.Sprintf(`
<h1>Welcome to %s!</h1>
<p>Congratulations! Your MetaBase CMS has been successfully installed and is ready to use.</p>
		`, req.SiteTitle),
		publishedAt, now)

	if err != nil {
		return fmt.Errorf("failed to create welcome post: %w", err)
	}

	return nil
}

// GetInstallationStatus returns the current installation status
func (i *CMSInstaller) GetInstallationStatus(ctx context.Context, tenantID string) (map[string]interface{}, error) {
	installed, err := i.CheckInstallation(ctx, tenantID)
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
	}

	return status, nil
}

// GetConfigurationSchema returns the configuration schema
func (i *CMSInstaller) GetConfigurationSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"enable_comments": map[string]interface{}{
				"type":        "boolean",
				"default":     true,
				"description": "Enable comments on content",
			},
			"enable_tags": map[string]interface{}{
				"type":        "boolean",
				"default":     true,
				"description": "Enable tagging system",
			},
			"max_file_size": map[string]interface{}{
				"type":        "integer",
				"default":     10485760, // 10MB
				"description": "Maximum file size in bytes",
			},
		},
	}
}

// ValidateConfiguration validates the provided configuration
func (i *CMSInstaller) ValidateConfiguration(config map[string]interface{}) error {
	if config == nil {
		return nil
	}

	// Validate max file size
	if maxSize, ok := config["max_file_size"]; ok {
		if maxSizeInt, ok := maxSize.(int); ok {
			if maxSizeInt < 1024 || maxSizeInt > 104857600 { // 1KB to 100MB
				return fmt.Errorf("max_file_size must be between 1024 and 104857600 bytes")
			}
		} else {
			return fmt.Errorf("max_file_size must be an integer")
		}
	}

	return nil
}

// CheckDependencies checks if required dependencies are met
func (i *CMSInstaller) CheckDependencies(ctx context.Context, db *sql.DB) error {
	// Check if database is accessible
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	return nil
}

// GetDependencies returns the list of project dependencies
func (i *CMSInstaller) GetDependencies() []string {
	return []string{interfaces.ProjectTypeAuthGateway}
}

// Uninstall removes the CMS from the specified tenant
func (i *CMSInstaller) Uninstall(ctx context.Context, tenantID string) error {
	// For safety, we don't actually delete data on uninstall
	i.logger.Info("CMS uninstalled", zap.String("tenant_id", tenantID))
	return nil
}
