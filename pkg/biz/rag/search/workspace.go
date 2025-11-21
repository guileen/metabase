package search

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/guileen/metabase/pkg/biz/rag/llm"
	"github.com/guileen/metabase/pkg/biz/rag/search/engine"
	"github.com/guileen/metabase/pkg/infra/skills"
)

// WorkspaceSearch provides enhanced local workspace search capabilities
type WorkspaceSearch struct {
	rootPath        string
	searchEngine    *engine.Engine
	templateManager *skills.TemplateManager
	config          *WorkspaceConfig
	indexer         *WorkspaceIndexer
	mutex           sync.RWMutex

	// Cache for recently accessed files
	fileCache map[string]*CachedFile
	cacheTTL  time.Duration
}

// WorkspaceConfig holds configuration for workspace search
type WorkspaceConfig struct {
	RootPath         string
	IndexInterval    time.Duration
	MaxFileSize      int64
	ExcludePatterns  []string
	IncludePatterns  []string
	EnableEmbeddings bool
	EmbeddingModel   string
	EnableSkills     bool
	CacheEnabled     bool
	CacheTTL         time.Duration
	ParallelIndexing bool
	MaxIndexWorkers  int
}

// CachedFile represents a cached file with metadata
type CachedFile struct {
	Path         string
	Content      string
	LastModified time.Time
	Size         int64
	Embedding    []float64
	Metadata     map[string]interface{}
}

// SearchResult represents a search result with enhanced metadata
type SearchResult struct {
	*engine.Document
	Score       float64                `json:"score"`
	Relevance   string                 `json:"relevance"`
	Highlights  []string               `json:"highlights"`
	FilePath    string                 `json:"file_path"`
	LineNumbers []int                  `json:"line_numbers"`
	Context     string                 `json:"context"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SearchQuery represents an enhanced search query
type SearchQuery struct {
	Query         string                 `json:"query"`
	Type          engine.QueryType       `json:"type"`
	MaxResults    int                    `json:"max_results"`
	MinScore      float64                `json:"min_score"`
	FileTypes     []string               `json:"file_types"`
	ExcludePaths  []string               `json:"exclude_paths"`
	IncludePaths  []string               `json:"include_paths"`
	DateRange     *DateRange             `json:"date_range,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	Skills        []string               `json:"skills,omitempty"`
	ExpandQuery   bool                   `json:"expand_query"`
	UseEmbeddings bool                   `json:"use_embeddings"`
	UseReranking  bool                   `json:"use_reranking"`
	Options       map[string]interface{} `json:"options,omitempty"`
}

// DateRange represents a date range filter
type DateRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// NewWorkspaceSearch creates a new workspace search instance
func NewWorkspaceSearch(config *WorkspaceConfig) (*WorkspaceSearch, error) {
	if config == nil {
		config = getDefaultWorkspaceConfig()
	}

	// Validate root path
	if config.RootPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		config.RootPath = cwd
	}

	if !filepath.IsAbs(config.RootPath) {
		abs, err := filepath.Abs(config.RootPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
		}
		config.RootPath = abs
	}

	ws := &WorkspaceSearch{
		rootPath:        config.RootPath,
		config:          config,
		templateManager: skills.NewTemplateManager(),
		fileCache:       make(map[string]*CachedFile),
		cacheTTL:        config.CacheTTL,
	}

	// Initialize workspace indexer
	var err error
	ws.indexer, err = NewWorkspaceIndexer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize workspace indexer: %w", err)
	}

	return ws, nil
}

// Search performs an enhanced search in the workspace
func (ws *WorkspaceSearch) Search(ctx context.Context, query *SearchQuery) (*SearchResults, error) {
	start := time.Now()

	// Validate query
	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	// Expand query if requested
	if query.ExpandQuery {
		expandedQuery, err := ws.expandQuery(query)
		if err != nil {
			return nil, fmt.Errorf("failed to expand query: %w", err)
		}
		query.Query = expandedQuery
	}

	// Convert to engine query
	engineQuery := ws.toEngineQuery(query)

	// Execute search
	result, err := ws.searchEngine.Search(ctx, engineQuery)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert and enhance results
	searchResults := ws.enhanceResults(result, query)

	// Apply reranking if requested
	if query.UseReranking && len(searchResults.Results) > 1 {
		searchResults = ws.rerankResults(searchResults, query)
	}

	// Apply post-processing filters
	searchResults = ws.applyFilters(searchResults, query)

	searchResults.TotalTime = time.Since(start)
	searchResults.QueryInfo = &QueryInfo{
		OriginalQuery: query.Query,
		ExpandedQuery: query.Query,
		Type:          fmt.Sprint(rune(query.Type)),
		ResultsCount:  len(searchResults.Results),
	}

	return searchResults, nil
}

