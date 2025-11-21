package vocab

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// VocabularyIndex 词表索引结构
type VocabularyIndex struct {
	// 词汇到文档的倒排索引
	TermToDocs map[string]*TermInfo `json:"term_to_docs"`

	// 文档到词汇的正向索引
	DocToTerms map[string]*DocumentInfo `json:"doc_to_terms"`

	// 全局统计信息
	GlobalStats *GlobalStats `json:"global_stats"`

	// 配置信息
	Config *Config `json:"config"`

	// 索引元数据
	Metadata *IndexMetadata `json:"metadata"`

	// 读写锁
	mutex sync.RWMutex
}

// TermInfo 词汇信息
type TermInfo struct {
	Term         string           `json:"term"`
	DocumentFreq int              `json:"document_freq"` // 包含该词汇的文档数
	TotalFreq    int              `json:"total_freq"`    // 该词汇在所有文档中的总频次
	Documents    map[string]int   `json:"documents"`     // 文档ID -> 在该文档中的频次
	Positions    map[string][]int `json:"positions"`     // 文档ID -> 词汇在文档中的位置列表
	LastSeen     time.Time        `json:"last_seen"`     // 最后一次见到该词汇的时间
	Weight       float64          `json:"weight"`        // 词汇权重 (类似TF-IDF)
	Category     string           `json:"category"`      // 词汇分类 (code, identifier, comment, etc.)
}

// DocumentInfo 文档信息
type DocumentInfo struct {
	Path          string           `json:"path"`
	FileHash      string           `json:"file_hash"`      // 文件内容哈希
	LastModified  time.Time        `json:"last_modified"`  // 文件最后修改时间
	TotalTerms    int              `json:"total_terms"`    // 文档中总词汇数
	UniqueTerms   int              `json:"unique_terms"`   // 文档中唯一词汇数
	TermFreqs     map[string]int   `json:"term_freqs"`     // 词汇 -> 在该文档中的频次
	TermPositions map[string][]int `json:"term_positions"` // 词汇 -> 在文档中的位置列表
	Language      string           `json:"language"`       // 文档语言 (go, rs, js, etc.)
	FileType      string           `json:"file_type"`      // 文件类型
	Size          int64            `json:"size"`           // 文件大小
}

// GlobalStats 全局统计信息
type GlobalStats struct {
	TotalDocuments int       `json:"total_documents"`
	TotalTerms     int       `json:"total_terms"`
	UniqueTerms    int       `json:"unique_terms"`
	TotalTokens    int       `json:"total_tokens"`
	LastUpdated    time.Time `json:"last_updated"`
	IndexSize      int64     `json:"index_size"`      // 索引大小（字节）
	AvgDocLength   float64   `json:"avg_doc_length"`  // 平均文档长度
	VocabularySize int       `json:"vocabulary_size"` // 词汇表大小
}

// IndexMetadata 索引元数据
type IndexMetadata struct {
	Version        string        `json:"version"`
	CreatedAt      time.Time     `json:"created_at"`
	LastUpdate     time.Time     `json:"last_update"`
	TotalUpdates   int           `json:"total_updates"`
	BuildDuration  time.Duration `json:"build_duration"`
	LastBuildFiles int           `json:"last_build_files"`
	IndexerVersion string        `json:"indexer_version"`
}

// Config 词表配置
type Config struct {
	// 存储配置
	DataDir   string `json:"data_dir"`   // 数据目录
	IndexFile string `json:"index_file"` // 索引文件路径

	// 处理配置
	MinTermLength int      `json:"min_term_length"` // 最小词汇长度
	MaxTermLength int      `json:"max_term_length"` // 最大词汇长度
	StopWords     []string `json:"stop_words"`      // 停用词列表

	// 更新配置
	AutoUpdate     bool `json:"auto_update"`     // 自动更新
	UpdateInterval int  `json:"update_interval"` // 更新间隔（分钟）

	// 过滤配置
	IncludePatterns []string `json:"include_patterns"` // 包含文件模式
	ExcludePatterns []string `json:"exclude_patterns"` // 排除文件模式

	// 性能配置
	MaxDocsPerTerm int `json:"max_docs_per_term"` // 每个词汇最大文档数
	MaxPositions   int `json:"max_positions"`     // 每个文档最大位置记录数
}

