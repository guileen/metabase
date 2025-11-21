package embedding

import ("context"
	"fmt"
	"runtime"
	"strings"
	"time")

// CybertronComparisonResult holds comparison results between Python and Cybertron implementations
type CybertronComparisonResult struct {
	Models       map[string]PerformanceMetrics `json:"models"`
	SpeedupRatio map[string]float64             `json:"speedup_ratios"`
	MemorySaving map[string]float64             `json:"memory_saving"`
	GeneratedAt  time.Time                      `json:"generated_at"`
	Recomendations []string                     `json:"recommendations"`
}

// ComparePythonVsCybertronPerformance performs comprehensive comparison
func ComparePythonVsCybertronPerformance(ctx context.Context, testTexts []string) (*CybertronComparisonResult, error) {
	if len(testTexts) == 0 {
		testTexts = []string{
			"Hello world", "你好世界", "Machine learning is fascinating", "机器学习很有趣",
			"Vector embeddings represent text as numerical vectors", "向量嵌入将文本表示为数值向量",
			"Deep learning models require large amounts of data", "深度学习模型需要大量数据",
			"Natural language processing helps computers understand text", "自然语言处理帮助计算机理解文本",
		}
	}

	result := &CybertronComparisonResult{
		Models:        make(map[string]PerformanceMetrics),
		SpeedupRatio:  make(map[string]float64),
		MemorySaving:  make(map[string]float64),
		GeneratedAt:   time.Now(),
	}

	// Models to compare
	comparisons := []struct {
		pythonModel    string
		cybertronModel string
		modelName      string
	}{
		{"all-minilm-l6-v2", "all-minilm-l6-v2-cybertron", "MiniLM-L6-v2"},
		{"stsb-bert-tiny", "stsb-bert-tiny-cybertron", "STS-Bert-Tiny"},
	}

	registry := GetDefaultRegistry()

	for _, comp := range comparisons {
		fmt.Printf("🔍 Comparing %s implementations...\n", comp.modelName)

		// Benchmark Python version
		pythonGen, err := registry.Get(comp.pythonModel, VectorGeneratorConfig{
			BatchSize:      32,
			MaxConcurrency: 4,
			EnableFallback: false,
			Timeout:        time.Minute * 2,
		})
		if err != nil {
			fmt.Printf("❌ Failed to create Python %s: %v\n", comp.pythonModel, err)
			continue
		}

		pythonMetrics, err := benchmarkSingleModel(ctx, pythonGen, testTexts)
		if err != nil {
			fmt.Printf("❌ Failed to benchmark Python %s: %v\n", comp.pythonModel, err)
			pythonGen.Close()
			continue
		}
		pythonGen.Close()

		// Benchmark Cybertron version
		cybertronGen, err := registry.Get(comp.cybertronModel, VectorGeneratorConfig{
			BatchSize:      32,
			MaxConcurrency: 4,
			EnableFallback: false,
			Timeout:        time.Minute * 2,
		})
		if err != nil {
			fmt.Printf("❌ Failed to create Cybertron %s: %v\n", comp.cybertronModel, err)
			result.Models[comp.pythonModel] = *pythonMetrics
			continue
		}

		cybertronMetrics, err := benchmarkSingleModel(ctx, cybertronGen, testTexts)
		if err != nil {
			fmt.Printf("❌ Failed to benchmark Cybertron %s: %v\n", comp.cybertronModel, err)
			cybertronGen.Close()
			result.Models[comp.pythonModel] = *pythonMetrics
			result.Models[comp.cybertronModel] = *cybertronMetrics
			continue
		}
		cybertronGen.Close()

		// Store results
		result.Models[comp.pythonModel] = *pythonMetrics
		result.Models[comp.cybertronModel] = *cybertronMetrics

		// Calculate improvements
		if pythonMetrics.LatencyMs > 0 {
			result.SpeedupRatio[comp.modelName] = pythonMetrics.LatencyMs / cybertronMetrics.LatencyMs
		}
		if pythonMetrics.MemoryUsageMB > 0 {
			result.MemorySaving[comp.modelName] = (pythonMetrics.MemoryUsageMB - cybertronMetrics.MemoryUsageMB) / pythonMetrics.MemoryUsageMB * 100
		}

		fmt.Printf("✅ %s comparison completed\n", comp.modelName)
		fmt.Printf("   Python: %.2fms, %.1f MB, %.1f QPS\n",
			pythonMetrics.LatencyMs, pythonMetrics.MemoryUsageMB, pythonMetrics.ThroughputQPS)
		fmt.Printf("   Cybertron: %.2fms, %.1f MB, %.1f QPS\n",
			cybertronMetrics.LatencyMs, cybertronMetrics.MemoryUsageMB, cybertronMetrics.ThroughputQPS)
		fmt.Printf("   Speedup: %.2fx, Memory saving: %.1f%%\n",
			result.SpeedupRatio[comp.modelName], result.MemorySaving[comp.modelName])
		fmt.Println()
	}

	// Generate recommendations
	result.Recomendations = generateCybertronRecommendations(result)

	return result, nil
}

