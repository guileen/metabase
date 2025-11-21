package embedding

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultRegistry is the default implementation of VectorGeneratorRegistry
type DefaultRegistry struct {
	mu           sync.RWMutex
	factories    map[string]func(config VectorGeneratorConfig) (VectorGenerator, error)
	capabilities map[string]ModelCapabilities
}

// NewDefaultRegistry creates a new default registry
func NewDefaultRegistry() *DefaultRegistry {
	registry := &DefaultRegistry{
		factories:    make(map[string]func(config VectorGeneratorConfig) (VectorGenerator, error)),
		capabilities: make(map[string]ModelCapabilities),
	}

	// Register built-in generators
	registry.registerBuiltinGenerators()

	return registry
}

// Register implements VectorGeneratorRegistry
func (r *DefaultRegistry) Register(name string, factory func(config VectorGeneratorConfig) (VectorGenerator, error)) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("generator '%s' already registered", name)
	}

	r.factories[name] = factory
	return nil
}

// Get implements VectorGeneratorRegistry
func (r *DefaultRegistry) Get(name string, config VectorGeneratorConfig) (VectorGenerator, error) {
	r.mu.RLock()
	factory, exists := r.factories[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("generator '%s' not found. Available generators: %v", name, r.List())
	}

	return factory(config)
}

// List implements VectorGeneratorRegistry
func (r *DefaultRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}

	return names
}

// GetCapabilities implements VectorGeneratorRegistry
func (r *DefaultRegistry) GetCapabilities(name string) (ModelCapabilities, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// If capabilities are pre-cached, return them
	if caps, exists := r.capabilities[name]; exists {
		return caps, nil
	}

	// Special handling for known models to avoid creation errors
	switch name {
	case "legacy-local":
		caps := ModelCapabilities{
			Languages:            []string{"en", "zh", "es", "fr", "de", "it", "pt", "ru", "ja", "ko"},
			MaxSequenceLength:    512,
			RecommendedBatchSize: 32,
			SupportsMultilingual: true,
			OptimizedForChinese:  false,
			SupportsGPU:          false,
			ModelSizeBytes:       80 * 1024 * 1024,
			EstimatedMemoryUsage: 200 * 1024 * 1024,
		}
		r.capabilities[name] = caps
		return caps, nil
	case "hash-fallback":
		caps := ModelCapabilities{
			Languages:            []string{"*"},
			MaxSequenceLength:    -1,
			RecommendedBatchSize: 1000,
			SupportsMultilingual: true,
			OptimizedForChinese:  false,
			SupportsGPU:          false,
			ModelSizeBytes:       0,
			EstimatedMemoryUsage: 1 * 1024 * 1024,
		}
		r.capabilities[name] = caps
		return caps, nil
	}

	// Otherwise, try to create a temporary instance to get capabilities
	factory, exists := r.factories[name]
	if !exists {
		return ModelCapabilities{}, fmt.Errorf("generator '%s' not found", name)
	}

	// Create a temporary instance with minimal config
	tempGen, err := factory(VectorGeneratorConfig{})
	if err != nil {
		return ModelCapabilities{}, fmt.Errorf("failed to create generator to get capabilities: %w", err)
	}

	caps := tempGen.GetCapabilities()
	tempGen.Close()

	// Cache capabilities for future calls
	r.capabilities[name] = caps

	return caps, nil
}

// registerBuiltinGenerators registers all built-in vector generators
func (r *DefaultRegistry) registerBuiltinGenerators() {
	// Register all-MiniLM-L6-v2
	r.Register("all-minilm-l6-v2", func(config VectorGeneratorConfig) (VectorGenerator, error) {
		return NewAllMiniLML6V2Generator(config)
	})

	// Register GTE-small-zh
	r.Register("gte-small-zh", func(config VectorGeneratorConfig) (VectorGenerator, error) {
		return NewGTEsmallZhGenerator(config)
	})

	// Register stsb-bert-tiny
	r.Register("stsb-bert-tiny", func(config VectorGeneratorConfig) (VectorGenerator, error) {
		return NewSTSBBertTinyGenerator(config)
	})

	// Register adapter for existing LocalEmbedder
	r.Register("legacy-local", func(config VectorGeneratorConfig) (VectorGenerator, error) {
		legacyConfig := &Config{
			LocalModelType: "python",
			LocalModelPath: "sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2",
			CacheDir:       config.CacheDir,
			BaseURL:        "",
			APIKey:         "",
			Model:          "",
			Timeout:        config.Timeout,
			BatchSize:      config.BatchSize,
			MaxConcurrency: config.MaxConcurrency,
			EnableFallback: config.EnableFallback,
		}

		// Safely extract values from ModelConfig if available
		if config.ModelConfig != nil {
			if val, ok := config.ModelConfig["local_model_type"].(string); ok {
				legacyConfig.LocalModelType = val
			}
			if val, ok := config.ModelConfig["local_model_path"].(string); ok {
				legacyConfig.LocalModelPath = val
			}
			if val, ok := config.ModelConfig["base_url"].(string); ok {
				legacyConfig.BaseURL = val
			}
			if val, ok := config.ModelConfig["api_key"].(string); ok {
				legacyConfig.APIKey = val
			}
			if val, ok := config.ModelConfig["model"].(string); ok {
				legacyConfig.Model = val
			}
		}

		legacy, err := NewLocalEmbedder(legacyConfig)
		if err != nil {
			return nil, err
		}

		return NewLegacyAdapter(legacy, "all-MiniLM-L6-v2"), nil
	})

	// Register hash-based fallback generator
	r.Register("hash-fallback", func(config VectorGeneratorConfig) (VectorGenerator, error) {
		return NewHashFallbackGenerator(config), nil
	})

	// Register Cybertron-based models (stub implementation for now)
	r.Register("all-minilm-l6-v2-cybertron", func(config VectorGeneratorConfig) (VectorGenerator, error) {
		return NewCybertronMiniLML6V2(config)
	})

	r.Register("gte-small-zh-cybertron", func(config VectorGeneratorConfig) (VectorGenerator, error) {
		return NewCybertronGTEsmallZh(config)
	})

	r.Register("stsb-bert-tiny-cybertron", func(config VectorGeneratorConfig) (VectorGenerator, error) {
		return NewCybertronSTSBbertTiny(config)
	})
}

