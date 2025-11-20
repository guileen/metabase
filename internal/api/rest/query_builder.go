package rest

import (
	"fmt"
	"regexp"
	"strings"
)

// QueryBuilder SQL查询构建器
type QueryBuilder struct {
	table     string
	operation OperationType
	options   *QueryOptions
	data      map[string]interface{}
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

// buildSelect 构建SELECT查询
func (qb *QueryBuilder) buildSelect() (string, []interface{}, error) {
	var query strings.Builder
	var args []interface{}
	argIndex := 1

	// SELECT子句
	query.WriteString("SELECT ")

	if len(qb.options.Select) > 0 {
		query.WriteString(strings.Join(qb.options.Select, ", "))
	} else {
		query.WriteString("*")
	}

	// FROM子句
	query.WriteString(" FROM ")
	query.WriteString(qb.quoteIdentifier(qb.table))

	// JOIN子句
	for _, join := range qb.options.Joins {
		query.WriteString(" ")
		query.WriteString(strings.ToUpper(join.Type))
		query.WriteString(" JOIN ")
		query.WriteString(qb.quoteIdentifier(join.Table))

		if join.Alias != "" {
			query.WriteString(" AS ")
			query.WriteString(qb.quoteIdentifier(join.Alias))
		}

		query.WriteString(" ON ")
		query.WriteString(join.Condition)
	}

	// WHERE子句
	if len(qb.options.Where) > 0 {
		whereClause, whereArgs := qb.buildWhereClause(qb.options.Where, &argIndex)
		query.WriteString(" WHERE ")
		query.WriteString(whereClause)
		args = append(args, whereArgs...)
	}

	// GROUP BY子句
	if len(qb.options.GroupBy) > 0 {
		query.WriteString(" GROUP BY ")
		groupBy := make([]string, len(qb.options.GroupBy))
		for i, col := range qb.options.GroupBy {
			groupBy[i] = qb.quoteIdentifier(col)
		}
		query.WriteString(strings.Join(groupBy, ", "))
	}

	// HAVING子句
	if len(qb.options.Having) > 0 {
		havingClause, havingArgs := qb.buildWhereClause(qb.options.Having, &argIndex)
		query.WriteString(" HAVING ")
		query.WriteString(havingClause)
		args = append(args, havingArgs...)
	}

	// ORDER BY子句
	if qb.options.OrderBy != "" {
		query.WriteString(" ORDER BY ")
		query.WriteString(qb.parseOrderBy(qb.options.OrderBy))
	}

	// LIMIT和OFFSET子句
	if qb.options.Limit > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT %d", qb.options.Limit))
	}

	if qb.options.Offset > 0 {
		query.WriteString(fmt.Sprintf(" OFFSET %d", qb.options.Offset))
	}

	return query.String(), args, nil
}

// buildInsert 构建INSERT查询
func (qb *QueryBuilder) buildInsert() (string, []interface{}, error) {
	if len(qb.data) == 0 {
		return "", nil, fmt.Errorf("no data provided for INSERT")
	}

	var query strings.Builder
	var args []interface{}
	argIndex := 1

	// INSERT子句
	query.WriteString("INSERT INTO ")
	query.WriteString(qb.quoteIdentifier(qb.table))
	query.WriteString(" (")

	columns := make([]string, 0, len(qb.data))
	placeholders := make([]string, 0, len(qb.data))

	for column := range qb.data {
		columns = append(columns, qb.quoteIdentifier(column))
		placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))
		argIndex++
	}

	query.WriteString(strings.Join(columns, ", "))
	query.WriteString(") VALUES (")
	query.WriteString(strings.Join(placeholders, ", "))
	query.WriteString(")")

	// 添加参数
	for _, value := range qb.data {
		args = append(args, value)
	}

	// RETURNING子句
	if len(qb.options.Returning) > 0 {
		query.WriteString(" RETURNING ")
		query.WriteString(strings.Join(qb.options.Returning, ", "))
	}

	return query.String(), args, nil
}