// benchmarkSingleModel benchmarks a single model implementation
func benchmarkSingleModel(ctx context.Context, gen VectorGenerator, testTexts []string) (*PerformanceMetrics, error) {
	// Record initial memory usage
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	start := time.Now()

	// Test single embedding
	_, err := gen.EmbedSingle(ctx, testTexts[0])
	if err != nil {
		return nil, fmt.Errorf("single embedding failed: %w", err)
	}

	// Test batch embeddings multiple times
	iterations := 3
	var totalLatency float64
	var totalThroughput float64

	for i := 0; i < iterations; i++ {
		iterStart := time.Now()
		_, err := gen.Embed(ctx, testTexts)
		iterTime := time.Since(iterStart)

		if err != nil {
			return nil, fmt.Errorf("batch embedding %d failed: %w", i, err)
		}

		avgLatency := float64(iterTime.Nanoseconds()) / float64(len(testTexts)) / 1e6
		throughput := float64(len(testTexts)) / iterTime.Seconds()

		totalLatency += avgLatency
		totalThroughput += throughput
	}

	modelLoadTime := time.Since(start)
	avgLatencyMs := totalLatency / float64(iterations)
	avgThroughput := totalThroughput / float64(iterations)

	// Record final memory usage
	runtime.ReadMemStats(&m2)
	memoryUsageMB := float64(m2.Alloc-m1.Alloc) / 1024 / 1024

	return &PerformanceMetrics{
		ModelName:         gen.GetModelName(),
		Dimension:         gen.GetDimension(),
		LatencyMs:         avgLatencyMs,
		ThroughputQPS:     avgThroughput,
		MemoryUsageMB:     memoryUsageMB,
		QualityScore:      estimateQualityScore(gen),
		CPUUsagePercent:   estimateCPUUsage(gen, testTexts),
		ModelLoadTimeMs:   float64(modelLoadTime.Nanoseconds()) / 1e6,
		TestTextCount:     len(testTexts),
		TestDate:          time.Now().Format(time.RFC3339),
	}, nil
}

// generateCybertronRecommendations generates recommendations based on comparison results
func generateCybertronRecommendations(result *CybertronComparisonResult) []string {
	recommendations := make([]string, 0)

	// Overall analysis
	hasCybertronModels := false
	for modelName := range result.Models {
		if len(modelName) > 10 && modelName[len(modelName)-10:] == "-cybertron" {
			hasCybertronModels = true
			break
		}
	}

	if !hasCybertronModels {
		recommendations = append(recommendations, "Cybertron models could not be loaded. Consider installing Cybertron dependencies.")
		return recommendations
	}

	// Speed-based recommendations
	fastestSpeedup := 0.0
	fastestModel := ""

	for model, speedup := range result.SpeedupRatio {
		if speedup > fastestSpeedup {
			fastestSpeedup = speedup
			fastestModel = model
		}
	}

	if fastestSpeedup > 2.0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Cybertron %s is %.1fx faster than Python version. Use Cybertron for high-throughput applications.",
				fastestModel, fastestSpeedup))
	}

	// Memory-based recommendations
	maxMemorySaving := 0.0
	memoryEfficientModel := ""

	for model, saving := range result.MemorySaving {
		if saving > maxMemorySaving {
			maxMemorySaving = saving
			memoryEfficientModel = model
		}
	}

	if maxMemorySaving > 20.0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Cybertron %s uses %.1f%% less memory. Ideal for memory-constrained environments.",
				memoryEfficientModel, maxMemorySaving))
	}

	// General recommendations
	recommendations = append(recommendations,
		"Cybertron provides pure Go implementation without Python dependencies.",
		"Use Cybertron models for production deployments to reduce external dependencies.",
		"Python models may provide better compatibility with the latest HuggingFace models.")

	return recommendations
}

// PrintCybertronComparison prints a formatted comparison table
func PrintCybertronComparison(result *CybertronComparisonResult) {
	fmt.Printf("🚀 Python vs Cybertron Performance Comparison\n")
	fmt.Printf("%s\n\n", strings.Repeat("=", 50))

	if len(result.Models) == 0 {
		fmt.Println("No comparison data available")
		return
	}

	// Group results by model type
	modelTypes := map[string][]string{
		"MiniLM-L6-v2": {"all-minilm-l6-v2", "all-minilm-l6-v2-cybertron"},
		"STS-Bert-Tiny": {"stsb-bert-tiny", "stsb-bert-tiny-cybertron"},
	}

	for modelType, models := range modelTypes {
		fmt.Printf("📊 %s Comparison:\n", modelType)
		fmt.Printf("%-25s %-12s %-15s %-12s %-10s\n", "Implementation", "Latency(ms)", "Throughput(QPS)", "Memory(MB)", "Score")
		fmt.Println(strings.Repeat("-", 75))

		for _, modelName := range models {
			if metrics, exists := result.Models[modelName]; exists {
				implementation := "Python"
				if strings.HasSuffix(modelName, "-cybertron") {
					implementation = "Cybertron"
				}
				fmt.Printf("%-25s %-12.2f %-15.1f %-12.1f %-10.2f\n",
					implementation, metrics.LatencyMs, metrics.ThroughputQPS, metrics.MemoryUsageMB, metrics.QualityScore)
			}
		}

		// Show improvements
		if speedup, exists := result.SpeedupRatio[modelType]; exists {
			fmt.Printf("🚀 Cybertron Speedup: %.2fx\n", speedup)
		}
		if saving, exists := result.MemorySaving[modelType]; exists {
			fmt.Printf("💾 Memory Saving: %.1f%%\n", saving)
		}
		fmt.Println()
	}

	// Recommendations
	fmt.Println("💡 Recommendations:")
	for i, rec := range result.Recomendations {
		fmt.Printf("%d. %s\n", i+1, rec)
	}
}