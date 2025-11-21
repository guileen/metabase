# RAG - 简化的语义搜索系统

这个 RAG 系统提供了一个极其简单易用的语义搜索接口，专为代码仓库搜索优化。

## 🚀 快速开始

### 最简单的使用

```go
import "github.com/guileen/metabase/pkg/rag"

// 一行代码完成语义搜索
results, err := rag.QuickSearch("如何使用嵌入系统")
if err != nil {
    log.Fatal(err)
}

for _, result := range results {
    fmt.Printf("%s:%d (score=%.3f)\n", result.File, result.Line, result.Score)
    fmt.Printf("  %s\n", result.Snippet)
}
```

### 更多选项

```go
// 自定义搜索选项
opts := rag.DefaultSearchOptions()
opts.TopK = 15                    // 返回 15 个结果
opts.Window = 10                  // 上下文窗口 10 行
opts.EnableSkills = true          // 启用技能系统
opts.IncludeGlobs = []string{"*.go", "*.md"}

rag := rag.NewWithOptions(opts)
results, err := rag.Query(context.Background(), "数据库设计")
```

## 📋 API 接口

### 简单接口

```go
// 最简单的搜索
rag.QuickSearch(query string) ([]*SearchResult, error)

// 指定结果数量的搜索
rag.QuickSearchWithTop(query string, topK int) ([]*SearchResult, error)

// 批量搜索
rag.BatchSearch(queries []string) (map[string][]*SearchResult, error)
```

### 完整控制接口

```go
// 创建 RAG 实例
rag := rag.New()                           // 使用默认配置
rag := rag.NewWithOptions(opts)            // 使用自定义配置

// 执行搜索
results, err := rag.Query(context.Background(), query, opts)
```

### 数据结构

```go
type SearchResult struct {
    File      string  `json:"file"`       // 文件路径
    Line      int     `json:"line"`       // 匹配行号
    Score     float64 `json:"score"`      // 相似度分数 (0-1)
    Snippet   string  `json:"snippet"`    // 代码片段
    Context   string  `json:"context"`    // 上下文
    FileType  string  `json:"file_type"`  // 文件类型
    Reason    string  `json:"reason"`     // 匹配原因
}

type SearchOptions struct {
    TopK            int      `json:"top_k"`             // 返回结果数量，默认 10
    Window          int      `json:"window"`            // 上下文窗口大小，默认 8
    IncludeGlobs    []string `json:"include_globs"`     // 包含的文件模式
    ExcludeGlobs    []string `json:"exclude_globs"`     // 排除的文件模式
    LocalMode       bool     `json:"local_mode"`        // 使用本地嵌入模式
    EnableExpansion bool     `json:"enable_expansion"`  // 启用查询扩展
    EnableSkills    bool     `json:"enable_skills"`     // 启用技能系统
    ForceReindex    bool     `json:"force_reindex"`     // 强制重新索引
}
```

## 💻 CLI 使用

### 基本用法

```bash
# 简单搜索
metabase rag "如何使用嵌入系统"

# 指定结果数量
metabase rag --top 5 "数据库连接"

# 本地模式（更快）
metabase rag --local "API 设计"

# 启用技能系统
metabase rag --skills "设计模式的使用"
```

### 文件过滤

```bash
# 只搜索 Go 文件
metabase rag --include "*.go" "并发处理"

# 排除测试文件
metabase rag --exclude "*_test.go" "核心业务逻辑"

# 多种文件类型
metabase rag --include "*.go" --include "*.md" "架构设计"
```

### 高级选项

```bash
# 完整配置示例
metabase rag \
  --top 15 \
  --window 10 \
  --local \
  --expand \
  --skills \
  --include "*.go" \
  --exclude "*_test.go" \
  "性能优化策略"
```

## 🎯 核心特性

### 1. 简单易用
- **一行代码**完成语义搜索
- **零配置**启动，智能默认值
- **自动处理**文件过滤、嵌入、排序等复杂逻辑

