package auth

import ("context"
	"fmt"
	"sync"
	"time")

// Permission represents a single permission
type Permission struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Resource    string            `json:"resource"`
	Action      string            `json:"action"`
	Effect      string            `json:"effect"` // "allow" or "deny"
	Description string            `json:"description"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

// Role represents a role with permissions
type Role struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Permissions []string              `json:"permissions"` // Permission IDs
	TenantID    string                `json:"tenant_id"`
	IsSystem    bool                  `json:"is_system"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

// UserRole represents user role assignment
type UserRole struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	RoleID    string    `json:"role_id"`
	TenantID  string    `json:"tenant_id"`
	ProjectID string    `json:"project_id,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// RBACManager manages role-based access control
type RBACManager struct {
	permissions map[string]*Permission
	roles       map[string]*Role
	userRoles   map[string][]*UserRole // userID -> userRoles
	mu          sync.RWMutex
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager() *RBACManager {
	return &RBACManager{
		permissions: make(map[string]*Permission),
		roles:       make(map[string]*Role),
		userRoles:   make(map[string][]*UserRole),
	}
}

// RegisterPermission registers a new permission
func (r *RBACManager) RegisterPermission(permission *Permission) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if permission.ID == "" {
		return fmt.Errorf("permission ID cannot be empty")
	}

	// Validate effect
	if permission.Effect != "allow" && permission.Effect != "deny" {
		return fmt.Errorf("permission effect must be 'allow' or 'deny'")
	}

	r.permissions[permission.ID] = permission
	return nil
}

// RegisterRole registers a new role
func (r *RBACManager) RegisterRole(role *Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if role.ID == "" {
		return fmt.Errorf("role ID cannot be empty")
	}

	// Validate permissions exist
	for _, permID := range role.Permissions {
		if _, exists := r.permissions[permID]; !exists {
			return fmt.Errorf("permission %s does not exist", permID)
		}
	}

	r.roles[role.ID] = role
	return nil
}

// AssignRole assigns a role to a user
func (r *RBACManager) AssignRole(userRole *UserRole) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if userRole.ID == "" {
		userRole.ID = generateRoleID()
	}

	if userRole.CreatedAt.IsZero() {
		userRole.CreatedAt = time.Now()
	}

	// Validate role exists
	if _, exists := r.roles[userRole.RoleID]; !exists {
		return fmt.Errorf("role %s does not exist", userRole.RoleID)
	}

	// Initialize user roles slice if not exists
	if r.userRoles[userRole.UserID] == nil {
		r.userRoles[userRole.UserID] = make([]*UserRole, 0)
	}

	r.userRoles[userRole.UserID] = append(r.userRoles[userRole.UserID], userRole)
	return nil
}

// RevokeRole revokes a role from a user
func (r *RBACManager) RevokeRole(userID, roleID, tenantID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	userRoles, exists := r.userRoles[userID]
	if !exists {
		return fmt.Errorf("user %s has no roles", userID)
	}

	var updatedRoles []*UserRole
	found := false

	for _, userRole := range userRoles {
		if userRole.RoleID == roleID && userRole.TenantID == tenantID {
			found = true
			continue // Skip this role to revoke it
		}
		updatedRoles = append(updatedRoles, userRole)
	}

	if !found {
		return fmt.Errorf("role assignment not found")
	}

	if len(updatedRoles) == 0 {
		delete(r.userRoles, userID)
	} else {
		r.userRoles[userID] = updatedRoles
	}

	return nil
}

// GetUserRoles gets all roles assigned to a user
func (r *RBACManager) GetUserRoles(userID, tenantID string) ([]*Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userRoles, exists := r.userRoles[userID]
	if !exists {
		return []*Role{}, nil
	}

	var roles []*Role
	for _, userRole := range userRoles {
		// Check if role matches tenant
		if userRole.TenantID != tenantID {
			continue
		}

		// Check if role is expired
		if userRole.ExpiresAt != nil && userRole.ExpiresAt.Before(time.Now()) {
			continue
		}

		if role, exists := r.roles[userRole.RoleID]; exists {
			roles = append(roles, role)
		}
	}

	return roles, nil
}

// GetUserPermissions gets all permissions for a user
func (r *RBACManager) GetUserPermissions(userID, tenantID string) ([]*Permission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userRoles, exists := r.userRoles[userID]
	if !exists {
		return []*Permission{}, nil
	}

	permissionSet := make(map[string]*Permission)

	for _, userRole := range userRoles {
		// Check if role matches tenant
		if userRole.TenantID != tenantID {
			continue
		}

		// Check if role is expired
		if userRole.ExpiresAt != nil && userRole.ExpiresAt.Before(time.Now()) {
			continue
		}

		if role, exists := r.roles[userRole.RoleID]; exists {
			// Add all permissions from this role
			for _, permID := range role.Permissions {
				if perm, exists := r.permissions[permID]; exists {
					permissionSet[perm.ID] = perm
				}
			}
		}
	}

	// Convert map to slice
	var permissions []*Permission
	for _, perm := range permissionSet {
		permissions = append(permissions, perm)
	}

	return permissions, nil
}

// HasPermission checks if a user has a specific permission
func (r *RBACManager) HasPermission(userID, tenantID, resource, action string) (bool, error) {
	permissions, err := r.GetUserPermissions(userID, tenantID)
	if err != nil {
		return false, err
	}

	// Check for explicit deny first (deny takes precedence)
	for _, perm := range permissions {
		if perm.Resource == resource && perm.Action == action && perm.Effect == "deny" {
			return false, nil
		}
	}

	// Check for allow permission
	for _, perm := range permissions {
		if perm.Resource == resource && perm.Action == action && perm.Effect == "allow" {
			return true, nil
		}
	}

	// Check for wildcard permissions
	for _, perm := range permissions {
		if (perm.Resource == "*" || perm.Resource == resource) &&
			(perm.Action == "*" || perm.Action == action) &&
			perm.Effect == "allow" {
			return true, nil
		}
	}

	return false, nil
}

// HasAnyPermission checks if a user has any of the specified permissions
func (r *RBACManager) HasAnyPermission(userID, tenantID string, permissionChecks []PermissionCheck) (bool, error) {
	for _, check := range permissionChecks {
		hasPerm, err := r.HasPermission(userID, tenantID, check.Resource, check.Action)
		if err != nil {
			return false, err
		}
		if hasPerm {
			return true, nil
		}
	}
	return false, nil
}

// HasAllPermissions checks if a user has all of the specified permissions
func (r *RBACManager) HasAllPermissions(userID, tenantID string, permissionChecks []PermissionCheck) (bool, error) {
	for _, check := range permissionChecks {
		hasPerm, err := r.HasPermission(userID, tenantID, check.Resource, check.Action)
		if err != nil {
			return false, err
		}
		if !hasPerm {
			return false, nil
		}
	}
	return true, nil
}

// PermissionCheck represents a permission to check
type PermissionCheck struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// InitializeDefaults initializes default roles and permissions
func (r *RBACManager) InitializeDefaults() error {
	// Default permissions
	defaultPermissions := []*Permission{
		{
			ID:          "system.admin",
			Name:        "System Administrator",
			Resource:    "*",
			Action:      "*",
			Effect:      "allow",
			Description: "Full system access",
		},
		{
			ID:          "tenant.admin",
			Name:        "Tenant Administrator",
			Resource:    "tenant:*",
			Action:      "*",
			Effect:      "allow",
			Description: "Full tenant access",
		},
		{
			ID:          "project.admin",
			Name:        "Project Administrator",
			Resource:    "project:*",
			Action:      "*",
			Effect:      "allow",
			Description: "Full project access",
		},
		{
			ID:          "data.read",
			Name:        "Read Data",
			Resource:    "data",
			Action:      "read",
			Effect:      "allow",
			Description: "Read data access",
		},
		{
			ID:          "data.write",
			Name:        "Write Data",
			Resource:    "data",
			Action:      "write",
			Effect:      "allow",
			Description: "Write data access",
		},
		{
			ID:          "data.delete",
			Name:        "Delete Data",
			Resource:    "data",
			Action:      "delete",
			Effect:      "allow",
			Description: "Delete data access",
		},
		{
			ID:          "user.manage",
			Name:        "Manage Users",
			Resource:    "users",
			Action:      "*",
			Effect:      "allow",
			Description: "User management access",
		},
		{
			ID:          "config.manage",
			Name:        "Manage Configuration",
			Resource:    "config",
			Action:      "*",
			Effect:      "allow",
			Description: "Configuration management access",
		},
	}

	for _, perm := range defaultPermissions {
		if err := r.RegisterPermission(perm); err != nil {
			return fmt.Errorf("failed to register permission %s: %w", perm.ID, err)
		}
	}

	// Default roles
	defaultRoles := []*Role{
		{
			ID:          "system.admin",
			Name:        "System Administrator",
			Description: "Full system access",
			Permissions: []string{"system.admin"},
			IsSystem:    true,
			TenantID:    "system",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "tenant.admin",
			Name:        "Tenant Administrator",
			Description: "Full tenant access",
			Permissions: []string{"tenant.admin", "data.read", "data.write", "data.delete", "user.manage"},
			IsSystem:    true,
			TenantID:    "system",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "project.admin",
			Name:        "Project Administrator",
			Description: "Full project access",
			Permissions: []string{"project.admin", "data.read", "data.write", "data.delete"},
			IsSystem:    true,
			TenantID:    "system",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "user.readonly",
			Name:        "Read Only User",
			Description: "Read-only access to data",
			Permissions: []string{"data.read"},
			IsSystem:    true,
			TenantID:    "system",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "user.standard",
			Name:        "Standard User",
			Description: "Standard data access",
			Permissions: []string{"data.read", "data.write"},
			IsSystem:    true,
			TenantID:    "system",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, role := range defaultRoles {
		if err := r.RegisterRole(role); err != nil {
			return fmt.Errorf("failed to register role %s: %w", role.ID, err)
		}
	}

	return nil
}

// GetPermission gets a permission by ID
func (r *RBACManager) GetPermission(id string) (*Permission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	perm, exists := r.permissions[id]
	if !exists {
		return nil, fmt.Errorf("permission %s not found", id)
	}

	return perm, nil
}

// GetRole gets a role by ID
func (r *RBACManager) GetRole(id string) (*Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	role, exists := r.roles[id]
	if !exists {
		return nil, fmt.Errorf("role %s not found", id)
	}

	return role, nil
}

// ListPermissions lists all permissions
func (r *RBACManager) ListPermissions() ([]*Permission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var permissions []*Permission
	for _, perm := range r.permissions {
		permissions = append(permissions, perm)
	}

	return permissions, nil
}

// ListRoles lists all roles
func (r *RBACManager) ListRoles(tenantID string) ([]*Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var roles []*Role
	for _, role := range r.roles {
		if tenantID == "" || role.TenantID == tenantID {
			roles = append(roles, role)
		}
	}

	return roles, nil
}

// generateRoleID generates a unique role assignment ID
func generateRoleID() string {
	return fmt.Sprintf("role_%d", time.Now().UnixNano())
}

// Policy represents an access control policy
type Policy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Effect      string                 `json:"effect"`
	Principal   string                 `json:"principal"` // user, role, or "*"
	Action      []string               `json:"action"`
	Resource    []string               `json:"resource"`
	Condition   map[string]interface{} `json:"condition,omitempty"`
	TenantID    string                 `json:"tenant_id"`
	Priority    int                    `json:"priority"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// PolicyEngine evaluates policies
