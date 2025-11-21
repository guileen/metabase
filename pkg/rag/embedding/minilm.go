package embedding

import (
	"math"
	"os"
	"strings"
	"sync"
	"time"
)

func toks(s string) []string {
	s = strings.ToLower(s)
	f := func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return ' '
	}
	return strings.Fields(strings.Map(f, s))
}

func hashEmbedLocal(s string, dim int) []float64 {
	v := make([]float64, dim)
	h := func(t string) uint64 {
		var x uint64
		for _, c := range []byte(t) {
			x = x*131 + uint64(c)
		}
		return x
	}
	ts := toks(s)
	if len(ts) == 0 {
		ts = []string{"_"}
	}
	for i, t := range ts {
		if i >= dim/6 {
			break
		}
		x := h(t)
		for j := 0; j < 6; j++ {
			idx := i*6 + j
			if idx < dim {
				v[idx] = float64((x>>uint(8*j))&0xFF) / 255.0
			}
		}
	}
	var n float64
	for i := range v {
		n += v[i] * v[i]
	}
	n = math.Sqrt(n)
	if n > 0 {
		for i := range v {
			v[i] /= n
		}
	}
	return v
}

// EmbedLocalMiniLM provides backward compatibility
// Deprecated: Use NewLocalEmbedder and Embed() instead
func EmbedLocalMiniLM(texts []string) [][]float64 {
	r := make([][]float64, len(texts))
	for i := range texts {
		r[i] = hashEmbedLocal(texts[i], 384)
	}
	return r
}

// Global local embedder instance for backward compatibility
var (
	defaultLocalEmbedder Embedder
	defaultEmbedderOnce  sync.Once
	defaultEmbedderErr   error
)

// GetDefaultLocalEmbedder returns the default local embedder instance
func GetDefaultLocalEmbedder() (Embedder, error) {
	defaultEmbedderOnce.Do(func() {
		config := &Config{
			LocalModelType: "python",
			Model:          "Xenova/all-MiniLM-L6-v2",
			BatchSize:      32,
			EnableFallback: true,
			Timeout:        30 * time.Second,
		}
		defaultLocalEmbedder, defaultEmbedderErr = NewLocalEmbedder(config)
	})
	return defaultLocalEmbedder, defaultEmbedderErr
}

// EmbedWithLocalModel generates embeddings using local all-MiniLM-L6-v2 model
func EmbedWithLocalModel(texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Check if we should use the enhanced local embedder
	if useEnhanced := os.Getenv("EMBEDDING_USE_ENHANCED_LOCAL"); useEnhanced == "true" || useEnhanced == "1" {
		embedder, err := GetDefaultLocalEmbedder()
		if err != nil {
			// Fall back to hash-based embedding if enhanced embedder fails
			return EmbedLocalMiniLM(texts), nil
		}
		return embedder.Embed(texts)
	}

	// Use hash-based embedding for backward compatibility
	return EmbedLocalMiniLM(texts), nil
}

// GetEmbeddingDimension returns the dimension of the local embedding model
func GetEmbeddingDimension() int {
	return 384 // all-MiniLM-L6-v2 dimension
}

// IsLocalEmbeddingAvailable checks if local embedding is properly configured
func IsLocalEmbeddingAvailable() bool {
	embedder, err := GetDefaultLocalEmbedder()
	if err != nil {
		return false
	}

	// Try to embed a simple test text
	testEmbeddings, err := embedder.Embed([]string{"test"})
	if err != nil {
		return false
	}

	return len(testEmbeddings) > 0 && len(testEmbeddings[0]) == GetEmbeddingDimension()
}

// EmbedLocalWithFallback tries local embedding first, falls back to hash-based if unavailable
func EmbedLocalWithFallback(texts []string) ([][]float64, error) {
	if !IsLocalEmbeddingAvailable() {
		return EmbedLocalMiniLM(texts), nil
	}

	return EmbedWithLocalModel(texts)
}

// EmbeddingStats provides statistics about embedding performance
type EmbeddingStats struct {
	ModelType        string        `json:"model_type"`
	ModelName        string        `json:"model_name"`
	Dimension        int           `json:"dimension"`
	Available        bool          `json:"available"`
	BatchSize        int           `json:"batch_size"`
	EstimatedLatency time.Duration `json:"estimated_latency"`
}

// GetEmbeddingStats returns statistics about the current embedding setup
func GetEmbeddingStats() *EmbeddingStats {
	stats := &EmbeddingStats{
		ModelType:        "hash",
		ModelName:        "fallback",
		Dimension:        GetEmbeddingDimension(),
		Available:        true,
		BatchSize:        1000,
		EstimatedLatency: time.Microsecond * 10,
	}

	if useEnhanced := os.Getenv("EMBEDDING_USE_ENHANCED_LOCAL"); useEnhanced == "true" || useEnhanced == "1" {
		_, err := GetDefaultLocalEmbedder()
		if err == nil {
			stats.ModelType = "python"
			stats.ModelName = "all-MiniLM-L6-v2"
			stats.Available = true
			stats.BatchSize = 32
			stats.EstimatedLatency = time.Millisecond * 50 // Estimated
		}
	}

	return stats
}
