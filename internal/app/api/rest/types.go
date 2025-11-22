package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// RequestMethod HTTP请求方法
type RequestMethod string

const (
	MethodGET    RequestMethod = "GET"
	MethodPOST   RequestMethod = "POST"
	MethodPUT    RequestMethod = "PUT"
	MethodPATCH  RequestMethod = "PATCH"
	MethodDELETE RequestMethod = "DELETE"
)

// TableSchema 表结构信息
type TableSchema struct {
	Name        string       `json:"name"`
	DisplayName string       `json:"display_name,omitempty"`
	Description string       `json:"description,omitempty"`
	Columns     []Column     `json:"columns"`
	Indexes     []Index      `json:"indexes,omitempty"`
	Constraints []Constraint `json:"constraints,omitempty"`
	PrimaryKeys []string     `json:"primary_keys,omitempty"`
	ForeignKeys []ForeignKey `json:"foreign_keys,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// Column 列信息
type Column struct {
	Name          string      `json:"name"`
	Type          string      `json:"type"`
	Nullable      bool        `json:"nullable"`
	DefaultValue  interface{} `json:"default_value,omitempty"`
	MaxLength     *int        `json:"max_length,omitempty"`
	Description   string      `json:"description,omitempty"`
	IsPrimaryKey  bool        `json:"is_primary_key"`
	IsUnique      bool        `json:"is_unique"`
	IsForeignKey  bool        `json:"is_foreign_key"`
	ForeignKeyRef *ForeignKey `json:"foreign_key_ref,omitempty"`
}

// Index 索引信息
type Index struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Type    string   `json:"type,omitempty"`
}

// Constraint 约束信息
type Constraint struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"` // CHECK, UNIQUE, PRIMARY KEY, FOREIGN KEY
	Columns    []string `json:"columns"`
	Condition  string   `json:"condition,omitempty"`
	Reference  string   `json:"reference,omitempty"`   // 外键引用表
	RefColumns []string `json:"ref_columns,omitempty"` // 外键引用字段
}

// ForeignKey 外键信息
type ForeignKey struct {
	Name             string   `json:"name"`
	Columns          []string `json:"columns"`
	ReferenceTable   string   `json:"reference_table"`
	ReferenceColumns []string `json:"reference_columns"`
	OnDelete         string   `json:"on_delete,omitempty"` // CASCADE, SET NULL, RESTRICT
	OnUpdate         string   `json:"on_update,omitempty"` // CASCADE, SET NULL, RESTRICT
}

// TableInfo 表信息
type TableInfo struct {
	Schema       *TableSchema `json:"schema"`
	RowCount     int64        `json:"row_count"`
	Size         int64        `json:"size"`
	LastModified time.Time    `json:"last_modified"`
}

// CreateTableRequest 创建表请求
type CreateTableRequest struct {
	Name        string             `json:"name"`
	DisplayName string             `json:"display_name,omitempty"`
	Description string             `json:"description,omitempty"`
	Columns     []CreateColumn     `json:"columns"`
	Indexes     []CreateIndex      `json:"indexes,omitempty"`
	Constraints []CreateConstraint `json:"constraints,omitempty"`
}

// CreateColumn 创建列请求
type CreateColumn struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Nullable     bool        `json:"nullable"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	MaxLength    *int        `json:"max_length,omitempty"`
	Description  string      `json:"description,omitempty"`
	IsUnique     bool        `json:"is_unique"`
	IsPrimaryKey bool        `json:"is_primary_key"`
}

// CreateIndex 创建索引请求
type CreateIndex struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Type    string   `json:"type,omitempty"`
}

// CreateConstraint 创建约束请求
type CreateConstraint struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Columns    []string `json:"columns"`
	Condition  string   `json:"condition,omitempty"`
	Reference  string   `json:"reference,omitempty"`
	RefColumns []string `json:"ref_columns,omitempty"`
	OnDelete   string   `json:"on_delete,omitempty"`
	OnUpdate   string   `json:"on_update,omitempty"`
}

// UpdateTableRequest 更新表请求
type UpdateTableRequest struct {
	DisplayName *string        `json:"display_name,omitempty"`
	Description *string        `json:"description,omitempty"`
	Columns     []UpdateColumn `json:"columns,omitempty"`
	AddColumns  []CreateColumn `json:"add_columns,omitempty"`
	DropColumns []string       `json:"drop_columns,omitempty"`
	AddIndexes  []CreateIndex  `json:"add_indexes,omitempty"`
	DropIndexes []string       `json:"drop_indexes,omitempty"`
}

// UpdateColumn 更新列请求
type UpdateColumn struct {
	Name         string      `json:"name"`
	NewName      *string     `json:"new_name,omitempty"`
	Type         *string     `json:"type,omitempty"`
	Nullable     *bool       `json:"nullable,omitempty"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Description  *string     `json:"description,omitempty"`
	IsUnique     *bool       `json:"is_unique,omitempty"`
}

// QueryRequest 查询请求
type QueryRequest struct {
	Table   string        `json:"table"`
	Method  RequestMethod `json:"method"`
	Options interface{}   `json:"options"`        // 根据方法类型使用不同的选项结构
	Data    interface{}   `json:"data,omitempty"` // 用于POST/PUT/PATCH的数据
}

