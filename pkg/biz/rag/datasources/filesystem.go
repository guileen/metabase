package datasources

import (
	"context"
	"crypto/sha256"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/guileen/metabase/pkg/biz/rag/core"
)

// FileSystemDataSource implements a file system data source for RAG
type FileSystemDataSource struct {
	BaseDataSource
	config *FileSystemConfig
	mu     sync.RWMutex
	cache  map[string]*cacheEntry
}

// cacheEntry represents a cached file entry
type cacheEntry struct {
	document   *core.Document
	modTime    time.Time
	expireTime time.Time
}

// NewFileSystemDataSource creates a new file system data source
func NewFileSystemDataSource(id string, config *FileSystemConfig) (*FileSystemDataSource, error) {
	if config == nil {
		config = &FileSystemConfig{}
	}

	// Set defaults
	if config.RootPath == "" {
		config.RootPath = "."
	}
	if config.MaxFileSize == 0 {
		config.MaxFileSize = 10 * 1024 * 1024 // 10MB default
	}
	if config.MaxWorkers == 0 {
		config.MaxWorkers = 4
	}
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}

	// Validate configuration
	if err := validateFileSystemConfig(config); err != nil {
		return nil, fmt.Errorf("invalid filesystem config: %w", err)
	}

	// Convert to absolute path
	rootPath, err := filepath.Abs(config.RootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve root path: %w", err)
	}
	config.RootPath = rootPath

	dataSource := &FileSystemDataSource{
		BaseDataSource: BaseDataSource{
			ID:       id,
			Type:     "filesystem",
			Config:   configToMap(config),
			Metadata: make(map[string]interface{}),
		},
		config: config,
		cache:  make(map[string]*cacheEntry),
	}

	// Initialize metadata
	dataSource.Metadata["root_path"] = config.RootPath
	dataSource.Metadata["recursive"] = config.Recursive
	dataSource.Metadata["created_at"] = time.Now()

	return dataSource, nil
}

// GetID implements the DataSource interface
func (fs *FileSystemDataSource) GetID() string {
	return fs.BaseDataSource.ID
}

// GetType implements the DataSource interface
func (fs *FileSystemDataSource) GetType() string {
	return fs.BaseDataSource.Type
}

// GetConfig implements the DataSource interface
func (fs *FileSystemDataSource) GetConfig() interface{} {
	return fs.config
}

// ListDocuments implements the DataSource interface
func (fs *FileSystemDataSource) ListDocuments(ctx context.Context) ([]core.Document, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var documents []core.Document
	err := filepath.WalkDir(fs.config.RootPath, func(path string, d iofs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue walking on errors
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Check recursion setting
		if !fs.config.Recursive {
			dir := filepath.Dir(path)
			if dir != fs.config.RootPath {
				return filepath.SkipDir
			}
		}

		// Filter files based on configuration
		if !fs.shouldIncludeFile(path, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Create document from file
		doc, err := fs.createDocumentFromFile(path)
		if err != nil {
			// Log error but continue processing other files
			return nil
		}

		documents = append(documents, *doc)
		return nil
	})

	return documents, err
}

// GetDocument implements the DataSource interface
func (fs *FileSystemDataSource) GetDocument(ctx context.Context, documentID string) (*core.Document, error) {
	// Parse document ID (it's the file path)
	filePath := documentID
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(fs.config.RootPath, filePath)
	}

	// Check cache first
	if fs.config.EnableCache {
		if doc := fs.getCachedDocument(filePath); doc != nil {
			return doc, nil
		}
	}

	// Create document from file
	doc, err := fs.createDocumentFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create document from file %s: %w", filePath, err)
	}

	// Cache the document
	if fs.config.EnableCache {
		fs.cacheDocument(filePath, doc)
	}

	return doc, nil
}

