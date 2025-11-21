package tenant

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/guileen/metabase/pkg/common/errors"
	"github.com/guileen/metabase/pkg/infra/auth"
)

// Tenant represents a tenant organization
type Tenant struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Domain    string                 `json:"domain"`
	Plan      string                 `json:"plan"`
	Status    TenantStatus           `json:"status"`
	Settings  *TenantSettings        `json:"settings"`
	Limits    *TenantLimits          `json:"limits"`
	Usage     *TenantUsage           `json:"usage"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	CreatedBy string                 `json:"created_by"`
	UpdatedBy string                 `json:"updated_by"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// TenantStatus represents tenant status
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusInactive  TenantStatus = "inactive"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusDeleted   TenantStatus = "deleted"
)

// TenantSettings represents tenant-specific settings
type TenantSettings struct {
	Theme         *ThemeSettings         `json:"theme,omitempty"`
	Email         *EmailSettings         `json:"email,omitempty"`
	Security      *SecuritySettings      `json:"security,omitempty"`
	Storage       *StorageSettings       `json:"storage,omitempty"`
	API           *APISettings           `json:"api,omitempty"`
	Notifications *NotificationSettings  `json:"notifications,omitempty"`
	Integration   *IntegrationSettings   `json:"integration,omitempty"`
	Features      map[string]bool        `json:"features,omitempty"`
	Custom        map[string]interface{} `json:"custom,omitempty"`
}

// ThemeSettings represents theme configuration
type ThemeSettings struct {
	LogoURL      string `json:"logo_url,omitempty"`
	PrimaryColor string `json:"primary_color,omitempty"`
	DarkMode     bool   `json:"dark_mode,omitempty"`
	CustomCSS    string `json:"custom_css,omitempty"`
}

// EmailSettings represents email configuration
type EmailSettings struct {
	FromName  string            `json:"from_name,omitempty"`
	FromEmail string            `json:"from_email,omitempty"`
	SMTPHost  string            `json:"smtp_host,omitempty"`
	SMTPPort  int               `json:"smtp_port,omitempty"`
	UseTLS    bool              `json:"use_tls,omitempty"`
	Templates map[string]string `json:"templates,omitempty"`
}

// SecuritySettings represents security configuration
type SecuritySettings struct {
	SessionTimeout    time.Duration   `json:"session_timeout,omitempty"`
	PasswordPolicy    *PasswordPolicy `json:"password_policy,omitempty"`
	TwoFactorAuth     bool            `json:"two_factor_auth,omitempty"`
	AllowedIPs        []string        `json:"allowed_ips,omitempty"`
	PasswordMinLength int             `json:"password_min_length,omitempty"`
}

// PasswordPolicy represents password policy
type PasswordPolicy struct {
	MinLength        int  `json:"min_length"`
	RequireUppercase bool `json:"require_uppercase"`
	RequireLowercase bool `json:"require_lowercase"`
	RequireNumbers   bool `json:"require_numbers"`
	RequireSymbols   bool `json:"require_symbols"`
	MaxAge           int  `json:"max_age_days,omitempty"`
	HistoryCount     int  `json:"history_count,omitempty"`
}

// StorageSettings represents storage configuration
type StorageSettings struct {
	MaxFileSize   int64    `json:"max_file_size,omitempty"`
	MaxStorage    int64    `json:"max_storage,omitempty"`
	AllowedTypes  []string `json:"allowed_types,omitempty"`
	AutoDelete    bool     `json:"auto_delete,omitempty"`
	RetentionDays int      `json:"retention_days,omitempty"`
}

// APISettings represents API configuration
type APISettings struct {
	RateLimit *RateLimitConfig `json:"rate_limit,omitempty"`
	CORS      *CORSConfig      `json:"cors,omitempty"`
	Webhooks  []string         `json:"webhooks,omitempty"`
	APIKeys   bool             `json:"api_keys_enabled,omitempty"`
}

// RateLimitConfig represents rate limit configuration
type RateLimitConfig struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	BurstSize         int `json:"burst_size"`
}

// CORSConfig represents CORS configuration
type CORSConfig struct {
	AllowedOrigins []string `json:"allowed_origins,omitempty"`
	AllowedMethods []string `json:"allowed_methods,omitempty"`
	AllowedHeaders []string `json:"allowed_headers,omitempty"`
	MaxAge         int      `json:"max_age,omitempty"`
}

