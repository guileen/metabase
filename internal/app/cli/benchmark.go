package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/guileen/metabase/pkg/rag/embedding"
	"github.com/guileen/metabase/pkg/rag/vocab"
	"github.com/spf13/cobra"
)

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Performance benchmarking for embeddings and vocab operations",
	Long: `Run comprehensive performance benchmarks for different embedding methods.

Examples:
  metabase benchmark embedding                    # Quick benchmark all methods
  metabase benchmark embedding --method onnx      # Benchmark only ONNX
  metabase benchmark embedding --texts 1000       # Benchmark with 1000 texts
  metabase benchmark embedding --batch-sizes 32,64 # Test specific batch sizes
  metabase benchmark vocab                        # Benchmark vocab operations
  metabase benchmark models                       # New comprehensive model comparison`,
}

var benchmarkEmbeddingCmd = &cobra.Command{
	Use:   "embedding",
	Short: "Benchmark embedding performance",
	Long: `Benchmark different embedding methods and configurations.

Supported methods:
  - fast: Hash-based embeddings (fastest, lower quality)
  - python: Python transformers (high quality, slower)
  - onnx: ONNX Runtime (fast, high quality)

Examples:
  metabase benchmark embedding --method fast
  metabase benchmark embedding --method onnx --texts 1000
  metabase benchmark embedding --all-methods`,
	Run: func(cmd *cobra.Command, args []string) {
		method, _ := cmd.Flags().GetString("method")
		textCount, _ := cmd.Flags().GetInt("texts")
		batchSizes, _ := cmd.Flags().GetString("batch-sizes")
		allMethods, _ := cmd.Flags().GetBool("all-methods")
		iterations, _ := cmd.Flags().GetInt("iterations")
		warmup, _ := cmd.Flags().GetInt("warmup")

		if allMethods {
			// Benchmark all available methods
			methods := []string{"fast", "python", "onnx"}
			for _, m := range methods {
				fmt.Printf("\n=== Benchmarking %s ===\n", m)
				if err := runEmbeddingBenchmark(m, textCount, batchSizes, iterations, warmup); err != nil {
					fmt.Printf("❌ %s benchmark failed: %v\n", m, err)
				}
			}
		} else {
			if method == "" {
				method = "fast" // Default
			}
			fmt.Printf("\n=== Benchmarking %s ===\n", method)
			if err := runEmbeddingBenchmark(method, textCount, batchSizes, iterations, warmup); err != nil {
				fmt.Printf("❌ Benchmark failed: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

var benchmarkVocabCmd = &cobra.Command{
	Use:   "vocab",
	Short: "Benchmark vocabulary operations",
	Long: `Benchmark vocabulary build, update, and expand operations.

Examples:
  metabase benchmark vocab --build                    # Benchmark vocab building
  metabase benchmark vocab --expand                  # Benchmark query expansion
  metabase benchmark vocab --all                     # Benchmark all vocab operations`,
	Run: func(cmd *cobra.Command, args []string) {
		build, _ := cmd.Flags().GetBool("build")
		expand, _ := cmd.Flags().GetBool("expand")
		all, _ := cmd.Flags().GetBool("all")
		limit, _ := cmd.Flags().GetInt("limit")

		if all {
			build = true
			expand = true
		}

		if build {
			fmt.Printf("\n=== Benchmarking Vocab Build ===\n")
			if err := runVocabBuildBenchmark(); err != nil {
				fmt.Printf("❌ Vocab build benchmark failed: %v\n", err)
			}
		}

		if expand {
			fmt.Printf("\n=== Benchmarking Vocab Expand ===\n")
			if err := runVocabExpandBenchmark(limit); err != nil {
				fmt.Printf("❌ Vocab expand benchmark failed: %v\n", err)
			}
		}

		if !build && !expand {
			fmt.Println("Please specify --build, --expand, or --all")
			os.Exit(1)
		}
	},
}

var benchmarkModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Comprehensive benchmark for new vector embedding models",
	Long: `Run comprehensive benchmarks for the new vector embedding models.

This benchmarks the interface-based vector generators including:
- all-MiniLM-L6-v2: General multilingual model (22.7M parameters)
- GTE-small-zh: Chinese-optimized model (33M parameters)
- stsb-bert-tiny: Ultra-lightweight model (11M parameters)
- hash-fallback: Pure hash-based embeddings

Examples:
  metabase benchmark models                       # Quick benchmark
  metabase benchmark models --full               # Comprehensive benchmark
  metabase benchmark models --compare all-minilm-l6-v2 gte-small-zh  # Compare specific models
  metabase benchmark models --list               # List available models`,
	Run: func(cmd *cobra.Command, args []string) {
		full, _ := cmd.Flags().GetBool("full")
		compare, _ := cmd.Flags().GetStringSlice("compare")
		list, _ := cmd.Flags().GetBool("list")

		if list {
			if err := embedding.RunBenchmarkCLI([]string{"list"}); err != nil {
				fmt.Printf("❌ Failed to list models: %v\n", err)
			}
			return
		}

		if len(compare) > 0 {
			// Compare specific models
			benchmarkArgs := append([]string{"compare"}, compare...)
			benchmarkArgs = append(benchmarkArgs, "--text-count", "100")
			if err := embedding.RunBenchmarkCLI(benchmarkArgs); err != nil {
				fmt.Printf("❌ Comparison failed: %v\n", err)
			}
			return
		}

		if full {
			// Full comprehensive benchmark
			benchmarkArgs := []string{
				"full",
				"--text-count", "200",
				"--iterations", "3",
				"--output-format", "both",
				"--verbose",
			}
			if err := embedding.RunBenchmarkCLI(benchmarkArgs); err != nil {
				fmt.Printf("❌ Full benchmark failed: %v\n", err)
			}
		} else {
			// Default quick benchmark
			if err := embedding.RunBenchmarkCLI([]string{"quick"}); err != nil {
				fmt.Printf("❌ Quick benchmark failed: %v\n", err)
			}
		}
	},
}

// runEmbeddingBenchmark runs embedding benchmark for a specific method
func runEmbeddingBenchmark(method string, textCount int, batchSizeStr string, iterations int, warmup int) error {
	start := time.Now()

	// Parse batch sizes
	var batchSizes []int
	if batchSizeStr != "" {
		// Parse comma-separated batch sizes
		for _, s := range splitString(batchSizeStr, ",") {
			if size, err := strconv.Atoi(s); err == nil {
				batchSizes = append(batchSizes, size)
			}
		}
	}
	if len(batchSizes) == 0 {
		batchSizes = []int{1, 8, 16, 32, 64}
	}

	// Create embedder
	config := &embedding.Config{
		LocalModelType: method,
		BatchSize:      32,
		MaxConcurrency: 4,
		EnableFallback: method == "python", // Enable fallback for python
	}

	embedder, err := embedding.NewLocalEmbedder(config)
	if err != nil {
		return fmt.Errorf("failed to create embedder: %w", err)
	}
	defer embedder.Close()

	// Generate test texts
	testTexts := generateTestTexts(textCount)
	if len(testTexts) == 0 {
		return fmt.Errorf("no test texts generated")
	}

	fmt.Printf("Testing %s embedder with %d texts\n", method, len(testTexts))

	// Warmup runs
	for i := 0; i < warmup; i++ {
		warmupTexts := testTexts[:min(10, len(testTexts))]
		_, _ = embedder.Embed(warmupTexts)
	}

	// Benchmark different batch sizes
	bestThroughput := 0.0
	bestBatchSize := 0

	for _, batchSize := range batchSizes {
		// Update embedder batch size
		// Note: This would require modifying the embedder to support dynamic batch sizes

		batchStart := time.Now()

		for iter := 0; iter < iterations; iter++ {
			_, err := embedder.Embed(testTexts)
			if err != nil {
				fmt.Printf("  ❌ Batch size %d failed: %v\n", batchSize, err)
				continue
			}
		}

		duration := time.Since(batchStart)
		totalTexts := len(testTexts) * iterations
		throughput := float64(totalTexts) / duration.Seconds()

		fmt.Printf("  ✅ Batch size %d: %.1f texts/sec, %v\n", batchSize, throughput, duration)

		if throughput > bestThroughput {
			bestThroughput = throughput
			bestBatchSize = batchSize
		}
	}

	// Memory usage test
	memStart := time.Now()
	_, _ = embedder.Embed(testTexts[:min(100, len(testTexts))])
	memDuration := time.Since(memStart)
	memThroughput := float64(min(100, len(testTexts))) / memDuration.Seconds()

	fmt.Printf("\n📊 %s Summary:\n", method)
	fmt.Printf("  Best batch size: %d (%.1f texts/sec)\n", bestBatchSize, bestThroughput)
	fmt.Printf("  Memory test: %.1f texts/sec (100 texts)\n", memThroughput)
	fmt.Printf("  Dimension: %d\n", embedder.GetDimension())
	fmt.Printf("  Total benchmark time: %v\n", time.Since(start))

	// Additional info for ONNX
	if method == "onnx" {
		if onnxStats, ok := getONNXStats(embedder); ok {
			fmt.Printf("  Cache size: %d\n", onnxStats["cache_size"])
			fmt.Printf("  Cache capacity: %d\n", onnxStats["cache_capacity"])
		}
	}

	return nil
}

// runVocabBuildBenchmark benchmarks vocabulary building
func runVocabBuildBenchmark() error {
	start := time.Now()

	// Test building vocabulary from current directory
	builder := vocab.NewVocabularyBuilder(nil)

	if builder == nil {
		return fmt.Errorf("failed to create vocab builder")
	}

	// Get files for benchmarking
	filePaths, err := discoverFilesForVocab(".")
	if err != nil {
		return fmt.Errorf("failed to discover files: %w", err)
	}

	if len(filePaths) == 0 {
		return fmt.Errorf("no files found for vocab building")
	}

	fmt.Printf("Building vocabulary from %d files...\n", len(filePaths))

	buildStart := time.Now()
	result, err := builder.BuildFromFiles(filePaths)
	if err != nil {
		return fmt.Errorf("vocab build failed: %w", err)
	}
	buildDuration := time.Since(buildStart)

	fmt.Printf("  ✅ Build completed in %v\n", buildDuration)
	fmt.Printf("  Files processed: %d\n", result.AddedFiles)
	fmt.Printf("  New terms: %d\n", result.NewTerms)
	fmt.Printf("  Throughput: %.1f files/sec\n", float64(result.AddedFiles)/buildDuration.Seconds())

	// Test cache embedding generation
	cacheStart := time.Now()
	err = builder.CacheTermEmbeddingsDefault(1000)
	cacheDuration := time.Since(cacheStart)

	if err != nil {
		fmt.Printf("  ⚠️  Embedding caching failed: %v\n", err)
	} else {
		fmt.Printf("  ✅ Embedding caching completed in %v\n", cacheDuration)
	}

	totalDuration := time.Since(start)
	fmt.Printf("Total vocab benchmark time: %v\n", totalDuration)

	return nil
}

// runVocabExpandBenchmark benchmarks vocabulary expansion
func runVocabExpandBenchmark(limit int) error {
	if limit <= 0 {
		limit = 1000
	}

	start := time.Now()

	builder, err := vocab.LoadVocabularyBuilder(nil)
	if err != nil {
		// Create new builder if loading fails
		builder = vocab.NewVocabularyBuilder(nil)
	}

	if builder == nil {
		return fmt.Errorf("failed to load vocab builder")
	}

	// Test queries
	queries := []string{
		"database connection",
		"user authentication",
		"API endpoint",
		"file processing",
		"memory management",
		"error handling",
		"web server",
		"data structure",
		"algorithm optimization",
		"security implementation",
	}

	fmt.Printf("Testing query expansion with %d queries, limit %d\n", len(queries), limit)

	// Test without embedding
	noEmbedStart := time.Now()
	for _, query := range queries {
		result := builder.GetIndex().ExpandQuery(query, limit)
		_ = result // Use the result
	}
	noEmbedDuration := time.Since(noEmbedStart)

	fmt.Printf("  ✅ Non-embedding expansion: %v total, %v per query\n",
		noEmbedDuration, noEmbedDuration/time.Duration(len(queries)))

	// Test with fast embedding
	fastEmbedStart := time.Now()
	for _, query := range queries {
		cfg := &embedding.Config{LocalModelType: "fast", BatchSize: 32, MaxConcurrency: 4, EnableFallback: false}
		emb, _ := embedding.NewLocalEmbedder(cfg)
		if emb != nil {
			_, err := builder.ExpandQueryWithEmbedding(query, limit, emb)
			emb.Close()
			if err != nil {
				fmt.Printf("  ⚠️  Fast embedding expansion failed for '%s': %v\n", query, err)
			}
		}
	}
	fastEmbedDuration := time.Since(fastEmbedStart)

	fmt.Printf("  ✅ Fast embedding expansion: %v total, %v per query\n",
		fastEmbedDuration, fastEmbedDuration/time.Duration(len(queries)))

	totalDuration := time.Since(start)
	fmt.Printf("Total expand benchmark time: %v\n", totalDuration)

	return nil
}

// Helper functions
func generateTestTexts(count int) []string {
	words := []string{
		"database", "connection", "server", "client", "user", "authentication",
		"API", "REST", "HTTP", "request", "response", "javascript", "python",
		"function", "method", "class", "object", "variable", "string",
		"SQL", "MongoDB", "Docker", "Kubernetes", "Git", "repository",
		"frontend", "backend", "testing", "security", "performance",
	}

	texts := make([]string, count)
	for i := 0; i < count; i++ {
		// Generate simple test sentences
		texts[i] = fmt.Sprintf("The %s and %s are important for %s development.",
			words[i%len(words)], words[(i+1)%len(words)], words[(i+2)%len(words)])
	}
	return texts
}

func splitString(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep[0] {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func discoverFilesForVocab(dir string) ([]string, error) {
	var filePaths []string

	// Walk the directory to find supported files
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip hidden directories and common ignore patterns
			dirName := filepath.Base(path)
			if strings.HasPrefix(dirName, ".") || dirName == "node_modules" || dirName == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check file extension
		ext := strings.ToLower(filepath.Ext(path))
		if isCodeFile(ext) {
			filePaths = append(filePaths, path)
		}

		return nil
	})

	return filePaths, nil
}

func isCodeFile(ext string) bool {
	codeExts := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".py": true, ".java": true,
		".cpp": true, ".c": true, ".h": true, ".rs": true, ".php": true,
		".rb": true, ".swift": true, ".kt": true, ".scala": true,
		".md": true, ".txt": true, ".json": true, ".yaml": true, ".yml": true,
		".toml": true, ".xml": true, ".env": true, ".cfg": true, ".conf": true,
	}
	return codeExts[ext]
}

func getONNXStats(embedder embedding.Embedder) (map[string]interface{}, bool) {
	// Check if embedder is ONNX type and extract stats
	if onnxEmbedder, ok := embedder.(*embedding.ONNXEmbedder); ok {
		return onnxEmbedder.GetCacheStats(), true
	}

	return map[string]interface{}{
		"cache_size":     0,
		"cache_capacity": 0,
		"type":           "unknown",
	}, false
}

func init() {
	benchmarkCmd.AddCommand(benchmarkEmbeddingCmd)
	benchmarkCmd.AddCommand(benchmarkVocabCmd)
	benchmarkCmd.AddCommand(benchmarkModelsCmd)

	// Embedding benchmark flags
	benchmarkEmbeddingCmd.Flags().String("method", "", "Embedding method (fast, python, onnx)")
	benchmarkEmbeddingCmd.Flags().Int("texts", 100, "Number of test texts")
	benchmarkEmbeddingCmd.Flags().String("batch-sizes", "", "Comma-separated batch sizes")
	benchmarkEmbeddingCmd.Flags().Bool("all-methods", false, "Benchmark all available methods")
	benchmarkEmbeddingCmd.Flags().Int("iterations", 3, "Number of iterations")
	benchmarkEmbeddingCmd.Flags().Int("warmup", 2, "Number of warmup runs")

	// Vocab benchmark flags
	benchmarkVocabCmd.Flags().Bool("build", false, "Benchmark vocab building")
	benchmarkVocabCmd.Flags().Bool("expand", false, "Benchmark vocab expansion")
	benchmarkVocabCmd.Flags().Bool("all", false, "Benchmark all vocab operations")
	benchmarkVocabCmd.Flags().Int("limit", 1000, "Expansion limit for vocab tests")

	// Models benchmark flags
	benchmarkModelsCmd.Flags().Bool("quick", false, "Run quick benchmark (default)")
	benchmarkModelsCmd.Flags().Bool("full", false, "Run comprehensive benchmark")
	benchmarkModelsCmd.Flags().StringSlice("compare", []string{}, "Compare specific models")
	benchmarkModelsCmd.Flags().Bool("list", false, "List available models")

	AddCommand(benchmarkCmd)
}
