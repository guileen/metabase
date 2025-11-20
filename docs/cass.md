# CASS - Code Analysis & Search System

CASS (Code Analysis & Search System) is a unified, extensible platform that combines static code analysis and intelligent search capabilities. Built for the MetaBase project, CASS provides reusable algorithms and shared capabilities that benefit both code analysis and search operations.

## 🚀 Key Features

### Unified Architecture
- **Shared Core**: Single engine powering both analysis and search
- **Reusable Algorithms**: Feature extraction, similarity computation, and indexing
- **Artifact-Centric**: Unified representation for all code elements
- **Multi-Language Support**: Go, JavaScript, TypeScript, Python, Java, C/C++, and more

### Built-in Analyzers
- **Duplicate Detection**: Find exact and near-duplicate code blocks
- **Security Scanning**: OWASP Top 10 vulnerability detection
- **Quality Analysis**: Cyclomatic complexity, maintainability, coverage metrics
- **Custom Analyzers**: Extensible framework for custom analysis rules

### Search Capabilities
- **Full-Text Search**: Fast inverted index for keyword search
- **Vector Search**: Semantic similarity using embeddings
- **Hybrid Search**: Combined text and vector search
- **Pattern Matching**: Regular expressions and code patterns

### CI/CD Integration
- **Quality Gates**: Automated build decisions based on analysis
- **Baseline Tracking**: Issue tracking across builds
- **Multiple Report Formats**: JSON, Markdown, JUnit, SARIF, GitHub Annotations
- **Multi-Platform**: GitHub Actions, GitLab CI, Jenkins, Azure DevOps

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    CASS Engine                          │
├─────────────────────────────────────────────────────────┤
│  Unified Artifact Processing Pipeline                   │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌──────────┐ │
│  │ Raw     │→ │ Tokenize│→ │ Parse   │→ │ Analyze  │ │
│  │ Artifact│  │ Content │  │ AST     │  │ & Index  │ │
│  └─────────┘  └─────────┘  └─────────┘  └──────────┘ │
├─────────────────────────────────────────────────────────┤
│  Feature Extraction & Vectorization                     │
│  ┌─────────────┐ ┌─────────────┐ ┌──────────────────┐ │
│  │ Lexical     │ │ Syntactic   │ │ Semantic         │ │
│  │ Features    │ │ Features    │ │ Features         │ │
│  └─────────────┘ └─────────────┘ └──────────────────┘ │
├─────────────────────────────────────────────────────────┤
│  Core Components                                         │
│  ┌─────────────┐ ┌─────────────┐ ┌──────────────────┐ │
│  │ Analyzers   │ │ Indexes     │ │ Search Engine    │ │
│  │ - Security  │ │ - Inverted  │ │ - Full-Text      │ │
│  │ - Quality   │ │ - Vector    │ │ - Semantic       │ │
│  │ - Duplicate │ │ - Hybrid    │ │ - Hybrid         │ │
│  └─────────────┘ └─────────────┘ └──────────────────┘ │
├─────────────────────────────────────────────────────────┤
│  Integration Layer                                      │
│  ┌─────────────┐ ┌─────────────┐ ┌──────────────────┐ │
│  │ REST API    │ │ WebSocket   │ │ NATS Messaging   │ │
│  │ - Analysis  │ │ - Real-time │ │ - Async Tasks    │ │
│  │ - Search    │ │ - Events    │ │ - Notifications  │ │
│  └─────────────┘ └─────────────┘ └──────────────────┘ │
├─────────────────────────────────────────────────────────┤
│  CI/CD Pipeline Integration                             │
│  ┌─────────────┐ ┌─────────────┐ ┌──────────────────┐ │
│  │ Quality     │ │ Report      │ │ Baseline         │ │
│  │ Gates       │ │ Generation  │ │ Management       │ │
│  │ - Thresholds│ │ - Multi     │ │ - Issue Tracking │ │
│  │ - Policies  │ │   Formats   │ │ - Trend Analysis │ │
│  └─────────────┘ └─────────────┘ └──────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

## 📁 Project Structure

```
internal/analysis/
├── engine.go              # Core CASS engine
├── analyzers.go           # Built-in analyzers
├── integration.go         # API and WebSocket integration
├── ci_integration.go      # CI/CD pipeline integration
├── config.go             # Configuration management
├── reporters.go           # Report generation
├── search.go             # Search functionality
└── processors.go         # Artifact processing

internal/search/
├── engine/
│   └── engine.go         # Search engine implementation
├── index/
│   └── inverted_index.go # Full-text search index
└── vector/
    └── hnsw_index.go     # Vector similarity search

storage/
├── hybrid.go             # SQLite + Pebble hybrid storage
└── engine.go             # Storage abstraction
```

## 🚀 Getting Started

### Installation

