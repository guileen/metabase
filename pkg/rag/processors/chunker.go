package processors

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/guileen/metabase/pkg/rag/core"
	"github.com/guileen/metabase/pkg/rag/embedding"
)

// FixedSizeChunkingStrategy implements fixed-size chunking
type FixedSizeChunkingStrategy struct {
	maxChunkSize    int
	minChunkSize    int
	overlapSize     int
	separator       string
	stripWhitespace bool
}

// NewFixedSizeChunkingStrategy creates a new fixed-size chunking strategy
func NewFixedSizeChunkingStrategy(maxChunkSize, minChunkSize, overlapSize int) *FixedSizeChunkingStrategy {
	return &FixedSizeChunkingStrategy{
		maxChunkSize:    maxChunkSize,
		minChunkSize:    minChunkSize,
		overlapSize:     overlapSize,
		separator:       "\n",
		stripWhitespace: true,
	}
}

// Chunk implements ChunkingStrategy interface
func (s *FixedSizeChunkingStrategy) Chunk(ctx context.Context, doc core.Document) ([]core.DocumentChunk, error) {
	if s.maxChunkSize <= 0 {
		return nil, fmt.Errorf("max_chunk_size must be positive")
	}

	content := doc.Content
	if s.stripWhitespace {
		content = strings.TrimSpace(content)
	}

	if len(content) == 0 {
		return nil, fmt.Errorf("document content is empty")
	}

	// If content is smaller than max chunk size, return single chunk
	if len(content) <= s.maxChunkSize {
		return []core.DocumentChunk{{
			ID:         fmt.Sprintf("%s_chunk_0", doc.ID),
			DocumentID: doc.ID,
			Content:    content,
			ChunkIndex: 0,
			StartPos:   0,
			EndPos:     len(content),
			StartLine:  1,
			EndLine:    strings.Count(content, "\n") + 1,
			ChunkType:  "fixed",
			ChunkSize:  len(content),
			CreatedAt:  time.Now(),
		}}, nil
	}

	var chunks []core.DocumentChunk
	position := 0
	chunkIndex := 0

	for position < len(content) {
		// Calculate chunk boundaries
		end := position + s.maxChunkSize
		if end > len(content) {
			end = len(content)
		}

		chunkContent := content[position:end]

		// Try to split at word boundaries
		if end < len(content) && !unicode.IsSpace(rune(content[end])) {
			lastSpace := strings.LastIndex(chunkContent, " ")
			if lastSpace > s.minChunkSize {
				chunkContent = chunkContent[:lastSpace]
				end = position + len(chunkContent)
			}
		}

		// Skip if chunk is too small
		if len(chunkContent) < s.minChunkSize {
			if len(chunks) > 0 {
				// Append to previous chunk
				chunks[len(chunks)-1].Content += chunkContent
				chunks[len(chunks)-1].EndPos = end
				chunks[len(chunks)-1].ChunkSize = len(chunks[len(chunks)-1].Content)
			}
			position += len(chunkContent)
			continue
		}

		// Calculate line numbers
		startLine := strings.Count(content[:position], "\n") + 1
		endLine := strings.Count(content[:end], "\n") + 1

		chunk := core.DocumentChunk{
			ID:         fmt.Sprintf("%s_chunk_%d", doc.ID, chunkIndex),
			DocumentID: doc.ID,
			Content:    strings.TrimSpace(chunkContent),
			ChunkIndex: chunkIndex,
			StartPos:   position,
			EndPos:     end,
			StartLine:  startLine,
			EndLine:    endLine,
			ChunkType:  "fixed",
			ChunkSize:  len(chunkContent),
			CreatedAt:  time.Now(),
		}

		chunks = append(chunks, chunk)

		position = end - s.overlapSize
		if position < 0 {
			position = 0
		}
		chunkIndex++
	}

	return chunks, nil
}

// GetName implements ChunkingStrategy interface
func (s *FixedSizeChunkingStrategy) GetName() string {
	return "fixed_size"
}

// GetDescription implements ChunkingStrategy interface
func (s *FixedSizeChunkingStrategy) GetDescription() string {
	return "Fixed-size chunking with configurable overlap"
}

