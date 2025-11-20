package storage

import ("context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/guileen/metabase/pkg/common/errors")

// SecureStorage provides a secure wrapper around the storage engine
type SecureStorage struct {
	*Engine
}

// NewSecureStorage creates a new secure storage wrapper
func NewSecureStorage(config *Config) (*SecureStorage, error) {
	engine, err := NewEngine(config)
	if err != nil {
		return nil, err
	}

	return &SecureStorage{Engine: engine}, nil
}

// SecureQuery performs a parameterized query to prevent SQL injection
func (s *SecureStorage) SecureQuery(ctx context.Context, table string, options *QueryOptions) (*QueryResult, error) {
	// Validate table name to prevent SQL injection
	if err := s.validateTableName(table); err != nil {
		return nil, err
	}

	// Build parameterized query
	query, args, err := s.buildParameterizedQuery(table, options)
	if err != nil {
		return nil, err
	}

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute secure query: %w", err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var record Record
		var dataJSON string

		err := rows.Scan(
			&record.ID,
			&record.Table,
			&dataJSON,
			&record.CreatedAt,
			&record.UpdatedAt,
			&record.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}

		if err := record.UnmarshalData(dataJSON); err != nil {
			return nil, fmt.Errorf("failed to unmarshal record data: %w", err)
		}

		records = append(records, record)
	}

	// Get total count with parameterized query
	countQuery, countArgs, err := s.buildCountQuery(table, options)
	if err != nil {
		return nil, err
	}

	var total int
	err = s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	offset := 0
	if options != nil {
		offset = options.Offset
	}

	return &QueryResult{
		Records: records,
		Total:   total,
		HasNext: offset+len(records) < total,
		Meta: map[string]interface{}{
			"limit":  100,
			"offset": offset,
			"table":  table,
		},
	}, nil
}

// validateTableName prevents SQL injection in table names
func (s *SecureStorage) validateTableName(table string) *common.AppError {
	// Check against allowed tables
	allowedTables := map[string]bool{
		"users":    true,
		"posts":    true,
		"comments": true,
		"settings": true,
		"logs":     true,
		"metrics":  true,
	}

	if !allowedTables[table] {
		return common.NewInvalidInputError(fmt.Sprintf("Table '%s' is not allowed", table))
	}

	// Additional validation: only allow alphanumeric characters and underscores
	for _, char := range table {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_') {
			return common.NewInvalidInputError(fmt.Sprintf("Invalid table name: %s", table))
		}
	}

	if len(table) > 64 {
		return common.NewInvalidInputError("Table name too long (max 64 characters)")
	}

	return nil
}

// buildParameterizedQuery builds a secure parameterized query
func (s *SecureStorage) buildParameterizedQuery(table string, options *QueryOptions) (string, []interface{}, error) {
	if options == nil {
		options = &QueryOptions{
			Limit:    100,
			Offset:   0,
			OrderBy:  "created_at",
			OrderDir: "DESC",
		}
	}

	// Base query
	query := `SELECT id, table_name, data, created_at, updated_at, version FROM records WHERE table_name = ?`
	args := []interface{}{table}

	// Add WHERE conditions with parameters
	if options.Where != nil {
		conditions, whereArgs := s.buildWhereConditions(options.Where)
		if len(conditions) > 0 {
			query += " AND " + strings.Join(conditions, " AND ")
			args = append(args, whereArgs...)
		}
	}

	// Add ORDER BY with validation
	orderBy, orderDir := s.validateOrderBy(options.OrderBy, options.OrderDir)
	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDir)

	// Add LIMIT and OFFSET
	query += " LIMIT ? OFFSET ?"
	args = append(args, options.Limit, options.Offset)

	return query, args, nil
}

// buildWhereConditions builds parameterized WHERE conditions
func (s *SecureStorage) buildWhereConditions(where map[string]interface{}) ([]string, []interface{}) {
	var conditions []string
	var args []interface{}

	allowedFields := map[string]bool{
		"status":     true,
		"user_id":    true,
		"created_at": true,
		"updated_at": true,
		"type":       true,
	}

	for field, value := range where {
		// Validate field name
		if !allowedFields[field] {
			continue // Skip invalid fields
		}

		switch v := value.(type) {
		case string:
			conditions = append(conditions, fmt.Sprintf("JSON_EXTRACT(data, '$.%s') = ?", field))
			args = append(args, v)
		case []string:
			if len(v) > 0 {
				placeholders := make([]string, len(v))
				for i := range v {
					placeholders[i] = "?"
				}
				conditions = append(conditions, fmt.Sprintf("JSON_EXTRACT(data, '$.%s') IN (%s)", field, strings.Join(placeholders, ",")))
				for _, val := range v {
					args = append(args, val)
				}
			}
		case int, int32, int64, float32, float64:
			conditions = append(conditions, fmt.Sprintf("JSON_EXTRACT(data, '$.%s') = ?", field))
			args = append(args, v)
		default:
			// Skip unsupported types
			continue
		}
	}

	return conditions, args
}

