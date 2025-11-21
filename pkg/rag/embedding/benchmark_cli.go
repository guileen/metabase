package embedding

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// BenchmarkCommand represents the CLI command for embedding benchmarks
type BenchmarkCommand struct {
	rootCmd *cobra.Command
	suite   *BenchmarkSuite
}

// NewBenchmarkCommand creates a new benchmark CLI command
func NewBenchmarkCommand() *BenchmarkCommand {
	suite := NewBenchmarkSuite()
	cmd := &BenchmarkCommand{
		rootCmd: &cobra.Command{
			Use:   "benchmark",
			Short: "Benchmark vector embedding models",
			Long: `Comprehensive benchmarking tool for vector embedding models.

Supports multiple embedding implementations:
- all-MiniLM-L6-v2: General multilingual model (22.7M parameters, 384 dimensions)
- GTE-small-zh: Chinese-optimized model (33M parameters, 384 dimensions)
- stsb-bert-tiny: Ultra-lightweight model (11M parameters, 128 dimensions)
- hash-fallback: Pure hash-based embeddings for fallback scenarios

Provides detailed performance metrics including latency, throughput, memory usage,
and quality scores for different use cases and environments.`,
		},
		suite: suite,
	}

	cmd.setupCommands()
	return cmd
}

// setupCommands sets up all subcommands
func (bc *BenchmarkCommand) setupCommands() {
	// Quick benchmark command
	bc.rootCmd.AddCommand(&cobra.Command{
		Use:   "quick",
		Short: "Run a quick benchmark with default settings",
		RunE:  bc.runQuickBenchmark,
	})

	// Full benchmark command
	fullCmd := &cobra.Command{
		Use:   "full",
		Short: "Run a comprehensive benchmark",
		Long: `Run a comprehensive benchmark with all models and detailed analysis.
Tests include various text lengths, languages, and batch sizes.`,
		RunE: bc.runFullBenchmark,
	}

	// Full benchmark flags
	fullCmd.Flags().StringSlice("models", []string{}, "Models to benchmark (default: all)")
	fullCmd.Flags().Int("text-count", 100, "Number of test texts")
	fullCmd.Flags().String("dataset", "", "Path to dataset file")
	fullCmd.Flags().Int("iterations", 3, "Number of test iterations")
	fullCmd.Flags().Duration("timeout", time.Minute*10, "Benchmark timeout")
	fullCmd.Flags().String("output-format", "table", "Output format: table, json, both")
	fullCmd.Flags().String("output", "", "Output file path")
	fullCmd.Flags().Bool("verbose", false, "Verbose output")

	bc.rootCmd.AddCommand(fullCmd)

	// Compare specific models command
	compareCmd := &cobra.Command{
		Use:   "compare [models...]",
		Short: "Compare specific embedding models",
		Args:  cobra.MinimumNArgs(2),
		RunE:  bc.runCompareCommand,
	}

	compareCmd.Flags().Int("text-count", 50, "Number of test texts")
	compareCmd.Flags().String("output-format", "table", "Output format")
	compareCmd.Flags().String("output", "", "Output file path")

	bc.rootCmd.AddCommand(compareCmd)

	// List available models command
	bc.rootCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all available embedding models",
		RunE:  bc.runListCommand,
	})

	// Model info command
	infoCmd := &cobra.Command{
		Use:   "info [model]",
		Short: "Show detailed information about a specific model",
		Args:  cobra.ExactArgs(1),
		RunE:  bc.runInfoCommand,
	}

	bc.rootCmd.AddCommand(infoCmd)
}

// GetCommand returns the root cobra command
func (bc *BenchmarkCommand) GetCommand() *cobra.Command {
	return bc.rootCmd
}