// buildUpdate 构建UPDATE查询
func (qb *QueryBuilder) buildUpdate() (string, []interface{}, error) {
	if len(qb.data) == 0 {
		return "", nil, fmt.Errorf("no data provided for UPDATE")
	}

	var query strings.Builder
	var args []interface{}
	argIndex := 1

	// UPDATE子句
	query.WriteString("UPDATE ")
	query.WriteString(qb.quoteIdentifier(qb.table))
	query.WriteString(" SET ")

	// SET子句
	setParts := make([]string, 0, len(qb.data))
	for column, value := range qb.data {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", qb.quoteIdentifier(column), argIndex))
		args = append(args, value)
		argIndex++
	}
	query.WriteString(strings.Join(setParts, ", "))

	// WHERE子句
	where := qb.options.Where
	if len(where) == 0 {
		where = map[string]interface{}{"id": qb.data["id"]} // 默认使用ID作为条件
	}

	whereClause, whereArgs := qb.buildWhereClause(where, &argIndex)
	query.WriteString(" WHERE ")
	query.WriteString(whereClause)
	args = append(args, whereArgs...)

	// RETURNING子句
	if len(qb.options.Returning) > 0 {
		query.WriteString(" RETURNING ")
		query.WriteString(strings.Join(qb.options.Returning, ", "))
	}

	return query.String(), args, nil
}

// buildDelete 构建DELETE查询
func (qb *QueryBuilder) buildDelete() (string, []interface{}, error) {
	var query strings.Builder
	var args []interface{}
	argIndex := 1

	// DELETE子句
	query.WriteString("DELETE FROM ")
	query.WriteString(qb.quoteIdentifier(qb.table))

	// WHERE子句
	where := qb.options.Where
	if len(where) == 0 {
		return "", nil, fmt.Errorf("DELETE operation requires WHERE clause for safety")
	}

	whereClause, whereArgs := qb.buildWhereClause(where, &argIndex)
	query.WriteString(" WHERE ")
	query.WriteString(whereClause)
	args = append(args, whereArgs...)

	// RETURNING子句
	if len(qb.options.Returning) > 0 {
		query.WriteString(" RETURNING ")
		query.WriteString(strings.Join(qb.options.Returning, ", "))
	}

	return query.String(), args, nil
}

// buildWhereClause 构建WHERE子句
func (qb *QueryBuilder) buildWhereClause(conditions map[string]interface{}, argIndex *int) (string, []interface{}) {
	if len(conditions) == 0 {
		return "", nil
	}

	var clauses []string
	var args []interface{}

	for field, value := range conditions {
		clause, clauseArgs := qb.buildCondition(field, value, argIndex)
		clauses = append(clauses, clause)
		args = append(args, clauseArgs...)
	}

	return strings.Join(clauses, " AND "), args
}

// buildCondition 构建单个条件
func (qb *QueryBuilder) buildCondition(field string, value interface{}, argIndex *int) (string, []interface{}) {
	switch v := value.(type) {
	case map[string]interface{}:
		// 复杂条件对象
		return qb.buildComplexCondition(field, v, argIndex)
	case []interface{}:
		// IN条件
		if len(v) == 0 {
			return "1=0", nil
		}
		placeholders := make([]string, len(v))
		args := make([]interface{}, len(v))
		for i, item := range v {
			placeholders[i] = fmt.Sprintf("$%d", *argIndex)
			args[i] = item
			*argIndex++
		}
		return fmt.Sprintf("%s IN (%s)", qb.quoteIdentifier(field), strings.Join(placeholders, ", ")), args
	default:
		// 简单等值条件
		return fmt.Sprintf("%s = $%d", qb.quoteIdentifier(field), *argIndex), []interface{}{v}
	}
}

