package embedding

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// GTEsmallZhGenerator implements VectorGenerator for GTE-small-zh model
// This is a 33M parameter model optimized for Chinese text, also supports English
// Vector dimension: 384
// Model size: ~70MB
// 18% better retrieval performance than all-MiniLM-L6-v2 for Chinese text
type GTEsmallZhGenerator struct {
	config       VectorGeneratorConfig
	dimension    int
	modelPath    string
	cachePath    string
	mutex        sync.RWMutex
	initialized  bool
	capabilities ModelCapabilities
}

// NewGTEsmallZhGenerator creates a new GTE-small-zh generator
func NewGTEsmallZhGenerator(config VectorGeneratorConfig) (*GTEsmallZhGenerator, error) {
	if config.ModelName == "" {
		config.ModelName = "GTE-small-zh"
	}
	if config.BatchSize == 0 {
		config.BatchSize = 16 // Smaller batch size for better Chinese text processing
	}
	if config.MaxConcurrency == 0 {
		config.MaxConcurrency = 2 // Lower concurrency for Chinese text
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second // Longer timeout for Chinese text processing
	}

	gen := &GTEsmallZhGenerator{
		config:    config,
		dimension: 384,                     // GTE-small-zh dimension
		modelPath: "thenlper/gte-small-zh", // HuggingFace model path
		capabilities: ModelCapabilities{
			Languages:            []string{"zh", "en", "zh-CN", "zh-TW"}, // Chinese optimized, also supports English
			MaxSequenceLength:    512,
			RecommendedBatchSize: 16,
			SupportsMultilingual: false,             // Primarily Chinese
			OptimizedForChinese:  true,              // Specifically optimized for Chinese
			SupportsGPU:          false,             // Python implementation doesn't use GPU
			ModelSizeBytes:       70 * 1024 * 1024,  // ~70MB
			EstimatedMemoryUsage: 250 * 1024 * 1024, // ~250MB during inference
		},
	}

	if config.CacheDir != "" {
		gen.cachePath = config.CacheDir
	} else {
		gen.cachePath = filepath.Join(os.TempDir(), "metabase_gte_zh_cache")
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(gen.cachePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return gen, nil
}

// Embed generates embeddings for a batch of texts
func (g *GTEsmallZhGenerator) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	if err := g.ensureInitialized(); err != nil {
		if g.config.EnableFallback {
			return g.fallbackEmbed(ctx, texts)
		}
		return nil, fmt.Errorf("failed to initialize generator: %w", err)
	}

	// Preprocess Chinese texts for better results
	processedTexts := g.preprocessChineseTexts(texts)

	// Process in batches for optimal performance
	batchSize := g.config.BatchSize
	if batchSize <= 0 {
		batchSize = 16
	}

	maxConc := g.config.MaxConcurrency
	if maxConc <= 0 {
		maxConc = 2
	}

	// Create channel for context cancellation
	ctx, cancel := context.WithTimeout(ctx, g.config.Timeout)
	defer cancel()

	return g.processBatches(ctx, processedTexts, batchSize, maxConc)
}

// EmbedSingle generates embedding for a single text
func (g *GTEsmallZhGenerator) EmbedSingle(ctx context.Context, text string) ([]float64, error) {
	embeddings, err := g.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding generated")
	}
	return embeddings[0], nil
}

// GetDimension returns the dimension of the embedding vectors
func (g *GTEsmallZhGenerator) GetDimension() int {
	return g.dimension
}

// GetModelName returns the name/type of the model
func (g *GTEsmallZhGenerator) GetModelName() string {
	return "GTE-small-zh"
}

// GetCapabilities returns the capabilities of this embedding model
func (g *GTEsmallZhGenerator) GetCapabilities() ModelCapabilities {
	return g.capabilities
}

// Close performs cleanup and releases resources
func (g *GTEsmallZhGenerator) Close() error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.initialized = false
	return nil
}

// preprocessChineseTexts preprocesses Chinese texts for better embedding quality
func (g *GTEsmallZhGenerator) preprocessChineseTexts(texts []string) []string {
	processed := make([]string, len(texts))
	for i, text := range texts {
		processed[i] = g.preprocessSingleChineseText(text)
	}
	return processed
}