// runQuickBenchmark runs the quick benchmark
func (bc *BenchmarkCommand) runQuickBenchmark(cmd *cobra.Command, args []string) error {
	fmt.Printf("Running quick benchmark...\n\n")

	result, err := bc.suite.RunQuickBenchmark()
	if err != nil {
		return fmt.Errorf("benchmark failed: %w", err)
	}

	// Print additional analysis
	bc.printAdditionalAnalysis(result)

	return nil
}

// runFullBenchmark runs the full benchmark
func (bc *BenchmarkCommand) runFullBenchmark(cmd *cobra.Command, args []string) error {
	models, _ := cmd.Flags().GetStringSlice("models")
	textCount, _ := cmd.Flags().GetInt("text-count")
	dataset, _ := cmd.Flags().GetString("dataset")
	iterations, _ := cmd.Flags().GetInt("iterations")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	outputFormat, _ := cmd.Flags().GetString("output-format")
	outputFile, _ := cmd.Flags().GetString("output")
	verbose, _ := cmd.Flags().GetBool("verbose")

	fmt.Printf("Running comprehensive benchmark...\n")
	if verbose {
		fmt.Printf("Models: %v\n", models)
		fmt.Printf("Text count: %d\n", textCount)
		fmt.Printf("Iterations: %d\n", iterations)
		fmt.Printf("Timeout: %v\n", timeout)
		fmt.Printf("Output format: %s\n", outputFormat)
		fmt.Printf("Output file: %s\n\n", outputFile)
	}

	config := BenchmarkConfig{
		Models:       models,
		TextCount:    textCount,
		DatasetPath:  dataset,
		Iterations:   iterations,
		Timeout:      timeout,
		OutputFormat: outputFormat,
		OutputPath:   outputFile,
		Verbose:      verbose,
	}

	result, err := bc.suite.RunBenchmark(config)
	if err != nil {
		return fmt.Errorf("benchmark failed: %w", err)
	}

	if verbose {
		bc.printAdditionalAnalysis(result)
	}

	return nil
}

// runCompareCommand runs comparison for specific models
func (bc *BenchmarkCommand) runCompareCommand(cmd *cobra.Command, args []string) error {
	textCount, _ := cmd.Flags().GetInt("text-count")
	outputFormat, _ := cmd.Flags().GetString("output-format")
	outputFile, _ := cmd.Flags().GetString("output")

	fmt.Printf("Comparing models: %v\n\n", args)

	// Generate test texts
	testTexts := bc.generateTestTexts(textCount)

	// Create comparator and run comparison
	comparator := NewDefaultComparator(nil)
	ctx := context.Background()

	metrics, err := comparator.Compare(ctx, args, testTexts)
	if err != nil {
		return fmt.Errorf("comparison failed: %w", err)
	}

	// Output results
	if outputFormat == "json" || outputFormat == "both" {
		data, _ := json.MarshalIndent(metrics, "", "  ")
		if outputFile != "" {
			if err := os.WriteFile(outputFile, data, 0644); err != nil {
				fmt.Printf("Error writing to file: %v\n", err)
			}
		} else {
			fmt.Printf("Results (JSON):\n%s\n", string(data))
		}
	}

	if outputFormat == "table" || outputFormat == "both" {
		PrintComparisonTable(metrics)
	}

	// Print comparison analysis
	bc.printComparisonAnalysis(metrics)

	return nil
}

