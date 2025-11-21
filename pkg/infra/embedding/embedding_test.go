package embedding

import (
	"os"
	"strings"
	"testing"
	"time"
)

// TestEmbedLocalMiniLM tests the basic embedding functionality
func TestEmbedLocalMiniLM(t *testing.T) {
	texts := []string{
		"Hello, world!",
		"This is a test.",
		"Embeddings are useful.",
	}

	embeddings := EmbedLocalMiniLM(texts)

	if len(embeddings) != len(texts) {
		t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
	}

	for i, embedding := range embeddings {
		if len(embedding) != 384 {
			t.Errorf("Expected embedding dimension 384, got %d for text %d", len(embedding), i)
		}

		// Check that embedding is normalized (approximately)
		var sum float64
		for _, val := range embedding {
			sum += val * val
		}
		if sum < 0.9 || sum > 1.1 { // Allow some tolerance for hash-based embeddings
			t.Errorf("Embedding %d may not be properly normalized, sum of squares: %f", i, sum)
		}
	}
}

// TestGetEmbeddingDimension tests the embedding dimension function
func TestGetEmbeddingDimension(t *testing.T) {
	dimension := GetEmbeddingDimension()
	expected := 384 // all-MiniLM-L6-v2 dimension

	if dimension != expected {
		t.Errorf("Expected dimension %d, got %d", expected, dimension)
	}
}

// TestEmbedWithLocalModel tests the enhanced local embedding functionality
func TestEmbedWithLocalModel(t *testing.T) {
	// Save original env var
	origUseEnhanced := os.Getenv("EMBEDDING_USE_ENHANCED_LOCAL")
	defer os.Setenv("EMBEDDING_USE_ENHANCED_LOCAL", origUseEnhanced)

	// Test with hash-based embedding (default)
	os.Setenv("EMBEDDING_USE_ENHANCED_LOCAL", "false")

	texts := []string{"Test text for embedding"}
	embeddings, err := EmbedWithLocalModel(texts)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
	}

	if len(embeddings[0]) != GetEmbeddingDimension() {
		t.Errorf("Expected embedding dimension %d, got %d", GetEmbeddingDimension(), len(embeddings[0]))
	}

	// Test with enhanced local embedding (may fail if transformers not available)
	os.Setenv("EMBEDDING_USE_ENHANCED_LOCAL", "true")
	embeddings, err = EmbedWithLocalModel(texts)

	// This may fail if Python transformers are not available, which is okay
	if err != nil {
		t.Logf("Enhanced local embedding failed (expected if transformers not available): %v", err)
	}
}

// TestIsLocalEmbeddingAvailable tests the availability check
func TestIsLocalEmbeddingAvailable(t *testing.T) {
	// Save original env var
	origUseEnhanced := os.Getenv("EMBEDDING_USE_ENHANCED_LOCAL")
	defer os.Setenv("EMBEDDING_USE_ENHANCED_LOCAL", origUseEnhanced)

	// Test with hash-based embedding (should always be available)
	os.Setenv("EMBEDDING_USE_ENHANCED_LOCAL", "false")
	available := IsLocalEmbeddingAvailable()
	if !available {
		t.Error("Expected local embedding to be available with hash-based method")
	}

	// Test with enhanced local embedding
	os.Setenv("EMBEDDING_USE_ENHANCED_LOCAL", "true")
	available = IsLocalEmbeddingAvailable()
	// This may be false if transformers are not available, which is okay
	t.Logf("Enhanced local embedding availability: %v", available)
}

// TestEmbedLocalWithFallback tests the fallback functionality
func TestEmbedLocalWithFallback(t *testing.T) {
	texts := []string{"Test fallback functionality"}

	embeddings, err := EmbedLocalWithFallback(texts)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
	}

	if len(embeddings[0]) != GetEmbeddingDimension() {
		t.Errorf("Expected embedding dimension %d, got %d", GetEmbeddingDimension(), len(embeddings[0]))
	}
}

// TestGetEmbeddingStats tests the embedding statistics
func TestGetEmbeddingStats(t *testing.T) {
	// Save original env var
	origUseEnhanced := os.Getenv("EMBEDDING_USE_ENHANCED_LOCAL")
	defer os.Setenv("EMBEDDING_USE_ENHANCED_LOCAL", origUseEnhanced)

	// Test with hash-based embedding
	os.Setenv("EMBEDDING_USE_ENHANCED_LOCAL", "false")
	stats := GetEmbeddingStats()

	if stats == nil {
		t.Error("Expected stats to not be nil")
	}

	if stats.ModelType != "hash" {
		t.Errorf("Expected model type 'hash', got '%s'", stats.ModelType)
	}

	if stats.Dimension != GetEmbeddingDimension() {
		t.Errorf("Expected dimension %d, got %d", GetEmbeddingDimension(), stats.Dimension)
	}

	if !stats.Available {
		t.Error("Expected embedding to be available")
	}

	// Test with enhanced local embedding
	os.Setenv("EMBEDDING_USE_ENHANCED_LOCAL", "true")
	stats = GetEmbeddingStats()
	t.Logf("Stats with enhanced embedding: %+v", stats)
}

