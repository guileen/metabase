package analysis

import ("context"
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/guileen/metabase/pkg/infra/storage")

// Unified System for Code Analysis and Search
// This package provides the core abstraction that unifies static analysis and search functionality

// ArtifactType represents the type of code artifact
type ArtifactType int

const (
	ArtifactTypeSource ArtifactType = iota
	ArtifactTypeBinary
	ArtifactTypeConfig
	ArtifactTypeDocumentation
	ArtifactTypeTest
	ArtifactTypeDependency
	ArtifactTypeAST
	ArtifactTypeBytecode
)

// ProcessingStage represents the processing pipeline stage
type ProcessingStage int

const (
	StageRaw ProcessingStage = iota
	StageTokenized
	StageParsed
	StageAnalyzed
	StageIndexed
	StageVectorized
)

// FeatureType represents different feature types for analysis
type FeatureType int

const (
	FeatureLexical FeatureType = iota
	FeatureSyntactic
	FeatureSemantic
	FeatureStructural
	FeatureMetric
	FeaturePattern
	FeatureSecurity
	FeatureQuality
)

// AnalyzerCapability represents what an analyzer can do
type AnalyzerCapability int

const (
	CapabilityAnalyze AnalyzerCapability = 1 << iota
	CapabilitySearch
	CapabilityIndex
	CapabilityCompare
	CapabilityTransform
	CapabilityValidate
	CapabilityRecommend
)

// Core abstraction - Artifact represents any code element
type Artifact struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	ProjectID    string                 `json:"project_id"`
	Type         ArtifactType           `json:"type"`
	Language     string                 `json:"language"`
	Path         string                 `json:"path"`
	Name         string                 `json:"name"`
	Content      []byte                 `json:"content"`
	Size         int64                  `json:"size"`
	Hash         string                 `json:"hash"`
	Stage        ProcessingStage        `json:"stage"`
	Features     map[FeatureType][]byte `json:"features"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Version      int                    `json:"version"`
	Dependencies []string               `json:"dependencies"`
	References   []string               `json:"references"`
}

// FeatureVector represents a feature vector for similarity computation
type FeatureVector struct {
	ArtifactID  string            `json:"artifact_id"`
	Type        FeatureType       `json:"type"`
	Vector      []float64         `json:"vector"`
	Metadata    map[string]string `json:"metadata"`
	Confidence  float64           `json:"confidence"`
	GeneratedAt time.Time         `json:"generated_at"`
}

// AnalysisResult represents the result of analysis
type AnalysisResult struct {
	ArtifactID    string                 `json:"artifact_id"`
	AnalyzerID    string                 `json:"analyzer_id"`
	Type          string                 `json:"type"`
	Findings      []Finding              `json:"findings"`
	Metrics       map[string]float64     `json:"metrics"`
	Score         float64                `json:"score"`
	Confidence    float64                `json:"confidence"`
	Metadata      map[string]interface{} `json:"metadata"`
	Duration      time.Duration          `json:"duration"`
	ProcessedAt   time.Time              `json:"processed_at"`
}

// Finding represents a single analysis finding
type Finding struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`        // "issue", "pattern", "duplicate", "vulnerability"
	Severity    string                 `json:"severity"`    // "low", "medium", "high", "critical"
	Line        int                    `json:"line"`
	Column      int                    `json:"column"`
	EndLine     int                    `json:"end_line"`
	EndColumn   int                    `json:"end_column"`
	Message     string                 `json:"message"`
	Rule        string                 `json:"rule"`
	Category    string                 `json:"category"`
	Context     string                 `json:"context"`
	Suggestion  string                 `json:"suggestion"`
	Metadata    map[string]interface{} `json:"metadata"`
	Confidence  float64                `json:"confidence"`
}