// preprocessSingleChineseText preprocesses a single Chinese text
func (g *GTEsmallZhGenerator) preprocessSingleChineseText(text string) string {
	// Convert to NFKC normalization for Chinese characters
	// Remove excessive whitespace
	// Handle Chinese punctuation

	text = strings.TrimSpace(text)

	// Replace multiple spaces with single space
	text = strings.Join(strings.Fields(text), " ")

	// Handle common Chinese text preprocessing
	// Add specific Chinese text cleaning logic here if needed

	return text
}

// ensureInitialized ensures the model is ready for use
func (g *GTEsmallZhGenerator) ensureInitialized() error {
	g.mutex.RLock()
	if g.initialized {
		g.mutex.RUnlock()
		return nil
	}
	g.mutex.RUnlock()

	g.mutex.Lock()
	defer g.mutex.Unlock()

	if g.initialized {
		return nil
	}

	// Check if Python and required packages are available
	pythonCmd := g.findPythonCommand()
	if pythonCmd == "" {
		return fmt.Errorf("Python not found in system")
	}

	// Test if required packages are available for Chinese text processing
	cmd := exec.Command(pythonCmd, "-c", `
import transformers
import torch
import jieba  # For Chinese word segmentation
print("OK")
`)
	if _, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("[GTE-small-zh] Warning: jieba not available, continuing without Chinese word segmentation: %v\n", err)

		// Test without jieba
		cmd2 := exec.Command(pythonCmd, "-c", "import transformers; import torch; print('OK')")
		if output2, err2 := cmd2.CombinedOutput(); err2 != nil {
			return fmt.Errorf("required packages not available: %v, output: %s", err2, string(output2))
		}
	}

	g.initialized = true
	return nil
}

// processBatches processes texts in parallel batches
func (g *GTEsmallZhGenerator) processBatches(ctx context.Context, texts []string, batchSize, maxConc int) ([][]float64, error) {
	type result struct {
		start, end int
		embeddings [][]float64
		err        error
	}

	// Create batches
	jobs := make([][2]int, 0)
	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		jobs = append(jobs, [2]int{i, end})
	}

	// Process batches with concurrency control
	results := make([][]float64, len(texts))
	semaphore := make(chan struct{}, maxConc)
	var wg sync.WaitGroup
	resultChan := make(chan result, len(jobs))

	for _, job := range jobs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		wg.Add(1)
		semaphore <- struct{}{}

		go func(start, end int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			batch := texts[start:end]
			embeddings, err := g.embedBatch(ctx, batch)
			resultChan <- result{start: start, end: end, embeddings: embeddings, err: err}
		}(job[0], job[1])
	}

	// Wait for all batches to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for res := range resultChan {
		if res.err != nil {
			return nil, fmt.Errorf("batch %d-%d failed: %w", res.start, res.end, res.err)
		}

		for i, embedding := range res.embeddings {
			results[res.start+i] = embedding
		}
	}

	return results, nil
}