```bash
# Clone the repository
git clone https://github.com/metabase/metabase.git
cd metabase

# Install dependencies
go mod tidy

# Build CASS-enabled binary
go build -o bin/metabase ./cmd/metabase
```

### Basic Usage

```bash
# Start with CASS enabled
./bin/metabase server --cass-enabled

# Run analysis on specific files
./bin/metabase cass analyze --path ./internal/analysis

# Search for code
./bin/metabase cass search --query "security vulnerability" --type semantic

# Run CI analysis
./bin/metabase cass ci --config .cass.yaml
```

### Configuration

Create `.cass.yaml` in your project root:

```yaml
# Analysis Settings
analyze_all_files: false
incremental_mode: true
fail_on_new_issues: true
fail_on_severity: "high"
parallelism: 4
timeout: "30m"

# File Patterns
include_patterns:
  - "**/*.go"
  - "**/*.js"
  - "**/*.ts"
  - "**/*.py"

exclude_patterns:
  - "**/vendor/**"
  - "**/node_modules/**"

# Enabled Analyzers
enabled_analyzers:
  - "duplicate-detector"
  - "security-scanner"
  - "quality-analyzer"

# Quality Thresholds
thresholds:
  quality_score: 70.0
  security_score: 80.0
  duplication_ratio: 5.0
  test_coverage: 70.0

# Reporting
report_formats:
  - "json"
  - "markdown"
  - "junit"
  - "github-annotations"
  - "sarif"

output_directory: "./cass-reports"
```

## 🔧 CI/CD Integration

### GitHub Actions

```yaml
name: CASS Analysis
on: [push, pull_request]

jobs:
  cass-analysis:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Run CASS Analysis
      env:
        CASS_FAIL_ON_NEW_ISSUES: "true"
        CASS_QUALITY_THRESHOLD: "70"
        CASS_REPORT_FORMATS: "json,markdown,junit,github-annotations,sarif"
      run: |
        go build -o bin/cass ./cmd/metabase
        ./bin/cass analyze --config .cass.yaml --ci

    - name: Upload Reports
      uses: actions/upload-artifact@v3
      with:
        name: cass-reports
        path: cass-reports/

    - name: Upload SARIF
      uses: github/codeql-action/upload-sarif@v2
      with:
        sarif_file: cass-reports/cass-report.sarif
```

### GitLab CI

```yaml
cass_analysis:
  stage: test
  image: golang:1.21
  script:
    - go build -o bin/cass ./cmd/metabase
    - ./bin/cass analyze --config .cass.yaml --ci
  artifacts:
    reports:
      junit: cass-reports/cass-junit.xml
    paths:
      - cass-reports/
    expire_in: 1 week
  only:
    - merge_requests
    - main
```

## 📊 Analyzers

### Duplicate Detector

Finds exact and near-duplicate code blocks using:

- **Token-based Analysis**: Language-specific tokenization
- **Structural Comparison**: AST pattern matching
- **Similarity Scoring**: Jaccard similarity and Levenshtein distance
- **Fingerprinting**: Hash-based quick comparison

```bash
# Run duplicate detection
./bin/metabase cass analyze --analyzers duplicate-detector --threshold 0.8

# Find duplicates for specific file
./bin/metabase cass duplicate --file ./internal/server.go --threshold 0.7
```

### Security Scanner

OWASP Top 10 vulnerability detection:

- **SQL Injection**: Parameterized query checking
- **XSS**: Input sanitization validation
- **Hard-coded Secrets**: Credential detection
- **Insecure Random**: Crypto API validation
- **Path Traversal**: File access validation

```bash
# Run security scan
./bin/metabase cass analyze --analyzers security-scanner

# Generate security report
./bin/metabase cass security-report --format sarif
```

### Quality Analyzer

Code quality metrics and analysis:

- **Cyclomatic Complexity**: Control flow complexity
- **Maintainability Index**: Code maintainability score
- **Test Coverage**: Coverage analysis and estimation
- **Documentation Ratio**: Comment and documentation metrics
- **Code Duplication**: Duplicate code percentage

```bash
# Run quality analysis
./bin/metabase cass analyze --analyzers quality-analyzer

# Get quality metrics
./bin/metabase cass quality-metrics --format json
```

## 🔍 Search Capabilities

### Full-Text Search

Fast keyword-based search with inverted index:

```bash
# Search for keywords
./bin/metabase cass search --query "authentication middleware" --type fulltext

# Advanced search with filters
./bin/metabase cass search --query "database" --language go --type security
```

### Semantic Search

Vector-based semantic similarity:

```bash
# Semantic search
./bin/metabase cass search --query "user authentication flow" --type semantic

# Hybrid search combining text and semantic
./bin/metabase cass search --query "SQL query builder" --type hybrid
```

### Pattern Search

Regular expression and code pattern matching:

