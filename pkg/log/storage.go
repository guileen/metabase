package log

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// LogStorage handles log persistence and querying
type LogStorage struct {
	db        *sql.DB
	mu        sync.RWMutex
	maxRows   int64
	retention time.Duration
}

// StoredLogEntry represents a log entry stored in the database
type StoredLogEntry struct {
	ID         int64     `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	Level      string    `json:"level"`
	Message    string    `json:"message"`
	RequestID  string    `json:"request_id,omitempty"`
	UserID     string    `json:"user_id,omitempty"`
	Component  string    `json:"component,omitempty"`
	Service    string    `json:"service,omitempty"`
	Method     string    `json:"method,omitempty"`
	Path       string    `json:"path,omitempty"`
	Status     int       `json:"status,omitempty"`
	DurationMs int64     `json:"duration_ms,omitempty"`
	RemoteAddr string    `json:"remote_addr,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
	TraceID    string    `json:"trace_id,omitempty"`
	SpanID     string    `json:"span_id,omitempty"`
	Fields     string    `json:"fields,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// LogQuery represents query parameters for log search
type LogQuery struct {
	StartTime  *time.Time `json:"start_time,omitempty"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	Levels     []string   `json:"levels,omitempty"`
	Components []string   `json:"components,omitempty"`
	Services   []string   `json:"services,omitempty"`
	UserIDs    []string   `json:"user_ids,omitempty"`
	RequestIDs []string   `json:"request_ids,omitempty"`
	Search     string     `json:"search,omitempty"`
	Method     string     `json:"method,omitempty"`
	Path       string     `json:"path,omitempty"`
	MinStatus  int        `json:"min_status,omitempty"`
	MaxStatus  int        `json:"max_status,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
	OrderBy    string     `json:"order_by,omitempty"`  // "timestamp", "level", "duration"
	OrderDir   string     `json:"order_dir,omitempty"` // "asc", "desc"
}

// LogStats represents log statistics
type LogStats struct {
	TotalLogs       int64            `json:"total_logs"`
	LogsByLevel     map[string]int64 `json:"logs_by_level"`
	LogsByService   map[string]int64 `json:"logs_by_service"`
	LogsByComponent map[string]int64 `json:"logs_by_component"`
	ErrorRate       float64          `json:"error_rate"`
	AvgResponseTime float64          `json:"avg_response_time"`
	P99ResponseTime float64          `json:"p99_response_time"`
	TopPaths        []PathStat       `json:"top_paths"`
	TopErrors       []ErrorStat      `json:"top_errors"`
}

// PathStat represents statistics for a specific path
type PathStat struct {
	Path         string  `json:"path"`
	RequestCount int64   `json:"request_count"`
	ErrorCount   int64   `json:"error_count"`
	AvgTime      float64 `json:"avg_time"`
}

// ErrorStat represents statistics for specific errors
type ErrorStat struct {
	Error       string    `json:"error"`
	Count       int64     `json:"count"`
	LastOccured time.Time `json:"last_occured"`
}

// NewLogStorage creates a new log storage instance
func NewLogStorage(dbPath string, maxRows int64, retention time.Duration) (*LogStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &LogStorage{
		db:        db,
		maxRows:   maxRows,
		retention: retention,
	}

	if err := storage.initDB(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Start cleanup routine
	go storage.cleanupRoutine()

	return storage, nil
}

// initDB creates the necessary tables
func (s *LogStorage) initDB() error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		level VARCHAR(20) NOT NULL,
		message TEXT NOT NULL,
		request_id VARCHAR(64),
		user_id VARCHAR(64),
		component VARCHAR(64),
		service VARCHAR(64),
		method VARCHAR(10),
		path VARCHAR(500),
		status INTEGER,
		duration_ms INTEGER,
		remote_addr VARCHAR(100),
		user_agent TEXT,
		trace_id VARCHAR(64),
		span_id VARCHAR(64),
		fields TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level);
	CREATE INDEX IF NOT EXISTS idx_logs_service ON logs(service);
	CREATE INDEX IF NOT EXISTS idx_logs_component ON logs(component);
	CREATE INDEX IF NOT EXISTS idx_logs_request_id ON logs(request_id);
	CREATE INDEX IF NOT EXISTS idx_logs_user_id ON logs(user_id);
	CREATE INDEX IF NOT EXISTS idx_logs_path ON logs(path);
	CREATE INDEX IF NOT EXISTS idx_logs_status ON logs(status);
	`

	_, err := s.db.Exec(createTableSQL)
	return err
}

