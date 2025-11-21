package rag

import ("context"
	"fmt"
	"time")

// QuickSearch 最简单的搜索接口（本地模式）
func QuickSearch(query string) ([]*SearchResult, error) {
	rag, err := NewLocal(".")
	if err != nil {
		return nil, err
	}
	defer rag.Close()

	return rag.Query(context.Background(), query)
}

// QuickSearchWithTop 指定结果数量的快速搜索
func QuickSearchWithTop(query string, topK int) ([]*SearchResult, error) {
	rag, err := NewLocal(".")
	if err != nil {
		return nil, err
	}
	defer rag.Close()

	// 更新配置
	config := rag.rag.config
	config.TopK = topK

	return rag.Query(context.Background(), query)
}

// BatchSearch 批量搜索多个查询
func BatchSearch(queries []string) (map[string][]*SearchResult, error) {
	rag, err := NewLocal(".")
	if err != nil {
		return nil, err
	}
	defer rag.Close()

	results := make(map[string][]*SearchResult)
	for _, query := range queries {
		result, err := rag.Query(context.Background(), query)
		if err != nil {
			return nil, err
		}
		results[query] = result
	}

	return results, nil
}

// QuickSearchCloud 云模式快速搜索
func QuickSearchCloud(serviceURL, apiKey, query string) ([]*SearchResult, error) {
	rag, err := NewCloud(serviceURL, apiKey)
	if err != nil {
		return nil, err
	}
	defer rag.Close()

	return rag.Query(context.Background(), query)
}

// BatchSearchCloud 云模式批量搜索
func BatchSearchCloud(serviceURL, apiKey string, queries []string) (map[string][]*SearchResult, error) {
	rag, err := NewCloud(serviceURL, apiKey)
	if err != nil {
		return nil, err
	}
	defer rag.Close()

	results := make(map[string][]*SearchResult)
	for _, query := range queries {
		result, err := rag.Query(context.Background(), query)
		if err != nil {
			return nil, err
		}
		results[query] = result
	}

	return results, nil
}

// PrintSimpleResults 简单打印结果
func PrintSimpleResults(results []*SearchResult) {
	if len(results) == 0 {
		fmt.Printf("未找到相关结果\n")
		return
	}

	fmt.Printf("找到 %d 个结果:\n", len(results))
	for i, result := range results {
		fmt.Printf("%d. %s:%d (score=%.3f)\n", i+1, result.File, result.Line, result.Score)
		if len(result.Snippet) > 100 {
			fmt.Printf("   %s...\n", result.Snippet[:100])
		} else {
			fmt.Printf("   %s\n", result.Snippet)
		}
	}
}

// GetVocabularyStats 获取词表统计信息
func GetVocabularyStats() (map[string]interface{}, error) {
	vocabMgr := NewVocabularyManager()
	err := vocabMgr.EnsureVocabulary(false, false, 0)
	if err != nil {
		return nil, err
	}

	return vocabMgr.GetVocabularyStats(), nil
}

// BuildVocabulary 构建词表
func BuildVocabulary() error {
	vocabMgr := NewVocabularyManager()
	return vocabMgr.EnsureVocabulary(true, false, 0)
}

// UpdateVocabulary 更新词表
func UpdateVocabulary() error {
	vocabMgr := NewVocabularyManager()
	return vocabMgr.EnsureVocabulary(false, true, 0)
}

// ExpandQuery 扩展查询
func ExpandQuery(query string, maxExpansions int) ([]string, error) {
	vocabMgr := NewVocabularyManager()
	err := vocabMgr.EnsureVocabulary(true, true, 24)
	if err != nil {
		return nil, err
	}

	return vocabMgr.ExpandQuery(query, maxExpansions)
}

// GetGlobalStats 获取全局统计信息
func GetGlobalStats() (*GlobalStats, error) {
	// 获取词表统计
	vocabStats, err := GetVocabularyStats()
	if err != nil {
		vocabStats = map[string]interface{}{"error": err.Error()}
	}

	// 获取本地搜索统计
	rag, err := NewLocal(".")
	if err != nil {
		return nil, err
	}
	defer rag.Close()

	ragStats, err := rag.GetStats()
	if err != nil {
		return nil, err
	}

	return &GlobalStats{
		Mode:                ragStats.Mode,
		VocabularyTerms:     ragStats.VocabularyTerms,
		VocabularyDocs:      ragStats.VocabularyDocs,
		VocabularyLastUpdated: ragStats.VocabularyLastUpdated,
		VocabularyDetails:   vocabStats,
		CloudStats:          ragStats.CloudStats,
	}, nil
}

// GlobalStats 全局统计信息
type GlobalStats struct {
	Mode                string                 `json:"mode"`
	VocabularyTerms     int                    `json:"vocabulary_terms"`
	VocabularyDocs      int                    `json:"vocabulary_docs"`
	VocabularyLastUpdated time.Time             `json:"vocabulary_last_updated"`
	VocabularyDetails   map[string]interface{} `json:"vocabulary_details"`
	CloudStats          *CloudStats            `json:"cloud_stats,omitempty"`
}