// Sync implements the DataSource interface
func (fs *FileSystemDataSource) Sync(ctx context.Context, since time.Time) (*core.SyncResult, error) {
	startTime := time.Now()
	result := &core.SyncResult{
		StartTime:    startTime,
		DataSourceID: fs.ID,
		SyncType:     "incremental",
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Walk through filesystem and find changes
	err := filepath.WalkDir(fs.config.RootPath, func(path string, d iofs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Filter files
		if !fs.shouldIncludeFile(path, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Get file info
		info, err := d.Info()
		if err != nil {
			return nil
		}

		// Check if file was modified since last sync
		if info.ModTime().After(since) {
			result.DocumentsUpdated++
		} else {
			result.DocumentsUnchanged++
		}

		return nil
	})

	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.ErrorCount++
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.LastSyncTime = result.EndTime

	return result, err
}

// Validate implements the DataSource interface
func (fs *FileSystemDataSource) Validate() error {
	// Check if root path exists and is accessible
	info, err := os.Stat(fs.config.RootPath)
	if err != nil {
		return fmt.Errorf("root path not accessible: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("root path is not a directory: %s", fs.config.RootPath)
	}

	// Check if directory is readable
	file, err := os.Open(fs.config.RootPath)
	if err != nil {
		return fmt.Errorf("cannot read root directory: %w", err)
	}
	file.Close()

	// Validate configuration
	return validateFileSystemConfig(fs.config)
}

// Close implements the DataSource interface
func (fs *FileSystemDataSource) Close() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.cache = nil
	return nil
}

// Helper methods

// shouldIncludeFile checks if a file should be included based on configuration
func (fs *FileSystemDataSource) shouldIncludeFile(path string, d iofs.DirEntry) bool {
	// Skip hidden files and directories if configured
	if fs.config.IgnoreHidden && strings.HasPrefix(filepath.Base(path), ".") {
		return false
	}

	// Skip symbolic links if not configured to follow
	if !fs.config.FollowSymlinks {
		if info, err := d.Info(); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return false
		}
	}

	relPath, err := filepath.Rel(fs.config.RootPath, path)
	if err != nil {
		return false
	}

	// Check include patterns
	if len(fs.config.IncludePatterns) > 0 {
		included := false
		for _, pattern := range fs.config.IncludePatterns {
			if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
				included = true
				break
			}
			if matched, _ := filepath.Match(pattern, relPath); matched {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	// Check exclude patterns
	for _, pattern := range fs.config.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return false
		}
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return false
		}
	}

	// Check file size
	info, err := d.Info()
	if err != nil {
		return false
	}

	if fs.config.MaxFileSize > 0 && info.Size() > fs.config.MaxFileSize {
		return false
	}
	if fs.config.MinFileSize > 0 && info.Size() < fs.config.MinFileSize {
		return false
	}

	// Check file type extensions
	ext := strings.ToLower(filepath.Ext(path))

	if len(fs.config.IncludeTypes) > 0 {
		included := false
		for _, includeType := range fs.config.IncludeTypes {
			if !strings.HasPrefix(includeType, ".") {
				includeType = "." + includeType
			}
			if ext == includeType {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	for _, excludeType := range fs.config.ExcludeTypes {
		if !strings.HasPrefix(excludeType, ".") {
			excludeType = "." + excludeType
		}
		if ext == excludeType {
			return false
		}
	}

	return true
}

// createDocumentFromFile creates a document from a file
func (fs *FileSystemDataSource) createDocumentFromFile(filePath string) (*core.Document, error) {
	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	contentStr := string(content)

	// Check content length constraints
	if fs.config.MinLength > 0 && len(contentStr) < fs.config.MinLength {
		return nil, fmt.Errorf("content too short: %d bytes", len(contentStr))
	}
	if fs.config.MaxLength > 0 && len(contentStr) > fs.config.MaxLength {
		return nil, fmt.Errorf("content too long: %d bytes", len(contentStr))
	}

	// Create relative path for URI
	relPath, err := filepath.Rel(fs.config.RootPath, filePath)
	if err != nil {
		relPath = filePath
	}

	// Create document metadata
	metadata := core.DocumentMetadata{
		FilePath:   filePath,
		FileName:   filepath.Base(filePath),
		FileSize:   info.Size(),
		FileType:   fs.getFileType(filePath),
		Extension:  filepath.Ext(filePath),
		CreatedAt:  info.ModTime(), // Use mod time as creation time for simplicity
		ModifiedAt: info.ModTime(),
		AccessedAt: time.Now(),
		Length:     len(contentStr),
		WordCount:  fs.estimateWordCount(contentStr),
		LineCount:  strings.Count(contentStr, "\n") + 1,
	}

	// Extract additional metadata if configured
	if fs.config.ExtractMetadata {
		fs.extractMetadata(filePath, contentStr, &metadata)
	}

	// Create document
	doc := &core.Document{
		ID:           filePath, // Use full path as ID
		Title:        fs.extractTitle(filePath, contentStr),
		Content:      contentStr,
		URI:          relPath,
		SourceType:   "filesystem",
		Metadata:     metadata,
		Tags:         fs.extractTags(filePath, contentStr),
		Language:     fs.detectLanguage(filePath, contentStr),
		ProcessedAt:  time.Now(),
		UpdatedAt:    info.ModTime(),
		Version:      1,
		DataSourceID: fs.ID,
	}

	return doc, nil
}

// getCachedDocument retrieves a document from cache
func (fs *FileSystemDataSource) getCachedDocument(filePath string) *core.Document {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	entry, exists := fs.cache[filePath]
	if !exists || time.Now().After(entry.expireTime) {
		return nil
	}

	// Check if file has been modified since caching
	if info, err := os.Stat(filePath); err == nil {
		if !info.ModTime().Equal(entry.modTime) {
			return nil
		}
	}

	return entry.document
}

// cacheDocument caches a document
func (fs *FileSystemDataSource) cacheDocument(filePath string, doc *core.Document) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	entry := &cacheEntry{
		document:   doc,
		modTime:    doc.Metadata.ModifiedAt,
		expireTime: time.Now().Add(fs.config.CacheTTL),
	}

	fs.cache[filePath] = entry

	// Simple cache size management - remove oldest entries if cache is too large
	maxCacheSize := 1000 // Configurable
	if len(fs.cache) > maxCacheSize {
		// Remove oldest entry (simple LRU-like behavior)
		var oldestKey string
		var oldestTime time.Time

		for key, entry := range fs.cache {
			if oldestKey == "" || entry.expireTime.Before(oldestTime) {
				oldestKey = key
				oldestTime = entry.expireTime
			}
		}

		if oldestKey != "" {
			delete(fs.cache, oldestKey)
		}
	}
}

// Helper methods for document processing

// getFileType determines the file type based on extension
func (fs *FileSystemDataSource) getFileType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	fileTypes := map[string]string{
		".go":         "go",
		".rs":         "rust",
		".js":         "javascript",
		".ts":         "typescript",
		".jsx":        "react",
		".tsx":        "react-typescript",
		".py":         "python",
		".java":       "java",
		".cpp":        "cpp",
		".c":          "c",
		".h":          "c",
		".hpp":        "cpp",
		".cs":         "csharp",
		".php":        "php",
		".rb":         "ruby",
		".swift":      "swift",
		".kt":         "kotlin",
		".scala":      "scala",
		".md":         "markdown",
		".txt":        "text",
		".json":       "json",
		".yaml":       "yaml",
		".yml":        "yaml",
		".xml":        "xml",
		".html":       "html",
		".htm":        "html",
		".css":        "css",
		".scss":       "scss",
		".less":       "less",
		".sql":        "sql",
		".sh":         "shell",
		".bash":       "shell",
		".zsh":        "shell",
		".fish":       "shell",
		".ps1":        "powershell",
		".bat":        "batch",
		".dockerfile": "dockerfile",
		".ini":        "config",
		".cfg":        "config",
		".conf":       "config",
		".toml":       "config",
		".pdf":        "pdf",
		".doc":        "word",
		".docx":       "word",
	}

	if fileType, exists := fileTypes[ext]; exists {
		return fileType
	}

	return "unknown"
}

// estimateWordCount estimates the number of words in content
func (fs *FileSystemDataSource) estimateWordCount(content string) int {
	// Simple word count estimation
	words := strings.Fields(content)
	return len(words)
}

// extractTitle extracts a title from the file path and content
func (fs *FileSystemDataSource) extractTitle(filePath, content string) string {
	// Use filename as default title
	title := filepath.Base(filePath)

	// Remove extension
	if ext := filepath.Ext(title); ext != "" {
		title = title[:len(title)-len(ext)]
	}

	// Replace common separators with spaces
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ReplaceAll(title, "-", " ")

	// Convert to title case
	title = strings.Title(title)

	// If content is available, try to extract a better title
	if fs.config.ExtractMetadata && len(content) > 0 {
		// For markdown files, look for first # header
		if strings.HasSuffix(strings.ToLower(filePath), ".md") {
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "# ") {
					return strings.TrimSpace(line[2:])
				}
			}
		}

		// For code files, look for comments
		if fs.isCodeFile(filePath) {
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "// ") || strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "/* ") {
					return strings.TrimSpace(line[3:])
				}
			}
		}
	}

	return title
}

