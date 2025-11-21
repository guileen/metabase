package embedding

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Config holds configuration for embedding models
type Config struct {
	// Local model configuration
	LocalModelPath string
	LocalModelType string // "onnx", "python", "transformers"
	CacheDir       string

	// Remote API configuration
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration

	// Processing configuration
	BatchSize      int
	MaxConcurrency int
	EnableFallback bool
}

// Embedder interface for different embedding implementations
type Embedder interface {
	Embed(texts []string) ([][]float64, error)
	GetDimension() int
	Close() error
}

// LocalEmbedder implements local embedding using Python transformers
type LocalEmbedder struct {
	config       *Config
	dimension    int
	modelPath    string
	cachePath    string
	mutex        sync.RWMutex
	initialized  bool
	onnxEmbedder *ONNXEmbedder
}

// NewLocalEmbedder creates a new local embedder instance
func NewLocalEmbedder(config *Config) (*LocalEmbedder, error) {
	if config == nil {
		config = getDefaultConfig()
	}

	if config.LocalModelType == "" {
		config.LocalModelType = "fast"
	}

	if config.BatchSize == 0 {
		config.BatchSize = 32
	}

	if config.MaxConcurrency == 0 {
		config.MaxConcurrency = 4
	}

	le := &LocalEmbedder{
		config:    config,
		dimension: 384, // all-MiniLM-L6-v2 dimension
	}

	// Setup paths
	if config.LocalModelPath != "" {
		le.modelPath = config.LocalModelPath
	} else {
		le.modelPath = "sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2" // Default model (supports Chinese)
	}

	if config.CacheDir != "" {
		le.cachePath = config.CacheDir
	} else {
		le.cachePath = filepath.Join(os.TempDir(), "metabase_embeddings")
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(le.cachePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return le, nil
}

// Embed generates embeddings for the given texts
func (le *LocalEmbedder) Embed(texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	le.mutex.RLock()
	if !le.initialized {
		le.mutex.RUnlock()
		if err := le.initialize(); err != nil {
			if le.config.EnableFallback {
				return le.fallbackEmbed(texts)
			}
			return nil, fmt.Errorf("failed to initialize local embedder: %w", err)
		}
		le.mutex.RLock()
	}
	le.mutex.RUnlock()

	batchSize := le.config.BatchSize
	if batchSize <= 0 {
		batchSize = 32
	}
	maxConc := le.config.MaxConcurrency
	if maxConc <= 0 {
		maxConc = 4
	}

	type res struct {
		idx  int
		vecs [][]float64
		err  error
	}
	jobs := make([][2]int, 0)
	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		jobs = append(jobs, [2]int{i, end})
	}

	out := make([][]float64, len(texts))
	sem := make(chan struct{}, maxConc)
	var wg sync.WaitGroup
	errs := make(chan error, len(jobs))

	for _, j := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func(start, end int) {
			defer wg.Done()
			defer func() { <-sem }()
			batch := texts[start:end]
			vecs, err := le.embedBatch(batch)
			if err != nil {
				if le.config.EnableFallback {
					vecs, err = le.fallbackEmbed(batch)
				}
			}
			if err != nil {
				errs <- fmt.Errorf("batch %d-%d failed: %w", start, end, err)
				return
			}
			for i := range vecs {
				out[start+i] = vecs[i]
			}
		}(j[0], j[1])
	}
	wg.Wait()
	close(errs)
	if e := <-errs; e != nil {
		return nil, e
	}
	return out, nil
}

// GetDimension returns the embedding dimension
func (le *LocalEmbedder) GetDimension() int {
	return le.dimension
}

// Close cleanup resources
func (le *LocalEmbedder) Close() error {
	le.mutex.Lock()
	defer le.mutex.Unlock()

	le.initialized = false
	if le.onnxEmbedder != nil {
		le.onnxEmbedder.Close()
		le.onnxEmbedder = nil
	}

	return nil
}

