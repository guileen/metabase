package keys

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// Manager API密钥管理器
type Manager struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewManager 创建新的API密钥管理器
func NewManager(db *sql.DB, logger *zap.Logger) *Manager {
	return &Manager{
		db:     db,
		logger: logger,
	}
}

// Initialize 初始化数据库表
func (m *Manager) Initialize(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS api_keys (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		api_key TEXT UNIQUE NOT NULL,
		key_prefix TEXT NOT NULL,
		type TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		scopes TEXT NOT NULL DEFAULT '[]',
		tenant_id TEXT,
		project_id TEXT,
		created_by TEXT NOT NULL,
		user_id TEXT,
		expires_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_used_at TIMESTAMP,
		usage_count INTEGER DEFAULT 0,
		metadata TEXT DEFAULT '{}'
	);

	CREATE INDEX IF NOT EXISTS idx_api_keys_key ON api_keys(api_key);
	CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);
	CREATE INDEX IF NOT EXISTS idx_api_keys_type ON api_keys(type);
	CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_api_keys_project_id ON api_keys(project_id);
	`

	_, err := m.db.ExecContext(ctx, query)
	if err != nil {
		m.logger.Error("failed to initialize api_keys table", zap.Error(err))
		return fmt.Errorf("failed to initialize api_keys table: %w", err)
	}

	return nil
}

// Create 创建新的API密钥
func (m *Manager) Create(ctx context.Context, req *CreateKeyRequest) (*APIKey, error) {
	// 生成密钥
	key, prefix, err := GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// 加密密钥
	hashedKey, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash key: %w", err)
	}

	// 设置默认权限
	if len(req.Scopes) == 0 {
		req.Scopes = GetDefaultScopes(req.Type)
	}

	// 将权限数组转换为JSON字符串
	scopesJSON, _ := json.Marshal(req.Scopes)
	metadataJSON, _ := json.Marshal(req.Metadata)

	// 创建API密钥对象
	apiKey := &APIKey{
		ID:        generateID(),
		Name:      req.Name,
		Key:       string(hashedKey),
		KeyPrefix: prefix,
		Type:      req.Type,
		Status:    KeyStatusActive,
		Scopes:    req.Scopes,
		TenantID:  req.TenantID,
		ProjectID: req.ProjectID,
		CreatedBy: "system", // 默认创建者
		UserID:    req.UserID,
		ExpiresAt: req.ExpiresAt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  req.Metadata,
	}

	// 插入数据库
	query := `
	INSERT INTO api_keys (id, name, api_key, key_prefix, type, status, scopes,
		tenant_id, project_id, created_by, user_id, expires_at, metadata)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = m.db.ExecContext(ctx, query,
		apiKey.ID, apiKey.Name, apiKey.Key, apiKey.KeyPrefix, apiKey.Type,
		apiKey.Status, string(scopesJSON), apiKey.TenantID, apiKey.ProjectID,
		apiKey.CreatedBy, apiKey.UserID, apiKey.ExpiresAt, string(metadataJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create api key: %w", err)
	}

	m.logger.Info("API key created",
		zap.String("id", apiKey.ID),
		zap.String("name", apiKey.Name),
		zap.String("type", string(apiKey.Type)),
	)

	// 返回原始密钥（只在创建时返回一次）
	apiKey.Key = key

	return apiKey, nil
}

// Validate 验证API密钥
func (m *Manager) Validate(ctx context.Context, key string) (*APIKey, error) {
	// 查询数据库
	query := `
	SELECT id, name, api_key, key_prefix, type, status, scopes,
		tenant_id, project_id, created_by, user_id, expires_at,
		created_at, updated_at, last_used_at, usage_count, metadata
	FROM api_keys
	WHERE key_prefix = $1
	`

	var apiKey APIKey
	var scopesJSON, metadataJSON []byte

	err := m.db.QueryRowContext(ctx, query, getKeyPrefix(key)).Scan(
		&apiKey.ID, &apiKey.Name, &apiKey.Key, &apiKey.KeyPrefix, &apiKey.Type,
		&apiKey.Status, &scopesJSON, &apiKey.TenantID, &apiKey.ProjectID,
		&apiKey.CreatedBy, &apiKey.UserID, &apiKey.ExpiresAt, &apiKey.CreatedAt,
		&apiKey.UpdatedAt, &apiKey.LastUsedAt, &apiKey.UsageCount, &metadataJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("api key not found")
		}
		return nil, fmt.Errorf("failed to query api key: %w", err)
	}

	// 验证密钥
	if err := bcrypt.CompareHashAndPassword([]byte(apiKey.Key), []byte(key)); err != nil {
		return nil, fmt.Errorf("invalid api key")
	}

	// 检查密钥状态
	if !apiKey.IsValid() {
		return nil, fmt.Errorf("api key is invalid or expired")
	}

	// 解析权限和metadata
	if len(scopesJSON) > 0 {
		json.Unmarshal(scopesJSON, &apiKey.Scopes)
	}
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &apiKey.Metadata)
	}

	// 更新使用统计
	go m.updateUsageStats(apiKey.ID)

	return &apiKey, nil
}