// extractMetadata extracts additional metadata from file
func (fs *FileSystemDataSource) extractMetadata(filePath, content string, metadata *core.DocumentMetadata) {
	// Add file hash
	metadata.Custom = make(map[string]interface{})
	metadata.Custom["file_hash"] = fs.calculateFileHash(filePath)

	// Extract file-specific metadata
	switch fs.getFileType(filePath) {
	case "go":
		fs.extractGoMetadata(filePath, content, metadata)
	case "python":
		fs.extractPythonMetadata(filePath, content, metadata)
	case "markdown":
		fs.extractMarkdownMetadata(filePath, content, metadata)
	}
}

// extractTags extracts tags from file
func (fs *FileSystemDataSource) extractTags(filePath, content string) []string {
	var tags []string

	// Add file type as tag
	tags = append(tags, fs.getFileType(filePath))

	// Add directory names as tags
	pathParts := strings.Split(filepath.Dir(filePath), string(filepath.Separator))
	for _, part := range pathParts {
		if part != "" && part != "." && part != ".." {
			tags = append(tags, strings.ToLower(part))
		}
	}

	// Extract programming language-specific tags
	if fs.isCodeFile(filePath) {
		tags = append(tags, "code", "source")

		// Look for test files
		if strings.Contains(strings.ToLower(filepath.Base(filePath)), "test") {
			tags = append(tags, "test")
		}

		// Look for configuration files
		if strings.Contains(strings.ToLower(filepath.Base(filePath)), "config") ||
			strings.Contains(strings.ToLower(filepath.Base(filePath)), "settings") {
			tags = append(tags, "config")
		}
	}

	// Extract common keywords from content
	if len(content) > 0 {
		lowerContent := strings.ToLower(content)
		commonKeywords := []string{
			"api", "database", "frontend", "backend", "server", "client",
			"authentication", "authorization", "security", "logging",
			"testing", "documentation", "example", "tutorial",
		}

		for _, keyword := range commonKeywords {
			if strings.Contains(lowerContent, keyword) {
				tags = append(tags, keyword)
			}
		}
	}

	// Remove duplicates
	uniqueTags := make(map[string]bool)
	var result []string
	for _, tag := range tags {
		if !uniqueTags[tag] {
			uniqueTags[tag] = true
			result = append(result, tag)
		}
	}

	return result
}

