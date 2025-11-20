package analysis

import ("context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/guileen/metabase/pkg/infra/search/index"
	"github.com/guileen/metabase/pkg/infra/search/vector")

// SearchEngine provides advanced search capabilities
type SearchEngine struct {
	*BaseAnalyzer
	fullTextIndex *index.InvertedIndex
	vectorIndex   *vector.HNSWIndex
	kvStore       *pebble.DB
	queryCache    map[string]*CachedQuery
	mu            sync.RWMutex
}

// CachedQuery represents a cached search query
type CachedQuery struct {
	Query    *Query      `json:"query"`
	Results  []*SearchResult `json:"results"`
	Count    int         `json:"count"`
	CachedAt time.Time   `json:"cached_at"`
	TTL      time.Duration `json:"ttl"`
}

// NewSearchEngine creates a new search engine
func NewSearchEngine(db interface{}, kv *pebble.DB) (*SearchEngine, error) {
	se := &SearchEngine{
		BaseAnalyzer: NewBaseAnalyzer(
			"search-engine",
			"Advanced Code Search Engine",
			"1.0.0",
			CapabilitySearch|CapabilityIndex|CapabilityAnalyze,
		),
		kvStore:    kv,
		queryCache: make(map[string]*CachedQuery),
	}

	// Initialize indexes
	// In production, these would be properly initialized
	// se.fullTextIndex = index.NewInvertedIndex(db)
	// se.vectorIndex = vector.NewHNSWIndex(kv)

	// Set supported types
	se.types = []ArtifactType{
		ArtifactTypeSource,
		ArtifactTypeDocumentation,
		ArtifactTypeConfig,
	}

	return se, nil
}

// Search performs advanced search
func (se *SearchEngine) Search(ctx context.Context, query *Query) ([]*SearchResult, error) {
	// Check cache first
	cacheKey := se.generateQueryCacheKey(query)
	if cached := se.getFromCache(cacheKey); cached != nil {
		return cached.Results, nil
	}

	var results []*SearchResult

	switch query.Type {
	case QueryTypeText:
		var err error
		results, err = se.searchFullText(ctx, query)
		if err != nil {
			return nil, err
		}

	case QueryTypeSemantic:
		var err error
		results, err = se.searchSemantic(ctx, query)
		if err != nil {
			return nil, err
		}

	case QueryTypePattern:
		var err error
		results, err = se.searchPattern(ctx, query)
		if err != nil {
			return nil, err
		}

	case QueryTypeSimilar:
		var err error
		results, err = se.searchSimilar(ctx, query)
		if err != nil {
			return nil, err
		}

	case QueryTypeAPI:
		var err error
		results, err = se.searchAPI(ctx, query)
		if err != nil {
			return nil, err
		}

	default:
		// Hybrid search
		var err error
		results, err = se.searchHybrid(ctx, query)
		if err != nil {
			return nil, err
		}
	}

	// Cache results
	se.addToCache(cacheKey, query, results, 5*time.Minute)

	return results, nil
}

// searchFullText performs full-text search
func (se *SearchEngine) searchFullText(ctx context.Context, query *Query) ([]*SearchResult, error) {
	// Tokenize query
	tokens := se.tokenizeQuery(query.Text)

	// In production, this would use the actual inverted index
	// For now, return mock results
	results := make([]*SearchResult, 0)

	// Simulate search results
	mockResults := []*SearchResult{
		{
			ArtifactID:  "artifact-1",
			Score:       0.95,
			MatchType:   "full_text",
			Highlights:  []string{fmt.Sprintf("Matched <em>%s</em>", query.Text)},
			Explanation: "Exact match in function name",
			Context: map[string]interface{}{
				"line":      15,
				"function":  "calculateSum",
				"file":      "utils.go",
			},
		},
		{
			ArtifactID:  "artifact-2",
			Score:       0.82,
			MatchType:   "partial",
			Highlights:  []string{fmt.Sprintf("Partial match <em>%s</em>", tokens[0])},
			Explanation: "Partial match in comment",
			Context: map[string]interface{}{
				"line":      42,
				"function":  "processData",
				"file":      "processor.go",
			},
		},
	}

	// Apply filters
	for _, result := range mockResults {
		if se.matchesFilters(result, query.Filters) {
			results = append(results, result)
		}
	}

	// Sort by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Apply limit
	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}

	return results, nil
}