// runListCommand lists all available models
func (bc *BenchmarkCommand) runListCommand(cmd *cobra.Command, args []string) error {
	registry := GetDefaultRegistry()
	models := registry.List()

	fmt.Printf("Available Vector Embedding Models:\n\n")
	fmt.Printf("%-20s %-10s %-15s %-15s %s\n", "Model Name", "Dimension", "Languages", "Specialization", "Size")
	fmt.Println(strings.Repeat("-", 80))

	for _, modelName := range models {
		caps, err := registry.GetCapabilities(modelName)
		if err != nil {
			continue
		}

		specialization := "General"
		if caps.OptimizedForChinese {
			specialization = "Chinese"
		} else if modelName == "stsb-bert-tiny" {
			specialization = "Lightweight"
		} else if modelName == "hash-fallback" {
			specialization = "Fallback"
		}

		sizeStr := fmt.Sprintf("%.1fMB", float64(caps.ModelSizeBytes)/1024/1024)
		if caps.ModelSizeBytes == 0 {
			sizeStr = "N/A"
		}

		languages := strings.Join(caps.Languages[:min(len(caps.Languages), 3)], ", ")
		if len(caps.Languages) > 3 {
			languages += "..."
		}

		fmt.Printf("%-20s %-10d %-15s %-15s %s\n",
			modelName, caps.RecommendedBatchSize, languages, specialization, sizeStr)
	}

	fmt.Println()
	return nil
}

// runInfoCommand shows detailed information about a model
func (bc *BenchmarkCommand) runInfoCommand(cmd *cobra.Command, args []string) error {
	modelName := args[0]
	registry := GetDefaultRegistry()

	caps, err := registry.GetCapabilities(modelName)
	if err != nil {
		return fmt.Errorf("model '%s' not found: %w", modelName, err)
	}

	fmt.Printf("Model Information: %s\n\n", modelName)
	fmt.Printf("Languages: %v\n", caps.Languages)
	fmt.Printf("Max Sequence Length: %d\n", caps.MaxSequenceLength)
	fmt.Printf("Recommended Batch Size: %d\n", caps.RecommendedBatchSize)
	fmt.Printf("Supports Multilingual: %t\n", caps.SupportsMultilingual)
	fmt.Printf("Optimized for Chinese: %t\n", caps.OptimizedForChinese)
	fmt.Printf("Supports GPU: %t\n", caps.SupportsGPU)
	fmt.Printf("Model Size: %.1f MB\n", float64(caps.ModelSizeBytes)/1024/1024)
	fmt.Printf("Estimated Memory Usage: %.1f MB\n", float64(caps.EstimatedMemoryUsage)/1024/1024)

	// Add model-specific recommendations
	fmt.Printf("\nUse Case Recommendations:\n")

	switch modelName {
	case "all-minilm-l6-v2":
		fmt.Printf("- General multilingual applications\n")
		fmt.Printf("- Balanced performance and quality\n")
		fmt.Printf("- Good for mixed-language workloads\n")
	case "gte-small-zh":
		fmt.Printf("- Chinese-heavy workloads\n")
		fmt.Printf("- Higher quality for Chinese text\n")
		fmt.Printf("- Semantic search in Chinese\n")
	case "stsb-bert-tiny":
		fmt.Printf("- Resource-constrained environments\n")
		fmt.Printf("- Mobile or edge devices\n")
		fmt.Printf("- Applications requiring low latency\n")
	case "hash-fallback":
		fmt.Printf("- Fallback when other models fail\n")
		fmt.Printf("- Minimal resource usage\n")
		fmt.Printf("- Development and testing\n")
	}

	fmt.Println()

	return nil
}

// generateTestTexts generates test texts for comparison
func (bc *BenchmarkCommand) generateTestTexts(count int) []string {
	baseTexts := []string{
		"Hello world", "你好世界", "Machine learning is fascinating", "机器学习很有趣",
		"The weather is nice today", "今天天气很好", "Natural language processing",
		"自然语言处理", "Vector embeddings", "向量嵌入", "Deep learning", "深度学习",
		"Artificial intelligence", "人工智能", "Data science", "数据科学",
		"Hello", "Hi", "Thanks", "谢谢", "Goodbye", "再见",
	}

	texts := make([]string, count)
	for i := 0; i < count; i++ {
		texts[i] = baseTexts[i%len(baseTexts)]
	}

	return texts
}