// detectLanguage detects the language of the content
func (fs *FileSystemDataSource) detectLanguage(filePath, content string) string {
	// Use file extension as primary indicator
	ext := strings.ToLower(filepath.Ext(filePath))

	languageMap := map[string]string{
		".go":    "go",
		".rs":    "rust",
		".js":    "javascript",
		".ts":    "typescript",
		".jsx":   "javascript",
		".tsx":   "typescript",
		".py":    "python",
		".java":  "java",
		".cpp":   "cpp",
		".c":     "c",
		".h":     "c",
		".hpp":   "cpp",
		".cs":    "csharp",
		".php":   "php",
		".rb":    "ruby",
		".swift": "swift",
		".kt":    "kotlin",
		".scala": "scala",
		".md":    "en", // English by default for markdown
		".txt":   "en",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	// Simple content-based language detection for text files
	if len(content) > 100 {
		// Check for Chinese characters
		if containsChinese(content) {
			return "zh"
		}
		// Check for other language indicators
		if containsArabic(content) {
			return "ar"
		}
		if containsJapanese(content) {
			return "ja"
		}
	}

	return "en" // Default to English
}

// Language detection helpers
func containsChinese(content string) bool {
	for _, r := range content {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}
	return false
}

func containsArabic(content string) bool {
	for _, r := range content {
		if r >= 0x0600 && r <= 0x06ff {
			return true
		}
	}
	return false
}

func containsJapanese(content string) bool {
	for _, r := range content {
		if (r >= 0x3040 && r <= 0x309f) || // Hiragana
			(r >= 0x30a0 && r <= 0x30ff) || // Katakana
			(r >= 0x4e00 && r <= 0x9fff) { // Kanji
			return true
		}
	}
	return false
}

// isCodeFile checks if file is a code file
func (fs *FileSystemDataSource) isCodeFile(filePath string) bool {
	fileType := fs.getFileType(filePath)
	codeTypes := []string{
		"go", "rust", "javascript", "typescript", "react", "react-typescript",
		"python", "java", "cpp", "c", "csharp", "php", "ruby", "swift",
		"kotlin", "scala", "shell", "powershell", "batch", "dockerfile", "sql",
	}

	for _, codeType := range codeTypes {
		if fileType == codeType {
			return true
		}
	}

	return false
}

// calculateFileHash calculates SHA256 hash of file
func (fs *FileSystemDataSource) calculateFileHash(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash)
}