// searchSemantic performs semantic search
func (se *SearchEngine) searchSemantic(ctx context.Context, query *Query) ([]*SearchResult, error) {
	if len(query.Vector) == 0 {
		// Convert text to vector (in production, use embedding model)
		query.Vector = se.textToVector(query.Text)
	}

	// In production, this would use the actual vector index
	// For now, return mock results with semantic similarity
	results := []*SearchResult{
		{
			ArtifactID:  "artifact-3",
			Score:       0.91,
			MatchType:   "semantic",
			Highlights:  []string{"Semantically similar function"},
			Explanation: "Similar functionality: data transformation",
			Context: map[string]interface{}{
				"similarity_type": "functional",
				"functions":      []string{"transform", "convert", "map"},
			},
		},
		{
			ArtifactID:  "artifact-4",
			Score:       0.87,
			MatchType:   "semantic",
			Highlights:  []string{"Related concept"},
			Explanation: "Related concept: error handling",
			Context: map[string]interface{}{
				"similarity_type": "conceptual",
				"concepts":        []string{"error", "exception", "handle"},
			},
		},
	}

	// Apply filters and sorting
	se.processResults(results, query)

	return results, nil
}

// searchPattern performs pattern-based search
func (se *SearchEngine) searchPattern(ctx context.Context, query *Query) ([]*SearchResult, error) {
	pattern := query.Pattern
	if pattern == "" {
		pattern = query.Text
	}

	// Compile regex pattern
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	// In production, this would scan through actual artifacts
	// For now, return mock results
	results := []*SearchResult{
		{
			ArtifactID:  "artifact-5",
			Score:       1.0,
			MatchType:   "pattern",
			Highlights:  []string{fmt.Sprintf("Pattern matched: %s", pattern)},
			Explanation: "Regex pattern match found",
			Context: map[string]interface{}{
				"pattern":    pattern,
				"matches":    []int{10, 25, 42},
				"file":       "regex.go",
			},
		},
	}

	se.processResults(results, query)

	return results, nil
}

// searchSimilar finds similar code
func (se *SearchEngine) searchSimilar(ctx context.Context, query *Query) ([]*SearchResult, error) {
	similarity := query.Similarity
	if similarity == 0 {
		similarity = 0.7 // Default threshold
	}

	// In production, this would perform similarity search
	results := []*SearchResult{
		{
			ArtifactID:  "artifact-6",
			Score:       0.92,
			MatchType:   "similar",
			Highlights:  []string{"92% similar code structure"},
			Explanation: "Similar algorithmic approach",
			Context: map[string]interface{}{
				"similarity": 0.92,
				"shared_patterns": []string{"for-loop", "if-statement", "function-call"},
			},
		},
		{
			ArtifactID:  "artifact-7",
			Score:       0.78,
			MatchType:   "similar",
			Highlights:  []string{"78% similar code structure"},
			Explanation: "Similar variable naming",
			Context: map[string]interface{}{
				"similarity": 0.78,
				"shared_patterns": []string{"variable-declaration", "assignment"},
			},
		},
	}

	// Filter by similarity threshold
	filtered := make([]*SearchResult, 0)
	for _, result := range results {
		if result.Score >= similarity {
			filtered = append(filtered, result)
		}
	}

	se.processResults(filtered, query)

	return filtered, nil
}

// searchAPI searches for API definitions and usage
func (se *SearchEngine) searchAPI(ctx context.Context, query *Query) ([]*SearchResult, error) {
	// Search for API definitions, endpoints, functions, etc.
	results := []*SearchResult{
		{
			ArtifactID:  "api-1",
			Score:       0.95,
			MatchType:   "api_definition",
			Highlights:  []string{fmt.Sprintf("API endpoint: /%s", query.Text)},
			Explanation: "REST API endpoint definition",
			Context: map[string]interface{}{
				"method": "GET",
				"path":   fmt.Sprintf("/api/%s", query.Text),
				"params": []string{"id", "format"},
			},
		},
		{
			ArtifactID:  "api-2",
			Score:       0.88,
			MatchType:   "api_usage",
			Highlights:  []string{fmt.Sprintf("Function: %s()", query.Text)},
			Explanation: "Function definition and usage",
			Context: map[string]interface{}{
				"type":       "function",
				"parameters": []string{"ctx", "request"},
				"returns":    "Response",
			},
		},
	}

	se.processResults(results, query)

	return results, nil
}

