package index

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	_ "github.com/mattn/go-sqlite3"
)

// InvertedIndex 倒排索引，基于SQLite FTS5实现
type InvertedIndex struct {
	db    *sql.DB
	table string
}

// Config 倒排索引配置
type Config struct {
	// 表名前缀
	TablePrefix string

	// 分词器配置
	Tokenizer string

	// BM25参数
	K1 float64
	B  float64
}

// DocumentIndex 索引中的文档信息
type DocumentIndex struct {
	ID       string
	TenantID string
	Type     string
	Title    string
	Content  string
	Metadata map[string]interface{}
}

// SearchHit 搜索命中
type SearchHit struct {
	DocumentID string
	Score      float64
	Rank       int
}

// NewInvertedIndex 创建倒排索引
func NewInvertedIndex(db *sql.DB, config *Config) (*InvertedIndex, error) {
	if config == nil {
		config = &Config{
			TablePrefix: "search",
			Tokenizer:   "unicode61 remove_diacritics 1",
			K1:          1.2,
			B:           0.75,
		}
	}

	table := config.TablePrefix + "_fts"

	idx := &InvertedIndex{
		db:    db,
		table: table,
	}

	// Validate table name to prevent SQL injection
	if err := idx.validateTableName(); err != nil {
		return nil, fmt.Errorf("invalid table configuration: %w", err)
	}

	if err := idx.initTables(); err != nil {
		return nil, fmt.Errorf("init tables: %w", err)
	}

	return idx, nil
}

// initTables 初始化数据库表
func (i *InvertedIndex) initTables() error {
	// 创建FTS5虚拟表
	sqlCreate := fmt.Sprintf(`
		CREATE VIRTUAL TABLE IF NOT EXISTS %s USING fts5(
			docid,
			tenant_id,
			doc_type,
			title,
			content,
			metadata,
			tokenize='%s',
			bm25=True
		)`, i.table, "unicode61 remove_diacritics 1")

	if _, err := i.db.Exec(sqlCreate); err != nil {
		return fmt.Errorf("create fts5 table: %w", err)
	}

	// 创建文档元数据表
	sqlMeta := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s_meta (
			docid TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			doc_type TEXT NOT NULL,
			title TEXT,
			content_length INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`, strings.TrimSuffix(i.table, "_fts"))

	if _, err := i.db.Exec(sqlMeta); err != nil {
		return fmt.Errorf("create meta table: %w", err)
	}

	// 创建索引
	sqlIdx := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_tenant ON %s_meta(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_%s_type ON %s_meta(doc_type);
		CREATE INDEX IF NOT EXISTS idx_%s_created ON %s_meta(created_at);
	`, strings.TrimSuffix(i.table, "_fts"), strings.TrimSuffix(i.table, "_fts"),
		strings.TrimSuffix(i.table, "_fts"), strings.TrimSuffix(i.table, "_fts"),
		strings.TrimSuffix(i.table, "_fts"), strings.TrimSuffix(i.table, "_fts"))

	if _, err := i.db.Exec(sqlIdx); err != nil {
		return fmt.Errorf("create indexes: %w", err)
	}

	return nil
}

