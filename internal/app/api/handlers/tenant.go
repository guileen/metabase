package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/guileen/metabase/internal/app/api/middleware"
	"github.com/guileen/metabase/pkg/infra/auth"
)

// TenantHandler handles tenant and project management requests
type TenantHandler struct {
	db            *sql.DB
	tenantManager *auth.TenantManager
	logger        *zap.Logger
}

// NewTenantHandler creates a new tenant handler
func NewTenantHandler(db *sql.DB, logger *zap.Logger) *TenantHandler {
	return &TenantHandler{
		db:            db,
		tenantManager: auth.NewTenantManager(),
		logger:        logger,
	}
}

// TenantRequest represents tenant creation/update request
type TenantRequest struct {
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug"`
	Domain      string                 `json:"domain,omitempty"`
	Logo        string                 `json:"logo,omitempty"`
	Description string                 `json:"description,omitempty"`
	Settings    auth.TenantSettings    `json:"settings,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Plan        string                 `json:"plan,omitempty"`
}

// ProjectRequest represents project creation/update request
type TenantProjectRequest struct {
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug"`
	Description string                 `json:"description,omitempty"`
	Logo        string                 `json:"logo,omitempty"`
	Settings    auth.ProjectSettings   `json:"settings,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	IsPublic    bool                   `json:"is_public,omitempty"`
	Environment string                 `json:"environment,omitempty"`
}

// UserTenantRequest represents user-tenant assignment request
type UserTenantRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// ProjectRequest represents user-project assignment request
type ProjectRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// InviteUserRequest represents user invitation request
type InviteUserRequest struct {
	UserID  string `json:"user_id"`
	Email   string `json:"email,omitempty"` // Alternative: invite by email
	Role    string `json:"role"`
	Message string `json:"message,omitempty"`
}

// TransferOwnershipRequest represents ownership transfer request
type TransferOwnershipRequest struct {
	ToUserID string `json:"to_user_id"`
	Message  string `json:"message,omitempty"`
}

// ListTenants handles tenant listing requests
func (h *TenantHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if user is system admin
	if !h.isSystemAdmin(ctx, r) {
		h.writeError(w, http.StatusForbidden, "Access denied: system admin required")
		return
	}

	// Get query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Query database
	rows, err := h.db.QueryContext(ctx, `
		SELECT id, name, slug, domain, logo, description, settings, metadata,
			   is_active, plan, limits, created_at, updated_at, deleted_at
		FROM tenants
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		h.logger.Error("Failed to query tenants", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to query tenants")
		return
	}
	defer rows.Close()

	var tenants []auth.Tenant
	for rows.Next() {
		var tenant auth.Tenant
		var settingsJSON, metadataJSON, limitsJSON sql.NullString
		var deletedAt sql.NullTime

		err := rows.Scan(
			&tenant.ID,
			&tenant.Name,
			&tenant.Slug,
			&tenant.Domain,
			&tenant.Logo,
			&tenant.Description,
			&settingsJSON,
			&metadataJSON,
			&tenant.IsActive,
			&tenant.Plan,
			&limitsJSON,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
			&deletedAt,
		)
		if err != nil {
			h.logger.Error("Failed to scan tenant row", zap.Error(err))
			continue
		}

		if deletedAt.Valid {
			tenant.DeletedAt = &deletedAt.Time
		}

		// Parse JSON fields
		if settingsJSON.Valid {
			json.Unmarshal([]byte(settingsJSON.String), &tenant.Settings)
		}
		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &tenant.Metadata)
		}
		if limitsJSON.Valid {
			json.Unmarshal([]byte(limitsJSON.String), &tenant.Limits)
		}

		tenants = append(tenants, tenant)
	}

	// Get total count
	var total int
	h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tenants WHERE deleted_at IS NULL").Scan(&total)

	response := map[string]interface{}{
		"tenants": tenants,
		"total":   total,
		"page":    page,
		"limit":   limit,
	}

	h.writeJSON(w, response)
}

