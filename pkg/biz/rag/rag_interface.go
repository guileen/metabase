package rag

import (
	"fmt"
	"strings"
)

// Mode 定义 RAG 的使用模式
type Mode int

const (
	// LocalMode 本地代码仓库模式 - 用于 CLI、本地开发
	LocalMode Mode = iota
	// CloudMode 云服务模式 - 用于服务端搜索底层存储
	CloudMode
)

// String 返回模式的字符串表示
func (m Mode) String() string {
	switch m {
	case LocalMode:
		return "local"
	case CloudMode:
		return "cloud"
	default:
		return "unknown"
	}
}

// RAGConfig 统一的 RAG 配置
type RAGConfig struct {
	// 基本配置
	Mode     Mode   `json:"mode"`      // 使用模式
	RootPath string `json:"root_path"` // 根路径（本地模式）或服务端地址（云模式）

	// 搜索配置
	TopK         int      `json:"top_k"`         // 返回结果数量
	Window       int      `json:"window"`        // 上下文窗口大小
	IncludeGlobs []string `json:"include_globs"` // 包含的文件模式
	ExcludeGlobs []string `json:"exclude_globs"` // 排除的文件模式

	// 功能开关
	EnableExpansion bool `json:"enable_expansion"` // 启用查询扩展
	EnableSkills    bool `json:"enable_skills"`    // 启用技能系统
	LocalMode       bool `json:"local_mode"`       // 使用本地嵌入模式

	// 词表配置
	VocabAutoBuild   bool `json:"vocab_auto_build"`    // 自动构建词表
	VocabAutoUpdate  bool `json:"vocab_auto_update"`   // 自动更新词表
	VocabMaxAgeHours int  `json:"vocab_max_age_hours"` // 词表最大有效时间

	// 云模式专用配置
	CloudConfig *CloudConfig `json:"cloud_config,omitempty"` // 云服务配置
}

// CloudConfig 云服务模式配置
type CloudConfig struct {
	ServiceURL    string `json:"service_url"`    // 服务端 URL
	APIKey        string `json:"api_key"`        // API 密钥
	DatabaseURL   string `json:"database_url"`   // 数据库连接
	IndexName     string `json:"index_name"`     // 索引名称
	EnableCache   bool   `json:"enable_cache"`   // 启用缓存
	CacheSize     int    `json:"cache_size"`     // 缓存大小
	EnableMetrics bool   `json:"enable_metrics"` // 启用指标收集
}

// NewLocalConfig 创建本地模式配置
func NewLocalConfig(rootPath string) *RAGConfig {
	return &RAGConfig{
		Mode:             LocalMode,
		RootPath:         rootPath,
		TopK:             10,
		Window:           8,
		EnableExpansion:  true,
		EnableSkills:     false,
		LocalMode:        true, // 本地模式默认使用本地嵌入
		VocabAutoBuild:   true,
		VocabAutoUpdate:  true,
		VocabMaxAgeHours: 24,
		IncludeGlobs: []string{
			"*.go", "*.rs", "*.js", "*.ts", "*.py", "*.java", "*.cpp", "*.c",
			"*.h", "*.hpp", "*.cs", "*.php", "*.rb", "*.swift", "*.kt",
			"*.scala", "*.clj", "*.hs", "*.ml", "*.sh", "*.sql", "*.html",
			"*.css", "*.scss", "*.less", "*.vue", "*.jsx", "*.tsx",
			"*.md", "*.txt", "*.json", "*.yaml", "*.yml", "*.toml", "*.xml",
			"*.dockerfile", "Dockerfile*", "*.env", "*.ini", "*.cfg", "*.conf",
		},
		ExcludeGlobs: []string{
			".git/*", "node_modules/*", "vendor/*", "dist/*", "build/*",
			"out/*", ".cache/*", "*.log", "*.tmp", "*.lock", "*.bak",
			"*.swp", "*.swo", ".DS_Store", "Thumbs.db", "*.pyc", "__pycache__/*",
			"*.class", "*.jar", "*.war", "*.ear", "*.exe", "*.dll", "*.so",
			"*.dylib", "*.a", "*.lib", "*.obj", "*.o", "*.bin",
			".vscode/*", ".idea/*", "*.sublime-*", ".svn/*", ".hg/*",
			"target/*", "cargo-lock", "Cargo.lock", "poetry.lock", "yarn.lock",
			"package-lock.json", "go.sum", "*.min.js", "*.min.css",
			"*.map", "*.tsbuildinfo", "coverage/*", ".coverage*", "*.prof",
			"*.orig", "*.rej", ".#*", "*~", "#*#",
		},
	}
}

