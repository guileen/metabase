package embedding

import (
	"context"
	"time"
)

// VectorGenerator defines the interface for all vector embedding implementations
// This interface provides a unified way to work with different embedding models
type VectorGenerator interface {
	// Embed generates embeddings for a batch of texts
	Embed(ctx context.Context, texts []string) ([][]float64, error)

	// EmbedSingle generates embedding for a single text
	EmbedSingle(ctx context.Context, text string) ([]float64, error)

	// GetDimension returns the dimension of the embedding vectors
	GetDimension() int

	// GetModelName returns the name/type of the model
	GetModelName() string

	// GetCapabilities returns the capabilities of this embedding model
	GetCapabilities() ModelCapabilities

	// Close performs cleanup and releases resources
	Close() error
}

// ModelCapabilities describes what a model can do
type ModelCapabilities struct {
	// Languages supported by the model
	Languages []string

	// Maximum sequence length
	MaxSequenceLength int

	// Recommended batch size for optimal performance
	RecommendedBatchSize int

	// Whether the model supports multilingual text
	SupportsMultilingual bool

	// Whether the model is optimized for Chinese text
	OptimizedForChinese bool

	// Whether the model supports GPU acceleration
	SupportsGPU bool

	// Model size in bytes (for resource planning)
	ModelSizeBytes int64

	// Memory usage during inference in bytes
	EstimatedMemoryUsage int64
}

// VectorGeneratorConfig holds configuration for vector generators
type VectorGeneratorConfig struct {
	// Model type/name
	ModelName string

	// Batch size for processing
	BatchSize int

	// Maximum concurrent requests
	MaxConcurrency int

	// Request timeout
	Timeout time.Duration

	// Whether to enable fallback mechanisms
	EnableFallback bool

	// Cache directory for models
	CacheDir string

	// Model-specific configuration
	ModelConfig map[string]interface{}
}

// Performance metrics for comparing different implementations
type PerformanceMetrics struct {
	// Model information
	ModelName string
	Dimension int

	// Performance metrics
	LatencyMs     float64 `json:"latency_ms"`      // Average latency per request
	ThroughputQPS float64 `json:"throughput_qps"`  // Queries per second
	MemoryUsageMB float64 `json:"memory_usage_mb"` // Memory usage in MB

	// Quality metrics (if available)
	QualityScore float64 `json:"quality_score"` // Normalized quality score (0-1)

	// Resource usage
	CPUUsagePercent float64 `json:"cpu_usage_percent"`  // CPU usage during inference
	ModelLoadTimeMs float64 `json:"model_load_time_ms"` // Time to load model

	// Test information
	TestTextCount int    `json:"test_text_count"` // Number of texts used for testing
	TestDate      string `json:"test_date"`       // When the test was performed
}

// Registry for available vector generators
type VectorGeneratorRegistry interface {
	// Register a new vector generator implementation
	Register(name string, factory func(config VectorGeneratorConfig) (VectorGenerator, error)) error

	// Get a vector generator by name
	Get(name string, config VectorGeneratorConfig) (VectorGenerator, error)

	// List all available generators
	List() []string

	// Get capabilities for a generator without instantiating it
	GetCapabilities(name string) (ModelCapabilities, error)
}

// FallbackGenerator defines a generator that can fall back to alternatives
type FallbackGenerator interface {
	VectorGenerator

	// SetPrimary sets the primary generator
	SetPrimary(primary VectorGenerator) error

	// AddFallback adds a fallback generator
	AddFallback(fallback VectorGenerator) error

	// GetFallbackChain returns the current fallback chain
	GetFallbackChain() []VectorGenerator
}

// BatchProcessor defines an interface for optimized batch processing
type BatchProcessor interface {
	// ProcessBatch processes a batch with optimal performance
	ProcessBatch(ctx context.Context, texts []string, opts BatchOptions) ([][]float64, error)

	// GetOptimalBatchSize returns the recommended batch size
	GetOptimalBatchSize() int
}

// BatchOptions provides options for batch processing
type BatchOptions struct {
	// Override the default batch size
	BatchSize int

	// Whether to use parallel processing
	UseParallel bool

	// Maximum number of parallel workers
	MaxWorkers int

	// Progress callback for long operations
	ProgressCallback func(processed, total int)
}

// CachingGenerator defines a generator that supports caching
type CachingGenerator interface {
	VectorGenerator

	// EnableCache enables or disables caching
	EnableCache(enabled bool) error

	// ClearCache clears all cached embeddings
	ClearCache() error

	// GetCacheStats returns cache statistics
	GetCacheStats() CacheStats
}

// CacheStats provides cache performance information
type CacheStats struct {
	// Cache hit rate (0-1)
	HitRate float64

	// Total number of entries
	TotalEntries int

	// Cache size in bytes
	SizeBytes int64

	// Maximum cache size
	MaxSizeBytes int64
}

// StreamingGenerator defines a generator that supports streaming for large texts
type StreamingGenerator interface {
	VectorGenerator

	// EmbedStream processes large texts by streaming
	EmbedStream(ctx context.Context, texts <-chan string) (<-chan []float64, <-chan error)

	// ChunkText splits text into optimal chunks for processing
	ChunkText(text string, maxChunkSize int) []string
}

// Comparator for performance testing different generators
type GeneratorComparator interface {
	// Compare benchmarks multiple generators
	Compare(ctx context.Context, generatorNames []string, testTexts []string) ([]PerformanceMetrics, error)

	// CompareWithDataset compares generators using a dataset
	CompareWithDataset(ctx context.Context, generatorNames []string, datasetPath string) ([]PerformanceMetrics, error)
}

// Utility functions for validation
func ValidateEmbeddings(embeddings [][]float64, expectedDim int) error {
	if len(embeddings) == 0 {
		return nil
	}

	for i, emb := range embeddings {
		if len(emb) != expectedDim {
			return &EmbeddingError{
				Type:     DimensionMismatch,
				Message:  "embedding dimension mismatch",
				Index:    i,
				Expected: expectedDim,
				Actual:   len(emb),
			}
		}

		// Check for NaN or infinite values
		for j, val := range emb {
			if isNaN(val) || isInf(val) {
				return &EmbeddingError{
					Type:    InvalidValue,
					Message: "embedding contains NaN or infinite values",
					Index:   i,
					Index2:  j,
					Value:   val,
				}
			}
		}
	}

	return nil
}

// Helper functions
func isNaN(f float64) bool {
	return f != f
}

func isInf(f float64) bool {
	return f == 0 || f != 0 && f == f*2
}

// EmbeddingError represents errors in embedding generation
type EmbeddingError struct {
	Type     ErrorType
	Message  string
	Index    int
	Index2   int
	Expected int
	Actual   int
	Value    float64
}

func (e *EmbeddingError) Error() string {
	return e.Message
}

type ErrorType int

const (
	UnknownError ErrorType = iota
	DimensionMismatch
	InvalidValue
	ModelNotLoaded
	TimeoutError
	ContextError
)