// CreateTenant handles tenant creation requests
func (h *TenantHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if user is system admin
	if !h.isSystemAdmin(ctx, r) {
		h.writeError(w, http.StatusForbidden, "Access denied: system admin required")
		return
	}

	var req TenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate request
	if req.Name == "" {
		h.writeError(w, http.StatusBadRequest, "Name is required")
		return
	}
	if req.Slug == "" {
		h.writeError(w, http.StatusBadRequest, "Slug is required")
		return
	}

	// Create tenant
	tenant := &auth.Tenant{
		Name:        req.Name,
		Slug:        req.Slug,
		Domain:      req.Domain,
		Logo:        req.Logo,
		Description: req.Description,
		Settings:    req.Settings,
		Metadata:    req.Metadata,
		IsActive:    true,
		Plan:        req.Plan,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Set default plan if not provided
	if tenant.Plan == "" {
		tenant.Plan = auth.PlanFree
	}

	// Set default settings
	if tenant.Settings.SessionTimeout == 0 {
		tenant.Settings.SessionTimeout = 1440 // 24 hours
	}

	// Set default limits
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

	// Serialize JSON fields
	settingsJSON, _ := json.Marshal(tenant.Settings)
	metadataJSON, _ := json.Marshal(tenant.Metadata)
	limitsJSON, _ := json.Marshal(tenant.Limits)

	// Insert into database
	query := `
		INSERT INTO tenants (id, name, slug, domain, logo, description, settings, metadata,
							is_active, plan, limits, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := h.db.ExecContext(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.Slug,
		tenant.Domain,
		tenant.Logo,
		tenant.Description,
		string(settingsJSON),
		string(metadataJSON),
		tenant.IsActive,
		tenant.Plan,
		string(limitsJSON),
		tenant.CreatedAt,
		tenant.UpdatedAt,
	)
	if err != nil {
		h.logger.Error("Failed to create tenant", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to create tenant")
		return
	}

	id, _ := result.LastInsertId()
	if id != 0 {
		// This shouldn't happen with UUID but handle it
		tenant.ID = strconv.FormatInt(id, 10)
	}

	h.logger.Info("Tenant created", zap.String("id", tenant.ID), zap.String("name", tenant.Name))
	h.writeJSON(w, tenant)
}

// GetTenant handles tenant retrieval requests
func (h *TenantHandler) GetTenant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := chi.URLParam(r, "id")

	// Check if user is system admin or has access to this tenant
	if !h.isSystemAdmin(ctx, r) && !h.hasTenantAccess(ctx, r, tenantID) {
		h.writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	var tenant auth.Tenant
	var settingsJSON, metadataJSON, limitsJSON sql.NullString
	var deletedAt sql.NullTime

	query := `
		SELECT id, name, slug, domain, logo, description, settings, metadata,
			   is_active, plan, limits, created_at, updated_at, deleted_at
		FROM tenants
		WHERE id = ?
	`
	err := h.db.QueryRowContext(ctx, query, tenantID).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.Slug,
		&tenant.Domain,
		&tenant.Logo,
		&tenant.Description,
		&settingsJSON,
		&metadataJSON,
		&tenant.IsActive,
		&tenant.Plan,
		&limitsJSON,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&deletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			h.writeError(w, http.StatusNotFound, "Tenant not found")
			return
		}
		h.logger.Error("Failed to get tenant", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to get tenant")
		return
	}

	if deletedAt.Valid {
		tenant.DeletedAt = &deletedAt.Time
	}

	// Parse JSON fields
	if settingsJSON.Valid {
		json.Unmarshal([]byte(settingsJSON.String), &tenant.Settings)
	}
	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &tenant.Metadata)
	}
	if limitsJSON.Valid {
		json.Unmarshal([]byte(limitsJSON.String), &tenant.Limits)
	}

	h.writeJSON(w, tenant)
}

// UpdateTenant handles tenant update requests
func (h *TenantHandler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := chi.URLParam(r, "id")

	// Check if user is system admin or tenant admin
	if !h.isSystemAdmin(ctx, r) && !h.hasTenantRole(ctx, r, tenantID, auth.TenantRoleAdmin) {
		h.writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	var req TenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Build update query
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != "" {
		updates = append(updates, "name = ?")
		args = append(args, req.Name)
		argIndex++
	}
	if req.Slug != "" {
		updates = append(updates, "slug = ?")
		args = append(args, req.Slug)
		argIndex++
	}
	if req.Domain != "" {
		updates = append(updates, "domain = ?")
		args = append(args, req.Domain)
		argIndex++
	}
	if req.Logo != "" {
		updates = append(updates, "logo = ?")
		args = append(args, req.Logo)
		argIndex++
	}
	if req.Description != "" {
		updates = append(updates, "description = ?")
		args = append(args, req.Description)
		argIndex++
	}
	if req.Plan != "" {
		updates = append(updates, "plan = ?")
		args = append(args, req.Plan)
		argIndex++
	}

	// Handle JSON fields
	if len(req.Settings.EnabledFeatures) > 0 || req.Settings.AllowUserRegistration {
		settingsJSON, _ := json.Marshal(req.Settings)
		updates = append(updates, "settings = ?")
		args = append(args, string(settingsJSON))
		argIndex++
	}
	if len(req.Metadata) > 0 {
		metadataJSON, _ := json.Marshal(req.Metadata)
		updates = append(updates, "metadata = ?")
		args = append(args, string(metadataJSON))
		argIndex++
	}

	if len(updates) == 0 {
		h.writeError(w, http.StatusBadRequest, "No updates provided")
		return
	}

	// Add updated_at and tenant ID
	updates = append(updates, "updated_at = ?")
	args = append(args, time.Now())
	argIndex++
	args = append(args, tenantID)

	query := "UPDATE tenants SET " + join(updates, ", ") + " WHERE id = ?"
	_, err := h.db.ExecContext(ctx, query, args...)
	if err != nil {
		h.logger.Error("Failed to update tenant", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to update tenant")
		return
	}

	h.logger.Info("Tenant updated", zap.String("id", tenantID))

	// Return updated tenant
	h.GetTenant(w, r)
}

// DeleteTenant handles tenant deletion requests (soft delete)
func (h *TenantHandler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := chi.URLParam(r, "id")

	// Only system admin can delete tenants
	if !h.isSystemAdmin(ctx, r) {
		h.writeError(w, http.StatusForbidden, "Access denied: system admin required")
		return
	}

	// Prevent deletion of system tenant
	if tenantID == auth.SystemTenantID {
		h.writeError(w, http.StatusBadRequest, "Cannot delete system tenant")
		return
	}

	query := "UPDATE tenants SET deleted_at = ?, is_active = 0 WHERE id = ?"
	_, err := h.db.ExecContext(ctx, query, time.Now(), tenantID)
	if err != nil {
		h.logger.Error("Failed to delete tenant", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to delete tenant")
		return
	}

	response := map[string]interface{}{
		"message": "Tenant deleted successfully",
		"id":      tenantID,
	}

	h.logger.Info("Tenant deleted", zap.String("id", tenantID))
	h.writeJSON(w, response)
}

// ListProjects handles project listing requests for a tenant
func (h *TenantHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := chi.URLParam(r, "tenantId")

	// Check if user is system admin or has tenant access
	if !h.isSystemAdmin(ctx, r) && !h.hasTenantAccess(ctx, r, tenantID) {
		h.writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Get query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Query database
	rows, err := h.db.QueryContext(ctx, `
		SELECT id, tenant_id, name, slug, description, logo, settings, metadata,
			   is_active, is_public, environment, owner_id, members, created_at, updated_at, deleted_at
		FROM projects
		WHERE tenant_id = ? AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, tenantID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to query projects", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to query projects")
		return
	}
	defer rows.Close()

	var projects []auth.Project
	for rows.Next() {
		var project auth.Project
		var settingsJSON, metadataJSON, membersJSON sql.NullString
		var deletedAt sql.NullTime

		err := rows.Scan(
			&project.ID,
			&project.TenantID,
			&project.Name,
			&project.Slug,
			&project.Description,
			&project.Logo,
			&settingsJSON,
			&metadataJSON,
			&project.IsActive,
			&project.IsPublic,
			&project.Environment,
			&project.OwnerID,
			&membersJSON,
			&project.CreatedAt,
			&project.UpdatedAt,
			&deletedAt,
		)
		if err != nil {
			h.logger.Error("Failed to scan project row", zap.Error(err))
			continue
		}

		if deletedAt.Valid {
			project.DeletedAt = &deletedAt.Time
		}

		// Parse JSON fields
		if settingsJSON.Valid {
			json.Unmarshal([]byte(settingsJSON.String), &project.Settings)
		}
		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &project.Metadata)
		}
		if membersJSON.Valid {
			json.Unmarshal([]byte(membersJSON.String), &project.Members)
		}

		projects = append(projects, project)
	}

	// Get total count
	var total int
	h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM projects WHERE tenant_id = ? AND deleted_at IS NULL", tenantID).Scan(&total)

	response := map[string]interface{}{
		"projects": projects,
		"total":    total,
		"page":     page,
		"limit":    limit,
	}

	h.writeJSON(w, response)
}