// GetByID 根据ID获取API密钥
func (m *Manager) GetByID(ctx context.Context, id string) (*APIKey, error) {
	query := `
	SELECT id, name, api_key, key_prefix, type, status, scopes,
		tenant_id, project_id, created_by, user_id, expires_at,
		created_at, updated_at, last_used_at, usage_count, metadata
	FROM api_keys
	WHERE id = $1
	`

	var apiKey APIKey
	var metadata []byte

	err := m.db.QueryRowContext(ctx, query, id).Scan(
		&apiKey.ID, &apiKey.Name, &apiKey.Key, &apiKey.KeyPrefix, &apiKey.Type,
		&apiKey.Status, &apiKey.Scopes, &apiKey.TenantID, &apiKey.ProjectID,
		&apiKey.CreatedBy, &apiKey.UserID, &apiKey.ExpiresAt, &apiKey.CreatedAt,
		&apiKey.UpdatedAt, &apiKey.LastUsedAt, &apiKey.UsageCount, &metadata,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("api key not found")
		}
		return nil, fmt.Errorf("failed to query api key: %w", err)
	}

	// 解析metadata
	if len(metadata) > 0 {
		apiKey.Metadata = make(map[string]interface{})
	}

	// 不返回实际密钥
	apiKey.Key = ""

	return &apiKey, nil
}

// List 列出API密钥
func (m *Manager) List(ctx context.Context, tenantID, projectID *string, limit, offset int) ([]*APIKey, error) {
	query := `
	SELECT id, name, api_key, key_prefix, type, status, scopes,
		tenant_id, project_id, created_by, user_id, expires_at,
		created_at, updated_at, last_used_at, usage_count, metadata
	FROM api_keys
	WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

	if tenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", argIndex)
		args = append(args, *tenantID)
		argIndex++
	}

	if projectID != nil {
		query += fmt.Sprintf(" AND project_id = $%d", argIndex)
		args = append(args, *projectID)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
		argIndex++
	}

	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, offset)
	}

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query api keys: %w", err)
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		var apiKey APIKey
		var metadata []byte

		err := rows.Scan(
			&apiKey.ID, &apiKey.Name, &apiKey.Key, &apiKey.KeyPrefix, &apiKey.Type,
			&apiKey.Status, &apiKey.Scopes, &apiKey.TenantID, &apiKey.ProjectID,
			&apiKey.CreatedBy, &apiKey.UserID, &apiKey.ExpiresAt, &apiKey.CreatedAt,
			&apiKey.UpdatedAt, &apiKey.LastUsedAt, &apiKey.UsageCount, &metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan api key: %w", err)
		}

		// 解析metadata
		if len(metadata) > 0 {
			apiKey.Metadata = make(map[string]interface{})
		}

		// 不返回实际密钥
		apiKey.Key = ""

		keys = append(keys, &apiKey)
	}

	return keys, nil
}

// Update 更新API密钥
func (m *Manager) Update(ctx context.Context, id string, req *UpdateKeyRequest) (*APIKey, error) {
	query := `
	UPDATE api_keys
	SET name = COALESCE($1, name),
		status = COALESCE($2, status),
		scopes = COALESCE($3, scopes),
		expires_at = COALESCE($4, expires_at),
		metadata = COALESCE($5, metadata),
		updated_at = CURRENT_TIMESTAMP
	WHERE id = $6
	RETURNING id, name, api_key, key_prefix, type, status, scopes,
		tenant_id, project_id, created_by, user_id, expires_at,
		created_at, updated_at, last_used_at, usage_count, metadata
	`

	var apiKey APIKey
	var metadata []byte

	err := m.db.QueryRowContext(ctx, query,
		req.Name, req.Status, req.Scopes, req.ExpiresAt, req.Metadata, id,
	).Scan(
		&apiKey.ID, &apiKey.Name, &apiKey.Key, &apiKey.KeyPrefix, &apiKey.Type,
		&apiKey.Status, &apiKey.Scopes, &apiKey.TenantID, &apiKey.ProjectID,
		&apiKey.CreatedBy, &apiKey.UserID, &apiKey.ExpiresAt, &apiKey.CreatedAt,
		&apiKey.UpdatedAt, &apiKey.LastUsedAt, &apiKey.UsageCount, &metadata,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("api key not found")
		}
		return nil, fmt.Errorf("failed to update api key: %w", err)
	}

	// 解析metadata
	if len(metadata) > 0 {
		apiKey.Metadata = make(map[string]interface{})
	}

	// 不返回实际密钥
	apiKey.Key = ""

	return &apiKey, nil
}

// Delete 删除API密钥
func (m *Manager) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM api_keys WHERE id = $1`

	result, err := m.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete api key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("api key not found")
	}

	return nil
}

// updateUsageStats 更新使用统计
func (m *Manager) updateUsageStats(keyID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
	UPDATE api_keys
	SET usage_count = usage_count + 1,
		last_used_at = CURRENT_TIMESTAMP
	WHERE id = $1
	`

	_, err := m.db.ExecContext(ctx, query, keyID)
	if err != nil {
		m.logger.Error("failed to update usage stats", zap.String("key_id", keyID), zap.Error(err))
	}
}

// getKeyPrefix 从完整密钥中提取前缀
func getKeyPrefix(key string) string {
	if len(key) >= 8 {
		return key[:8] + "..."
	}
	return key
}

// generateID 生成唯一ID
func generateID() string {
	return fmt.Sprintf("key_%d", time.Now().UnixNano())
}
