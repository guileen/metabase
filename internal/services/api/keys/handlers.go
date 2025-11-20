package keys

import ("context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"go.uber.org/zap")

// Handlers API密钥HTTP处理器
type Handlers struct {
	manager *Manager
	logger  *zap.Logger
}

// NewHandlers 创建API密钥处理器
func NewHandlers(manager *Manager, logger *zap.Logger) *Handlers {
	return &Handlers{
		manager: manager,
		logger:  logger,
	}
}

// RegisterRoutes 注册路由
func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/keys", func(r chi.Router) {
		// 公开路由（不需要API密钥认证）
		r.Post("/", h.createKey) // 创建密钥时可能使用其他认证方式

		// 需要API密钥认证的路由
		r.Group(func(r chi.Router) {
			r.Use(h.authenticateMiddleware)
			r.Get("/", h.listKeys)
			r.Get("/{id}", h.getKey)
			r.Put("/{id}", h.updateKey)
			r.Delete("/{id}", h.deleteKey)
			r.Post("/{id}/revoke", h.revokeKey)
			r.Get("/{id}/stats", h.getKeyStats)
		})
	})

	// 管理路由（需要管理员权限）
	r.Route("/api/v1/admin/keys", func(r chi.Router) {
		r.Use(h.authenticateMiddleware)
		r.Use(h.adminMiddleware)
		r.Get("/", h.listAllKeys)
		r.Post("/", h.createSystemKey)
		r.Get("/stats", h.getAllKeyStats)
	})
}

// createKey 创建API密钥
func (h *Handlers) createKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateKeyRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		h.logger.Error("failed to decode request", zap.Error(err))
		render.JSON(w, r, &ErrorResponse{
			Error: "invalid_request",
			Message: "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	// 验证请求
	if err := h.validateCreateKeyRequest(&req); err != nil {
		h.logger.Error("invalid create key request", zap.Error(err))
		render.JSON(w, r, &ErrorResponse{
			Error: "validation_error",
			Message: err.Error(),
		})
		return
	}

	// 创建API密钥
	key, actualKey, err := h.manager.CreateKey(ctx, &req)
	if err != nil {
		h.logger.Error("failed to create API key", zap.Error(err))
		render.JSON(w, r, &ErrorResponse{
			Error: "creation_failed",
			Message: "Failed to create API key",
			Details: err.Error(),
		})
		return
	}

	h.logger.Info("API key created",
		zap.String("id", key.ID),
		zap.String("name", key.Name),
		zap.String("type", string(key.Type)),
		zap.Strings("scopes", key.Scopes))

	// 返回响应（包含完整的密钥，只在创建时返回一次）
	render.JSON(w, r, &CreateKeyResponse{
		Key:        key,
		APIKey:     actualKey, // 只在创建时返回
		Message:    "API key created successfully",
	})
}

// listKeys 列出API密钥
func (h *Handlers) listKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 获取当前API密钥信息
	currentKey := ctx.Value("apiKey").(*APIKey)

	// 解析查询参数
	filter := &KeyFilter{
		Limit:  20,
		Offset: 0,
	}

	if currentKey.TenantID != nil {
		filter.TenantID = currentKey.TenantID
	}
	if currentKey.ProjectID != nil {
		filter.ProjectID = currentKey.ProjectID
	}

	// 非系统密钥只能查看自己的密钥
	if currentKey.Type != KeyTypeSystem && currentKey.UserID != nil {
		filter.UserID = currentKey.UserID
	}

	// 解析查询参数
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	if keyType := r.URL.Query().Get("type"); keyType != "" {
		filter.Type = (*KeyType)(&keyType)
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = (*KeyStatus)(&status)
	}

	keys, total, err := h.manager.ListKeys(ctx, filter)
	if err != nil {
		h.logger.Error("failed to list API keys", zap.Error(err))
		render.JSON(w, r, &ErrorResponse{
			Error: "list_failed",
			Message: "Failed to list API keys",
			Details: err.Error(),
		})
		return
	}

	render.JSON(w, r, &ListKeysResponse{
		Keys:  keys,
		Total: total,
		Limit: filter.Limit,
		Offset: filter.Offset,
	})
}

// getKey 获取单个API密钥
func (h *Handlers) getKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	keyID := chi.URLParam(r, "id")
	currentKey := ctx.Value("apiKey").(*APIKey)

	key, err := h.manager.GetKeyByID(ctx, keyID)
	if err != nil {
		h.logger.Error("failed to get API key", zap.String("id", keyID), zap.Error(err))
		render.JSON(w, r, &ErrorResponse{
			Error: "not_found",
			Message: "API key not found",
		})
		return
	}

	// 权限检查：只能查看自己有权访问的密钥
	if !h.canAccessKey(currentKey, key) {
		render.JSON(w, r, &ErrorResponse{
			Error: "access_denied",
			Message: "Access denied",
		})
		return
	}

	render.JSON(w, r, &GetKeyResponse{
		Key: key,
	})
}

