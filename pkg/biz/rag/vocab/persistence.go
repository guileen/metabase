package vocab

import ("encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time")

// Save 保存词表索引到文件
func (vi *VocabularyIndex) Save() error {
	vi.mutex.Lock()
	defer vi.mutex.Unlock()
	return vi.saveIndex()
}

// saveIndex 内部保存方法（不加锁）
func (vi *VocabularyIndex) saveIndex() error {

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(vi.Config.IndexFile), 0755); err != nil {
		return fmt.Errorf("failed to create index directory: %w", err)
	}

	// 创建临时文件
	tempFile := vi.Config.IndexFile + ".tmp"

	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	// 使用 gob 编码器（比 JSON 更快更紧凑）
	encoder := gob.NewEncoder(file)

	if err := encoder.Encode(vi); err != nil {
		return fmt.Errorf("failed to encode index: %w", err)
	}

	// 原子性重命名
	if err := os.Rename(tempFile, vi.Config.IndexFile); err != nil {
		os.Remove(tempFile) // 清理临时文件
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// 同时保存 JSON 版本用于调试
	jsonFile := vi.Config.IndexFile + ".json"
	if err := vi.saveJSON(jsonFile); err != nil {
		// JSON 保存失败不影响主要功能
		fmt.Printf("[Vocab] Warning: failed to save JSON version: %v\n", err)
	}

	return nil
}

// saveJSON 保存 JSON 格式的索引（用于调试和可视化）
func (vi *VocabularyIndex) saveJSON(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	return encoder.Encode(vi)
}

// Load 从文件加载词表索引
func LoadVocabularyIndex(config *Config) (*VocabularyIndex, error) {
	if config == nil {
		config = CreateDefaultConfig()
	}

	// 检查索引文件是否存在
	if _, err := os.Stat(config.IndexFile); os.IsNotExist(err) {
		// 文件不存在，创建新索引
		return NewVocabularyIndex(config), nil
	}

	// 打开索引文件
	file, err := os.Open(config.IndexFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open index file: %w", err)
	}
	defer file.Close()

	// 解码索引
	var index VocabularyIndex
	decoder := gob.NewDecoder(file)

	if err := decoder.Decode(&index); err != nil {
		// 如果 gob 解码失败，尝试 JSON 格式（向后兼容）
		if err := index.tryLoadJSON(config.IndexFile); err != nil {
			return nil, fmt.Errorf("failed to decode index (both gob and json failed): %w", err)
		}
	}

	// 更新配置（可能已经改变）
	index.Config = config

	// 初始化 mutex（gob 不会序列化这个字段）
	index.mutex = sync.RWMutex{}

	index.updateGlobalStats()
	index.calculateWeights()

	fmt.Printf("[Vocab] Loaded vocabulary index: %d documents, %d unique terms\n",
		index.GlobalStats.TotalDocuments, index.GlobalStats.UniqueTerms)

	return &index, nil
}

// tryLoadJSON 尝试从 JSON 文件加载索引
func (vi *VocabularyIndex) tryLoadJSON(filename string) error {
	jsonFile := filename + ".json"
	file, err := os.Open(jsonFile)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(vi)
}

// IncrementalUpdate 增量更新索引
func (vi *VocabularyIndex) IncrementalUpdate(filePaths []string) (*UpdateResult, error) {
	startTime := time.Now()
	result := &UpdateResult{}

	vi.mutex.Lock()
	defer vi.mutex.Unlock()

	fmt.Printf("[Vocab] Starting incremental update with %d files\n", len(filePaths))

	// 获取所有现有文件路径
	existingFiles := make(map[string]bool)
	for filePath := range vi.DocToTerms {
		existingFiles[filePath] = true
	}

	// 处理提供的文件
	for _, filePath := range filePaths {
		if err := vi.processFile(filePath, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error processing %s: %v", filePath, err))
			continue
		}

		// 从待删除列表中移除
		delete(existingFiles, filePath)
	}

	// 删除不再存在的文件
	for filePath := range existingFiles {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			vi.removeDocument(filePath)
			result.DeletedFiles++
		}
	}

    vi.updateGlobalStats()
    vi.calculateWeights()

	// 更新元数据
	vi.Metadata.LastUpdate = time.Now()
	vi.Metadata.TotalUpdates++
	result.Duration = time.Since(startTime)

	// 保存索引
	if err := vi.Save(); err != nil {
		return result, fmt.Errorf("failed to save updated index: %w", err)
	}

	fmt.Printf("[Vocab] Incremental update completed in %v\n", result.Duration)
	fmt.Printf("[Vocab] Added: %d, Updated: %d, Deleted: %d files\n",
		result.AddedFiles, result.UpdatedFiles, result.DeletedFiles)

	return result, nil
}

// GetStats 获取索引统计信息
func (vi *VocabularyIndex) GetStats() *GlobalStats {
	vi.mutex.RLock()
	defer vi.mutex.RUnlock()

	// 返回统计信息的副本
	stats := *vi.GlobalStats
	return &stats
}