// SimilarityResult represents similarity between artifacts
type SimilarityResult struct {
	ArtifactID1   string            `json:"artifact_id1"`
	ArtifactID2   string            `json:"artifact_id2"`
	Score         float64           `json:"score"`
	Method        string            `json:"method"`         // "exact", "near", "semantic"
	MatchType     string            `json:"match_type"`     // "full", "partial", "pattern"
	SharedTokens  []string          `json:"shared_tokens"`
	Differences   []string          `json:"differences"`
	Metadata      map[string]string `json:"metadata"`
	ComputedAt    time.Time         `json:"computed_at"`
}

// Analyzer interface - core abstraction
type Analyzer interface {
	// Basic info
	ID() string
	Name() string
	Version() string
	Capabilities() AnalyzerCapability

	// Supported languages and types
	SupportedLanguages() []string
	SupportedTypes() []ArtifactType

	// Core analysis methods
	Analyze(ctx context.Context, artifact *Artifact) (*AnalysisResult, error)
	ExtractFeatures(ctx context.Context, artifact *Artifact) ([]*FeatureVector, error)
	Compare(ctx context.Context, artifact1, artifact2 *Artifact) (*SimilarityResult, error)

	// Search support
	BuildIndex(ctx context.Context, artifacts []*Artifact) error
	Search(ctx context.Context, query *Query) ([]*SearchResult, error)

	// Lifecycle
	Initialize(ctx context.Context) error
	Cleanup() error
}

// Query represents a unified query for both search and analysis
type Query struct {
	ID          string                 `json:"id"`
	Type        QueryType              `json:"type"`
	Text        string                 `json:"text"`
	Vector      []float64              `json:"vector"`
	Pattern     string                 `json:"pattern"`
	Languages   []string               `json:"languages"`
	Types       []ArtifactType         `json:"types"`
	Filters     map[string]interface{} `json:"filters"`
	Similarity  float64                `json:"similarity"`
	Limit       int                    `json:"limit"`
	Offset      int                    `json:"offset"`
	Options     map[string]interface{} `json:"options"`
}

// QueryType represents different query types
type QueryType int

const (
	QueryTypeText QueryType = iota
	QueryTypeSemantic
	QueryTypePattern
	QueryTypeSimilar
	QueryTypeDuplicate
	QueryTypeSecurity
	QueryTypeQuality
	QueryTypeAPI
	QueryTypeCustom
)