// Global default registry
var defaultRegistry = NewDefaultRegistry()
var registryOnce sync.Once

// GetDefaultRegistry returns the default global registry
func GetDefaultRegistry() VectorGeneratorRegistry {
	registryOnce.Do(func() {
		defaultRegistry = NewDefaultRegistry()
	})
	return defaultRegistry
}

// CreateGenerator creates a generator using the default registry
func CreateGenerator(name string, config VectorGeneratorConfig) (VectorGenerator, error) {
	return GetDefaultRegistry().Get(name, config)
}

// ListGenerators returns all available generator names
func ListGenerators() []string {
	return GetDefaultRegistry().List()
}

// GetGeneratorCapabilities returns capabilities for a generator
func GetGeneratorCapabilities(name string) (ModelCapabilities, error) {
	return GetDefaultRegistry().GetCapabilities(name)
}

// LegacyAdapter adapts the existing LocalEmbedder to implement VectorGenerator interface
type LegacyAdapter struct {
	legacy    *LocalEmbedder
	modelName string
	dimension int
}

// NewLegacyAdapter creates a new adapter for legacy embedder
func NewLegacyAdapter(legacy *LocalEmbedder, modelName string) *LegacyAdapter {
	return &LegacyAdapter{
		legacy:    legacy,
		modelName: modelName,
		dimension: legacy.GetDimension(),
	}
}

// Embed implements VectorGenerator
func (la *LegacyAdapter) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	// Convert context to timeout if available
	if deadline, ok := ctx.Deadline(); ok {
		timeout := time.Until(deadline)
		if timeout > 0 {
			// Update the legacy embedder's timeout if possible
			// This is a limitation of the legacy interface
		}
	}

	return la.legacy.Embed(texts)
}

// EmbedSingle implements VectorGenerator
func (la *LegacyAdapter) EmbedSingle(ctx context.Context, text string) ([]float64, error) {
	embeddings, err := la.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding generated")
	}
	return embeddings[0], nil
}

// GetDimension implements VectorGenerator
func (la *LegacyAdapter) GetDimension() int {
	return la.dimension
}

// GetModelName implements VectorGenerator
func (la *LegacyAdapter) GetModelName() string {
	return la.modelName
}

// GetCapabilities implements VectorGenerator
func (la *LegacyAdapter) GetCapabilities() ModelCapabilities {
	return ModelCapabilities{
		Languages:            []string{"en", "zh", "es", "fr", "de", "it", "pt", "ru", "ja", "ko"},
		MaxSequenceLength:    512,
		RecommendedBatchSize: 32,
		SupportsMultilingual: true,
		OptimizedForChinese:  false,
		SupportsGPU:          false,
		ModelSizeBytes:       80 * 1024 * 1024,
		EstimatedMemoryUsage: 200 * 1024 * 1024,
	}
}

// Close implements VectorGenerator
func (la *LegacyAdapter) Close() error {
	return la.legacy.Close()
}

// HashFallbackGenerator provides a pure hash-based embedding generator
type HashFallbackGenerator struct {
	config       VectorGeneratorConfig
	dimension    int
	capabilities ModelCapabilities
}

// NewHashFallbackGenerator creates a new hash-based fallback generator
func NewHashFallbackGenerator(config VectorGeneratorConfig) *HashFallbackGenerator {
	if config.ModelName == "" {
		config.ModelName = "hash-fallback"
	}

	return &HashFallbackGenerator{
		config:    config,
		dimension: 384, // Default hash embedding dimension
		capabilities: ModelCapabilities{
			Languages:            []string{"*"}, // Supports all languages
			MaxSequenceLength:    -1,            // No limit
			RecommendedBatchSize: 1000,
			SupportsMultilingual: true,
			OptimizedForChinese:  false,
			SupportsGPU:          false,
			ModelSizeBytes:       0,               // No model file
			EstimatedMemoryUsage: 1 * 1024 * 1024, // Minimal memory usage
		},
	}
}

// Embed implements VectorGenerator
func (hfg *HashFallbackGenerator) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))
	for i, text := range texts {
		embeddings[i] = hashEmbed(text, hfg.dimension)
	}
	return embeddings, nil
}

// EmbedSingle implements VectorGenerator
func (hfg *HashFallbackGenerator) EmbedSingle(ctx context.Context, text string) ([]float64, error) {
	return hashEmbed(text, hfg.dimension), nil
}

// GetDimension implements VectorGenerator
func (hfg *HashFallbackGenerator) GetDimension() int {
	return hfg.dimension
}

// GetModelName implements VectorGenerator
func (hfg *HashFallbackGenerator) GetModelName() string {
	return "hash-fallback"
}

// GetCapabilities implements VectorGenerator
func (hfg *HashFallbackGenerator) GetCapabilities() ModelCapabilities {
	return hfg.capabilities
}

// Close implements VectorGenerator
func (hfg *HashFallbackGenerator) Close() error {
	return nil // No resources to cleanup
}