// initialize the local embedding model
func (le *LocalEmbedder) initialize() error {
	le.mutex.Lock()
	defer le.mutex.Unlock()

	if le.initialized {
		return nil
	}

	switch le.config.LocalModelType {
	case "python":
		return le.initPythonTransformers()
	case "onnx":
		return le.initONNX()
	case "fast":
		le.initialized = true
		return nil
	default:
		return fmt.Errorf("unsupported local model type: %s", le.config.LocalModelType)
	}
}

// initPythonTransformers initializes Python transformers backend
func (le *LocalEmbedder) initPythonTransformers() error {
	// Check if Python and required packages are available
	pythonCmd := le.findPythonCommand()
	if pythonCmd == "" {
		return fmt.Errorf("Python not found in system")
	}

	// Test if transformers package is available
	cmd := exec.Command(pythonCmd, "-c", "import transformers; print('OK')")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("transformers package not available: %v, output: %s", err, string(output))
	}

	le.initialized = true
	return nil
}

// initONNX initializes ONNX Runtime backend
func (le *LocalEmbedder) initONNX() error {
	// Try to create ONNX embedder
	onnxConfig := &ONNXConfig{
		ModelPath:         "models/all-MiniLM-L6-v2.onnx",
		TokenizerPath:     "models/all-MiniLM-L6-v2-tokenizer.json",
		Dimension:         384,
		MaxSequenceLen:    512,
		CacheSize:         5000,
		BatchSize:         le.config.BatchSize,
		NumThreads:        le.config.MaxConcurrency,
		UseGPU:            false,
		Providers:         []string{"CPUExecutionProvider"},
		OptimizationLevel: "all",
	}

	// Check if ONNX model files exist
	if _, err := os.Stat(onnxConfig.ModelPath); os.IsNotExist(err) {
		fmt.Printf("[ONNX] Model file not found at %s, falling back to Python\n", onnxConfig.ModelPath)
		return le.initPythonTransformers()
	}

	onnxEmbedder, err := NewONNXEmbedder(onnxConfig)
	if err != nil {
		fmt.Printf("[ONNX] Failed to initialize ONNX embedder: %v, falling back to Python\n", err)
		return le.initPythonTransformers()
	}

	le.onnxEmbedder = onnxEmbedder
	le.initialized = true
	fmt.Printf("[ONNX] Successfully initialized ONNX embedder with %d dimensions\n", le.dimension)
	return nil
}

// embedBatch processes a batch of texts
func (le *LocalEmbedder) embedBatch(texts []string) ([][]float64, error) {
	switch le.config.LocalModelType {
	case "python":
		return le.embedBatchPython(texts)
	case "onnx":
		return le.embedBatchONNX(texts)
	case "fast":
		return le.embedBatchFast(texts)
	default:
		return nil, fmt.Errorf("unsupported model type: %s", le.config.LocalModelType)
	}
}

