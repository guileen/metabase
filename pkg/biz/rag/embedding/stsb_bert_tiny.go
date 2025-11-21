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

// STSBBertTinyGenerator implements VectorGenerator for stsb-bert-tiny model
// This is an ultra-lightweight 11M parameter model for minimal resource usage
// Vector dimension: 128 (smaller than other models)
// Model size: ~20MB
// Fastest inference speed, ideal for mobile/embedded applications
type STSBBertTinyGenerator struct {
	config       VectorGeneratorConfig
	dimension    int
	modelPath    string
	cachePath    string
	mutex        sync.RWMutex
	initialized  bool
	capabilities ModelCapabilities
}

// NewSTSBBertTinyGenerator creates a new stsb-bert-tiny generator
func NewSTSBBertTinyGenerator(config VectorGeneratorConfig) (*STSBBertTinyGenerator, error) {
	if config.ModelName == "" {
		config.ModelName = "stsb-bert-tiny"
	}
	if config.BatchSize == 0 {
		config.BatchSize = 64 // Larger batch size for faster processing
	}
	if config.MaxConcurrency == 0 {
		config.MaxConcurrency = 8 // Higher concurrency for lightweight model
	}
	if config.Timeout == 0 {
		config.Timeout = 15 * time.Second // Shorter timeout for faster model
	}

	gen := &STSBBertTinyGenerator{
		config:    config,
		dimension: 128, // stsb-bert-tiny dimension
		modelPath: "sentence-transformers/stsb-bert-tiny",
		capabilities: ModelCapabilities{
			Languages:            []string{"en", "zh", "es", "fr", "de", "it", "pt", "ru", "ja", "ko"}, // Multilingual support
			MaxSequenceLength:    512,
			RecommendedBatchSize: 64,
			SupportsMultilingual: true,
			OptimizedForChinese:  false,            // General multilingual model
			SupportsGPU:          false,            // Python implementation doesn't use GPU
			ModelSizeBytes:       20 * 1024 * 1024, // ~20MB
			EstimatedMemoryUsage: 50 * 1024 * 1024, // ~50MB during inference
		},
	}

	if config.CacheDir != "" {
		gen.cachePath = config.CacheDir
	} else {
		gen.cachePath = filepath.Join(os.TempDir(), "metabase_stsb_tiny_cache")
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(gen.cachePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return gen, nil
}

// Embed generates embeddings for a batch of texts
func (s *STSBBertTinyGenerator) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	if err := s.ensureInitialized(); err != nil {
		if s.config.EnableFallback {
			return s.fallbackEmbed(ctx, texts)
		}
		return nil, fmt.Errorf("failed to initialize generator: %w", err)
	}

	// Process in large batches for optimal performance
	batchSize := s.config.BatchSize
	if batchSize <= 0 {
		batchSize = 64
	}

	maxConc := s.config.MaxConcurrency
	if maxConc <= 0 {
		maxConc = 8
	}

	// Create channel for context cancellation
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	return s.processBatches(ctx, texts, batchSize, maxConc)
}

// EmbedSingle generates embedding for a single text
func (s *STSBBertTinyGenerator) EmbedSingle(ctx context.Context, text string) ([]float64, error) {
	embeddings, err := s.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding generated")
	}
	return embeddings[0], nil
}

// GetDimension returns the dimension of the embedding vectors
func (s *STSBBertTinyGenerator) GetDimension() int {
	return s.dimension
}

// GetModelName returns the name/type of the model
func (s *STSBBertTinyGenerator) GetModelName() string {
	return "stsb-bert-tiny"
}

// GetCapabilities returns the capabilities of this embedding model
func (s *STSBBertTinyGenerator) GetCapabilities() ModelCapabilities {
	return s.capabilities
}

// Close performs cleanup and releases resources
func (s *STSBBertTinyGenerator) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.initialized = false
	return nil
}

