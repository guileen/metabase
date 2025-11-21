package core

import (
	"context"
	"time"

	"github.com/guileen/metabase/pkg/biz/rag/embedding"
	"github.com/guileen/metabase/pkg/biz/rag/llm"
)

// RAGSystem defines the main interface for the RAG (Retrieval-Augmented Generation) system
type RAGSystem interface {
	// Index processes and indexes documents from data sources
	Index(ctx context.Context, options IndexOptions) (*IndexResult, error)

	// Query performs a RAG query
	Query(ctx context.Context, query string, options QueryOptions) (*QueryResult, error)

	// AddDataSource adds a new data source to the system
	AddDataSource(source DataSource) error

	// RemoveDataSource removes a data source from the system
	RemoveDataSource(sourceID string) error

	// ListDataSources returns all registered data sources
	ListDataSources() []DataSource

	// GetStats returns system statistics and performance metrics
	GetStats() (*SystemStats, error)

	// Close performs cleanup and releases resources
	Close() error
}

// DataSource represents a source of documents for the RAG system
type DataSource interface {
	// GetID returns a unique identifier for this data source
	GetID() string

	// GetType returns the type of data source (filesystem, database, web, etc.)
	GetType() string

	// GetConfig returns the data source configuration
	GetConfig() interface{}

	// ListDocuments returns a list of all available documents
	ListDocuments(ctx context.Context) ([]Document, error)

	// GetDocument retrieves a specific document by ID
	GetDocument(ctx context.Context, documentID string) (*Document, error)

	// Sync synchronizes the data source and returns changed documents
	Sync(ctx context.Context, since time.Time) (*SyncResult, error)

	// Validate checks if the data source is properly configured and accessible
	Validate() error

	// Close performs cleanup for this data source
	Close() error
}

// DocumentProcessor handles document processing, chunking, and embedding
type DocumentProcessor interface {
	// ProcessDocument processes a document and returns chunks
	ProcessDocument(ctx context.Context, doc Document) ([]DocumentChunk, error)

	// SetChunkingStrategy sets the chunking strategy
	SetChunkingStrategy(strategy ChunkingStrategy)

	// GetChunkingStrategy returns the current chunking strategy
	GetChunkingStrategy() ChunkingStrategy

	// SetEmbeddingGenerator sets the embedding generator
	SetEmbeddingGenerator(generator embedding.VectorGenerator)

	// GetEmbeddingGenerator returns the current embedding generator
	GetEmbeddingGenerator() embedding.VectorGenerator
}

// Retriever handles document retrieval and search
type Retriever interface {
	// Retrieve retrieves relevant documents for a query
	Retrieve(ctx context.Context, query string, options RetrieveOptions) ([]RetrievalResult, error)

	// AddDocument adds a document chunk to the retriever index
	AddDocument(ctx context.Context, chunk DocumentChunk) error

	// RemoveDocument removes a document chunk from the retriever index
	RemoveDocument(ctx context.Context, chunkID string) error

	// UpdateDocument updates an existing document chunk
	UpdateDocument(ctx context.Context, chunk DocumentChunk) error

	// Clear clears all documents from the retriever
	Clear(ctx context.Context) error

	// GetStats returns retriever statistics
	GetStats() (*RetrieverStats, error)
}

// Generator handles response generation using retrieved context
type Generator interface {
	// Generate generates a response using the query and retrieved context
	Generate(ctx context.Context, query string, context []RetrievalResult, options GenerateOptions) (*GenerationResult, error)

	// SetPromptTemplate sets the prompt template for generation
	SetPromptTemplate(template PromptTemplate)

	// GetPromptTemplate returns the current prompt template
	GetPromptTemplate() PromptTemplate

	// SetLLMClient sets the LLM client for generation
	SetLLMClient(client LLMClient)

	// GetLLMClient returns the current LLM client
	GetLLMClient() LLMClient
}

// Storage handles persistence of RAG data
type Storage interface {
	// StoreDocument stores a document and its metadata
	StoreDocument(ctx context.Context, doc Document) error

	// GetDocument retrieves a document by ID
	GetDocument(ctx context.Context, documentID string) (*Document, error)

	// StoreChunk stores a document chunk
	StoreChunk(ctx context.Context, chunk DocumentChunk) error

	// GetChunk retrieves a document chunk by ID
	GetChunk(ctx context.Context, chunkID string) (*DocumentChunk, error)

	// StoreEmbedding stores an embedding for a chunk
	StoreEmbedding(ctx context.Context, chunkID string, embedding []float64) error

	// GetEmbedding retrieves an embedding for a chunk
	GetEmbedding(ctx context.Context, chunkID string) ([]float64, error)

	// SearchEmbeddings performs similarity search on embeddings
	SearchEmbeddings(ctx context.Context, queryEmbedding []float64, limit int) ([]EmbeddingMatch, error)

	// StoreQuery stores a query and its results for caching/analysis
	StoreQuery(ctx context.Context, query QueryRecord) error

	// GetQuery retrieves a stored query
	GetQuery(ctx context.Context, queryID string) (*QueryRecord, error)

	// ListDocuments returns all documents
	ListDocuments(ctx context.Context, options ListOptions) ([]Document, error)

	// ListChunks returns all chunks for a document
	ListChunks(ctx context.Context, documentID string) ([]DocumentChunk, error)

	// DeleteDocument deletes a document and all its chunks
	DeleteDocument(ctx context.Context, documentID string) error

	// Clear clears all stored data
	Clear(ctx context.Context) error

	// GetStorageStats returns storage statistics
	GetStorageStats() (*StorageStats, error)

	// Close performs cleanup
	Close() error
}

