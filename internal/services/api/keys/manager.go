package keys

import ("context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt")

// Manager API密钥管理器
type Manager struct {
	db *sql.DB
}

// NewManager 创建API密钥管理器
func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// CreateKey 创建新的API密钥
func (m *Manager) CreateKey(ctx context.Context, req *CreateKeyRequest) (*APIKey, string, error) {
	key := &APIKey{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Type:      req.Type,
		Status:    KeyStatusActive,
		TenantID:  req.TenantID,
		ProjectID: req.ProjectID,
		UserID:    req.UserID,
		CreatedBy: "system", // TODO: 从上下文获取实际创建者
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  req.Metadata,
	}

	// 设置默认权限
	if len(req.Scopes) == 0 {
		key.Scopes = GetDefaultScopes(req.Type)
	} else {
		key.Scopes = req.Scopes
	}

	// 设置过期时间
	if req.ExpiresAt != nil {
		key.ExpiresAt = req.ExpiresAt
	}

	// 生成实际的API密钥
	actualKey, err := generateAPIKey(key.Type)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate API key: %w", err)
	}
	key.KeyPrefix = extractKeyPrefix(actualKey)

	// 加密存储密钥
	hashedKey, err := hashAPIKey(actualKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash API key: %w", err)
	}
	key.Key = hashedKey

	// 保存到数据库
	if err := m.saveKey(ctx, key); err != nil {
		return nil, "", fmt.Errorf("failed to save API key: %w", err)
	}

	return key, actualKey, nil
}

// GetKeyByKey 根据实际API密钥获取密钥信息
func (m *Manager) GetKeyByKey(ctx context.Context, actualKey string) (*APIKey, error) {
	// 提取密钥前缀用于快速查找
	prefix := extractKeyPrefix(actualKey)

	// 先通过前缀查找可能的密钥
	candidates, err := m.getKeysByPrefix(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to get keys by prefix: %w", err)
	}

	// 验证每个候选密钥
	for _, key := range candidates {
		if key.Status != KeyStatusActive {
			continue
		}

		if key.IsExpired() {
			continue
		}

		if verifyAPIKey(actualKey, key.Key) {
			// 更新最后使用时间
			if err := m.updateLastUsed(ctx, key.ID); err != nil {
				// 记录警告但不阻止请求
				fmt.Printf("Warning: failed to update last used time for key %s: %v\n", key.ID, err)
			}
			return key, nil
		}
	}

	return nil, fmt.Errorf("invalid API key")
}