// extractGoMetadata extracts Go-specific metadata
func (fs *FileSystemDataSource) extractGoMetadata(filePath, content string, metadata *core.DocumentMetadata) {
	// Extract package name
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			metadata.Custom["package"] = strings.TrimSpace(line[8:])
			break
		}
	}
}

// extractPythonMetadata extracts Python-specific metadata
func (fs *FileSystemDataSource) extractPythonMetadata(filePath, content string, metadata *core.DocumentMetadata) {
	// Extract module docstring
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, `"""`) || strings.HasPrefix(line, "'''") {
			if i+1 < len(lines) {
				metadata.Custom["module_docstring"] = strings.TrimSpace(lines[i+1])
			}
			break
		}
	}
}

// extractMarkdownMetadata extracts Markdown-specific metadata
func (fs *FileSystemDataSource) extractMarkdownMetadata(filePath, content string, metadata *core.DocumentMetadata) {
	lines := strings.Split(content, "\n")

	// Count headers
	headers := 0
	codeBlocks := 0
	links := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			headers++
		}
		if strings.HasPrefix(line, "```") {
			codeBlocks++
		}
		if strings.Contains(line, "[") && strings.Contains(line, "](") {
			links++
		}
	}

	metadata.Custom["headers"] = headers
	metadata.Custom["code_blocks"] = codeBlocks
	metadata.Custom["links"] = links
}

// validateFileSystemConfig validates the filesystem configuration
func validateFileSystemConfig(config *FileSystemConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.RootPath == "" {
		return fmt.Errorf("root_path is required")
	}

	if config.MaxFileSize < 0 {
		return fmt.Errorf("max_file_size cannot be negative")
	}

	if config.MaxWorkers <= 0 {
		return fmt.Errorf("max_workers must be positive")
	}

	if config.BatchSize <= 0 {
		return fmt.Errorf("batch_size must be positive")
	}

	return nil
}

// configToMap converts FileSystemConfig to map for BaseDataSource
func configToMap(config *FileSystemConfig) map[string]interface{} {
	if config == nil {
		return nil
	}

	return map[string]interface{}{
		"root_path":        config.RootPath,
		"recursive":        config.Recursive,
		"include_patterns": config.IncludePatterns,
		"exclude_patterns": config.ExcludePatterns,
		"max_file_size":    config.MaxFileSize,
		"min_file_size":    config.MinFileSize,
		"include_types":    config.IncludeTypes,
		"exclude_types":    config.ExcludeTypes,
		"follow_symlinks":  config.FollowSymlinks,
		"ignore_hidden":    config.IgnoreHidden,
		"extract_metadata": config.ExtractMetadata,
		"detect_language":  config.DetectLanguage,
		"max_workers":      config.MaxWorkers,
		"batch_size":       config.BatchSize,
		"enable_cache":     config.EnableCache,
		"cache_dir":        config.CacheDir,
		"cache_ttl":        config.CacheTTL,
	}
}