// GetTopTerms 获取高频词汇
func (vi *VocabularyIndex) GetTopTerms(limit int, category string) []*TermInfo {
	vi.mutex.RLock()
	defer vi.mutex.RUnlock()

	var terms []*TermInfo
	for _, termInfo := range vi.TermToDocs {
		if category == "" || termInfo.Category == category {
			terms = append(terms, termInfo)
		}
	}

	// 按权重排序
	sort.Slice(terms, func(i, j int) bool {
		return terms[i].Weight > terms[j].Weight
	})

	if len(terms) > limit {
		terms = terms[:limit]
	}

	return terms
}

// GetDocumentTerms 获取文档的词汇信息
func (vi *VocabularyIndex) GetDocumentTerms(filePath string) *DocumentInfo {
	vi.mutex.RLock()
	defer vi.mutex.RUnlock()

	if docInfo, exists := vi.DocToTerms[filePath]; exists {
		// 返回副本
		docCopy := *docInfo
		return &docCopy
	}

	return nil
}

// FindSimilarTerms 查找相似词汇
func (vi *VocabularyIndex) FindSimilarTerms(term string, limit int) []string {
	vi.mutex.RLock()
	defer vi.mutex.RUnlock()

	termInfo, exists := vi.TermToDocs[term]
	if !exists {
		return nil
	}

	// 查找共享文档的词汇
	sharedDocs := make(map[string]bool)
	for doc := range termInfo.Documents {
		sharedDocs[doc] = true
	}

	type similarTerm struct {
		term  string
		score float64
	}

	var similar []similarTerm

	for otherTerm, otherInfo := range vi.TermToDocs {
		if otherTerm == term {
			continue
		}

		// 计算 Jaccard 相似度
		sharedDocCount := 0
		for doc := range otherInfo.Documents {
			if sharedDocs[doc] {
				sharedDocCount++
			}
		}

		if sharedDocCount > 0 {
			totalDocs := len(termInfo.Documents) + len(otherInfo.Documents) - sharedDocCount
			jaccard := float64(sharedDocCount) / float64(totalDocs)

			similar = append(similar, similarTerm{
				term:  otherTerm,
				score: jaccard,
			})
		}
	}

	// 按相似度排序
	sort.Slice(similar, func(i, j int) bool {
		return similar[i].score > similar[j].score
	})

	result := make([]string, 0, limit)
	for i, st := range similar {
		if i >= limit {
			break
		}
		result = append(result, st.term)
	}

	return result
}

// ExpandQuery 扩展查询
func (vi *VocabularyIndex) ExpandQuery(query string, maxExpansions int) *QueryExpansionResult {
	vi.mutex.RLock()
	defer vi.mutex.RUnlock()

	result := &QueryExpansionResult{
		OriginalTerms: vi.extractTerms(query),
		ExpandedTerms: make([]string, 0),
		SimilarTerms:  make(map[string][]string),
		CategoryTerms: make(map[string][]string),
		WeightedTerms: make(map[string]float64),
	}

	termSet := make(map[string]bool)
	for _, term := range result.OriginalTerms {
		termSet[term] = true

		// 查找相似词汇
		similar := vi.findSimilarTermsInternal(term, 5)
		if len(similar) > 0 {
			result.SimilarTerms[term] = similar
			for _, simTerm := range similar {
				if !termSet[simTerm] {
					termSet[simTerm] = true
					result.ExpandedTerms = append(result.ExpandedTerms, simTerm)
				}
			}
		}
	}

	// 按分类添加词汇
	categories := map[string][]string{
		"keyword":    {},
		"identifier": {},
		"concept":    {},
	}

	for term, termInfo := range vi.TermToDocs {
		if termInfo.Category != "" {
			if categoryTerms, exists := categories[termInfo.Category]; exists {
				if len(categoryTerms) < 10 { // 每个分类最多10个词
					categories[termInfo.Category] = append(categoryTerms, term)
				}
			}
		}
	}

	result.CategoryTerms = categories

	// 按权重排序扩展词汇
	type weightedTerm struct {
		term   string
		weight float64
	}

	var weighted []weightedTerm
	for term := range termSet {
		if termInfo, exists := vi.TermToDocs[term]; exists {
			weighted = append(weighted, weightedTerm{
				term:   term,
				weight: termInfo.Weight,
			})
		}
	}

	sort.Slice(weighted, func(i, j int) bool {
		return weighted[i].weight > weighted[j].weight
	})

	// 限制扩展词汇数量
	maxTerms := maxExpansions
	if maxTerms <= 0 {
		maxTerms = 20
	}

	for i, wt := range weighted {
		if i >= maxTerms {
			break
		}
		result.WeightedTerms[wt.term] = wt.weight
	}

	return result
}

