package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/guileen/metabase/pkg/infra/auth"
)

// ProjectContext represents project information in the request context
type ProjectContext struct {
	ProjectID              string
	UserID                 string
	UserRole               string
	TenantID               string
	IsCreator              bool
	IsOwner                bool
	IsExternalCollaborator bool
	IsSystem               bool
	CanInvite              bool
	CanManageMembers       bool
}

// ProjectMiddleware handles project authorization and collaboration
type ProjectMiddleware struct {
	db            interface{} // *sql.DB placeholder
	rbacManager   *auth.RBACManager
	tenantManager *auth.TenantManager
	logger        *zap.Logger
}

// NewProjectMiddleware creates a new project middleware
func NewProjectMiddleware(db interface{}, rbacManager *auth.RBACManager, tenantManager *auth.TenantManager, logger *zap.Logger) *ProjectMiddleware {
	return &ProjectMiddleware{
		db:            db,
		rbacManager:   rbacManager,
		tenantManager: tenantManager,
		logger:        logger,
	}
}

// SystemAdminMiddleware ensures the user is a system administrator
func (pm *ProjectMiddleware) SystemAdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := pm.extractUserID(r)
		if userID == "" {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Check if user is system admin
		isAdmin, err := pm.checkSystemAdmin(userID)
		if err != nil {
			pm.logger.Error("Failed to check system admin status", zap.String("user_id", userID), zap.Error(err))
			http.Error(w, "Failed to verify permissions", http.StatusInternalServerError)
			return
		}

		if !isAdmin {
			http.Error(w, "Access denied: system administrator required", http.StatusForbidden)
			return
		}

		// Add admin context
		ctx := context.WithValue(r.Context(), "is_system_admin", true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TenantAccessMiddleware ensures the user has access to tenant operations (system admin only)
func (pm *ProjectMiddleware) TenantAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := pm.extractUserID(r)
		if userID == "" {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Only system admins can access tenant management
		isAdmin, err := pm.checkSystemAdmin(userID)
		if err != nil {
			pm.logger.Error("Failed to check system admin status", zap.String("user_id", userID), zap.Error(err))
			http.Error(w, "Failed to verify permissions", http.StatusInternalServerError)
			return
		}

		if !isAdmin {
			http.Error(w, "Access denied: system administrator required", http.StatusForbidden)
			return
		}

		// Add context
		ctx := context.WithValue(r.Context(), "user_id", userID)
		ctx = context.WithValue(ctx, "is_system_admin", true)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ProjectAccessMiddleware ensures the user has access to the project
func (pm *ProjectMiddleware) ProjectAccessMiddleware(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := pm.extractUserID(r)
			if userID == "" {
				http.Error(w, "User not authenticated", http.StatusUnauthorized)
				return
			}

			projectID := chi.URLParam(r, "projectId")
			if projectID == "" {
				http.Error(w, "Project not found in request", http.StatusBadRequest)
				return
			}

			// Check user access
			userProject, err := pm.getUserProjectRole(userID, projectID)
			if err != nil {
				pm.logger.Error("Failed to check project access", zap.String("user_id", userID), zap.String("project_id", projectID), zap.Error(err))
				http.Error(w, "Failed to verify project access", http.StatusInternalServerError)
				return
			}

			// Check if user meets required role
			if !pm.meetsRoleRequirement(userProject.Role, requiredRole) {
				// Also check if user is system admin
				isSystemAdmin, _ := pm.checkSystemAdmin(userID)
				if !isSystemAdmin {
					http.Error(w, "Access denied: insufficient project permissions", http.StatusForbidden)
					return
				}
				// Set system admin as highest role
				userProject.Role = auth.RoleSuperAdmin
			}

			// Get project tenant ID
			project, err := pm.tenantManager.GetProject(projectID)
			if err != nil {
				http.Error(w, "Project not found", http.StatusNotFound)
				return
			}

			// Add project context
			ctx := context.WithValue(r.Context(), "user_id", userID)
			ctx = context.WithValue(ctx, "tenant_id", project.TenantID)
			ctx = context.WithValue(ctx, "project_id", projectID)
			ctx = context.WithValue(ctx, "user_role", userProject.Role)
			ctx = context.WithValue(ctx, "is_creator", userProject.IsCreator)
			ctx = context.WithValue(ctx, "is_owner", userProject.Role == auth.ProjectRoleOwner || userProject.Role == auth.ProjectRoleCreator)
			ctx = context.WithValue(ctx, "is_external_collaborator", userProject.IsExternalCollaborator)
			ctx = context.WithValue(ctx, "can_invite", userProject.CanInvite)
			ctx = context.WithValue(ctx, "can_manage_members", userProject.CanManageMembers)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ProjectOwnerMiddleware ensures the user is project owner or creator
func (pm *ProjectMiddleware) ProjectOwnerMiddleware(next http.Handler) http.Handler {
	return pm.ProjectAccessMiddleware(auth.ProjectRoleOwner)(next)
}

// ProjectCollaboratorMiddleware ensures the user has at least collaborator access
func (pm *ProjectMiddleware) ProjectCollaboratorMiddleware(next http.Handler) http.Handler {
	return pm.ProjectAccessMiddleware(auth.ProjectRoleCollaborator)(next)
}

// ProjectViewerMiddleware ensures the user has at least viewer access
func (pm *ProjectMiddleware) ProjectViewerMiddleware(next http.Handler) http.Handler {
	return pm.ProjectAccessMiddleware(auth.ProjectRoleViewer)(next)
}

// CanManageProjectMiddleware ensures the user can manage project (owner or collaborator with management permissions)
func (pm *ProjectMiddleware) CanManageProjectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := pm.extractUserID(r)
		if userID == "" {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		projectID := chi.URLParam(r, "projectId")
		if projectID == "" {
			http.Error(w, "Project not found in request", http.StatusBadRequest)
			return
		}

		// Check user access
		userProject, err := pm.getUserProjectRole(userID, projectID)
		if err != nil {
			pm.logger.Error("Failed to check project access", zap.String("user_id", userID), zap.String("project_id", projectID), zap.Error(err))
			http.Error(w, "Failed to verify project access", http.StatusInternalServerError)
			return
		}

		// Check if user can manage project
		canManage := userProject.Role == auth.ProjectRoleCreator ||
			userProject.Role == auth.ProjectRoleOwner ||
			userProject.CanManageMembers

		if !canManage {
			// Also check if user is system admin
			isSystemAdmin, _ := pm.checkSystemAdmin(userID)
			if !isSystemAdmin {
				http.Error(w, "Access denied: insufficient project management permissions", http.StatusForbidden)
				return
			}
		}

		// Add project context
		ctx := context.WithValue(r.Context(), "can_manage_project", true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetProjectContext extracts project information from the request context
func GetProjectContext(r *http.Request) *ProjectContext {
	return &ProjectContext{
		UserID:                 getStringFromContext(r.Context(), "user_id"),
		TenantID:               getStringFromContext(r.Context(), "tenant_id"),
		ProjectID:              getStringFromContext(r.Context(), "project_id"),
		UserRole:               getStringFromContext(r.Context(), "user_role"),
		IsCreator:              getBoolFromContext(r.Context(), "is_creator"),
		IsOwner:                getBoolFromContext(r.Context(), "is_owner"),
		IsExternalCollaborator: getBoolFromContext(r.Context(), "is_external_collaborator"),
		IsSystem:               getBoolFromContext(r.Context(), "is_system_admin"),
		CanInvite:              getBoolFromContext(r.Context(), "can_invite"),
		CanManageMembers:       getBoolFromContext(r.Context(), "can_manage_members"),
	}
}

// Helper methods

func (pm *ProjectMiddleware) extractUserID(r *http.Request) string {
	// Try to get from Authorization header (JWT)
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		// TODO: Parse JWT and extract user ID
		// For now, return mock user ID
		return "system_admin"
	}

	// Try to get from context (set by other middleware)
	if userID := r.Context().Value("user_id"); userID != nil {
		return userID.(string)
	}

	// Try to get from API key context
	if apiKey := r.Context().Value("apiKey"); apiKey != nil {
		// TODO: Extract user ID from API key
		// For now, return mock user ID
		return "system_admin"
	}

	return ""
}

func (pm *ProjectMiddleware) checkSystemAdmin(userID string) (bool, error) {
	// TODO: Check against user database or JWT claims
	// For now, assume user with specific ID is system admin
	return userID == "system_admin" || userID == "admin", nil
}

func (pm *ProjectMiddleware) getUserProjectRole(userID, projectID string) (*auth.UserProject, error) {
	// System admin gets highest privileges
	if isAdmin, _ := pm.checkSystemAdmin(userID); isAdmin {
		return &auth.UserProject{
			UserID:           userID,
			ProjectID:        projectID,
			Role:             auth.RoleSuperAdmin,
			IsActive:         true,
			CanInvite:        true,
			CanManageMembers: true,
		}, nil
	}

	// Check user project role
	userProjects, err := pm.tenantManager.GetUserProjects(userID)
	if err != nil {
		return nil, err
	}

	for _, up := range userProjects {
		if up.ProjectID == projectID && up.IsActive {
			return (*auth.UserProject)(up), nil
		}
	}

	return nil, fmt.Errorf("user %s is not a member of project %s", userID, projectID)
}

func (pm *ProjectMiddleware) meetsRoleRequirement(userRole, requiredRole string) bool {
	roleHierarchy := map[string]int{
		auth.RoleSuperAdmin:          5,
		auth.ProjectRoleCreator:      4, // Highest level - project creator
		auth.ProjectRoleOwner:        3, // Can manage project including ownership transfer
		auth.ProjectRoleCollaborator: 2, // Full access but cannot manage ownership
		auth.ProjectRoleViewer:       1, // Read-only access
	}

	userLevel, userExists := roleHierarchy[userRole]
	requiredLevel, requiredExists := roleHierarchy[requiredRole]

	if !userExists || !requiredExists {
		return false
	}

	return userLevel >= requiredLevel
}

func getStringFromContext(ctx context.Context, key string) string {
	if value := ctx.Value(key); value != nil {
		return value.(string)
	}
	return ""
}

func getBoolFromContext(ctx context.Context, key string) bool {
	if value := ctx.Value(key); value != nil {
		return value.(bool)
	}
	return false
}
