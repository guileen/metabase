package rest

import (
	"fmt"
	"regexp"
	"strings"
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
	Where map[string]interface{} `json:"where,omitempty"`

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
	Table     string   `json:"table"`
	Type      string   `json:"type"` // INNER, LEFT, RIGHT, FULL
	Condition string   `json:"condition"`
	Select    []string `json:"select,omitempty"`
	Alias     string   `json:"alias,omitempty"`
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

// QueryBuilder SQL查询构建器
type QueryBuilder struct {
	table         string
	operation     OperationType
	options       *QueryOptions
	data          map[string]interface{}
	insertOptions *InsertOptions
	updateOptions *UpdateOptions
	deleteOptions *DeleteOptions
}

// NewQueryBuilder 创建查询构建器
func NewQueryBuilder(table string, operation OperationType, options *QueryOptions) *QueryBuilder {
	if options == nil {
		options = &QueryOptions{}
	}

	return &QueryBuilder{
		table:     table,
		operation: operation,
		options:   options,
		data:      make(map[string]interface{}),
	}
}

// SetData 设置数据（用于INSERT/UPDATE操作）
func (qb *QueryBuilder) SetData(data map[string]interface{}) *QueryBuilder {
	qb.data = data
	return qb
}

// SetInsertOptions 设置插入选项
func (qb *QueryBuilder) SetInsertOptions(options *InsertOptions) *QueryBuilder {
	qb.insertOptions = options
	return qb
}

// SetUpdateOptions 设置更新选项
func (qb *QueryBuilder) SetUpdateOptions(options *UpdateOptions) *QueryBuilder {
	qb.updateOptions = options
	return qb
}

// SetDeleteOptions 设置删除选项
func (qb *QueryBuilder) SetDeleteOptions(options *DeleteOptions) *QueryBuilder {
	qb.deleteOptions = options
	return qb
}

// Build 构建SQL查询
func (qb *QueryBuilder) Build() (string, []interface{}, error) {
	switch qb.operation {
	case OperationSelect:
		return qb.buildSelect()
	case OperationInsert:
		return qb.buildInsert()
	case OperationUpdate:
		return qb.buildUpdate()
	case OperationDelete:
		return qb.buildDelete()
	default:
		return "", nil, fmt.Errorf("unsupported operation: %s", qb.operation)
	}
}

// BuildCountQuery 构建计数查询
func (qb *QueryBuilder) BuildCountQuery() (string, []interface{}, error) {
	// 构建基础计数查询
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", qb.quoteIdentifier(qb.table))
	var args []interface{}

	// 添加WHERE条件
	if len(qb.options.Where) > 0 {
		whereClause, whereArgs := qb.buildWhereClause(qb.options.Where)
		if whereClause != "" {
			query += " WHERE " + whereClause
			args = append(args, whereArgs...)
		}
	}

	return query, args, nil
}

// ValidateQuery 验证查询
func (qb *QueryBuilder) ValidateQuery() error {
	// 验证表名
	if !isValidIdentifier(qb.table) {
		return fmt.Errorf("invalid table name: %s", qb.table)
	}

	// 验证字段选择
	for _, field := range qb.options.Select {
		if !isValidIdentifier(field) && field != "*" {
			return fmt.Errorf("invalid field name: %s", field)
		}
	}

	// 验证排序字段
	if qb.options.OrderBy != "" {
		parts := strings.Fields(qb.options.OrderBy)
		if len(parts) > 2 {
			return fmt.Errorf("invalid order by clause: %s", qb.options.OrderBy)
		}
		if len(parts) >= 1 && !isValidIdentifier(parts[0]) && parts[0] != "*" {
			return fmt.Errorf("invalid order by field: %s", parts[0])
		}
		if len(parts) == 2 && !isValidDirection(parts[1]) {
			return fmt.Errorf("invalid order direction: %s", parts[1])
		}
	}

	return nil
}

// buildSelect 构建SELECT查询
func (qb *QueryBuilder) buildSelect() (string, []interface{}, error) {
	var parts []string
	var args []interface{}

	// SELECT字段
	selectClause := "*"
	if len(qb.options.Select) > 0 {
		var fields []string
		for _, field := range qb.options.Select {
			fields = append(fields, qb.quoteIdentifier(field))
		}
		selectClause = strings.Join(fields, ", ")
	}
	parts = append(parts, fmt.Sprintf("SELECT %s FROM %s", selectClause, qb.quoteIdentifier(qb.table)))

	// WHERE条件
	if len(qb.options.Where) > 0 {
		whereClause, whereArgs := qb.buildWhereClause(qb.options.Where)
		if whereClause != "" {
			parts = append(parts, "WHERE "+whereClause)
			args = append(args, whereArgs...)
		}
	}

	// ORDER BY
	if qb.options.OrderBy != "" {
		parts = append(parts, "ORDER BY "+qb.options.OrderBy)
	}

	// LIMIT和OFFSET
	if qb.options.Limit > 0 {
		parts = append(parts, fmt.Sprintf("LIMIT %d", qb.options.Limit))
	}
	if qb.options.Offset > 0 {
		parts = append(parts, fmt.Sprintf("OFFSET %d", qb.options.Offset))
	}

	return strings.Join(parts, " "), args, nil
}

// buildInsert 构建INSERT查询
func (qb *QueryBuilder) buildInsert() (string, []interface{}, error) {
	if len(qb.data) == 0 {
		return "", nil, fmt.Errorf("no data provided for insert")
	}

	var columns []string
	var placeholders []string
	var args []interface{}

	for column, value := range qb.data {
		columns = append(columns, qb.quoteIdentifier(column))
		placeholders = append(placeholders, "?")
		args = append(args, value)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		qb.quoteIdentifier(qb.table),
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	// RETURNING子句
	if qb.insertOptions != nil && len(qb.insertOptions.Returning) > 0 {
		var returningFields []string
		for _, field := range qb.insertOptions.Returning {
			returningFields = append(returningFields, qb.quoteIdentifier(field))
		}
		query += " RETURNING " + strings.Join(returningFields, ", ")
	}

	return query, args, nil
}

// buildUpdate 构建UPDATE查询
func (qb *QueryBuilder) buildUpdate() (string, []interface{}, error) {
	if len(qb.data) == 0 {
		return "", nil, fmt.Errorf("no data provided for update")
	}

	var setParts []string
	var args []interface{}

	for column, value := range qb.data {
		setParts = append(setParts, fmt.Sprintf("%s = ?", qb.quoteIdentifier(column)))
		args = append(args, value)
	}

	query := fmt.Sprintf("UPDATE %s SET %s", qb.quoteIdentifier(qb.table), strings.Join(setParts, ", "))

	// WHERE条件
	if len(qb.options.Where) > 0 {
		whereClause, whereArgs := qb.buildWhereClause(qb.options.Where)
		if whereClause != "" {
			query += " WHERE " + whereClause
			args = append(args, whereArgs...)
		}
	}

	// RETURNING子句
	if qb.updateOptions != nil && len(qb.updateOptions.Returning) > 0 {
		var returningFields []string
		for _, field := range qb.updateOptions.Returning {
			returningFields = append(returningFields, qb.quoteIdentifier(field))
		}
		query += " RETURNING " + strings.Join(returningFields, ", ")
	}

	return query, args, nil
}

// buildDelete 构建DELETE查询
func (qb *QueryBuilder) buildDelete() (string, []interface{}, error) {
	var parts []string
	var args []interface{}

	parts = append(parts, fmt.Sprintf("DELETE FROM %s", qb.quoteIdentifier(qb.table)))

	// WHERE条件
	if len(qb.options.Where) > 0 {
		whereClause, whereArgs := qb.buildWhereClause(qb.options.Where)
		if whereClause != "" {
			parts = append(parts, "WHERE "+whereClause)
			args = append(args, whereArgs...)
		}
	}

	// RETURNING子句
	if qb.deleteOptions != nil && len(qb.deleteOptions.Returning) > 0 {
		var returningFields []string
		for _, field := range qb.deleteOptions.Returning {
			returningFields = append(returningFields, qb.quoteIdentifier(field))
		}
		parts = append(parts, "RETURNING "+strings.Join(returningFields, ", "))
	}

	return strings.Join(parts, " "), args, nil
}

// buildWhereClause 构建WHERE子句
func (qb *QueryBuilder) buildWhereClause(conditions map[string]interface{}) (string, []interface{}) {
	if len(conditions) == 0 {
		return "", nil
	}

	var clauses []string
	var args []interface{}

	for field, value := range conditions {
		switch v := value.(type) {
		case map[string]interface{}:
			// 处理复杂条件: {"field": {"gt": 100, "lt": 200}}
			for op, val := range v {
				clause, arg := qb.buildCondition(field, op, val)
				if clause != "" {
					clauses = append(clauses, clause)
					args = append(args, arg)
				}
			}
		case []interface{}:
			// 处理IN条件: {"field": [1,2,3]}
			if len(v) > 0 {
				placeholders := make([]string, len(v))
				for i, item := range v {
					placeholders[i] = "?"
					args = append(args, item)
				}
				clauses = append(clauses, fmt.Sprintf("%s IN (%s)", qb.quoteIdentifier(field), strings.Join(placeholders, ", ")))
			}
		default:
			// 处理等值条件: {"field": "value"}
			clauses = append(clauses, fmt.Sprintf("%s = ?", qb.quoteIdentifier(field)))
			args = append(args, value)
		}
	}

	return strings.Join(clauses, " AND "), args
}

// buildCondition 构建单个条件
func (qb *QueryBuilder) buildCondition(field, operator string, value interface{}) (string, interface{}) {
	quotedField := qb.quoteIdentifier(field)

	switch operator {
	case "eq", "=":
		return fmt.Sprintf("%s = ?", quotedField), value
	case "ne", "!=":
		return fmt.Sprintf("%s != ?", quotedField), value
	case "gt", ">":
		return fmt.Sprintf("%s > ?", quotedField), value
	case "gte", ">=":
		return fmt.Sprintf("%s >= ?", quotedField), value
	case "lt", "<":
		return fmt.Sprintf("%s < ?", quotedField), value
	case "lte", "<=":
		return fmt.Sprintf("%s <= ?", quotedField), value
	case "like":
		return fmt.Sprintf("%s LIKE ?", quotedField), value
	case "ilike":
		return fmt.Sprintf("%s ILIKE ?", quotedField), value
	case "is":
		if value == nil {
			return fmt.Sprintf("%s IS NULL", quotedField), nil
		}
		return fmt.Sprintf("%s IS NOT NULL", quotedField), nil
	case "in":
		if arr, ok := value.([]interface{}); ok && len(arr) > 0 {
			placeholders := make([]string, len(arr))
			for i := range placeholders {
				placeholders[i] = "?"
			}
			return fmt.Sprintf("%s IN (%s)", quotedField, strings.Join(placeholders, ", ")), arr
		}
		return "", nil
	default:
		return "", nil
	}
}

// quoteIdentifier 标识符引号
func (qb *QueryBuilder) quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// isValidIdentifier 检查是否为有效标识符
func isValidIdentifier(name string) bool {
	if name == "" {
		return false
	}
	// 只允许字母、数字、下划线，且不能以数字开头
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	return matched
}

// isValidDirection 检查排序方向
func isValidDirection(direction string) bool {
	direction = strings.ToUpper(direction)
	return direction == "ASC" || direction == "DESC"
}