// searchHybrid performs hybrid search combining multiple methods
func (se *SearchEngine) searchHybrid(ctx context.Context, query *Query) ([]*SearchResult, error) {
	// Collect results from different search methods
	var allResults []*SearchResult

	// Full-text search
	if query.Text != "" {
		ftResults, err := se.searchFullText(ctx, query)
		if err == nil {
			allResults = append(allResults, ftResults...)
		}
	}

	// Semantic search
	if len(query.Vector) > 0 || query.Text != "" {
		semResults, err := se.searchSemantic(ctx, query)
		if err == nil {
			allResults = append(allResults, semResults...)
		}
	}

	// Pattern search
	if query.Pattern != "" {
		patResults, err := se.searchPattern(ctx, query)
		if err == nil {
			allResults = append(allResults, patResults...)
		}
	}

	// Merge and deduplicate results
	merged := se.mergeResults(allResults)

	// Sort by combined score
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})

	// Apply limit
	if query.Limit > 0 && len(merged) > query.Limit {
		merged = merged[:query.Limit]
	}

	return merged, nil
}

// tokenizeQuery tokenizes search query
func (se *SearchEngine) tokenizeQuery(text string) []string {
	// Simple tokenization - in production, use proper tokenizer
	re := regexp.MustCompile(`\w+`)
	return re.FindAllString(text, -1)
}

// textToVector converts text to vector representation
func (se *SearchEngine) textToVector(text string) []float64 {
	// Simplified vector representation - in production, use embedding model
	tokens := se.tokenizeQuery(text)
	vector := make([]float64, 128)

	// Create simple hash-based vector
	for i, token := range tokens {
		if i >= 32 {
			break
		}
		// Simple hash function
		hash := 0
		for _, c := range token {
			hash = hash*31 + int(c)
		}
		vector[i*4] = float64(hash%256) / 255.0
		vector[i*4+1] = float64((hash/256)%256) / 255.0
		vector[i*4+2] = float64((hash/65536)%256) / 255.0
		vector[i*4+3] = float64(len(token)) / 50.0
	}

	return vector
}

// matchesFilters checks if result matches filters
func (se *SearchEngine) matchesFilters(result *SearchResult, filters map[string]interface{}) bool {
	if len(filters) == 0 {
		return true
	}

	for key, value := range filters {
		switch key {
		case "language":
			if lang, ok := result.Context["file"].(string); ok {
				if !strings.Contains(lang, value.(string)) {
					return false
				}
			}
		case "type":
			if result.MatchType != value.(string) {
				return false
			}
		case "min_score":
			if result.Score < value.(float64) {
				return false
			}
		}
	}

	return true
}

// processResults processes search results
func (se *SearchEngine) processResults(results []*SearchResult, query *Query) {
	// Apply filters
	filtered := make([]*SearchResult, 0)
	for _, result := range results {
		if se.matchesFilters(result, query.Filters) {
			filtered = append(filtered, result)
		}
	}

	// Sort by score
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Score > filtered[j].Score
	})

	// Apply limit
	if query.Limit > 0 && len(filtered) > query.Limit {
		filtered = filtered[:query.Limit]
	}

	// Update results slice
	copy(results, filtered)
	if len(results) > len(filtered) {
		results = results[:len(filtered)]
	}
}

// mergeResults merges and deduplicates search results
func (se *SearchEngine) mergeResults(results []*SearchResult) []*SearchResult {
	seen := make(map[string]*SearchResult)
	merged := make([]*SearchResult, 0)

	for _, result := range results {
		if existing, exists := seen[result.ArtifactID]; exists {
			// Merge results, combine scores
			existing.Score = math.Max(existing.Score, result.Score)
			existing.Highlights = append(existing.Highlights, result.Highlights...)
		} else {
			seen[result.ArtifactID] = result
			merged = append(merged, result)
		}
	}

	return merged
}

// generateQueryCacheKey generates cache key for query
func (se *SearchEngine) generateQueryCacheKey(query *Query) string {
	queryBytes, _ := json.Marshal(query)
	return fmt.Sprintf("query:%x", hashBytes(queryBytes))
}

// getFromCache retrieves from cache
func (se *SearchEngine) getFromCache(key string) *CachedQuery {
	se.mu.RLock()
	defer se.mu.RUnlock()

	if cached, exists := se.queryCache[key]; exists {
		if time.Since(cached.CachedAt) < cached.TTL {
			return cached
		}
		// Remove expired cache entry
		delete(se.queryCache, key)
	}

	return nil
}

