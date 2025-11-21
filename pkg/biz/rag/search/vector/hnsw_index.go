package vector

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/pebble"
)

// HNSWIndex HNSW (Hierarchical Navigable Small World) 向量索引
type HNSWIndex struct {
	db     *pebble.DB
	config *Config
	logger *slog.Logger

	// HNSW参数
	m   int     // 每层最大连接数
	ef  int     // 搜索时的候选数量
	ml  float64 // 层数参数
	eps float64 // 构建时的候选数量

	// 统计信息 - 使用同步访问
	stats *Stats
}

// Config HNSW配置
type Config struct {
	// 维度
	Dimension int

	// 距离函数类型
	DistanceType DistanceType

	// HNSW参数
	M   int     // 每层最大连接数 (默认16)
	EF  int     // 搜索候选数量 (默认200)
	ML  float64 // 层数参数 (默认1/ln(2))
	EPS float64 // 构建候选数量 (默认200)

	// Pebble前缀
	Prefix string
}

// DistanceType 距离函数类型
type DistanceType int

const (
	DistanceTypeCosine DistanceType = iota
	DistanceTypeL2
	DistanceTypeInnerProduct
)

// Stats 索引统计信息 - 使用原子操作保证线程安全
type Stats struct {
	// 原子操作的计数器
	vectorCount int64
	searchCount int64
	insertCount int64

	// 需要互斥锁保护的非原子字段
	mu             sync.RWMutex
	levelCount     int64
	totalEdges     int64
	averageDegree  float64
	lastInsertTime int64 // Unix纳秒
	lastSearchTime int64 // Unix纳秒
}

// VectorCount returns the current vector count atomically
func (s *Stats) VectorCount() int64 {
	return atomic.LoadInt64(&s.vectorCount)
}

// SearchCount returns the current search count atomically
func (s *Stats) SearchCount() int64 {
	return atomic.LoadInt64(&s.searchCount)
}

// InsertCount returns the current insert count atomically
func (s *Stats) InsertCount() int64 {
	return atomic.LoadInt64(&s.insertCount)
}

// AddVectorCount atomically adds to the vector count
func (s *Stats) AddVectorCount(delta int64) {
	atomic.AddInt64(&s.vectorCount, delta)
}

// AddSearchCount atomically adds to the search count
func (s *Stats) AddSearchCount(delta int64) {
	atomic.AddInt64(&s.searchCount, delta)
}

// AddInsertCount atomically adds to the insert count
func (s *Stats) AddInsertCount(delta int64) {
	atomic.AddInt64(&s.insertCount, delta)
}

// UpdateStats updates non-atomic stats under lock
func (s *Stats) UpdateStats(levelCount, totalEdges int64, averageDegree float64, lastInsertTime, lastSearchTime int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.levelCount = levelCount
	s.totalEdges = totalEdges
	s.averageDegree = averageDegree
	s.lastInsertTime = lastInsertTime
	s.lastSearchTime = lastSearchTime
}

// Clone returns a thread-safe copy of the stats
func (s *Stats) Clone() *Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &Stats{
		vectorCount:    atomic.LoadInt64(&s.vectorCount),
		searchCount:    atomic.LoadInt64(&s.searchCount),
		insertCount:    atomic.LoadInt64(&s.insertCount),
		levelCount:     s.levelCount,
		totalEdges:     s.totalEdges,
		averageDegree:  s.averageDegree,
		lastInsertTime: s.lastInsertTime,
		lastSearchTime: s.lastSearchTime,
	}
}

// VectorEntry 向量条目
type VectorEntry struct {
	ID        string
	Vector    []float64
	Level     int
	Neighbors []Neighbor
}

// Neighbor 邻居节点
type Neighbor struct {
	ID    string
	Dist  float64
	Level int
}

// SearchResult 搜索结果
type SearchResult struct {
	ID    string
	Dist  float64
	Score float64
}