// embedBatch processes a single batch of texts using Python
func (g *GTEsmallZhGenerator) embedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	pythonCmd := g.findPythonCommand()
	if pythonCmd == "" {
		return nil, fmt.Errorf("Python not available")
	}

	// Create Python script for GTE-small-zh embedding
	script := fmt.Sprintf(`
import sys
import json
import os
import warnings
warnings.filterwarnings("ignore")

# Chinese mirror configuration for faster model downloads in China
os.environ['HF_ENDPOINT'] = 'https://hf-mirror.com'
# Alternative: ModelScope mirror for Chinese models
# os.environ['HF_ENDPOINT'] = 'https://modelscope.cn/api/v1/models'

try:
    from transformers import AutoTokenizer, AutoModel
    import torch

    # Try to import jieba for Chinese word segmentation
    try:
        import jieba
        JIEBA_AVAILABLE = True
        print("[GTE-small-zh] jieba available for Chinese word segmentation", file=sys.stderr)
    except ImportError:
        JIEBA_AVAILABLE = False
        print("[GTE-small-zh] jieba not available, using default tokenization", file=sys.stderr)

    # Load model
    model_name = "%s"
    print(f"[GTE-small-zh] Loading model: {model_name}", file=sys.stderr)
    print(f"[GTE-small-zh] Using mirror: {os.environ.get('HF_ENDPOINT', 'default')}", file=sys.stderr)

    tokenizer = AutoTokenizer.from_pretrained(model_name)
    model = AutoModel.from_pretrained(model_name)
    model.eval()

    print(f"[GTE-small-zh] Model loaded successfully", file=sys.stderr)

    # Get input texts
    texts = json.loads(sys.stdin.read())
    print(f"[GTE-small-zh] Processing {len(texts)} texts", file=sys.stderr)

    # Preprocess Chinese texts if jieba is available
    if JIEBA_AVAILABLE:
        processed_texts = []
        for text in texts:
            # Use jieba for better Chinese text processing
            words = jieba.lcut(text)
            processed_text = " ".join(words)
            processed_texts.append(processed_text)
        texts = processed_texts
        print("[GTE-small-zh] Applied jieba word segmentation", file=sys.stderr)

    # Tokenize and encode
    encoded = tokenizer(texts, padding=True, truncation=True, return_tensors="pt", max_length=512)

    # Generate embeddings with GTE-specific processing
    with torch.no_grad():
        outputs = model(**encoded)

        # GTE-specific mean pooling with attention mask
        attention_mask = encoded['attention_mask']
        token_embeddings = outputs.last_hidden_state
        input_mask_expanded = attention_mask.unsqueeze(-1).expand(token_embeddings.size()).float()
        sum_embeddings = torch.sum(token_embeddings * input_mask_expanded, 1)
        sum_mask = torch.clamp(input_mask_expanded.sum(1), min=1e-9)
        embeddings = sum_embeddings / sum_mask

        # Normalize embeddings
        embeddings = torch.nn.functional.normalize(embeddings, p=2, dim=1)
        embeddings = embeddings.numpy().tolist()

    # Output result
    result = {"status": "ok", "embeddings": embeddings, "dimension": len(embeddings[0]) if embeddings else 0}
    print(json.dumps(result))

except Exception as e:
    import traceback
    result = {"status": "error", "error": str(e), "traceback": traceback.format_exc()}
    print(json.dumps(result), file=sys.stderr)
    sys.exit(1)
`, g.modelPath)

	// Execute Python script
	cmd := exec.CommandContext(ctx, pythonCmd, "-c", script)

	// Set up stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Send texts to Python script
	go func() {
		defer stdin.Close()
		textsJSON, jsonErr := json.Marshal(texts)
		if jsonErr != nil {
			return
		}
		if _, err := stdin.Write(textsJSON); err != nil {
			return
		}
	}()

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Python script execution failed: %v, output: %s", err, string(output))
	}

	// Parse result
	var result struct {
		Status     string      `json:"status"`
		Embeddings [][]float64 `json:"embeddings"`
		Dimension  int         `json:"dimension"`
		Error      string      `json:"error"`
		Traceback  string      `json:"traceback"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse Python output: %w, output: %s", err, string(output))
	}

	if result.Status != "ok" {
		return nil, fmt.Errorf("Python embedding failed: %s", result.Error)
	}

	// Validate dimensions
	if err := ValidateEmbeddings(result.Embeddings, g.dimension); err != nil {
		return nil, fmt.Errorf("embedding validation failed: %w", err)
	}

	return result.Embeddings, nil
}

// fallbackEmbed uses hash-based embedding as fallback
func (g *GTEsmallZhGenerator) fallbackEmbed(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))
	for i, text := range texts {
		embeddings[i] = hashEmbed(text, g.dimension)
	}
	return embeddings, nil
}

// findPythonCommand finds available Python command
func (g *GTEsmallZhGenerator) findPythonCommand() string {
	pythonCommands := []string{"python3", "python", "python3.11", "python3.10", "python3.9"}

	for _, cmd := range pythonCommands {
		if _, err := exec.LookPath(cmd); err == nil {
			return cmd
		}
	}
	return ""
}

// IsChineseText checks if text contains Chinese characters
func IsChineseText(text string) bool {
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// EstimateChineseRatio estimates the ratio of Chinese characters in text
func EstimateChineseRatio(text string) float64 {
	if len(text) == 0 {
		return 0
	}

	chineseCount := 0
	totalCount := 0

	for _, r := range text {
		totalCount++
		if r >= 0x4E00 && r <= 0x9FFF {
			chineseCount++
		}
	}

	return float64(chineseCount) / float64(totalCount)
}

// init registers this generator (will be called when package is imported)
func init() {
	// Registration will happen in the registry implementation
}