// Index 索引文档
func (i *InvertedIndex) Index(doc *DocumentIndex) error {
	sqlInsert := fmt.Sprintf(`
		INSERT OR REPLACE INTO %s (docid, tenant_id, doc_type, title, content, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`, i.table)

	metadataJSON := "{}"
	if doc.Metadata != nil {
		// 这里应该使用json.Marshal，简化处理
		metadataJSON = fmt.Sprintf("%v", doc.Metadata)
	}

	_, err := i.db.Exec(sqlInsert, doc.ID, doc.TenantID, doc.Type, doc.Title, doc.Content, metadataJSON)
	if err != nil {
		return fmt.Errorf("insert document: %w", err)
	}

	// 更新元数据
	sqlMetaInsert := fmt.Sprintf(`
		INSERT OR REPLACE INTO %s_meta (docid, tenant_id, doc_type, title, content_length, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, strings.TrimSuffix(i.table, "_fts"))

	_, err = i.db.Exec(sqlMetaInsert, doc.ID, doc.TenantID, doc.Type, doc.Title, len(doc.Content))
	if err != nil {
		return fmt.Errorf("insert metadata: %w", err)
	}

	return nil
}

// Delete 删除文档
func (i *InvertedIndex) Delete(docID string) error {
	sqlDelete := fmt.Sprintf("DELETE FROM %s WHERE docid = ?", i.table)
	if _, err := i.db.Exec(sqlDelete, docID); err != nil {
		return fmt.Errorf("delete from fts: %w", err)
	}

	sqlMetaDelete := fmt.Sprintf("DELETE FROM %s_meta WHERE docid = ?", strings.TrimSuffix(i.table, "_fts"))
	if _, err := i.db.Exec(sqlMetaDelete, docID); err != nil {
		return fmt.Errorf("delete from meta: %w", err)
	}

	return nil
}

// Search 搜索文档
func (i *InvertedIndex) Search(ctx context.Context, query, tenantID string, limit int) ([]*DocumentIndex, []float64, error) {
	if query == "" {
		return nil, nil, fmt.Errorf("empty query")
	}

	// 验证输入参数
	if err := i.validateSearchInput(query, tenantID, limit); err != nil {
		return nil, nil, fmt.Errorf("invalid search input: %w", err)
	}

	// 构建FTS查询
	ftsQuery := i.buildFTSQuery(query)

	// 添加租户过滤 - 使用参数化查询
	if tenantID != "" {
		// 注意：FTS5的MATCH语法不支持参数化，所以我们需要在应用层进行严格验证
		if !i.isValidTenantID(tenantID) {
			return nil, nil, fmt.Errorf("invalid tenant ID format")
		}
		ftsQuery += " tenant_id:" + sanitizeFTSTerm(tenantID)
	}

	sqlSearch := fmt.Sprintf(`
		SELECT
			f.docid,
			f.tenant_id,
			f.doc_type,
			f.title,
			f.content,
			f.metadata,
			bm25(%s) as score,
			rank
		FROM %s f
		JOIN (
			SELECT docid, rank() OVER (ORDER BY bm25(%s)) as rank
			FROM %s
			WHERE %s MATCH ?
			ORDER BY bm25(%s)
			LIMIT ?
		) r ON f.docid = r.docid
		ORDER BY r.rank
	`, i.table, i.table, i.table, i.table, i.table, i.table)

	rows, err := i.db.QueryContext(ctx, sqlSearch, ftsQuery, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()

	var docs []*DocumentIndex
	var scores []float64

	for rows.Next() {
		doc := &DocumentIndex{}
		var score float64
		var rank int
		var metadata string

		err := rows.Scan(
			&doc.ID,
			&doc.TenantID,
			&doc.Type,
			&doc.Title,
			&doc.Content,
			&metadata,
			&score,
			&rank,
		)
		if err != nil {
			continue
		}

		// 解析metadata (简化处理)
		doc.Metadata = make(map[string]interface{})

		docs = append(docs, doc)
		scores = append(scores, score)
	}

	return docs, scores, nil
}

// buildFTSQuery 构建安全的FTS查询
func (i *InvertedIndex) buildFTSQuery(query string) string {
	// Sanitize and validate input
	if len(query) > 1000 {
		query = query[:1000] // Truncate to prevent DoS
	}

	query = strings.TrimSpace(query)

	// Remove potentially dangerous SQL syntax patterns
	dangerousPatterns := []string{
		"--", ";", "/*", "*/", "xp_", "sp_",
		"INSERT", "UPDATE", "DELETE", "DROP", "CREATE",
		"ALTER", "EXEC", "UNION", "SELECT", "FROM",
	}

	upperQuery := strings.ToUpper(query)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(upperQuery, pattern) {
			// Return a safe query if dangerous patterns are found
			return ""
		}
	}

	// 处理布尔操作符
	query = strings.ReplaceAll(query, " AND ", " ")
	query = strings.ReplaceAll(query, " OR ", " OR ")

	// 处理短语搜索 - ensure proper quoting
	if strings.Contains(query, `"`) {
		// Validate phrase quotes are balanced
		if strings.Count(query, `"`) % 2 == 0 {
			return query
		}
		// Remove unbalanced quotes
		query = strings.ReplaceAll(query, `"`, "")
	}

	// 处理排除词和通配符
	words := strings.Fields(query)
	var result []string

	for _, word := range words {
		if len(word) > 100 { // Prevent overly long terms
			continue
		}

		// Sanitize each term
		sanitized := sanitizeFTSTerm(word)
		if sanitized == "" {
			continue
		}

		if strings.HasPrefix(word, "-") && len(sanitized) > 1 {
			result = append(result, "-"+sanitized[1:])
		} else if strings.Contains(sanitized, "*") {
			// Only allow trailing wildcards
			if strings.LastIndex(sanitized, "*") == len(sanitized)-1 {
				result = append(result, sanitized)
			} else {
				// Move wildcard to end
				result = append(result, strings.ReplaceAll(sanitized, "*", "")+"*")
			}
		} else {
			result = append(result, sanitized+"*") // 添加前缀匹配
		}
	}

	// Limit number of terms to prevent complex queries
	if len(result) > 20 {
		result = result[:20]
	}

	return strings.Join(result, " ")
}