// buildCountQuery builds a parameterized count query
func (s *SecureStorage) buildCountQuery(table string, options *QueryOptions) (string, []interface{}, error) {
	query := "SELECT COUNT(*) FROM records WHERE table_name = ?"
	args := []interface{}{table}

	if options != nil && options.Where != nil {
		conditions, whereArgs := s.buildWhereConditions(options.Where)
		if len(conditions) > 0 {
			query += " AND " + strings.Join(conditions, " AND ")
			args = append(args, whereArgs...)
		}
	}

	return query, args, nil
}

// validateOrderBy validates and sanitizes ORDER BY clauses
func (s *SecureStorage) validateOrderBy(orderBy, orderDir string) (string, string) {
	allowedFields := map[string]bool{
		"created_at": true,
		"updated_at": true,
		"id":         true,
		"version":    true,
	}

	// Default values
	if orderBy == "" {
		orderBy = "created_at"
	}
	if orderDir == "" {
		orderDir = "DESC"
	}

	// Validate field
	if !allowedFields[orderBy] {
		orderBy = "created_at"
	}

	// Validate direction
	orderDir = strings.ToUpper(orderDir)
	if orderDir != "ASC" && orderDir != "DESC" {
		orderDir = "DESC"
	}

	return orderBy, orderDir
}

// CreateWithValidation creates a record with validation
func (s *SecureStorage) CreateWithValidation(ctx context.Context, table string, data map[string]interface{}) (*Record, error) {
	// Validate table name
	if err := s.validateTableName(table); err != nil {
		return nil, err
	}

	// Validate data size
	if err := s.validateDataSize(data); err != nil {
		return nil, err
	}

	// Sanitize data
	sanitized := s.sanitizeData(data)

	// Call original create method
	return s.Create(ctx, table, sanitized)
}

// UpdateWithValidation updates a record with validation
func (s *SecureStorage) UpdateWithValidation(ctx context.Context, table, id string, data map[string]interface{}) (*Record, error) {
	// Validate inputs
	if err := s.validateTableName(table); err != nil {
		return nil, err
	}

	if err := s.validateRecordID(id); err != nil {
		return nil, err
	}

	if err := s.validateDataSize(data); err != nil {
		return nil, err
	}

	// Sanitize data
	sanitized := s.sanitizeData(data)

	// Call original update method
	return s.Update(ctx, table, id, sanitized)
}

// DeleteWithValidation deletes a record with validation
func (s *SecureStorage) DeleteWithValidation(ctx context.Context, table, id string) error {
	// Validate inputs
	if err := s.validateTableName(table); err != nil {
		return err
	}

	if err := s.validateRecordID(id); err != nil {
		return err
	}

	// Call original delete method
	return s.Delete(ctx, table, id)
}

// validateDataSize validates data size to prevent abuse
func (s *SecureStorage) validateDataSize(data map[string]interface{}) *common.AppError {
	maxDataSize := 1024 * 1024 // 1MB

	dataStr := fmt.Sprintf("%v", data)
	if len(dataStr) > maxDataSize {
		return common.NewInvalidInputError(fmt.Sprintf("Data too large. Max size: %d bytes", maxDataSize))
	}

	// Check number of fields
	if len(data) > 100 {
		return common.NewInvalidInputError("Too many fields (max 100)")
	}

	return nil
}

// validateRecordID validates record ID format
func (s *SecureStorage) validateRecordID(id string) *common.AppError {
	if id == "" {
		return common.NewInvalidInputError("Record ID cannot be empty")
	}

	if len(id) > 100 {
		return common.NewInvalidInputError("Record ID too long (max 100 characters)")
	}

	// Check for valid characters
	for _, char := range id {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' || char == '-') {
			return common.NewInvalidInputError("Invalid record ID format")
		}
	}

	return nil
}

// sanitizeData sanitizes input data
func (s *SecureStorage) sanitizeData(data map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})

	for key, value := range data {
		// Sanitize key
		if len(key) > 100 {
			continue // Skip long keys
		}

		// Basic key validation
		validKey := true
		for _, char := range key {
			if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_') {
				validKey = false
				break
			}
		}

		if !validKey {
			continue
		}

		// Sanitize value based on type
		switch v := value.(type) {
		case string:
			if len(v) > 1000 { // Limit string length
				v = v[:1000]
			}
			sanitized[key] = v
		case int, int32, int64, float32, float64, bool:
			sanitized[key] = v
		case []interface{}:
			if len(v) > 50 { // Limit array size
				v = v[:50]
			}
			sanitized[key] = v
		default:
			// Convert to string and limit length
			str := fmt.Sprintf("%v", v)
			if len(str) > 500 {
				str = str[:500]
			}
			sanitized[key] = str
		}
	}

	return sanitized
}

// UnmarshalData safely unmarshals record data
func (r *Record) UnmarshalData(dataJSON string) error {
	// Validate JSON size
	if len(dataJSON) > 10*1024*1024 { // 10MB limit
		return common.NewInvalidInputError("Record data too large")
	}

	// Use safe JSON parsing
	return json.Unmarshal([]byte(dataJSON), &r.Data)
}

// CreateTransaction creates a database transaction with timeout
func (s *SecureStorage) CreateTransaction(ctx context.Context, timeout time.Duration) (*sql.Tx, error) {
	// Create context with timeout
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return tx, nil
}