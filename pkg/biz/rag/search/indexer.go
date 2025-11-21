package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/guileen/metabase/pkg/biz/rag/embedding"
	"github.com/guileen/metabase/pkg/biz/rag/search/engine"
	"github.com/guileen/metabase/pkg/infra/skills"
)

// WorkspaceIndexer handles indexing of workspace files
type WorkspaceIndexer struct {
	config          *WorkspaceConfig
	searchEngine    *engine.Engine
	templateManager *skills.TemplateManager
	embedder        embedding.Embedder
	indexPath       string
	mutex           sync.RWMutex

	// Indexing state
	indexing      bool
	indexProgress *IndexProgress
	cancelIndex   context.CancelFunc
	indexWorkers  int

	// File tracking
	lastIndexTime time.Time
	trackedFiles  map[string]*FileInfo
}

// IndexProgress tracks indexing progress
type IndexProgress struct {
	TotalFiles     int           `json:"total_files"`
	ProcessedFiles int           `json:"processed_files"`
	FailedFiles    int           `json:"failed_files"`
	CurrentFile    string        `json:"current_file"`
	StartTime      time.Time     `json:"start_time"`
	EstimatedETA   time.Duration `json:"estimated_eta"`
	Speed          float64       `json:"files_per_second"`
	Errors         []string      `json:"errors,omitempty"`
}

// FileInfo contains information about a tracked file
type FileInfo struct {
	Path         string
	Size         int64
	LastModified time.Time
	Hash         string
	ContentType  string
	Language     string
	Tags         []string
	Metadata     map[string]interface{}
}

// IndexStats contains indexing statistics
type IndexStats struct {
	TotalFiles      int            `json:"total_files"`
	TotalSize       int64          `json:"total_size"`
	IndexedFiles    int            `json:"indexed_files"`
	LastIndexTime   time.Time      `json:"last_index_time"`
	IndexDuration   time.Duration  `json:"index_duration"`
	FilesByType     map[string]int `json:"files_by_type"`
	FilesByLanguage map[string]int `json:"files_by_language"`
}