// updateKey 更新API密钥
func (h *Handlers) updateKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	keyID := chi.URLParam(r, "id")
	currentKey := ctx.Value("apiKey").(*APIKey)

	var req UpdateKeyRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, &ErrorResponse{
			Error: "invalid_request",
			Message: "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	// 获取现有密钥
	key, err := h.manager.GetKeyByID(ctx, keyID)
	if err != nil {
		render.JSON(w, r, &ErrorResponse{
			Error: "not_found",
			Message: "API key not found",
		})
		return
	}

	// 权限检查
	if !h.canAccessKey(currentKey, key) {
		render.JSON(w, r, &ErrorResponse{
			Error: "access_denied",
			Message: "Access denied",
		})
		return
	}

	// 更新密钥
	updatedKey, err := h.manager.UpdateKey(ctx, keyID, &req)
	if err != nil {
		h.logger.Error("failed to update API key", zap.String("id", keyID), zap.Error(err))
		render.JSON(w, r, &ErrorResponse{
			Error: "update_failed",
			Message: "Failed to update API key",
			Details: err.Error(),
		})
		return
	}

	h.logger.Info("API key updated", zap.String("id", keyID))
	render.JSON(w, r, &UpdateKeyResponse{
		Key:     updatedKey,
		Message: "API key updated successfully",
	})
}

// deleteKey 删除API密钥
func (h *Handlers) deleteKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	keyID := chi.URLParam(r, "id")
	currentKey := ctx.Value("apiKey").(*APIKey)

	// 获取现有密钥
	key, err := h.manager.GetKeyByID(ctx, keyID)
	if err != nil {
		render.JSON(w, r, &ErrorResponse{
			Error: "not_found",
			Message: "API key not found",
		})
		return
	}

	// 权限检查
	if !h.canAccessKey(currentKey, key) {
		render.JSON(w, r, &ErrorResponse{
			Error: "access_denied",
			Message: "Access denied",
		})
		return
	}

	if err := h.manager.DeleteKey(ctx, keyID); err != nil {
		h.logger.Error("failed to delete API key", zap.String("id", keyID), zap.Error(err))
		render.JSON(w, r, &ErrorResponse{
			Error: "delete_failed",
			Message: "Failed to delete API key",
			Details: err.Error(),
		})
		return
	}

	h.logger.Info("API key deleted", zap.String("id", keyID))
	render.JSON(w, r, &DeleteKeyResponse{
		Message: "API key deleted successfully",
	})
}

// revokeKey 吊销API密钥
func (h *Handlers) revokeKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	keyID := chi.URLParam(r, "id")
	currentKey := ctx.Value("apiKey").(*APIKey)

	// 获取现有密钥
	key, err := h.manager.GetKeyByID(ctx, keyID)
	if err != nil {
		render.JSON(w, r, &ErrorResponse{
			Error: "not_found",
			Message: "API key not found",
		})
		return
	}

	// 权限检查
	if !h.canAccessKey(currentKey, key) {
		render.JSON(w, r, &ErrorResponse{
			Error: "access_denied",
			Message: "Access denied",
		})
		return
	}

	if err := h.manager.RevokeKey(ctx, keyID); err != nil {
		h.logger.Error("failed to revoke API key", zap.String("id", keyID), zap.Error(err))
		render.JSON(w, r, &ErrorResponse{
			Error: "revoke_failed",
			Message: "Failed to revoke API key",
			Details: err.Error(),
		})
		return
	}

	h.logger.Info("API key revoked", zap.String("id", keyID))
	render.JSON(w, r, &RevokeKeyResponse{
		Message: "API key revoked successfully",
	})
}

// getKeyStats 获取API密钥使用统计
func (h *Handlers) getKeyStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	keyID := chi.URLParam(r, "id")
	currentKey := ctx.Value("apiKey").(*APIKey)

	// 获取密钥信息
	key, err := h.manager.GetKeyByID(ctx, keyID)
	if err != nil {
		render.JSON(w, r, &ErrorResponse{
			Error: "not_found",
			Message: "API key not found",
		})
		return
	}

	// 权限检查
	if !h.canAccessKey(currentKey, key) {
		render.JSON(w, r, &ErrorResponse{
			Error: "access_denied",
			Message: "Access denied",
		})
		return
	}

	stats, err := h.manager.GetKeyStats(ctx, keyID)
	if err != nil {
		h.logger.Error("failed to get API key stats", zap.String("id", keyID), zap.Error(err))
		render.JSON(w, r, &ErrorResponse{
			Error: "stats_failed",
			Message: "Failed to get API key stats",
			Details: err.Error(),
		})
		return
	}

	render.JSON(w, r, &GetKeyStatsResponse{
		Stats: stats,
	})
}