// SetParameters implements ChunkingStrategy interface
func (s *FixedSizeChunkingStrategy) SetParameters(params map[string]interface{}) error {
	if maxChunkSize, ok := params["max_chunk_size"].(int); ok {
		s.maxChunkSize = maxChunkSize
	}
	if minChunkSize, ok := params["min_chunk_size"].(int); ok {
		s.minChunkSize = minChunkSize
	}
	if overlapSize, ok := params["overlap_size"].(int); ok {
		s.overlapSize = overlapSize
	}
	if separator, ok := params["separator"].(string); ok {
		s.separator = separator
	}
	if stripWhitespace, ok := params["strip_whitespace"].(bool); ok {
		s.stripWhitespace = stripWhitespace
	}
	return nil
}

// GetParameters implements ChunkingStrategy interface
func (s *FixedSizeChunkingStrategy) GetParameters() map[string]interface{} {
	return map[string]interface{}{
		"max_chunk_size":   s.maxChunkSize,
		"min_chunk_size":   s.minChunkSize,
		"overlap_size":     s.overlapSize,
		"separator":        s.separator,
		"strip_whitespace": s.stripWhitespace,
	}
}

// ParagraphChunkingStrategy implements paragraph-based chunking
type ParagraphChunkingStrategy struct {
	maxChunkSize  int
	maxParagraphs int
	mergeShort    bool
	minChunkSize  int
	overlapSize   int
}

// NewParagraphChunkingStrategy creates a new paragraph chunking strategy
func NewParagraphChunkingStrategy(maxChunkSize, maxParagraphs, minChunkSize, overlapSize int) *ParagraphChunkingStrategy {
	return &ParagraphChunkingStrategy{
		maxChunkSize:  maxChunkSize,
		maxParagraphs: maxParagraphs,
		mergeShort:    true,
		minChunkSize:  minChunkSize,
		overlapSize:   overlapSize,
	}
}

// Chunk implements ChunkingStrategy interface
func (s *ParagraphChunkingStrategy) Chunk(ctx context.Context, doc core.Document) ([]core.DocumentChunk, error) {
	content := doc.Content
	if len(content) == 0 {
		return nil, fmt.Errorf("document content is empty")
	}

	// Split into paragraphs
	paragraphs := s.splitIntoParagraphs(content)
	if len(paragraphs) == 0 {
		return nil, fmt.Errorf("no paragraphs found in document")
	}

	var chunks []core.DocumentChunk
	currentChunk := ""
	currentChunkIndex := 0
	position := 0

	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if len(paragraph) == 0 {
			continue
		}

		// Check if adding this paragraph would exceed max chunk size
		if len(currentChunk)+len(paragraph)+2 > s.maxChunkSize && len(currentChunk) > 0 {
			// Create chunk from current content
			if len(currentChunk) >= s.minChunkSize || !s.mergeShort {
				chunk := s.createChunk(doc, currentChunk, currentChunkIndex, position)
				chunks = append(chunks, chunk)
				position += len(currentChunk)
				currentChunkIndex++
			}

			// Start new chunk with overlap if configured
			if s.overlapSize > 0 && len(chunks) > 0 {
				overlap := s.getOverlapContent(chunks[len(chunks)-1].Content, s.overlapSize)
				currentChunk = overlap + "\n\n" + paragraph
			} else {
				currentChunk = paragraph
			}
		} else {
			// Add paragraph to current chunk
			if len(currentChunk) > 0 {
				currentChunk += "\n\n" + paragraph
			} else {
				currentChunk = paragraph
			}
		}

		// Check max paragraph limit
		paragraphCount := len(strings.Split(currentChunk, "\n\n"))
		if s.maxParagraphs > 0 && paragraphCount >= s.maxParagraphs {
			chunk := s.createChunk(doc, currentChunk, currentChunkIndex, position)
			chunks = append(chunks, chunk)
			position += len(currentChunk)
			currentChunkIndex++
			currentChunk = ""
		}
	}

	// Add remaining content
	if len(currentChunk) >= s.minChunkSize || !s.mergeShort {
		chunk := s.createChunk(doc, currentChunk, currentChunkIndex, position)
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// splitIntoParagraphs splits content into paragraphs
func (s *ParagraphChunkingStrategy) splitIntoParagraphs(content string) []string {
	// Normalize line endings
	content = regexp.MustCompile(`\r\n?`).ReplaceAllString(content, "\n")

	// Split by double newlines (paragraphs)
	paragraphs := regexp.MustCompile(`\n\s*\n`).Split(content, -1)

	// Clean up paragraphs
	var cleanParagraphs []string
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if len(paragraph) > 0 {
			cleanParagraphs = append(cleanParagraphs, paragraph)
		}
	}

	return cleanParagraphs
}

