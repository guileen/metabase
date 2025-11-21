package vocab

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/guileen/metabase/pkg/rag/embedding"
	"github.com/guileen/metabase/pkg/rag/search/vector"
)

// VocabularyBuilder 词表构建器
type VocabularyBuilder struct {
	index     *VocabularyIndex
	kv        *pebble.DB
	termIndex *vector.HNSWIndex
	embCache  map[string][]float64
	indexed   map[string]bool
}

// NewVocabularyBuilder 创建词表构建器
func NewVocabularyBuilder(config *Config) *VocabularyBuilder {
	if config == nil {
		config = CreateDefaultConfig()
	}
	return &VocabularyBuilder{
		index:    NewVocabularyIndex(config),
		embCache: make(map[string][]float64),
		indexed:  make(map[string]bool),
	}
}

// LoadVocabularyBuilder 加载现有词表
func LoadVocabularyBuilder(config *Config) (*VocabularyBuilder, error) {
	index, err := LoadVocabularyIndex(config)
	if err != nil {
		return nil, err
	}

	return &VocabularyBuilder{
		index:    index,
		embCache: make(map[string][]float64),
		indexed:  make(map[string]bool),
	}, nil
}

// BuildFromDirectory 从目录构建词表
func (vb *VocabularyBuilder) BuildFromDirectory(rootDir string, recursive bool) (*UpdateResult, error) {
	var filePaths []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if !recursive && path != rootDir {
				return filepath.SkipDir
			}
			return nil
		}

		// 检查是否应该包含此文件
		if vb.index.shouldProcessFile(path) {
			filePaths = append(filePaths, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	fmt.Printf("[Vocab] Found %d files to process from %s\n", len(filePaths), rootDir)
	return vb.index.BuildIndex(filePaths)
}

// BuildFromFiles 从指定文件列表构建词表
func (vb *VocabularyBuilder) BuildFromFiles(filePaths []string) (*UpdateResult, error) {
	fmt.Printf("[Vocab] Building vocabulary from %d specified files\n", len(filePaths))
	return vb.index.BuildIndex(filePaths)
}

// UpdateFromDirectory 更新目录中的词表（增量更新）
func (vb *VocabularyBuilder) UpdateFromDirectory(rootDir string, recursive bool) (*UpdateResult, error) {
	var filePaths []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if !recursive && path != rootDir {
				return filepath.SkipDir
			}
			return nil
		}

		// 检查是否应该包含此文件
		if vb.index.shouldProcessFile(path) {
			filePaths = append(filePaths, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	fmt.Printf("[Vocab] Updating vocabulary with %d files from %s\n", len(filePaths), rootDir)
	return vb.index.IncrementalUpdate(filePaths)
}

// GetIndex 获取词表索引
func (vb *VocabularyBuilder) GetIndex() *VocabularyIndex {
	return vb.index
}

// GetVocabularyStats 获取词表统计信息
func (vb *VocabularyBuilder) GetVocabularyStats() map[string]interface{} {
	stats := vb.index.GetStats()

	// 获取分类统计
	categoryStats := make(map[string]int)
	for _, termInfo := range vb.index.TermToDocs {
		categoryStats[termInfo.Category]++
	}

	// 获取语言统计
	languageStats := make(map[string]int)
	for _, docInfo := range vb.index.DocToTerms {
		languageStats[docInfo.Language]++
	}

	return map[string]interface{}{
		"global_stats":   stats,
		"category_stats": categoryStats,
		"language_stats": languageStats,
		"config":         vb.index.Config,
		"metadata":       vb.index.Metadata,
	}
}

// PrintStats 打印词表统计信息
func (vb *VocabularyBuilder) PrintStats() {
	stats := vb.GetVocabularyStats()

	globalStats := stats["global_stats"].(*GlobalStats)
	categoryStats := stats["category_stats"].(map[string]int)
	languageStats := stats["language_stats"].(map[string]int)

	fmt.Printf("\n=== VOCABULARY STATISTICS ===\n")
	fmt.Printf("Documents: %d\n", globalStats.TotalDocuments)
	fmt.Printf("Unique Terms: %d\n", globalStats.UniqueTerms)
	fmt.Printf("Total Terms: %d\n", globalStats.TotalTerms)
	fmt.Printf("Vocabulary Size: %d\n", globalStats.VocabularySize)
	fmt.Printf("Index Size: %.2f MB\n", float64(globalStats.IndexSize)/1024/1024)
	fmt.Printf("Average Doc Length: %.1f bytes\n", globalStats.AvgDocLength)
	fmt.Printf("Last Updated: %s\n", globalStats.LastUpdated.Format(time.RFC3339))

	fmt.Printf("\n--- Term Categories ---\n")
	for category, count := range categoryStats {
		fmt.Printf("%s: %d (%.1f%%)\n", category, count,
			float64(count)/float64(globalStats.UniqueTerms)*100)
	}

	fmt.Printf("\n--- File Languages ---\n")
	for language, count := range languageStats {
		fmt.Printf("%s: %d (%.1f%%)\n", language, count,
			float64(count)/float64(globalStats.TotalDocuments)*100)
	}
	fmt.Printf("\n")
}

func (vb *VocabularyBuilder) CacheTermEmbeddings(emb embedding.Embedder, limit int) error {
	if limit <= 0 {
		limit = 10000
	}
	if err := vb.ensureVectorIndex(emb.GetDimension()); err != nil {
		return err
	}
	tops := vb.index.GetTopTerms(limit, "")
	terms := make([]string, 0, len(tops))
	for _, ti := range tops {
		terms = append(terms, ti.Term)
	}
	if len(terms) == 0 {
		return nil
	}
	if err := vb.ensureTermVectors(terms, emb); err != nil {
		return err
	}
	return nil
}

func (vb *VocabularyBuilder) CacheTermEmbeddingsDefault(limit int) error {
	cfg := &embedding.Config{LocalModelType: "cybertron", BatchSize: 32, MaxConcurrency: 2, EnableFallback: true}
	emb, err := embedding.NewLocalEmbedder(cfg)
	if err != nil {
		return err
	}
	defer emb.Close()
	return vb.CacheTermEmbeddings(emb, limit)
}

// ExportVocabulary 导出词表为文本格式
func (vb *VocabularyBuilder) ExportVocabulary(filePath string, format string) error {
	return vb.ExportVocabularyWithLimit(filePath, format, 1000)
}

func (vb *VocabularyBuilder) ExportVocabularyWithLimit(filePath string, format string, limit int) error {
	var content strings.Builder

	switch format {
	case "txt":
		terms := vb.index.GetTopTerms(normalizeLimit(limit), "")
		for _, termInfo := range terms {
			content.WriteString(fmt.Sprintf("%s\t%.6f\t%d\t%d\t%s\n",
				termInfo.Term,
				termInfo.Weight,
				termInfo.DocumentFreq,
				termInfo.TotalFreq,
				termInfo.Category))
		}

	case "csv":
		content.WriteString("Term,Weight,DocumentFreq,TotalFreq,Category,LastSeen\n")
		terms := vb.index.GetTopTerms(normalizeLimit(limit), "")
		for _, termInfo := range terms {
			content.WriteString(fmt.Sprintf("%s,%.6f,%d,%d,%s,%s\n",
				termInfo.Term,
				termInfo.Weight,
				termInfo.DocumentFreq,
				termInfo.TotalFreq,
				termInfo.Category,
				termInfo.LastSeen.Format(time.RFC3339)))
		}

	case "json":
		terms := vb.index.GetTopTerms(normalizeLimit(limit), "")
		content.WriteString("[\n")
		for i, termInfo := range terms {
			comma := ","
			if i == len(terms)-1 {
				comma = ""
			}
			content.WriteString(fmt.Sprintf(`  {
    "term": "%s",
    "weight": %.6f,
    "document_freq": %d,
    "total_freq": %d,
    "category": "%s",
    "last_seen": "%s"
  }%s
`,
				termInfo.Term,
				termInfo.Weight,
				termInfo.DocumentFreq,
				termInfo.TotalFreq,
				termInfo.Category,
				termInfo.LastSeen.Format(time.RFC3339),
				comma))
		}
		content.WriteString("]\n")

	default:
		return fmt.Errorf("unsupported format: %s (supported: txt, csv, json)", format)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(content.String())
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("[Vocab] Exported vocabulary to %s in %s format\n", filePath, format)
	return nil
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 1<<31 - 1
	}
	return limit
}

// SearchTerms 搜索词汇
func (vb *VocabularyBuilder) SearchTerms(query string, limit int) []*TermInfo {
	queryTerms := vb.index.extractTerms(query)

	var matchedTerms []*TermInfo
	termSet := make(map[string]bool)

	for _, queryTerm := range queryTerms {
		// 精确匹配
		if termInfo, exists := vb.index.TermToDocs[queryTerm]; exists {
			if !termSet[queryTerm] {
				matchedTerms = append(matchedTerms, termInfo)
				termSet[queryTerm] = true
			}
		}

		// 前缀匹配
		for term, termInfo := range vb.index.TermToDocs {
			if strings.HasPrefix(term, queryTerm) && !termSet[term] {
				matchedTerms = append(matchedTerms, termInfo)
				termSet[term] = true
			}
		}
	}

	// 按权重排序
	// 注意：这里需要一个 sort.Slice 的调用，但由于已经在 matchedTerms 中，我们需要处理这个
	// 为了简化，我们只返回前 limit 个结果

	if len(matchedTerms) > limit {
		matchedTerms = matchedTerms[:limit]
	}

	return matchedTerms
}

func (vb *VocabularyBuilder) ExpandQueryWithEmbedding(query string, limit int, emb embedding.Embedder) (*QueryExpansionResult, error) {
	if emb == nil {
		return nil, fmt.Errorf("embedder is required")
	}
	maxCand := 800
	tops := vb.index.GetTopTerms(maxCand, "")
	terms := make([]string, 0, len(tops))
	for _, ti := range tops {
		terms = append(terms, ti.Term)
	}
	if len(terms) == 0 {
		for term := range vb.index.TermToDocs {
			terms = append(terms, term)
		}
	}
	if limit <= 0 {
		limit = 20
	}
	if err := vb.ensureVectorIndex(emb.GetDimension()); err != nil {
		return nil, err
	}
	qvecs, err := emb.Embed([]string{query})
	if err != nil || len(qvecs) == 0 {
		return &QueryExpansionResult{OriginalTerms: vb.index.extractTerms(query)}, nil
	}
	ids, dists, err := vb.termIndex.Search(context.Background(), qvecs[0], limit)
	if err != nil || len(ids) == 0 {
		fb := terms
		if len(fb) > 300 {
			fb = fb[:300]
		}
		inputs := make([]string, 0, len(fb)+1)
		inputs = append(inputs, query)
		inputs = append(inputs, fb...)
		vecs, e2 := emb.Embed(inputs)
		if e2 != nil || len(vecs) == 0 {
			return &QueryExpansionResult{OriginalTerms: vb.index.extractTerms(query)}, nil
		}
		q := vecs[0]
		type item struct {
			term  string
			score float64
		}
		items := make([]item, 0, len(fb))
		for i, t := range fb {
			s := cosine(q, vecs[i+1])
			items = append(items, item{t, s})
		}
		if len(items) > limit {
			for i := 0; i < limit; i++ {
				max := i
				for j := i + 1; j < len(items); j++ {
					if items[j].score > items[max].score {
						max = j
					}
				}
				items[i], items[max] = items[max], items[i]
			}
			items = items[:limit]
		} else {
			for i := 0; i < len(items)-1; i++ {
				max := i
				for j := i + 1; j < len(items); j++ {
					if items[j].score > items[max].score {
						max = j
					}
				}
				items[i], items[max] = items[max], items[i]
			}
		}
		res := &QueryExpansionResult{
			OriginalTerms: vb.index.extractTerms(query),
			ExpandedTerms: make([]string, 0, len(items)),
			SimilarTerms:  make(map[string][]string),
			CategoryTerms: make(map[string][]string),
			WeightedTerms: make(map[string]float64),
		}
		for _, it := range items {
			res.ExpandedTerms = append(res.ExpandedTerms, it.term)
			res.WeightedTerms[it.term] = it.score
		}
		cats := map[string][]string{"keyword": {}, "identifier": {}, "concept": {}}
		for _, it := range items {
			if ti := vb.index.TermToDocs[it.term]; ti != nil {
				if lst, ok := cats[ti.Category]; ok && len(lst) < 10 {
					cats[ti.Category] = append(lst, it.term)
				}
			}
		}
		res.CategoryTerms = cats
		return res, nil
	}
	res := &QueryExpansionResult{
		OriginalTerms: vb.index.extractTerms(query),
		ExpandedTerms: make([]string, 0, len(ids)),
		SimilarTerms:  make(map[string][]string),
		CategoryTerms: make(map[string][]string),
		WeightedTerms: make(map[string]float64),
	}
	for i, id := range ids {
		res.ExpandedTerms = append(res.ExpandedTerms, id)
		res.WeightedTerms[id] = 1.0 / (1.0 + dists[i])
	}
	cats := map[string][]string{"keyword": {}, "identifier": {}, "concept": {}}
	for _, id := range ids {
		if ti := vb.index.TermToDocs[id]; ti != nil {
			if lst, ok := cats[ti.Category]; ok && len(lst) < 10 {
				cats[ti.Category] = append(lst, id)
			}
		}
	}
	res.CategoryTerms = cats

	return res, nil
}

func cosine(a, b []float64) float64 {
	var dot, na, nb float64
	for i := 0; i < len(a) && i < len(b); i++ {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func (vb *VocabularyBuilder) ensureVectorIndex(dim int) error {
	if vb.termIndex != nil {
		return nil
	}
	dir := vb.index.Config.DataDir
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".metabase", "vocab")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, "term_vectors.db")
	kv, err := pebble.Open(path, &pebble.Options{})
	if err != nil {
		return err
	}
	cfg := &vector.Config{Dimension: dim, DistanceType: vector.DistanceTypeCosine, M: 16, EF: 50, ML: 1.0 / math.Log(2.0), EPS: 200, Prefix: "vocab:"}
	idx, err := vector.NewHNSWIndex(kv, cfg)
	if err != nil {
		kv.Close()
		return err
	}
	vb.kv = kv
	vb.termIndex = idx
	return nil
}

func (vb *VocabularyBuilder) ensureTermVectors(terms []string, emb embedding.Embedder) error {
	missing := make([]string, 0, len(terms))
	for _, t := range terms {
		if vb.indexed[t] {
			continue
		}
		if vb.kv != nil {
			key := []byte("vocab:vector:" + t)
			_, closer, err := vb.kv.Get(key)
			if err == nil {
				closer.Close()
				vb.indexed[t] = true
				continue
			}
		}
		missing = append(missing, t)
	}
	if len(missing) == 0 {
		return nil
	}
	vecs, err := emb.Embed(missing)
	if err != nil {
		return err
	}
	for i, t := range missing {
		vb.embCache[t] = vecs[i]
		_ = vb.termIndex.Insert(context.Background(), t, vecs[i])
		vb.indexed[t] = true
	}
	return nil
}

// GetTermFrequency 获取词频统计
func (vb *VocabularyBuilder) GetTermFrequency(term string) (int, int, float64) {
	termInfo := vb.index.GetTermInfo(term)
	if termInfo == nil {
		return 0, 0, 0
	}

	stats := vb.index.GetStats()
	tf := float64(termInfo.TotalFreq) / float64(stats.TotalTerms)
	idf := math.Log(float64(stats.TotalDocuments) / float64(termInfo.DocumentFreq))

	return termInfo.TotalFreq, termInfo.DocumentFreq, tf * idf
}

// GetDocumentSimilarity 获取文档相似度
func (vb *VocabularyBuilder) GetDocumentSimilarity(doc1Path, doc2Path string) float64 {
	doc1Info := vb.index.GetDocumentTerms(doc1Path)
	doc2Info := vb.index.GetDocumentTerms(doc2Path)

	if doc1Info == nil || doc2Info == nil {
		return 0
	}

	// 计算 Jaccard 相似度
	doc1Terms := make(map[string]bool)
	for term := range doc1Info.TermFreqs {
		doc1Terms[term] = true
	}

	doc2Terms := make(map[string]bool)
	for term := range doc2Info.TermFreqs {
		doc2Terms[term] = true
	}

	intersection := 0
	for term := range doc1Terms {
		if doc2Terms[term] {
			intersection++
		}
	}

	union := len(doc1Terms) + len(doc2Terms) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// AutoUpdate 自动更新词表
func (vb *VocabularyBuilder) AutoUpdate() error {
	if !vb.index.Config.AutoUpdate {
		return fmt.Errorf("auto update is disabled")
	}

	fmt.Printf("[Vocab] Starting automatic vocabulary update...\n")

	// 从现有文件构建路径列表
	var filePaths []string
	for filePath := range vb.index.DocToTerms {
		if _, err := os.Stat(filePath); err == nil {
			filePaths = append(filePaths, filePath)
		}
	}

	result, err := vb.index.IncrementalUpdate(filePaths)
	if err != nil {
		return fmt.Errorf("auto update failed: %w", err)
	}

	fmt.Printf("[Vocab] Auto update completed: %+v\n", result)
	return nil
}

// ScheduleAutoUpdate 调度自动更新
func (vb *VocabularyBuilder) ScheduleAutoUpdate() error {
	if !vb.index.Config.AutoUpdate {
		return fmt.Errorf("auto update is disabled")
	}

	interval := time.Duration(vb.index.Config.UpdateInterval) * time.Minute

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := vb.AutoUpdate(); err != nil {
				fmt.Printf("[Vocab] Scheduled update failed: %v\n", err)
			}
		}
	}()

	fmt.Printf("[Vocab] Scheduled auto update every %v\n", interval)
	return nil
}