// buildComplexCondition 构建复杂条件
func (qb *QueryBuilder) buildComplexCondition(field string, condition map[string]interface{}, argIndex *int) (string, []interface{}) {
	var args []interface{}

	// 解析操作符
	operator := "="
	if op, exists := condition["op"]; exists {
		if opStr, ok := op.(string); ok {
			operator = opStr
		}
	}

	value := condition["value"]
	column := qb.quoteIdentifier(field)

	switch strings.ToUpper(operator) {
	case ">", ">=", "<", "<=", "!=":
		return fmt.Sprintf("%s %s $%d", column, operator, *argIndex), []interface{}{value}
	case "LIKE":
		return fmt.Sprintf("%s LIKE $%d", column, *argIndex), []interface{}{value}
	case "ILIKE":
		return fmt.Sprintf("%s ILIKE $%d", column, *argIndex), []interface{}{value}
	case "IS NULL":
		return fmt.Sprintf("%s IS NULL", column), nil
	case "IS NOT NULL":
		return fmt.Sprintf("%s IS NOT NULL", column), nil
	case "IN":
		if values, ok := value.([]interface{}); ok {
			if len(values) == 0 {
				return "1=0", nil
			}
			placeholders := make([]string, len(values))
			args := make([]interface{}, len(values))
			for i, item := range values {
				placeholders[i] = fmt.Sprintf("$%d", *argIndex)
				args[i] = item
				*argIndex++
			}
			return fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ", ")), args
		}
		return "1=0", nil
	case "NOT IN":
		if values, ok := value.([]interface{}); ok {
			if len(values) == 0 {
				return "1=1", nil
			}
			placeholders := make([]string, len(values))
			args := make([]interface{}, len(values))
			for i, item := range values {
				placeholders[i] = fmt.Sprintf("$%d", *argIndex)
				args[i] = item
				*argIndex++
			}
			return fmt.Sprintf("%s NOT IN (%s)", column, strings.Join(placeholders, ", ")), args
		}
		return "1=1", nil
	case "BETWEEN":
		if rangeValues, ok := value.([]interface{}); ok && len(rangeValues) == 2 {
			*argIndex++
			*argIndex++
			return fmt.Sprintf("%s BETWEEN $%d AND $%d", column, *argIndex-2, *argIndex-1), []interface{}{rangeValues[0], rangeValues[1]}
		}
		return "1=0", nil
	default:
		return fmt.Sprintf("%s = $%d", column, *argIndex), []interface{}{value}
	}
}

// parseOrderBy 解析ORDER BY子句
func (qb *QueryBuilder) parseOrderBy(orderBy string) string {
	parts := strings.Split(orderBy, ",")
	var result []string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// 检查是否有DESC或ASC后缀
		field := part
		direction := "ASC"

		if strings.HasSuffix(strings.ToUpper(part), " DESC") {
			field = strings.TrimSpace(strings.TrimSuffix(part, " DESC"))
			direction = "DESC"
		} else if strings.HasSuffix(strings.ToUpper(part), " ASC") {
			field = strings.TrimSpace(strings.TrimSuffix(part, " ASC"))
		}

		result = append(result, fmt.Sprintf("%s %s", qb.quoteIdentifier(field), direction))
	}

	return strings.Join(result, ", ")
}

// quoteIdentifier 引用标识符，加强安全性
func (qb *QueryBuilder) quoteIdentifier(name string) string {
	// 清理和验证标识符
	if err := qb.validateIdentifier(name); err != nil {
		// 如果标识符无效，返回空字符串而不是直接使用
		return ""
	}

	// 转义标识符中的双引号
	escaped := strings.ReplaceAll(name, `"`, `""`)
	return `"` + escaped + `"`
}

// BuildCountQuery 构建COUNT查询
func (qb *QueryBuilder) BuildCountQuery() (string, []interface{}, error) {
	var query strings.Builder
	var args []interface{}
	argIndex := 1

	query.WriteString("SELECT COUNT(*) as count FROM ")
	query.WriteString(qb.quoteIdentifier(qb.table))

	// JOIN子句（计数通常不需要复杂的JOIN，但为了完整性保留）
	for _, join := range qb.options.Joins {
		query.WriteString(" ")
		query.WriteString(strings.ToUpper(join.Type))
		query.WriteString(" JOIN ")
		query.WriteString(qb.quoteIdentifier(join.Table))
		if join.Alias != "" {
			query.WriteString(" AS ")
			query.WriteString(qb.quoteIdentifier(join.Alias))
		}
		query.WriteString(" ON ")
		query.WriteString(join.Condition)
	}

	// WHERE子句
	if len(qb.options.Where) > 0 {
		whereClause, whereArgs := qb.buildWhereClause(qb.options.Where, &argIndex)
		query.WriteString(" WHERE ")
		query.WriteString(whereClause)
		args = append(args, whereArgs...)
	}

	// GROUP BY子句（影响计数逻辑）
	if len(qb.options.GroupBy) > 0 {
		// 如果有GROUP BY，返回分组后的数量
		query.WriteString(" GROUP BY ")
		groupBy := make([]string, len(qb.options.GroupBy))
		for i, col := range qb.options.GroupBy {
			groupBy[i] = qb.quoteIdentifier(col)
		}
		query.WriteString(strings.Join(groupBy, ", "))
	}

	return query.String(), args, nil
}

