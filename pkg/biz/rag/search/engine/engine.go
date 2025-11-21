package engine

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/guileen/metabase/pkg/biz/rag/search/index"
	"github.com/guileen/metabase/pkg/biz/rag/search/vector"
)

// Engine 统一搜索引擎，集成全文搜索和向量搜索
type Engine struct {
	db       *sql.DB
	kv       *pebble.DB
	fullText *index.InvertedIndex
	vector   *vector.HNSWIndex

	// 索引队列
	indexQueue chan *IndexTask
	workers    int

	// 统计信息
	stats *Stats

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
}

// Config 搜索引擎配置
type Config struct {
	// SQLite 数据库连接
	DB *sql.DB

	// Pebble KV 存储
	KV *pebble.DB

	// 索引配置
	IndexConfig *index.Config

	// 向量索引配置
	VectorConfig *vector.Config

	// 并发工作协程数
	Workers int

	// 队列大小
	QueueSize int
}

// IndexTask 索引任务
type IndexTask struct {
	ID       string
	Type     TaskType
	Document *Document
	Context  context.Context
	Result   chan error
}

// TaskType 任务类型
type TaskType int

const (
	TaskTypeIndex TaskType = iota
	TaskTypeUpdate
	TaskTypeDelete
)

// Document 文档结构
type Document struct {
	ID        string
	TenantID  string
	Type      string // "table", "file", "record"等
	Title     string
	Content   string
	Metadata  map[string]interface{}
	Vector    []float64 // 预计算的向量
	Timestamp time.Time
}

// Query 查询结构
type Query struct {
	Text     string                 // 文本查询
	Vector   []float64              // 向量查询
	Type     QueryType              // 查询类型
	TenantID string                 // 租户ID
	Filters  map[string]interface{} // 过滤条件
	Limit    int                    // 结果数量
	Offset   int                    // 偏移量
}

// QueryType 查询类型
type QueryType int

const (
	QueryTypeFullText QueryType = iota
	QueryTypeVector
	QueryTypeHybrid // 混合搜索
	QueryTypeSQL    // 原生SQL查询
)

// Result 搜索结果
type Result struct {
	Documents []*Document
	Scores    []float64
	Total     int
	QueryTime time.Duration
}

// Stats 统计信息 - 使用原子操作和读写锁保证线程安全
type Stats struct {
	// 原子操作的计数器
	documentsIndexed int64
	queriesProcessed int64

	// 需要互斥锁保护的非原子字段
	mu               sync.RWMutex
	indexQueueSize   int64
	averageQueryTime time.Duration
	lastIndexTime    time.Time
	lastQueryTime    time.Time
}

// DocumentsIndexed returns the current documents indexed count atomically
func (s *Stats) DocumentsIndexed() int64 {
	return atomic.LoadInt64(&s.documentsIndexed)
}

// QueriesProcessed returns the current queries processed count atomically
func (s *Stats) QueriesProcessed() int64 {
	return atomic.LoadInt64(&s.queriesProcessed)
}

// AddDocumentsIndexed atomically adds to the documents indexed count
func (s *Stats) AddDocumentsIndexed(delta int64) {
	atomic.AddInt64(&s.documentsIndexed, delta)
}

// AddQueriesProcessed atomically adds to the queries processed count
func (s *Stats) AddQueriesProcessed(delta int64) {
	atomic.AddInt64(&s.queriesProcessed, delta)
}

// UpdateTimingStats updates timing statistics under lock
func (s *Stats) UpdateTimingStats(averageQueryTime time.Duration, lastIndexTime, lastQueryTime time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.averageQueryTime = averageQueryTime
	s.lastIndexTime = lastIndexTime
	s.lastQueryTime = lastQueryTime
}

// SetIndexQueueSize updates the index queue size under lock
func (s *Stats) SetIndexQueueSize(size int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.indexQueueSize = size
}

// Clone returns a thread-safe copy of the stats
func (s *Stats) Clone() *Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &Stats{
		documentsIndexed: atomic.LoadInt64(&s.documentsIndexed),
		queriesProcessed: atomic.LoadInt64(&s.queriesProcessed),
		indexQueueSize:   s.indexQueueSize,
		averageQueryTime: s.averageQueryTime,
		lastIndexTime:    s.lastIndexTime,
		lastQueryTime:    s.lastQueryTime,
	}
}