// getOverlapContent gets overlap content from previous chunk
func (s *ParagraphChunkingStrategy) getOverlapContent(content string, overlapSize int) string {
	if len(content) <= overlapSize {
		return content
	}

	// Try to split at paragraph boundary
	paragraphs := strings.Split(content, "\n\n")
	var overlap strings.Builder
	size := 0

	// Start from the end and work backwards
	for i := len(paragraphs) - 1; i >= 0; i-- {
		paragraph := paragraphs[i]
		paragraphSize := len(paragraph)
		if size+paragraphSize > overlapSize {
			break
		}
		if overlap.Len() > 0 {
			overlap.WriteString("\n\n")
		}
		overlap.WriteString(paragraph)
		size += paragraphSize + 2
	}

	result := overlap.String()
	paragraphs = strings.Split(result, "\n\n")

	// Reverse to maintain original order
	for i, j := 0, len(paragraphs)-1; i < j; i, j = i+1, j-1 {
		paragraphs[i], paragraphs[j] = paragraphs[j], paragraphs[i]
	}

	return strings.Join(paragraphs, "\n\n")
}

// createChunk creates a document chunk
func (s *ParagraphChunkingStrategy) createChunk(doc core.Document, content string, index int, position int) core.DocumentChunk {
	// Calculate line numbers
	startLine := strings.Count(doc.Content[:position], "\n") + 1
	endLine := strings.Count(doc.Content[:position+len(content)], "\n") + 1

	return core.DocumentChunk{
		ID:         fmt.Sprintf("%s_chunk_%d", doc.ID, index),
		DocumentID: doc.ID,
		Content:    content,
		ChunkIndex: index,
		StartPos:   position,
		EndPos:     position + len(content),
		StartLine:  startLine,
		EndLine:    endLine,
		ChunkType:  "paragraph",
		ChunkSize:  len(content),
		CreatedAt:  time.Now(),
	}
}

// GetName implements ChunkingStrategy interface
func (s *ParagraphChunkingStrategy) GetName() string {
	return "paragraph"
}

// GetDescription implements ChunkingStrategy interface
func (s *ParagraphChunkingStrategy) GetDescription() string {
	return "Paragraph-based chunking with configurable size limits"
}

// SetParameters implements ChunkingStrategy interface
func (s *ParagraphChunkingStrategy) SetParameters(params map[string]interface{}) error {
	if maxChunkSize, ok := params["max_chunk_size"].(int); ok {
		s.maxChunkSize = maxChunkSize
	}
	if maxParagraphs, ok := params["max_paragraphs"].(int); ok {
		s.maxParagraphs = maxParagraphs
	}
	if mergeShort, ok := params["merge_short"].(bool); ok {
		s.mergeShort = mergeShort
	}
	if minChunkSize, ok := params["min_chunk_size"].(int); ok {
		s.minChunkSize = minChunkSize
	}
	if overlapSize, ok := params["overlap_size"].(int); ok {
		s.overlapSize = overlapSize
	}
	return nil
}

// GetParameters implements ChunkingStrategy interface
func (s *ParagraphChunkingStrategy) GetParameters() map[string]interface{} {
	return map[string]interface{}{
		"max_chunk_size": s.maxChunkSize,
		"max_paragraphs": s.maxParagraphs,
		"merge_short":    s.mergeShort,
		"min_chunk_size": s.minChunkSize,
		"overlap_size":   s.overlapSize,
	}
}

// SemanticChunkingStrategy implements semantic chunking based on content similarity
type SemanticChunkingStrategy struct {
	maxChunkSize        int
	minChunkSize        int
	similarityThreshold float64
	embeddingGen        embedding.VectorGenerator
	overlapSize         int
	windowSize          int
}

