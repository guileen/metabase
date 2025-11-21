package embedding

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

// BenchmarkSuite provides comprehensive benchmarking functionality
type BenchmarkSuite struct {
	registry VectorGeneratorRegistry
	comparator GeneratorComparator
}

// NewBenchmarkSuite creates a new benchmark suite
func NewBenchmarkSuite() *BenchmarkSuite {
	registry := GetDefaultRegistry()
	return &BenchmarkSuite{
		registry: registry,
		comparator: NewDefaultComparator(registry),
	}
}

// BenchmarkConfig holds configuration for benchmark runs
type BenchmarkConfig struct {
	// Which models to benchmark
	Models []string

	// Test dataset configuration
	TestTexts     []string
	DatasetPath   string
	TextCount     int
	TextLanguages []string

	// Performance test configuration
	Iterations    int
	EnableMemory  bool
	EnableCPU     bool
	Timeout       time.Duration

	// Output configuration
	OutputFormat string // "json", "table", "both"
	OutputPath   string
	Verbose      bool
}

// BenchmarkResult contains detailed benchmark results
type BenchmarkResult struct {
	Summary    PerformanceSummary      `json:"summary"`
	Models     []PerformanceMetrics    `json:"models"`
	Comparison ComparisonDetails       `json:"comparison"`
	GeneratedAt time.Time              `json:"generated_at"`
}

// PerformanceSummary provides a high-level summary
type PerformanceSummary struct {
	TotalModels    int     `json:"total_models"`
	TestTextCount  int     `json:"test_text_count"`
	AvgLatency     float64 `json:"avg_latency_ms"`
	AvgThroughput  float64 `json:"avg_throughput_qps"`
	AvgMemory      float64 `json:"avg_memory_mb"`
	BestThroughput string  `json:"best_throughput_model"`
	BestLatency    string  `json:"best_latency_model"`
	BestMemory     string  `json:"best_memory_model"`
	BestQuality    string  `json:"best_quality_model"`
}

// ComparisonDetails provides detailed comparison metrics
type ComparisonDetails struct {
	SpeedRatios    map[string]float64 `json:"speed_ratios"`
	MemoryRatios   map[string]float64 `json:"memory_ratios"`
	QualityScores  map[string]float64 `json:"quality_scores"`
	Recommendations []string          `json:"recommendations"`
}

// RunBenchmark executes a comprehensive benchmark
func (bs *BenchmarkSuite) RunBenchmark(config BenchmarkConfig) (*BenchmarkResult, error) {
	// Prepare test texts
	testTexts, err := bs.prepareTestTexts(config)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare test texts: %w", err)
	}

	// Determine which models to test
	models := config.Models
	if len(models) == 0 {
		models = bs.registry.List()
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// Run performance comparison
	metrics, err := bs.comparator.Compare(ctx, models, testTexts)
	if err != nil {
		return nil, fmt.Errorf("failed to run comparison: %w", err)
	}

	// Generate detailed results
	result := &BenchmarkResult{
		Models:     metrics,
		GeneratedAt: time.Now(),
	}

	// Calculate summary statistics
	result.Summary = bs.calculateSummary(metrics, testTexts)

	// Calculate detailed comparisons
	result.Comparison = bs.calculateDetailedComparison(metrics)

	// Output results if requested
	if err := bs.outputResults(result, config); err != nil {
		return nil, fmt.Errorf("failed to output results: %w", err)
	}

	return result, nil
}

// prepareTestTexts prepares the test texts for benchmarking
func (bs *BenchmarkSuite) prepareTestTexts(config BenchmarkConfig) ([]string, error) {
	// Use provided texts if available
	if len(config.TestTexts) > 0 {
		return config.TestTexts, nil
	}

	// Load from dataset file if specified
	if config.DatasetPath != "" {
		return bs.loadDataset(config.DatasetPath)
	}

	// Generate default test texts
	return bs.generateDefaultTexts(config)
}

// loadDataset loads texts from a dataset file
func (bs *BenchmarkSuite) loadDataset(datasetPath string) ([]string, error) {
	// TODO: Implement dataset loading from JSON, CSV, etc.
	// For now, return default texts
	return bs.generateDefaultTexts(BenchmarkConfig{})
}

