package rag

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/guileen/metabase/pkg/biz/rag/vocab"
)

// SearchResult 表示一个搜索结果项
type SearchResult struct {
	File     string  `json:"file"`      // 文件路径
	Line     int     `json:"line"`      // 匹配行号
	Score    float64 `json:"score"`     // 相似度分数 (0-1)
	Snippet  string  `json:"snippet"`   // 代码片段
	Context  string  `json:"context"`   // 上下文
	FileType string  `json:"file_type"` // 文件类型
	Reason   string  `json:"reason"`    // 匹配原因（可选）
}

// UnifiedRAG 统一的 RAG 接口，支持本地和云两种模式
type UnifiedRAG struct {
	config      *RAGConfig
	vocabMgr    *VocabularyManager
	client      *CloudClient // 云模式客户端
	initialized bool
}

// NewUnifiedRAG 创建统一的 RAG 实例
func NewUnifiedRAG(config *RAGConfig) (*UnifiedRAG, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	r := &UnifiedRAG{
		config:   config,
		vocabMgr: NewVocabularyManager(),
	}

	// 根据模式初始化客户端
	switch config.Mode {
	case LocalMode:
		// 本地模式，使用现有实现
		r.initialized = true
	case CloudMode:
		// 云模式，初始化云客户端
		client, err := NewCloudClient(config.CloudConfig)
		if err != nil {
			return nil, fmt.Errorf("初始化云客户端失败: %w", err)
		}
		r.client = client
		r.initialized = true
	}

	return r, nil
}

// Query 执行搜索查询（统一接口）
func (r *UnifiedRAG) Query(ctx context.Context, query string) ([]*SearchResult, error) {
	if !r.initialized {
		return nil, fmt.Errorf("RAG 未初始化")
	}

	if query = strings.TrimSpace(query); query == "" {
		return nil, fmt.Errorf("查询不能为空")
	}

	switch r.config.Mode {
	case LocalMode:
		return r.queryLocal(ctx, query)
	case CloudMode:
		return r.queryCloud(ctx, query)
	default:
		return nil, fmt.Errorf("未知的 RAG 模式: %v", r.config.Mode)
	}
}

// queryLocal 本地模式查询
func (r *UnifiedRAG) queryLocal(ctx context.Context, query string) ([]*SearchResult, error) {
	// 确保词表已准备就绪
	if r.config.VocabAutoBuild || r.config.VocabAutoUpdate {
		if err := r.vocabMgr.EnsureVocabulary(
			r.config.VocabAutoBuild,
			r.config.VocabAutoUpdate,
			r.config.VocabMaxAgeHours,
		); err != nil {
			fmt.Printf("[Warning] 词表管理失败: %v\n", err)
		}
	}

	// 转换为 SearchOptions 进行本地搜索
	opts := &SearchOptions{
		TopK:            r.config.TopK,
		Window:          r.config.Window,
		LocalMode:       r.config.LocalMode,
		EnableExpansion: r.config.EnableExpansion,
		EnableSkills:    r.config.EnableSkills,
		IncludeGlobs:    r.config.GetIncludeGlobs(),
		ExcludeGlobs:    r.config.GetExcludeGlobs(),
	}

	// 使用现有的本地搜索实现
	localRag := NewWithOptions(opts)
	return localRag.Query(ctx, query)
}

// queryCloud 云模式查询
func (r *UnifiedRAG) queryCloud(ctx context.Context, query string) ([]*SearchResult, error) {
	if r.client == nil {
		return nil, fmt.Errorf("云客户端未初始化")
	}

	// 构建云搜索请求
	req := &CloudSearchRequest{
		Query:           query,
		TopK:            r.config.TopK,
		Window:          r.config.Window,
		EnableExpansion: r.config.EnableExpansion,
		EnableSkills:    r.config.EnableSkills,
		IncludeGlobs:    r.config.GetIncludeGlobs(),
		ExcludeGlobs:    r.config.GetExcludeGlobs(),
	}

	return r.client.Search(ctx, req)
}

// GetStats 获取 RAG 统计信息
func (r *UnifiedRAG) GetStats() (*RAGStats, error) {
	switch r.config.Mode {
	case LocalMode:
		return r.getLocalStats()
	case CloudMode:
		return r.getCloudStats()
	default:
		return nil, fmt.Errorf("未知的 RAG 模式: %v", r.config.Mode)
	}
}

// getLocalStats 获取本地模式统计
func (r *UnifiedRAG) getLocalStats() (*RAGStats, error) {
	stats := &RAGStats{
		Mode: r.config.Mode.String(),
	}

	// 获取词表统计
	if r.vocabMgr != nil && r.vocabMgr.GetVocabularyBuilder() != nil {
		vocabStats := r.vocabMgr.GetVocabularyStats()
		if globalStats, ok := vocabStats["global_stats"]; ok {
			if gs, ok := globalStats.(*vocab.GlobalStats); ok {
				stats.VocabularyTerms = gs.UniqueTerms
				stats.VocabularyDocs = gs.TotalDocuments
				stats.VocabularyLastUpdated = gs.LastUpdated
			}
		}
	}

	return stats, nil
}

// getCloudStats 获取云模式统计
func (r *UnifiedRAG) getCloudStats() (*RAGStats, error) {
	if r.client == nil {
		return nil, fmt.Errorf("云客户端未初始化")
	}

	return r.client.GetStats()
}

// Close 关闭 RAG 实例
func (r *UnifiedRAG) Close() error {
	switch r.config.Mode {
	case LocalMode:
		// 本地模式无需特殊清理
		return nil
	case CloudMode:
		// 云模式关闭客户端
		if r.client != nil {
			return r.client.Close()
		}
		return nil
	default:
		return fmt.Errorf("未知的 RAG 模式: %v", r.config.Mode)
	}
}