// GetKeyByID 根据ID获取API密钥
func (m *Manager) GetKeyByID(ctx context.Context, id string) (*APIKey, error) {
	query := `
		SELECT id, name, api_key, key_prefix, type, status, scopes,
		       tenant_id, project_id, created_by, user_id, expires_at,
		       created_at, updated_at, last_used_at, usage_count, metadata
		FROM api_keys
		WHERE id = $1`

	var key APIKey
	var scopesJSON sql.NullString
	var metadataJSON sql.NullString
	var tenantID, projectID, userID sql.NullString
	var expiresAt, lastUsedAt sql.NullTime

	err := m.db.QueryRowContext(ctx, query, id).Scan(
		&key.ID, &key.Name, &key.Key, &key.KeyPrefix, &key.Type, &key.Status, &scopesJSON,
		&tenantID, &projectID, &key.CreatedBy, &userID, &expiresAt,
		&key.CreatedAt, &key.UpdatedAt, &lastUsedAt, &key.UsageCount, &metadataJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to query API key: %w", err)
	}

	// 处理可空字段
	if tenantID.Valid {
		key.TenantID = &tenantID.String
	}
	if projectID.Valid {
		key.ProjectID = &projectID.String
	}
	if userID.Valid {
		key.UserID = &userID.String
	}
	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}
	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	}

	// 解析JSON字段
	if scopesJSON.Valid {
		if err := json.Unmarshal([]byte(scopesJSON.String), &key.Scopes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal scopes: %w", err)
		}
	}
	if metadataJSON.Valid {
		if err := json.Unmarshal([]byte(metadataJSON.String), &key.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &key, nil
}

// ListKeys 列出API密钥（支持分页和过滤）
func (m *Manager) ListKeys(ctx context.Context, filter *KeyFilter) ([]*APIKey, int64, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	argIndex := 1

	if filter.TenantID != nil {
		where = append(where, fmt.Sprintf("tenant_id = $%d", argIndex))
		args = append(args, *filter.TenantID)
		argIndex++
	}

	if filter.ProjectID != nil {
		where = append(where, fmt.Sprintf("project_id = $%d", argIndex))
		args = append(args, *filter.ProjectID)
		argIndex++
	}

	if filter.Type != nil {
		where = append(where, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, *filter.Type)
		argIndex++
	}

	if filter.Status != nil {
		where = append(where, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.UserID != nil {
		where = append(where, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, *filter.UserID)
		argIndex++
	}

	whereClause := strings.Join(where, " AND ")

	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM api_keys WHERE %s", whereClause)
	var total int64
	if err := m.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count API keys: %w", err)
	}

	// 查询数据
	query := fmt.Sprintf(`
		SELECT id, name, api_key, key_prefix, type, status, scopes,
		       tenant_id, project_id, created_by, user_id, expires_at,
		       created_at, updated_at, last_used_at, usage_count, metadata
		FROM api_keys
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query API keys: %w", err)
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		var key APIKey
		var scopesJSON sql.NullString
		var metadataJSON sql.NullString
		var tenantID, projectID, userID sql.NullString
		var expiresAt, lastUsedAt sql.NullTime

		err := rows.Scan(
			&key.ID, &key.Name, &key.Key, &key.KeyPrefix, &key.Type, &key.Status, &scopesJSON,
			&tenantID, &projectID, &key.CreatedBy, &userID, &expiresAt,
			&key.CreatedAt, &key.UpdatedAt, &lastUsedAt, &key.UsageCount, &metadataJSON,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan API key: %w", err)
		}

		// 处理可空字段
		if tenantID.Valid {
			key.TenantID = &tenantID.String
		}
		if projectID.Valid {
			key.ProjectID = &projectID.String
		}
		if userID.Valid {
			key.UserID = &userID.String
		}
		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Time
		}
		if lastUsedAt.Valid {
			key.LastUsedAt = &lastUsedAt.Time
		}

		// 解析JSON字段
		if scopesJSON.Valid {
			if err := json.Unmarshal([]byte(scopesJSON.String), &key.Scopes); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal scopes: %w", err)
			}
		}
		if metadataJSON.Valid {
			if err := json.Unmarshal([]byte(metadataJSON.String), &key.Metadata); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		keys = append(keys, &key)
	}

	return keys, total, nil
}

// UpdateKey 更新API密钥
func (m *Manager) UpdateKey(ctx context.Context, id string, req *UpdateKeyRequest) (*APIKey, error) {
	key, err := m.GetKeyByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if req.Name != nil {
		key.Name = *req.Name
	}
	if req.Status != nil {
		key.Status = *req.Status
	}
	if req.Scopes != nil {
		key.Scopes = req.Scopes
	}
	if req.ExpiresAt != nil {
		key.ExpiresAt = req.ExpiresAt
	}
	if req.Metadata != nil {
		key.Metadata = req.Metadata
	}

	key.UpdatedAt = time.Now()

	if err := m.updateKey(ctx, key); err != nil {
		return nil, fmt.Errorf("failed to update API key: %w", err)
	}

	return key, nil
}

// RevokeKey 吊销API密钥
func (m *Manager) RevokeKey(ctx context.Context, id string) error {
	_, err := m.UpdateKey(ctx, id, &UpdateKeyRequest{
		Status: &[]KeyStatus{KeyStatusRevoked}[0],
	})
	return err
}

// DeleteKey 删除API密钥
func (m *Manager) DeleteKey(ctx context.Context, id string) error {
	query := `DELETE FROM api_keys WHERE id = $1`
	_, err := m.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}
	return nil
}

// GetKeyStats 获取API密钥使用统计
func (m *Manager) GetKeyStats(ctx context.Context, id string) (*KeyUsageStats, error) {
	query := `
		SELECT usage_count, last_used_at
		FROM api_keys
		WHERE id = $1`

	var stats KeyUsageStats
	stats.KeyID = id

	err := m.db.QueryRowContext(ctx, query, id).Scan(&stats.UsageCount, &stats.LastUsedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get key stats: %w", err)
	}

	// TODO: 获取端点使用统计
	stats.TopEndpoints = []EndpointUsage{}

	return &stats, nil
}

// 私有辅助方法

func generateAPIKey(keyType KeyType) (string, error) {
	// 生成32字节随机数据，增加安全性
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	// 使用base64url编码，移除padding以增加安全性
	key := base64.RawURLEncoding.EncodeToString(randomBytes)

	// 添加更长的随机前缀以防止枚举攻击
	var prefix string
	switch keyType {
	case KeyTypeSystem:
		prefix = "metabase_sys_" + generateRandomSuffix(4)
	case KeyTypeUser:
		prefix = "metabase_usr_" + generateRandomSuffix(4)
	case KeyTypeService:
		prefix = "metabase_svc_" + generateRandomSuffix(4)
	default:
		prefix = "metabase_key_" + generateRandomSuffix(4)
	}

	return prefix + key, nil
}

// generateRandomSuffix 生成随机后缀
func generateRandomSuffix(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func extractKeyPrefix(fullKey string) string {
	// 提取前缀部分用于快速查找
	// 新格式: metabase_sys_XXXX, metabase_usr_XXXX, etc.
	parts := strings.SplitN(fullKey, "_", 3)
	if len(parts) >= 3 {
		// 返回 metabase_sys_ 这样的前缀用于快速查找
		return parts[0] + "_" + parts[1] + "_" + parts[2][:4] + "_"
	}

	// 兼容旧格式，返回前8个字符
	if len(fullKey) < 8 {
		return fullKey
	}
	return fullKey[:8]
}

func hashAPIKey(key string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func verifyAPIKey(key, hashedKey string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedKey), []byte(key))
	return err == nil
}

func (m *Manager) saveKey(ctx context.Context, key *APIKey) error {
	scopesJSON, _ := json.Marshal(key.Scopes)
	metadataJSON, _ := json.Marshal(key.Metadata)

	query := `
		INSERT INTO api_keys (
			id, name, api_key, key_prefix, type, status, scopes,
			tenant_id, project_id, created_by, user_id, expires_at,
			created_at, updated_at, usage_count, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15, $16
		)`

	_, err := m.db.ExecContext(ctx, query,
		key.ID, key.Name, key.Key, key.KeyPrefix, key.Type, key.Status, string(scopesJSON),
		key.TenantID, key.ProjectID, key.CreatedBy, key.UserID, key.ExpiresAt,
		key.CreatedAt, key.UpdatedAt, key.UsageCount, string(metadataJSON),
	)

	return err
}

func (m *Manager) updateKey(ctx context.Context, key *APIKey) error {
	scopesJSON, _ := json.Marshal(key.Scopes)
	metadataJSON, _ := json.Marshal(key.Metadata)

	query := `
		UPDATE api_keys SET
			name = $2, status = $3, scopes = $4,
			expires_at = $5, updated_at = $6, metadata = $7
		WHERE id = $1`

	_, err := m.db.ExecContext(ctx, query,
		key.ID, key.Name, key.Status, string(scopesJSON),
		key.ExpiresAt, key.UpdatedAt, string(metadataJSON),
	)

	return err
}

func (m *Manager) getKeysByPrefix(ctx context.Context, prefix string) ([]*APIKey, error) {
	query := `
		SELECT id, name, api_key, key_prefix, type, status, scopes,
		       tenant_id, project_id, created_by, user_id, expires_at,
		       created_at, updated_at, last_used_at, usage_count, metadata
		FROM api_keys
		WHERE key_prefix LIKE $1 || '%'
		ORDER BY created_at DESC`

	rows, err := m.db.QueryContext(ctx, query, prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		var key APIKey
		var scopesJSON sql.NullString
		var metadataJSON sql.NullString
		var tenantID, projectID, userID sql.NullString
		var expiresAt, lastUsedAt sql.NullTime

		err := rows.Scan(
			&key.ID, &key.Name, &key.Key, &key.KeyPrefix, &key.Type, &key.Status, &scopesJSON,
			&tenantID, &projectID, &key.CreatedBy, &userID, &expiresAt,
			&key.CreatedAt, &key.UpdatedAt, &lastUsedAt, &key.UsageCount, &metadataJSON,
		)
		if err != nil {
			return nil, err
		}

		// 处理可空字段和JSON解析（简化版）
		if tenantID.Valid {
			key.TenantID = &tenantID.String
		}
		if projectID.Valid {
			key.ProjectID = &projectID.String
		}
		if userID.Valid {
			key.UserID = &userID.String
		}
		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Time
		}
		if lastUsedAt.Valid {
			key.LastUsedAt = &lastUsedAt.Time
		}
		if scopesJSON.Valid {
			json.Unmarshal([]byte(scopesJSON.String), &key.Scopes)
		}
		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &key.Metadata)
		}

		keys = append(keys, &key)
	}

	return keys, nil
}

func (m *Manager) updateLastUsed(ctx context.Context, keyID string) error {
	query := `UPDATE api_keys SET last_used_at = $1, usage_count = usage_count + 1 WHERE id = $2`
	_, err := m.db.ExecContext(ctx, query, time.Now(), keyID)
	return err
}

// KeyFilter 过滤条件
type KeyFilter struct {
	TenantID  *string    `json:"tenant_id,omitempty"`
	ProjectID *string    `json:"project_id,omitempty"`
	Type      *KeyType   `json:"type,omitempty"`
	Status    *KeyStatus `json:"status,omitempty"`
	UserID    *string    `json:"user_id,omitempty"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
}