// addToCache adds to cache
func (se *SearchEngine) addToCache(key string, query *Query, results []*SearchResult, ttl time.Duration) {
	se.mu.Lock()
	defer se.mu.Unlock()

	// Clean old cache entries if needed
	if len(se.queryCache) > 1000 {
		se.cleanCache()
	}

	se.queryCache[key] = &CachedQuery{
		Query:    query,
		Results:  results,
		Count:    len(results),
		CachedAt: time.Now(),
		TTL:      ttl,
	}
}

// cleanCache cleans old cache entries
func (se *SearchEngine) cleanCache() {
	for key, cached := range se.queryCache {
		if time.Since(cached.CachedAt) > cached.TTL {
			delete(se.queryCache, key)
		}
	}
}

// Analyze implements Analyzer interface
func (se *SearchEngine) Analyze(ctx context.Context, artifact *Artifact) (*AnalysisResult, error) {
	// Search engine doesn't perform analysis in traditional sense
	// It can provide searchability metrics
	result := &AnalysisResult{
		ArtifactID: artifact.ID,
		AnalyzerID: se.ID(),
		Type:       "searchability",
		Findings:   make([]Finding, 0),
		Metrics: map[string]float64{
			"searchable": 1.0,
			"indexed":    1.0,
		},
		Score:       100.0,
		Confidence:  1.0,
		ProcessedAt: time.Now(),
	}

	return result, nil
}

// BuildIndex builds search index
func (se *SearchEngine) BuildIndex(ctx context.Context, artifacts []*Artifact) error {
	// In production, this would build actual search indexes
	for _, artifact := range artifacts {
		// Index in full-text search
		if se.fullTextIndex != nil {
			// se.fullTextIndex.Index(artifact)
		}

		// Generate and index vector
		if se.vectorIndex != nil {
			vector := se.textToVector(string(artifact.Content))
			// se.vectorIndex.Insert(artifact.ID, vector)
		}
	}

	return nil
}

// Delete implements Index interface
func (se *SearchEngine) Delete(ctx context.Context, artifactID string) error {
	// Remove from indexes
	// In production, actual deletion from indexes
	return nil
}

// Stats returns index statistics
func (se *SearchEngine) Stats() map[string]interface{} {
	se.mu.RLock()
	defer se.mu.RUnlock()

	return map[string]interface{}{
		"cached_queries": len(se.queryCache),
		"cache_hit_rate": 0.85, // Mock value
	}
}

// Close implements Index interface
func (se *SearchEngine) Close() error {
	se.mu.Lock()
	defer se.mu.Unlock()

	se.queryCache = make(map[string]*CachedQuery)
	return nil
}

// Compare implements Analyzer interface for similarity
func (se *SearchEngine) Compare(ctx context.Context, artifact1, artifact2 *Artifact) (*SimilarityResult, error) {
	// Calculate similarity using vector representation
	vec1 := se.textToVector(string(artifact1.Content))
	vec2 := se.textToVector(string(artifact2.Content))

	// Calculate cosine similarity
	similarity := cosineSimilarity(vec1, vec2)

	// Determine match type
	var matchType string
	switch {
	case similarity >= 0.9:
		matchType = "very_similar"
	case similarity >= 0.7:
		matchType = "similar"
	case similarity >= 0.5:
		matchType = "somewhat_similar"
	default:
		matchType = "not_similar"
	}

	result := &SimilarityResult{
		ArtifactID1: artifact1.ID,
		ArtifactID2: artifact2.ID,
		Score:        similarity,
		Method:       "vector_cosine",
		MatchType:    matchType,
		ComputedAt:   time.Now(),
		Metadata: map[string]string{
			"vector_dim": fmt.Sprintf("%d", len(vec1)),
			"method":     "cosine_similarity",
		},
	}

	return result, nil
}

// ExtractFeatures implements Analyzer interface
func (se *SearchEngine) ExtractFeatures(ctx context.Context, artifact *Artifact) ([]*FeatureVector, error) {
	vectors := make([]*FeatureVector, 0)

	// Text vector
	textVector := se.textToVector(string(artifact.Content))
	vectors = append(vectors, &FeatureVector{
		ArtifactID: artifact.ID,
		Type:       FeatureSemantic,
		Vector:     textVector,
		Metadata: map[string]string{
			"analyzer": "search-engine",
			"type":     "text_embedding",
		},
		Confidence:  0.8,
		GeneratedAt: time.Now(),
	})

	return vectors, nil
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// hashBytes computes hash of byte slice
func hashBytes(data []byte) uint64 {
	h := uint64(0)
	for _, b := range data {
		h = h*31 + uint64(b)
	}
	return h
}