// UpdateResult 更新结果
type UpdateResult struct {
	AddedFiles   int           `json:"added_files"`
	UpdatedFiles int           `json:"updated_files"`
	DeletedFiles int           `json:"deleted_files"`
	NewTerms     int           `json:"new_terms"`
	RemovedTerms int           `json:"removed_terms"`
	Duration     time.Duration `json:"duration"`
	Errors       []string      `json:"errors"`
}

// QueryExpansionResult 查询扩展结果
type QueryExpansionResult struct {
	OriginalTerms []string            `json:"original_terms"`
	ExpandedTerms []string            `json:"expanded_terms"`
	SimilarTerms  map[string][]string `json:"similar_terms"`  // 原词 -> 相似词列表
	CategoryTerms map[string][]string `json:"category_terms"` // 按分类的词汇
	WeightedTerms map[string]float64  `json:"weighted_terms"` // 词汇 -> 权重
}

// NewVocabularyIndex 创建新的词表索引
func NewVocabularyIndex(config *Config) *VocabularyIndex {
	if config == nil {
		config = CreateDefaultConfig()
	}

	// 确保数据目录存在
	os.MkdirAll(config.DataDir, 0755)

	return &VocabularyIndex{
		TermToDocs:  make(map[string]*TermInfo),
		DocToTerms:  make(map[string]*DocumentInfo),
		GlobalStats: &GlobalStats{},
		Config:      config,
		Metadata: &IndexMetadata{
			Version:        "1.0.0",
			IndexerVersion: "vocabbuilder/1.0.0",
		},
	}
}

// CreateDefaultConfig 创建默认配置
func CreateDefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".metabase", "vocab")

	return &Config{
		DataDir:        dataDir,
		IndexFile:      filepath.Join(dataDir, "vocabulary.idx"),
		MinTermLength:  2,
		MaxTermLength:  50,
		AutoUpdate:     true,
		UpdateInterval: 60, // 1小时

		IncludePatterns: []string{
			"*.go", "*.rs", "*.js", "*.ts", "*.py", "*.java",
			"*.cpp", "*.c", "*.h", "*.hpp", "*.cs", "*.php",
			"*.md", "*.txt", "*.json", "*.yaml", "*.yml",
		},
		ExcludePatterns: []string{
			"*.log", "*.tmp", "*.lock", "*.bak",
			".git/*", "node_modules/*", "vendor/*", "target/*",
		},
		MaxDocsPerTerm: 10000,
		MaxPositions:   100,
	}
}

// BuildIndex 构建词表索引
func (vi *VocabularyIndex) BuildIndex(filePaths []string) (*UpdateResult, error) {
	startTime := time.Now()
	result := &UpdateResult{}

	vi.mutex.Lock()
	defer vi.mutex.Unlock()

	fmt.Printf("[Vocab] Building vocabulary index from %d files\n", len(filePaths))

	// 清理现有索引（如果是从头构建）
	if vi.Metadata.TotalUpdates == 0 {
		vi.TermToDocs = make(map[string]*TermInfo)
		vi.DocToTerms = make(map[string]*DocumentInfo)
		vi.GlobalStats = &GlobalStats{}
	}

	// 处理每个文件
	for _, filePath := range filePaths {
		if err := vi.processFile(filePath, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error processing %s: %v", filePath, err))
			continue
		}
	}

	vi.updateGlobalStats()
	vi.calculateWeights()

	// 更新元数据
	vi.Metadata.LastUpdate = time.Now()
	vi.Metadata.TotalUpdates++
	vi.Metadata.LastBuildFiles = len(filePaths)
	vi.Metadata.BuildDuration = time.Since(startTime)

	result.Duration = time.Since(startTime)

	// 保存索引（BuildIndex 已经有锁了，这里不加锁）
	if err := vi.saveIndex(); err != nil {
		return result, fmt.Errorf("failed to save index: %w", err)
	}

	fmt.Printf("[Vocab] Index build completed in %v\n", result.Duration)
	fmt.Printf("[Vocab] Added: %d, Updated: %d, Deleted: %d files\n",
		result.AddedFiles, result.UpdatedFiles, result.DeletedFiles)
	fmt.Printf("[Vocab] New terms: %d, Removed terms: %d\n",
		result.NewTerms, result.RemovedTerms)

	return result, nil
}