// NewWorkspaceIndexer creates a new workspace indexer
func NewWorkspaceIndexer(config *WorkspaceConfig) (*WorkspaceIndexer, error) {
	if config == nil {
		config = getDefaultWorkspaceConfig()
	}

	wi := &WorkspaceIndexer{
		config:          config,
		templateManager: skills.NewTemplateManager(),
		trackedFiles:    make(map[string]*FileInfo),
	}

	// Initialize embedder if enabled
	if config.EnableEmbeddings {
		var err error
		wi.embedder, err = embedding.NewLocalEmbedder(&embedding.Config{
			LocalModelType: "python",
			Model:          config.EmbeddingModel,
			EnableFallback: true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize embedder: %w", err)
		}
	}

	// Setup index path
	wi.indexPath = filepath.Join(config.RootPath, ".metabase", "index")
	if err := os.MkdirAll(wi.indexPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create index directory: %w", err)
	}

	// Load existing index metadata
	if err := wi.loadIndexMetadata(); err != nil {
		// Log warning but continue
		fmt.Printf("Warning: failed to load index metadata: %v\n", err)
	}

	return wi, nil
}

// StartIndexing starts the indexing process
func (wi *WorkspaceIndexer) StartIndexing(ctx context.Context) error {
	wi.mutex.Lock()
	defer wi.mutex.Unlock()

	if wi.indexing {
		return fmt.Errorf("indexing already in progress")
	}

	ctx, cancel := context.WithCancel(ctx)
	wi.cancelIndex = cancel

	wi.indexing = true
	wi.indexProgress = &IndexProgress{
		StartTime: time.Now(),
		Errors:    make([]string, 0),
	}

	// Start indexing in goroutine
	go wi.performIndexing(ctx)

	return nil
}

// StopIndexing stops the indexing process
func (wi *WorkspaceIndexer) StopIndexing() error {
	wi.mutex.Lock()
	defer wi.mutex.Unlock()

	if !wi.indexing {
		return fmt.Errorf("no indexing in progress")
	}

	if wi.cancelIndex != nil {
		wi.cancelIndex()
	}

	wi.indexing = false
	return nil
}

// GetIndexProgress returns current indexing progress
func (wi *WorkspaceIndexer) GetIndexProgress() *IndexProgress {
	wi.mutex.RLock()
	defer wi.mutex.RUnlock()

	if wi.indexProgress == nil {
		return &IndexProgress{}
	}

	// Calculate progress metrics
	if wi.indexProgress.ProcessedFiles > 0 {
		elapsed := time.Since(wi.indexProgress.StartTime)
		wi.indexProgress.Speed = float64(wi.indexProgress.ProcessedFiles) / elapsed.Seconds()

		if wi.indexProgress.TotalFiles > 0 {
			remaining := wi.indexProgress.TotalFiles - wi.indexProgress.ProcessedFiles
			if wi.indexProgress.Speed > 0 {
				wi.indexProgress.EstimatedETA = time.Duration(float64(remaining)/wi.indexProgress.Speed) * time.Second
			}
		}
	}

	// Return a copy to avoid race conditions
	progressCopy := *wi.indexProgress
	return &progressCopy
}

// IsIndexing checks if indexing is currently in progress
func (wi *WorkspaceIndexer) IsIndexing() bool {
	wi.mutex.RLock()
	defer wi.mutex.RUnlock()
	return wi.indexing
}

// GetIndexStats returns indexing statistics
func (wi *WorkspaceIndexer) GetIndexStats() *IndexStats {
	wi.mutex.RLock()
	defer wi.mutex.RUnlock()

	stats := &IndexStats{
		TotalFiles:      len(wi.trackedFiles),
		LastIndexTime:   wi.lastIndexTime,
		FilesByType:     make(map[string]int),
		FilesByLanguage: make(map[string]int),
	}

	// Calculate statistics
	for _, file := range wi.trackedFiles {
		stats.TotalSize += file.Size
		stats.FilesByType[getFileType(file.Path)]++

		if file.Language != "" {
			stats.FilesByLanguage[file.Language]++
		}
	}

	return stats
}

// performIndexing performs the actual indexing work
func (wi *WorkspaceIndexer) performIndexing(ctx context.Context) {
	defer func() {
		wi.mutex.Lock()
		wi.indexing = false
		wi.cancelIndex = nil
		wi.mutex.Unlock()
	}()

	// Discover files
	files, err := wi.discoverFiles(ctx)
	if err != nil {
		wi.addError(fmt.Sprintf("Failed to discover files: %v", err))
		return
	}

	wi.indexProgress.TotalFiles = len(files)

	// Process files
	if wi.config.ParallelIndexing {
		wi.processFilesParallel(ctx, files)
	} else {
		wi.processFilesSequential(ctx, files)
	}

	// Update index metadata
	wi.lastIndexTime = time.Now()
	wi.saveIndexMetadata()

	// Final progress update
	wi.mutex.Lock()
	wi.indexProgress.CurrentFile = "Indexing complete"
	wi.indexProgress.EstimatedETA = 0
	wi.mutex.Unlock()
}

// discoverFiles discovers all files to be indexed
func (wi *WorkspaceIndexer) discoverFiles(ctx context.Context) ([]string, error) {
	var files []string

	err := filepath.WalkDir(wi.config.RootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Skip if matches exclude patterns
		if wi.shouldExcludeFile(path) {
			return nil
		}

		// Check file size
		info, err := d.Info()
		if err != nil {
			return nil
		}

		if info.Size() > wi.config.MaxFileSize {
			return nil
		}

		// Check if matches include patterns
		if !wi.shouldIncludeFile(path) {
			return nil
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

// processFilesSequential processes files sequentially
func (wi *WorkspaceIndexer) processFilesSequential(ctx context.Context, files []string) {
	for i, filePath := range files {
		select {
		case <-ctx.Done():
			return
		default:
		}

		wi.updateProgress(i, filePath)
		if err := wi.indexFile(filePath); err != nil {
			wi.addError(fmt.Sprintf("Failed to index %s: %v", filePath, err))
		}
	}
}

// processFilesParallel processes files in parallel
func (wi *WorkspaceIndexer) processFilesParallel(ctx context.Context, files []string) {
	workerCount := wi.config.MaxIndexWorkers
	if workerCount <= 0 {
		workerCount = 4
	}

	filesChan := make(chan string, workerCount*2)
	errorsChan := make(chan error, len(files))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go wi.indexWorker(ctx, filesChan, errorsChan, &wg)
	}

	// Send files to workers
	go func() {
		defer close(filesChan)
		for i, filePath := range files {
			select {
			case <-ctx.Done():
				return
			case filesChan <- filePath:
				wi.updateProgress(i, filePath)
			}
		}
	}()

	// Wait for workers to finish
	wg.Wait()
	close(errorsChan)

	// Collect errors
	for err := range errorsChan {
		wi.addError(err.Error())
	}
}

// indexWorker processes files from the files channel
func (wi *WorkspaceIndexer) indexWorker(ctx context.Context, files <-chan string, errors chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	for filePath := range files {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := wi.indexFile(filePath); err != nil {
			errors <- fmt.Errorf("failed to index %s: %w", filePath, err)
		}
	}
}

// indexFile indexes a single file
func (wi *WorkspaceIndexer) indexFile(filePath string) error {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Convert to string and limit size if necessary
	contentStr := string(content)
	if len(contentStr) > 1024*1024 { // 1MB limit
		contentStr = contentStr[:1024*1024]
	}

	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Create file info
	fileInfo := &FileInfo{
		Path:         filePath,
		Size:         info.Size(),
		LastModified: info.ModTime(),
		Hash:         wi.calculateHash(contentStr),
		ContentType:  getContentType(filePath),
		Language:     detectLanguage(filePath),
		Metadata:     make(map[string]interface{}),
	}

	// Check if file needs reindexing
	if wi.shouldReindex(fileInfo) {
		// Generate embeddings if enabled
		var embeddingVector []float64
		if wi.config.EnableEmbeddings && wi.embedder != nil {
			embeddings, err := wi.embedder.Embed([]string{contentStr})
			if err == nil && len(embeddings) > 0 {
				embeddingVector = embeddings[0]
			}
		}

		// Generate tags using skills if enabled
		var tags []string
		if wi.config.EnableSkills {
			tags = wi.generateTags(contentStr, fileInfo)
		}
		fileInfo.Tags = tags

		// Create search document
		doc := &engine.Document{
			ID:      filePath,
			Type:    "file",
			Title:   filepath.Base(filePath),
			Content: contentStr,
			Vector:  embeddingVector,
			Metadata: map[string]interface{}{
				"file_path":     filePath,
				"file_size":     info.Size(),
				"file_type":     filepath.Ext(filePath),
				"content_type":  fileInfo.ContentType,
				"language":      fileInfo.Language,
				"tags":          tags,
				"last_modified": info.ModTime(),
				"indexed_at":    time.Now(),
			},
			Timestamp: time.Now(),
		}

		// Index document
		if wi.searchEngine != nil {
			if err := wi.searchEngine.Index(doc); err != nil {
				return fmt.Errorf("failed to index document: %w", err)
			}
		}
	}

	// Update tracked files
	wi.mutex.Lock()
	wi.trackedFiles[filePath] = fileInfo
	wi.mutex.Unlock()

	return nil
}

// updateProgress updates indexing progress
func (wi *WorkspaceIndexer) updateProgress(processed int, currentFile string) {
	wi.mutex.Lock()
	defer wi.mutex.Unlock()

	if wi.indexProgress != nil {
		wi.indexProgress.ProcessedFiles = processed
		wi.indexProgress.CurrentFile = currentFile
	}
}

// addError adds an error to the progress
func (wi *WorkspaceIndexer) addError(err string) {
	wi.mutex.Lock()
	defer wi.mutex.Unlock()

	if wi.indexProgress != nil {
		wi.indexProgress.FailedFiles++
		wi.indexProgress.Errors = append(wi.indexProgress.Errors, err)

		// Limit error history
		if len(wi.indexProgress.Errors) > 100 {
			wi.indexProgress.Errors = wi.indexProgress.Errors[1:]
		}
	}
}

// Helper functions
func (wi *WorkspaceIndexer) shouldExcludeFile(path string) bool {
	relPath := strings.TrimPrefix(path, wi.config.RootPath)
	relPath = strings.TrimPrefix(relPath, string(filepath.Separator))

	for _, pattern := range wi.config.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
		if strings.Contains(relPath, pattern) {
			return true
		}
	}

	return false
}

func (wi *WorkspaceIndexer) shouldIncludeFile(path string) bool {
	if len(wi.config.IncludePatterns) == 0 {
		return true
	}

	for _, pattern := range wi.config.IncludePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}

	return false
}

func (wi *WorkspaceIndexer) shouldReindex(fileInfo *FileInfo) bool {
	wi.mutex.RLock()
	defer wi.mutex.RUnlock()

	tracked, exists := wi.trackedFiles[fileInfo.Path]
	if !exists {
		return true
	}

	return tracked.LastModified.Before(fileInfo.LastModified) ||
		tracked.Hash != fileInfo.Hash
}

func (wi *WorkspaceIndexer) calculateHash(content string) string {
	// Simple hash implementation - in production, use proper hash like SHA256
	return fmt.Sprintf("%x", len(content)) // Placeholder
}

func (wi *WorkspaceIndexer) generateTags(content string, fileInfo *FileInfo) []string {
	skillInput := &skills.SkillInput{
		Query: content,
		Parameters: map[string]interface{}{
			"content_type": fileInfo.ContentType,
			"file_type":    filepath.Ext(fileInfo.Path),
			"language":     fileInfo.Language,
			"max_tags":     5,
		},
	}

	output, err := wi.templateManager.ExecuteSkill("generateTags", skillInput, nil)
	if err != nil || !output.Success {
		return []string{}
	}

	if result, ok := output.Result.(map[string]interface{}); ok {
		if tags, ok := result["tags"].([]string); ok {
			return tags
		}
	}

	return []string{}
}

func (wi *WorkspaceIndexer) loadIndexMetadata() error {
	metadataPath := filepath.Join(wi.indexPath, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No metadata yet
		}
		return err
	}

	var metadata struct {
		LastIndexTime time.Time            `json:"last_index_time"`
		TrackedFiles  map[string]*FileInfo `json:"tracked_files"`
	}

	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}

	wi.lastIndexTime = metadata.LastIndexTime
	wi.trackedFiles = metadata.TrackedFiles

	return nil
}

