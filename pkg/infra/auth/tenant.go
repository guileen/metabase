package auth

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Tenant represents a multi-tenant organization
type Tenant struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Slug        string `json:"slug" yaml:"slug"`
	Domain      string `json:"domain,omitempty" yaml:"domain,omitempty"`
	Logo        string `json:"logo,omitempty" yaml:"logo,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Configuration
	Settings TenantSettings         `json:"settings" yaml:"settings"`
	Metadata map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Status
	IsActive bool   `json:"is_active" yaml:"is_active"`
	Plan     string `json:"plan" yaml:"plan"` // "free", "pro", "enterprise"

	// Limits
	Limits TenantLimits `json:"limits" yaml:"limits"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" yaml:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" yaml:"deleted_at,omitempty"`
}

// TenantSettings contains tenant-specific configuration
type TenantSettings struct {
	// User management
	AllowUserRegistration bool     `json:"allow_user_registration"`
	DefaultUserRole       string   `json:"default_user_role"`
	RequiredEmailDomains  []string `json:"required_email_domains,omitempty"`

	// Security
	RequireEmailVerification bool `json:"require_email_verification"`
	RequireTwoFactor         bool `json:"require_two_factor"`
	SessionTimeout           int  `json:"session_timeout_minutes"`

	// Features
	EnabledFeatures []string `json:"enabled_features,omitempty"`

	// UI Customization
	Theme     ThemeSettings `json:"theme,omitempty"`
	CustomCSS string        `json:"custom_css,omitempty"`
	CustomJS  string        `json:"custom_js,omitempty"`

	// Integration
	WebhookURL string            `json:"webhook_url,omitempty"`
	Webhooks   map[string]string `json:"webhooks,omitempty"`
}

// ThemeSettings defines UI theme customization
type ThemeSettings struct {
	PrimaryColor   string `json:"primary_color,omitempty"`
	SecondaryColor string `json:"secondary_color,omitempty"`
	LogoURL        string `json:"logo_url,omitempty"`
	FaviconURL     string `json:"favicon_url,omitempty"`
	CompanyName    string `json:"company_name,omitempty"`
}

// TenantLimits defines tenant limits and quotas
type TenantLimits struct {
	MaxUsers       int `json:"max_users"`
	MaxProjects    int `json:"max_projects"`
	MaxStorage     int `json:"max_storage_mb"`
	MaxAPIRequests int `json:"max_api_requests_per_day"`
}

// Project represents a project within a tenant
type Project struct {
	ID          string `json:"id" yaml:"id"`
	TenantID    string `json:"tenant_id" yaml:"tenant_id"`
	Name        string `json:"name" yaml:"name"`
	Slug        string `json:"slug" yaml:"slug"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Logo        string `json:"logo,omitempty" yaml:"logo,omitempty"`

	// Configuration
	Settings ProjectSettings        `json:"settings" yaml:"settings"`
	Metadata map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Status
	IsActive bool `json:"is_active" yaml:"is_active"`
	IsPublic bool `json:"is_public" yaml:"is_public"`

	// Environment
	Environment string `json:"environment" yaml:"environment"` // "development", "staging", "production"

	// Owner and team
	OwnerID string          `json:"owner_id" yaml:"owner_id"`
	Members []ProjectMember `json:"members,omitempty" yaml:"members,omitempty"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" yaml:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" yaml:"deleted_at,omitempty"`
}

// ProjectSettings contains project-specific configuration
type ProjectSettings struct {
	// Database settings
	DatabaseName string `json:"database_name,omitempty"`
	DatabaseType string `json:"database_type,omitempty"` // "sqlite", "postgres", "mysql"

	// Security
	RequireAuthForRead  bool     `json:"require_auth_for_read"`
	RequireAuthForWrite bool     `json:"require_auth_for_write"`
	AllowedOrigins      []string `json:"allowed_origins,omitempty"`

	// Features
	EnabledFeatures []string `json:"enabled_features,omitempty"`

	// Rate limiting
	RateLimit RateLimitSettings `json:"rate_limit,omitempty"`

	// Webhooks
	Webhooks map[string]string `json:"webhooks,omitempty"`
}

// RateLimitSettings defines rate limiting configuration
type RateLimitSettings struct {
	RequestsPerMinute int  `json:"requests_per_minute"`
	BurstSize         int  `json:"burst_size"`
	Enabled           bool `json:"enabled"`
}