// generateDefaultTexts generates default test texts
func (bs *BenchmarkSuite) generateDefaultTexts(config BenchmarkConfig) ([]string, error) {
	textCount := config.TextCount
	if textCount == 0 {
		textCount = 50
	}

	baseTexts := []string{
		// English texts
		"Hello world",
		"Machine learning is a fascinating field",
		"The quick brown fox jumps over the lazy dog",
		"Artificial intelligence will change the world",
		"Natural language processing helps computers understand text",
		"Deep learning models require large amounts of data",
		"Vector embeddings represent text as numerical vectors",
		"Semantic search finds documents based on meaning",
		"Transformer models have revolutionized NLP",
		"Attention mechanisms help models focus on relevant parts",
		// Chinese texts
		"你好世界",
		"机器学习是一个非常有趣的领域",
		"人工智能将改变世界",
		"自然语言处理帮助计算机理解文本",
		"深度学习模型需要大量数据",
		"向量嵌入将文本表示为数值向量",
		"语义搜索根据含义查找文档",
		"Transformer模型彻底改变了自然语言处理",
		"注意力机制帮助模型关注相关部分",
		"语义相似度计算是许多应用的基础",
		// Mixed short texts
		"Hello",
		"Hi",
		"Thanks",
		"Goodbye",
		"Please help me",
		"How are you",
		"What's the weather",
		"Best practices",
		"你好",
		"谢谢",
		"再见",
		"请帮助我",
		"你好吗",
		"天气怎么样",
		"最佳实践",
	}

	// Expand to requested count
	texts := make([]string, 0, textCount)
	for i := 0; i < textCount; i++ {
		texts = append(texts, baseTexts[i%len(baseTexts)])
	}

	return texts, nil
}

// calculateSummary calculates summary statistics
func (bs *BenchmarkSuite) calculateSummary(metrics []PerformanceMetrics, testTexts []string) PerformanceSummary {
	if len(metrics) == 0 {
		return PerformanceSummary{}
	}

	var totalLatency, totalThroughput, totalMemory float64
	var bestThroughput, bestLatency, bestMemory, bestQuality *PerformanceMetrics

	for i := range metrics {
		m := &metrics[i]

		totalLatency += m.LatencyMs
		totalThroughput += m.ThroughputQPS
		totalMemory += m.MemoryUsageMB

		if bestThroughput == nil || m.ThroughputQPS > bestThroughput.ThroughputQPS {
			bestThroughput = m
		}

		if bestLatency == nil || m.LatencyMs < bestLatency.LatencyMs {
			bestLatency = m
		}

		if bestMemory == nil || m.MemoryUsageMB < bestMemory.MemoryUsageMB {
			bestMemory = m
		}

		if bestQuality == nil || m.QualityScore > bestQuality.QualityScore {
			bestQuality = m
		}
	}

	return PerformanceSummary{
		TotalModels:    len(metrics),
		TestTextCount:  len(testTexts),
		AvgLatency:     totalLatency / float64(len(metrics)),
		AvgThroughput:  totalThroughput / float64(len(metrics)),
		AvgMemory:      totalMemory / float64(len(metrics)),
		BestThroughput: bestThroughput.ModelName,
		BestLatency:    bestLatency.ModelName,
		BestMemory:     bestMemory.ModelName,
		BestQuality:    bestQuality.ModelName,
	}
}

// calculateDetailedComparison calculates detailed comparison metrics
func (bs *BenchmarkSuite) calculateDetailedComparison(metrics []PerformanceMetrics) ComparisonDetails {
	speedRatios := make(map[string]float64)
	memoryRatios := make(map[string]float64)
	qualityScores := make(map[string]float64)

	if len(metrics) == 0 {
		return ComparisonDetails{}
	}

	// Find baseline (use stsb-bert-tiny as baseline if available, otherwise first model)
	baselineLatency := metrics[0].LatencyMs
	baselineMemory := metrics[0].MemoryUsageMB

	for _, m := range metrics {
		speedRatios[m.ModelName] = baselineLatency / m.LatencyMs
		memoryRatios[m.ModelName] = baselineMemory / m.MemoryUsageMB
		qualityScores[m.ModelName] = m.QualityScore
	}

	// Generate recommendations
	recommendations := bs.generateRecommendations(metrics)

	return ComparisonDetails{
		SpeedRatios:    speedRatios,
		MemoryRatios:   memoryRatios,
		QualityScores:  qualityScores,
		Recommendations: recommendations,
	}
}