// processFile 处理单个文件
func (vi *VocabularyIndex) processFile(filePath string, result *UpdateResult) error {
	// 检查文件是否应该被处理
	if !vi.shouldProcessFile(filePath) {
		return nil
	}

	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// 计算文件哈希
	fileHash, err := vi.calculateFileHash(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate file hash: %w", err)
	}

	// 检查文档是否已存在
	existingDoc, exists := vi.DocToTerms[filePath]

	if exists {
		// 检查文件是否有变更
		if existingDoc.FileHash == fileHash && existingDoc.LastModified.Equal(fileInfo.ModTime()) {
			// 文件未变更，跳过
			return nil
		}

		// 文件已变更，先删除旧的词汇信息
		vi.removeDocument(filePath)
		result.UpdatedFiles++
	} else {
		result.AddedFiles++
	}

	// 解析文件内容
	terms, _, err := vi.parseFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	if len(terms) == 0 {
		return nil
	}

	// 创建文档信息
	docInfo := &DocumentInfo{
		Path:          filePath,
		FileHash:      fileHash,
		LastModified:  fileInfo.ModTime(),
		TotalTerms:    len(terms),
		UniqueTerms:   len(uniqueTerms(terms)),
		TermFreqs:     make(map[string]int),
		TermPositions: make(map[string][]int),
		Language:      vi.detectLanguage(filePath),
		FileType:      filepath.Ext(filePath),
		Size:          fileInfo.Size(),
	}

	// 统计词汇频次和位置
	for i, term := range terms {
		docInfo.TermFreqs[term]++
		if posList, ok := docInfo.TermPositions[term]; ok {
			if len(posList) < vi.Config.MaxPositions {
				docInfo.TermPositions[term] = append(posList, i)
			}
		} else {
			docInfo.TermPositions[term] = []int{i}
		}
	}

	// 更新倒排索引
	for term, freq := range docInfo.TermFreqs {
		if termInfo, exists := vi.TermToDocs[term]; exists {
			// 更新现有词汇信息
			termInfo.DocumentFreq++
			termInfo.TotalFreq += freq
			termInfo.Documents[filePath] = freq
			if positions, ok := docInfo.TermPositions[term]; ok {
				termInfo.Positions[filePath] = positions
			}
			termInfo.LastSeen = time.Now()
		} else {
			// 创建新词汇信息
			vi.TermToDocs[term] = &TermInfo{
				Term:         term,
				DocumentFreq: 1,
				TotalFreq:    freq,
				Documents:    map[string]int{filePath: freq},
				Positions:    map[string][]int{filePath: docInfo.TermPositions[term]},
				LastSeen:     time.Now(),
				Category:     vi.categorizeTerm(term),
			}
			result.NewTerms++
		}
	}

	// 添加文档信息
	vi.DocToTerms[filePath] = docInfo

	return nil
}

// shouldProcessFile 检查文件是否应该被处理
func (vi *VocabularyIndex) shouldProcessFile(filePath string) bool {
	if len(vi.Config.IncludePatterns) > 0 {
		inc := false
		for _, pattern := range vi.Config.IncludePatterns {
			if vi.matchPattern(filePath, pattern) {
				inc = true
				break
			}
		}
		if !inc {
			return false
		}
	}

	for _, pattern := range vi.Config.ExcludePatterns {
		if vi.matchPattern(filePath, pattern) {
			return false
		}
	}

	return true
}