// ProjectMember represents a user's role in a project
type ProjectMember struct {
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"` // "owner", "admin", "developer", "viewer"
	JoinedAt  time.Time `json:"joined_at"`
	InvitedBy string    `json:"invited_by,omitempty"`
}

// TenantProject represents a tenant's association with a project
type TenantProject struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	ProjectID string `json:"project_id"`
	Role      string `json:"role"` // "creator", "owner", "collaborator", "viewer"

	// Status and relationship info
	IsActive  bool       `json:"is_active"`
	IsCreator bool       `json:"is_creator"`           // If this tenant is the project creator
	InvitedBy string     `json:"invited_by,omitempty"` // Tenant ID that invited this tenant
	InvitedAt *time.Time `json:"invited_at,omitempty"`
	JoinedAt  time.Time  `json:"joined_at"`
	LeftAt    *time.Time `json:"left_at,omitempty"`

	// Collaboration info
	IsExternalCollaborator bool `json:"is_external_collaborator"` // If this tenant is from external collaboration
	CanInvite              bool `json:"can_invite"`               // Can this tenant invite other tenants
	CanManageMembers       bool `json:"can_manage_members"`       // Can this tenant manage project members

	// Contact person from this tenant
	ContactUserID string `json:"contact_user_id,omitempty"` // Main contact person for this tenant in the project

	// Permissions (custom permissions for this tenant in this project)
	CustomPermissions []string `json:"custom_permissions,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UserTenantProject represents a user's access through their tenant's project relationship
type UserTenantProject struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	TenantID  string `json:"tenant_id"`
	ProjectID string `json:"project_id"`
	Role      string `json:"effective_role"` // Effective role based on tenant's role

	// Status info
	IsActive bool       `json:"is_active"`
	JoinedAt time.Time  `json:"joined_at"`
	LeftAt   *time.Time `json:"left_at,omitempty"`

	// Tenant relationship info
	TenantRole string `json:"tenant_role"` // Role of tenant in project
	CanManage  bool   `json:"can_manage"`  // Whether user can manage based on tenant permissions

	// Permission flags
	IsCreator              bool   `json:"is_creator"`
	InvitedBy              string `json:"invited_by,omitempty"`
	IsExternalCollaborator bool   `json:"is_external_collaborator"`
	CanInvite              bool   `json:"can_invite"`
	CanManageMembers       bool   `json:"can_manage_members"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UserProject represents a user's project relationship (alias for UserTenantProject)
type UserProject = UserTenantProject

// TenantManager manages tenants, projects, and tenant-project associations
type TenantManager struct {
	tenants            map[string]*Tenant
	projects           map[string]*Project             // project_id -> project
	tenantProjects     map[string][]*TenantProject     // tenant_id -> tenant_projects
	userTenantProjects map[string][]*UserTenantProject // user_id -> user_tenant_projects

	mu sync.RWMutex
}

// NewTenantManager creates a new tenant manager
func NewTenantManager() *TenantManager {
	return &TenantManager{
		tenants:            make(map[string]*Tenant),
		projects:           make(map[string]*Project),
		tenantProjects:     make(map[string][]*TenantProject),
		userTenantProjects: make(map[string][]*UserTenantProject),
	}
}

// CreateTenant creates a new tenant
func (tm *TenantManager) CreateTenant(tenant *Tenant) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tenant.ID == "" {
		tenant.ID = generateUUID()
	}

	if tenant.CreatedAt.IsZero() {
		tenant.CreatedAt = time.Now()
		tenant.UpdatedAt = time.Now()
	}

	// Check if tenant slug already exists
	for _, existing := range tm.tenants {
		if existing.Slug == tenant.Slug {
			return fmt.Errorf("tenant with slug '%s' already exists", tenant.Slug)
		}
	}

	// Set default limits if not provided
	if tenant.Limits.MaxUsers == 0 {
		tenant.Limits.MaxUsers = 10
	}
	if tenant.Limits.MaxProjects == 0 {
		tenant.Limits.MaxProjects = 5
	}
	if tenant.Limits.MaxStorage == 0 {
		tenant.Limits.MaxStorage = 1024 // 1GB
	}
	if tenant.Limits.MaxAPIRequests == 0 {
		tenant.Limits.MaxAPIRequests = 10000
	}

	tm.tenants[tenant.ID] = tenant
	return nil
}