// SearchResult represents a search result
type SearchResult struct {
	ArtifactID   string                 `json:"artifact_id"`
	Score        float64                `json:"score"`
	MatchType    string                 `json:"match_type"`
	Highlights   []string               `json:"highlights"`
	Explanation  string                 `json:"explanation"`
	Context      map[string]interface{} `json:"context"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Config represents the system configuration
type Config struct {
	Storage        *storage.HybridStorage `json:"-"`
	CacheSize      int                    `json:"cache_size"`
	Workers        int                    `json:"workers"`
	BatchSize      int                    `json:"batch_size"`
	VectorDim      int                    `json:"vector_dim"`
	MaxTokens      int                    `json:"max_tokens"`
	EnableRealtime bool                   `json:"enable_realtime"`
}

// Engine is the core engine that unifies analysis and search
type Engine struct {
	config       *Config
	storage      *storage.HybridStorage
	analyzers    map[string]Analyzer
	indexes      map[string]Index
	processors   map[ArtifactType]Processor
	cache        *AnalysisCache
	queue        chan *AnalysisTask
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	stats        *EngineStats
}

// Index interface for different index types
type Index interface {
	Type() string
	Index(ctx context.Context, artifact *Artifact) error
	Search(ctx context.Context, query *Query) ([]*SearchResult, error)
	Delete(ctx context.Context, artifactID string) error
	Stats() map[string]interface{}
	Close() error
}

// Processor interface for artifact processing
type Processor interface {
	CanProcess(artifactType ArtifactType) bool
	Process(ctx context.Context, artifact *Artifact) (*Artifact, error)
	Tokenize(ctx context.Context, content []byte) ([]Token, error)
	Parse(ctx context.Context, content []byte, language string) (interface{}, error)
}

// Token represents a tokenized element
type Token struct {
	Type      string    `json:"type"`      // "keyword", "identifier", "literal", "operator"
	Value     string    `json:"value"`
	Position  Position  `json:"position"`
	Line      int       `json:"line"`
	Column    int       `json:"column"`
	Length    int       `json:"length"`
	Hash      string    `json:"hash"`
	Features  []byte    `json:"features"`
}

// Position represents a position in source code
type Position struct {
	Offset int `json:"offset"`
	Line   int `json:"line"`
	Column int `json:"column"`
}

// AnalysisTask represents a task for the analysis pipeline
type AnalysisTask struct {
	ID        string      `json:"id"`
	Type      TaskType    `json:"type"`
	Artifact  *Artifact   `json:"artifact"`
	Query     *Query      `json:"query"`
	Context   context.Context `json:"-"`
	Result    chan *TaskResult `json:"-"`
	Priority  int         `json:"priority"`
	CreatedAt time.Time   `json:"created_at"`
}

// TaskType represents the task type
type TaskType int

const (
	TaskTypeAnalyze TaskType = iota
	TaskTypeSearch
	TaskTypeIndex
	TaskTypeCompare
	TaskTypeExtract
)

// TaskResult represents the result of a task
type TaskResult struct {
	TaskID    string                 `json:"task_id"`
	Type      TaskType               `json:"type"`
	Success   bool                   `json:"success"`
	Data      interface{}            `json:"data"`
	Error     error                  `json:"error"`
	Duration  time.Duration          `json:"duration"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// AnalysisCache provides caching for analysis results
type AnalysisCache struct {
	cache map[string]*CacheEntry
	max   int
	mu    sync.RWMutex
	ttl   time.Duration
}

// CacheEntry represents a cache entry
type CacheEntry struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	ExpiresAt time.Time   `json:"expires_at"`
	HitCount  int64       `json:"hit_count"`
}

// EngineStats represents engine statistics
type EngineStats struct {
	ArtifactsProcessed int64     `json:"artifacts_processed"`
	AnalysisCount      int64     `json:"analysis_count"`
	SearchCount        int64     `json:"search_count"`
	CacheHits          int64     `json:"cache_hits"`
	CacheMisses        int64     `json:"cache_misses"`
	AverageLatency     time.Duration `json:"average_latency"`
	LastActivity       time.Time `json:"last_activity"`
	mu                 sync.RWMutex
}

// NewEngine creates a new analysis and search engine
func NewEngine(config *Config) (*Engine, error) {
	if config.Storage == nil {
		return nil, fmt.Errorf("storage is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	engine := &Engine{
		config:     config,
		storage:    config.Storage,
		analyzers:  make(map[string]Analyzer),
		indexes:    make(map[string]Index),
		processors: make(map[ArtifactType]Processor),
		cache:      NewAnalysisCache(config.CacheSize),
		queue:      make(chan *AnalysisTask, config.BatchSize*10),
		ctx:        ctx,
		cancel:     cancel,
		stats:      &EngineStats{},
	}

	// Start worker pool
	for i := 0; i < config.Workers; i++ {
		engine.wg.Add(1)
		go engine.worker(i)
	}

	return engine, nil
}

// RegisterAnalyzer registers an analyzer
func (e *Engine) RegisterAnalyzer(analyzer Analyzer) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	id := analyzer.ID()
	if _, exists := e.analyzers[id]; exists {
		return fmt.Errorf("analyzer %s already registered", id)
	}

	// Initialize analyzer
	if err := analyzer.Initialize(e.ctx); err != nil {
		return fmt.Errorf("failed to initialize analyzer %s: %w", id, err)
	}

	e.analyzers[id] = analyzer
	return nil
}

// RegisterProcessor registers a processor
func (e *Engine) RegisterProcessor(processor Processor) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, artifactType := range []ArtifactType{
		ArtifactTypeSource,
		ArtifactTypeConfig,
		ArtifactTypeDocumentation,
		ArtifactTypeTest,
	} {
		if processor.CanProcess(artifactType) {
			e.processors[artifactType] = processor
		}
	}
}

