package rest

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"go.uber.org/zap"
)

// Handler REST API处理器
type Handler struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewHandler 创建REST API处理器
func NewHandler(db *sql.DB, logger *zap.Logger) *Handler {
	return &Handler{
		db:     db,
		logger: logger,
	}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r chi.Router) {
	// RESTful API路由 - 需要API密钥认证
	r.Route("/rest/v1/{table}", func(r chi.Router) {
		r.Use(h.tableAccessMiddleware) // 表访问权限检查
		r.Get("/", h.handleQuery)
		r.Post("/", h.handleInsert)
		r.Patch("/", h.handleUpdate) // 批量更新
		r.Put("/", h.handleUpsert)    // 插入或更新
		r.Delete("/", h.handleDelete) // 批量删除
	})

	// 单个记录操作
	r.Route("/rest/v1/{table}/{id}", func(r chi.Router) {
		r.Use(h.tableAccessMiddleware)
		r.Get("/", h.handleGet)
		r.Patch("/", h.handleUpdateOne)
		r.Put("/", h.handleUpdateOne) // 单个记录更新
		r.Delete("/", h.handleDeleteOne)
	})

	// 表管理
	r.Route("/rest/v1", func(r chi.Router) {
		r.Get("/", h.handleListTables)
		r.Post("/", h.handleCreateTable)
		r.Get("/{table}/schema", h.handleGetTableSchema)
		r.Patch("/{table}/schema", h.handleUpdateTableSchema)
		r.Delete("/{table}", h.handleDropTable)
	})

	// 健康检查
	r.Get("/rest/health", h.handleHealth)

	// 实时订阅
	r.Route("/rest/realtime", func(r chi.Router) {
		r.Get("/{table}", h.handleRealtimeSubscribe)
	})
}

// handleQuery 处理查询请求
func (h *Handler) handleQuery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	table := chi.URLParam(r, "table")
	key := ctx.Value("apiKey").(*APIKey)

	// 解析查询参数
	options := h.parseQueryOptions(r)

	// 验证查询选项
	if err := h.validateQueryOptions(options); err != nil {
		h.logger.Error("invalid query options", zap.Error(err))
		render.JSON(w, r, &QueryResponse{
			Error: &QueryError{
				Code:    "invalid_query",
				Message: err.Error(),
			},
		})
		return
	}

	// 构建查询
	queryBuilder := NewQueryBuilder(table, OperationSelect, options)
	if err := queryBuilder.ValidateQuery(); err != nil {
		h.logger.Error("invalid query", zap.Error(err))
		render.JSON(w, r, &QueryResponse{
			Error: &QueryError{
				Code:    "validation_error",
				Message: err.Error(),
			},
		})
		return
	}

	query, args, err := queryBuilder.Build()
	if err != nil {
		h.logger.Error("failed to build query", zap.Error(err))
		render.JSON(w, r, &QueryResponse{
			Error: &QueryError{
				Code:    "query_build_error",
				Message: "Failed to build query",
				Details: err.Error(),
			},
		})
		return
	}

	// 记录查询
	h.logger.Debug("executing query",
		zap.String("table", table),
		zap.String("query", query),
		zap.Any("args", args),
		zap.String("api_key", key.ID),
	)

	// 执行查询
	startTime := time.Now()
	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		h.logger.Error("query execution failed",
			zap.String("query", query),
			zap.Any("args", args),
			zap.Error(err),
		)

		render.JSON(w, r, &QueryResponse{
			Error: &QueryError{
				Code:    "query_execution_error",
				Message: "Query execution failed",
				Details: err.Error(),
			},
		})
		return
	}
	defer rows.Close()

	// 获取列信息
	columns, err := rows.Columns()
	if err != nil {
		h.logger.Error("failed to get columns", zap.Error(err))
		render.JSON(w, r, &QueryResponse{
			Error: &QueryError{
				Code:    "schema_error",
				Message: "Failed to get schema information",
			},
		})
		return
	}

	// 读取数据
	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			h.logger.Error("failed to scan row", zap.Error(err))
			continue
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	// 获取总数（如果有分页）
	var count int64
	if options.Limit > 0 {
		countQuery, countArgs, err := queryBuilder.BuildCountQuery()
		if err == nil {
			h.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&count)
		}
	}

	// 记录成功
	h.logger.Info("query executed successfully",
		zap.String("table", table),
		zap.Int("rows", len(results)),
		zap.Duration("duration", time.Since(startTime)),
		zap.String("api_key", key.ID),
	)

	// 返回响应
	render.JSON(w, r, &QueryResponse{
		Data:    results,
		Count:   count,
		Limit:   options.Limit,
		Offset:  options.Offset,
		HasNext: int64(options.Offset+len(results)) < count,
	})
}