// GetTenant retrieves a tenant by ID
func (tm *TenantManager) GetTenant(tenantID string) (*Tenant, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tenant, exists := tm.tenants[tenantID]
	if !exists {
		return nil, fmt.Errorf("tenant %s not found", tenantID)
	}

	return tenant, nil
}

// GetTenantBySlug retrieves a tenant by slug
func (tm *TenantManager) GetTenantBySlug(slug string) (*Tenant, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	for _, tenant := range tm.tenants {
		if tenant.Slug == slug {
			return tenant, nil
		}
	}

	return nil, fmt.Errorf("tenant with slug '%s' not found", slug)
}

// UpdateTenant updates an existing tenant
func (tm *TenantManager) UpdateTenant(tenantID string, updates map[string]interface{}) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tenant, exists := tm.tenants[tenantID]
	if !exists {
		return fmt.Errorf("tenant %s not found", tenantID)
	}

	// Apply updates (simplified - in production, use reflection or a proper update mechanism)
	for key, value := range updates {
		switch key {
		case "name":
			if name, ok := value.(string); ok {
				tenant.Name = name
			}
		case "description":
			if desc, ok := value.(string); ok {
				tenant.Description = desc
			}
		case "is_active":
			if active, ok := value.(bool); ok {
				tenant.IsActive = active
			}
		case "plan":
			if plan, ok := value.(string); ok {
				tenant.Plan = plan
			}
		}
	}

	tenant.UpdatedAt = time.Now()
	return nil
}

// ListTenants returns all tenants
func (tm *TenantManager) ListTenants() ([]*Tenant, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tenants := make([]*Tenant, 0, len(tm.tenants))
	for _, tenant := range tm.tenants {
		tenants = append(tenants, tenant)
	}

	return tenants, nil
}

// DeleteTenant soft-deletes a tenant
func (tm *TenantManager) DeleteTenant(tenantID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tenant, exists := tm.tenants[tenantID]
	if !exists {
		return fmt.Errorf("tenant %s not found", tenantID)
	}

	now := time.Now()
	tenant.DeletedAt = &now
	tenant.IsActive = false

	return nil
}

// CreateProject creates a new project within a tenant
func (tm *TenantManager) CreateProject(project *Project) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Validate tenant exists
	_, exists := tm.tenants[project.TenantID]
	if !exists {
		return fmt.Errorf("tenant %s not found", project.TenantID)
	}

	if project.ID == "" {
		project.ID = generateUUID()
	}

	if project.CreatedAt.IsZero() {
		project.CreatedAt = time.Now()
		project.UpdatedAt = time.Now()
	}

	tm.projects[project.ID] = project
	return nil
}

// GetProject retrieves a project by ID
func (tm *TenantManager) GetProject(projectID string) (*Project, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	project, exists := tm.projects[projectID]
	if !exists {
		return nil, fmt.Errorf("project %s not found", projectID)
	}

	return project, nil
}

// ListProjects returns all projects for a tenant
func (tm *TenantManager) ListProjects(tenantID string) ([]*Project, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var projects []*Project
	for _, project := range tm.projects {
		if project.TenantID == tenantID {
			projects = append(projects, project)
		}
	}

	return projects, nil
}

// UpdateProject updates an existing project
func (tm *TenantManager) UpdateProject(projectID string, updates map[string]interface{}) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	project, exists := tm.projects[projectID]
	if !exists {
		return fmt.Errorf("project %s not found", projectID)
	}

	// Apply updates
	for key, value := range updates {
		switch key {
		case "name":
			if name, ok := value.(string); ok {
				project.Name = name
			}
		case "description":
			if desc, ok := value.(string); ok {
				project.Description = desc
			}
		case "is_active":
			if active, ok := value.(bool); ok {
				project.IsActive = active
			}
		case "environment":
			if env, ok := value.(string); ok {
				project.Environment = env
			}
		}
	}

	project.UpdatedAt = time.Now()
	return nil
}

// DeleteProject soft-deletes a project
func (tm *TenantManager) DeleteProject(projectID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	project, exists := tm.projects[projectID]
	if !exists {
		return fmt.Errorf("project %s not found", projectID)
	}

	now := time.Now()
	project.DeletedAt = &now
	project.IsActive = false

	return nil
}

