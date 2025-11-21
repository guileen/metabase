package vector

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/guileen/metabase/pkg/common/errors"
)

// Vector represents a high-dimensional vector for similarity search
type Vector []float32

// VectorDocument represents a document with its vector representation
type VectorDocument struct {
	ID       string                 `json:"id"`
	Vector   Vector                 `json:"vector"`
	Metadata map[string]interface{} `json:"metadata"`
}

// VectorSearchQuery represents a vector similarity search query
type VectorSearchQuery struct {
	QueryVector Vector  `json:"query_vector"`
	TopK        int     `json:"top_k"`
	Threshold   float32 `json:"threshold,omitempty"`
}

// VectorSearchResult represents a vector search result
type VectorSearchResult struct {
	DocumentID string                 `json:"document_id"`
	Score      float32                `json:"score"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// VectorIndex represents a vector index interface
type VectorIndex interface {
	// Add adds a vector document to the index
	Add(ctx context.Context, doc *VectorDocument) error

	// Update updates a vector document in the index
	Update(ctx context.Context, doc *VectorDocument) error

	// Remove removes a vector document from the index
	Remove(ctx context.Context, id string) error

	// Search performs similarity search
	Search(ctx context.Context, query *VectorSearchQuery) ([]*VectorSearchResult, error)

	// Get retrieves a vector document by ID
	Get(ctx context.Context, id string) (*VectorDocument, error)

	// Clear clears the entire index
	Clear(ctx context.Context) error

	// Stats returns index statistics
	Stats() map[string]interface{}
}

// MemoryVectorIndex is a simple in-memory vector index implementation
type MemoryVectorIndex struct {
	documents map[string]*VectorDocument
	vectors   map[string]Vector
	mutex     sync.RWMutex
	dimension int
}

// NewMemoryVectorIndex creates a new in-memory vector index
func NewMemoryVectorIndex(dimension int) *MemoryVectorIndex {
	return &MemoryVectorIndex{
		documents: make(map[string]*VectorDocument),
		vectors:   make(map[string]Vector),
		dimension: dimension,
	}
}

// Add adds a vector document to the memory index
func (m *MemoryVectorIndex) Add(ctx context.Context, doc *VectorDocument) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if doc.ID == "" {
		return errors.InvalidInput("document ID is required")
	}

	if len(doc.Vector) != m.dimension {
		return errors.InvalidInput(fmt.Sprintf("vector dimension mismatch: expected %d, got %d", m.dimension, len(doc.Vector)))
	}

	m.documents[doc.ID] = doc
	m.vectors[doc.ID] = make(Vector, len(doc.Vector))
	copy(m.vectors[doc.ID], doc.Vector)

	return nil
}

// Update updates a vector document in the memory index
func (m *MemoryVectorIndex) Update(ctx context.Context, doc *VectorDocument) error {
	return m.Add(ctx, doc) // Add handles both add and update
}

// Remove removes a vector document from the memory index
func (m *MemoryVectorIndex) Remove(ctx context.Context, id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, exists := m.documents[id]
	if !exists {
		return errors.NotFound("vector document")
	}

	delete(m.documents, id)
	delete(m.vectors, id)

	return nil
}

// Search performs similarity search using cosine similarity
func (m *MemoryVectorIndex) Search(ctx context.Context, query *VectorSearchQuery) ([]*VectorSearchResult, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if query == nil {
		return nil, errors.InvalidInput("search query is required")
	}

	if len(query.QueryVector) != m.dimension {
		return nil, errors.InvalidInput(fmt.Sprintf("query vector dimension mismatch: expected %d, got %d", m.dimension, len(query.QueryVector)))
	}

	topK := query.TopK
	if topK <= 0 {
		topK = 10 // default
	}

	var results []*VectorSearchResult

	for id, vector := range m.vectors {
		similarity := cosineSimilarity(query.QueryVector, vector)

		// Apply threshold if specified
		if query.Threshold > 0 && similarity < query.Threshold {
			continue
		}

		result := &VectorSearchResult{
			DocumentID: id,
			Score:      similarity,
			Metadata:   m.documents[id].Metadata,
		}
		results = append(results, result)
	}

	// Sort by similarity score (descending)
	results = sortResults(results)

	// Return topK results
	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// Get retrieves a vector document by ID
func (m *MemoryVectorIndex) Get(ctx context.Context, id string) (*VectorDocument, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	doc, exists := m.documents[id]
	if !exists {
		return nil, errors.NotFound("vector document")
	}

	// Return a copy to avoid modification
	docCopy := &VectorDocument{
		ID:       doc.ID,
		Vector:   make(Vector, len(doc.Vector)),
		Metadata: make(map[string]interface{}),
	}
	copy(docCopy.Vector, doc.Vector)

	for k, v := range doc.Metadata {
		docCopy.Metadata[k] = v
	}

	return docCopy, nil
}

// Clear clears the entire index
func (m *MemoryVectorIndex) Clear(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.documents = make(map[string]*VectorDocument)
	m.vectors = make(map[string]Vector)
	return nil
}

// Stats returns index statistics
func (m *MemoryVectorIndex) Stats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return map[string]interface{}{
		"documents_count":    len(m.documents),
		"vector_dimension":   m.dimension,
		"memory_usage_bytes": len(m.documents) * (m.dimension*4 + 100), // rough estimate
	}
}

// Helper functions

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b Vector) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// euclideanDistance calculates the Euclidean distance between two vectors
func euclideanDistance(a, b Vector) float32 {
	if len(a) != len(b) {
		return math.MaxFloat32
	}

	var sum float32
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return float32(math.Sqrt(float64(sum)))
}

// manhattanDistance calculates the Manhattan distance between two vectors
func manhattanDistance(a, b Vector) float32 {
	if len(a) != len(b) {
		return math.MaxFloat32
	}

	var sum float32
	for i := range a {
		sum += float32(math.Abs(float64(a[i] - b[i])))
	}

	return sum
}

// sortResults sorts search results by score (descending)
func sortResults(results []*VectorSearchResult) []*VectorSearchResult {
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Score < results[j].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
	return results
}

// Normalize normalizes a vector to unit length
func Normalize(v Vector) Vector {
	norm := float32(0)
	for _, val := range v {
		norm += val * val
	}
	norm = float32(math.Sqrt(float64(norm)))

	if norm == 0 {
		return v
	}

	normalized := make(Vector, len(v))
	for i, val := range v {
		normalized[i] = val / norm
	}
	return normalized
}

// AddVectors performs vector addition
func AddVectors(a, b Vector) Vector {
	if len(a) != len(b) {
		return nil
	}

	result := make(Vector, len(a))
	for i := range a {
		result[i] = a[i] + b[i]
	}
	return result
}

// SubtractVectors performs vector subtraction (a - b)
func SubtractVectors(a, b Vector) Vector {
	if len(a) != len(b) {
		return nil
	}

	result := make(Vector, len(a))
	for i := range a {
		result[i] = a[i] - b[i]
	}
	return result
}

// MultiplyScalar performs scalar multiplication
func MultiplyScalar(v Vector, scalar float32) Vector {
	result := make(Vector, len(v))
	for i, val := range v {
		result[i] = val * scalar
	}
	return result
}

// DotProduct calculates the dot product of two vectors
func DotProduct(a, b Vector) float32 {
	if len(a) != len(b) {
		return 0
	}

	var result float32
	for i := range a {
		result += a[i] * b[i]
	}
	return result
}