// findSimilarTermsInternal 内部查找相似词汇方法（无需加锁）
func (vi *VocabularyIndex) findSimilarTermsInternal(term string, limit int) []string {
	termInfo, exists := vi.TermToDocs[term]
	if !exists {
		return nil
	}

	// 查找共享文档的词汇
	sharedDocs := make(map[string]bool)
	for doc := range termInfo.Documents {
		sharedDocs[doc] = true
	}

	type similarTerm struct {
		term  string
		score float64
	}

	var similar []similarTerm

	for otherTerm, otherInfo := range vi.TermToDocs {
		if otherTerm == term {
			continue
		}

		// 计算 Jaccard 相似度
		sharedDocCount := 0
		for doc := range otherInfo.Documents {
			if sharedDocs[doc] {
				sharedDocCount++
			}
		}

		if sharedDocCount > 0 {
			totalDocs := len(termInfo.Documents) + len(otherInfo.Documents) - sharedDocCount
			jaccard := float64(sharedDocCount) / float64(totalDocs)

			similar = append(similar, similarTerm{
				term:  otherTerm,
				score: jaccard,
			})
		}
	}

	// 按相似度排序
	sort.Slice(similar, func(i, j int) bool {
		return similar[i].score > similar[j].score
	})

	result := make([]string, 0, limit)
	for i, st := range similar {
		if i >= limit {
			break
		}
		result = append(result, st.term)
	}

	return result
}

// GetTermInfo 获取词汇信息
func (vi *VocabularyIndex) GetTermInfo(term string) *TermInfo {
	vi.mutex.RLock()
	defer vi.mutex.RUnlock()

	if termInfo, exists := vi.TermToDocs[term]; exists {
		// 返回副本
		copy := *termInfo
		return &copy
	}

	return nil
}

// GetDocumentsContainingTerm 获取包含指定词汇的文档
func (vi *VocabularyIndex) GetDocumentsContainingTerm(term string) []string {
	vi.mutex.RLock()
	defer vi.mutex.RUnlock()

	termInfo, exists := vi.TermToDocs[term]
	if !exists {
		return nil
	}

	docs := make([]string, 0, len(termInfo.Documents))
	for doc := range termInfo.Documents {
		docs = append(docs, doc)
	}

	return docs
}

// CleanupOldTerms 清理旧词汇（基于最后见到的時間）
func (vi *VocabularyIndex) CleanupOldTerms(maxAge time.Duration) int {
	vi.mutex.Lock()
	defer vi.mutex.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for term, termInfo := range vi.TermToDocs {
		if termInfo.LastSeen.Before(cutoff) && termInfo.DocumentFreq <= 2 {
			// 只删除很少文档中出现的旧词汇
			delete(vi.TermToDocs, term)
			removed++

			// 从文档中删除该词汇
			for doc := range termInfo.Documents {
				if docInfo, exists := vi.DocToTerms[doc]; exists {
					delete(docInfo.TermFreqs, term)
					delete(docInfo.TermPositions, term)
				}
			}
		}
	}

	if removed > 0 {
		vi.calculateWeights()
		vi.updateGlobalStats()
		vi.Save()
	}

	fmt.Printf("[Vocab] Cleaned up %d old terms\n", removed)
	return removed
}

// OptimizeIndex 优化索引（压缩和重新组织）
func (vi *VocabularyIndex) OptimizeIndex() error {
	vi.mutex.Lock()
	defer vi.mutex.Unlock()

	fmt.Printf("[Vocab] Optimizing vocabulary index...\n")

	// 重新计算所有权重
	vi.calculateWeights()
	vi.updateGlobalStats()

	// 压缩位置信息（限制每个词汇的位置记录数）
	for _, termInfo := range vi.TermToDocs {
		for doc, positions := range termInfo.Positions {
			if len(positions) > vi.Config.MaxPositions {
				// 只保留前 N 个位置
				termInfo.Positions[doc] = positions[:vi.Config.MaxPositions]
			}
		}

		// 限制每个词汇的文档数
		if len(termInfo.Documents) > vi.Config.MaxDocsPerTerm {
			// 按频次排序，只保留前 N 个文档
			type docFreq struct {
				doc  string
				freq int
			}

			var sortedDocs []docFreq
			for doc, freq := range termInfo.Documents {
				sortedDocs = append(sortedDocs, docFreq{doc, freq})
			}

			sort.Slice(sortedDocs, func(i, j int) bool {
				return sortedDocs[i].freq > sortedDocs[j].freq
			})

			// 重建 Documents 和 Positions 映射
			newDocuments := make(map[string]int)
			newPositions := make(map[string][]int)

			for i, df := range sortedDocs {
				if i >= vi.Config.MaxDocsPerTerm {
					break
				}
				newDocuments[df.doc] = df.freq
				if pos, exists := termInfo.Positions[df.doc]; exists {
					newPositions[df.doc] = pos
				}
			}

			termInfo.Documents = newDocuments
			termInfo.Positions = newPositions
		}
	}

	// 更新元数据
	vi.Metadata.LastUpdate = time.Now()

	// 保存优化后的索引
	return vi.Save()
}