// RAGStats RAG 统计信息
type RAGStats struct {
	Mode                  string      `json:"mode"`
	VocabularyTerms       int         `json:"vocabulary_terms"`
	VocabularyDocs        int         `json:"vocabulary_docs"`
	VocabularyLastUpdated time.Time   `json:"vocabulary_last_updated"`
	CloudStats            *CloudStats `json:"cloud_stats,omitempty"`
}

// CloudStats 云模式统计信息
type CloudStats struct {
	TotalDocuments int64     `json:"total_documents"`
	IndexSize      int64     `json:"index_size"`
	CacheHitRate   float64   `json:"cache_hit_rate"`
	LastSync       time.Time `json:"last_sync"`
}

// RAG 提供简化的 RAG 搜索接口（兼容性保留）
type RAG struct {
	rag *UnifiedRAG
}

// New 创建一个新的 RAG 实例（默认本地模式）
func New() *RAG {
	config := NewLocalConfig(".")
	unifiedRag, _ := NewUnifiedRAG(config)
	return &RAG{rag: unifiedRag}
}

// NewWithOptions 使用自定义选项创建 RAG 实例
func NewWithOptions(opts *SearchOptions) *RAG {
	config := NewLocalConfig(".")
	config.TopK = opts.TopK
	config.Window = opts.Window
	config.LocalMode = opts.LocalMode
	config.EnableExpansion = opts.EnableExpansion
	config.EnableSkills = opts.EnableSkills
	config.IncludeGlobs = opts.IncludeGlobs
	config.ExcludeGlobs = opts.ExcludeGlobs

	unifiedRag, _ := NewUnifiedRAG(config)
	return &RAG{rag: unifiedRag}
}

// NewLocal 创建本地模式 RAG
func NewLocal(rootPath string) (*RAG, error) {
	config := NewLocalConfig(rootPath)
	unifiedRag, err := NewUnifiedRAG(config)
	if err != nil {
		return nil, err
	}
	return &RAG{rag: unifiedRag}, nil
}

// NewCloud 创建云模式 RAG
func NewCloud(serviceURL, apiKey string) (*RAG, error) {
	config := NewCloudConfig(serviceURL, apiKey)
	unifiedRag, err := NewUnifiedRAG(config)
	if err != nil {
		return nil, err
	}
	return &RAG{rag: unifiedRag}, nil
}

// Query 执行简单的语义搜索查询
func (r *RAG) Query(ctx context.Context, query string) ([]*SearchResult, error) {
	return r.rag.Query(ctx, query)
}

// QueryWithOptions 使用自定义选项执行查询
func (r *RAG) QueryWithOptions(ctx context.Context, query string, opts *SearchOptions) ([]*SearchResult, error) {
	// 创建临时配置
	config := r.rag.config
	if config.Mode == LocalMode {
		config.TopK = opts.TopK
		config.Window = opts.Window
		config.LocalMode = opts.LocalMode
		config.EnableExpansion = opts.EnableExpansion
		config.EnableSkills = opts.EnableSkills
		config.IncludeGlobs = opts.IncludeGlobs
		config.ExcludeGlobs = opts.ExcludeGlobs
	}

	// 创建临时 RAG 实例
	tempRag, err := NewUnifiedRAG(config)
	if err != nil {
		return nil, err
	}
	defer tempRag.Close()

	return tempRag.Query(ctx, query)
}

// GetStats 获取统计信息
func (r *RAG) GetStats() (*RAGStats, error) {
	return r.rag.GetStats()
}

// GetConfig 获取配置（公开方法）
func (r *RAG) GetConfig() *RAGConfig {
	return r.rag.config
}

// Close 关闭 RAG 实例
func (r *RAG) Close() error {
	return r.rag.Close()
}

// SearchOptions 搜索配置选项
type SearchOptions struct {
	TopK            int      `json:"top_k"`            // 返回结果数量，默认 10
	Window          int      `json:"window"`           // 上下文窗口大小，默认 8
	IncludeGlobs    []string `json:"include_globs"`    // 包含的文件模式
	ExcludeGlobs    []string `json:"exclude_globs"`    // 排除的文件模式
	LocalMode       bool     `json:"local_mode"`       // 使用本地嵌入模式
	EnableExpansion bool     `json:"enable_expansion"` // 启用查询扩展
	EnableSkills    bool     `json:"enable_skills"`    // 启用技能系统
	ForceReindex    bool     `json:"force_reindex"`    // 强制重新索引
}

// DefaultSearchOptions 返回默认搜索配置
func DefaultSearchOptions() *SearchOptions {
	return &SearchOptions{
		TopK:            10,
		Window:          8,
		LocalMode:       os.Getenv("LLM_EMBEDDING_MODEL") == "",
		EnableExpansion: true,
		EnableSkills:    false,
		ForceReindex:    false,
		IncludeGlobs: []string{
			"*.go", "*.rs", "*.js", "*.ts", "*.py", "*.java", "*.cpp", "*.c",
			"*.h", "*.hpp", "*.cs", "*.php", "*.rb", "*.swift", "*.kt",
			"*.scala", "*.md", "*.txt", "*.json", "*.yaml", "*.yml", "*.toml",
			"*.sql", "*.sh", "*.html", "*.css", "*.vue", "*.jsx", "*.tsx",
		},
		ExcludeGlobs: []string{
			".git/*", "node_modules/*", "vendor/*", "dist/*", "build/*",
			"out/*", ".cache/*", "*.log", "*.tmp", "*.lock", "*.bak",
			"*.min.js", "*.min.css", "*.coverage/*", ".coverage/*",
		},
	}
}