// generateRecommendations generates usage recommendations
func (bs *BenchmarkSuite) generateRecommendations(metrics []PerformanceMetrics) []string {
	recommendations := make([]string, 0)

	if len(metrics) == 0 {
		return recommendations
	}

	// Sort by different criteria
	sortedByThroughput := make([]PerformanceMetrics, len(metrics))
	copy(sortedByThroughput, metrics)
	sort.Slice(sortedByThroughput, func(i, j int) bool {
		return sortedByThroughput[i].ThroughputQPS > sortedByThroughput[j].ThroughputQPS
	})

	sortedByLatency := make([]PerformanceMetrics, len(metrics))
	copy(sortedByLatency, metrics)
	sort.Slice(sortedByLatency, func(i, j int) bool {
		return sortedByLatency[i].LatencyMs < sortedByLatency[j].LatencyMs
	})

	sortedByMemory := make([]PerformanceMetrics, len(metrics))
	copy(sortedByMemory, metrics)
	sort.Slice(sortedByMemory, func(i, j int) bool {
		return sortedByMemory[i].MemoryUsageMB < sortedByMemory[j].MemoryUsageMB
	})

	// Generate specific recommendations
	if len(sortedByThroughput) > 0 && sortedByThroughput[0].ThroughputQPS > 500 {
		recommendations = append(recommendations, fmt.Sprintf("For high-throughput applications: %s (%.1f QPS)",
			sortedByThroughput[0].ModelName, sortedByThroughput[0].ThroughputQPS))
	}

	if len(sortedByLatency) > 0 && sortedByLatency[0].LatencyMs < 2.0 {
		recommendations = append(recommendations, fmt.Sprintf("For low-latency applications: %s (%.2f ms)",
			sortedByLatency[0].ModelName, sortedByLatency[0].LatencyMs))
	}

	if len(sortedByMemory) > 0 && sortedByMemory[0].MemoryUsageMB < 50 {
		recommendations = append(recommendations, fmt.Sprintf("For resource-constrained environments: %s (%.1f MB)",
			sortedByMemory[0].ModelName, sortedByMemory[0].MemoryUsageMB))
	}

	// Chinese-specific recommendations
	hasChinese := false
	highestQuality := 0.0

	for _, m := range metrics {
		if m.QualityScore > highestQuality {
			highestQuality = m.QualityScore
		}

		if m.ModelName == "gte-small-zh" {
			hasChinese = true
		}
	}

	if hasChinese {
		recommendations = append(recommendations, "For Chinese-heavy workloads: GTE-small-zh (optimized for Chinese text)")
	}

	// Best overall recommendation
	if bestOverallModel := BestOverallModel(metrics); bestOverallModel != nil {
		recommendations = append(recommendations, fmt.Sprintf("Best overall balance: %s", bestOverallModel.ModelName))
	}

	return recommendations
}

// outputResults outputs the benchmark results
func (bs *BenchmarkSuite) outputResults(result *BenchmarkResult, config BenchmarkConfig) error {
	switch config.OutputFormat {
	case "json":
		return bs.outputJSON(result, config)
	case "table":
		return bs.outputTable(result, config)
	case "both":
		if err := bs.outputTable(result, config); err != nil {
			return err
		}
		return bs.outputJSON(result, config)
	default:
		return bs.outputTable(result, config)
	}
}

// outputJSON outputs results in JSON format
func (bs *BenchmarkSuite) outputJSON(result *BenchmarkResult, config BenchmarkConfig) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if config.OutputPath != "" {
		return os.WriteFile(config.OutputPath, data, 0644)
	}

	fmt.Printf("Benchmark Results (JSON):\n%s\n\n", string(data))
	return nil
}

// outputTable outputs results in table format
func (bs *BenchmarkSuite) outputTable(result *BenchmarkResult, config BenchmarkConfig) error {
	fmt.Printf("=== Vector Embedding Benchmark Results ===\n")
	fmt.Printf("Generated: %s\n", result.GeneratedAt.Format(time.RFC3339))
	fmt.Printf("Test Texts: %d\n", result.Summary.TestTextCount)
	fmt.Printf("Models Tested: %d\n\n", result.Summary.TotalModels)

	// Summary table
	fmt.Printf("Summary:\n")
	fmt.Printf("  Average Latency: %.2f ms\n", result.Summary.AvgLatency)
	fmt.Printf("  Average Throughput: %.1f QPS\n", result.Summary.AvgThroughput)
	fmt.Printf("  Average Memory Usage: %.1f MB\n\n", result.Summary.AvgMemory)

	// Detailed results table
	fmt.Printf("Detailed Results:\n")
	PrintComparisonTable(result.Models)

	// Recommendations
	if len(result.Comparison.Recommendations) > 0 {
		fmt.Printf("Recommendations:\n")
		for i, rec := range result.Comparison.Recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec)
		}
		fmt.Println()
	}

	return nil
}

// RunQuickBenchmark provides a quick benchmark with default settings
func (bs *BenchmarkSuite) RunQuickBenchmark() (*BenchmarkResult, error) {
	config := BenchmarkConfig{
		Models:       []string{"all-minilm-l6-v2", "gte-small-zh", "stsb-bert-tiny", "hash-fallback"},
		TextCount:    20,
		Iterations:   3,
		Timeout:      time.Minute * 5,
		OutputFormat: "table",
		Verbose:      true,
	}

	return bs.RunBenchmark(config)
}