// StoreLog stores a log entry in the database
func (s *LogStorage) StoreLog(entry *StoredLogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	insertSQL := `
	INSERT INTO logs (
		timestamp, level, message, request_id, user_id, component, service,
		method, path, status, duration_ms, remote_addr, user_agent,
		trace_id, span_id, fields
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(insertSQL,
		entry.Timestamp, entry.Level, entry.Message, entry.RequestID,
		entry.UserID, entry.Component, entry.Service, entry.Method,
		entry.Path, entry.Status, entry.DurationMs, entry.RemoteAddr,
		entry.UserAgent, entry.TraceID, entry.SpanID, entry.Fields,
	)

	return err
}

// QueryLogs retrieves logs based on query parameters
func (s *LogStorage) QueryLogs(ctx context.Context, query *LogQuery) ([]*StoredLogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	whereConditions := []string{}
	args := []interface{}{}
	argIndex := 1

	// Build WHERE clause
	if query.StartTime != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, *query.StartTime)
		argIndex++
	}

	if query.EndTime != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, *query.EndTime)
		argIndex++
	}

	if len(query.Levels) > 0 {
		placeholders := make([]string, len(query.Levels))
		for i, level := range query.Levels {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, level)
			argIndex++
		}
		whereConditions = append(whereConditions, fmt.Sprintf("level IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(query.Services) > 0 {
		placeholders := make([]string, len(query.Services))
		for i, service := range query.Services {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, service)
			argIndex++
		}
		whereConditions = append(whereConditions, fmt.Sprintf("service IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(query.Components) > 0 {
		placeholders := make([]string, len(query.Components))
		for i, component := range query.Components {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, component)
			argIndex++
		}
		whereConditions = append(whereConditions, fmt.Sprintf("component IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(query.UserIDs) > 0 {
		placeholders := make([]string, len(query.UserIDs))
		for i, userID := range query.UserIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, userID)
			argIndex++
		}
		whereConditions = append(whereConditions, fmt.Sprintf("user_id IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(query.RequestIDs) > 0 {
		placeholders := make([]string, len(query.RequestIDs))
		for i, requestID := range query.RequestIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, requestID)
			argIndex++
		}
		whereConditions = append(whereConditions, fmt.Sprintf("request_id IN (%s)", strings.Join(placeholders, ",")))
	}

	if query.Search != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("(message LIKE $%d OR path LIKE $%d)", argIndex, argIndex+1))
		args = append(args, "%"+query.Search+"%", "%"+query.Search+"%")
		argIndex += 2
	}

	if query.Method != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("method = $%d", argIndex))
		args = append(args, query.Method)
		argIndex++
	}

	if query.Path != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("path LIKE $%d", argIndex))
		args = append(args, "%"+query.Path+"%")
		argIndex++
	}

	if query.MinStatus > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("status >= $%d", argIndex))
		args = append(args, query.MinStatus)
		argIndex++
	}

	if query.MaxStatus > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("status <= $%d", argIndex))
		args = append(args, query.MaxStatus)
		argIndex++
	}

	// Build the full query
	baseSQL := "SELECT id, timestamp, level, message, request_id, user_id, component, service, " +
		"method, path, status, duration_ms, remote_addr, user_agent, trace_id, span_id, fields, created_at " +
		"FROM logs"

	if len(whereConditions) > 0 {
		baseSQL += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Add ORDER BY
	orderBy := query.OrderBy
	if orderBy == "" {
		orderBy = "timestamp"
	}
	orderDir := strings.ToUpper(query.OrderDir)
	if orderDir != "ASC" && orderDir != "DESC" {
		orderDir = "DESC"
	}
	baseSQL += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDir)

	// Add LIMIT and OFFSET
	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	baseSQL += fmt.Sprintf(" LIMIT %d", limit)

	if query.Offset > 0 {
		baseSQL += fmt.Sprintf(" OFFSET %d", query.Offset)
	}

	rows, err := s.db.QueryContext(ctx, baseSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []*StoredLogEntry
	for rows.Next() {
		log := &StoredLogEntry{}
		err := rows.Scan(
			&log.ID, &log.Timestamp, &log.Level, &log.Message,
			&log.RequestID, &log.UserID, &log.Component, &log.Service,
			&log.Method, &log.Path, &log.Status, &log.DurationMs,
			&log.RemoteAddr, &log.UserAgent, &log.TraceID, &log.SpanID,
			&log.Fields, &log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log row: %w", err)
		}
		logs = append(logs, log)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating log rows: %w", err)
	}

	return logs, nil
}

// GetLogStats retrieves log statistics
func (s *LogStorage) GetLogStats(ctx context.Context, timeRange *time.Time) (*LogStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &LogStats{
		LogsByLevel:     make(map[string]int64),
		LogsByService:   make(map[string]int64),
		LogsByComponent: make(map[string]int64),
		TopPaths:        []PathStat{},
		TopErrors:       []ErrorStat{},
	}

	whereClause := ""
	args := []interface{}{}
	if timeRange != nil {
		whereClause = "WHERE timestamp >= ?"
		args = append(args, *timeRange)
	}

	// Get total logs
	var totalLogs int64
	err := s.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM logs %s", whereClause), args...).Scan(&totalLogs)
	if err != nil {
		return nil, fmt.Errorf("failed to get total logs: %w", err)
	}
	stats.TotalLogs = totalLogs

	// Get logs by level
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("SELECT level, COUNT(*) FROM logs %s GROUP BY level", whereClause), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs by level: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var level string
		var count int64
		if err := rows.Scan(&level, &count); err != nil {
			continue
		}
		stats.LogsByLevel[level] = count
	}

	// Get logs by service
	rows, err = s.db.QueryContext(ctx, fmt.Sprintf("SELECT service, COUNT(*) FROM logs %s GROUP BY service", whereClause), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs by service: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var service string
		var count int64
		if err := rows.Scan(&service, &count); err != nil {
			continue
		}
		if service != "" {
			stats.LogsByService[service] = count
		}
	}

	// Get logs by component
	rows, err = s.db.QueryContext(ctx, fmt.Sprintf("SELECT component, COUNT(*) FROM logs %s GROUP BY component", whereClause), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs by component: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var component string
		var count int64
		if err := rows.Scan(&component, &count); err != nil {
			continue
		}
		if component != "" {
			stats.LogsByComponent[component] = count
		}
	}

	// Calculate error rate
	errorCount := stats.LogsByLevel["ERROR"] + stats.LogsByLevel["FATAL"]
	if totalLogs > 0 {
		stats.ErrorRate = float64(errorCount) / float64(totalLogs) * 100
	}

	// Get average response time (only for successful requests)
	var avgResponseTime sql.NullFloat64
	err = s.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT AVG(duration_ms) FROM logs %s AND status < 400 AND duration_ms > 0", whereClause),
		args...).Scan(&avgResponseTime)
	if err == nil && avgResponseTime.Valid {
		stats.AvgResponseTime = avgResponseTime.Float64
	}

	// Get 99th percentile response time (only for successful requests)
	var p99ResponseTime sql.NullFloat64
	err = s.db.QueryRowContext(ctx,
		fmt.Sprintf(`
			SELECT CASE
				WHEN COUNT(*) * 0.99 >= COUNT(*) THEN (
					SELECT MAX(duration_ms) FROM logs %s AND status < 400 AND duration_ms > 0
				)
				ELSE (
					SELECT duration_ms FROM logs %s AND status < 400 AND duration_ms > 0
					ORDER BY duration_ms ASC
					LIMIT 1 OFFSET CAST(COUNT(*) * 0.99 AS INTEGER) - 1
				)
			END as p99
			FROM logs %s AND status < 400 AND duration_ms > 0
		`, whereClause, whereClause, whereClause),
		args...).Scan(&p99ResponseTime)
	if err == nil && p99ResponseTime.Valid {
		stats.P99ResponseTime = p99ResponseTime.Float64
	}

	// Get top paths
	rows, err = s.db.QueryContext(ctx,
		fmt.Sprintf(`
			SELECT path, COUNT(*) as req_count,
				   SUM(CASE WHEN status >= 400 THEN 1 ELSE 0 END) as error_count,
				   AVG(CASE WHEN duration_ms > 0 THEN duration_ms END) as avg_time
			FROM logs %s
			WHERE path != '' AND path IS NOT NULL
			GROUP BY path
			ORDER BY req_count DESC
			LIMIT 10
		`, whereClause), args...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var path string
			var reqCount, errorCount int64
			var avgTime sql.NullFloat64
			if err := rows.Scan(&path, &reqCount, &errorCount, &avgTime); err != nil {
				continue
			}
			stat := PathStat{
				Path:         path,
				RequestCount: reqCount,
				ErrorCount:   errorCount,
			}
			if avgTime.Valid {
				stat.AvgTime = avgTime.Float64
			}
			stats.TopPaths = append(stats.TopPaths, stat)
		}
	}

	// Get top errors
	rows, err = s.db.QueryContext(ctx,
		fmt.Sprintf(`
			SELECT message, COUNT(*) as count, MAX(timestamp) as last_occurred
			FROM logs %s
			WHERE level IN ('ERROR', 'FATAL')
			GROUP BY message
			ORDER BY count DESC
			LIMIT 10
		`, whereClause), args...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var message string
			var count int64
			var lastOccurred time.Time
			if err := rows.Scan(&message, &count, &lastOccurred); err != nil {
				continue
			}
			stat := ErrorStat{
				Error:       message,
				Count:       count,
				LastOccured: lastOccurred,
			}
			stats.TopErrors = append(stats.TopErrors, stat)
		}
	}

	return stats, nil
}

// cleanupRoutine removes old logs based on retention policy
func (s *LogStorage) cleanupRoutine() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanup()
	}
}

func (s *LogStorage) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Delete logs older than retention period
	if s.retention > 0 {
		cutoff := time.Now().Add(-s.retention)
		_, err := s.db.Exec("DELETE FROM logs WHERE timestamp < ?", cutoff)
		if err != nil {
			// Log error but don't fail
			fmt.Printf("Failed to cleanup old logs: %v\n", err)
		}
	}

	// If we have too many rows, delete the oldest ones
	if s.maxRows > 0 {
		var count int64
		err := s.db.QueryRow("SELECT COUNT(*) FROM logs").Scan(&count)
		if err == nil && count > s.maxRows {
			// Delete rows beyond the limit, keeping the most recent ones
			deleteSQL := `
				DELETE FROM logs WHERE id IN (
					SELECT id FROM logs ORDER BY timestamp ASC LIMIT ?
				)
			`
			_, err = s.db.Exec(deleteSQL, count-s.maxRows)
			if err != nil {
				fmt.Printf("Failed to cleanup excess logs: %v\n", err)
			}
		}
	}
}

// Close closes the database connection
func (s *LogStorage) Close() error {
	return s.db.Close()
}