type PolicyEngine struct {
	policies map[string]*Policy
	rbac     *RBACManager
	mu       sync.RWMutex
}

// NewPolicyEngine creates a new policy engine
func NewPolicyEngine(rbac *RBACManager) *PolicyEngine {
	return &PolicyEngine{
		policies: make(map[string]*Policy),
		rbac:     rbac,
	}
}

// AddPolicy adds a new policy
func (pe *PolicyEngine) AddPolicy(policy *Policy) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if policy.ID == "" {
		return fmt.Errorf("policy ID cannot be empty")
	}

	pe.policies[policy.ID] = policy
	return nil
}

// Evaluate evaluates policies for a user
func (pe *PolicyEngine) Evaluate(ctx context.Context, userID, tenantID, resource, action string) (bool, error) {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	// First check RBAC permissions
	hasPermission, err := pe.rbac.HasPermission(userID, tenantID, resource, action)
	if err != nil {
		return false, err
	}

	if hasPermission {
		return true, nil
	}

	// Evaluate policies (sorted by priority)
	var policies []*Policy
	for _, policy := range pe.policies {
		if policy.TenantID != tenantID && policy.TenantID != "system" {
			continue
		}
		policies = append(policies, policy)
	}

	// Sort by priority (higher priority first)
	for i := 0; i < len(policies)-1; i++ {
		for j := i + 1; j < len(policies); j++ {
			if policies[i].Priority < policies[j].Priority {
				policies[i], policies[j] = policies[j], policies[i]
			}
		}
	}

	// Evaluate policies
	for _, policy := range policies {
		if pe.matchesPolicy(policy, userID, resource, action) {
			return policy.Effect == "allow", nil
		}
	}

	// Default deny
	return false, nil
}

// matchesPolicy checks if a request matches a policy
func (pe *PolicyEngine) matchesPolicy(policy *Policy, userID, resource, action string) bool {
	// Check principal match
	if policy.Principal != "*" && policy.Principal != userID {
		return false
	}

	// Check action match
	if !pe.matchesSlice(policy.Action, action) {
		return false
	}

	// Check resource match
	if !pe.matchesSlice(policy.Resource, resource) {
		return false
	}

	// Check conditions (basic implementation)
	return true
}

// matchesSlice checks if a value matches any pattern in a slice
func (pe *PolicyEngine) matchesSlice(s []string, value string) bool {
	for _, pattern := range s {
		if pattern == "*" || pattern == value {
			return true
		}
	}
	return false
}