// CreateProject handles project creation requests
func (h *TenantHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := chi.URLParam(r, "tenantId")

	// Check if user is system admin or tenant admin
	if !h.isSystemAdmin(ctx, r) && !h.hasTenantRole(ctx, r, tenantID, auth.TenantRoleAdmin) {
		h.writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	var req TenantProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate request
	if req.Name == "" {
		h.writeError(w, http.StatusBadRequest, "Name is required")
		return
	}
	if req.Slug == "" {
		h.writeError(w, http.StatusBadRequest, "Slug is required")
		return
	}

	// Get user ID from context (from JWT/auth middleware)
	userID := h.getUserID(ctx)
	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Create project
	project := &auth.Project{
		TenantID:    tenantID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Logo:        req.Logo,
		Settings:    req.Settings,
		Metadata:    req.Metadata,
		IsActive:    true,
		IsPublic:    req.IsPublic,
		Environment: req.Environment,
		OwnerID:     userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Set default environment if not provided
	if project.Environment == "" {
		project.Environment = auth.EnvDevelopment
	}

	// Serialize JSON fields
	settingsJSON, _ := json.Marshal(project.Settings)
	metadataJSON, _ := json.Marshal(project.Metadata)
	membersJSON, _ := json.Marshal(project.Members)

	// Insert into database
	query := `
		INSERT INTO projects (id, tenant_id, name, slug, description, logo, settings, metadata,
							is_active, is_public, environment, owner_id, members, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := h.db.ExecContext(ctx, query,
		project.ID,
		project.TenantID,
		project.Name,
		project.Slug,
		project.Description,
		project.Logo,
		string(settingsJSON),
		string(metadataJSON),
		project.IsActive,
		project.IsPublic,
		project.Environment,
		project.OwnerID,
		string(membersJSON),
		project.CreatedAt,
		project.UpdatedAt,
	)
	if err != nil {
		h.logger.Error("Failed to create project", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to create project")
		return
	}

	id, _ := result.LastInsertId()
	if id != 0 {
		// This shouldn't happen with UUID but handle it
		project.ID = strconv.FormatInt(id, 10)
	}

	// Add owner as project member
	h.addUserToProject(ctx, userID, tenantID, project.ID, auth.ProjectRoleOwner)

	h.logger.Info("Project created",
		zap.String("id", project.ID),
		zap.String("name", project.Name),
		zap.String("tenant_id", tenantID))

	h.writeJSON(w, project)
}

// GetProject handles project retrieval requests
func (h *TenantHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := chi.URLParam(r, "projectId")

	// Get project from database
	var project auth.Project
	var settingsJSON, metadataJSON, membersJSON sql.NullString
	var deletedAt sql.NullTime

	query := `
		SELECT id, tenant_id, name, slug, description, logo, settings, metadata,
			   is_active, is_public, environment, owner_id, members, created_at, updated_at, deleted_at
		FROM projects
		WHERE id = ?
	`
	err := h.db.QueryRowContext(ctx, query, projectID).Scan(
		&project.ID,
		&project.TenantID,
		&project.Name,
		&project.Slug,
		&project.Description,
		&project.Logo,
		&settingsJSON,
		&metadataJSON,
		&project.IsActive,
		&project.IsPublic,
		&project.Environment,
		&project.OwnerID,
		&membersJSON,
		&project.CreatedAt,
		&project.UpdatedAt,
		&deletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			h.writeError(w, http.StatusNotFound, "Project not found")
			return
		}
		h.logger.Error("Failed to get project", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to get project")
		return
	}

	if deletedAt.Valid {
		project.DeletedAt = &deletedAt.Time
	}

	// Parse JSON fields
	if settingsJSON.Valid {
		json.Unmarshal([]byte(settingsJSON.String), &project.Settings)
	}
	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &project.Metadata)
	}
	if membersJSON.Valid {
		json.Unmarshal([]byte(membersJSON.String), &project.Members)
	}

	// Check access permissions
	if !h.isSystemAdmin(ctx, r) && !h.hasProjectAccess(ctx, r, projectID) {
		h.writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	h.writeJSON(w, project)
}

// UpdateProject handles project update requests
func (h *TenantHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := chi.URLParam(r, "projectId")

	// Get project tenant ID first
	var tenantID string
	err := h.db.QueryRowContext(ctx, "SELECT tenant_id FROM projects WHERE id = ?", projectID).Scan(&tenantID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "Project not found")
		return
	}

	// Check permissions
	if !h.isSystemAdmin(ctx, r) && !h.hasProjectRole(ctx, r, projectID, auth.ProjectRoleAdmin) {
		h.writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	var req TenantProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Build update query
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != "" {
		updates = append(updates, "name = ?")
		args = append(args, req.Name)
		argIndex++
	}
	if req.Slug != "" {
		updates = append(updates, "slug = ?")
		args = append(args, req.Slug)
		argIndex++
	}
	if req.Description != "" {
		updates = append(updates, "description = ?")
		args = append(args, req.Description)
		argIndex++
	}
	if req.Logo != "" {
		updates = append(updates, "logo = ?")
		args = append(args, req.Logo)
		argIndex++
	}
	if req.Environment != "" {
		updates = append(updates, "environment = ?")
		args = append(args, req.Environment)
		argIndex++
	}

	updates = append(updates, "is_public = ?")
	args = append(args, req.IsPublic)
	argIndex++

	// Handle JSON fields
	if len(req.Settings.EnabledFeatures) > 0 || req.Settings.RequireAuthForRead {
		settingsJSON, _ := json.Marshal(req.Settings)
		updates = append(updates, "settings = ?")
		args = append(args, string(settingsJSON))
		argIndex++
	}
	if len(req.Metadata) > 0 {
		metadataJSON, _ := json.Marshal(req.Metadata)
		updates = append(updates, "metadata = ?")
		args = append(args, string(metadataJSON))
		argIndex++
	}

	if len(updates) == 1 { // Only is_public was updated
		h.writeError(w, http.StatusBadRequest, "No meaningful updates provided")
		return
	}

	// Add updated_at and project ID
	updates = append(updates, "updated_at = ?")
	args = append(args, time.Now())
	argIndex++
	args = append(args, projectID)

	query := "UPDATE projects SET " + join(updates, ", ") + " WHERE id = ?"
	_, err = h.db.ExecContext(ctx, query, args...)
	if err != nil {
		h.logger.Error("Failed to update project", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to update project")
		return
	}

	h.logger.Info("Project updated", zap.String("id", projectID))

	// Return updated project
	h.GetProject(w, r)
}

// DeleteProject handles project deletion requests (soft delete)
func (h *TenantHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := chi.URLParam(r, "projectId")

	// Get project tenant ID first
	var tenantID string
	err := h.db.QueryRowContext(ctx, "SELECT tenant_id FROM projects WHERE id = ?", projectID).Scan(&tenantID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "Project not found")
		return
	}

	// Check permissions - only system admin or tenant admin can delete projects
	if !h.isSystemAdmin(ctx, r) && !h.hasTenantRole(ctx, r, tenantID, auth.TenantRoleAdmin) {
		h.writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Prevent deletion of system project
	if projectID == auth.SystemProjectID {
		h.writeError(w, http.StatusBadRequest, "Cannot delete system project")
		return
	}

	query := "UPDATE projects SET deleted_at = ?, is_active = 0 WHERE id = ?"
	_, err = h.db.ExecContext(ctx, query, time.Now(), projectID)
	if err != nil {
		h.logger.Error("Failed to delete project", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to delete project")
		return
	}

	response := map[string]interface{}{
		"message": "Project deleted successfully",
		"id":      projectID,
	}

	h.logger.Info("Project deleted", zap.String("id", projectID))
	h.writeJSON(w, response)
}

// AddUserToTenant handles adding a user to a tenant
func (h *TenantHandler) AddUserToTenant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := chi.URLParam(r, "tenantId")

	// Check if user is system admin or tenant admin
	if !h.isSystemAdmin(ctx, r) && !h.hasTenantRole(ctx, r, tenantID, auth.TenantRoleAdmin) {
		h.writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	var req UserTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.UserID == "" {
		h.writeError(w, http.StatusBadRequest, "User ID is required")
		return
	}
	if req.Role == "" {
		req.Role = auth.TenantRoleMember
	}

	// Add user to tenant
	err := h.addUserToTenant(ctx, req.UserID, tenantID, req.Role)
	if err != nil {
		h.logger.Error("Failed to add user to tenant", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to add user to tenant")
		return
	}

	response := map[string]interface{}{
		"message":   "User added to tenant successfully",
		"user_id":   req.UserID,
		"tenant_id": tenantID,
		"role":      req.Role,
	}

	h.writeJSON(w, response)
}

// AddUserToProject handles adding a user to a project
func (h *TenantHandler) AddUserToProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := chi.URLParam(r, "projectId")

	// Check if user is system admin or project admin
	if !h.isSystemAdmin(ctx, r) && !h.hasProjectRole(ctx, r, projectID, auth.ProjectRoleAdmin) {
		h.writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	var req ProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.UserID == "" {
		h.writeError(w, http.StatusBadRequest, "User ID is required")
		return
	}
	if req.Role == "" {
		req.Role = auth.ProjectRoleViewer
	}

	// Get project tenant ID first
	var tenantID string
	err := h.db.QueryRowContext(ctx, "SELECT tenant_id FROM projects WHERE id = ?", projectID).Scan(&tenantID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "Project not found")
		return
	}

	// Add user to project
	err = h.addUserToProject(ctx, req.UserID, tenantID, projectID, req.Role)
	if err != nil {
		h.logger.Error("Failed to add user to project", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to add user to project")
		return
	}

	response := map[string]interface{}{
		"message":    "User added to project successfully",
		"user_id":    req.UserID,
		"project_id": projectID,
		"role":       req.Role,
	}

	h.writeJSON(w, response)
}

// GetProjects handles getting user's projects in a tenant
func (h *TenantHandler) GetProjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := chi.URLParam(r, "tenantId")

	userID := h.getUserID(ctx)
	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Query user's projects
	rows, err := h.db.QueryContext(ctx, `
		SELECT p.id, p.tenant_id, p.name, p.slug, p.description, p.logo, p.settings, p.metadata,
			   p.is_active, p.is_public, p.environment, p.owner_id, p.members, p.created_at, p.updated_at,
			   up.role as user_role
		FROM projects p
		INNER JOIN user_projects up ON p.id = up.project_id
		WHERE up.user_id = ? AND up.tenant_id = ? AND up.is_active = 1 AND p.deleted_at IS NULL
		ORDER BY p.created_at DESC
	`, userID, tenantID)
	if err != nil {
		h.logger.Error("Failed to query user projects", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to query user projects")
		return
	}
	defer rows.Close()

	type Project struct {
		auth.Project
		UserRole string `json:"user_role"`
	}

	var projects []Project
	for rows.Next() {
		var project Project
		var settingsJSON, metadataJSON, membersJSON sql.NullString

		err := rows.Scan(
			&project.ID,
			&project.TenantID,
			&project.Name,
			&project.Slug,
			&project.Description,
			&project.Logo,
			&settingsJSON,
			&metadataJSON,
			&project.IsActive,
			&project.IsPublic,
			&project.Environment,
			&project.OwnerID,
			&membersJSON,
			&project.CreatedAt,
			&project.UpdatedAt,
			&project.UserRole,
		)
		if err != nil {
			h.logger.Error("Failed to scan user project row", zap.Error(err))
			continue
		}

		// Parse JSON fields
		if settingsJSON.Valid {
			json.Unmarshal([]byte(settingsJSON.String), &project.Settings)
		}
		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &project.Metadata)
		}
		if membersJSON.Valid {
			json.Unmarshal([]byte(membersJSON.String), &project.Members)
		}

		projects = append(projects, project)
	}

	response := map[string]interface{}{
		"projects": projects,
		"total":    len(projects),
	}

	h.writeJSON(w, response)
}

// InviteUserToProject handles inviting users to projects (cross-tenant collaboration)
func (h *TenantHandler) InviteUserToProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := chi.URLParam(r, "projectId")

	// Get project context
	projectCtx := middleware.GetProjectContext(r)
	if projectCtx == nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to get project context")
		return
	}

	var req InviteUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Role == "" {
		req.Role = auth.ProjectRoleViewer // Default role
	}

	// Validate role
	validRoles := []string{auth.ProjectRoleViewer, auth.ProjectRoleCollaborator, auth.ProjectRoleOwner}
	validRole := false
	for _, role := range validRoles {
		if req.Role == role {
			validRole = true
			break
		}
	}
	if !validRole {
		h.writeError(w, http.StatusBadRequest, "Invalid role")
		return
	}

	// Get current user ID for "invited by" field
	invitedBy := h.getUserID(ctx)

	// Add user to project (supports cross-tenant collaboration)
	err := h.tenantManager.AddUserToProject(req.UserID, projectID, req.Role, invitedBy)
	if err != nil {
		h.logger.Error("Failed to invite user to project",
			zap.String("user_id", req.UserID),
			zap.String("project_id", projectID),
			zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to invite user to project")
		return
	}

	response := map[string]interface{}{
		"message":    "User invited to project successfully",
		"user_id":    req.UserID,
		"project_id": projectID,
		"role":       req.Role,
	}

	h.logger.Info("User invited to project",
		zap.String("invited_user", req.UserID),
		zap.String("project_id", projectID),
		zap.String("invited_by", invitedBy))

	h.writeJSON(w, response)
}

// ListProjectMembers handles listing project members
func (h *TenantHandler) ListProjectMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := chi.URLParam(r, "projectId")

	// Check project access (viewer level is sufficient to see members)
	if !h.isSystemAdmin(ctx, r) && !h.hasProjectAccess(ctx, r, projectID) {
		h.writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Get project members
	members, err := h.tenantManager.GetProjectMembers(projectID)
	if err != nil {
		h.logger.Error("Failed to get project members",
			zap.String("project_id", projectID),
			zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to get project members")
		return
	}

	response := map[string]interface{}{
		"members": members,
		"total":   len(members),
	}

	h.writeJSON(w, response)
}

// RemoveUserFromProject handles removing users from projects
func (h *TenantHandler) RemoveUserFromProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := chi.URLParam(r, "projectId")
	userID := chi.URLParam(r, "userId")

	// Check if current user can manage project members
	if !h.isSystemAdmin(ctx, r) && !h.canManageProject(ctx, r, projectID) {
		h.writeError(w, http.StatusForbidden, "Access denied: insufficient permissions")
		return
	}

	// Prevent removing project creator
	members, err := h.tenantManager.GetProjectMembers(projectID)
	if err == nil {
		for _, member := range members {
			if member.UserID == userID && member.IsCreator {
				h.writeError(w, http.StatusBadRequest, "Cannot remove project creator from project")
				return
			}
		}
	}

	// Remove user from project (soft delete - set is_active to false)
	query := `
		UPDATE user_projects
		SET is_active = 0, left_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND project_id = ?
	`
	_, err = h.db.ExecContext(ctx, query, userID, projectID)
	if err != nil {
		h.logger.Error("Failed to remove user from project",
			zap.String("user_id", userID),
			zap.String("project_id", projectID),
			zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to remove user from project")
		return
	}

	response := map[string]interface{}{
		"message":    "User removed from project successfully",
		"user_id":    userID,
		"project_id": projectID,
	}

	h.logger.Info("User removed from project",
		zap.String("removed_user", userID),
		zap.String("project_id", projectID))

	h.writeJSON(w, response)
}

// TransferOwnership handles transferring project ownership
func (h *TenantHandler) TransferOwnership(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := chi.URLParam(r, "projectId")

	var req TransferOwnershipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.ToUserID == "" {
		h.writeError(w, http.StatusBadRequest, "Target user ID is required")
		return
	}

	currentUserID := h.getUserID(ctx)

	// Transfer ownership
	err := h.tenantManager.TransferProjectOwnership(projectID, currentUserID, req.ToUserID)
	if err != nil {
		h.logger.Error("Failed to transfer project ownership",
			zap.String("project_id", projectID),
			zap.String("from_user", currentUserID),
			zap.String("to_user", req.ToUserID),
			zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to transfer ownership")
		return
	}

	response := map[string]interface{}{
		"message":    "Project ownership transferred successfully",
		"project_id": projectID,
		"from_user":  currentUserID,
		"to_user":    req.ToUserID,
	}

	h.logger.Info("Project ownership transferred",
		zap.String("project_id", projectID),
		zap.String("from_user", currentUserID),
		zap.String("to_user", req.ToUserID))

	h.writeJSON(w, response)
}

// ListProjects handles listing all projects for the current user
func (h *TenantHandler) ListUserProjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := h.getUserID(ctx)

	// Get query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Query user's projects with full project details
	query := `
		SELECT p.id, p.tenant_id, p.name, p.slug, p.description, p.logo, p.settings, p.metadata,
			   p.is_active, p.is_public, p.environment, p.owner_id, p.created_at, p.updated_at,
			   up.role as user_role, up.is_creator, up.is_external_collaborator,
			   up.can_invite, up.can_manage_members, up.joined_at
		FROM projects p
		INNER JOIN user_projects up ON p.id = up.project_id
		WHERE up.user_id = ? AND up.is_active = 1 AND p.deleted_at IS NULL
		ORDER BY up.joined_at DESC
		LIMIT ? OFFSET ?
	`
	rows, err := h.db.QueryContext(ctx, query, userID, limit, (page-1)*limit)
	if err != nil {
		h.logger.Error("Failed to query user projects", zap.String("user_id", userID), zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to query projects")
		return
	}
	defer rows.Close()

	type ProjectDetail struct {
		auth.Project
		UserRole               string    `json:"user_role"`
		IsCreator              bool      `json:"is_creator"`
		IsExternalCollaborator bool      `json:"is_external_collaborator"`
		CanInvite              bool      `json:"can_invite"`
		CanManageMembers       bool      `json:"can_manage_members"`
		JoinedAt               time.Time `json:"joined_at"`
	}

	var projects []ProjectDetail
	for rows.Next() {
		var project ProjectDetail
		var settingsJSON, metadataJSON sql.NullString

		err := rows.Scan(
			&project.ID,
			&project.TenantID,
			&project.Name,
			&project.Slug,
			&project.Description,
			&project.Logo,
			&settingsJSON,
			&metadataJSON,
			&project.IsActive,
			&project.IsPublic,
			&project.Environment,
			&project.OwnerID,
			&project.CreatedAt,
			&project.UpdatedAt,
			&project.UserRole,
			&project.IsCreator,
			&project.IsExternalCollaborator,
			&project.CanInvite,
			&project.CanManageMembers,
			&project.JoinedAt,
		)
		if err != nil {
			h.logger.Error("Failed to scan user project row", zap.Error(err))
			continue
		}

		// Parse JSON fields
		if settingsJSON.Valid {
			json.Unmarshal([]byte(settingsJSON.String), &project.Settings)
		}
		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &project.Metadata)
		}

		projects = append(projects, project)
	}

	// Get total count
	var total int
	h.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM projects p
		INNER JOIN user_projects up ON p.id = up.project_id
		WHERE up.user_id = ? AND up.is_active = 1 AND p.deleted_at IS NULL
	`, userID).Scan(&total)

	response := map[string]interface{}{
		"projects": projects,
		"total":    total,
		"page":     page,
		"limit":    limit,
	}

	h.writeJSON(w, response)
}

// Helper methods for checking permissions
func (h *TenantHandler) canManageProject(ctx context.Context, r *http.Request, projectID string) bool {
	projectCtx := middleware.GetProjectContext(r)
	return projectCtx != nil && (projectCtx.CanManageMembers || projectCtx.IsOwner || projectCtx.IsSystem)
}

func (h *TenantHandler) hasProjectAccess(ctx context.Context, r *http.Request, projectID string) bool {
	// Check if user has any access to project via middleware
	projectCtx := middleware.GetProjectContext(r)
	return projectCtx != nil && projectCtx.ProjectID == projectID
}

// Helper methods

func (h *TenantHandler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *TenantHandler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   message,
		"status":  status,
		"success": false,
	})
}

