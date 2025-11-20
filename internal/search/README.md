# Integrated Search System

A powerful, embedded search engine deeply integrated with SQLite and Pebble, inspired by Tantivy's architecture and modern vector database principles.

## Features

### 🔍 **Full-Text Search**
- Google-like text search with highlighting
- Multiple tokenizers (Unicode, lowercase, stemmer, n-gram)
- BM25 ranking algorithm
- Phrase matching and boolean queries
- Multi-language support

### 🧠 **Vector Search**
- Semantic search using embeddings
- HNSW algorithm for fast approximate nearest neighbor search
- Multiple distance metrics (cosine, L2, inner product)
- Hybrid search combining full-text and vector

### 📁 **File Search**
- Search inside documents (PDF, Word, text)
- Metadata-based file search
- Content extraction and indexing
- Deduplication using content hashing

### 🏗️ **Embedded Architecture**
- Zero external dependencies
- Runs in-process with your application
- ACID transactions across all data
- Single backup point

## Quick Start

```go
package main

import (
    "context"
    "log"

    "github.com/metabase/metabase/internal/search"
    "github.com/metabase/metabase/internal/storage"
    "github.com/metabase/metabase/internal/files"
)

func main() {
    // Initialize storage engines
    storageConfig := storage.NewConfig()
    storageEngine, err := storage.NewEngine(storageConfig)
    if err != nil {
        log.Fatal(err)
    }
    defer storageEngine.Close()

    filesConfig := files.NewConfig()
    filesEngine, err := files.NewEngine(filesConfig, storageEngine)
    if err != nil {
        log.Fatal(err)
    }
    defer filesEngine.Close()

    // Initialize search engine
    searchConfig := search.NewConfig()
    searchEngine, err := search.NewEngine(searchConfig, storageEngine, filesEngine)
    if err != nil {
        log.Fatal(err)
    }
    defer searchEngine.Close()

    // Create integration
    integration := search.NewIntegration(searchEngine, storageEngine, filesEngine)

    // Hook into storage operations for automatic indexing
    integration.HookIntoStorage()

    // Index existing data
    ctx := context.Background()
    if err := integration.IndexAll(ctx); err != nil {
        log.Printf("Failed to index data: %v", err)
    }

    // Search
    results, err := searchEngine.Search(ctx, &search.Query{
        Text:     "example search query",
        TenantID: "tenant123",
        Limit:    10,
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Found %d results", results.Total)
    for _, hit := range results.Hits {
        log.Printf("- %s: %s", hit.ID, hit.Source["title"])
    }
}
```

## API Reference

### Search API

#### GET /api/v1/search

```json
{
  "q": "search query",
  "type": "hybrid",
  "tenant": "tenant123",
  "table": "documents",
  "limit": 10,
  "offset": 0,
  "filter.category": "tech",
  "filter.date_from": "2024-01-01"
}
```

#### POST /api/v1/search

```json
{
  "text": "search query",
  "type": "vector",
  "filters": {
    "tenant_id": "tenant123",
    "category": "tech"
  },
  "boost_fields": {
    "title": 2.0,
    "content": 1.0
  },
  "limit": 10
}
```

#### Response

```json
{
  "hits": [
    {
      "id": "doc_123",
      "score": 0.95,
      "source": {
        "title": "Document Title",
        "content": "Document content...",
        "strategy": "hybrid"
      },
      "highlights": [
        {
          "field": "content",
          "value": "...<mark>search</mark> query..."
        }
      ]
    }
  ],
  "total": 42,
  "took": "15ms",
  "max_score": 0.95
}
```

### Indexing API

#### POST /api/v1/search/index

```json
{
  "id": "doc_123",
  "tenant_id": "tenant123",
  "table": "documents",
  "title": "Document Title",
  "content": "Document content...",
  "tags": ["tech", "search"],
  "created_at": "2024-01-01T00:00:00Z"
}
```

## Configuration

### Full-Text Search Configuration

```go
FullTextConfig{
    Tokenizers:    []string{"unicode", "lowercase", "stemmer"},
    NGramSize:     3,
    MinTermLength: 2,
    MaxTermLength: 50,
}
```

### Vector Search Configuration

```go
VectorConfig{
    Dimension:      384, // Embedding dimension
    Distance:       "cosine",
    EmbeddingModel: "all-MiniLM-L6-v2",
    HNSW: HNSWConfig{
        M:             16,  // Connections per node
        EfConstruction: 200, // Build-time accuracy
        EfSearch:      50,  // Search-time accuracy
    },
}
```

## Performance Tuning

### 1. Indexing Performance

- Batch index operations for better throughput
- Use asynchronous indexing for real-time updates
- Tune index buffer size based on memory constraints

```go
config := search.NewConfig()
config.IndexBuffer = 1000  // Buffer size for batch processing
config.MergeThreads = 2    // Number of merge threads
```

### 2. Query Performance

- Use tenant filtering for multi-tenant setups
- Limit result sets with pagination
- Choose appropriate query type

```go
query := &search.Query{
    Text:     "search",
    Type:     search.QueryTypeHybrid,
    TenantID: tenantID,  // Always include tenant for multi-tenant
    Limit:    20,        // Reasonable limit
    Offset:   0,
}
```

### 3. Memory Usage

- Adjust cache size based on available memory
- Vector indexes require memory for HNSW graph

```go
config := search.NewConfig()
config.CacheSize = 100 * 1024 * 1024 // 100MB cache
```