// NewSemanticChunkingStrategy creates a new semantic chunking strategy
func NewSemanticChunkingStrategy(maxChunkSize, minChunkSize int, similarityThreshold float64, embeddingGen embedding.VectorGenerator) *SemanticChunkingStrategy {
	return &SemanticChunkingStrategy{
		maxChunkSize:        maxChunkSize,
		minChunkSize:        minChunkSize,
		similarityThreshold: similarityThreshold,
		embeddingGen:        embeddingGen,
		overlapSize:         100,
		windowSize:          3,
	}
}

// Chunk implements ChunkingStrategy interface
func (s *SemanticChunkingStrategy) Chunk(ctx context.Context, doc core.Document) ([]core.DocumentChunk, error) {
	if s.embeddingGen == nil {
		// Fallback to fixed-size chunking if no embedding generator
		fallbackStrategy := NewFixedSizeChunkingStrategy(s.maxChunkSize, s.minChunkSize, s.overlapSize)
		return fallbackStrategy.Chunk(ctx, doc)
	}

	content := doc.Content
	if len(content) == 0 {
		return nil, fmt.Errorf("document content is empty")
	}

	// Split into sentences
	sentences := s.splitIntoSentences(content)
	if len(sentences) == 0 {
		return nil, fmt.Errorf("no sentences found in document")
	}

	// Generate embeddings for sentences
	embeddings, err := s.generateSentenceEmbeddings(ctx, sentences)
	if err != nil {
		// Fallback to fixed-size chunking
		fallbackStrategy := NewFixedSizeChunkingStrategy(s.maxChunkSize, s.minChunkSize, s.overlapSize)
		return fallbackStrategy.Chunk(ctx, doc)
	}

	// Group sentences into chunks based on semantic similarity
	chunks := s.groupSentencesBySimilarity(doc, sentences, embeddings)

	return chunks, nil
}

// splitIntoSentences splits content into sentences
func (s *SemanticChunkingStrategy) splitIntoSentences(content string) []string {
	// Simple sentence splitting - could be enhanced with NLP libraries
	sentenceEndings := regexp.MustCompile(`[.!?]+\s+`)
	sentences := sentenceEndings.Split(content, -1)

	var cleanSentences []string
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 0 {
			cleanSentences = append(cleanSentences, sentence)
		}
	}

	return cleanSentences
}

// generateSentenceEmbeddings generates embeddings for sentences
func (s *SemanticChunkingStrategy) generateSentenceEmbeddings(ctx context.Context, sentences []string) ([][]float64, error) {
	if s.embeddingGen == nil {
		return nil, fmt.Errorf("no embedding generator available")
	}

	return s.embeddingGen.Embed(ctx, sentences)
}

// groupSentencesBySimilarity groups sentences into chunks based on similarity
func (s *SemanticChunkingStrategy) groupSentencesBySimilarity(doc core.Document, sentences []string, embeddings [][]float64) []core.DocumentChunk {
	var chunks []core.DocumentChunk
	chunkIndex := 0
	position := 0

	for i := 0; i < len(sentences); {
		currentChunk := strings.Builder{}
		currentChunk.WriteString(sentences[i])
		chunkSize := len(sentences[i])
		lastSentenceIndex := i

		// Add subsequent sentences if they're similar enough
		for j := i + 1; j < len(sentences); j++ {
			if chunkSize+len(sentences[j]) > s.maxChunkSize {
				break
			}

			// Calculate similarity between adjacent sentences
			similarity := s.calculateCosineSimilarity(embeddings[j-1], embeddings[j])
			if similarity < s.similarityThreshold {
				break
			}

			currentChunk.WriteString(" " + sentences[j])
			chunkSize += len(sentences[j]) + 1
			lastSentenceIndex = j
		}

		chunkContent := currentChunk.String()
		if len(chunkContent) >= s.minChunkSize {
			// Calculate line numbers
			startLine := strings.Count(doc.Content[:position], "\n") + 1
			endLine := strings.Count(doc.Content[:position+len(chunkContent)], "\n") + 1

			chunk := core.DocumentChunk{
				ID:         fmt.Sprintf("%s_chunk_%d", doc.ID, chunkIndex),
				DocumentID: doc.ID,
				Content:    chunkContent,
				ChunkIndex: chunkIndex,
				StartPos:   position,
				EndPos:     position + len(chunkContent),
				StartLine:  startLine,
				EndLine:    endLine,
				ChunkType:  "semantic",
				ChunkSize:  len(chunkContent),
				CreatedAt:  time.Now(),
			}

			chunks = append(chunks, chunk)
			position += len(chunkContent)
			chunkIndex++
		}

		// Move to next sentence (with some overlap for continuity)
		i = lastSentenceIndex + 1
		if s.overlapSize > 0 && i < len(sentences) {
			// Add some overlap by stepping back
			overlapSentences := s.getOverlapSentences(sentences[:lastSentenceIndex], s.overlapSize)
			if len(overlapSentences) > 0 {
				// Create new chunk starting with overlap content
				i = len(overlapSentences)
			}
		}
	}

	return chunks
}