// NewHNSWIndex 创建HNSW索引
func NewHNSWIndex(db *pebble.DB, config *Config) (*HNSWIndex, error) {
	// Validate database connection
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	// Set default config if nil
	if config == nil {
		config = &Config{
			Dimension:    768, // 默认维度
			DistanceType: DistanceTypeCosine,
			M:            16,
			EF:           200,
			ML:           1.0 / math.Log(2.0),
			EPS:          200,
			Prefix:       "vector:",
		}
	}

	// Validate and normalize config
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create logger
	logger := slog.With("component", "hnsw_index", "dimension", config.Dimension)

	index := &HNSWIndex{
		db:     db,
		config: config,
		logger: logger,
		m:      config.M,
		ef:     config.EF,
		ml:     config.ML,
		eps:    config.EPS,
		stats:  &Stats{},
	}

	// Test database connection
	if err := index.testConnection(); err != nil {
		return nil, fmt.Errorf("database connection test failed: %w", err)
	}

	logger.Info("HNSW index initialized",
		"m", config.M,
		"ef", config.EF,
		"distance_type", config.DistanceType)

	return index, nil
}

// Insert 插入向量
func (h *HNSWIndex) Insert(ctx context.Context, id string, vector []float64) error {
	// Validate inputs
	if err := h.validateInsertInput(id, vector); err != nil {
		h.logger.Warn("Invalid insert input", "id", id, "error", err)
		return fmt.Errorf("invalid input: %w", err)
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Check if vector already exists
	if exists, err := h.vectorExists(id); err != nil {
		return fmt.Errorf("check vector existence: %w", err)
	} else if exists {
		h.logger.Warn("Vector already exists, updating", "id", id)
		if err := h.deleteVector(id); err != nil {
			return fmt.Errorf("delete existing vector: %w", err)
		}
	}

	// 生成层级
	level := h.getRandomLevel()

	// 创建向量条目
	entry := &VectorEntry{
		ID:     id,
		Vector: make([]float64, len(vector)),
		Level:  level,
	}
	copy(entry.Vector, vector)

	// 获取入口点
	entryPointID, err := h.getEntryPoint()
	if err != nil {
		h.logger.Error("Failed to get entry point", "error", err)
		return fmt.Errorf("get entry point: %w", err)
	}

	// 如果是第一个向量
	if entryPointID == "" {
		if err := h.insertFirst(entry); err != nil {
			h.logger.Error("Failed to insert first vector", "id", id, "error", err)
			return fmt.Errorf("insert first vector: %w", err)
		}
		h.updateInsertStats()
		h.logger.Info("Inserted first vector", "id", id, "level", level)
		return nil
	}

	// HNSW插入算法
	if err := h.hnswInsert(entry, entryPointID); err != nil {
		h.logger.Error("HNSW insert failed", "id", id, "error", err)
		return fmt.Errorf("hnsw insert: %w", err)
	}

	// 更新入口点
	if level > h.getMaxLevel() {
		if err := h.setEntryPoint(id, level); err != nil {
			h.logger.Error("Failed to set entry point", "id", id, "level", level, "error", err)
			return fmt.Errorf("set entry point: %w", err)
		}
	}

	h.updateInsertStats()
	h.logger.Debug("Vector inserted successfully", "id", id, "level", level, "current_count", h.stats.VectorCount())
	return nil
}

// Delete 删除向量
func (h *HNSWIndex) Delete(id string) error {
	// 从Pebble中删除向量数据
	key := h.getVectorKey(id)
	err := h.db.Delete(key, nil)
	if err != nil {
		return fmt.Errorf("delete vector: %w", err)
	}

	// 删除邻居关系
	// 这里需要更复杂的逻辑来移除所有指向该节点的边
	// 简化实现，实际生产中需要完整实现

	h.stats.AddVectorCount(-1)
	return nil
}

// Search 搜索最近邻
func (h *HNSWIndex) Search(ctx context.Context, query []float64, k int) ([]string, []float64, error) {
	// Validate inputs
	if err := h.validateSearchInput(query, k); err != nil {
		h.logger.Warn("Invalid search input", "error", err, "k", k)
		return nil, nil, fmt.Errorf("invalid input: %w", err)
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}

	// Check if index is empty
	entryPointID, err := h.getEntryPoint()
	if err != nil {
		h.logger.Error("Failed to get entry point during search", "error", err)
		return nil, nil, fmt.Errorf("get entry point: %w", err)
	}

	if entryPointID == "" {
		h.logger.Warn("Search attempted on empty index")
		return nil, nil, fmt.Errorf("no vectors in index")
	}

	// HNSW搜索算法
	results, err := h.hnswSearch(ctx, query, entryPointID, k)
	if err != nil {
		h.logger.Error("HNSW search failed", "error", err, "k", k)
		return nil, nil, fmt.Errorf("hnsw search: %w", err)
	}

	// 提取ID和距离
	if len(results) == 0 {
		h.logger.Warn("No results found", "k", k)
		return []string{}, []float64{}, nil
	}

	ids := make([]string, len(results))
	dists := make([]float64, len(results))
	for i, r := range results {
		ids[i] = r.ID
		dists[i] = r.Dist
	}

	// 更新统计信息
	h.updateSearchStats()
	h.logger.Debug("Search completed", "k", k, "found", len(results), "best_score", dists[0])

	return ids, dists, nil
}

// hnswInsert HNSW插入算法核心
func (h *HNSWIndex) hnswInsert(entry *VectorEntry, entryPointID string) error {
	// 从顶层开始，逐层向下
	maxLevel := h.getMaxLevel()

	// 当前最近邻
	nearest := []string{entryPointID}

	// 从上到下搜索最近邻
	for level := maxLevel; level >= 0; level-- {
		if level > entry.Level {
			// 在高层搜索最近邻
			nearest, _ = h.searchLayer(entry.Vector, nearest, 1, level)
		} else {
			// 在当前层建立连接
			candidates := make([]string, len(nearest))
			copy(candidates, nearest)

			// 搜索更多候选
			candidates, _ = h.searchLayer(entry.Vector, candidates, int(h.eps), level)

			// 选择M个最近邻作为邻居
			neighbors := h.selectNeighbors(entry.Vector, candidates, h.m, level)

			// 建立双向连接
			for _, neighborID := range neighbors {
				h.addConnection(level, entry.ID, neighborID, entry.Vector)
				h.addConnection(level, neighborID, entry.ID, entry.Vector)
			}

			// 为下一层准备
			nearest = candidates
		}
	}

	// 保存向量条目
	return h.saveVectorEntry(entry)
}

// hnswSearch HNSW搜索算法核心
func (h *HNSWIndex) hnswSearch(ctx context.Context, query []float64, entryPointID string, k int) ([]*SearchResult, error) {
	// Check for context cancellation at the start of search
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	maxLevel := h.getMaxLevel()

	// 从顶层开始
	nearest := []string{entryPointID}

	// 从上到下搜索，每层缩小搜索范围
	for level := maxLevel; level >= 0; level-- {
		// 在当前层搜索
		ef := h.ef
		if level == 0 {
			ef = max(h.ef, k*2) // 底层使用更大的ef
		}

		candidates, _ := h.searchLayer(query, nearest, ef, level)
		nearest = candidates
	}

	// 在底层进行精确搜索
	candidates, dists := h.searchLayer(query, nearest, k, 0)

	// 构建结果
	results := make([]*SearchResult, len(candidates))
	for i, id := range candidates {
		results[i] = &SearchResult{
			ID:   id,
			Dist: dists[i],
		}
	}

	// 按距离排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Dist < results[j].Dist
	})

	return results, nil
}