## Architecture Overview

```
┌─────────────────────────────────────┐
│         Application Layer           │
├─────────────────────────────────────┤
│  Search Engine (Tantivy-inspired)  │
│  ┌─────────────┐ ┌─────────────────┐ │
│  │   Inverted  │ │   Vector Index  │ │
│  │   Index     │ │    (HNSW)       │ │
│  └─────────────┘ └─────────────────┘ │
│  ┌─────────────┐ ┌─────────────────┐ │
│  │ Tokenizer   │ │   Embedding     │ │
│  │ Pipeline    │ │   Generator     │ │
│  └─────────────┘ └─────────────────┘ │
├─────────────────────────────────────┤
│        Storage Integration          │
│  ┌─────────────┐ ┌─────────────────┐ │
│  │   SQLite    │ │    Pebble       │ │
│  │ (Metadata)  │ │ (Vectors/Cache) │ │
│  └─────────────┘ └─────────────────┘ │
│  ┌─────────────────────────────────┐ │
│  │        File System              │ │
│  │    (Content-addressable)        │ │
│  └─────────────────────────────────┘ │
└─────────────────────────────────────┘
```

## Data Flow

### Indexing Flow

1. **Document Creation** → Storage engine creates record
2. **Hook Triggered** → Search integration notified
3. **Content Extraction** → Text extracted from fields/files
4. **Tokenization** → Text broken into tokens
5. **Inverted Index** → Tokens stored in SQLite with positions
6. **Embedding Generation** → Vector created from text
7. **Vector Index** → Vector stored in HNSW structure (Pebble)

### Search Flow

1. **Query Parsing** → Analyze query type and parameters
2. **Query Planning** → Choose optimal strategy
3. **Parallel Execution** → Full-text and/or vector search
4. **Result Merging** → Combine and rank results
5. **Filtering** → Apply tenant and field filters
6. **Pagination** → Return requested slice

## Multi-Tenancy

The search system is designed for multi-tenancy:

```go
// Automatic tenant isolation
query := &search.Query{
    Text:     "search",
    TenantID: "tenant123",  // Isolates results
}

// Documents are automatically tagged with tenant
doc := &search.Document{
    ID:       "doc_123",
    TenantID: "tenant123",
    Content:  "Document content",
}
```

## File Search Integration

Search inside documents:

```go
// Index a file
err := integration.IndexFile(ctx, fileID)

// Files are automatically extracted and indexed
// - PDF: Text extraction
// - Office documents: Content parsing
// - Images: OCR (configurable)
// - Audio: Speech-to-text (configurable)
```

## Comparison with Traditional Stack

| Feature | Traditional Stack | Integrated Search |
|---------|------------------|-------------------|
| **Services** | 5+ (Elasticsearch, S3, PostgreSQL, Redis, etc.) | 1 process |
| **Network** | Multiple network hops | In-process calls |
| **Transactions** | Eventually consistent | ACID across all data |
| **Backup** | Multiple systems to backup | Single filesystem |
| **Scaling** | Horizontal scaling | Vertical scaling |
| **Complexity** | High operational overhead | Zero configuration |
| **Latency** | Network latency added | In-memory speed |

## Best Practices

### 1. Indexing Strategy

- Index important fields first (title, content)
- Use field boosting for relevance
- Index files asynchronously

### 2. Query Optimization

- Use specific query types when possible
- Apply tenant filtering at search time
- Use pagination for large result sets

### 3. Maintenance

- Regular index optimization
- Monitor cache hit rates
- Track query performance

### 4. Security

- Always include tenant_id in multi-tenant setups
- Validate query parameters
- Use RBAC for search access control

## Advanced Features

### 1. Custom Tokenizers

```go
// Add custom tokenizer
type CustomTokenizer struct{}

func (ct *CustomTokenizer) Tokenize(tokens []string) ([]string, error) {
    // Custom tokenization logic
    return tokens, nil
}
```

### 2. Custom Embedding Models

```go
// Replace with ONNX model
embedder := &ONNXEmbedder{
    ModelPath: "model.onnx",
    Dimension: 768,
}
```

### 3. Query-Time Boosting

```go
query := &search.Query{
    Text: "search query",
    BoostFields: map[string]float64{
        "title":   2.0,  // Boost title matches
        "content": 1.0,  // Normal content weight
    },
}
```

## Monitoring

### Search Statistics

```go
stats := searchEngine.Stats()
fmt.Printf("Documents indexed: %d\n", stats.DocumentsIndexed)
fmt.Printf("Queries served: %d\n", stats.QueriesServed)
fmt.Printf("Average query time: %v\n", stats.AvgQueryTime)
fmt.Printf("Cache hit rate: %.2f%%\n", stats.CacheHitRate*100)
```

### Health Checks

```go
// Check search engine health
err := searchEngine.Ping()
if err != nil {
    log.Printf("Search engine unhealthy: %v", err)
}
```

## Troubleshooting

### Common Issues

1. **Slow queries**
   - Check index size
   - Increase cache size
   - Optimize query type

2. **High memory usage**
   - Reduce vector dimensions
   - Adjust HNSW parameters
   - Limit index buffer size

3. **Missing results**
   - Check document indexing
   - Verify tenant filtering
   - Validate query syntax

## Development

### Running Tests

```bash
cd internal/search
go test -v ./...
```

### Building

```bash
go build -o metabase-search ./cmd/search
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Submit a pull request

## License

This search system is part of the Metabase project and follows the same license terms.