// 私有辅助方法

func (h *Handlers) validateCreateKeyRequest(req *CreateKeyRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}

	if req.Type == "" {
		return fmt.Errorf("type is required")
	}

	// 验证密钥类型
	switch req.Type {
	case KeyTypeSystem, KeyTypeUser, KeyTypeService:
		// 有效类型
	default:
		return fmt.Errorf("invalid key type: %s", req.Type)
	}

	// 验证权限范围
	for _, scope := range req.Scopes {
		if _, exists := KeyScopes[scope]; !exists {
			return fmt.Errorf("invalid scope: %s", scope)
		}
	}

	return nil
}

func (h *Handlers) canAccessKey(currentKey, targetKey *APIKey) bool {
	// 系统密钥可以访问所有密钥
	if currentKey.Type == KeyTypeSystem {
		return true
	}

	// 只能访问自己的密钥
	if targetKey.ID == currentKey.ID {
		return true
	}

	// 检查租户权限
	if currentKey.TenantID != nil && targetKey.TenantID != nil {
		if *currentKey.TenantID != *targetKey.TenantID {
			return false
		}
	}

	// 检查项目权限
	if currentKey.ProjectID != nil && targetKey.ProjectID != nil {
		if *currentKey.ProjectID != *targetKey.ProjectID {
			return false
		}
	}

	// 检查用户权限
	if currentKey.UserID != nil && targetKey.UserID != nil {
		if *currentKey.UserID != *targetKey.UserID {
			return false
		}
	}

	return true
}

// 中间件

func (h *Handlers) authenticateMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			render.JSON(w, r, &ErrorResponse{
				Error: "missing_auth",
				Message: "Authorization header required",
			})
			return
		}

		// 解析Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			render.JSON(w, r, &ErrorResponse{
				Error: "invalid_auth",
				Message: "Invalid authorization header format",
			})
			return
		}

		apiKey := parts[1]
		key, err := h.manager.GetKeyByKey(r.Context(), apiKey)
		if err != nil {
			h.logger.Warn("invalid API key", zap.Error(err))
			render.JSON(w, r, &ErrorResponse{
				Error: "invalid_key",
				Message: "Invalid API key",
			})
			return
		}

		if !key.IsValid() {
			render.JSON(w, r, &ErrorResponse{
				Error: "key_invalid",
				Message: "API key is invalid or expired",
			})
			return
		}

		// 将密钥信息添加到上下文
		ctx := context.WithValue(r.Context(), "apiKey", key)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handlers) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Context().Value("apiKey").(*APIKey)

		// 只有系统密钥或具有系统管理权限的密钥才能访问管理路由
		if key.Type != KeyTypeSystem && !key.HasScope("system:write") {
			render.JSON(w, r, &ErrorResponse{
				Error: "access_denied",
				Message: "Admin access required",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// 响应类型

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type CreateKeyResponse struct {
	Key        *APIKey `json:"key"`
	APIKey     string  `json:"api_key"` // 只在创建时返回
	Message    string  `json:"message"`
}

type GetKeyResponse struct {
	Key *APIKey `json:"key"`
}

type ListKeysResponse struct {
	Keys   []*APIKey `json:"keys"`
	Total  int64     `json:"total"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
}

type UpdateKeyResponse struct {
	Key     *APIKey `json:"key"`
	Message string  `json:"message"`
}

type DeleteKeyResponse struct {
	Message string `json:"message"`
}

type RevokeKeyResponse struct {
	Message string `json:"message"`
}

type GetKeyStatsResponse struct {
	Stats *KeyUsageStats `json:"stats"`
}

// 管理路由的处理器（简化实现）

func (h *Handlers) listAllKeys(w http.ResponseWriter, r *http.Request) {
	// 管理员可以查看所有密钥
	h.listKeys(w, r)
}

func (h *Handlers) createSystemKey(w http.ResponseWriter, r *http.Request) {
	h.createKey(w, r)
}

func (h *Handlers) getAllKeyStats(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, map[string]interface{}{
		"message": "Global key statistics - to be implemented",
	})
}