// TestHashEmbed tests the hash embedding function directly
func TestHashEmbed(t *testing.T) {
	text := "Hello, world!"
	dimension := 384

	embedding := hashEmbed(text, dimension)

	if len(embedding) != dimension {
		t.Errorf("Expected dimension %d, got %d", dimension, len(embedding))
	}

	// Test that same text produces same embedding
	embedding2 := hashEmbedLocal(text, dimension)
	for i := 0; i < dimension; i++ {
		if embedding[i] != embedding2[i] {
			t.Errorf("Hash embeddings should be deterministic, difference at index %d", i)
			break
		}
	}

	// Test that different texts produce different embeddings
	differentText := "Different text"
	embedding3 := hashEmbedLocal(differentText, dimension)
	same := true
	for i := 0; i < dimension; i++ {
		if embedding[i] != embedding3[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("Different texts should produce different embeddings")
	}
}

// TestNewLocalEmbedder tests the local embedder creation
func TestNewLocalEmbedder(t *testing.T) {
	// Test with default config
	embedder, err := NewLocalEmbedder(nil)
	if err != nil {
		t.Errorf("Unexpected error creating embedder: %v", err)
	}
	if embedder == nil {
		t.Error("Expected embedder to not be nil")
	}

	// Test with custom config
	config := &Config{
		LocalModelType: "python",
		Model:          "test-model",
		BatchSize:      16,
		MaxConcurrency: 2,
		EnableFallback: true,
		Timeout:        30 * time.Second,
	}

	embedder, err = NewLocalEmbedder(config)
	if err != nil {
		t.Errorf("Unexpected error creating embedder with config: %v", err)
	}
	if embedder == nil {
		t.Error("Expected embedder to not be nil")
	}
}

// TestLocalEmbedderInterface tests that LocalEmbedder implements the Embedder interface
func TestLocalEmbedderInterface(t *testing.T) {
	config := &Config{
		LocalModelType: "python",
		EnableFallback: true,
	}

	embedder, err := NewLocalEmbedder(config)
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}

	// Test that it implements Embedder interface
	var _ Embedder = embedder

	// Test GetDimension
	dimension := embedder.GetDimension()
	if dimension != 384 {
		t.Errorf("Expected dimension 384, got %d", dimension)
	}

	// Test Close
	err = embedder.Close()
	if err != nil {
		t.Errorf("Unexpected error closing embedder: %v", err)
	}
}

// TestEmptyInputs tests edge cases with empty inputs
func TestEmptyInputs(t *testing.T) {
	// Test with empty slice
	embeddings := EmbedLocalMiniLM([]string{})
	if len(embeddings) != 0 {
		t.Errorf("Expected empty result for empty input, got %d embeddings", len(embeddings))
	}

	// Test enhanced function with empty input
	embeddings, err := EmbedWithLocalModel([]string{})
	if err != nil {
		t.Errorf("Unexpected error with empty input: %v", err)
	}
	if len(embeddings) != 0 {
		t.Errorf("Expected empty result for empty input, got %d embeddings", len(embeddings))
	}
}

// TestLargeInput tests with larger inputs
func TestLargeInput(t *testing.T) {
	// Generate a large text
	largeText := "This is a test. " + strings.Repeat("Large text test. ", 100)

	texts := []string{largeText}

	start := time.Now()
	embeddings := EmbedLocalMiniLM(texts)
	duration := time.Since(start)

	if len(embeddings) != 1 {
		t.Errorf("Expected 1 embedding, got %d", len(embeddings))
	}

	if len(embeddings[0]) != GetEmbeddingDimension() {
		t.Errorf("Expected embedding dimension %d, got %d", GetEmbeddingDimension(), len(embeddings[0]))
	}

	t.Logf("Embedding generation took %v for %d characters", duration, len(largeText))
}

// TestConsistency tests embedding consistency
func TestConsistency(t *testing.T) {
	text := "Consistency test text"
	texts := []string{text}

	// Generate embeddings multiple times
	embeddings1 := EmbedLocalMiniLM(texts)
	embeddings2 := EmbedLocalMiniLM(texts)

	// They should be identical for hash-based embeddings
	if len(embeddings1) != len(embeddings2) {
		t.Errorf("Expected same number of embeddings, got %d and %d", len(embeddings1), len(embeddings2))
	}

	for i := range embeddings1 {
		if len(embeddings1[i]) != len(embeddings2[i]) {
			t.Errorf("Embedding %d has different dimensions", i)
			continue
		}

		for j := range embeddings1[i] {
			if embeddings1[i][j] != embeddings2[i][j] {
				t.Errorf("Embedding %d differs at index %d", i, j)
				break
			}
		}
	}
}

// Benchmark tests
func BenchmarkEmbedLocalMiniLM(b *testing.B) {
	texts := []string{
		"Hello, world!",
		"This is a test.",
		"Embeddings are useful.",
		"Benchmark test text.",
		"Performance testing.",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EmbedLocalMiniLM(texts)
	}
}

func BenchmarkHashEmbed(b *testing.B) {
	text := "Benchmark hash embedding test"
	dimension := 384

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hashEmbedLocal(text, dimension)
	}
}

func BenchmarkGetEmbeddingDimension(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetEmbeddingDimension()
	}
}

func BenchmarkGetEmbeddingStats(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetEmbeddingStats()
	}
}