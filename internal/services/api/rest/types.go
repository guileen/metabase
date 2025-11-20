package rest

import ("time")

// RequestMethod HTTP请求方法
type RequestMethod string

const (
	MethodGET    RequestMethod = "GET"
	MethodPOST   RequestMethod = "POST"
	MethodPUT    RequestMethod = "PUT"
	MethodPATCH  RequestMethod = "PATCH"
	MethodDELETE RequestMethod = "DELETE"
)

// OperationType 数据库操作类型
type OperationType string

const (
	OperationSelect OperationType = "select"
	OperationInsert OperationType = "insert"
	OperationUpdate OperationType = "update"
	OperationDelete OperationType = "delete"
)

// QueryOptions 查询选项
type QueryOptions struct {
	// 字段选择
	Select []string `json:"select,omitempty"`

	// 过滤条件
	Where  map[string]interface{} `json:"where,omitempty"`

	// 排序
	OrderBy string `json:"order,omitempty"`

	// 分页
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`

	// 关联查询
	Joins []JoinOption `json:"joins,omitempty"`

	// 聚合
	Aggregates []AggregateOption `json:"aggregates,omitempty"`

	// 分组
	GroupBy []string `json:"group_by,omitempty"`

	// Having条件
	Having map[string]interface{} `json:"having,omitempty"`
}

// JoinOption 关联查询选项
type JoinOption struct {
	Table     string            `json:"table"`
	Type      string            `json:"type"` // INNER, LEFT, RIGHT, FULL
	Condition string            `json:"condition"`
	Select    []string          `json:"select,omitempty"`
	Alias     string            `json:"alias,omitempty"`
}

// AggregateOption 聚合选项
type AggregateOption struct {
	Function string `json:"function"` // COUNT, SUM, AVG, MAX, MIN
	Field    string `json:"field"`
	Alias    string `json:"alias,omitempty"`
}

// InsertOptions 插入选项
type InsertOptions struct {
	// 返回字段
	Returning []string `json:"returning,omitempty"`

	// 冲突处理
	OnConflict *ConflictOption `json:"on_conflict,omitempty"`

	// 批量插入
	BatchSize int `json:"batch_size,omitempty"`
}

// ConflictOption 冲突处理选项
type ConflictOption struct {
	Action  string   `json:"action"` // IGNORE, UPDATE, MERGE
	Target  []string `json:"target"`  // 冲突检测字段
	Update  []string `json:"update,omitempty"` // 要更新的字段
}

// UpdateOptions 更新选项
type UpdateOptions struct {
	// 返回字段
	Returning []string `json:"returning,omitempty"`

	// 条件
	Where map[string]interface{} `json:"where,omitempty"`
}

// DeleteOptions 删除选项
type DeleteOptions struct {
	// 返回字段
	Returning []string `json:"returning,omitempty"`

	// 条件
	Where map[string]interface{} `json:"where,omitempty"`
}

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
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Nullable     bool        `json:"nullable"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	MaxLength    *int        `json:"max_length,omitempty"`
	Description  string      `json:"description,omitempty"`
	IsPrimaryKey bool        `json:"is_primary_key"`
	IsUnique     bool        `json:"is_unique"`
	IsForeignKey bool        `json:"is_foreign_key"`
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
	Name        string   `json:"name"`
	Type        string   `json:"type"` // CHECK, UNIQUE, PRIMARY KEY, FOREIGN KEY
	Columns     []string `json:"columns"`
	Condition   string   `json:"condition,omitempty"`
	Reference   string   `json:"reference,omitempty"` // 外键引用表
	RefColumns  []string `json:"ref_columns,omitempty"` // 外键引用字段
}

// ForeignKey 外键信息
type ForeignKey struct {
	Name          string   `json:"name"`
	Columns       []string `json:"columns"`
	ReferenceTable string  `json:"reference_table"`
	ReferenceColumns []string `json:"reference_columns"`
	OnDelete      string   `json:"on_delete,omitempty"` // CASCADE, SET NULL, RESTRICT
	OnUpdate      string   `json:"on_update,omitempty"` // CASCADE, SET NULL, RESTRICT
}