// FileSystemDataSourceFactory implements DataSourceFactory for filesystem
type FileSystemDataSourceFactory struct{}

// NewFileSystemDataSourceFactory creates a new filesystem data source factory
func NewFileSystemDataSourceFactory() *FileSystemDataSourceFactory {
	return &FileSystemDataSourceFactory{}
}

// CreateDataSource implements DataSourceFactory interface
func (f *FileSystemDataSourceFactory) CreateDataSource(config map[string]interface{}) (core.DataSource, error) {
	fileConfig := &FileSystemConfig{}

	// Parse configuration
	if rootPath, ok := config["root_path"].(string); ok {
		fileConfig.RootPath = rootPath
	}
	if recursive, ok := config["recursive"].(bool); ok {
		fileConfig.Recursive = recursive
	}
	if includePatterns, ok := config["include_patterns"].([]string); ok {
		fileConfig.IncludePatterns = includePatterns
	}
	if excludePatterns, ok := config["exclude_patterns"].([]string); ok {
		fileConfig.ExcludePatterns = excludePatterns
	}
	if maxFileSize, ok := config["max_file_size"].(int64); ok {
		fileConfig.MaxFileSize = maxFileSize
	}
	if minFileSize, ok := config["min_file_size"].(int64); ok {
		fileConfig.MinFileSize = minFileSize
	}
	if includeTypes, ok := config["include_types"].([]string); ok {
		fileConfig.IncludeTypes = includeTypes
	}
	if excludeTypes, ok := config["exclude_types"].([]string); ok {
		fileConfig.ExcludeTypes = excludeTypes
	}
	if followSymlinks, ok := config["follow_symlinks"].(bool); ok {
		fileConfig.FollowSymlinks = followSymlinks
	}
	if ignoreHidden, ok := config["ignore_hidden"].(bool); ok {
		fileConfig.IgnoreHidden = ignoreHidden
	}
	if extractMetadata, ok := config["extract_metadata"].(bool); ok {
		fileConfig.ExtractMetadata = extractMetadata
	}
	if detectLanguage, ok := config["detect_language"].(bool); ok {
		fileConfig.DetectLanguage = detectLanguage
	}
	if maxWorkers, ok := config["max_workers"].(int); ok {
		fileConfig.MaxWorkers = maxWorkers
	}
	if batchSize, ok := config["batch_size"].(int); ok {
		fileConfig.BatchSize = batchSize
	}
	if enableCache, ok := config["enable_cache"].(bool); ok {
		fileConfig.EnableCache = enableCache
	}
	if cacheDir, ok := config["cache_dir"].(string); ok {
		fileConfig.CacheDir = cacheDir
	}
	if cacheTTL, ok := config["cache_ttl"].(time.Duration); ok {
		fileConfig.CacheTTL = cacheTTL
	}

	// Generate ID if not provided
	id, _ := config["id"].(string)
	if id == "" {
		id = fmt.Sprintf("fs_%d", time.Now().Unix())
	}

	return NewFileSystemDataSource(id, fileConfig)
}

// GetSupportedTypes implements DataSourceFactory interface
func (f *FileSystemDataSourceFactory) GetSupportedTypes() []string {
	return []string{"filesystem", "fs", "local", "file"}
}

// ValidateConfig implements DataSourceFactory interface
func (f *FileSystemDataSourceFactory) ValidateConfig(config map[string]interface{}) error {
	fileConfig := &FileSystemConfig{}

	if rootPath, ok := config["root_path"].(string); ok {
		fileConfig.RootPath = rootPath
	}

	return validateFileSystemConfig(fileConfig)
}