func (vi *VocabularyIndex) matchPattern(filePath, pattern string) bool {
	if pattern == "" {
		return false
	}

	p := filepath.ToSlash(filePath)
	pat := filepath.ToSlash(pattern)

	if strings.HasSuffix(pat, "/*") {
		dir := strings.TrimSuffix(pat, "/*")
		if strings.Contains(p, dir+"/") {
			return true
		}
	}

	if !strings.Contains(pat, "/") {
		if matched, _ := filepath.Match(pat, filepath.Base(filePath)); matched {
			return true
		}
	}

	rx := regexp.QuoteMeta(pat)
	rx = strings.ReplaceAll(rx, "\\*\\*", ".*")
	rx = strings.ReplaceAll(rx, "\\*", "[^/]*")
	rx = strings.ReplaceAll(rx, "\\?", ".")
	rx = "^" + rx + "$"

	re, err := regexp.Compile(rx)
	if err != nil {
		return false
	}
	return re.MatchString(p)
}

// parseFile 解析文件内容，提取词汇
func (vi *VocabularyIndex) parseFile(filePath string) ([]string, map[string][]int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	var terms []string
	positions := make(map[string][]int)

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineTerms := vi.extractTerms(line)

		for _, term := range lineTerms {
			if vi.isValidTerm(term) {
				termIndex := len(terms)
				terms = append(terms, term)

				if posList, ok := positions[term]; ok {
					positions[term] = append(posList, termIndex)
				} else {
					positions[term] = []int{termIndex}
				}
			}
		}
		lineNumber++
	}

	return terms, positions, scanner.Err()
}

// extractTerms 从文本中提取词汇
func (vi *VocabularyIndex) extractTerms(text string) []string {
	// 简单的词汇提取，可以根据需要增强
	terms := strings.FieldsFunc(text, func(r rune) bool {
		// 分割非字母数字字符
		return !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') &&
			!(r >= '0' && r <= '9') && r != '_'
	})

	var result []string
	for _, term := range terms {
		term = strings.ToLower(term)
		if vi.isValidTerm(term) {
			result = append(result, term)
		}
	}

	return result
}

// isValidTerm 检查词汇是否有效
func (vi *VocabularyIndex) isValidTerm(term string) bool {
	if len(term) < vi.Config.MinTermLength || len(term) > vi.Config.MaxTermLength {
		return false
	}

	// 检查是否为停用词
	for _, stopWord := range vi.Config.StopWords {
		if term == stopWord {
			return false
		}
	}

	// 检查是否包含数字
	for _, r := range term {
		if r >= '0' && r <= '9' {
			return false // 纯数字或包含数字的词汇
		}
	}

	return true
}

// categorizeTerm 给词汇分类
func (vi *VocabularyIndex) categorizeTerm(term string) string {
	// 代码相关关键词
	codeKeywords := map[string]bool{
		"func": true, "function": true, "class": true, "method": true,
		"var": true, "let": true, "const": true, "if": true, "else": true,
		"for": true, "while": true, "return": true, "import": true,
		"export": true, "async": true, "await": true, "try": true, "catch": true,
	}

	if codeKeywords[term] {
		return "keyword"
	}

	// 检查是否可能是标识符（驼峰或下划线命名）
	if strings.Contains(term, "_") || (len(term) > 3 && isCamelCase(term)) {
		return "identifier"
	}

	// 检查长度，长词汇可能是复杂概念
	if len(term) > 8 {
		return "concept"
	}

	return "general"
}

// isCamelCase 检查是否为驼峰命名
func isCamelCase(term string) bool {
	if len(term) <= 1 {
		return false
	}

	hasUpper := false
	hasLower := false

	for _, r := range term {
		if r >= 'A' && r <= 'Z' {
			hasUpper = true
		} else if r >= 'a' && r <= 'z' {
			hasLower = true
		}
	}

	return hasUpper && hasLower
}

