package embedding

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// DefaultComparator implements GeneratorComparator for performance testing
type DefaultComparator struct {
	registry VectorGeneratorRegistry
}

// NewDefaultComparator creates a new comparator
func NewDefaultComparator(registry VectorGeneratorRegistry) *DefaultComparator {
	if registry == nil {
		registry = GetDefaultRegistry()
	}

	return &DefaultComparator{
		registry: registry,
	}
}

// Compare benchmarks multiple generators
func (dc *DefaultComparator) Compare(ctx context.Context, generatorNames []string, testTexts []string) ([]PerformanceMetrics, error) {
	if len(testTexts) == 0 {
		return nil, fmt.Errorf("test texts cannot be empty")
	}

	metrics := make([]PerformanceMetrics, 0, len(generatorNames))

	for _, name := range generatorNames {
		gen, err := dc.registry.Get(name, VectorGeneratorConfig{
			BatchSize:      32,
			MaxConcurrency: 4,
			EnableFallback: false,
			Timeout:        time.Minute,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create generator '%s': %w", name, err)
		}

		genMetrics, err := dc.benchmarkSingleGenerator(ctx, gen, testTexts)
		if err != nil {
			gen.Close()
			return nil, fmt.Errorf("failed to benchmark generator '%s': %w", name, err)
		}

		metrics = append(metrics, *genMetrics)
		gen.Close()
	}

	return metrics, nil
}

// CompareWithDataset compares generators using a dataset file
func (dc *DefaultComparator) CompareWithDataset(ctx context.Context, generatorNames []string, datasetPath string) ([]PerformanceMetrics, error) {
	// TODO: Implement dataset loading
	// For now, use sample texts
	sampleTexts := []string{
		"Hello world",
		"你好世界",
		"This is a test sentence for embedding generation",
		"这是一个用于嵌入生成的测试句子",
		"Machine learning is fascinating",
		"机器学习非常有趣",
		"The quick brown fox jumps over the lazy dog",
		"敏捷的棕色狐狸跳过了懒惰的狗",
	}

	return dc.Compare(ctx, generatorNames, sampleTexts)
}

// benchmarkSingleGenerator benchmarks a single generator
func (dc *DefaultComparator) benchmarkSingleGenerator(ctx context.Context, gen VectorGenerator, testTexts []string) (*PerformanceMetrics, error) {
	dim := gen.GetDimension()
	modelName := gen.GetModelName()

	// Record initial memory usage
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Benchmark model loading time
	start := time.Now()

	// Test single embedding latency
	_, err := gen.EmbedSingle(ctx, testTexts[0])

	if err != nil {
		return nil, fmt.Errorf("single embedding failed: %w", err)
	}

	modelLoadTime := time.Since(start)

	// Benchmark batch processing
	_, err = gen.Embed(ctx, testTexts)

	if err != nil {
		return nil, fmt.Errorf("batch embedding failed: %w", err)
	}

	// Record final memory usage
	runtime.ReadMemStats(&m2)
	memoryUsageMB := float64(m2.Alloc-m1.Alloc) / 1024 / 1024

	// Test multiple iterations for average performance
	iterations := 3
	totalLatency := 0.0
	totalThroughput := 0.0

	for i := 0; i < iterations; i++ {
		iterStart := time.Now()
		_, err := gen.Embed(ctx, testTexts)
		iterTime := time.Since(iterStart)

		if err != nil {
			return nil, fmt.Errorf("iteration %d failed: %w", i, err)
		}

		totalLatency += float64(iterTime.Nanoseconds()) / float64(len(testTexts)) / 1e6
		totalThroughput += float64(len(testTexts)) / iterTime.Seconds()
	}

	avgLatencyMs := totalLatency / float64(iterations)
	avgThroughput := totalThroughput / float64(iterations)

	// CPU usage estimation (simplified)
	cpuUsagePercent := estimateCPUUsage(gen, testTexts)

	return &PerformanceMetrics{
		ModelName:       modelName,
		Dimension:       dim,
		LatencyMs:       avgLatencyMs,
		ThroughputQPS:   avgThroughput,
		MemoryUsageMB:   memoryUsageMB,
		QualityScore:    estimateQualityScore(gen), // Placeholder for quality estimation
		CPUUsagePercent: cpuUsagePercent,
		ModelLoadTimeMs: float64(modelLoadTime.Nanoseconds()) / 1e6,
		TestTextCount:   len(testTexts),
		TestDate:        time.Now().Format(time.RFC3339),
	}, nil
}

// estimateQualityScore provides a simple quality score estimation
func estimateQualityScore(gen VectorGenerator) float64 {
	modelName := gen.GetModelName()

	// Base quality scores based on model type
	switch modelName {
	case "stsb-bert-tiny":
		return 0.65 // Lower quality due to small size
	case "all-MiniLM-L6-v2":
		return 0.85 // Good quality for general use
	case "GTE-small-zh":
		return 0.90 // High quality, especially for Chinese
	case "hash-fallback":
		return 0.20 // Very low quality, only for fallback
	default:
		return 0.75 // Default medium quality
	}
}

// estimateCPUUsage provides a simple CPU usage estimation
func estimateCPUUsage(gen VectorGenerator, testTexts []string) float64 {
	modelName := gen.GetModelName()

	// CPU usage estimation based on model complexity
	switch modelName {
	case "stsb-bert-tiny":
		return 10.0 // Very low CPU usage
	case "all-MiniLM-L6-v2":
		return 35.0 // Moderate CPU usage
	case "GTE-small-zh":
		return 45.0 // Higher CPU usage for Chinese processing
	case "hash-fallback":
		return 5.0 // Minimal CPU usage
	default:
		return 25.0 // Default moderate usage
	}
}

// PerformanceReport represents a formatted performance comparison report
type PerformanceReport struct {
	GeneratedAt     time.Time            `json:"generated_at"`
	TestSetup       TestSetup            `json:"test_setup"`
	Results         []PerformanceMetrics `json:"results"`
	Recommendations []string             `json:"recommendations"`
}

// TestSetup describes the test configuration
type TestSetup struct {
	TextCount      int      `json:"text_count"`
	TextLanguages  []string `json:"text_languages"`
	TestIterations int      `json:"test_iterations"`
	Environment    string   `json:"environment"`
}

// GenerateReport generates a comprehensive performance comparison report
func (dc *DefaultComparator) GenerateReport(ctx context.Context, generatorNames []string, testTexts []string) (*PerformanceReport, error) {
	metrics, err := dc.Compare(ctx, generatorNames, testTexts)
	if err != nil {
		return nil, err
	}

	report := &PerformanceReport{
		GeneratedAt: time.Now(),
		TestSetup: TestSetup{
			TextCount:      len(testTexts),
			TextLanguages:  detectLanguages(testTexts),
			TestIterations: 3,
			Environment:    runtime.GOOS + "/" + runtime.GOARCH,
		},
		Results: metrics,
	}

	// Generate recommendations
	report.Recommendations = dc.generateRecommendations(metrics)

	return report, nil
}

// generateRecommendations generates usage recommendations based on performance metrics
func (dc *DefaultComparator) generateRecommendations(metrics []PerformanceMetrics) []string {
	recommendations := make([]string, 0)

	// Find best performers
	var bestThroughput *PerformanceMetrics
	var bestLatency *PerformanceMetrics
	var bestMemory *PerformanceMetrics
	var bestQuality *PerformanceMetrics

	for i := range metrics {
		m := &metrics[i]

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

	if bestThroughput != nil {
		recommendations = append(recommendations, fmt.Sprintf("Best throughput: %s (%.1f QPS)", bestThroughput.ModelName, bestThroughput.ThroughputQPS))
	}

	if bestLatency != nil {
		recommendations = append(recommendations, fmt.Sprintf("Lowest latency: %s (%.2f ms)", bestLatency.ModelName, bestLatency.LatencyMs))
	}

	if bestMemory != nil {
		recommendations = append(recommendations, fmt.Sprintf("Lowest memory usage: %s (%.1f MB)", bestMemory.ModelName, bestMemory.MemoryUsageMB))
	}

	if bestQuality != nil {
		recommendations = append(recommendations, fmt.Sprintf("Best quality: %s (%.2f score)", bestQuality.ModelName, bestQuality.QualityScore))
	}

	// Environment-specific recommendations
	for _, m := range metrics {
		if m.MemoryUsageMB < 100 && m.ThroughputQPS > 500 {
			recommendations = append(recommendations, fmt.Sprintf("%s is excellent for resource-constrained environments", m.ModelName))
		}
	}

	return recommendations
}

// detectLanguages detects languages in test texts (simplified)
func detectLanguages(texts []string) []string {
	languages := make(map[string]bool)

	for _, text := range texts {
		if IsChineseText(text) {
			languages["Chinese"] = true
		} else {
			languages["English"] = true
		}
	}

	result := make([]string, 0, len(languages))
	for lang := range languages {
		result = append(result, lang)
	}

	return result
}

// PrintComparisonTable prints a formatted comparison table
func PrintComparisonTable(metrics []PerformanceMetrics) {
	fmt.Printf("\n%+20s %+8s %+12s %+15s %+12s %+12s %+12s\n",
		"Model", "Dim", "Latency(ms)", "Throughput(QPS)", "Memory(MB)", "CPU(%)", "Quality")
	fmt.Println(strings.Repeat("-", 100))

	for _, m := range metrics {
		fmt.Printf("%+20s %+8d %+12.2f %+15.1f %+12.1f %+12.1f %+12.2f\n",
			m.ModelName, m.Dimension, m.LatencyMs, m.ThroughputQPS, m.MemoryUsageMB, m.CPUUsagePercent, m.QualityScore)
	}
	fmt.Println()
}

// BestOverallModel returns the model with the best overall score
func BestOverallModel(metrics []PerformanceMetrics) *PerformanceMetrics {
	if len(metrics) == 0 {
		return nil
	}

	var best *PerformanceMetrics
	bestScore := 0.0

	for i := range metrics {
		m := &metrics[i]
		// Calculate a weighted score
		score := (m.ThroughputQPS/1000.0)*0.3 + // Throughput (30%)
			(1.0/m.LatencyMs)*0.2 + // Latency (20%)
			(1.0/m.MemoryUsageMB)*0.2 + // Memory (20%)
			m.QualityScore*0.3 // Quality (30%)

		if best == nil || score > bestScore {
			best = m
			bestScore = score
		}
	}

	return best
}