// TableInfo 表信息
type TableInfo struct {
	Schema    *TableSchema `json:"schema"`
	RowCount  int64        `json:"row_count"`
	Size      int64        `json:"size"`
	LastModified time.Time `json:"last_modified"`
}

// CreateTableRequest 创建表请求
type CreateTableRequest struct {
	Name        string          `json:"name"`
	DisplayName string          `json:"display_name,omitempty"`
	Description string          `json:"description,omitempty"`
	Columns     []CreateColumn  `json:"columns"`
	Indexes     []CreateIndex   `json:"indexes,omitempty"`
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
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Columns     []string `json:"columns"`
	Condition   string   `json:"condition,omitempty"`
	Reference   string   `json:"reference,omitempty"`
	RefColumns  []string `json:"ref_columns,omitempty"`
	OnDelete    string   `json:"on_delete,omitempty"`
	OnUpdate    string   `json:"on_update,omitempty"`
}

// UpdateTableRequest 更新表请求
type UpdateTableRequest struct {
	DisplayName *string       `json:"display_name,omitempty"`
	Description *string       `json:"description,omitempty"`
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
	Table   string       `json:"table"`
	Method  RequestMethod `json:"method"`
	Options interface{}   `json:"options"` // 根据方法类型使用不同的选项结构
	Data    interface{}   `json:"data,omitempty"` // 用于POST/PUT/PATCH的数据
}

// QueryResponse 查询响应
type QueryResponse struct {
	Data     interface{} `json:"data"`
	Count    int64       `json:"count,omitempty"`
	Limit    int         `json:"limit,omitempty"`
	Offset   int         `json:"offset,omitempty"`
	HasNext  bool        `json:"has_next,omitempty"`
	Error    *QueryError `json:"error,omitempty"`
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
	Operations []BatchOperation `json:"operations"`
	Transactional bool          `json:"transactional"`
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
	ID         string                 `json:"id"`
	Table      string                 `json:"table"`
	Filter     map[string]interface{} `json:"filter,omitempty"`
	Events     []string               `json:"events"` // INSERT, UPDATE, DELETE, ALL
	Columns    []string               `json:"columns,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// RealtimeEvent 实时事件
type RealtimeEvent struct {
	Type      string                 `json:"type"`      // INSERT, UPDATE, DELETE
	Table     string                 `json:"table"`
	Record    map[string]interface{} `json:"record"`
	OldRecord map[string]interface{} `json:"old_record,omitempty"` // UPDATE事件中的旧值
	Timestamp time.Time              `json:"timestamp"`
}

// DatabaseInfo 数据库信息
type DatabaseInfo struct {
	Version   string      `json:"version"`
	Tables    []TableInfo `json:"tables"`
	Size      int64       `json:"size"`
	TableCount int       `json:"table_count"`
	Schema    map[string]interface{} `json:"schema"`
}

// ExportRequest 导出请求
type ExportRequest struct {
	Tables  []string     `json:"tables,omitempty"`
	Format  string       `json:"format"` // JSON, CSV, SQL
	Options QueryOptions `json:"options,omitempty"`
}

// ImportRequest 导入请求
type ImportRequest struct {
	Table      string                 `json:"table"`
	Format     string                 `json:"format"` // JSON, CSV, SQL
	Data       interface{}            `json:"data"`
	Options    *InsertOptions         `json:"options,omitempty"`
	Mappings   map[string]string      `json:"mappings,omitempty"` // 字段映射
	Validation bool                   `json:"validation"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Uptime    string            `json:"uptime"`
	Database  DatabaseStatus    `json:"database"`
	Cache     CacheStatus       `json:"cache"`
	Timestamp time.Time         `json:"timestamp"`
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