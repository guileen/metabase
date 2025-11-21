package keys

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/guileen/metabase/internal/app/api/rest"
)

// KeyType 定义API密钥类型
type KeyType string

const (
	KeyTypeSystem  KeyType = "system"  // 系统级别密钥，拥有所有权限
	KeyTypeUser    KeyType = "user"    // 用户级别密钥，基于用户权限
	KeyTypeService KeyType = "service" // 服务账户密钥，用于服务间通信
)

// KeyStatus 定义API密钥状态
type KeyStatus string

const (
	KeyStatusActive   KeyStatus = "active"   // 激活状态
	KeyStatusInactive KeyStatus = "inactive" // 非激活状态
	KeyStatusRevoked  KeyStatus = "revoked"  // 已吊销
	KeyStatusExpired  KeyStatus = "expired"  // 已过期
)

// APIKey 表示API密钥实体
type APIKey struct {
	ID          string                 `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Key         string                 `json:"key" db:"api_key"`          // 加密存储的密钥
	KeyPrefix   string                 `json:"key_prefix" db:"key_prefix"` // 密钥前缀，用于显示
	Type        KeyType                `json:"type" db:"type"`
	Status      KeyStatus              `json:"status" db:"status"`

	// 权限范围
	Scopes      []string               `json:"scopes" db:"scopes"`         // 权限范围列表

	// 租户和项目隔离
	TenantID    *string                `json:"tenant_id,omitempty" db:"tenant_id"`
	ProjectID   *string                `json:"project_id,omitempty" db:"project_id"`

	// 用户关联
	CreatedBy   string                 `json:"created_by" db:"created_by"`
	UserID      *string                `json:"user_id,omitempty" db:"user_id"` // 关联的用户ID

	// 时间信息
	ExpiresAt   *time.Time             `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
	LastUsedAt  *time.Time             `json:"last_used_at,omitempty" db:"last_used_at"`

	// 使用统计
	UsageCount  int64                  `json:"usage_count" db:"usage_count"`

	// 元数据
	Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

// KeyScope 定义预定义的权限范围
var KeyScopes = map[string]string{
	// 数据操作权限
	"read":         "读取数据权限",
	"write":        "写入数据权限",
	"delete":       "删除数据权限",

	// 表结构权限
	"table:create": "创建表权限",
	"table:read":   "读取表结构权限",
	"table:update": "修改表结构权限",
	"table:delete": "删除表权限",

	// 系统管理权限
	"user:read":    "读取用户信息权限",
	"user:write":   "管理用户权限",
	"system:read":  "读取系统信息权限",
	"system:write": "系统管理权限",

	// 文件管理权限
	"file:read":    "读取文件权限",
	"file:write":   "上传文件权限",
	"file:delete":  "删除文件权限",

	// 分析权限
	"analytics:read": "读取分析数据权限",

	// 实时权限
	"realtime":     "实时订阅权限",
}

// GetDefaultScopes 根据密钥类型返回默认权限
func GetDefaultScopes(keyType KeyType) []string {
	switch keyType {
	case KeyTypeSystem:
		return []string{"read", "write", "delete", "table:create", "table:read", "table:update", "table:delete",
			"user:read", "user:write", "system:read", "system:write",
			"file:read", "file:write", "file:delete", "analytics:read", "realtime"}
	case KeyTypeUser:
		return []string{"read", "write", "file:read", "file:write", "analytics:read"}
	case KeyTypeService:
		return []string{"read", "write", "table:read", "analytics:read"}
	default:
		return []string{}
	}
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
	return k.Status == KeyStatusActive && !k.IsExpired()
}

// CanAccessTenant 检查密钥是否可以访问指定租户
func (k *APIKey) CanAccessTenant(tenantID string) bool {
	// 系统密钥可以访问所有租户
	if k.Type == KeyTypeSystem {
		return true
	}
	// 检查租户ID是否匹配
	if k.TenantID == nil {
		return false
	}
	return *k.TenantID == tenantID
}

// CanAccessProject 检查密钥是否可以访问指定项目
func (k *APIKey) CanAccessProject(projectID string) bool {
	// 系统密钥可以访问所有项目
	if k.Type == KeyTypeSystem {
		return true
	}
	// 检查项目ID是否匹配
	if k.ProjectID == nil {
		return false
	}
	return *k.ProjectID == projectID
}

// CreateKeyRequest 创建API密钥的请求
type CreateKeyRequest struct {
	Name        string                 `json:"name"`
	Type        KeyType                `json:"type"`
	Scopes      []string               `json:"scopes"`
	TenantID    *string                `json:"tenant_id,omitempty"`
	ProjectID   *string                `json:"project_id,omitempty"`
	UserID      *string                `json:"user_id,omitempty"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateKeyRequest 更新API密钥的请求
type UpdateKeyRequest struct {
	Name      *string    `json:"name,omitempty"`
	Status    *KeyStatus `json:"status,omitempty"`
	Scopes    []string   `json:"scopes,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// KeyUsageStats API密钥使用统计
type KeyUsageStats struct {
	KeyID       string         `json:"key_id"`
	UsageCount  int64          `json:"usage_count"`
	LastUsedAt  time.Time      `json:"last_used_at"`
	TopEndpoints []EndpointUsage `json:"top_endpoints"`
}

// EndpointUsage 端点使用统计
type EndpointUsage struct {
	Endpoint   string    `json:"endpoint"`
	Count      int64     `json:"count"`
	LastUsed   time.Time `json:"last_used"`
}

// GenerateKey 生成新的API密钥
func GenerateKey() (string, string, error) {
	// 生成32字节随机数据
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", err
	}

	// 转换为十六进制字符串
	key := hex.EncodeToString(randomBytes)

	// 生成前缀 (前8位用于显示)
	prefix := key[:8] + "..."

	return key, prefix, nil
}

// ToRestAPIKey 转换为rest包中的APIKey类型
func (k *APIKey) ToRestAPIKey() *rest.APIKey {
	return &rest.APIKey{
		ID:          k.ID,
		Name:        k.Name,
		Type:        string(k.Type),
		Scopes:      k.Scopes,
		TenantID:    k.TenantID,
		ProjectID:   k.ProjectID,
		ExpiresAt:   k.ExpiresAt,
		CreatedAt:   k.CreatedAt,
		UpdatedAt:   k.UpdatedAt,
		LastUsedAt:  k.LastUsedAt,
		UsageCount:  k.UsageCount,
		Metadata:    k.Metadata,
	}
}