// SearchResults represents the complete search results
type SearchResults struct {
	Results      []*SearchResult        `json:"results"`
	Total        int                    `json:"total"`
	QueryTime    time.Duration          `json:"query_time"`
	TotalTime    time.Duration          `json:"total_time"`
	QueryInfo    *QueryInfo             `json:"query_info,omitempty"`
	Aggregations map[string]interface{} `json:"aggregations,omitempty"`
}

// QueryInfo contains information about the executed query
type QueryInfo struct {
	OriginalQuery string `json:"original_query"`
	ExpandedQuery string `json:"expanded_query"`
	Type          string `json:"type"`
	ResultsCount  int    `json:"results_count"`
}

// expandQuery expands the search query using skills
func (ws *WorkspaceSearch) expandQuery(query *SearchQuery) (string, error) {
	skillInput := &skills.SkillInput{
		Query: query.Query,
		Parameters: map[string]interface{}{
			"max_expansions": 8,
			"file_types":     query.FileTypes,
		},
	}

	output, err := ws.templateManager.ExecuteSkill("expandQuery", skillInput, nil)
	if err != nil {
		return query.Query, err
	}

	if !output.Success {
		return query.Query, fmt.Errorf("skill execution failed: %s", output.Error)
	}

	if result, ok := output.Result.(map[string]interface{}); ok {
		if expanded, ok := result["expanded_terms"].([]string); ok && len(expanded) > 0 {
			return fmt.Sprintf("%s %s", query.Query, strings.Join(expanded, " ")), nil
		}
	}

	return query.Query, nil
}

// toEngineQuery converts workspace query to engine query
func (ws *WorkspaceSearch) toEngineQuery(query *SearchQuery) *engine.Query {
	engineQuery := &engine.Query{
		Text:    query.Query,
		Type:    query.Type,
		Limit:   query.MaxResults,
		Filters: make(map[string]interface{}),
	}

	// Add file type filters
	if len(query.FileTypes) > 0 {
		engineQuery.Filters["file_types"] = query.FileTypes
	}

	// Add path filters
	if len(query.ExcludePaths) > 0 {
		engineQuery.Filters["exclude_paths"] = query.ExcludePaths
	}
	if len(query.IncludePaths) > 0 {
		engineQuery.Filters["include_paths"] = query.IncludePaths
	}

	// Add tag filters
	if len(query.Tags) > 0 {
		engineQuery.Filters["tags"] = query.Tags
	}

	// Add date range filter
	if query.DateRange != nil {
		engineQuery.Filters["date_from"] = query.DateRange.From
		engineQuery.Filters["date_to"] = query.DateRange.To
	}

	return engineQuery
}

// enhanceResults converts engine results to enhanced search results
func (ws *WorkspaceSearch) enhanceResults(engineResult *engine.Result, query *SearchQuery) *SearchResults {
	results := make([]*SearchResult, 0, len(engineResult.Documents))

	for i, doc := range engineResult.Documents {
		score := float64(0)
		if i < len(engineResult.Scores) {
			score = engineResult.Scores[i]
		}

		// Skip if below minimum score
		if query.MinScore > 0 && score < query.MinScore {
			continue
		}

		// Create enhanced search result
		result := &SearchResult{
			Document:    doc,
			Score:       score,
			Relevance:   ws.calculateRelevance(score),
			FilePath:    doc.ID,
			LineNumbers: ws.extractLineNumbers(doc),
			Context:     ws.extractContext(doc),
			Tags:        ws.extractTags(doc),
			Metadata:    ws.enhanceMetadata(doc, query),
		}

		// Generate highlights if needed
		if query.Options != nil && query.Options["highlights"].(bool) {
			result.Highlights = ws.generateHighlights(doc, query.Query)
		}

		results = append(results, result)
	}

	return &SearchResults{
		Results:   results,
		Total:     len(results),
		QueryTime: engineResult.QueryTime,
	}
}

// rerankResults reranks search results using LLM reranking
func (ws *WorkspaceSearch) rerankResults(results *SearchResults, query *SearchQuery) *SearchResults {
	if len(results.Results) == 0 {
		return results
	}

	// Prepare documents for reranking
	docs := make([]string, len(results.Results))
	for i, result := range results.Results {
		docs[i] = result.Content
	}

	// Use LLM reranking if available
	scores, err := llm.Rerank(query.Query, docs)
	if err != nil {
		// Fallback: keep original scores
		return results
	}

	// Update scores and resort
	for i, score := range scores {
		if i < len(results.Results) {
			results.Results[i].Score = score
			results.Results[i].Relevance = ws.calculateRelevance(score)
		}
	}

	// Sort results by new scores
	ws.sortResults(results.Results)

	return results
}