func (wi *WorkspaceIndexer) saveIndexMetadata() error {
	metadataPath := filepath.Join(wi.indexPath, "metadata.json")

	metadata := struct {
		LastIndexTime time.Time            `json:"last_index_time"`
		TrackedFiles  map[string]*FileInfo `json:"tracked_files"`
	}{
		LastIndexTime: wi.lastIndexTime,
		TrackedFiles:  wi.trackedFiles,
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metadataPath, data, 0644)
}

// Utility functions for file type detection
func getFileType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "source"
	case ".js", ".ts", ".jsx", ".tsx":
		return "source"
	case ".py", ".pyx":
		return "source"
	case ".java", ".kt", ".scala":
		return "source"
	case ".cpp", ".c", ".h", ".hpp":
		return "source"
	case ".cs":
		return "source"
	case ".rb":
		return "source"
	case ".php":
		return "source"
	case ".swift", ".m":
		return "source"
	case ".rs":
		return "source"
	case ".md", ".txt", ".rst":
		return "documentation"
	case ".json", ".yaml", ".yml", ".toml", ".ini", ".cfg":
		return "config"
	case ".sql":
		return "database"
	default:
		return "other"
	}
}

func getContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "text/x-go"
	case ".js":
		return "application/javascript"
	case ".ts":
		return "application/typescript"
	case ".py":
		return "text/x-python"
	case ".java":
		return "text/x-java"
	case ".md":
		return "text/markdown"
	case ".json":
		return "application/json"
	case ".yaml", ".yml":
		return "application/x-yaml"
	default:
		return "text/plain"
	}
}

func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".js", ".mjs":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".jsx":
		return "javascript-jsx"
	case ".tsx":
		return "typescript-jsx"
	case ".py", ".pyx":
		return "python"
	case ".java":
		return "java"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".c":
		return "c"
	case ".h", ".hpp":
		return "c"
	case ".cs":
		return "csharp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".swift":
		return "swift"
	case ".m":
		return "objective-c"
	case ".rs":
		return "rust"
	case ".md":
		return "markdown"
	case ".sql":
		return "sql"
	default:
		return ""
	}
}