// LLMClient defines the interface for Language Model clients
type LLMClient interface {
	// GenerateCompletion generates a text completion
	GenerateCompletion(ctx context.Context, messages []llm.ChatMessage, options CompletionOptions) (*CompletionResponse, error)

	// GenerateEmbedding generates embeddings for text
	GenerateEmbedding(ctx context.Context, texts []string) ([][]float64, error)

	// Rerank reranks documents based on relevance to query
	Rerank(ctx context.Context, query string, documents []string) ([]float64, error)

	// GetModelInfo returns information about the available models
	GetModelInfo() (*ModelInfo, error)

	// Validate checks if the LLM client is properly configured
	Validate() error

	// Close performs cleanup
	Close() error
}

// ChunkingStrategy defines how documents are split into chunks
type ChunkingStrategy interface {
	// Chunk splits a document into chunks
	Chunk(ctx context.Context, doc Document) ([]DocumentChunk, error)

	// GetName returns the name of the chunking strategy
	GetName() string

	// GetDescription returns a description of the chunking strategy
	GetDescription() string

	// SetParameters sets strategy-specific parameters
	SetParameters(params map[string]interface{}) error

	// GetParameters returns the current parameters
	GetParameters() map[string]interface{}
}

// PromptTemplate defines the interface for prompt templates
type PromptTemplate interface {
	// Format formats a prompt using the query and context
	Format(query string, context []RetrievalResult, options map[string]interface{}) (string, error)

	// GetName returns the template name
	GetName() string

	// GetDescription returns the template description
	GetDescription() string

	// Validate checks if the template is valid
	Validate() error
}

// EventListener defines the interface for RAG system event listeners
type EventListener interface {
	// OnDocumentIndexed called when a document is indexed
	OnDocumentIndexed(ctx context.Context, doc Document, chunks []DocumentChunk)

	// OnQueryExecuted called when a query is executed
	OnQueryExecuted(ctx context.Context, query string, result *QueryResult)

	// OnError called when an error occurs
	OnError(ctx context.Context, operation string, err error)

	// OnPerformanceMetrics called when performance metrics are available
	OnPerformanceMetrics(ctx context.Context, metrics *PerformanceMetrics)
}

// MetricsCollector defines the interface for collecting RAG system metrics
type MetricsCollector interface {
	// RecordQuery records a query execution
	RecordQuery(ctx context.Context, queryID string, query string, duration time.Duration, result *QueryResult)

	// RecordDocumentProcessing records document processing metrics
	RecordDocumentProcessing(ctx context.Context, documentID string, duration time.Duration, chunkCount int)

	// RecordRetrieval records retrieval metrics
	RecordRetrieval(ctx context.Context, queryID string, duration time.Duration, retrievedCount, returnedCount int)

	// RecordGeneration records generation metrics
	RecordGeneration(ctx context.Context, queryID string, duration time.Duration, inputTokens, outputTokens int)

	// GetMetrics returns collected metrics
	GetMetrics(ctx context.Context, timeRange TimeRange) (*Metrics, error)

	// Reset resets collected metrics
	Reset(context.Context) error
}

// Cache defines the interface for caching RAG results
type Cache interface {
	// Get retrieves a cached query result
	Get(ctx context.Context, key string) (*QueryResult, error)

	// Set stores a query result in cache
	Set(ctx context.Context, key string, result *QueryResult, ttl time.Duration) error

	// Delete removes a cached result
	Delete(ctx context.Context, key string) error

	// Clear clears all cached results
	Clear(ctx context.Context) error

	// GetStats returns cache statistics
	GetStats() (*CacheStats, error)

	// Close performs cleanup
	Close() error
}

// Filter defines the interface for filtering retrieval results
type Filter interface {
	// Filter filters retrieval results based on criteria
	Filter(ctx context.Context, results []RetrievalResult, criteria FilterCriteria) ([]RetrievalResult, error)

	// GetName returns the filter name
	GetName() string

	// GetDescription returns the filter description
	GetDescription() string

	// Validate checks if the filter is valid
	Validate() error
}

// Ranker defines the interface for ranking retrieval results
type Ranker interface {
	// Rank ranks retrieval results based on relevance
	Rank(ctx context.Context, query string, results []RetrievalResult) ([]RetrievalResult, error)

	// GetName returns the ranker name
	GetName() string

	// GetDescription returns the ranker description
	GetDescription() string

	// Validate checks if the ranker is valid
	Validate() error
}