// AddUserToProject adds a user to a project with a specific role
func (tm *TenantManager) AddUserToProject(userTenantID, projectID, role, invitedBy string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Validate project exists
	project, exists := tm.projects[projectID]
	if !exists {
		return fmt.Errorf("project %s not found", projectID)
	}

	// userTenantID is already the user's tenant ID (in real implementation, this would come from user context)

	// Check if this is external collaboration (user from different tenant)
	isExternalCollaborator := userTenantID != project.TenantID

	userProject := &UserTenantProject{
		ID:                     generateUUID(),
		UserID:                 userTenantID, // For now, using tenant ID as user ID
		TenantID:               userTenantID,
		ProjectID:              projectID,
		Role:                   role,
		IsActive:               true,
		IsCreator:              false, // Set to true only for actual project creation
		InvitedBy:              invitedBy,
		JoinedAt:               time.Now(),
		IsExternalCollaborator: isExternalCollaborator,
		CanInvite:              role == ProjectRoleOwner || role == ProjectRoleCollaborator,
		CanManageMembers:       role == ProjectRoleOwner,
	}

	if tm.userTenantProjects[userTenantID] == nil {
		tm.userTenantProjects[userTenantID] = make([]*UserTenantProject, 0)
	}

	tm.userTenantProjects[userTenantID] = append(tm.userTenantProjects[userTenantID], userProject)
	return nil
}

// CreateProjectWithOwner creates a project and sets the creator as owner
func (tm *TenantManager) CreateProjectWithOwner(project *Project, creatorID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Validate tenant exists
	_, exists := tm.tenants[project.TenantID]
	if !exists {
		return fmt.Errorf("tenant %s not found", project.TenantID)
	}

	if project.ID == "" {
		project.ID = generateUUID()
	}

	if project.CreatedAt.IsZero() {
		project.CreatedAt = time.Now()
		project.UpdatedAt = time.Now()
	}

	// Store project
	tm.projects[project.ID] = project

	// Add creator as project owner
	userProject := &UserTenantProject{
		ID:                     generateUUID(),
		UserID:                 creatorID,
		TenantID:               project.TenantID,
		ProjectID:              project.ID,
		Role:                   ProjectRoleCreator,
		IsActive:               true,
		IsCreator:              true,
		JoinedAt:               time.Now(),
		IsExternalCollaborator: false,
		CanInvite:              true,
		CanManageMembers:       true,
	}

	if tm.userTenantProjects[creatorID] == nil {
		tm.userTenantProjects[creatorID] = make([]*UserTenantProject, 0)
	}

	tm.userTenantProjects[creatorID] = append(tm.userTenantProjects[creatorID], userProject)
	return nil
}

// GetUserProjects returns all projects for a user
func (tm *TenantManager) GetUserProjects(userID string) ([]*UserTenantProject, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	userProjects, exists := tm.userTenantProjects[userID]
	if !exists {
		return []*UserTenantProject{}, nil
	}

	return userProjects, nil
}

// GetProjectMembers returns all members of a project
func (tm *TenantManager) GetProjectMembers(projectID string) ([]*UserTenantProject, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var members []*UserTenantProject
	for _, userProjects := range tm.userTenantProjects {
		for _, up := range userProjects {
			if up.ProjectID == projectID && up.IsActive {
				members = append(members, up)
			}
		}
	}

	return members, nil
}