// searchLayer 在指定层搜索
func (h *HNSWIndex) searchLayer(query []float64, entryPoints []string, ef int, level int) ([]string, []float64) {
	// 使用最大堆维护候选集
	type candidate struct {
		id   string
		dist float64
	}

	var candidates []candidate
	var visited = make(map[string]bool)

	// 初始化候选集
	for _, id := range entryPoints {
		vector, err := h.getVector(id)
		if err != nil {
			continue
		}
		dist := h.distance(query, vector)
		candidates = append(candidates, candidate{id, dist})
		visited[id] = true
	}

	// 优先队列实现
	for len(candidates) > 0 {
		// 找到最近的未访问候选
		minIdx := 0
		for i := 1; i < len(candidates); i++ {
			if candidates[i].dist < candidates[minIdx].dist {
				minIdx = i
			}
		}

		current := candidates[minIdx]
		candidates = append(candidates[:minIdx], candidates[minIdx+1:]...)

		// 检查是否可以停止
		if len(candidates) >= ef {
			maxDist := candidates[0].dist
			for _, c := range candidates {
				if c.dist > maxDist {
					maxDist = c.dist
				}
			}
			if current.dist > maxDist {
				break
			}
		}

		// 获取邻居
		neighbors, err := h.getNeighbors(current.id, level)
		if err != nil {
			continue
		}

		// 处理邻居
		for _, neighborID := range neighbors {
			if visited[neighborID] {
				continue
			}
			visited[neighborID] = true

			vector, err := h.getVector(neighborID)
			if err != nil {
				continue
			}

			dist := h.distance(query, vector)
			if len(candidates) < ef {
				candidates = append(candidates, candidate{neighborID, dist})
			} else if dist < candidates[0].dist {
				candidates[0] = candidate{neighborID, dist}
			}
		}
	}

	// 返回结果
	resultIDs := make([]string, len(candidates))
	resultDists := make([]float64, len(candidates))
	for i, c := range candidates {
		resultIDs[i] = c.id
		resultDists[i] = c.dist
	}

	return resultIDs, resultDists
}