// calculateCosineSimilarity calculates cosine similarity between two vectors
func (s *SemanticChunkingStrategy) calculateCosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (normA * normB)
}

// getOverlapSentences gets sentences for overlap
func (s *SemanticChunkingStrategy) getOverlapSentences(sentences []string, overlapSize int) []string {
	if overlapSize <= 0 || len(sentences) == 0 {
		return nil
	}

	var result []string
	size := 0

	// Start from the end and work backwards
	for i := len(sentences) - 1; i >= 0; i-- {
		sentence := sentences[i]
		if size+len(sentence) > overlapSize {
			break
		}
		result = append([]string{sentence}, result...)
		size += len(sentence)
	}

	return result
}

// GetName implements ChunkingStrategy interface
func (s *SemanticChunkingStrategy) GetName() string {
	return "semantic"
}

// GetDescription implements ChunkingStrategy interface
func (s *SemanticChunkingStrategy) GetDescription() string {
	return "Semantic chunking based on content similarity using embeddings"
}

// SetParameters implements ChunkingStrategy interface
func (s *SemanticChunkingStrategy) SetParameters(params map[string]interface{}) error {
	if maxChunkSize, ok := params["max_chunk_size"].(int); ok {
		s.maxChunkSize = maxChunkSize
	}
	if minChunkSize, ok := params["min_chunk_size"].(int); ok {
		s.minChunkSize = minChunkSize
	}
	if similarityThreshold, ok := params["similarity_threshold"].(float64); ok {
		s.similarityThreshold = similarityThreshold
	}
	if overlapSize, ok := params["overlap_size"].(int); ok {
		s.overlapSize = overlapSize
	}
	if windowSize, ok := params["window_size"].(int); ok {
		s.windowSize = windowSize
	}
	return nil
}

// GetParameters implements ChunkingStrategy interface
func (s *SemanticChunkingStrategy) GetParameters() map[string]interface{} {
	return map[string]interface{}{
		"max_chunk_size":       s.maxChunkSize,
		"min_chunk_size":       s.minChunkSize,
		"similarity_threshold": s.similarityThreshold,
		"overlap_size":         s.overlapSize,
		"window_size":          s.windowSize,
	}
}

// CodeChunkingStrategy implements code-aware chunking
type CodeChunkingStrategy struct {
	maxChunkSize      int
	minChunkSize      int
	preserveFunctions bool
	preserveClasses   bool
	overlapSize       int
	language          string
}

// NewCodeChunkingStrategy creates a new code chunking strategy
func NewCodeChunkingStrategy(maxChunkSize, minChunkSize, overlapSize int) *CodeChunkingStrategy {
	return &CodeChunkingStrategy{
		maxChunkSize:      maxChunkSize,
		minChunkSize:      minChunkSize,
		preserveFunctions: true,
		preserveClasses:   true,
		overlapSize:       overlapSize,
	}
}

