package keys

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"go.uber.org/zap"
)

// Handler API密钥HTTP处理器
type Handler struct {
	manager *Manager
	logger  *zap.Logger
}

// NewHandler 创建新的API密钥处理器
func NewHandler(manager *Manager, logger *zap.Logger) *Handler {
	return &Handler{
		manager: manager,
		logger:  logger,
	}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.handleList)
	r.Post("/", h.handleCreate)
	r.Get("/{id}", h.handleGet)
	r.Put("/{id}", h.handleUpdate)
	r.Delete("/{id}", h.handleDelete)
}

// handleCreate 创建新的API密钥
func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateKeyRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, map[string]interface{}{
			"error":   "Invalid JSON data",
			"details": err.Error(),
		})
		return
	}

	// 验证请求
	if req.Name == "" {
		render.JSON(w, r, map[string]interface{}{
			"error": "Name is required",
		})
		return
	}

	// 创建密钥
	apiKey, err := h.manager.Create(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create api key", zap.Error(err))
		render.JSON(w, r, map[string]interface{}{
			"error":   "Failed to create API key",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("API key created successfully",
		zap.String("id", apiKey.ID),
		zap.String("name", apiKey.Name),
	)

	render.JSON(w, r, map[string]interface{}{
		"data": apiKey,
	})
}

// handleList 列出API密钥
func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	var tenantID, projectID *string

	if tenantIDStr := r.URL.Query().Get("tenant_id"); tenantIDStr != "" {
		tenantID = &tenantIDStr
	}

	if projectIDStr := r.URL.Query().Get("project_id"); projectIDStr != "" {
		projectID = &projectIDStr
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 || limit > 100 {
		limit = 20 // 默认限制
	}

	// 获取密钥列表
	keys, err := h.manager.List(r.Context(), tenantID, projectID, limit, offset)
	if err != nil {
		h.logger.Error("failed to list api keys", zap.Error(err))
		render.JSON(w, r, map[string]interface{}{
			"error":   "Failed to list API keys",
			"details": err.Error(),
		})
		return
	}

	render.JSON(w, r, map[string]interface{}{
		"data": keys,
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"count":  len(keys),
		},
	})
}

// handleGet 获取单个API密钥
func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, map[string]interface{}{
			"error": "API key ID is required",
		})
		return
	}

	apiKey, err := h.manager.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to get api key", zap.String("id", id), zap.Error(err))
		render.JSON(w, r, map[string]interface{}{
			"error":   "Failed to get API key",
			"details": err.Error(),
		})
		return
	}

	render.JSON(w, r, map[string]interface{}{
		"data": apiKey,
	})
}

// handleUpdate 更新API密钥
func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, map[string]interface{}{
			"error": "API key ID is required",
		})
		return
	}

	var req UpdateKeyRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, map[string]interface{}{
			"error":   "Invalid JSON data",
			"details": err.Error(),
		})
		return
	}

	// 更新密钥
	apiKey, err := h.manager.Update(r.Context(), id, &req)
	if err != nil {
		h.logger.Error("failed to update api key", zap.String("id", id), zap.Error(err))
		render.JSON(w, r, map[string]interface{}{
			"error":   "Failed to update API key",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("API key updated successfully",
		zap.String("id", id),
	)

	render.JSON(w, r, map[string]interface{}{
		"data": apiKey,
	})
}

// handleDelete 删除API密钥
func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, map[string]interface{}{
			"error": "API key ID is required",
		})
		return
	}

	err := h.manager.Delete(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to delete api key", zap.String("id", id), zap.Error(err))
		render.JSON(w, r, map[string]interface{}{
			"error":   "Failed to delete API key",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("API key deleted successfully",
		zap.String("id", id),
	)

	render.JSON(w, r, map[string]interface{}{
		"message": "API key deleted successfully",
	})
}