// selectNeighbors 选择最近的邻居
func (h *HNSWIndex) selectNeighbors(query []float64, candidates []string, M int, level int) []string {
	type candidate struct {
		id   string
		dist float64
	}

	var results []candidate

	for _, id := range candidates {
		vector, err := h.getVector(id)
		if err != nil {
			continue
		}
		dist := h.distance(query, vector)
		results = append(results, candidate{id, dist})
	}

	// 按距离排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].dist < results[j].dist
	})

	// 返回前M个
	if len(results) > M {
		results = results[:M]
	}

	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.id
	}

	return ids
}

// distance 计算距离
func (h *HNSWIndex) distance(a, b []float64) float64 {
	switch h.config.DistanceType {
	case DistanceTypeCosine:
		return h.cosineDistance(a, b)
	case DistanceTypeL2:
		return h.l2Distance(a, b)
	case DistanceTypeInnerProduct:
		return -h.innerProduct(a, b)
	default:
		return h.l2Distance(a, b)
	}
}

// cosineDistance 余弦距离
func (h *HNSWIndex) cosineDistance(a, b []float64) float64 {
	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 1.0
	}
	return 1.0 - dotProduct/(math.Sqrt(normA)*math.Sqrt(normB))
}

// l2Distance 欧几里得距离
func (h *HNSWIndex) l2Distance(a, b []float64) float64 {
	var sum float64
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return math.Sqrt(sum)
}