### 2. 高度可配置
- **灵活的文件过滤**：支持 glob 模式包含/排除
- **多种嵌入模式**：本地模式 + 远程模式
- **查询扩展**：词表扩展 + 技能系统
- **自定义评分**：文件类型优先级、路径评分

### 3. 性能优化
- **智能缓存**：词表缓存、嵌入缓存
- **批量处理**：文件批量读取、嵌入批量生成
- **并行计算**：文件过滤、内容搜索并行执行
- **内存优化**：限制候选数量、流式处理

### 4. 现有集成
- **词表系统**：自动构建和更新
- **技能系统**：AI 增强的查询扩展
- **嵌入系统**：本地 + 远程嵌入支持
- **重排序**：LLM 重排序（可选）

## 🔧 配置选项

### 环境变量

```bash
# 嵌入模型配置
LLM_EMBEDDING_MODEL=gte-small-zh
LLM_BASE_URL=https://api.openai.com/v1
LLM_API_KEY=your-api-key

# 重排序模型
LLM_RERANK_MODEL=bge-reranker-base

# 搜索限制
SEARCH_MAX_CANDIDATES=300
```

### 默认文件过滤

**包含的文件类型**：
- 代码文件：`.go`, `.rs`, `.py`, `.js`, `.ts`, `.java`, `.cpp`, `.c`, `.h` 等
- 配置文件：`.md`, `.json`, `.yaml`, `.yml`, `.toml`, `.sql`, `.sh` 等
- Web 文件：`.html`, `.css`, `.vue`, `.jsx`, `.tsx` 等

**排除的文件**：
- 构建产物：`node_modules`, `vendor`, `dist`, `build`, `target`
- 缓存文件：`.cache`, `*.log`, `*.tmp`
- 版本控制：`.git`
- 压缩文件：`*.min.js`, `*.min.css`

## 📊 使用示例

### 代码搜索

```go
// 查找特定功能的实现
results, err := rag.QuickSearch("用户认证的实现")

// 查找错误处理
results, err := rag.QuickSearch("错误处理和日志记录")

// 查找性能优化
results, err := rag.QuickSearch("数据库查询优化")
```

### 架构分析

```go
// 查找架构模式
results, err := rag.QuickSearch("依赖注入和控制反转")

// 查找设计模式
results, err := rag.QuickSearch("工厂模式和单例模式")

// 查找最佳实践
results, err := rag.QuickSearch("代码复用和模块化")
```

### 学习代码库

```go
// 查找核心功能
results, err := rag.QuickSearch("主要业务流程")

// 查找 API 设计
results, err := rag.QuickSearch("RESTful API 设计")

// 查找测试覆盖
results, err := rag.QuickSearch("单元测试和集成测试")
```

## 🔍 与原有搜索的区别

### 原有 search 命令
- **705 行复杂实现**
- 需要手动管理词表、嵌入、重排序
- 大量底层细节需要处理
- 配置分散在多个参数中

### 新的 RAG 系统
- **一行代码完成搜索**
- 自动处理所有复杂逻辑
- 统一的配置选项
- 更好的错误处理和用户体验

### 代码简化对比

```go
// 原来的方式（简化版）
vocabMgr := NewVocabularyManager()
vocabMgr.EnsureVocabulary(true, true, 24)
files := rgFiles()
filteredFiles := filterFiles(files, fileFilter)
terms := expandQuery(query)
candidates := findCandidates(terms, filteredFiles)
embeddings := generateEmbeddings(candidates)
scores := calculateSimilarity(query, candidates)
results := rerankResults(query, scores)
sortResults(results)

// 新的方式
results, err := rag.QuickSearch(query)
```

## 🚀 下一步计划

1. **更多数据源**：数据库、API、文档等
2. **实时索引**：文件变更自动更新索引
3. **高级查询**：布尔查询、范围查询、过滤查询
4. **可视化界面**：Web 界面、图形化结果展示
5. **性能监控**：详细的性能指标和分析

## 📝 贡献指南

欢迎提交 Issue 和 Pull Request！

主要开发方向：
- 新的数据源支持
- 性能优化
- 新的查询功能
- 更好的用户体验