// NewCloudConfig 创建云模式配置
func NewCloudConfig(serviceURL, apiKey string) *RAGConfig {
	return &RAGConfig{
		Mode:             CloudMode,
		RootPath:         serviceURL,
		TopK:             20, // 云模式默认返回更多结果
		Window:           10, // 云模式默认更大的上下文
		EnableExpansion:  true,
		EnableSkills:     true, // 云模式默认启用技能系统
		LocalMode:        false,
		VocabAutoBuild:   false, // 云模式不管理词表
		VocabAutoUpdate:  false,
		VocabMaxAgeHours: 0,
		CloudConfig: &CloudConfig{
			ServiceURL:    serviceURL,
			APIKey:        apiKey,
			EnableCache:   true,
			CacheSize:     1000,
			EnableMetrics: true,
		},
	}
}

// Validate 验证配置
func (c *RAGConfig) Validate() error {
	switch c.Mode {
	case LocalMode:
		if c.RootPath == "" {
			return fmt.Errorf("本地模式需要指定根路径")
		}
		if c.TopK <= 0 {
			c.TopK = 10
		}
		if c.Window <= 0 {
			c.Window = 8
		}
	case CloudMode:
		if c.CloudConfig == nil {
			return fmt.Errorf("云模式需要提供 CloudConfig")
		}
		if c.CloudConfig.ServiceURL == "" {
			return fmt.Errorf("云模式需要指定服务端 URL")
		}
		if c.CloudConfig.APIKey == "" {
			return fmt.Errorf("云模式需要指定 API 密钥")
		}
	default:
		return fmt.Errorf("未知的 RAG 模式: %v", c.Mode)
	}

	return nil
}

// GetIncludeGlobs 获取包含的文件模式
func (c *RAGConfig) GetIncludeGlobs() []string {
	if len(c.IncludeGlobs) == 0 {
		return c.getDefaultIncludeGlobs()
	}
	return c.IncludeGlobs
}

// GetExcludeGlobs 获取排除的文件模式
func (c *RAGConfig) GetExcludeGlobs() []string {
	if len(c.ExcludeGlobs) == 0 {
		return c.getDefaultExcludeGlobs()
	}
	return c.ExcludeGlobs
}

// getDefaultIncludeGlobs 获取默认包含的文件模式
func (c *RAGConfig) getDefaultIncludeGlobs() []string {
	return []string{
		"*.go", "*.rs", "*.js", "*.ts", "*.py", "*.java", "*.cpp", "*.c",
		"*.h", "*.hpp", "*.cs", "*.php", "*.rb", "*.swift", "*.kt",
		"*.scala", "*.clj", "*.hs", "*.ml", "*.sh", "*.sql", "*.html",
		"*.css", "*.scss", "*.less", "*.vue", "*.jsx", "*.tsx",
		"*.md", "*.txt", "*.json", "*.yaml", "*.yml", "*.toml", "*.xml",
		"*.dockerfile", "Dockerfile*", "*.env", "*.ini", "*.cfg", "*.conf",
	}
}

// getDefaultExcludeGlobs 获取默认排除的文件模式
func (c *RAGConfig) getDefaultExcludeGlobs() []string {
	return []string{
		".git/*", "node_modules/*", "vendor/*", "dist/*", "build/*",
		"out/*", ".cache/*", "*.log", "*.tmp", "*.lock", "*.bak",
		"*.swp", "*.swo", ".DS_Store", "Thumbs.db", "*.pyc", "__pycache__/*",
		"*.class", "*.jar", "*.war", "*.ear", "*.exe", "*.dll", "*.so",
		"*.dylib", "*.a", "*.lib", "*.obj", "*.o", "*.bin",
		".vscode/*", ".idea/*", "*.sublime-*", ".svn/*", ".hg/*",
		"target/*", "cargo-lock", "Cargo.lock", "poetry.lock", "yarn.lock",
		"package-lock.json", "go.sum", "*.min.js", "*.min.css",
		"*.map", "*.tsbuildinfo", "coverage/*", ".coverage*", "*.prof",
		"*.orig", "*.rej", ".#*", "*~", "#*#",
	}
}

// String 返回配置的字符串表示
func (c *RAGConfig) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("mode=%s", c.Mode.String()))
	parts = append(parts, fmt.Sprintf("topK=%d", c.TopK))
	parts = append(parts, fmt.Sprintf("window=%d", c.Window))
	parts = append(parts, fmt.Sprintf("expand=%t", c.EnableExpansion))
	parts = append(parts, fmt.Sprintf("skills=%t", c.EnableSkills))
	parts = append(parts, fmt.Sprintf("localMode=%t", c.LocalMode))

	if c.Mode == LocalMode {
		parts = append(parts, fmt.Sprintf("root=%s", c.RootPath))
		parts = append(parts, fmt.Sprintf("vocabBuild=%t", c.VocabAutoBuild))
		parts = append(parts, fmt.Sprintf("vocabUpdate=%t", c.VocabAutoUpdate))
	}

	return strings.Join(parts, ", ")
}