// innerProduct 内积
func (h *HNSWIndex) innerProduct(a, b []float64) float64 {
	var sum float64
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

// getRandomLevel 生成随机层级
func (h *HNSWIndex) getRandomLevel() int {
	level := 0
	for rand.Float64() < math.Exp(-1.0/h.ml) {
		level++
	}
	return level
}

// 辅助方法
func (h *HNSWIndex) getVectorKey(id string) []byte {
	return []byte(h.config.Prefix + "vector:" + id)
}

func (h *HNSWIndex) getNeighborsKey(id string, level int) []byte {
	return []byte(fmt.Sprintf("%sneighbors:%d:%s", h.config.Prefix, level, id))
}

func (h *HNSWIndex) getEntryPointKey() []byte {
	return []byte(h.config.Prefix + "entrypoint")
}

func (h *HNSWIndex) saveVectorEntry(entry *VectorEntry) error {
	if entry == nil {
		return fmt.Errorf("vector entry cannot be nil")
	}

	if entry.ID == "" {
		return fmt.Errorf("vector entry ID cannot be empty")
	}

	if len(entry.Vector) != h.config.Dimension {
		return fmt.Errorf("vector dimension mismatch: expected %d, got %d", h.config.Dimension, len(entry.Vector))
	}

	// Validate vector values before saving
	for i, v := range entry.Vector {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("invalid vector value at index %d: %v", i, v)
		}
	}

	// 保存向量数据
	key := h.getVectorKey(entry.ID)
	value := make([]byte, len(entry.Vector)*8)
	for i, v := range entry.Vector {
		binary.LittleEndian.PutUint64(value[i*8:], math.Float64bits(v))
	}

	if err := h.db.Set(key, value, nil); err != nil {
		return fmt.Errorf("save vector data: %w", err)
	}

	// 保存元数据
	metaKey := []byte(h.config.Prefix + "meta:" + entry.ID)
	metaValue := []byte(fmt.Sprintf("%d", entry.Level))
	if err := h.db.Set(metaKey, metaValue, nil); err != nil {
		// Attempt to rollback vector data
		_ = h.db.Delete(key, nil)
		return fmt.Errorf("save vector metadata: %w", err)
	}

	return nil
}

func (h *HNSWIndex) getVector(id string) ([]float64, error) {
	key := h.getVectorKey(id)
	value, closer, err := h.db.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, fmt.Errorf("vector not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get vector: %w", err)
	}
	defer func() {
		if closeErr := closer.Close(); closeErr != nil {
			h.logger.Warn("Failed to close vector reader", "id", id, "error", closeErr)
		}
	}()

	expectedLen := h.config.Dimension * 8
	if len(value) != expectedLen {
		return nil, fmt.Errorf("invalid vector data length: expected %d bytes, got %d", expectedLen, len(value))
	}

	vector := make([]float64, h.config.Dimension)
	for i := 0; i < h.config.Dimension; i++ {
		if i*8+8 > len(value) {
			return nil, fmt.Errorf("vector data truncated at index %d", i)
		}
		vector[i] = math.Float64frombits(binary.LittleEndian.Uint64(value[i*8:]))
	}

	// Validate loaded vector values
	for i, v := range vector {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("invalid vector value at index %d: %v", i, v)
		}
	}

	return vector, nil
}

func (h *HNSWIndex) addConnection(level int, from, to string, fromVector []float64) error {
	// 实现连接添加逻辑
	// 这里需要维护邻接表，简化实现
	return nil
}

func (h *HNSWIndex) getNeighbors(id string, level int) ([]string, error) {
	// 获取指定层的邻居
	// 这里需要从邻接表读取，简化实现返回空
	return []string{}, nil
}

func (h *HNSWIndex) getEntryPoint() (string, error) {
	value, closer, err := h.db.Get(h.getEntryPointKey())
	if err != nil {
		if err == pebble.ErrNotFound {
			return "", nil
		}
		return "", err
	}
	defer closer.Close()
	return string(value), nil
}

func (h *HNSWIndex) setEntryPoint(id string, level int) error {
	value := []byte(fmt.Sprintf("%s:%d", id, level))
	return h.db.Set(h.getEntryPointKey(), value, nil)
}

func (h *HNSWIndex) getMaxLevel() int {
	// 从存储中获取最大层级，简化实现
	return 0
}