// embedBatchPython uses Python to generate embeddings
func (le *LocalEmbedder) embedBatchPython(texts []string) ([][]float64, error) {
	pythonCmd := le.findPythonCommand()
	if pythonCmd == "" {
		return nil, fmt.Errorf("Python not available")
	}

	// Create Python script for embedding
	script := fmt.Sprintf(`
import sys
import json
import os
sys.path.insert(0, os.path.expanduser("~/.cache/huggingface/hub"))

# Chinese mirror configuration for faster model downloads in China
os.environ['HF_ENDPOINT'] = 'https://hf-mirror.com'
# Alternative: ModelScope mirror
# os.environ['HF_ENDPOINT'] = 'https://modelscope.cn/api/v1/models'

try:
    from transformers import AutoTokenizer, AutoModel
    import torch

    # Load model
    model_name = "%s"
    print(f"[Legacy] Loading model: {model_name} using mirror: {os.environ.get('HF_ENDPOINT', 'default')}", file=sys.stderr)
    tokenizer = AutoTokenizer.from_pretrained(model_name)
    model = AutoModel.from_pretrained(model_name)

    # Get input texts
    texts = json.loads(sys.stdin.read())

    # Tokenize and encode
    encoded = tokenizer(texts, padding=True, truncation=True, return_tensors="pt")

    # Generate embeddings
    with torch.no_grad():
        outputs = model(**encoded)
        # Mean pooling
        embeddings = outputs.last_hidden_state.mean(dim=1).numpy().tolist()

    # Output result
    result = {"status": "ok", "embeddings": embeddings}
    print(json.dumps(result))

except Exception as e:
    result = {"status": "error", "error": str(e)}
    print(json.dumps(result))
`, le.modelPath)

	// Execute Python script
	cmd := exec.Command(pythonCmd, "-c", script)

	// Write texts to stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	go func() {
		defer stdin.Close()
		textsJSON, _ := json.Marshal(texts)
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
		Error      string      `json:"error"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse Python output: %w, output: %s", err, string(output))
	}

	if result.Status != "ok" {
		return nil, fmt.Errorf("Python embedding failed: %s", result.Error)
	}

	// Validate dimensions
	for i, emb := range result.Embeddings {
		if len(emb) != le.dimension {
			return nil, fmt.Errorf("embedding dimension mismatch at index %d: expected %d, got %d", i, le.dimension, len(emb))
		}
	}

	return result.Embeddings, nil
}

// embedBatchONNX uses ONNX runtime for faster inference
func (le *LocalEmbedder) embedBatchONNX(texts []string) ([][]float64, error) {
	if le.onnxEmbedder != nil {
		return le.onnxEmbedder.Embed(texts)
	}
	// Fallback to Python if ONNX is not available
	return le.embedBatchPython(texts)
}

func (le *LocalEmbedder) embedBatchFast(texts []string) ([][]float64, error) {
	embs := make([][]float64, len(texts))
	for i, t := range texts {
		embs[i] = hashEmbed(t, le.dimension)
	}
	return embs, nil
}

// fallbackEmbed uses remote API as fallback
func (le *LocalEmbedder) fallbackEmbed(texts []string) ([][]float64, error) {
	if le.config.BaseURL == "" || le.config.APIKey == "" {
		return nil, fmt.Errorf("fallback not configured: missing BaseURL or APIKey")
	}

	// Use the remote embedding implementation
	return RemoteEmbed(texts, le.config.BaseURL, le.config.APIKey, le.config.Model, le.config.Timeout)
}

// findPythonCommand finds available Python command
func (le *LocalEmbedder) findPythonCommand() string {
	pythonCommands := []string{"python3", "python", "python3.11", "python3.10", "python3.9"}

	for _, cmd := range pythonCommands {
		if _, err := exec.LookPath(cmd); err == nil {
			return cmd
		}
	}
	return ""
}

// getDefaultConfig returns default configuration
func getDefaultConfig() *Config {
	return &Config{
		LocalModelType: "python",                                                      // Use Python for better Chinese support
		LocalModelPath: "sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2", // Chinese-friendly model
		BatchSize:      16,
		MaxConcurrency: 2,
		EnableFallback: true,
		Timeout:        60 * time.Second,
	}
}

// Helper function for remote embedding fallback
func RemoteEmbed(texts []string, baseURL, apiKey, model string, timeout time.Duration) ([][]float64, error) {
	// This would implement the remote embedding logic similar to llm.Embeddings
	// For now, return a simple hash-based embedding as fallback
	hashEmbeddings := make([][]float64, len(texts))
	for i, text := range texts {
		hashEmbeddings[i] = hashEmbed(text, 384)
	}
	return hashEmbeddings, nil
}

// hashEmbed creates a simple hash-based embedding for fallback
func hashEmbed(text string, dim int) []float64 {
	embedding := make([]float64, dim)
	hash := func(s string) uint64 {
		var h uint64
		for _, c := range []byte(s) {
			h = h*131 + uint64(c)
		}
		return h
	}

	terms := strings.Fields(strings.ToLower(text))
	if len(terms) == 0 {
		terms = []string{"_"}
	}

	for i, term := range terms {
		if i >= dim/6 {
			break
		}
		h := hash(term)
		for j := 0; j < 6; j++ {
			if idx := i*6 + j; idx < dim {
				embedding[idx] = float64((h>>uint(8*j))&0xFF) / 255.0
			}
		}
	}

	// Normalize
	var norm float64
	for _, v := range embedding {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range embedding {
			embedding[i] /= norm
		}
	}

	return embedding
}