// handleInsert 处理插入请求
func (h *Handler) handleInsert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	table := chi.URLParam(r, "table")
	key := ctx.Value("apiKey").(*APIKey)

	var data map[string]interface{}
	if err := render.DecodeJSON(r.Body, &data); err != nil {
		render.JSON(w, r, &QueryResponse{
			Error: &QueryError{
				Code:    "invalid_data",
				Message: "Invalid JSON data",
				Details: err.Error(),
			},
		})
		return
	}

	// 检查插入权限
	if !key.HasScope("write") {
		render.JSON(w, r, &QueryResponse{
			Error: &QueryError{
				Code:    "permission_denied",
				Message: "Write permission required",
			},
		})
		return
	}

	// 解析插入选项
	options := h.parseInsertOptions(r)

	// 构建插入查询
	queryBuilder := NewQueryBuilder(table, OperationInsert, nil)
	queryBuilder.SetData(data)
	queryBuilder.options = options

	query, args, err := queryBuilder.Build()
	if err != nil {
		h.logger.Error("failed to build insert query", zap.Error(err))
		render.JSON(w, r, &QueryResponse{
			Error: &QueryError{
				Code:    "query_build_error",
				Message: "Failed to build insert query",
				Details: err.Error(),
			},
		})
		return
	}

	// 执行插入
	var result map[string]interface{}
	err = h.db.QueryRowContext(ctx, query, args...).Scan(
		// 这里需要根据RETURNING字段动态扫描
	)
	if err != nil {
		h.logger.Error("insert execution failed",
			zap.String("query", query),
			zap.Any("args", args),
			zap.Error(err),
		)

		render.JSON(w, r, &QueryResponse{
			Error: &QueryError{
				Code:    "insert_execution_error",
				Message: "Insert execution failed",
				Details: err.Error(),
			},
		})
		return
	}

	// 记录成功
	h.logger.Info("insert executed successfully",
		zap.String("table", table),
		zap.Any("data", data),
		zap.String("api_key", key.ID),
	)

	render.JSON(w, r, &QueryResponse{
		Data: result,
	})
}

// handleGet 处理获取单个记录
func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	table := chi.URLParam(r, "table")
	id := chi.URLParam(r, "id")
	key := ctx.Value("apiKey").(*APIKey)

	// 检查读取权限
	if !key.HasScope("read") {
		render.JSON(w, r, &QueryResponse{
			Error: &QueryError{
				Code:    "permission_denied",
				Message: "Read permission required",
			},
		})
		return
	}

	// 构建查询
	options := &QueryOptions{
		Where: map[string]interface{}{"id": id},
		Limit: 1,
	}

	queryBuilder := NewQueryBuilder(table, OperationSelect, options)
	query, args, err := queryBuilder.Build()
	if err != nil {
		render.JSON(w, r, &QueryResponse{
			Error: &QueryError{
				Code:    "query_build_error",
				Message: "Failed to build query",
				Details: err.Error(),
			},
		})
		return
	}

	// 执行查询
	row := h.db.QueryRowContext(ctx, query, args...)
	columns, err := rows.Columns()
	if err != nil {
		render.JSON(w, r, &QueryResponse{
			Error: &QueryError{
				Code:    "schema_error",
				Message: "Failed to get schema information",
			},
		})
		return
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := row.Scan(valuePtrs...); err != nil {
		if err == sql.ErrNoRows {
			render.JSON(w, r, &QueryResponse{
				Error: &QueryError{
					Code:    "not_found",
					Message: "Record not found",
				},
			})
			return
		}

		render.JSON(w, r, &QueryResponse{
			Error: &QueryError{
				Code:    "query_execution_error",
				Message: "Query execution failed",
				Details: err.Error(),
			},
		})
		return
	}

	// 构建结果
	result := make(map[string]interface{})
	for i, col := range columns {
		val := values[i]
		b, ok := val.([]byte)
		if ok {
			result[col] = string(b)
		} else {
			result[col] = val
		}
	}

	render.JSON(w, r, &QueryResponse{
		Data: result,
	})
}

// parseQueryOptions 解析查询选项
func (h *Handler) parseQueryOptions(r *http.Request) *QueryOptions {
	options := &QueryOptions{}

	// select字段
	if selectStr := r.URL.Query().Get("select"); selectStr != "" {
		options.Select = strings.Split(selectStr, ",")
	}

	// where条件
	where := make(map[string]interface{})
	for key, values := range r.URL.Query() {
		if key == "select" || key == "order" || key == "limit" || key == "offset" {
			continue
		}

		if len(values) > 0 {
			// 尝试解析JSON值
			var value interface{} = values[0]
			if err := json.Unmarshal([]byte(values[0]), &value); err != nil {
				// 如果不是JSON，使用字符串值
				if len(values) == 1 {
					value = values[0]
				} else {
					value = values
				}
			}

			where[key] = value
		}
	}
	options.Where = where

	// order
	if orderStr := r.URL.Query().Get("order"); orderStr != "" {
		options.OrderBy = orderStr
	}

	// limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			options.Limit = limit
		}
	}

	// offset
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			options.Offset = offset
		}
	}

	return options
}