// TransferProjectOwnership transfers project ownership to another user
func (tm *TenantManager) TransferProjectOwnership(projectID, fromUserID, toUserID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Find current owner's project association
	var fromUserProject *UserTenantProject
	for _, up := range tm.userTenantProjects[fromUserID] {
		if up.ProjectID == projectID && up.IsActive {
			fromUserProject = up
			break
		}
	}

	if fromUserProject == nil {
		return fmt.Errorf("user %s is not a member of project %s", fromUserID, projectID)
	}

	if fromUserProject.Role != ProjectRoleCreator && fromUserProject.Role != ProjectRoleOwner {
		return fmt.Errorf("user %s does not have permission to transfer ownership", fromUserID)
	}

	// Find or create target user's project association
	var toUserProject *UserTenantProject
	if tm.userTenantProjects[toUserID] == nil {
		tm.userTenantProjects[toUserID] = make([]*UserTenantProject, 0)
	}

	for _, up := range tm.userTenantProjects[toUserID] {
		if up.ProjectID == projectID && up.IsActive {
			toUserProject = up
			break
		}
	}

	if toUserProject == nil {
		// Add new user to project
		toUserProject = &UserProject{
			ID:                     generateUUID(),
			UserID:                 toUserID,
			ProjectID:              projectID,
			Role:                   ProjectRoleOwner,
			IsActive:               true,
			IsCreator:              false,
			JoinedAt:               time.Now(),
			IsExternalCollaborator: fromUserProject.TenantID != toUserProject.TenantID,
			CanInvite:              true,
			CanManageMembers:       true,
		}
		tm.userTenantProjects[toUserID] = append(tm.userTenantProjects[toUserID], toUserProject)
	} else {
		// Update existing user's role to owner
		toUserProject.Role = ProjectRoleOwner
		toUserProject.CanInvite = true
		toUserProject.CanManageMembers = true
	}

	// Update previous owner to collaborator
	if fromUserProject.Role == ProjectRoleCreator {
		fromUserProject.Role = ProjectRoleOwner // Creator can't be demoted fully
	} else {
		fromUserProject.Role = ProjectRoleCollaborator
	}

	return nil
}

// CheckUserProjectRole checks if a user has a specific role in a project
func (tm *TenantManager) CheckUserProjectRole(userID, projectID, requiredRole string) (bool, error) {
	userProjects, err := tm.GetUserProjects(userID)
	if err != nil {
		return false, err
	}

	for _, up := range userProjects {
		if up.ProjectID == projectID && up.IsActive {
			// Check role hierarchy
			if tm.hasRequiredRole(up.Role, requiredRole) {
				return true, nil
			}
		}
	}

	return false, nil
}

// hasRequiredRole checks if the user's role meets the required role level
func (tm *TenantManager) hasRequiredRole(userRole, requiredRole string) bool {
	roleHierarchy := map[string]int{
		ProjectRoleCreator:      4, // Highest level - project creator
		ProjectRoleOwner:        3, // Can manage project including ownership transfer
		ProjectRoleCollaborator: 2, // Full access but cannot manage ownership
		ProjectRoleViewer:       1, // Read-only access
	}

	userLevel, userExists := roleHierarchy[userRole]
	requiredLevel, requiredExists := roleHierarchy[requiredRole]

	if !userExists || !requiredExists {
		return false
	}

	return userLevel >= requiredLevel
}

// Constants
const (
	// Project roles - based on user relationship to project
	ProjectRoleCreator      = "creator"      // Project creator (highest privileges)
	ProjectRoleOwner        = "owner"        // Project owner (can transfer ownership)
	ProjectRoleCollaborator = "collaborator" // Full access collaborator
	ProjectRoleViewer       = "viewer"       // Read-only access
	ProjectRoleAdmin        = "admin"        // Project administrator

	// Tenant roles - based on user relationship to tenant
	TenantRoleAdmin  = "admin"  // Tenant administrator
	TenantRoleOwner  = "owner"  // Tenant owner
	TenantRoleMember = "member" // Regular tenant member

	// Tenant plans
	PlanFree       = "free"
	PlanPro        = "pro"
	PlanEnterprise = "enterprise"

	// Project environments
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"

	// System tenant and project
	SystemTenantID  = "system"
	SystemProjectID = "system"
)

// Helper functions

// generateUUID generates a new UUID
func generateUUID() string {
	return uuid.New().String()
}

// NewSystemTenant creates the system tenant
func NewSystemTenant() *Tenant {
	return &Tenant{
		ID:        SystemTenantID,
		Name:      "System",
		Slug:      "system",
		IsActive:  true,
		Plan:      PlanEnterprise,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Settings: TenantSettings{
			AllowUserRegistration: true,
			DefaultUserRole:       "user",
		},
		Limits: TenantLimits{
			MaxUsers:       999999,
			MaxProjects:    999999,
			MaxStorage:     999999999,
			MaxAPIRequests: 999999999,
		},
	}
}

// NewSystemProject creates the system project
func NewSystemProject() *Project {
	return &Project{
		ID:          SystemProjectID,
		TenantID:    SystemTenantID,
		Name:        "System",
		Slug:        "system",
		Description: "System administration and configuration",
		IsActive:    true,
		Environment: EnvProduction,
		OwnerID:     "system",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Settings: ProjectSettings{
			RequireAuthForRead:  true,
			RequireAuthForWrite: true,
		},
	}
}