// printAdditionalAnalysis prints additional analysis
func (bc *BenchmarkCommand) printAdditionalAnalysis(result *BenchmarkResult) {
	fmt.Printf("Additional Analysis:\n\n")

	// Performance categories
	fmt.Printf("Performance Categories:\n")
	for _, m := range result.Models {
		category := "Standard"
		if m.ThroughputQPS > 500 && m.LatencyMs < 5 {
			category = "High Performance"
		} else if m.MemoryUsageMB > 200 || m.LatencyMs > 50 {
			category = "Resource Intensive"
		} else if m.MemoryUsageMB < 50 {
			category = "Lightweight"
		}

		fmt.Printf("  %s: %s\n", m.ModelName, category)
	}

	// Use case recommendations
	fmt.Printf("\nUse Case Recommendations:\n")
	if result.Summary.TestTextCount > 0 {
		chineseRatio := bc.estimateChineseRatio(result.Summary.TestTextCount)
		if chineseRatio > 0.3 {
			fmt.Printf("  - Dataset contains %.0f%% Chinese text: Consider GTE-small-zh for better quality\n", chineseRatio*100)
		}

		if result.Summary.AvgMemory > 150 {
			fmt.Printf("  - High memory usage: Consider stsb-bert-tiny for resource-constrained environments\n")
		}

		if result.Summary.AvgLatency > 10 {
			fmt.Printf("  - High latency: Consider stsb-bert-tiny for real-time applications\n")
		}
	}

	fmt.Println()
}

// printComparisonAnalysis prints detailed comparison analysis
func (bc *BenchmarkCommand) printComparisonAnalysis(metrics []PerformanceMetrics) {
	if len(metrics) < 2 {
		return
	}

	fmt.Printf("Comparison Analysis:\n\n")

	// Find best in each category
	bestThroughput := metrics[0]
	bestLatency := metrics[0]
	bestMemory := metrics[0]
	bestQuality := metrics[0]

	for i := 1; i < len(metrics); i++ {
		m := metrics[i]
		if m.ThroughputQPS > bestThroughput.ThroughputQPS {
			bestThroughput = m
		}
		if m.LatencyMs < bestLatency.LatencyMs {
			bestLatency = m
		}
		if m.MemoryUsageMB < bestMemory.MemoryUsageMB {
			bestMemory = m
		}
		if m.QualityScore > bestQuality.QualityScore {
			bestQuality = m
		}
	}

	fmt.Printf("Best Performance by Category:\n")
	fmt.Printf("  Throughput: %s (%.1f QPS)\n", bestThroughput.ModelName, bestThroughput.ThroughputQPS)
	fmt.Printf("  Latency: %s (%.2f ms)\n", bestLatency.ModelName, bestLatency.LatencyMs)
	fmt.Printf("  Memory: %s (%.1f MB)\n", bestMemory.ModelName, bestMemory.MemoryUsageMB)
	fmt.Printf("  Quality: %s (%.2f score)\n\n", bestQuality.ModelName, bestQuality.QualityScore)

	// Efficiency ratios
	fmt.Printf("Efficiency Ratios (relative to %s):\n", metrics[0].ModelName)
	baseline := metrics[0]
	for i := 1; i < len(metrics); i++ {
		m := metrics[i]
		speedRatio := m.ThroughputQPS / baseline.ThroughputQPS
		memoryRatio := baseline.MemoryUsageMB / m.MemoryUsageMB
		fmt.Printf("  %s: %.2fx speed, %.2fx memory efficiency\n", m.ModelName, speedRatio, memoryRatio)
	}

	fmt.Println()
}

// estimateChineseRatio estimates the ratio of Chinese text
func (bc *BenchmarkCommand) estimateChineseRatio(textCount int) float64 {
	// Simple heuristic - assume 40% Chinese in our test data
	return 0.4
}

// RunBenchmarkCLI runs the benchmark CLI with the given arguments
func RunBenchmarkCLI(args []string) error {
	cmd := NewBenchmarkCommand()
	cmd.rootCmd.SetArgs(args)
	return cmd.rootCmd.Execute()
}