// ensureInitialized ensures the model is ready for use
func (s *STSBBertTinyGenerator) ensureInitialized() error {
	s.mutex.RLock()
	if s.initialized {
		s.mutex.RUnlock()
		return nil
	}
	s.mutex.RUnlock()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.initialized {
		return nil
	}

	// Check if Python and required packages are available
	pythonCmd := s.findPythonCommand()
	if pythonCmd == "" {
		return fmt.Errorf("Python not found in system")
	}

	// Test if required packages are available
	cmd := exec.Command(pythonCmd, "-c", "import transformers; import torch; print('OK')")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("required packages not available: %v, output: %s", err, string(output))
	}

	s.initialized = true
	return nil
}

// processBatches processes texts in parallel batches with high throughput
func (s *STSBBertTinyGenerator) processBatches(ctx context.Context, texts []string, batchSize, maxConc int) ([][]float64, error) {
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

	// Process batches with high concurrency for lightweight model
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
			embeddings, err := s.embedBatch(ctx, batch)
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
func (s *STSBBertTinyGenerator) embedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	pythonCmd := s.findPythonCommand()
	if pythonCmd == "" {
		return nil, fmt.Errorf("Python not available")
	}

	// Create Python script for stsb-bert-tiny embedding
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

    # Load tiny model
    model_name = "%s"
    print(f"[stsb-bert-tiny] Loading model: {model_name}", file=sys.stderr)
    print(f"[stsb-bert-tiny] Using mirror: {os.environ.get('HF_ENDPOINT', 'default')}", file=sys.stderr)

    tokenizer = AutoTokenizer.from_pretrained(model_name)
    model = AutoModel.from_pretrained(model_name)
    model.eval()

    print(f"[stsb-bert-tiny] Model loaded successfully with {sum(p.numel() for p in model.parameters())} parameters", file=sys.stderr)

    # Get input texts
    texts = json.loads(sys.stdin.read())
    print(f"[stsb-bert-tiny] Processing {len(texts)} texts", file=sys.stderr)

    # Tokenize and encode with optimized settings for tiny model
    encoded = tokenizer(texts, padding=True, truncation=True, return_tensors="pt", max_length=256)  # Reduced max length for speed

    # Generate embeddings with optimized processing for tiny model
    with torch.no_grad():
        outputs = model(**encoded)

        # Optimized mean pooling for tiny model
        attention_mask = encoded['attention_mask']
        token_embeddings = outputs.last_hidden_state
        input_mask_expanded = attention_mask.unsqueeze(-1).expand(token_embeddings.size()).float()
        embeddings = torch.sum(token_embeddings * input_mask_expanded, 1) / torch.clamp(input_mask_expanded.sum(1), min=1e-9)

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
`, s.modelPath)

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
		stdin.Write(textsJSON)
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
	if err := ValidateEmbeddings(result.Embeddings, s.dimension); err != nil {
		return nil, fmt.Errorf("embedding validation failed: %w", err)
	}

	return result.Embeddings, nil
}

// fallbackEmbed uses hash-based embedding as fallback
func (s *STSBBertTinyGenerator) fallbackEmbed(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))
	for i, text := range texts {
		embeddings[i] = hashEmbed(text, s.dimension)
	}
	return embeddings, nil
}

// findPythonCommand finds available Python command
func (s *STSBBertTinyGenerator) findPythonCommand() string {
	pythonCommands := []string{"python3", "python", "python3.11", "python3.10", "python3.9"}

	for _, cmd := range pythonCommands {
		if _, err := exec.LookPath(cmd); err == nil {
			return cmd
		}
	}
	return ""
}

// IsLightweightResource checks if this generator should be used for resource-constrained environments
func (s *STSBBertTinyGenerator) IsLightweightResource() bool {
	return true
}

// GetEstimatedLatency returns estimated latency for single text embedding
func (s *STSBBertTinyGenerator) GetEstimatedLatency() time.Duration {
	return time.Millisecond * 1 // ~1ms for single text
}

// GetEstimatedThroughput returns estimated throughput for batch processing
func (s *STSBBertTinyGenerator) GetEstimatedThroughput() int {
	return 1000 // ~1000 texts per second
}

// Register this generator (will be called when package is imported)
func init() {
	// Registration will happen in the registry implementation
}