// Chunk implements ChunkingStrategy interface
func (s *CodeChunkingStrategy) Chunk(ctx context.Context, doc core.Document) ([]core.DocumentChunk, error) {
	content := doc.Content
	if len(content) == 0 {
		return nil, fmt.Errorf("document content is empty")
	}

	// Detect programming language if not specified
	if s.language == "" {
		s.language = s.detectLanguage(doc)
	}

	// Use language-specific chunking
	switch s.language {
	case "go":
		return s.chunkGoCode(doc)
	case "python":
		return s.chunkPythonCode(doc)
	case "javascript", "typescript":
		return s.chunkJavaScriptCode(doc)
	default:
		// Fallback to fixed-size chunking with line preservation
		return s.chunkGenericCode(doc)
	}
}

// detectLanguage detects the programming language from document metadata
func (s *CodeChunkingStrategy) detectLanguage(doc core.Document) string {
	if doc.Language != "" && doc.Language != "unknown" {
		return doc.Language
	}

	// Detect from file extension
	ext := strings.ToLower(doc.Metadata.Extension)
	languageMap := map[string]string{
		".go":   "go",
		".py":   "python",
		".js":   "javascript",
		".ts":   "typescript",
		".jsx":  "javascript",
		".tsx":  "typescript",
		".java": "java",
		".cpp":  "cpp",
		".c":    "c",
		".cs":   "csharp",
		".rs":   "rust",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	return "unknown"
}

// chunkGoCode chunks Go code preserving functions and types
func (s *CodeChunkingStrategy) chunkGoCode(doc core.Document) ([]core.DocumentChunk, error) {
	lines := strings.Split(doc.Content, "\n")
	var chunks []core.DocumentChunk

	currentChunk := strings.Builder{}
	chunkStart := 0
	chunkIndex := 0
	lineNumber := 0

	for _, line := range lines {
		lineNumber++
		lineContent := strings.TrimSpace(line)

		// Check if line starts a function, type, or struct
		isNewBlock := false
		if strings.HasPrefix(lineContent, "func ") ||
			strings.HasPrefix(lineContent, "type ") ||
			strings.HasPrefix(lineContent, "var ") ||
			strings.HasPrefix(lineContent, "const ") {
			isNewBlock = true
		}

		// Create chunk if block boundary and chunk is large enough
		if isNewBlock && currentChunk.Len() > s.minChunkSize {
			chunk := s.createCodeChunk(doc, currentChunk.String(), chunkIndex, chunkStart, lineNumber-1)
			chunks = append(chunks, chunk)
			chunkIndex++
			chunkStart = lineNumber - 1
			currentChunk.Reset()
		}

		currentChunk.WriteString(line + "\n")

		// Check max size
		if currentChunk.Len() > s.maxChunkSize {
			chunk := s.createCodeChunk(doc, currentChunk.String(), chunkIndex, chunkStart, lineNumber)
			chunks = append(chunks, chunk)
			chunkIndex++
			chunkStart = lineNumber
			currentChunk.Reset()
		}
	}

	// Add remaining content
	if currentChunk.Len() > 0 {
		chunk := s.createCodeChunk(doc, currentChunk.String(), chunkIndex, chunkStart, lineNumber)
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// chunkPythonCode chunks Python code preserving functions and classes
func (s *CodeChunkingStrategy) chunkPythonCode(doc core.Document) ([]core.DocumentChunk, error) {
	lines := strings.Split(doc.Content, "\n")
	var chunks []core.DocumentChunk

	currentChunk := strings.Builder{}
	chunkStart := 0
	chunkIndex := 0
	lineNumber := 0

	for _, line := range lines {
		lineNumber++
		lineContent := strings.TrimSpace(line)

		// Check if line starts a function, class, or method
		isNewBlock := false
		if strings.HasPrefix(lineContent, "def ") ||
			strings.HasPrefix(lineContent, "class ") ||
			strings.HasPrefix(lineContent, "@") { // Decorator
			isNewBlock = true
		}

		// Create chunk if block boundary and chunk is large enough
		if isNewBlock && currentChunk.Len() > s.minChunkSize {
			chunk := s.createCodeChunk(doc, currentChunk.String(), chunkIndex, chunkStart, lineNumber-1)
			chunks = append(chunks, chunk)
			chunkIndex++
			chunkStart = lineNumber - 1
			currentChunk.Reset()
		}

		currentChunk.WriteString(line + "\n")

		// Check max size
		if currentChunk.Len() > s.maxChunkSize {
			chunk := s.createCodeChunk(doc, currentChunk.String(), chunkIndex, chunkStart, lineNumber)
			chunks = append(chunks, chunk)
			chunkIndex++
			chunkStart = lineNumber
			currentChunk.Reset()
		}
	}

	// Add remaining content
	if currentChunk.Len() > 0 {
		chunk := s.createCodeChunk(doc, currentChunk.String(), chunkIndex, chunkStart, lineNumber)
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// chunkJavaScriptCode chunks JavaScript/TypeScript code
func (s *CodeChunkingStrategy) chunkJavaScriptCode(doc core.Document) ([]core.DocumentChunk, error) {
	lines := strings.Split(doc.Content, "\n")
	var chunks []core.DocumentChunk

	currentChunk := strings.Builder{}
	chunkStart := 0
	chunkIndex := 0
	lineNumber := 0

	for _, line := range lines {
		lineNumber++
		lineContent := strings.TrimSpace(line)

		// Check if line starts a function, class, or method
		isNewBlock := false
		if strings.HasPrefix(lineContent, "function ") ||
			strings.HasPrefix(lineContent, "class ") ||
			strings.HasPrefix(lineContent, "const ") ||
			strings.HasPrefix(lineContent, "let ") ||
			strings.HasPrefix(lineContent, "var ") ||
			strings.Contains(lineContent, "=>") { // Arrow function
			isNewBlock = true
		}

		// Create chunk if block boundary and chunk is large enough
		if isNewBlock && currentChunk.Len() > s.minChunkSize {
			chunk := s.createCodeChunk(doc, currentChunk.String(), chunkIndex, chunkStart, lineNumber-1)
			chunks = append(chunks, chunk)
			chunkIndex++
			chunkStart = lineNumber - 1
			currentChunk.Reset()
		}

		currentChunk.WriteString(line + "\n")

		// Check max size
		if currentChunk.Len() > s.maxChunkSize {
			chunk := s.createCodeChunk(doc, currentChunk.String(), chunkIndex, chunkStart, lineNumber)
			chunks = append(chunks, chunk)
			chunkIndex++
			chunkStart = lineNumber
			currentChunk.Reset()
		}
	}

	// Add remaining content
	if currentChunk.Len() > 0 {
		chunk := s.createCodeChunk(doc, currentChunk.String(), chunkIndex, chunkStart, lineNumber)
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// chunkGenericCode chunks generic code preserving line structure
func (s *CodeChunkingStrategy) chunkGenericCode(doc core.Document) ([]core.DocumentChunk, error) {
	lines := strings.Split(doc.Content, "\n")
	var chunks []core.DocumentChunk

	currentChunk := strings.Builder{}
	chunkStart := 0
	chunkIndex := 0
	lineNumber := 0

	for _, line := range lines {
		lineNumber++
		currentChunk.WriteString(line + "\n")

		// Check max size
		if currentChunk.Len() > s.maxChunkSize {
			chunk := s.createCodeChunk(doc, currentChunk.String(), chunkIndex, chunkStart, lineNumber)
			chunks = append(chunks, chunk)
			chunkIndex++
			chunkStart = lineNumber
			currentChunk.Reset()
		}
	}

	// Add remaining content
	if currentChunk.Len() > 0 {
		chunk := s.createCodeChunk(doc, currentChunk.String(), chunkIndex, chunkStart, lineNumber)
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// createCodeChunk creates a code document chunk
func (s *CodeChunkingStrategy) createCodeChunk(doc core.Document, content string, index int, startLine, endLine int) core.DocumentChunk {
	// Calculate character positions
	lines := strings.Split(doc.Content, "\n")
	startPos := 0
	endPos := 0

	for i := 0; i < startLine-1 && i < len(lines); i++ {
		startPos += len(lines[i]) + 1 // +1 for newline
	}

	for i := 0; i < endLine && i < len(lines); i++ {
		endPos += len(lines[i]) + 1 // +1 for newline
	}

	return core.DocumentChunk{
		ID:         fmt.Sprintf("%s_chunk_%d", doc.ID, index),
		DocumentID: doc.ID,
		Content:    content,
		ChunkIndex: index,
		StartPos:   startPos,
		EndPos:     endPos,
		StartLine:  startLine,
		EndLine:    endLine,
		ChunkType:  "code",
		ChunkSize:  len(content),
		CreatedAt:  time.Now(),
	}
}

// GetName implements ChunkingStrategy interface
func (s *CodeChunkingStrategy) GetName() string {
	return "code"
}

// GetDescription implements ChunkingStrategy interface
func (s *CodeChunkingStrategy) GetDescription() string {
	return "Code-aware chunking that preserves functions, classes, and other code structures"
}

// SetParameters implements ChunkingStrategy interface
func (s *CodeChunkingStrategy) SetParameters(params map[string]interface{}) error {
	if maxChunkSize, ok := params["max_chunk_size"].(int); ok {
		s.maxChunkSize = maxChunkSize
	}
	if minChunkSize, ok := params["min_chunk_size"].(int); ok {
		s.minChunkSize = minChunkSize
	}
	if preserveFunctions, ok := params["preserve_functions"].(bool); ok {
		s.preserveFunctions = preserveFunctions
	}
	if preserveClasses, ok := params["preserve_classes"].(bool); ok {
		s.preserveClasses = preserveClasses
	}
	if overlapSize, ok := params["overlap_size"].(int); ok {
		s.overlapSize = overlapSize
	}
	if language, ok := params["language"].(string); ok {
		s.language = language
	}
	return nil
}

// GetParameters implements ChunkingStrategy interface
func (s *CodeChunkingStrategy) GetParameters() map[string]interface{} {
	return map[string]interface{}{
		"max_chunk_size":     s.maxChunkSize,
		"min_chunk_size":     s.minChunkSize,
		"preserve_functions": s.preserveFunctions,
		"preserve_classes":   s.preserveClasses,
		"overlap_size":       s.overlapSize,
		"language":           s.language,
	}
}

// ChunkingStrategyRegistry manages available chunking strategies
type ChunkingStrategyRegistry struct {
	strategies map[string]core.ChunkingStrategy
}

// NewChunkingStrategyRegistry creates a new registry
func NewChunkingStrategyRegistry() *ChunkingStrategyRegistry {
	return &ChunkingStrategyRegistry{
		strategies: make(map[string]core.ChunkingStrategy),
	}
}

// RegisterStrategy registers a chunking strategy
func (r *ChunkingStrategyRegistry) RegisterStrategy(name string, strategy core.ChunkingStrategy) {
	r.strategies[name] = strategy
}

// GetStrategy returns a strategy by name
func (r *ChunkingStrategyRegistry) GetStrategy(name string) (core.ChunkingStrategy, error) {
	strategy, exists := r.strategies[name]
	if !exists {
		return nil, fmt.Errorf("unknown chunking strategy: %s", name)
	}
	return strategy, nil
}

// ListStrategies returns all available strategy names
func (r *ChunkingStrategyRegistry) ListStrategies() []string {
	names := make([]string, 0, len(r.strategies))
	for name := range r.strategies {
		names = append(names, name)
	}
	return names
}

// Default registry with common strategies
var defaultChunkingRegistry = NewChunkingStrategyRegistry()

func init() {
	// Register default strategies
	defaultChunkingRegistry.RegisterStrategy("fixed", NewFixedSizeChunkingStrategy(1000, 100, 200))
	defaultChunkingRegistry.RegisterStrategy("paragraph", NewParagraphChunkingStrategy(2000, 10, 100, 200))
	defaultChunkingRegistry.RegisterStrategy("semantic", NewSemanticChunkingStrategy(1500, 100, 0.7, nil))
	defaultChunkingRegistry.RegisterStrategy("code", NewCodeChunkingStrategy(1500, 50, 100))
}

// GetChunkingStrategy returns a strategy from the default registry
func GetChunkingStrategy(name string) (core.ChunkingStrategy, error) {
	return defaultChunkingRegistry.GetStrategy(name)
}

// ListChunkingStrategies returns all available strategy names from default registry
func ListChunkingStrategies() []string {
	return defaultChunkingRegistry.ListStrategies()
}

// RegisterChunkingStrategy registers a strategy with the default registry
func RegisterChunkingStrategy(name string, strategy core.ChunkingStrategy) {
	defaultChunkingRegistry.RegisterStrategy(name, strategy)
}