// NotificationSettings represents notification configuration
type NotificationSettings struct {
	Email    bool     `json:"email_notifications,omitempty"`
	SMS      bool     `json:"sms_notifications,omitempty"`
	Slack    string   `json:"slack_webhook,omitempty"`
	Discord  string   `json:"discord_webhook,omitempty"`
	Channels []string `json:"channels,omitempty"`
}

// IntegrationSettings represents third-party integrations
type IntegrationSettings struct {
	Google *GoogleIntegration     `json:"google,omitempty"`
	GitHub *GitHubIntegration     `json:"github,omitempty"`
	Slack  *SlackIntegration      `json:"slack,omitempty"`
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// GoogleIntegration represents Google integration
type GoogleIntegration struct {
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	Enabled      bool   `json:"enabled,omitempty"`
}

// GitHubIntegration represents GitHub integration
type GitHubIntegration struct {
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	Enabled      bool   `json:"enabled,omitempty"`
}

// SlackIntegration represents Slack integration
type SlackIntegration struct {
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	Enabled      bool   `json:"enabled,omitempty"`
}

// TenantLimits represents tenant resource limits
type TenantLimits struct {
	MaxUsers        int   `json:"max_users,omitempty"`
	MaxProjects     int   `json:"max_projects,omitempty"`
	MaxAPIKeys      int   `json:"max_api_keys,omitempty"`
	MaxStorageMB    int64 `json:"max_storage_mb,omitempty"`
	MaxBandwidthGB  int64 `json:"max_bandwidth_gb,omitempty"`
	RequestsPerHour int   `json:"requests_per_hour,omitempty"`
}

// TenantUsage represents current tenant usage
type TenantUsage struct {
	Users       int64     `json:"users,omitempty"`
	Projects    int64     `json:"projects,omitempty"`
	APIKeys     int64     `json:"api_keys,omitempty"`
	StorageMB   int64     `json:"storage_mb,omitempty"`
	BandwidthGB int64     `json:"bandwidth_gb,omitempty"`
	Requests    int64     `json:"requests,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Project represents a tenant project
type Project struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	TenantID    string                 `json:"tenant_id"`
	OwnerID     string                 `json:"owner_id"`
	Status      ProjectStatus          `json:"status"`
	Settings    *ProjectSettings       `json:"settings"`
	Members     []*ProjectMember       `json:"members"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CreatedBy   string                 `json:"created_by"`
	UpdatedBy   string                 `json:"updated_by"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ProjectStatus represents project status
type ProjectStatus string

const (
	ProjectStatusActive   ProjectStatus = "active"
	ProjectStatusInactive ProjectStatus = "inactive"
	ProjectStatusArchived ProjectStatus = "archived"
	ProjectStatusDeleted  ProjectStatus = "deleted"
)

// ProjectSettings represents project-specific settings
type ProjectSettings struct {
	Database    *DatabaseSettings      `json:"database,omitempty"`
	API         *ProjectAPISettings    `json:"api,omitempty"`
	Environment map[string]string      `json:"environment,omitempty"`
	Custom      map[string]interface{} `json:"custom,omitempty"`
}

// DatabaseSettings represents database settings
type DatabaseSettings struct {
	AutoBackup bool          `json:"auto_backup,omitempty"`
	Retention  time.Duration `json:"retention,omitempty"`
	Region     string        `json:"region,omitempty"`
	Tables     []string      `json:"tables,omitempty"`
}

// ProjectAPISettings represents project API settings
type ProjectAPISettings struct {
	BaseURL     string              `json:"base_url,omitempty"`
	Enabled     bool                `json:"enabled,omitempty"`
	RateLimit   *RateLimitConfig    `json:"rate_limit,omitempty"`
	Permissions map[string][]string `json:"permissions,omitempty"`
}

// ProjectMember represents a project member
type ProjectMember struct {
	UserID     string     `json:"user_id"`
	Role       MemberRole `json:"role"`
	JoinedAt   time.Time  `json:"joined_at"`
	InvitedBy  string     `json:"invited_by,omitempty"`
	LastActive time.Time  `json:"last_active,omitempty"`
}

// MemberRole represents member role
type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleEditor MemberRole = "editor"
	MemberRoleViewer MemberRole = "viewer"
)

// TenantManager manages tenants and projects
type TenantManager struct {
	db     *sql.DB
	rbac   *auth.RBACManager
	cache  *TenantCache
	config *TenantConfig
}

// TenantConfig represents tenant manager configuration
type TenantConfig struct {
	DefaultPlan             string                 `json:"default_plan"`
	DefaultLimits           *TenantLimits          `json:"default_limits"`
	PlanConfigurations      map[string]*PlanConfig `json:"plan_configurations"`
	EnableMultiTenancy      bool                   `json:"enable_multi_tenancy"`
	EnableProjectManagement bool                   `json:"enable_project_management"`
	RequireDomainUnique     bool                   `json:"require_domain_unique"`
	DefaultSettings         *TenantSettings        `json:"default_settings"`
}

// PlanConfig represents plan-specific configuration
type PlanConfig struct {
	Name         string          `json:"name"`
	Price        float64         `json:"price,omitempty"`
	BillingCycle string          `json:"billing_cycle,omitempty"`
	Limits       *TenantLimits   `json:"limits"`
	Features     []string        `json:"features"`
	Settings     *TenantSettings `json:"settings,omitempty"`
}

// TenantCache provides caching for tenant data
type TenantCache struct {
	tenants  map[string]*Tenant
	projects map[string]*Project
	mu       sync.RWMutex
	ttl      time.Duration
}

// NewTenantManager creates a new tenant manager
func NewTenantManager(db *sql.DB, rbac *auth.RBACManager, config *TenantConfig) *TenantManager {
	if config == nil {
		config = &TenantConfig{
			DefaultPlan:             "free",
			EnableMultiTenancy:      true,
			EnableProjectManagement: true,
			RequireDomainUnique:     false,
		}
	}

	return &TenantManager{
		db:     db,
		rbac:   rbac,
		cache:  NewTenantCache(10 * time.Minute),
		config: config,
	}
}

// NewTenantCache creates a new tenant cache
func NewTenantCache(ttl time.Duration) *TenantCache {
	return &TenantCache{
		tenants:  make(map[string]*Tenant),
		projects: make(map[string]*Project),
		ttl:      ttl,
	}
}

// CreateTenant creates a new tenant
func (tm *TenantManager) CreateTenant(ctx context.Context, tenant *Tenant) error {
	// Validate tenant
	if err := tm.validateTenant(tenant); err != nil {
		return fmt.Errorf("invalid tenant: %w", err)
	}

	// Check if tenant already exists
	if exists, err := tm.tenantExists(ctx, tenant.Name, tenant.Domain); err != nil {
		return fmt.Errorf("failed to check tenant existence: %w", err)
	} else if exists {
		return errors.InvalidInput("Tenant with this name or domain already exists")
	}

	// Set defaults
	if tenant.ID == "" {
		tenant.ID = generateTenantID()
	}
	if tenant.Status == "" {
		tenant.Status = TenantStatusActive
	}
	if tenant.Plan == "" {
		tenant.Plan = tm.config.DefaultPlan
	}
	if tenant.CreatedAt.IsZero() {
		tenant.CreatedAt = time.Now()
	}
	if tenant.UpdatedAt.IsZero() {
		tenant.UpdatedAt = time.Now()
	}

	// Apply plan defaults
	if err := tm.applyPlanDefaults(tenant); err != nil {
		return fmt.Errorf("failed to apply plan defaults: %w", err)
	}

	// Insert into database
	query := `INSERT INTO tenants
		(id, name, domain, plan, status, settings, limits, usage, created_at, updated_at, created_by, updated_by, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	settingsJSON, _ := json.Marshal(tenant.Settings)
	limitsJSON, _ := json.Marshal(tenant.Limits)
	usageJSON, _ := json.Marshal(tenant.Usage)
	metadataJSON, _ := json.Marshal(tenant.Metadata)

	_, err := tm.db.ExecContext(ctx, query,
		tenant.ID, tenant.Name, tenant.Domain, tenant.Plan, tenant.Status,
		string(settingsJSON), string(limitsJSON), string(usageJSON),
		tenant.CreatedAt, tenant.UpdatedAt,
		tenant.CreatedBy, tenant.UpdatedBy, string(metadataJSON))

	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	// Update cache
	tm.cache.SetTenant(tenant)

	return nil
}

// GetTenant retrieves a tenant by ID
func (tm *TenantManager) GetTenant(ctx context.Context, tenantID string) (*Tenant, error) {
	// Check cache first
	if tenant := tm.cache.GetTenant(tenantID); tenant != nil {
		return tenant, nil
	}

	query := `SELECT id, name, domain, plan, status, settings, limits, usage,
			  created_at, updated_at, created_by, updated_by, metadata
			  FROM tenants WHERE id = ? AND status != 'deleted'`

	var tenant Tenant
	var settingsJSON, limitsJSON, usageJSON, metadataJSON sql.NullString

	err := tm.db.QueryRowContext(ctx, query, tenantID).Scan(
		&tenant.ID, &tenant.Name, &tenant.Domain, &tenant.Plan, &tenant.Status,
		&settingsJSON, &limitsJSON, &usageJSON,
		&tenant.CreatedAt, &tenant.UpdatedAt,
		&tenant.CreatedBy, &tenant.UpdatedBy, &metadataJSON)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NotFound("tenant")
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Parse JSON fields
	if settingsJSON.Valid {
		_ = json.Unmarshal([]byte(settingsJSON.String), &tenant.Settings)
	}
	if limitsJSON.Valid {
		_ = json.Unmarshal([]byte(limitsJSON.String), &tenant.Limits)
	}
	if usageJSON.Valid {
		_ = json.Unmarshal([]byte(usageJSON.String), &tenant.Usage)
	}
	if metadataJSON.Valid {
		_ = json.Unmarshal([]byte(metadataJSON.String), &tenant.Metadata)
	}

	// Update cache
	tm.cache.SetTenant(&tenant)

	return &tenant, nil
}

// UpdateTenant updates a tenant
func (tm *TenantManager) UpdateTenant(ctx context.Context, tenant *Tenant) error {
	// Validate tenant
	if err := tm.validateTenant(tenant); err != nil {
		return fmt.Errorf("invalid tenant: %w", err)
	}

	tenant.UpdatedAt = time.Now()

	query := `UPDATE tenants SET
		name = ?, domain = ?, plan = ?, status = ?, settings = ?, limits = ?,
		updated_at = ?, updated_by = ?, metadata = ?
		WHERE id = ?`

	settingsJSON, _ := json.Marshal(tenant.Settings)
	limitsJSON, _ := json.Marshal(tenant.Limits)
	metadataJSON, _ := json.Marshal(tenant.Metadata)

	_, err := tm.db.ExecContext(ctx, query,
		tenant.Name, tenant.Domain, tenant.Plan, tenant.Status,
		string(settingsJSON), string(limitsJSON),
		tenant.UpdatedAt, tenant.UpdatedBy, string(metadataJSON),
		tenant.ID)

	if err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	// Update cache
	tm.cache.SetTenant(tenant)

	return nil
}

// DeleteTenant soft deletes a tenant
func (tm *TenantManager) DeleteTenant(ctx context.Context, tenantID, userID string) error {
	query := `UPDATE tenants SET status = 'deleted', updated_at = ?, updated_by = ? WHERE id = ?`

	_, err := tm.db.ExecContext(ctx, query, time.Now(), userID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	// Remove from cache
	tm.cache.DeleteTenant(tenantID)

	return nil
}

// CreateProject creates a new project
func (tm *TenantManager) CreateProject(ctx context.Context, project *Project) error {
	// Validate project
	if err := tm.validateProject(project); err != nil {
		return fmt.Errorf("invalid project: %w", err)
	}

	// Check if project name is unique within tenant
	if exists, err := tm.projectExists(ctx, project.TenantID, project.Name); err != nil {
		return fmt.Errorf("failed to check project existence: %w", err)
	} else if exists {
		return errors.InvalidInput("Project with this name already exists")
	}

	// Set defaults
	if project.ID == "" {
		project.ID = generateProjectID()
	}
	if project.Status == "" {
		project.Status = ProjectStatusActive
	}
	if project.CreatedAt.IsZero() {
		project.CreatedAt = time.Now()
	}
	if project.UpdatedAt.IsZero() {
		project.UpdatedAt = time.Now()
	}

	// Add owner as member
	if project.Members == nil {
		project.Members = make([]*ProjectMember, 0)
	}
	project.Members = append(project.Members, &ProjectMember{
		UserID:   project.OwnerID,
		Role:     MemberRoleOwner,
		JoinedAt: time.Now(),
	})

	query := `INSERT INTO projects
		(id, name, description, tenant_id, owner_id, status, settings, members,
		 created_at, updated_at, created_by, updated_by, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	settingsJSON, _ := json.Marshal(project.Settings)
	membersJSON, _ := json.Marshal(project.Members)
	metadataJSON, _ := json.Marshal(project.Metadata)

	_, err := tm.db.ExecContext(ctx, query,
		project.ID, project.Name, project.Description, project.TenantID, project.OwnerID,
		project.Status, string(settingsJSON), string(membersJSON),
		project.CreatedAt, project.UpdatedAt,
		project.CreatedBy, project.UpdatedBy, string(metadataJSON))

	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	// Update cache
	tm.cache.SetProject(project)

	return nil
}

// GetProject retrieves a project by ID
func (tm *TenantManager) GetProject(ctx context.Context, projectID string) (*Project, error) {
	// Check cache first
	if project := tm.cache.GetProject(projectID); project != nil {
		return project, nil
	}

	query := `SELECT id, name, description, tenant_id, owner_id, status, settings, members,
			  created_at, updated_at, created_by, updated_by, metadata
			  FROM projects WHERE id = ? AND status != 'deleted'`

	var project Project
	var settingsJSON, membersJSON, metadataJSON sql.NullString

	err := tm.db.QueryRowContext(ctx, query, projectID).Scan(
		&project.ID, &project.Name, &project.Description, &project.TenantID, &project.OwnerID,
		&project.Status, &settingsJSON, &membersJSON,
		&project.CreatedAt, &project.UpdatedAt,
		&project.CreatedBy, &project.UpdatedBy, &metadataJSON)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NotFound("project")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Parse JSON fields
	if settingsJSON.Valid {
		_ = json.Unmarshal([]byte(settingsJSON.String), &project.Settings)
	}
	if membersJSON.Valid {
		_ = json.Unmarshal([]byte(membersJSON.String), &project.Members)
	}
	if metadataJSON.Valid {
		_ = json.Unmarshal([]byte(metadataJSON.String), &project.Metadata)
	}

	// Update cache
	tm.cache.SetProject(&project)

	return &project, nil
}

// ListTenantProjects lists all projects for a tenant
func (tm *TenantManager) ListTenantProjects(ctx context.Context, tenantID string) ([]*Project, error) {
	query := `SELECT id, name, description, tenant_id, owner_id, status, settings, members,
			  created_at, updated_at, created_by, updated_by, metadata
			  FROM projects WHERE tenant_id = ? AND status != 'deleted'
			  ORDER BY created_at DESC`

	rows, err := tm.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		var project Project
		var settingsJSON, membersJSON, metadataJSON sql.NullString

		err := rows.Scan(
			&project.ID, &project.Name, &project.Description, &project.TenantID, &project.OwnerID,
			&project.Status, &settingsJSON, &membersJSON,
			&project.CreatedAt, &project.UpdatedAt,
			&project.CreatedBy, &project.UpdatedBy, &metadataJSON)

		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}

		// Parse JSON fields
		if settingsJSON.Valid {
			_ = json.Unmarshal([]byte(settingsJSON.String), &project.Settings)
		}
		if membersJSON.Valid {
			_ = json.Unmarshal([]byte(membersJSON.String), &project.Members)
		}
		if metadataJSON.Valid {
			_ = json.Unmarshal([]byte(metadataJSON.String), &project.Metadata)
		}

		projects = append(projects, &project)
	}

	return projects, nil
}

// UpdateProject updates a project
func (tm *TenantManager) UpdateProject(ctx context.Context, project *Project) error {
	// Validate project
	if err := tm.validateProject(project); err != nil {
		return fmt.Errorf("invalid project: %w", err)
	}

	project.UpdatedAt = time.Now()

	query := `UPDATE projects SET
		name = ?, description = ?, status = ?, settings = ?, members = ?,
		updated_at = ?, updated_by = ?, metadata = ?
		WHERE id = ?`

	settingsJSON, _ := json.Marshal(project.Settings)
	membersJSON, _ := json.Marshal(project.Members)
	metadataJSON, _ := json.Marshal(project.Metadata)

	_, err := tm.db.ExecContext(ctx, query,
		project.Name, project.Description, project.Status,
		string(settingsJSON), string(membersJSON),
		project.UpdatedAt, project.UpdatedBy, string(metadataJSON),
		project.ID)

	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	// Update cache
	tm.cache.SetProject(project)

	return nil
}

// AddProjectMember adds a member to a project
func (tm *TenantManager) AddProjectMember(ctx context.Context, projectID, userID string, role MemberRole) error {
	project, err := tm.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	// Check if user is already a member
	for _, member := range project.Members {
		if member.UserID == userID {
			return errors.InvalidInput("User is already a project member")
		}
	}

	// Add new member
	member := &ProjectMember{
		UserID:   userID,
		Role:     role,
		JoinedAt: time.Now(),
	}

	project.Members = append(project.Members, member)
	project.UpdatedAt = time.Now()

	return tm.UpdateProject(ctx, project)
}

// RemoveProjectMember removes a member from a project
func (tm *TenantManager) RemoveProjectMember(ctx context.Context, projectID, userID string) error {
	project, err := tm.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	// Find and remove member
	var updatedMembers []*ProjectMember
	found := false

	for _, member := range project.Members {
		if member.UserID == userID {
			found = true
			continue
		}
		updatedMembers = append(updatedMembers, member)
	}

	if !found {
		return errors.NotFound("Project member")
	}

	// Don't allow removing the owner
	if project.OwnerID == userID {
		return errors.InvalidInput("Cannot remove project owner")
	}

	project.Members = updatedMembers
	project.UpdatedAt = time.Now()

	return tm.UpdateProject(ctx, project)
}

// CheckProjectAccess checks if a user has access to a project
func (tm *TenantManager) CheckProjectAccess(ctx context.Context, projectID, userID string) (*ProjectMember, bool) {
	project, err := tm.GetProject(ctx, projectID)
	if err != nil {
		return nil, false
	}

	for _, member := range project.Members {
		if member.UserID == userID {
			return member, true
		}
	}

	return nil, false
}

// Validate helper methods
func (tm *TenantManager) validateTenant(tenant *Tenant) error {
	if tenant.Name == "" {
		return errors.InvalidInput("Tenant name is required")
	}
	if len(tenant.Name) > 100 {
		return errors.InvalidInput("Tenant name too long (max 100 characters)")
	}
	if tenant.Domain != "" && len(tenant.Domain) > 255 {
		return errors.InvalidInput("Domain too long (max 255 characters)")
	}
	return nil
}

func (tm *TenantManager) validateProject(project *Project) error {
	if project.Name == "" {
		return errors.InvalidInput("Project name is required")
	}
	if len(project.Name) > 100 {
		return errors.InvalidInput("Project name too long (max 100 characters)")
	}
	if project.TenantID == "" {
		return errors.InvalidInput("Tenant ID is required")
	}
	if project.OwnerID == "" {
		return errors.InvalidInput("Owner ID is required")
	}
	return nil
}

func (tm *TenantManager) tenantExists(ctx context.Context, name, domain string) (bool, error) {
	query := `SELECT COUNT(*) FROM tenants WHERE (name = ? OR domain = ?) AND status != 'deleted'`
	var count int
	err := tm.db.QueryRowContext(ctx, query, name, domain).Scan(&count)
	return count > 0, err
}

func (tm *TenantManager) projectExists(ctx context.Context, tenantID, name string) (bool, error) {
	query := `SELECT COUNT(*) FROM projects WHERE tenant_id = ? AND name = ? AND status != 'deleted'`
	var count int
	err := tm.db.QueryRowContext(ctx, query, tenantID, name).Scan(&count)
	return count > 0, err
}

func (tm *TenantManager) applyPlanDefaults(tenant *Tenant) error {
	planConfig, exists := tm.config.PlanConfigurations[tenant.Plan]
	if !exists {
		planConfig = &PlanConfig{
			Name:   tenant.Plan,
			Limits: tm.config.DefaultLimits,
		}
	}

	if tenant.Limits == nil {
		tenant.Limits = planConfig.Limits
	}

	if tenant.Settings == nil {
		tenant.Settings = tm.config.DefaultSettings
	}

	return nil
}

// Cache methods
func (tc *TenantCache) GetTenant(tenantID string) *Tenant {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.tenants[tenantID]
}

func (tc *TenantCache) SetTenant(tenant *Tenant) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.tenants[tenant.ID] = tenant
}

func (tc *TenantCache) DeleteTenant(tenantID string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	delete(tc.tenants, tenantID)
}

func (tc *TenantCache) GetProject(projectID string) *Project {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.projects[projectID]
}

func (tc *TenantCache) SetProject(project *Project) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.projects[project.ID] = project
}

func (tc *TenantCache) DeleteProject(projectID string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	delete(tc.projects, projectID)
}

// ID generation functions
func generateTenantID() string {
	return fmt.Sprintf("tenant_%d", time.Now().UnixNano())
}

func generateProjectID() string {
	return fmt.Sprintf("proj_%d", time.Now().UnixNano())
}