// Analyze analyzes an artifact
func (e *Engine) Analyze(ctx context.Context, artifact *Artifact) ([]*AnalysisResult, error) {
	// Check cache first
	cacheKey := e.generateCacheKey("analyze", artifact.ID, artifact.Hash)
	if cached := e.cache.Get(cacheKey); cached != nil {
		if results, ok := cached.([]*AnalysisResult); ok {
			e.updateStats(func(s *EngineStats) {
				s.CacheHits++
			})
			return results, nil
		}
	}

	e.updateStats(func(s *EngineStats) {
		s.CacheMisses++
	})

	// Submit analysis task
	task := &AnalysisTask{
		ID:       generateID(),
		Type:     TaskTypeAnalyze,
		Artifact: artifact,
		Context:  ctx,
		Result:   make(chan *TaskResult, 1),
		Priority: 1,
	}

	select {
	case e.queue <- task:
		select {
		case result := <-task.Result:
			if result.Error != nil {
				return nil, result.Error
			}
			if results, ok := result.Data.([]*AnalysisResult); ok {
				// Cache results
				e.cache.Set(cacheKey, results, 30*time.Minute)
				return results, nil
			}
			return nil, fmt.Errorf("invalid result type")
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(30 * time.Second):
			return nil, fmt.Errorf("analysis timeout")
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, fmt.Errorf("analysis queue full")
	}
}

// Search performs a search query
func (e *Engine) Search(ctx context.Context, query *Query) ([]*SearchResult, error) {
	start := time.Now()
	defer func() {
		e.updateStats(func(s *EngineStats) {
			s.SearchCount++
			s.AverageLatency = time.Duration((int64(s.AverageLatency) + time.Since(start).Nanoseconds()) / 2)
			s.LastActivity = time.Now()
		})
	}()

	// Check cache
	cacheKey := e.generateQueryCacheKey(query)
	if cached := e.cache.Get(cacheKey); cached != nil {
		if results, ok := cached.([]*SearchResult); ok {
			e.updateStats(func(s *EngineStats) {
				s.CacheHits++
			})
			return results, nil
		}
	}

	e.updateStats(func(s *EngineStats) {
		s.CacheMisses++
	})

	// Submit search task
	task := &AnalysisTask{
		ID:     generateID(),
		Type:   TaskTypeSearch,
		Query:  query,
		Result: make(chan *TaskResult, 1),
		Context: ctx,
	}

	select {
	case e.queue <- task:
		select {
		case result := <-task.Result:
			if result.Error != nil {
				return nil, result.Error
			}
			if results, ok := result.Data.([]*SearchResult); ok {
				// Cache results for shorter time
				e.cache.Set(cacheKey, results, 5*time.Minute)
				return results, nil
			}
			return nil, fmt.Errorf("invalid search result type")
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Second):
			return nil, fmt.Errorf("search timeout")
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, fmt.Errorf("search queue full")
	}
}

// FindDuplicates finds duplicate artifacts
func (e *Engine) FindDuplicates(ctx context.Context, artifact *Artifact, threshold float64) ([]*SimilarityResult, error) {
	query := &Query{
		ID:         generateID(),
		Type:       QueryTypeDuplicate,
		Similarity: threshold,
		Limit:      100,
		Filters: map[string]interface{}{
			"type":     artifact.Type,
			"language": artifact.Language,
			"exclude":  artifact.ID,
		},
	}

	task := &AnalysisTask{
		ID:       generateID(),
		Type:     TaskTypeCompare,
		Artifact: artifact,
		Query:    query,
		Context:  ctx,
		Result:   make(chan *TaskResult, 1),
	}

	select {
	case e.queue <- task:
		select {
		case result := <-task.Result:
			if result.Error != nil {
				return nil, result.Error
			}
			if similarities, ok := result.Data.([]*SimilarityResult); ok {
				return similarities, nil
			}
			return nil, fmt.Errorf("invalid similarity result type")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	default:
		return nil, fmt.Errorf("queue full")
	}
}

// worker processes tasks from the queue
func (e *Engine) worker(id int) {
	defer e.wg.Done()

	for {
		select {
		case task := <-e.queue:
			e.processTask(task)
		case <-e.ctx.Done():
			return
		}
	}
}

// processTask processes a single task
func (e *Engine) processTask(task *AnalysisTask) {
	start := time.Now()
	result := &TaskResult{
		TaskID:   task.ID,
		Type:     task.Type,
		Success:  true,
		Duration: time.Since(start),
		Metadata: make(map[string]interface{}),
	}

	defer func() {
		select {
		case task.Result <- result:
		default:
		}
	}()

	switch task.Type {
	case TaskTypeAnalyze:
		results, err := e.performAnalysis(task.Context, task.Artifact)
		if err != nil {
			result.Error = err
			result.Success = false
		} else {
			result.Data = results
			e.updateStats(func(s *EngineStats) {
				s.AnalysisCount++
				s.ArtifactsProcessed++
				s.LastActivity = time.Now()
			})
		}

	case TaskTypeSearch:
		results, err := e.performSearch(task.Context, task.Query)
		if err != nil {
			result.Error = err
			result.Success = false
		} else {
			result.Data = results
		}

	case TaskTypeCompare:
		results, err := e.performComparison(task.Context, task.Artifact, task.Query)
		if err != nil {
			result.Error = err
			result.Success = false
		} else {
			result.Data = results
		}
	}
}

// performAnalysis performs actual analysis
func (e *Engine) performAnalysis(ctx context.Context, artifact *Artifact) ([]*AnalysisResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var allResults []*AnalysisResult

	// Process artifact if needed
	if artifact.Stage < StageAnalyzed {
		if processor, exists := e.processors[artifact.Type]; exists {
			processed, err := processor.Process(ctx, artifact)
			if err != nil {
				return nil, fmt.Errorf("processing failed: %w", err)
			}
			artifact = processed
		}
	}

	// Run all applicable analyzers
	for _, analyzer := range e.analyzers {
		// Check if analyzer supports this artifact
		supported := false
		for _, lang := range analyzer.SupportedLanguages() {
			if lang == artifact.Language || lang == "*" {
				supported = true
				break
			}
		}

		if !supported {
			continue
		}

		// Check analyzer capabilities
		if analyzer.Capabilities()&CapabilityAnalyze == 0 {
			continue
		}

		// Run analysis
		result, err := analyzer.Analyze(ctx, artifact)
		if err != nil {
			continue // Log error but continue with other analyzers
		}

		allResults = append(allResults, result)
	}

	return allResults, nil
}

// performSearch performs actual search
func (e *Engine) performSearch(ctx context.Context, query *Query) ([]*SearchResult, error) {
	var allResults []*SearchResult

	// Search in indexes
	for _, index := range e.indexes {
		results, err := index.Search(ctx, query)
		if err != nil {
			continue
		}
		allResults = append(allResults, results...)
	}

	// Use analyzers with search capability
	e.mu.RLock()
	for _, analyzer := range e.analyzers {
		if analyzer.Capabilities()&CapabilitySearch == 0 {
			continue
		}

		results, err := analyzer.Search(ctx, query)
		if err != nil {
			continue
		}
		allResults = append(allResults, results...)
	}
	e.mu.RUnlock()

	// Deduplicate and rank results
	return e.rankResults(allResults, query.Limit), nil
}

// performComparison performs artifact comparison
func (e *Engine) performComparison(ctx context.Context, artifact *Artifact, query *Query) ([]*SimilarityResult, error) {
	var similarities []*SimilarityResult

	e.mu.RLock()
	for _, analyzer := range e.analyzers {
		if analyzer.Capabilities()&CapabilityCompare == 0 {
			continue
		}

		// Get candidate artifacts for comparison
		// This would query storage for similar artifacts
		// For now, return empty result
	}
	e.mu.RUnlock()

	return similarities, nil
}

// rankResults ranks and deduplicates search results
func (e *Engine) rankResults(results []*SearchResult, limit int) []*SearchResult {
	if len(results) == 0 {
		return results
	}

	// Simple scoring and deduplication
	seen := make(map[string]bool)
	var ranked []*SearchResult

	for _, result := range results {
		if !seen[result.ArtifactID] {
			seen[result.ArtifactID] = true
			ranked = append(ranked, result)
		}
	}

	// Sort by score
	for i := 0; i < len(ranked)-1; i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[i].Score < ranked[j].Score {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}

	if limit > 0 && len(ranked) > limit {
		ranked = ranked[:limit]
	}

	return ranked
}

// generateCacheKey generates a cache key
func (e *Engine) generateCacheKey(parts ...string) string {
	h := fnv.New64a()
	for _, part := range parts {
		h.Write([]byte(part))
		h.Write([]byte(":"))
	}
	return fmt.Sprintf("analysis:%x", h.Sum64())
}

// generateQueryCacheKey generates a query cache key
func (e *Engine) generateQueryCacheKey(query *Query) string {
	queryBytes, _ := json.Marshal(query)
	h := fnv.New64a()
	h.Write(queryBytes)
	return fmt.Sprintf("query:%x", h.Sum64())
}

// updateStats updates engine stats thread-safely
func (e *Engine) updateStats(fn func(*EngineStats)) {
	e.stats.mu.Lock()
	defer e.stats.mu.Unlock()
	fn(e.stats)
}

// GetStats returns engine statistics
func (e *Engine) GetStats() *EngineStats {
	e.stats.mu.RLock()
	defer e.stats.mu.RUnlock()

	return &EngineStats{
		ArtifactsProcessed: e.stats.ArtifactsProcessed,
		AnalysisCount:      e.stats.AnalysisCount,
		SearchCount:        e.stats.SearchCount,
		CacheHits:          e.stats.CacheHits,
		CacheMisses:        e.stats.CacheMisses,
		AverageLatency:     e.stats.AverageLatency,
		LastActivity:       e.stats.LastActivity,
	}
}

// Close closes the engine
func (e *Engine) Close() error {
	e.cancel()
	e.wg.Wait()

	e.mu.Lock()
	defer e.mu.Unlock()

	// Cleanup analyzers
	for _, analyzer := range e.analyzers {
		analyzer.Cleanup()
	}

	// Close indexes
	for _, index := range e.indexes {
		index.Close()
	}

	return nil
}

// NewAnalysisCache creates a new analysis cache
func NewAnalysisCache(size int) *AnalysisCache {
	if size <= 0 {
		size = 10000
	}
	return &AnalysisCache{
		cache: make(map[string]*CacheEntry),
		max:   size,
		ttl:   30 * time.Minute,
	}
}

// Get gets value from cache
func (c *AnalysisCache) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[key]
	if !exists || time.Now().After(entry.ExpiresAt) {
		return nil
	}

	entry.HitCount++
	return entry.Value
}

// Set sets value in cache
func (c *AnalysisCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict if necessary
	if len(c.cache) >= c.max {
		c.evictLRU()
	}

	c.cache[key] = &CacheEntry{
		Key:       key,
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
		HitCount:  0,
	}
}

// evictLRU evicts least recently used entries
func (c *AnalysisCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time
	var minHits int64 = -1

	for key, entry := range c.cache {
		if minHits == -1 || entry.HitCount < minHits ||
		   (entry.HitCount == minHits && entry.ExpiresAt.Before(oldestTime)) {
			oldestKey = key
			oldestTime = entry.ExpiresAt
			minHits = entry.HitCount
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
	}
}

// generateID generates a unique ID
func generateID() string {
	return fmt.Sprintf("%d_%x", time.Now().UnixNano(), fnv.New64a().Sum64())
}