```bash
# Pattern search
./bin/metabase cass search --pattern "func.*\w+.*\(" --language go

# Security pattern search
./bin/metabase cass search --pattern "query.*\+.*input" --type security
```

## 📈 API Integration

### REST API

```bash
# Analyze code via API
curl -X POST http://localhost:7609/api/v1/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "artifact": {
      "path": "main.go",
      "content": "package main\nfunc main() {}",
      "language": "go"
    }
  }'

# Search via API
curl -X POST http://localhost:7609/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "text": "database connection",
    "type": "hybrid",
    "limit": 10
  }'
```

### WebSocket API

```javascript
// Real-time search and analysis
const ws = new WebSocket('ws://localhost:7609/api/v1/ws');

ws.send(JSON.stringify({
  type: 'search',
  data: {
    text: 'security vulnerability',
    type: 'semantic'
  }
}));
```

## 📋 Report Formats

### JSON Report

```json
{
  "context": {
    "repository": "metabase/metabase",
    "branch": "main",
    "commit": "abc123"
  },
  "summary": {
    "status": "passed",
    "total_issues": 5,
    "overall_score": 85.5
  },
  "artifacts": [...],
  "issues": {...}
}
```

### Markdown Report

Human-readable report with:
- Executive summary
- Issue breakdown by severity
- Failed artifact details
- Recommendations
- Trend analysis

### JUnit XML

Standard JUnit format for CI integration:
```xml
<testsuite name="CASS Analysis" tests="100" failures="2" errors="1">
  <testcase name="internal/server.go" time="0.123">
    <failure message="Security vulnerability detected"/>
  </testcase>
</testsuite>
```

### SARIF

Security analysis results interchange format:
```json
{
  "$schema": "https://json.schemastore.org/sarif-2.1.0",
  "version": "2.1.0",
  "runs": [{
    "tool": {"driver": {"name": "CASS"}},
    "results": [...]
  }]
}
```

## 🎯 Best Practices

### Performance Optimization

1. **Parallel Processing**: Use appropriate parallelism based on CPU cores
2. **Incremental Analysis**: Only analyze changed files in CI
3. **Result Caching**: Enable caching for repeated analyses
4. **Batch Operations**: Process multiple artifacts together

### Quality Gates

1. **Gradual Adoption**: Start with warnings, then enforce failures
2. **Baseline Management**: Track issues over time, allow exceptions
3. **Severity-Based Policies**: Focus on critical and high severity issues
4. **Team-Specific Rules**: Customize rules per team and project

### Search Optimization

1. **Hybrid Search**: Combine text and semantic search for best results
2. **Query Suggestions**: Use search history and autocomplete
3. **Result Ranking**: Implement relevance scoring and result ranking
4. **Index Management**: Regularly update and optimize search indexes

## 🔧 Customization

### Custom Analyzers

```go
type CustomAnalyzer struct {
    *BaseAnalyzer
}

func (a *CustomAnalyzer) Analyze(ctx context.Context, artifact *Artifact) (*AnalysisResult, error) {
    // Custom analysis logic
    return result, nil
}

// Register analyzer
engine.RegisterAnalyzer(NewCustomAnalyzer())
```

### Custom Reporters

```go
type CustomReporter struct {
    outputDir string
}

func (r *CustomReporter) Generate(ctx context.Context, results *CIResults) error {
    // Custom report generation
    return nil
}
```

### Custom Rules

```yaml
# .cass-rules.yaml
custom_rules:
  - id: "CUSTOM-001"
    name: "Custom Pattern Check"
    pattern: "TODO|FIXME|HACK"
    severity: "medium"
    message: "Contains temporary code markers"
```

## 🐛 Troubleshooting

### Common Issues

1. **Memory Usage**: Large codebases may require increased memory limits
2. **Analysis Timeout**: Increase timeout for complex analyses
3. **False Positives**: Tune analyzers and add ignore patterns
4. **Performance**: Use incremental analysis and caching

### Debug Mode

```bash
# Enable debug logging
export CASS_DEBUG=true

# Run with verbose output
./bin/metabase cass analyze --verbose --debug

# Check engine stats
curl http://localhost:7609/api/v1/stats
```

## 🤝 Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Add tests for new functionality
4. Ensure all tests pass (`go test ./...`)
5. Run CASS analysis on your changes
6. Submit pull request

### Development Setup

```bash
# Install development dependencies
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/air-verse/air@latest  # Hot reload

# Run development server with hot reload
air

# Run tests
go test ./internal/analysis/... -v

# Run CASS on itself
./bin/metabase cass analyze --path ./internal/analysis
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](../../LICENSE) file for details.

## 🙏 Acknowledgments

- **NATS**: High-performance messaging system
- **SQLite**: Reliable embedded database
- **Pebble**: High-performance KV storage
- **HNSW**: Efficient approximate nearest neighbor search
- **OWASP**: Security standards and guidelines