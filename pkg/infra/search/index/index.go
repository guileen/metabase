package index

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/guileen/metabase/pkg/common/errors"
)

// Document represents a searchable document
type Document struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title"`
	Content  string                 `json:"content"`
	Type     string                 `json:"type"`
	Source   string                 `json:"source"`
	Tags     []string               `json:"tags"`
	Metadata map[string]interface{} `json:"metadata"`
	Created  time.Time              `json:"created"`
	Updated  time.Time              `json:"updated"`
}

// SearchQuery represents a search query
type SearchQuery struct {
	Query     string            `json:"query"`
	Type      string            `json:"type,omitempty"`
	Source    string            `json:"source,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	Limit     int               `json:"limit,omitempty"`
	Offset    int               `json:"offset,omitempty"`
	Filters   map[string]string `json:"filters,omitempty"`
	SortBy    string            `json:"sort_by,omitempty"`
	SortOrder string            `json:"sort_order,omitempty"`
}

// SearchResult represents a search result
type SearchResult struct {
	Documents []*Document   `json:"documents"`
	Total     int           `json:"total"`
	Query     string        `json:"query"`
	Duration  time.Duration `json:"duration"`
}

// Index represents a search index interface
type Index interface {
	// Add adds a document to the index
	Add(ctx context.Context, doc *Document) error

	// Update updates a document in the index
	Update(ctx context.Context, doc *Document) error

	// Remove removes a document from the index
	Remove(ctx context.Context, id string) error

	// Search performs a search query
	Search(ctx context.Context, query *SearchQuery) (*SearchResult, error)

	// Get retrieves a document by ID
	Get(ctx context.Context, id string) (*Document, error)

	// Clear clears the entire index
	Clear(ctx context.Context) error

	// Stats returns index statistics
	Stats() map[string]interface{}
}

// MemoryIndex is an in-memory implementation of the search index
type MemoryIndex struct {
	documents map[string]*Document
	index     map[string][]string // keyword -> document IDs
	mutex     sync.RWMutex
}

// NewMemoryIndex creates a new in-memory search index
func NewMemoryIndex() *MemoryIndex {
	return &MemoryIndex{
		documents: make(map[string]*Document),
		index:     make(map[string][]string),
	}
}

// Add adds a document to the memory index
func (m *MemoryIndex) Add(ctx context.Context, doc *Document) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if doc.ID == "" {
		return errors.InvalidInput("document ID is required")
	}

	// Remove existing document if it exists
	if existing, exists := m.documents[doc.ID]; exists {
		m.removeFromIndex(existing)
	}

	// Add document
	m.documents[doc.ID] = doc

	// Index the document
	m.indexDocument(doc)

	return nil
}

// Update updates a document in the memory index
func (m *MemoryIndex) Update(ctx context.Context, doc *Document) error {
	return m.Add(ctx, doc) // Add handles both add and update
}

// Remove removes a document from the memory index
func (m *MemoryIndex) Remove(ctx context.Context, id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	doc, exists := m.documents[id]
	if !exists {
		return errors.NotFound("document")
	}

	// Remove from documents
	delete(m.documents, id)

	// Remove from index
	m.removeFromIndex(doc)

	return nil
}

// Search performs a search query on the memory index
func (m *MemoryIndex) Search(ctx context.Context, query *SearchQuery) (*SearchResult, error) {
	start := time.Now()
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if query == nil {
		return nil, errors.InvalidInput("search query is required")
	}

	var results []*Document
	var documentIDs []string

	// Simple keyword-based search
	if query.Query != "" {
		keywords := m.extractKeywords(query.Query)
		documentIDSet := make(map[string]bool)

		for _, keyword := range keywords {
			if ids, exists := m.index[keyword]; exists {
				for _, id := range ids {
					documentIDSet[id] = true
				}
			}
		}

		for id := range documentIDSet {
			documentIDs = append(documentIDs, id)
		}
	} else {
		// Return all documents if no query
		for id := range m.documents {
			documentIDs = append(documentIDs, id)
		}
	}

	// Apply filters
	for _, id := range documentIDs {
		doc := m.documents[id]
		if m.matchesFilters(doc, query) {
			results = append(results, doc)
		}
	}

	// Sort results (simple implementation)
	results = m.sortResults(results, query)

	// Apply pagination
	total := len(results)
	if query.Offset > 0 {
		if query.Offset >= len(results) {
			results = []*Document{}
		} else {
			results = results[query.Offset:]
		}
	}

	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}

	return &SearchResult{
		Documents: results,
		Total:     total,
		Query:     query.Query,
		Duration:  time.Since(start),
	}, nil
}

// Get retrieves a document by ID
func (m *MemoryIndex) Get(ctx context.Context, id string) (*Document, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	doc, exists := m.documents[id]
	if !exists {
		return nil, errors.NotFound("document")
	}

	// Return a copy to avoid modification
	docCopy := *doc
	return &docCopy, nil
}

// Clear clears the entire index
func (m *MemoryIndex) Clear(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.documents = make(map[string]*Document)
	m.index = make(map[string][]string)
	return nil
}

// Stats returns index statistics
func (m *MemoryIndex) Stats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return map[string]interface{}{
		"documents_count": len(m.documents),
		"keywords_count":  len(m.index),
		"memory_usage":    fmt.Sprintf("%d bytes", m.estimateMemoryUsage()),
	}
}

// Helper methods

func (m *MemoryIndex) indexDocument(doc *Document) {
	// Simple keyword extraction from title and content
	text := doc.Title + " " + doc.Content
	keywords := m.extractKeywords(text)

	for _, keyword := range keywords {
		m.index[keyword] = append(m.index[keyword], doc.ID)
	}

	// Index tags
	for _, tag := range doc.Tags {
		m.index["tag:"+tag] = append(m.index["tag:"+tag], doc.ID)
	}

	// Index type
	if doc.Type != "" {
		m.index["type:"+doc.Type] = append(m.index["type:"+doc.Type], doc.ID)
	}

	// Index source
	if doc.Source != "" {
		m.index["source:"+doc.Source] = append(m.index["source:"+doc.Source], doc.ID)
	}
}

func (m *MemoryIndex) removeFromIndex(doc *Document) {
	// Remove from keyword index
	text := doc.Title + " " + doc.Content
	keywords := m.extractKeywords(text)

	for _, keyword := range keywords {
		if ids, exists := m.index[keyword]; exists {
			m.index[keyword] = m.removeID(ids, doc.ID)
			if len(m.index[keyword]) == 0 {
				delete(m.index, keyword)
			}
		}
	}

	// Remove from tags index
	for _, tag := range doc.Tags {
		tagKey := "tag:" + tag
		if ids, exists := m.index[tagKey]; exists {
			m.index[tagKey] = m.removeID(ids, doc.ID)
			if len(m.index[tagKey]) == 0 {
				delete(m.index, tagKey)
			}
		}
	}

	// Remove from type index
	if doc.Type != "" {
		typeKey := "type:" + doc.Type
		if ids, exists := m.index[typeKey]; exists {
			m.index[typeKey] = m.removeID(ids, doc.ID)
			if len(m.index[typeKey]) == 0 {
				delete(m.index, typeKey)
			}
		}
	}

	// Remove from source index
	if doc.Source != "" {
		sourceKey := "source:" + doc.Source
		if ids, exists := m.index[sourceKey]; exists {
			m.index[sourceKey] = m.removeID(ids, doc.ID)
			if len(m.index[sourceKey]) == 0 {
				delete(m.index, sourceKey)
			}
		}
	}
}

func (m *MemoryIndex) extractKeywords(text string) []string {
	// Very simple keyword extraction - split on spaces and remove empty strings
	var keywords []string
	for _, word := range strings.Fields(strings.ToLower(text)) {
		if len(word) > 2 { // Skip very short words
			keywords = append(keywords, word)
		}
	}
	return keywords
}

func (m *MemoryIndex) removeID(ids []string, targetID string) []string {
	var result []string
	for _, id := range ids {
		if id != targetID {
			result = append(result, id)
		}
	}
	return result
}

func (m *MemoryIndex) matchesFilters(doc *Document, query *SearchQuery) bool {
	// Filter by type
	if query.Type != "" && doc.Type != query.Type {
		return false
	}

	// Filter by source
	if query.Source != "" && doc.Source != query.Source {
		return false
	}

	// Filter by tags
	if len(query.Tags) > 0 {
		hasAllTags := true
		for _, tag := range query.Tags {
			found := false
			for _, docTag := range doc.Tags {
				if docTag == tag {
					found = true
					break
				}
			}
			if !found {
				hasAllTags = false
				break
			}
		}
		if !hasAllTags {
			return false
		}
	}

	// Apply custom filters
	for key, value := range query.Filters {
		if key == "created_after" {
			if timestamp, err := time.Parse(time.RFC3339, value); err == nil {
				if doc.Created.Before(timestamp) {
					return false
				}
			}
		}
		if key == "created_before" {
			if timestamp, err := time.Parse(time.RFC3339, value); err == nil {
				if doc.Created.After(timestamp) {
					return false
				}
			}
		}
	}

	return true
}

func (m *MemoryIndex) sortResults(docs []*Document, query *SearchQuery) []*Document {
	// Simple sort by updated time (newest first)
	for i := 0; i < len(docs)-1; i++ {
		for j := i + 1; j < len(docs); j++ {
			if query.SortOrder == "asc" {
				if docs[i].Updated.After(docs[j].Updated) {
					docs[i], docs[j] = docs[j], docs[i]
				}
			} else {
				// default: descending (newest first)
				if docs[i].Updated.Before(docs[j].Updated) {
					docs[i], docs[j] = docs[j], docs[i]
				}
			}
		}
	}
	return docs
}

func (m *MemoryIndex) estimateMemoryUsage() int {
	// Rough estimate
	size := 0
	for _, doc := range m.documents {
		size += len(doc.ID) + len(doc.Title) + len(doc.Content) + len(doc.Type) + len(doc.Source)
		for _, tag := range doc.Tags {
			size += len(tag)
		}
	}

	for keyword, ids := range m.index {
		size += len(keyword)
		size += len(ids) * 8 // rough estimate for string pointers
	}

	return size
}