// QueryResponse 查询响应
type QueryResponse struct {
	Data     interface{}            `json:"data"`
	Count    int64                  `json:"count,omitempty"`
	Limit    int                    `json:"limit,omitempty"`
	Offset   int                    `json:"offset,omitempty"`
	HasNext  bool                   `json:"has_next,omitempty"`
	Error    *QueryError            `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// QueryError 查询错误
type QueryError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// BatchOperation 批量操作
type BatchOperation struct {
	Operation string      `json:"operation"` // CREATE, UPDATE, DELETE
	Table     string      `json:"table"`
	Data      interface{} `json:"data"`
	Options   interface{} `json:"options,omitempty"`
}

// BatchRequest 批量请求
type BatchRequest struct {
	Operations    []BatchOperation `json:"operations"`
	Transactional bool             `json:"transactional"`
}

// BatchResponse 批量响应
type BatchResponse struct {
	Results []BatchResult `json:"results"`
	Errors  []BatchError  `json:"errors,omitempty"`
}

// BatchResult 批量操作结果
type BatchResult struct {
	Index   int         `json:"index"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Count   int64       `json:"count,omitempty"`
}

// BatchError 批量操作错误
type BatchError struct {
	Index   int    `json:"index"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// RealtimeSubscription 实时订阅
type RealtimeSubscription struct {
	ID        string                 `json:"id"`
	Table     string                 `json:"table"`
	Filter    map[string]interface{} `json:"filter,omitempty"`
	Events    []string               `json:"events"` // INSERT, UPDATE, DELETE, ALL
	Columns   []string               `json:"columns,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// RealtimeEvent 实时事件
type RealtimeEvent struct {
	Type      string                 `json:"type"` // INSERT, UPDATE, DELETE
	Table     string                 `json:"table"`
	Record    map[string]interface{} `json:"record"`
	OldRecord map[string]interface{} `json:"old_record,omitempty"` // UPDATE事件中的旧值
	Timestamp time.Time              `json:"timestamp"`
}

// DatabaseInfo 数据库信息
type DatabaseInfo struct {
	Version    string                 `json:"version"`
	Tables     []TableInfo            `json:"tables"`
	Size       int64                  `json:"size"`
	TableCount int                    `json:"table_count"`
	Schema     map[string]interface{} `json:"schema"`
}

// ExportRequest 导出请求
type ExportRequest struct {
	Tables  []string     `json:"tables,omitempty"`
	Format  string       `json:"format"` // JSON, CSV, SQL
	Options QueryOptions `json:"options,omitempty"`
}

// ImportRequest 导入请求
type ImportRequest struct {
	Table      string            `json:"table"`
	Format     string            `json:"format"` // JSON, CSV, SQL
	Data       interface{}       `json:"data"`
	Options    *InsertOptions    `json:"options,omitempty"`
	Mappings   map[string]string `json:"mappings,omitempty"` // 字段映射
	Validation bool              `json:"validation"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string                 `json:"status"`
	Version   string                 `json:"version"`
	Uptime    string                 `json:"uptime"`
	Database  DatabaseStatus         `json:"database"`
	Cache     CacheStatus            `json:"cache"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// DatabaseStatus 数据库状态
type DatabaseStatus struct {
	Connected bool   `json:"connected"`
	Latency   string `json:"latency"`
	Version   string `json:"version"`
}

// CacheStatus 缓存状态
type CacheStatus struct {
	Connected bool   `json:"connected"`
	Latency   string `json:"latency"`
	Type      string `json:"type"`
}

// APIKey 简化版API密钥类型（从keys包导入的简化版本）
type APIKey struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Scopes     []string               `json:"scopes"`
	TenantID   *string                `json:"tenant_id,omitempty"`
	ProjectID  *string                `json:"project_id,omitempty"`
	ExpiresAt  *time.Time             `json:"expires_at,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	LastUsedAt *time.Time             `json:"last_used_at,omitempty"`
	UsageCount int64                  `json:"usage_count"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// HasScope 检查API密钥是否具有指定权限
func (k *APIKey) HasScope(scope string) bool {
	for _, s := range k.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// IsExpired 检查API密钥是否已过期
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsValid 检查API密钥是否有效（激活且未过期）
func (k *APIKey) IsValid() bool {
	return !k.IsExpired()
}

// WriteError writes an error response
func WriteError(w http.ResponseWriter, statusCode int, message string, err error) {
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    statusCode,
			"message": message,
		},
		"success": false,
	}

	if err != nil {
		response["error"].(map[string]interface{})["details"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	response := map[string]interface{}{
		"success": true,
		"data":    data,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// ValidateStruct validates a struct and returns errors
func ValidateStruct(v interface{}) error {
	// TODO: Implement proper validation using a validation library
	// For now, just return nil to skip validation
	return nil
}

// GetClientIP gets the client IP address from request
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP if multiple
		if idx := len(xff); idx > 0 {
			if commaIdx := 0; commaIdx < idx {
				for i, c := range xff {
					if c == ',' {
						commaIdx = i
						break
					}
				}
				if commaIdx > 0 {
					return xff[:commaIdx]
				}
			}
			return xff
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// ExtractBearerToken extracts bearer token from Authorization header
func ExtractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return ""
	}

	return authHeader[7:]
}

// GetUserIDFromContext extracts user ID from request context
func GetUserIDFromContext(r *http.Request) string {
	if userID := r.Context().Value("userID"); userID != nil {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

// GetPathParam extracts path parameter from request using chi
func GetPathParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}