// ValidateQuery 验证查询参数
func (qb *QueryBuilder) ValidateQuery() error {
	// 验证表名
	if qb.table == "" {
		return fmt.Errorf("table name is required")
	}

	// 验证字段名
	for _, field := range qb.options.Select {
		if err := qb.validateIdentifier(field); err != nil {
			return fmt.Errorf("invalid select field '%s': %w", field, err)
		}
	}

	// 验证ORDER BY字段
	if qb.options.OrderBy != "" {
		parts := strings.Split(qb.options.OrderBy, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			field := part
			if strings.HasSuffix(strings.ToUpper(part), " DESC") {
				field = strings.TrimSpace(strings.TrimSuffix(part, " DESC"))
			} else if strings.HasSuffix(strings.ToUpper(part), " ASC") {
				field = strings.TrimSpace(strings.TrimSuffix(part, " ASC"))
			}

			if err := qb.validateIdentifier(field); err != nil {
				return fmt.Errorf("invalid order field '%s': %w", field, err)
			}
		}
	}

	// 验证GROUP BY字段
	for _, field := range qb.options.GroupBy {
		if err := qb.validateIdentifier(field); err != nil {
			return fmt.Errorf("invalid group field '%s': %w", field, err)
		}
	}

	return nil
}

// validateIdentifier 验证标识符，加强安全性检查
func (qb *QueryBuilder) validateIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("empty identifier")
	}

	// 检查长度限制
	if len(name) > 64 {
		return fmt.Errorf("identifier too long")
	}

	// 检查是否包含危险字符
	dangerousChars := []string{"'", ";", "--", "/*", "*/", "xp_", "sp_", "0x", "0X"}
	for _, char := range dangerousChars {
		if strings.Contains(strings.ToLower(name), char) {
			return fmt.Errorf("potentially dangerous identifier contains: %s", char)
		}
	}

	// 检查是否是有效的标识符格式（字母开头，只能包含字母、数字、下划线）
	matched, _ := regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9_]*$`, name)
	if !matched {
		return fmt.Errorf("invalid identifier format")
	}

	// 检查是否是SQL关键字
	keywords := []string{
		"SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "DELETE", "DROP", "CREATE",
		"ALTER", "TRUNCATE", "UNION", "JOIN", "INNER", "LEFT", "RIGHT", "OUTER",
		"GROUP", "ORDER", "BY", "HAVING", "LIMIT", "OFFSET", "AND", "OR", "NOT",
		"NULL", "TRUE", "FALSE", "CASE", "WHEN", "THEN", "ELSE", "END", "IF",
		"EXISTS", "IN", "BETWEEN", "LIKE", "ILIKE", "IS", "DISTINCT", "ALL",
		"ANY", "SOME", "CAST", "AS", "ON", "USING", "INDEX", "TABLE", "VIEW",
		"DATABASE", "SCHEMA", "FUNCTION", "PROCEDURE", "TRIGGER", "CONSTRAINT",
		"PRIMARY", "FOREIGN", "KEY", "REFERENCES", "CHECK", "UNIQUE", "DEFAULT",
	}

	upperName := strings.ToUpper(name)
	for _, keyword := range keywords {
		if upperName == keyword {
			return fmt.Errorf("identifier cannot be a SQL keyword: %s", name)
		}
	}

	return nil
}