// parseInsertOptions 解析插入选项
func (h *Handler) parseInsertOptions(r *http.Request) *InsertOptions {
	options := &InsertOptions{}

	// returning字段
	if returningStr := r.URL.Query().Get("returning"); returningStr != "" {
		options.Returning = strings.Split(returningStr, ",")
	}

	return options
}

// validateQueryOptions 验证查询选项
func (h *Handler) validateQueryOptions(options *QueryOptions) error {
	if options.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}

	if options.Limit > 1000 {
		return fmt.Errorf("limit cannot exceed 1000")
	}

	if options.Offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}

	return nil
}

// 中间件

func (h *Handler) tableAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		key := ctx.Value("apiKey").(*APIKey)
		table := chi.URLParam(r, "table")

		// 验证表名
		if err := h.validateTableName(table); err != nil {
			render.JSON(w, r, &QueryResponse{
				Error: &QueryError{
					Code:    "invalid_table",
					Message: err.Error(),
				},
			})
			return
		}

		// 检查表是否存在
		if !h.tableExists(ctx, table) {
			render.JSON(w, r, &QueryResponse{
				Error: &QueryError{
					Code:    "table_not_found",
					Message: fmt.Sprintf("Table '%s' not found", table),
				},
			})
			return
		}

		// TODO: 检查表访问权限（基于RLS或配置）
		// 这里可以实现更细粒度的表级别权限控制

		next.ServeHTTP(w, r)
	})
}

func (h *Handler) validateTableName(table string) error {
	if table == "" {
		return fmt.Errorf("table name is required")
	}

	// 检查是否包含危险字符
	if strings.ContainsAny(table, "';--") {
		return fmt.Errorf("potentially dangerous table name")
	}

	// 检查是否以特定前缀开头（受保护的系统表）
	protectedPrefixes := []string{"pg_", "information_schema", "sys_", "api_"}
	for _, prefix := range protectedPrefixes {
		if strings.HasPrefix(table, prefix) {
			return fmt.Errorf("access to table '%s' is not allowed", table)
		}
	}

	return nil
}

func (h *Handler) tableExists(ctx context.Context, table string) bool {
	var exists bool
	query := "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)"
	err := h.db.QueryRowContext(ctx, query, table).Scan(&exists)
	return err == nil && exists
}

// 健康检查
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 检查数据库连接
	var dbVersion string
	dbConnected := true
	if err := h.db.QueryRowContext(ctx, "SELECT version()").Scan(&dbVersion); err != nil {
		dbConnected = false
	}

	render.JSON(w, r, &HealthResponse{
		Status:    "ok",
		Version:   "1.0.0",
		Uptime:    "0h", // TODO: 实际运行时间
		Database: DatabaseStatus{
			Connected: dbConnected,
			Version:   dbVersion,
		},
		Cache: CacheStatus{
			Connected: true,
			Type:      "memory",
		},
		Timestamp: time.Now(),
	})
}

// 占位符方法 - 需要进一步实现
func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, &QueryResponse{
		Error: &QueryError{
			Code:    "not_implemented",
			Message: "Batch update not yet implemented",
		},
	})
}

func (h *Handler) handleUpsert(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, &QueryResponse{
		Error: &QueryError{
			Code:    "not_implemented",
			Message: "Upsert not yet implemented",
		},
	})
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, &QueryResponse{
		Error: &QueryError{
			Code:    "not_implemented",
			Message: "Batch delete not yet implemented",
		},
	})
}

func (h *Handler) handleUpdateOne(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, &QueryResponse{
		Error: &QueryError{
			Code:    "not_implemented",
			Message: "Single record update not yet implemented",
		},
	})
}

func (h *Handler) handleDeleteOne(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, &QueryResponse{
		Error: &QueryError{
			Code:    "not_implemented",
			Message: "Single record delete not yet implemented",
		},
	})
}

func (h *Handler) handleListTables(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, &QueryResponse{
		Error: &QueryError{
			Code:    "not_implemented",
			Message: "Table listing not yet implemented",
		},
	})
}

func (h *Handler) handleCreateTable(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, &QueryResponse{
		Error: &QueryError{
			Code:    "not_implemented",
			Message: "Table creation not yet implemented",
		},
	})
}

func (h *Handler) handleGetTableSchema(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, &QueryResponse{
		Error: &QueryError{
			Code:    "not_implemented",
			Message: "Table schema retrieval not yet implemented",
		},
	})
}

func (h *Handler) handleUpdateTableSchema(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, &QueryResponse{
		Error: &QueryError{
			Code:    "not_implemented",
			Message: "Table schema update not yet implemented",
		},
	})
}

func (h *Handler) handleDropTable(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, &QueryResponse{
		Error: &QueryError{
			Code:    "not_implemented",
			Message: "Table drop not yet implemented",
		},
	})
}

func (h *Handler) handleRealtimeSubscribe(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, &QueryResponse{
		Error: &QueryError{
			Code:    "not_implemented",
			Message: "Realtime subscription not yet implemented",
		},
	})
}

// APIKey类型 - 简化版本，实际应该从keys包导入
type APIKey struct {
	ID     string
	Type   string
	Scopes []string
}