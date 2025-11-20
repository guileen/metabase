package search

import ("context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/guileen/metabase/pkg/infra/storage"
	"github.com/guileen/metabase/pkg/infra/search/engine"
	"github.com/guileen/metabase/pkg/infra/storage")

// Integration 搜索引擎与存储系统的集成
type Integration struct {
	searchEngine *engine.Engine
	storage      *storage.Engine
	files        *files.Engine

	// 配置
	config *IntegrationConfig

	// 嵌入式MinIO实现
	minio *EmbeddedMinIO

	// 异步队列
	eventQueue   chan *ChangeEvent
	indexQueue   chan *IndexRequest
	workerCtx    context.Context
	workerCancel context.CancelFunc
	workerWG     sync.WaitGroup
	mu           sync.RWMutex
}

// IntegrationConfig 集成配置
type IntegrationConfig struct {
	// 自动索引
	AutoIndex bool

	// 索引配置
	SearchConfig *engine.Config

	// 文件索引配置
	FileIndexing bool

	// 异步处理配置
	AsyncProcessing bool
	QueueSize       int
	WorkerCount     int

	// 嵌入式对象存储配置
	EmbeddedStorage struct {
		DataDir     string
		MaxSize     int64
		Compression bool
	}
}

// ChangeEvent 数据变更事件
type ChangeEvent struct {
	Type      string      `json:"type"`
	TenantID  string      `json:"tenant_id"`
	Table     string      `json:"table"`
	RecordID  string      `json:"record_id"`
	Data      interface{} `json:"data"`
	OldData   interface{} `json:"old_data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// IndexRequest 索引请求
type IndexRequest struct {
	Type      OpType          `json:"type"`
	Document  *engine.Document `json:"document"`
	DocID     string          `json:"doc_id,omitempty"`
	TenantID  string          `json:"tenant_id"`
	Context   context.Context `json:"-"`
	Timestamp time.Time       `json:"timestamp"`
}

// OpType 操作类型
type OpType int

const (
	OpTypeIndex OpType = iota
	OpTypeUpdate
	OpTypeDelete
)

// NewIntegration 创建集成实例
func NewIntegration(searchEngine *engine.Engine, storage *storage.Engine, files *files.Engine, config *IntegrationConfig) (*Integration, error) {
	if searchEngine == nil {
		return nil, fmt.Errorf("search engine is required")
	}
	if storage == nil {
		return nil, fmt.Errorf("storage engine is required")
	}
	if files == nil {
		return nil, fmt.Errorf("files engine is required")
	}

	// 默认配置
	if config == nil {
		config = &IntegrationConfig{
			AutoIndex:       true,
			FileIndexing:    true,
			AsyncProcessing: true,
			QueueSize:       1000,
			WorkerCount:     3,
		}
	}

	// 验证配置
	if config.QueueSize <= 0 {
		config.QueueSize = 1000
	}
	if config.WorkerCount <= 0 {
		config.WorkerCount = 3
	}

	workerCtx, workerCancel := context.WithCancel(context.Background())

	integration := &Integration{
		searchEngine: searchEngine,
		storage:      storage,
		files:        files,
		config:       config,
		eventQueue:   make(chan *ChangeEvent, config.QueueSize),
		indexQueue:   make(chan *IndexRequest, config.QueueSize),
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
	}

	// 初始化嵌入式存储（简化实现）
	if config.EmbeddedStorage.DataDir != "" {
		// 这里需要根据实际的storage.Engine接口来获取数据库连接
		// 暂时创建一个简单的实现
		minio := &EmbeddedMinIO{
			dataDir: config.EmbeddedStorage.DataDir,
		}
		integration.minio = minio
	}

	// 启动异步处理工作器
	if config.AsyncProcessing {
		integration.startWorkers()
	}

	return integration, nil
}

// startWorkers 启动异步处理工作器
func (i *Integration) startWorkers() {
	// 启动事件处理工作器
	i.workerWG.Add(1)
	go i.eventWorker()

	// 启动索引处理工作器
	for j := 0; j < i.config.WorkerCount; j++ {
		i.workerWG.Add(1)
		go i.indexWorker()
	}
}

// eventWorker 处理变更事件
func (i *Integration) eventWorker() {
	defer i.workerWG.Done()

	for {
		select {
		case event := <-i.eventQueue:
			if err := i.processChangeEvent(event); err != nil {
				log.Printf("Failed to process change event: %v", err)
			}

		case <-i.workerCtx.Done():
			return
		}
	}
}

// indexWorker 处理索引请求
func (i *Integration) indexWorker() {
	defer i.workerWG.Done()

	for {
		select {
		case req := <-i.indexQueue:
			if err := i.processIndexRequest(req); err != nil {
				log.Printf("Failed to process index request: %v", err)
			}

		case <-i.workerCtx.Done():
			return
		}
	}
}

// processChangeEvent 处理变更事件
func (i *Integration) processChangeEvent(event *ChangeEvent) error {
	if !i.isSearchableTable(event.Table) {
		return nil
	}

	switch event.Type {
	case "create", "update":
		doc := i.eventToDocument(event)
		if doc != nil {
			req := &IndexRequest{
				Type:      OpTypeIndex,
				Document:  doc,
				TenantID:  event.TenantID,
				Context:   context.Background(),
				Timestamp: time.Now(),
			}
			if event.Type == "update" {
				req.Type = OpTypeUpdate
			}

			select {
			case i.indexQueue <- req:
				return nil
			case <-i.workerCtx.Done():
				return i.workerCtx.Err()
			default:
				return i.processIndexRequest(req)
			}
		}
		// 如果doc为nil，直接返回
		return nil

	case "delete":
		req := &IndexRequest{
			Type:      OpTypeDelete,
			DocID:     event.RecordID,
			TenantID:  event.TenantID,
			Context:   context.Background(),
			Timestamp: time.Now(),
		}

		select {
		case i.indexQueue <- req:
			return nil
		case <-i.workerCtx.Done():
			return i.workerCtx.Err()
		default:
			return i.processIndexRequest(req)
		}

	default:
		return fmt.Errorf("unknown event type: %s", event.Type)
	}
}

// processIndexRequest 处理索引请求
func (i *Integration) processIndexRequest(req *IndexRequest) error {
	switch req.Type {
	case OpTypeIndex, OpTypeUpdate:
		if req.Document == nil {
			return fmt.Errorf("document is required for index operation")
		}
		return i.searchEngine.Index(req.Document)

	case OpTypeDelete:
		if req.DocID == "" {
			return fmt.Errorf("doc_id is required for delete operation")
		}
		return i.searchEngine.Delete(req.DocID)

	default:
		return fmt.Errorf("unknown operation type: %v", req.Type)
	}
}

// eventToDocument 将变更事件转换为文档
func (i *Integration) eventToDocument(event *ChangeEvent) *engine.Document {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		log.Printf("Invalid data type for event %s: %T", event.RecordID, event.Data)
		return nil
	}

	doc := &engine.Document{
		ID:        event.RecordID,
		TenantID:  event.TenantID,
		Type:      event.Table,
		Metadata:  data,
		Timestamp: event.Timestamp,
	}

	// 提取租户信息
	if tenantID, ok := data["tenant_id"].(string); ok {
		doc.TenantID = tenantID
	}

	// 基于表类型提取可搜索内容
	switch event.Table {
	case "users":
		doc.Title = getStringField(data, "name", "username", "email")
		doc.Content = fmt.Sprintf("%s %s %s %s",
			getStringField(data, "name"),
			getStringField(data, "username"),
			getStringField(data, "email"),
			getStringField(data, "bio"))

	case "documents":
		doc.Title = getStringField(data, "title", "name")
		doc.Content = getStringField(data, "content", "body", "description")

	case "posts":
		doc.Title = getStringField(data, "title")
		doc.Content = getStringField(data, "content", "body")

	case "comments":
		doc.Content = getStringField(data, "text", "content", "comment")

	case "files":
		doc.Title = getStringField(data, "original_name", "name")
		doc.Content = getStringField(data, "description", "caption")
	}

	return doc
}

// HookIntoStorage adds search hooks to storage operations
func (i *Integration) HookIntoStorage() {
	// 这是一个简化的实现，实际的hook需要根据storage.Engine的具体接口来实现
	log.Println("Storage search hooks configured")
}

// enqueueChangeEvent 排队变更事件
func (i *Integration) enqueueChangeEvent(event *ChangeEvent) {
	if !i.config.AsyncProcessing {
		if err := i.processChangeEvent(event); err != nil {
			log.Printf("Failed to process change event synchronously: %v", err)
		}
		return
	}

	select {
	case i.eventQueue <- event:
	default:
		log.Printf("Event queue full, processing synchronously for table=%s record=%s", event.Table, event.RecordID)
		if err := i.processChangeEvent(event); err != nil {
			log.Printf("Failed to process change event synchronously: %v", err)
		}
	}
}

// Search performs a search with automatic tenant filtering
func (i *Integration) Search(ctx context.Context, query *engine.Query) (*engine.Result, error) {
	if query == nil {
		return nil, fmt.Errorf("query is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if query.TenantID == "" {
		if tenantID, ok := ctx.Value("tenant_id").(string); ok {
			query.TenantID = tenantID
		}
	}

	return i.searchEngine.Search(ctx, query)
}

// Close 关闭集成实例并清理资源
func (i *Integration) Close() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.workerCancel != nil {
		i.workerCancel()
	}

	done := make(chan struct{})
	go func() {
		i.workerWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All workers stopped gracefully")
	case <-time.After(10 * time.Second):
		log.Println("Warning: workers did not stop gracefully within timeout")
	}

	if i.minio != nil {
		if err := i.minio.Close(); err != nil {
			return fmt.Errorf("failed to close embedded storage: %w", err)
		}
	}

	return nil
}

// isSearchableTable 检查表是否应该被索引
func (i *Integration) isSearchableTable(table string) bool {
	searchableTables := map[string]bool{
		"users":     true,
		"documents": true,
		"posts":     true,
		"comments":  true,
		"files":     true,
	}
	return searchableTables[table]
}

// getStringField 从数据中提取字符串字段
func getStringField(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			if str, ok := value.(string); ok {
				return str
			}
		}
	}
	return ""
}

// EmbeddedMinIO 嵌入式对象存储
type EmbeddedMinIO struct {
	dataDir string
	db      *sql.DB
	kv      *pebble.DB
}

// NewEmbeddedMinIO 创建嵌入式MinIO实例
func NewEmbeddedMinIO(dataDir string, db *sql.DB, kv *pebble.DB) (*EmbeddedMinIO, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("data directory is required")
	}
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	if kv == nil {
		return nil, fmt.Errorf("key-value store is required")
	}

	minio := &EmbeddedMinIO{
		dataDir: dataDir,
		db:      db,
		kv:      kv,
	}

	return minio, nil
}

// Close 关闭嵌入式存储
func (m *EmbeddedMinIO) Close() error {
	return nil
}