// Optimize 优化索引
func (i *InvertedIndex) Optimize() error {
	sqlOptimize := fmt.Sprintf("INSERT INTO %s(%s) VALUES('optimize')", i.table, i.table)
	if _, err := i.db.Exec(sqlOptimize); err != nil {
		return fmt.Errorf("optimize fts: %w", err)
	}
	return nil
}

// GetStats 获取统计信息
func (i *InvertedIndex) GetStats() (*IndexStats, error) {
	var stats IndexStats

	// 文档总数
	sqlCount := fmt.Sprintf("SELECT COUNT(*) FROM %s", strings.TrimSuffix(i.table, "_fts"))
	err := i.db.QueryRow(sqlCount).Scan(&stats.TotalDocuments)
	if err != nil {
		return nil, fmt.Errorf("count documents: %w", err)
	}

	// 总内容长度
	sqlLength := fmt.Sprintf("SELECT SUM(content_length) FROM %s_meta", strings.TrimSuffix(i.table, "_fts"))
	err = i.db.QueryRow(sqlLength).Scan(&stats.TotalContentLength)
	if err != nil {
		stats.TotalContentLength = 0
	}

	// 按类型统计
	sqlByType := fmt.Sprintf(`
		SELECT doc_type, COUNT(*)
		FROM %s_meta
		GROUP BY doc_type
	`, strings.TrimSuffix(i.table, "_fts"))

	rows, err := i.db.Query(sqlByType)
	if err != nil {
		return nil, fmt.Errorf("query by type: %w", err)
	}
	defer rows.Close()

	stats.DocumentsByType = make(map[string]int64)
	for rows.Next() {
		var docType string
		var count int64
		if err := rows.Scan(&docType, &count); err == nil {
			stats.DocumentsByType[docType] = count
		}
	}

	return &stats, nil
}

// IndexStats 索引统计信息
type IndexStats struct {
	TotalDocuments    int64
	TotalContentLength int64
	DocumentsByType   map[string]int64
}

// Close 关闭索引
func (i *InvertedIndex) Close() error {
	// SQLite连接由外部管理，这里不需要关闭
	return nil
}

// Rebuild 重建索引
func (i *InvertedIndex) Rebuild() error {
	// 删除现有表
	sqlDrop := fmt.Sprintf("DROP TABLE IF EXISTS %s", i.table)
	if _, err := i.db.Exec(sqlDrop); err != nil {
		return fmt.Errorf("drop fts table: %w", err)
	}

	sqlMetaDrop := fmt.Sprintf("DROP TABLE IF EXISTS %s_meta", strings.TrimSuffix(i.table, "_fts"))
	if _, err := i.db.Exec(sqlMetaDrop); err != nil {
		return fmt.Errorf("drop meta table: %w", err)
	}

	// 重新创建表
	return i.initTables()
}