// detectLanguage 检测文件语言
func (vi *VocabularyIndex) detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	languageMap := map[string]string{
		".go":    "go",
		".rs":    "rust",
		".js":    "javascript",
		".ts":    "typescript",
		".py":    "python",
		".java":  "java",
		".cpp":   "cpp",
		".c":     "c",
		".h":     "c",
		".hpp":   "cpp",
		".cs":    "csharp",
		".php":   "php",
		".rb":    "ruby",
		".swift": "swift",
		".kt":    "kotlin",
		".md":    "markdown",
		".txt":   "text",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	return "unknown"
}

// removeDocument 从索引中移除文档
func (vi *VocabularyIndex) removeDocument(filePath string) {
	docInfo, exists := vi.DocToTerms[filePath]
	if !exists {
		return
	}

	// 从倒排索引中移除文档相关的词汇信息
	for term, freq := range docInfo.TermFreqs {
		if termInfo, exists := vi.TermToDocs[term]; exists {
			termInfo.DocumentFreq--
			termInfo.TotalFreq -= freq
			delete(termInfo.Documents, filePath)
			delete(termInfo.Positions, filePath)

			// 如果没有文档再包含该词汇，删除词汇
			if termInfo.DocumentFreq <= 0 {
				delete(vi.TermToDocs, term)
			}
		}
	}

	// 删除文档信息
	delete(vi.DocToTerms, filePath)
}

// calculateWeights 计算词汇权重（类似TF-IDF）
func (vi *VocabularyIndex) calculateWeights() {
	totalDocs := float64(len(vi.DocToTerms))
	totalTerms := float64(vi.GlobalStats.TotalTerms)
	if totalDocs <= 0 || totalTerms <= 0 {
		// 无有效统计，避免除零
		for _, termInfo := range vi.TermToDocs {
			termInfo.Weight = 0
		}
		return
	}

	for _, termInfo := range vi.TermToDocs {
		tf := float64(termInfo.TotalFreq) / totalTerms
		// 加 1 平滑，避免 log(0) 与极端比例
		idf := math.Log((totalDocs + 1) / (float64(termInfo.DocumentFreq) + 1))

		w := tf * idf
		if math.IsNaN(w) || math.IsInf(w, 0) || w < 0 {
			w = 0
		}

		// 根据分类调整权重
		switch termInfo.Category {
		case "keyword":
			w *= 1.2
		case "identifier":
			w *= 1.1
		case "concept":
			w *= 1.3
		}

		termInfo.Weight = w
	}
}

// updateGlobalStats 更新全局统计信息
func (vi *VocabularyIndex) updateGlobalStats() {
	vi.GlobalStats.TotalDocuments = len(vi.DocToTerms)
	vi.GlobalStats.UniqueTerms = len(vi.TermToDocs)

	totalTokens := 0
	totalTerms := 0
	var totalLength int64

	for _, docInfo := range vi.DocToTerms {
		totalTerms += docInfo.TotalTerms
		totalTokens += docInfo.UniqueTerms
		totalLength += docInfo.Size
	}

	vi.GlobalStats.TotalTerms = totalTerms
	vi.GlobalStats.TotalTokens = totalTokens

	if vi.GlobalStats.TotalDocuments > 0 {
		vi.GlobalStats.AvgDocLength = float64(totalLength) / float64(vi.GlobalStats.TotalDocuments)
	}

	vi.GlobalStats.LastUpdated = time.Now()

	// 词汇表大小与唯一词汇一致
	vi.GlobalStats.VocabularySize = vi.GlobalStats.UniqueTerms

	// 计算索引大小
	if info, err := os.Stat(vi.Config.IndexFile); err == nil {
		vi.GlobalStats.IndexSize = info.Size()
	}
}

// uniqueTerms 获取唯一词汇
func uniqueTerms(terms []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, term := range terms {
		if !seen[term] {
			seen[term] = true
			result = append(result, term)
		}
	}

	return result
}

// calculateFileHash 计算文件哈希
func (vi *VocabularyIndex) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