// NewEngine 创建搜索引擎
func NewEngine(config *Config) (*Engine, error) {
	if config.DB == nil || config.KV == nil {
		return nil, fmt.Errorf("database and kv storage required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建倒排索引
	fullText, err := index.NewInvertedIndex(config.DB, config.IndexConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create fulltext index: %w", err)
	}

	// 创建向量索引
	vectorIndex, err := vector.NewHNSWIndex(config.KV, config.VectorConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create vector index: %w", err)
	}

	workers := config.Workers
	if workers <= 0 {
		workers = 4
	}

	queueSize := config.QueueSize
	if queueSize <= 0 {
		queueSize = 1000
	}

	engine := &Engine{
		db:         config.DB,
		kv:         config.KV,
		fullText:   fullText,
		vector:     vectorIndex,
		indexQueue: make(chan *IndexTask, queueSize),
		workers:    workers,
		stats:      &Stats{},
		ctx:        ctx,
		cancel:     cancel,
	}

	// 启动工作协程
	for i := 0; i < workers; i++ {
		engine.wg.Add(1)
		go engine.indexWorker(i)
	}

	return engine, nil
}

// Search 执行搜索
func (e *Engine) Search(ctx context.Context, query *Query) (*Result, error) {
	defer func() {
		e.stats.AddQueriesProcessed(1)
		e.stats.UpdateTimingStats(e.stats.averageQueryTime, e.stats.lastIndexTime, time.Now())
	}()

	switch query.Type {
	case QueryTypeFullText:
		return e.searchFullText(ctx, query)
	case QueryTypeVector:
		return e.searchVector(ctx, query)
	case QueryTypeHybrid:
		return e.searchHybrid(ctx, query)
	case QueryTypeSQL:
		return e.searchSQL(ctx, query)
	default:
		return nil, fmt.Errorf("unsupported query type: %v", query.Type)
	}
}

// Index 异步索引文档
func (e *Engine) Index(doc *Document) error {
	task := &IndexTask{
		ID:       doc.ID,
		Type:     TaskTypeIndex,
		Document: doc,
		Context:  context.Background(),
		Result:   make(chan error, 1),
	}

	select {
	case e.indexQueue <- task:
		return <-task.Result
	case <-time.After(5 * time.Second):
		return fmt.Errorf("index queue timeout")
	}
}

// Update 更新文档
func (e *Engine) Update(doc *Document) error {
	task := &IndexTask{
		ID:       doc.ID,
		Type:     TaskTypeUpdate,
		Document: doc,
		Context:  context.Background(),
		Result:   make(chan error, 1),
	}

	select {
	case e.indexQueue <- task:
		return <-task.Result
	default:
		return fmt.Errorf("index queue full")
	}
}

// Delete 删除文档
func (e *Engine) Delete(id string) error {
	task := &IndexTask{
		ID:      id,
		Type:    TaskTypeDelete,
		Context: context.Background(),
		Result:  make(chan error, 1),
	}

	select {
	case e.indexQueue <- task:
		return <-task.Result
	default:
		return fmt.Errorf("index queue full")
	}
}

// indexWorker 索引工作协程
func (e *Engine) indexWorker(id int) {
	defer e.wg.Done()

	for {
		select {
		case task := <-e.indexQueue:
			var err error
			switch task.Type {
			case TaskTypeIndex:
				err = e.indexDocument(task.Document)
			case TaskTypeUpdate:
				err = e.updateDocument(task.Document)
			case TaskTypeDelete:
				err = e.deleteDocument(task.ID)
			}

			select {
			case task.Result <- err:
			default:
			}

			e.stats.SetIndexQueueSize(int64(len(e.indexQueue)))
			e.stats.UpdateTimingStats(e.stats.averageQueryTime, time.Now(), e.stats.lastQueryTime)

		case <-e.ctx.Done():
			return
		}
	}
}

// toIndexDocument converts engine Document to index DocumentIndex
func (e *Engine) toIndexDocument(doc *Document) *index.DocumentIndex {
	return &index.DocumentIndex{
		ID:       doc.ID,
		TenantID: doc.TenantID,
		Type:     doc.Type,
		Title:    doc.Title,
		Content:  doc.Content,
		Metadata: doc.Metadata,
	}
}

// fromIndexDocument converts index DocumentIndex to engine Document
func (e *Engine) fromIndexDocument(doc *index.DocumentIndex) *Document {
	return &Document{
		ID:        doc.ID,
		TenantID:  doc.TenantID,
		Type:      doc.Type,
		Title:     doc.Title,
		Content:   doc.Content,
		Metadata:  doc.Metadata,
		Timestamp: time.Now(), // Use current time as fallback
	}
}

// indexDocument 索引单个文档
func (e *Engine) indexDocument(doc *Document) error {
	// 转换为倒排索引文档格式
	indexDoc := e.toIndexDocument(doc)

	// 索引到倒排索引
	if err := e.fullText.Index(indexDoc); err != nil {
		return fmt.Errorf("fulltext index: %w", err)
	}

	// 索引到向量索引
	if len(doc.Vector) > 0 {
		if err := e.vector.Insert(context.Background(), doc.ID, doc.Vector); err != nil {
			return fmt.Errorf("vector index: %w", err)
		}
	}

	e.stats.AddDocumentsIndexed(1)

	return nil
}

// updateDocument 更新文档
func (e *Engine) updateDocument(doc *Document) error {
	// 先删除旧索引
	_ = e.deleteDocument(doc.ID)
	// 重新索引
	return e.indexDocument(doc)
}

// deleteDocument 删除文档
func (e *Engine) deleteDocument(id string) error {
	// 从倒排索引删除
	if err := e.fullText.Delete(id); err != nil {
		return fmt.Errorf("fulltext delete: %w", err)
	}

	// 从向量索引删除
	if err := e.vector.Delete(id); err != nil {
		return fmt.Errorf("vector delete: %w", err)
	}

	return nil
}

// searchFullText 全文搜索
func (e *Engine) searchFullText(ctx context.Context, query *Query) (*Result, error) {
	start := time.Now()
	indexDocs, scores, err := e.fullText.Search(ctx, query.Text, query.TenantID, query.Limit)
	if err != nil {
		return nil, err
	}

	// 转换为engine Document类型
	docs := make([]*Document, len(indexDocs))
	for i, indexDoc := range indexDocs {
		docs[i] = e.fromIndexDocument(indexDoc)
	}

	return &Result{
		Documents: docs,
		Scores:    scores,
		Total:     len(docs),
		QueryTime: time.Since(start),
	}, nil
}

// searchVector 向量搜索
func (e *Engine) searchVector(ctx context.Context, query *Query) (*Result, error) {
	start := time.Now()
	if len(query.Vector) == 0 {
		return nil, fmt.Errorf("vector query requires vector")
	}

	ids, scores, err := e.vector.Search(ctx, query.Vector, query.Limit)
	if err != nil {
		return nil, err
	}

	// 根据ID获取文档
	docs := make([]*Document, 0, len(ids))
	for _, id := range ids {
		doc, err := e.getDocumentByID(id, query.TenantID)
		if err != nil {
			continue
		}
		docs = append(docs, doc)
	}

	return &Result{
		Documents: docs,
		Scores:    scores,
		Total:     len(docs),
		QueryTime: time.Since(start),
	}, nil
}

// searchHybrid 混合搜索
func (e *Engine) searchHybrid(ctx context.Context, query *Query) (*Result, error) {
	start := time.Now()
	// 并行执行全文和向量搜索
	type result struct {
		docs   []*Document
		scores []float64
		err    error
	}

	fullTextCh := make(chan result, 1)
	vectorCh := make(chan result, 1)

	// 全文搜索
	go func() {
		indexDocs, scores, err := e.fullText.Search(ctx, query.Text, query.TenantID, query.Limit*2)
		// 转换文档类型
		docs := make([]*Document, len(indexDocs))
		for i, indexDoc := range indexDocs {
			docs[i] = e.fromIndexDocument(indexDoc)
		}
		fullTextCh <- result{docs, scores, err}
	}()

	// 向量搜索
	go func() {
		if len(query.Vector) > 0 {
			ids, scores, err := e.vector.Search(ctx, query.Vector, query.Limit*2)
			if err == nil {
				docs := make([]*Document, 0, len(ids))
				for _, id := range ids {
					doc, err := e.getDocumentByID(id, query.TenantID)
					if err == nil {
						docs = append(docs, doc)
					}
				}
				vectorCh <- result{docs, scores, nil}
				return
			}
		}
		vectorCh <- result{nil, nil, fmt.Errorf("no vector query")}
	}()

	// 等待结果
	ftResult := <-fullTextCh
	vResult := <-vectorCh

	if ftResult.err != nil && vResult.err != nil {
		return nil, fmt.Errorf("both search methods failed: ft=%v, vector=%v", ftResult.err, vResult.err)
	}

	// 合并和重排序
	mergedDocs, mergedScores := e.mergeResults(
		ftResult.docs, ftResult.scores, 0.6, // 全文权重
		vResult.docs, vResult.scores, 0.4, // 向量权重
		query.Limit,
	)

	return &Result{
		Documents: mergedDocs,
		Scores:    mergedScores,
		Total:     len(mergedDocs),
		QueryTime: time.Since(start),
	}, nil
}

// searchSQL SQL查询
func (e *Engine) searchSQL(ctx context.Context, query *Query) (*Result, error) {
	// 实现SQL查询逻辑
	// 这里可以根据需要实现复杂的SQL查询
	return nil, fmt.Errorf("SQL search not implemented yet")
}

// getDocumentByID 根据ID获取文档
func (e *Engine) getDocumentByID(id, tenantID string) (*Document, error) {
	// 从数据库获取文档
	var doc Document
	err := e.db.QueryRowContext(context.Background(),
		"SELECT id, tenant_id, type, title, content, metadata, created_at FROM search_documents WHERE id = ? AND tenant_id = ?",
		id, tenantID).Scan(&doc.ID, &doc.TenantID, &doc.Type, &doc.Title, &doc.Content, &doc.Metadata, &doc.Timestamp)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// mergeResults 合并搜索结果
func (e *Engine) mergeResults(docs1 []*Document, scores1 []float64, weight1 float64,
	docs2 []*Document, scores2 []float64, weight2 float64, limit int) ([]*Document, []float64) {

	// 简单的合并算法，实际可以使用更复杂的排序算法
	docScoreMap := make(map[string]*DocumentScore)

	// 处理第一组结果
	for i, doc := range docs1 {
		score := scores1[i] * weight1
		if existing, ok := docScoreMap[doc.ID]; ok {
			existing.Score += score
		} else {
			docScoreMap[doc.ID] = &DocumentScore{Document: doc, Score: score}
		}
	}

	// 处理第二组结果
	for i, doc := range docs2 {
		score := scores2[i] * weight2
		if existing, ok := docScoreMap[doc.ID]; ok {
			existing.Score += score
		} else {
			docScoreMap[doc.ID] = &DocumentScore{Document: doc, Score: score}
		}
	}

	// 排序并返回top结果
	results := make([]*DocumentScore, 0, len(docScoreMap))
	for _, ds := range docScoreMap {
		results = append(results, ds)
	}

	// 按分数排序
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Score < results[j].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// 限制结果数量
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	docs := make([]*Document, len(results))
	scores := make([]float64, len(results))
	for i, r := range results {
		docs[i] = r.Document
		scores[i] = r.Score
	}

	return docs, scores
}

// DocumentScore 文档和分数
type DocumentScore struct {
	Document *Document
	Score    float64
}

// updateStats 更新统计信息
func (e *Engine) updateStats(fn func(*Stats)) {
	e.stats.mu.Lock()
	defer e.stats.mu.Unlock()
	fn(e.stats)
}

// GetStats 获取统计信息的线程安全副本
func (e *Engine) GetStats() *Stats {
	statsClone := e.stats.Clone()
	// Update current queue size
	statsClone.indexQueueSize = int64(len(e.indexQueue))
	return statsClone
}

// Close 关闭搜索引擎
func (e *Engine) Close() error {
	e.cancel()
	e.wg.Wait()

	if e.fullText != nil {
		e.fullText.Close()
	}

	if e.vector != nil {
		e.vector.Close()
	}

	return nil
}