func (h *HNSWIndex) insertFirst(entry *VectorEntry) error {
	// 保存第一个向量
	err := h.saveVectorEntry(entry)
	if err != nil {
		return err
	}

	// 设置为入口点
	return h.setEntryPoint(entry.ID, entry.Level)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GetStats 获取统计信息的线程安全副本
func (h *HNSWIndex) GetStats() *Stats {
	return h.stats.Clone()
}

// Close 关闭索引
func (h *HNSWIndex) Close() error {
	// Pebble由外部管理
	h.logger.Info("HNSW index closed", "final_vector_count", h.stats.VectorCount())
	return nil
}

// Helper functions for validation and error handling

// validateConfig validates HNSW configuration
func validateConfig(config *Config) error {
	if config.Dimension <= 0 || config.Dimension > 10000 {
		return fmt.Errorf("dimension must be between 1 and 10000")
	}

	if config.M <= 0 || config.M > 100 {
		return fmt.Errorf("M must be between 1 and 100")
	}

	if config.EF <= 0 || config.EF > 1000 {
		return fmt.Errorf("EF must be between 1 and 1000")
	}

	if config.ML <= 0 || config.ML > 10 {
		return fmt.Errorf("ML must be between 0 and 10")
	}

	if config.EPS <= 0 || config.EPS > 1000 {
		return fmt.Errorf("EPS must be between 1 and 1000")
	}

	if config.Prefix == "" {
		return fmt.Errorf("prefix cannot be empty")
	}

	return nil
}

// testConnection tests the database connection
func (h *HNSWIndex) testConnection() error {
	testKey := []byte(h.config.Prefix + "test")
	testValue := []byte("test")

	if err := h.db.Set(testKey, testValue, nil); err != nil {
		return fmt.Errorf("failed to write test key: %w", err)
	}

	if err := h.db.Delete(testKey, nil); err != nil {
		return fmt.Errorf("failed to delete test key: %w", err)
	}

	return nil
}

// validateInsertInput validates insert operation inputs
func (h *HNSWIndex) validateInsertInput(id string, vector []float64) error {
	if id == "" {
		return fmt.Errorf("vector ID cannot be empty")
	}

	if len(id) > 256 {
		return fmt.Errorf("vector ID too long (max 256 characters)")
	}

	if len(vector) != h.config.Dimension {
		return fmt.Errorf("vector dimension mismatch: expected %d, got %d", h.config.Dimension, len(vector))
	}

	// Validate vector values
	for i, v := range vector {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("invalid vector value at index %d: %v", i, v)
		}
	}

	return nil
}

// validateSearchInput validates search operation inputs
func (h *HNSWIndex) validateSearchInput(query []float64, k int) error {
	if len(query) != h.config.Dimension {
		return fmt.Errorf("query dimension mismatch: expected %d, got %d", h.config.Dimension, len(query))
	}

	if k <= 0 || k > 1000 {
		return fmt.Errorf("k must be between 1 and 1000")
	}

	// Validate query vector values
	for i, v := range query {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("invalid query value at index %d: %v", i, v)
		}
	}

	return nil
}

// vectorExists checks if a vector already exists
func (h *HNSWIndex) vectorExists(id string) (bool, error) {
	key := h.getVectorKey(id)
	_, closer, err := h.db.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return false, nil
		}
		return false, fmt.Errorf("check vector existence: %w", err)
	}
	closer.Close()
	return true, nil
}

// deleteVector removes a vector from storage
func (h *HNSWIndex) deleteVector(id string) error {
	key := h.getVectorKey(id)
	if err := h.db.Delete(key, nil); err != nil {
		return fmt.Errorf("delete vector: %w", err)
	}

	// Also delete metadata
	metaKey := []byte(h.config.Prefix + "meta:" + id)
	_ = h.db.Delete(metaKey, nil) // Ignore metadata deletion errors

	return nil
}

// updateInsertStats updates insertion statistics atomically
func (h *HNSWIndex) updateInsertStats() {
	h.stats.AddVectorCount(1)
	h.stats.AddInsertCount(1)

	// Update timestamp under lock
	h.stats.mu.Lock()
	h.stats.lastInsertTime = time.Now().UnixNano()
	h.stats.mu.Unlock()
}

// updateSearchStats updates search statistics atomically
func (h *HNSWIndex) updateSearchStats() {
	h.stats.AddSearchCount(1)

	// Update timestamp under lock
	h.stats.mu.Lock()
	h.stats.lastSearchTime = time.Now().UnixNano()
	h.stats.mu.Unlock()
}
