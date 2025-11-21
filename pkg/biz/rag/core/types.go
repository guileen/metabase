package core

import (
	"time"

	"github.com/guileen/metabase/pkg/biz/rag/llm"
)

// Document represents a document that can be processed by the RAG system
type Document struct {
	// Core identification
	ID         string `json:"id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	URI        string `json:"uri"`         // Original location/identifier
	SourceType string `json:"source_type"` // filesystem, database, web, etc.

	// Metadata
	Metadata   DocumentMetadata `json:"metadata"`
	Tags       []string         `json:"tags"`
	Categories []string         `json:"categories"`
	Language   string           `json:"language"`

	// Processing information
	ProcessedAt time.Time `json:"processed_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Version     int       `json:"version"`

	// Data source information
	DataSourceID string      `json:"data_source_id"`
	DataSource   interface{} `json:"data_source,omitempty"` // Reference to data source
}

// DocumentMetadata contains metadata about a document
type DocumentMetadata struct {
	// File/directory information (for filesystem sources)
	FilePath  string `json:"file_path,omitempty"`
	FileName  string `json:"file_name,omitempty"`
	FileSize  int64  `json:"file_size,omitempty"`
	FileType  string `json:"file_type,omitempty"`
	Extension string `json:"extension,omitempty"`

	// Time information
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
	AccessedAt time.Time `json:"accessed_at"`

	// Author/ownership information
	Author string `json:"author,omitempty"`
	Owner  string `json:"owner,omitempty"`

	// Content information
	Length    int `json:"length"`     // Content length in characters
	WordCount int `json:"word_count"` // Estimated word count
	LineCount int `json:"line_count"` // Number of lines

	// Custom metadata
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// DocumentChunk represents a chunk of a document
type DocumentChunk struct {
	// Core identification
	ID         string `json:"id"`
	DocumentID string `json:"document_id"`
	Content    string `json:"content"`

	// Position information
	ChunkIndex int `json:"chunk_index"` // Index within the document
	StartPos   int `json:"start_pos"`   // Start character position
	EndPos     int `json:"end_pos"`     // End character position
	StartLine  int `json:"start_line"`  // Start line number
	EndLine    int `json:"end_line"`    // End line number

	// Chunking metadata
	ChunkType  string `json:"chunk_type"`  // sentence, paragraph, semantic, etc.
	ChunkSize  int    `json:"chunk_size"`  // Size in characters
	TokenCount int    `json:"token_count"` // Estimated token count

	// Embedding information
	Embedding      []float64 `json:"embedding,omitempty"`
	EmbeddingModel string    `json:"embedding_model,omitempty"`
	EmbeddingDim   int       `json:"embedding_dim,omitempty"`

	// Metadata
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`

	// Context information
	PrecedingContext string `json:"preceding_context,omitempty"` // Content before this chunk
	FollowingContext string `json:"following_context,omitempty"` // Content after this chunk
}

// RetrievalResult represents a result from document retrieval
type RetrievalResult struct {
	// Document information
	DocumentID string         `json:"document_id"`
	Document   *Document      `json:"document,omitempty"`
	Chunk      *DocumentChunk `json:"chunk"`

	// Relevance scoring
	Score        float64 `json:"score"`         // Overall relevance score
	Similarity   float64 `json:"similarity"`    // Vector similarity score
	KeywordScore float64 `json:"keyword_score"` // Keyword match score
	RerankScore  float64 `json:"rerank_score"`  // Reranking score

	// Match information
	Matches    []TextMatch `json:"matches"`    // Text matches within the chunk
	Highlights []string    `json:"highlights"` // Highlighted passages

	// Metadata
	Explanation string `json:"explanation,omitempty"` // Why this was retrieved
	Method      string `json:"method"`                // retrieval method used
	Position    int    `json:"position"`              // position in results
}

// TextMatch represents a text match within a chunk
type TextMatch struct {
	Text     string  `json:"text"`      // Matched text
	StartPos int     `json:"start_pos"` // Start position
	EndPos   int     `json:"end_pos"`   // End position
	Score    float64 `json:"score"`     // Match score
	Type     string  `json:"type"`      // exact, fuzzy, semantic, etc.
}

// QueryResult represents the result of a RAG query
type QueryResult struct {
	// Query information
	QueryID        string   `json:"query_id"`
	Query          string   `json:"query"`
	ProcessedQuery string   `json:"processed_query"` // Query after preprocessing
	ExpandedTerms  []string `json:"expanded_terms"`  // Expanded query terms

	// Retrieval results
	RetrievalResults []RetrievalResult `json:"retrieval_results"`
	TotalRetrieved   int               `json:"total_retrieved"`
	TotalReturned    int               `json:"total_returned"`

	// Generation results
	GeneratedResponse string   `json:"generated_response"`
	GeneratedAnswer   string   `json:"generated_answer"`  // Extracted answer
	GeneratedSummary  string   `json:"generated_summary"` // Generated summary
	Sources           []Source `json:"sources"`           // Source citations

	// Performance metrics
	QueryTime      time.Duration `json:"query_time"`
	RetrievalTime  time.Duration `json:"retrieval_time"`
	GenerationTime time.Duration `json:"generation_time"`
	TotalTime      time.Duration `json:"total_time"`

	// Usage statistics
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalTokens  int     `json:"total_tokens"`
	Cost         float64 `json:"cost"` // Estimated cost

	// Metadata
	Options          QueryOptions `json:"options"`
	FilterApplied    bool         `json:"filter_applied"`
	RerankingApplied bool         `json:"reranking_applied"`
	CacheHit         bool         `json:"cache_hit"`
	CreatedAt        time.Time    `json:"created_at"`
}

// Source represents a source citation in the generated response
type Source struct {
	DocumentID    string  `json:"document_id"`
	DocumentTitle string  `json:"document_title"`
	DocumentURI   string  `json:"document_uri"`
	ChunkID       string  `json:"chunk_id"`
	Relevance     float64 `json:"relevance"`
	Excerpt       string  `json:"excerpt"`
	PageNumber    int     `json:"page_number,omitempty"`
}

// GenerationResult represents the result of text generation
type GenerationResult struct {
	// Generated content
	Response string `json:"response"`
	Answer   string `json:"answer,omitempty"`
	Summary  string `json:"summary,omitempty"`

	// Prompt information
	UsedPrompt   string `json:"used_prompt"`
	PromptTokens int    `json:"prompt_tokens"`

	// Generation metadata
	Model        string  `json:"model"`
	Temperature  float64 `json:"temperature"`
	MaxTokens    int     `json:"max_tokens"`
	OutputTokens int     `json:"output_tokens"`
	FinishReason string  `json:"finish_reason"`

	// Timing and cost
	Duration time.Duration `json:"duration"`
	Cost     float64       `json:"cost"`

	// Quality metrics
	QualityScore float64 `json:"quality_score,omitempty"`
	Confidence   float64 `json:"confidence,omitempty"`

	// Sources used
	Sources []Source `json:"sources"`

	CreatedAt time.Time `json:"created_at"`
}

// IndexResult represents the result of an indexing operation
type IndexResult struct {
	// Processing statistics
	DocumentsProcessed int `json:"documents_processed"`
	DocumentsUpdated   int `json:"documents_updated"`
	DocumentsAdded     int `json:"documents_added"`
	DocumentsSkipped   int `json:"documents_skipped"`
	DocumentsErrored   int `json:"documents_errored"`

	// Chunk statistics
	ChunksCreated int `json:"chunks_created"`
	ChunksUpdated int `json:"chunks_updated"`
	ChunksDeleted int `json:"chunks_deleted"`

	// Embedding statistics
	EmbeddingsGenerated int           `json:"embeddings_generated"`
	EmbeddingTime       time.Duration `json:"embedding_time"`

	// Performance metrics
	TotalTime      time.Duration `json:"total_time"`
	ProcessingRate float64       `json:"processing_rate"` // docs per second
	Errors         []string      `json:"errors,omitempty"`

	// Data source information
	DataSourceID string `json:"data_source_id"`
	IndexType    string `json:"index_type"` // full, incremental

	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

// SyncResult represents the result of a data source synchronization
type SyncResult struct {
	// Change statistics
	DocumentsAdded     int `json:"documents_added"`
	DocumentsUpdated   int `json:"documents_updated"`
	DocumentsDeleted   int `json:"documents_deleted"`
	DocumentsUnchanged int `json:"documents_unchanged"`

	// Error information
	Errors     []string `json:"errors,omitempty"`
	ErrorCount int      `json:"error_count"`

	// Timing information
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`

	// Sync metadata
	LastSyncTime time.Time `json:"last_sync_time"`
	SyncType     string    `json:"sync_type"` // full, incremental
	DataSourceID string    `json:"data_source_id"`
}

// EmbeddingMatch represents a matching embedding
type EmbeddingMatch struct {
	ChunkID    string         `json:"chunk_id"`
	DocumentID string         `json:"document_id"`
	Score      float64        `json:"score"`
	Distance   float64        `json:"distance"`
	Chunk      *DocumentChunk `json:"chunk,omitempty"`
}

// QueryRecord represents a stored query for caching and analysis
type QueryRecord struct {
	ID             string         `json:"id"`
	Query          string         `json:"query"`
	ProcessedQuery string         `json:"processed_query"`
	Result         *QueryResult   `json:"result,omitempty"`
	UserID         string         `json:"user_id,omitempty"`
	SessionID      string         `json:"session_id,omitempty"`
	DataSourceIDs  []string       `json:"data_source_ids,omitempty"`
	Options        QueryOptions   `json:"options"`
	Feedback       *QueryFeedback `json:"feedback,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	ExpiresAt      time.Time      `json:"expires_at,omitempty"`
}

// QueryFeedback represents user feedback on a query result
type QueryFeedback struct {
	Rating         int       `json:"rating"`                    // 1-5 rating
	Useful         bool      `json:"useful"`                    // Was the result useful?
	Comments       string    `json:"comments,omitempty"`        // User comments
	ClickedSources []string  `json:"clicked_sources,omitempty"` // Sources user clicked
	CreatedAt      time.Time `json:"created_at"`
}

// Configuration types

// IndexOptions defines options for document indexing
type IndexOptions struct {
	// Data source filtering
	DataSourceIDs []string `json:"data_source_ids,omitempty"`
	DocumentIDs   []string `json:"document_ids,omitempty"`
	FilePaths     []string `json:"file_paths,omitempty"`

	// Processing options
	ForceReindex bool `json:"force_reindex"` // Force reindexing all documents
	Incremental  bool `json:"incremental"`   // Use incremental indexing
	BatchSize    int  `json:"batch_size"`    // Batch processing size
	MaxWorkers   int  `json:"max_workers"`   // Maximum parallel workers

	// Filtering options
	IncludePatterns []string `json:"include_patterns"` // File patterns to include
	ExcludePatterns []string `json:"exclude_patterns"` // File patterns to exclude
	MaxFileSize     int64    `json:"max_file_size"`    // Maximum file size to process
	MinFileSize     int64    `json:"min_file_size"`    // Minimum file size to process

	// Content options
	IncludeContent  bool `json:"include_content"`  // Include document content in index
	ExtractMetadata bool `json:"extract_metadata"` // Extract document metadata

	// Timeout options
	Timeout time.Duration `json:"timeout,omitempty"`
	DryRun  bool          `json:"dry_run"` // Simulate without actually indexing
}

// QueryOptions defines options for query execution
type QueryOptions struct {
	// Retrieval options
	RetrievalOptions RetrieveOptions `json:"retrieval"`

	// Generation options
	GenerateOptions GenerateOptions `json:"generate"`

	// Performance options
	EnableCache     bool          `json:"enable_cache"`     // Enable result caching
	CacheTTL        time.Duration `json:"cache_ttl"`        // Cache TTL
	EnableRerank    bool          `json:"enable_rerank"`    // Enable reranking
	EnableStreaming bool          `json:"enable_streaming"` // Enable streaming responses

	// Filtering options
	DataSourceIDs []string   `json:"data_source_ids,omitempty"`
	DocumentIDs   []string   `json:"document_ids,omitempty"`
	FileTypes     []string   `json:"file_types,omitempty"`
	Tags          []string   `json:"tags,omitempty"`
	DateRange     *TimeRange `json:"date_range,omitempty"`

	// Result options
	MaxResults int     `json:"max_results"` // Maximum results to return
	MinScore   float64 `json:"min_score"`   // Minimum relevance score

	// User context
	UserID    string                 `json:"user_id,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// RetrieveOptions defines options for document retrieval
type RetrieveOptions struct {
	// Search configuration
	TopK                int     `json:"top_k"`                // Number of documents to retrieve
	SimilarityThreshold float64 `json:"similarity_threshold"` // Minimum similarity score

	// Search methods
	EnableVectorSearch  bool `json:"enable_vector_search"`  // Enable vector similarity search
	EnableKeywordSearch bool `json:"enable_keyword_search"` // Enable keyword search
	EnableHybridSearch  bool `json:"enable_hybrid_search"`  // Enable hybrid search

	// Hybrid search configuration
	VectorWeight  float64 `json:"vector_weight"`  // Weight for vector search in hybrid
	KeywordWeight float64 `json:"keyword_weight"` // Weight for keyword search in hybrid

	// Filtering options
	FilterOptions FilterCriteria `json:"filter_options"`

	// Reranking options
	EnableRerank bool   `json:"enable_rerank"`          // Enable result reranking
	RerankTopK   int    `json:"rerank_top_k"`           // Number of results to rerank
	RerankModel  string `json:"rerank_model,omitempty"` // Reranking model to use

	// Performance options
	MaxQueryTime time.Duration `json:"max_query_time,omitempty"`
	EnableCache  bool          `json:"enable_cache"` // Enable query caching
}

// GenerateOptions defines options for response generation
type GenerateOptions struct {
	// Model configuration
	Model            string  `json:"model,omitempty"`
	Temperature      float64 `json:"temperature"`
	MaxTokens        int     `json:"max_tokens"`
	TopP             float64 `json:"top_p"`
	FrequencyPenalty float64 `json:"frequency_penalty"`
	PresencePenalty  float64 `json:"presence_penalty"`

	// Prompt configuration
	PromptTemplate   string `json:"prompt_template,omitempty"`
	SystemPrompt     string `json:"system_prompt,omitempty"`
	MaxContextLength int    `json:"max_context_length"` // Maximum context length in tokens

	// Generation behavior
	EnableCitations bool   `json:"enable_citations"` // Include source citations
	AnswerOnly      bool   `json:"answer_only"`      // Return only the answer
	IncludeSummary  bool   `json:"include_summary"`  // Include summary
	Format          string `json:"format"`           // Response format (markdown, json, etc.)

	// Quality options
	MinConfidence   float64 `json:"min_confidence"`    // Minimum confidence threshold
	EnableFactCheck bool    `json:"enable_fact_check"` // Enable fact checking

	// Performance options
	EnableStreaming   bool          `json:"enable_streaming"` // Enable streaming responses
	MaxGenerationTime time.Duration `json:"max_generation_time,omitempty"`
}

// CompletionOptions defines options for LLM completion
type CompletionOptions struct {
	Model            string        `json:"model,omitempty"`
	Temperature      float64       `json:"temperature"`
	MaxTokens        int           `json:"max_tokens"`
	TopP             float64       `json:"top_p"`
	FrequencyPenalty float64       `json:"frequency_penalty"`
	PresencePenalty  float64       `json:"presence_penalty"`
	Stream           bool          `json:"stream"`
	Stop             []string      `json:"stop,omitempty"`
	Timeout          time.Duration `json:"timeout,omitempty"`
}

// FilterCriteria defines criteria for filtering results
type FilterCriteria struct {
	// Document criteria
	DocumentIDs   []string `json:"document_ids,omitempty"`
	DataSourceIDs []string `json:"data_source_ids,omitempty"`
	FileTypes     []string `json:"file_types,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Categories    []string `json:"categories,omitempty"`
	Authors       []string `json:"authors,omitempty"`

	// Time-based filtering
	CreatedAfter   *time.Time `json:"created_after,omitempty"`
	CreatedBefore  *time.Time `json:"created_before,omitempty"`
	ModifiedAfter  *time.Time `json:"modified_after,omitempty"`
	ModifiedBefore *time.Time `json:"modified_before,omitempty"`

	// Content-based filtering
	MinLength    int    `json:"min_length,omitempty"`
	MaxLength    int    `json:"max_length,omitempty"`
	MinWordCount int    `json:"min_word_count,omitempty"`
	MaxWordCount int    `json:"max_word_count,omitempty"`
	Language     string `json:"language,omitempty"`

	// Score-based filtering
	MinScore float64 `json:"min_score,omitempty"`
	MaxScore float64 `json:"max_score,omitempty"`

	// Custom filtering
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// ListOptions defines options for listing documents
type ListOptions struct {
	Limit     int            `json:"limit,omitempty"`
	Offset    int            `json:"offset,omitempty"`
	SortBy    string         `json:"sort_by,omitempty"`    // created_at, modified_at, title, etc.
	SortOrder string         `json:"sort_order,omitempty"` // asc, desc
	Filter    FilterCriteria `json:"filter,omitempty"`
}

// TimeRange defines a time range for filtering
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// CompletionResponse represents a completion response from LLM
type CompletionResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []CompletionChoice `json:"choices"`
	Usage   CompletionUsage    `json:"usage"`
}

// CompletionChoice represents a choice in completion response
type CompletionChoice struct {
	Index        int             `json:"index"`
	Message      llm.ChatMessage `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

// CompletionUsage represents token usage in completion
type CompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ModelInfo represents information about an LLM model
type ModelInfo struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"` // chat, embedding, rerank
	Provider     string                 `json:"provider"`
	MaxTokens    int                    `json:"max_tokens"`
	CostPerToken float64                `json:"cost_per_token"`
	Capabilities []string               `json:"capabilities"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Statistics types

// SystemStats represents overall RAG system statistics
type SystemStats struct {
	// Document statistics
	TotalDocuments  int   `json:"total_documents"`
	TotalChunks     int   `json:"total_chunks"`
	TotalEmbeddings int   `json:"total_embeddings"`
	IndexedSize     int64 `json:"indexed_size"` // Size in bytes

	// Data source statistics
	DataSourcesCount int      `json:"data_sources_count"`
	ActiveSources    []string `json:"active_sources"`

	// Query statistics
	TotalQueries int64         `json:"total_queries"`
	CacheHitRate float64       `json:"cache_hit_rate"`
	AvgQueryTime time.Duration `json:"avg_query_time"`

	// Performance metrics
	Uptime      time.Duration `json:"uptime"`
	LastUpdated time.Time     `json:"last_updated"`

	// Resource usage
	MemoryUsage int64   `json:"memory_usage"` // in bytes
	CPUUsage    float64 `json:"cpu_usage"`    // percentage

	// Error statistics
	ErrorRate float64 `json:"error_rate"`
	LastError *string `json:"last_error,omitempty"`
}

// RetrieverStats represents retriever-specific statistics
type RetrieverStats struct {
	// Document counts
	TotalDocuments int `json:"total_documents"`
	TotalChunks    int `json:"total_chunks"`
	IndexedChunks  int `json:"indexed_chunks"`

	// Vector statistics
	EmbeddingDim    int   `json:"embedding_dim"`
	VectorIndexSize int64 `json:"vector_index_size"` // in bytes

	// Performance metrics
	AvgRetrievalTime time.Duration `json:"avg_retrieval_time"`
	QueriesProcessed int64         `json:"queries_processed"`

	// Index information
	IndexType   string        `json:"index_type"`
	BuildTime   time.Duration `json:"build_time"`
	LastIndexed time.Time     `json:"last_indexed"`
}

// StorageStats represents storage-specific statistics
type StorageStats struct {
	// Size information
	TotalSize     int64 `json:"total_size"`     // in bytes
	DocumentSize  int64 `json:"document_size"`  // in bytes
	ChunkSize     int64 `json:"chunk_size"`     // in bytes
	EmbeddingSize int64 `json:"embedding_size"` // in bytes
	IndexSize     int64 `json:"index_size"`     // in bytes

	// Count information
	DocumentCount  int `json:"document_count"`
	ChunkCount     int `json:"chunk_count"`
	EmbeddingCount int `json:"embedding_count"`
	IndexCount     int `json:"index_count"`

	// Performance metrics
	CompressionRatio float64       `json:"compression_ratio"`
	ReadLatency      time.Duration `json:"read_latency"`
	WriteLatency     time.Duration `json:"write_latency"`

	// Cache statistics
	CacheSize    int64   `json:"cache_size"` // in bytes
	CacheHitRate float64 `json:"cache_hit_rate"`
}

// Metrics represents collected RAG system metrics
type Metrics struct {
	// Query metrics
	QueryMetrics QueryMetrics `json:"query_metrics"`

	// Document processing metrics
	ProcessingMetrics ProcessingMetrics `json:"processing_metrics"`

	// Retrieval metrics
	RetrievalMetrics RetrievalMetrics `json:"retrieval_metrics"`

	// Generation metrics
	GenerationMetrics GenerationMetrics `json:"generation_metrics"`

	// System metrics
	SystemMetrics SystemMetrics `json:"system_metrics"`

	// Time range
	TimeRange   TimeRange `json:"time_range"`
	GeneratedAt time.Time `json:"generated_at"`
}

// QueryMetrics represents query-related metrics
type QueryMetrics struct {
	TotalQueries     int64         `json:"total_queries"`
	UniqueQueries    int64         `json:"unique_queries"`
	AvgQueryLength   float64       `json:"avg_query_length"`
	AvgQueryTime     time.Duration `json:"avg_query_time"`
	AvgRetrievedDocs float64       `json:"avg_retrieved_docs"`
	CacheHitRate     float64       `json:"cache_hit_rate"`
	SuccessRate      float64       `json:"success_rate"`
}

// ProcessingMetrics represents document processing metrics
type ProcessingMetrics struct {
	DocumentsProcessed  int64         `json:"documents_processed"`
	ChunksCreated       int64         `json:"chunks_created"`
	EmbeddingsGenerated int64         `json:"embeddings_generated"`
	AvgProcessingTime   time.Duration `json:"avg_processing_time"`
	ProcessingRate      float64       `json:"processing_rate"` // docs per second
	ErrorRate           float64       `json:"error_rate"`
}

// RetrievalMetrics represents retrieval-related metrics
type RetrievalMetrics struct {
	TotalRetrievals   int64         `json:"total_retrievals"`
	AvgRetrievalTime  time.Duration `json:"avg_retrieval_time"`
	AvgResultsCount   float64       `json:"avg_results_count"`
	AvgRelevanceScore float64       `json:"avg_relevance_score"`
	RerankRate        float64       `json:"rerank_rate"`
	FilterRate        float64       `json:"filter_rate"`
}

// GenerationMetrics represents generation-related metrics
type GenerationMetrics struct {
	TotalGenerations  int64         `json:"total_generations"`
	AvgGenerationTime time.Duration `json:"avg_generation_time"`
	AvgInputTokens    float64       `json:"avg_input_tokens"`
	AvgOutputTokens   float64       `json:"avg_output_tokens"`
	TotalCost         float64       `json:"total_cost"`
	AvgCost           float64       `json:"avg_cost"`
	QualityScore      float64       `json:"quality_score"`
}

// SystemMetrics represents system-level metrics
type SystemMetrics struct {
	MemoryUsage int64   `json:"memory_usage"` // in bytes
	CPUUsage    float64 `json:"cpu_usage"`    // percentage
	DiskUsage   int64   `json:"disk_usage"`   // in bytes
	NetworkIO   int64   `json:"network_io"`   // in bytes
	GoRoutines  int     `json:"goroutines"`
	HeapSize    int64   `json:"heap_size"`    // in bytes
	GCCount     uint32  `json:"gc_count"`
}

// PerformanceMetrics represents performance metrics
type PerformanceMetrics struct {
	Operation    string        `json:"operation"`
	Duration     time.Duration `json:"duration"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Success      bool          `json:"success"`
	ErrorMessage string        `json:"error_message,omitempty"`

	// Resource usage
	MemoryUsage int64   `json:"memory_usage"` // in bytes
	CPUUsage    float64 `json:"cpu_usage"`    // percentage

	// Operation-specific metrics
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CacheStats represents cache statistics
type CacheStats struct {
	// Size information
	TotalEntries int   `json:"total_entries"`
	TotalSize    int64 `json:"total_size"` // in bytes

	// Performance metrics
	HitRate     float64       `json:"hit_rate"`
	MissRate    float64       `json:"miss_rate"`
	AvgHitTime  time.Duration `json:"avg_hit_time"`
	AvgMissTime time.Duration `json:"avg_miss_time"`

	// Eviction statistics
	Evictions      int64   `json:"evictions"`
	ExpirationRate float64 `json:"expiration_rate"`

	// TTL statistics
	AvgTTL time.Duration `json:"avg_ttl"`
	MinTTL time.Duration `json:"min_ttl"`
	MaxTTL time.Duration `json:"max_ttl"`
}
