package embedding

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// AllMiniLML6V2Generator implements VectorGenerator for all-MiniLM-L6-v2 model
// This is a 22.7M parameter model that supports 30+ languages including Chinese
// Vector dimension: 384
// Model size: ~80MB
type AllMiniLML6V2Generator struct {
	config       VectorGeneratorConfig
	dimension    int
	modelPath    string
	cachePath    string
	mutex        sync.RWMutex
	initialized  bool
	capabilities ModelCapabilities
}

// NewAllMiniLML6V2Generator creates a new all-MiniLM-L6-v2 generator
func NewAllMiniLML6V2Generator(config VectorGeneratorConfig) (*AllMiniLML6V2Generator, error) {
	if config.ModelName == "" {
		config.ModelName = "all-MiniLM-L6-v2"
	}
	if config.BatchSize == 0 {
		config.BatchSize = 32
	}
	if config.MaxConcurrency == 0 {
		config.MaxConcurrency = 4
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	gen := &AllMiniLML6V2Generator{
		config:    config,
		dimension: 384, // all-MiniLM-L6-v2 dimension
		modelPath: "sentence-transformers/all-MiniLM-L6-v2",
		capabilities: ModelCapabilities{
			Languages:            []string{"en", "zh", "es", "fr", "de", "it", "pt", "ru", "ja", "ko"}, // Supported languages
			MaxSequenceLength:    512,
			RecommendedBatchSize: 32,
			SupportsMultilingual: true,
			OptimizedForChinese:  false,             // General multilingual model
			SupportsGPU:          false,             // Python implementation doesn't use GPU
			ModelSizeBytes:       80 * 1024 * 1024,  // ~80MB
			EstimatedMemoryUsage: 200 * 1024 * 1024, // ~200MB during inference
		},
	}

	if config.CacheDir != "" {
		gen.cachePath = config.CacheDir
	} else {
		gen.cachePath = filepath.Join(os.TempDir(), "metabase_allminilm_cache")
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(gen.cachePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return gen, nil
}

// Embed generates embeddings for a batch of texts
func (g *AllMiniLML6V2Generator) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	if err := g.ensureInitialized(); err != nil {
		if g.config.EnableFallback {
			return g.fallbackEmbed(ctx, texts)
		}
		return nil, fmt.Errorf("failed to initialize generator: %w", err)
	}

	// Process in batches for optimal performance
	batchSize := g.config.BatchSize
	if batchSize <= 0 {
		batchSize = 32
	}

	maxConc := g.config.MaxConcurrency
	if maxConc <= 0 {
		maxConc = 4
	}

	// Create channel for context cancellation
	ctx, cancel := context.WithTimeout(ctx, g.config.Timeout)
	defer cancel()

	return g.processBatches(ctx, texts, batchSize, maxConc)
}

// EmbedSingle generates embedding for a single text
func (g *AllMiniLML6V2Generator) EmbedSingle(ctx context.Context, text string) ([]float64, error) {
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
func (g *AllMiniLML6V2Generator) GetDimension() int {
	return g.dimension
}

// GetModelName returns the name/type of the model
func (g *AllMiniLML6V2Generator) GetModelName() string {
	return "all-MiniLM-L6-v2"
}

// GetCapabilities returns the capabilities of this embedding model
func (g *AllMiniLML6V2Generator) GetCapabilities() ModelCapabilities {
	return g.capabilities
}

// Close performs cleanup and releases resources
func (g *AllMiniLML6V2Generator) Close() error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.initialized = false
	return nil
}

// ensureInitialized ensures the model is ready for use
func (g *AllMiniLML6V2Generator) ensureInitialized() error {
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

	// Test if transformers and torch packages are available
	cmd := exec.Command(pythonCmd, "-c", "import transformers; import torch; print('OK')")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("required packages not available: %v, output: %s", err, string(output))
	}

	g.initialized = true
	return nil
}

// processBatches processes texts in parallel batches
func (g *AllMiniLML6V2Generator) processBatches(ctx context.Context, texts []string, batchSize, maxConc int) ([][]float64, error) {
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
func (g *AllMiniLML6V2Generator) embedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	pythonCmd := g.findPythonCommand()
	if pythonCmd == "" {
		return nil, fmt.Errorf("Python not available")
	}

	// Create Python script for embedding
	script := fmt.Sprintf(`
import sys
import json
import os
import warnings
warnings.filterwarnings("ignore")

# Chinese mirror configuration for faster model downloads in China
os.environ['HF_ENDPOINT'] = 'https://hf-mirror.com'
# Alternative: ModelScope mirror
# os.environ['HF_ENDPOINT'] = 'https://modelscope.cn/api/v1/models'

try:
    from transformers import AutoTokenizer, AutoModel
    import torch

    # Load model
    model_name = "%s"
    print(f"[all-MiniLM-L6-v2] Loading model: {model_name}", file=sys.stderr)
    print(f"[all-MiniLM-L6-v2] Using mirror: {os.environ.get('HF_ENDPOINT', 'default')}", file=sys.stderr)

    tokenizer = AutoTokenizer.from_pretrained(model_name)
    model = AutoModel.from_pretrained(model_name)
    model.eval()

    print(f"[all-MiniLM-L6-v2] Model loaded successfully", file=sys.stderr)

    # Get input texts
    texts = json.loads(sys.stdin.read())
    print(f"[all-MiniLM-L6-v2] Processing {len(texts)} texts", file=sys.stderr)

    # Tokenize and encode
    encoded = tokenizer(texts, padding=True, truncation=True, return_tensors="pt", max_length=512)

    # Generate embeddings
    with torch.no_grad():
        outputs = model(**encoded)
        # Mean pooling
        attention_mask = encoded['attention_mask']
        token_embeddings = outputs.last_hidden_state
        input_mask_expanded = attention_mask.unsqueeze(-1).expand(token_embeddings.size()).float()
        embeddings = torch.sum(token_embeddings * input_mask_expanded, 1) / torch.clamp(input_mask_expanded.sum(1), min=1e-9)
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
func (g *AllMiniLML6V2Generator) fallbackEmbed(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))
	for i, text := range texts {
		embeddings[i] = hashEmbed(text, g.dimension)
	}
	return embeddings, nil
}

// findPythonCommand finds available Python command
func (g *AllMiniLML6V2Generator) findPythonCommand() string {
	pythonCommands := []string{"python3", "python", "python3.11", "python3.10", "python3.9"}

	for _, cmd := range pythonCommands {
		if _, err := exec.LookPath(cmd); err == nil {
			return cmd
		}
	}
	return ""
}

// Register this generator with the registry (will be implemented later)
func init() {
	// This will be called when the package is imported
	// Registration will happen in the registry implementation
}