// GetDocumentByID 根据ID获取文档
func (i *InvertedIndex) GetDocumentByID(docID string) (*DocumentIndex, error) {
	sqlSelect := fmt.Sprintf(`
		SELECT docid, tenant_id, doc_type, title, content, metadata
		FROM %s
		WHERE docid = ?
	`, i.table)

	doc := &DocumentIndex{}
	var metadata string
	err := i.db.QueryRow(sqlSelect, docID).Scan(
		&doc.ID,
		&doc.TenantID,
		&doc.Type,
		&doc.Title,
		&doc.Content,
		&metadata,
	)

	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}

	// 解析metadata
	doc.Metadata = make(map[string]interface{})

	return doc, nil
}

// UpdateDocument 更新文档
func (i *InvertedIndex) UpdateDocument(doc *DocumentIndex) error {
	return i.Index(doc) // INSERT OR REPLACE 会处理更新
}

// GetDocumentsByTenant 获取租户的所有文档
func (i *InvertedIndex) GetDocumentsByTenant(tenantID string, limit, offset int) ([]*DocumentIndex, error) {
	sqlSelect := fmt.Sprintf(`
		SELECT f.docid, f.tenant_id, f.doc_type, f.title, f.content, f.metadata
		FROM %s f
		JOIN %s_meta m ON f.docid = m.docid
		WHERE f.tenant_id = ?
		ORDER BY m.updated_at DESC
		LIMIT ? OFFSET ?
	`, i.table, strings.TrimSuffix(i.table, "_fts"))

	rows, err := i.db.Query(sqlSelect, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query tenant docs: %w", err)
	}
	defer rows.Close()

	var docs []*DocumentIndex
	for rows.Next() {
		doc := &DocumentIndex{}
		var metadata string

		err := rows.Scan(
			&doc.ID,
			&doc.TenantID,
			&doc.Type,
			&doc.Title,
			&doc.Content,
			&metadata,
		)
		if err != nil {
			continue
		}

		doc.Metadata = make(map[string]interface{})
		docs = append(docs, doc)
	}

	return docs, nil
}

// validateSearchInput validates search input parameters
func (i *InvertedIndex) validateSearchInput(query, tenantID string, limit int) error {
	// Validate query length and content
	if len(query) > 1000 {
		return fmt.Errorf("query too long (max 1000 characters)")
	}

	// Validate tenant ID format
	if tenantID != "" && !i.isValidTenantID(tenantID) {
		return fmt.Errorf("invalid tenant ID format")
	}

	// Validate limit
	if limit <= 0 || limit > 10000 {
		return fmt.Errorf("limit must be between 1 and 10000")
	}

	return nil
}

// isValidTenantID checks if tenant ID has valid format (alphanumeric, underscore, dash)
func (i *InvertedIndex) isValidTenantID(tenantID string) bool {
	if len(tenantID) == 0 || len(tenantID) > 64 {
		return false
	}

	// Only allow alphanumeric characters, underscores, and dashes
	validTenantID := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	return validTenantID.MatchString(tenantID)
}

// sanitizeFTSTerm sanitizes a term for FTS queries to prevent injection
func sanitizeFTSTerm(term string) string {
	// Remove or escape potentially dangerous characters
	var result strings.Builder

	for _, r := range term {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-':
			result.WriteRune(r)
		case r == ' ':
			result.WriteRune(' ')
		// Skip dangerous characters
		default:
			// Skip or replace with safe alternative
		}
	}

	return strings.TrimSpace(result.String())
}

// validateTableName validates table name to prevent SQL injection in dynamic queries
func (i *InvertedIndex) validateTableName() error {
	// Table name should be a valid identifier (letters, numbers, underscore)
	if len(i.table) == 0 || len(i.table) > 64 {
		return fmt.Errorf("invalid table name length")
	}

	validTable := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !validTable.MatchString(i.table) {
		return fmt.Errorf("invalid table name format")
	}

	return nil
}