func (h *TenantHandler) isSystemAdmin(ctx context.Context, r *http.Request) bool {
	// TODO: Implement proper authentication/authorization check
	// This would check JWT claims or session data
	return true // For now, allow all
}

func (h *TenantHandler) hasTenantAccess(ctx context.Context, r *http.Request, tenantID string) bool {
	// TODO: Implement proper tenant access check
	return true // For now, allow all
}

func (h *TenantHandler) hasTenantRole(ctx context.Context, r *http.Request, tenantID, requiredRole string) bool {
	// TODO: Implement proper tenant role check
	return true // For now, allow all
}

func (h *TenantHandler) hasProjectRole(ctx context.Context, r *http.Request, projectID, requiredRole string) bool {
	// TODO: Implement proper project role check
	return true // For now, allow all
}

func (h *TenantHandler) getUserID(ctx context.Context) string {
	// TODO: Extract user ID from JWT/session
	return "user_1" // For now, return mock user ID
}

func (h *TenantHandler) addUserToTenant(ctx context.Context, userID, tenantID, role string) error {
	query := `
		INSERT OR IGNORE INTO user_tenants (user_id, tenant_id, role, is_active, joined_at)
		VALUES (?, ?, ?, 1, ?)
	`
	_, err := h.db.ExecContext(ctx, query, userID, tenantID, role, time.Now())
	return err
}

func (h *TenantHandler) addUserToProject(ctx context.Context, userID, tenantID, projectID, role string) error {
	query := `
		INSERT OR IGNORE INTO user_projects (user_id, tenant_id, project_id, role, is_active, joined_at)
		VALUES (?, ?, ?, ?, 1, ?)
	`
	_, err := h.db.ExecContext(ctx, query, userID, tenantID, projectID, role, time.Now())
	return err
}

// join is a helper function to join string slices
func join(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