// applyFilters applies post-processing filters to results
func (ws *WorkspaceSearch) applyFilters(results *SearchResults, query *SearchQuery) *SearchResults {
	if len(results.Results) == 0 {
		return results
	}

	filtered := make([]*SearchResult, 0, len(results.Results))

	for _, result := range results.Results {
		// Apply file type filter
		if len(query.FileTypes) > 0 {
			ext := strings.ToLower(filepath.Ext(result.FilePath))
			match := false
			for _, fileType := range query.FileTypes {
				if strings.ToLower(fileType) == ext || strings.ToLower(fileType) == "*"+ext {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		// Apply path filters
		if len(query.ExcludePaths) > 0 {
			shouldExclude := false
			for _, excludePath := range query.ExcludePaths {
				if strings.Contains(result.FilePath, excludePath) {
					shouldExclude = true
					break
				}
			}
			if shouldExclude {
				continue
			}
		}

		filtered = append(filtered, result)
	}

	results.Results = filtered
	results.Total = len(filtered)

	return results
}

// Helper methods
func (ws *WorkspaceSearch) calculateRelevance(score float64) string {
	if score >= 0.9 {
		return "high"
	} else if score >= 0.7 {
		return "medium"
	} else if score >= 0.5 {
		return "low"
	}
	return "very_low"
}

func (ws *WorkspaceSearch) extractLineNumbers(doc *engine.Document) []int {
	// This would extract line numbers from document metadata
	// Implementation depends on how line numbers are stored
	return []int{}
}

func (ws *WorkspaceSearch) extractContext(doc *engine.Document) string {
	// Return a snippet of content for context
	if len(doc.Content) > 200 {
		return doc.Content[:200] + "..."
	}
	return doc.Content
}

func (ws *WorkspaceSearch) extractTags(doc *engine.Document) []string {
	// Extract tags from document metadata
	if tags, ok := doc.Metadata["tags"].([]string); ok {
		return tags
	}
	return []string{}
}

func (ws *WorkspaceSearch) enhanceMetadata(doc *engine.Document, query *SearchQuery) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Copy existing metadata
	for k, v := range doc.Metadata {
		metadata[k] = v
	}

	// Add enhanced metadata
	metadata["file_size"] = len(doc.Content)
	metadata["file_type"] = filepath.Ext(doc.ID)
	metadata["search_time"] = time.Now()

	return metadata
}

func (ws *WorkspaceSearch) generateHighlights(doc *engine.Document, query string) []string {
	// Simple highlighting implementation
	content := strings.ToLower(doc.Content)
	queryLower := strings.ToLower(query)

	var highlights []string
	if strings.Contains(content, queryLower) {
		// Find first occurrence and create highlight
		index := strings.Index(content, queryLower)
		if index >= 0 {
			start := max(0, index-50)
			end := min(len(doc.Content), index+len(query)+50)
			highlight := doc.Content[start:end]
			highlights = append(highlights, "..."+highlight+"...")
		}
	}

	return highlights
}

func (ws *WorkspaceSearch) sortResults(results []*SearchResult) {
	// Sort by score descending
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Score < results[j].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

// Validate validates the search query
func (q *SearchQuery) Validate() error {
	if strings.TrimSpace(q.Query) == "" {
		return fmt.Errorf("query cannot be empty")
	}

	if q.MaxResults <= 0 {
		q.MaxResults = 50 // Default
	}

	if q.MinScore < 0 {
		q.MinScore = 0
	} else if q.MinScore > 1 {
		q.MinScore = 1
	}

	return nil
}

// getDefaultWorkspaceConfig returns default workspace configuration
func getDefaultWorkspaceConfig() *WorkspaceConfig {
	return &WorkspaceConfig{
		IndexInterval:    5 * time.Minute,
		MaxFileSize:      10 * 1024 * 1024, // 10MB
		ExcludePatterns:  []string{".git", "node_modules", "vendor", "dist", "build"},
		IncludePatterns:  []string{"*"},
		EnableEmbeddings: true,
		EmbeddingModel:   "all-MiniLM-L6-v2",
		EnableSkills:     true,
		CacheEnabled:     true,
		CacheTTL:         10 * time.Minute,
		ParallelIndexing: true,
		MaxIndexWorkers:  4,
	}
}

// Utility functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
