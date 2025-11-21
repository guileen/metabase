package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/guileen/metabase/pkg/biz/rag/embedding"
	"github.com/guileen/metabase/pkg/biz/rag/vocab"
	"github.com/spf13/cobra"
)

var vocabCmd = &cobra.Command{
	Use:   "vocab",
	Short: "词表管理和构建工具",
	Long: `词表管理和构建工具，支持增量更新、查询扩展和统计信息。

该工具会自动在用户主目录创建隐藏文件夹 ~/.metabase/vocab/ 来存储词表数据。

示例:
  # 从当前目录构建词表
  metabase vocab build .

  # 增量更新词表
  metabase vocab update .

  # 显示词表统计信息
  metabase vocab stats

  # 扩展查询
  metabase vocab expand "database connection" --limit 10

  # 导出词表
  metabase vocab export vocabulary.txt --format txt`,
}

var buildCmd = &cobra.Command{
	Use:   "build [path]",
	Short: "构建词表索引",
	Long: `从指定路径构建词表索引。支持递归扫描目录。

示例:
  metabase vocab build .                    # 从当前目录构建
  metabase vocab build ./src --recursive   # 递归构建 src 目录
  metabase vocab build file1.go file2.rs   # 从指定文件构建`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config := createVocabConfig(cmd)
		builder := vocab.NewVocabularyBuilder(config)

		recursive, _ := cmd.Flags().GetBool("recursive")

		// 判断是目录还是文件
		if len(args) == 1 && isDirectory(args[0]) {
			// 目录构建
			result, err := builder.BuildFromDirectory(args[0], recursive)
			if err != nil {
				cmd.PrintErrln("构建词表失败:", err)
				return
			}
			printUpdateResult(result)
			if err := builder.CacheTermEmbeddingsDefault(10000); err != nil {
				cmd.PrintErrln("词向量缓存失败:", err)
			}
		} else {
			// 文件构建
			result, err := builder.BuildFromFiles(args)
			if err != nil {
				cmd.PrintErrln("构建词表失败:", err)
				return
			}
			printUpdateResult(result)
			if err := builder.CacheTermEmbeddingsDefault(10000); err != nil {
				cmd.PrintErrln("词向量缓存失败:", err)
			}
		}

		// 显示统计信息
		builder.PrintStats()
	},
}

var updateCmd = &cobra.Command{
	Use:   "update [path]",
	Short: "增量更新词表索引",
	Long: `增量更新词表索引，只处理变更的文件。

示例:
  metabase vocab update .                    # 更新当前目录
  metabase vocab update ./src --recursive   # 递归更新 src 目录`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config := createVocabConfig(cmd)
		builder, err := vocab.LoadVocabularyBuilder(config)
		if err != nil {
			cmd.PrintErrln("加载词表失败，将创建新的词表:", err)
			builder = vocab.NewVocabularyBuilder(config)
		}

		recursive, _ := cmd.Flags().GetBool("recursive")

		var result *vocab.UpdateResult
		if len(args) == 1 && isDirectory(args[0]) {
			result, err = builder.UpdateFromDirectory(args[0], recursive)
		} else {
			result, err = builder.BuildFromFiles(args)
		}

		if err != nil {
			cmd.PrintErrln("更新词表失败:", err)
			return
		}

		printUpdateResult(result)
		builder.PrintStats()
		if err := builder.CacheTermEmbeddingsDefault(10000); err != nil {
			cmd.PrintErrln("词向量缓存失败:", err)
		}
	},
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "显示词表统计信息",
	Long: `显示词表的详细统计信息，包括词汇数量、分类分布、语言分布等。

示例:
  metabase vocab stats              # 显示基本信息
  metabase vocab stats --detail     # 显示详细信息`,
	Run: func(cmd *cobra.Command, args []string) {
		config := createVocabConfig(cmd)
		builder, err := vocab.LoadVocabularyBuilder(config)
		if err != nil {
			cmd.PrintErrln("加载词表失败:", err)
			return
		}

		detail, _ := cmd.Flags().GetBool("detail")
		if detail {
			// 显示详细信息
			vocabStats := builder.GetVocabularyStats()
			printDetailedStats(vocabStats)
		} else {
			// 显示基本统计信息
			builder.PrintStats()
		}
	},
}

var expandCmd = &cobra.Command{
	Use:   "expand \"query\"",
	Short: "扩展查询词汇",
	Long: `基于词表扩展查询词汇，提供相似词和分类词汇建议。

示例:
  metabase vocab expand "database" --limit 10
  metabase vocab expand "user authentication" --category keyword
  metabase vocab expand "api" --format json`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config := createVocabConfig(cmd)
		builder, err := vocab.LoadVocabularyBuilder(config)
		if err != nil {
			cmd.PrintErrln("加载词表失败:", err)
			return
		}

		query := args[0]
		limit, _ := cmd.Flags().GetInt("limit")
		format, _ := cmd.Flags().GetString("format")
		category, _ := cmd.Flags().GetString("category")
		useEmbedding, _ := cmd.Flags().GetBool("use-embedding")
		embType, _ := cmd.Flags().GetString("embedding-type")

		if useEmbedding {
			// Use local embedding for expansion
			embCfg := &embedding.Config{LocalModelType: embType, BatchSize: 64, MaxConcurrency: 4, EnableFallback: false}
			emb, e := embedding.NewLocalEmbedder(embCfg)
			if e != nil {
				cmd.PrintErrln("初始化本地嵌入失败:", e)
				return
			}
			res, e := builder.ExpandQueryWithEmbedding(query, limit, emb)
			if e != nil {
				cmd.PrintErrln("嵌入扩展失败:", e)
				return
			}
			if format == "json" {
				printExpansionResultAsJSON(res)
			} else {
				printExpansionResult(res)
			}
			return
		}

		if format == "json" {
			// 搜索指定分类的词汇
			if category != "" {
				terms := builder.GetIndex().GetTopTerms(limit, category)
				printTermsAsJSON(terms, query, category)
			} else {
				result := builder.GetIndex().ExpandQuery(query, limit)
				printExpansionResultAsJSON(result)
			}
		} else {
			// 文本格式输出
			if category != "" {
				terms := builder.GetIndex().GetTopTerms(limit, category)
				printTermsByCategory(terms, category)
			} else {
				result := builder.GetIndex().ExpandQuery(query, limit)
				printExpansionResult(result)
			}
		}
	},
}

var vocabSearchCmd = &cobra.Command{
	Use:   "search \"term\"",
	Short: "搜索词表中的词汇",
	Long: `在词表中搜索匹配的词汇。

示例:
  metabase vocab search "database"
  metabase vocab search "user" --limit 20
  metabase vocab search "auth" --format json`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config := createVocabConfig(cmd)
		builder, err := vocab.LoadVocabularyBuilder(config)
		if err != nil {
			cmd.PrintErrln("加载词表失败:", err)
			return
		}

		term := args[0]
		limit, _ := cmd.Flags().GetInt("limit")
		format, _ := cmd.Flags().GetString("format")

		terms := builder.SearchTerms(term, limit)

		if format == "json" {
			printTermsAsJSON(terms, term, "search")
		} else {
			printVocabSearchResults(terms, term)
		}
	},
}

var exportCmd = &cobra.Command{
	Use:   "export [filename]",
	Short: "导出词表",
	Long: `导出词表为指定格式文件。

示例:
  metabase vocab export vocab.txt          # 导出为文本格式
  metabase vocab export vocab.csv --format csv    # 导出为 CSV 格式
  metabase vocab export vocab.json --format json  # 导出为 JSON 格式`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config := createVocabConfig(cmd)
		builder, err := vocab.LoadVocabularyBuilder(config)
		if err != nil {
			cmd.PrintErrln("加载词表失败:", err)
			return
		}

		filename := args[0]
		format, _ := cmd.Flags().GetString("format")
		limit, _ := cmd.Flags().GetInt("limit")

		if err := builder.ExportVocabularyWithLimit(filename, format, limit); err != nil {
			cmd.PrintErrln("导出词表失败:", err)
			return
		}
	},
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "清理旧词汇",
	Long: `清理长时间未使用的词汇，优化词表大小。

示例:
  metabase vocab cleanup                 # 清理 30 天未使用的词汇
  metabase vocab cleanup --days 7       # 清理 7 天未使用的词汇
  metabase vocab cleanup --optimize     # 同时优化索引`,
	Run: func(cmd *cobra.Command, args []string) {
		config := createVocabConfig(cmd)
		builder, err := vocab.LoadVocabularyBuilder(config)
		if err != nil {
			cmd.PrintErrln("加载词表失败:", err)
			return
		}

		days, _ := cmd.Flags().GetInt("days")
		optimize, _ := cmd.Flags().GetBool("optimize")

		maxAge := time.Duration(days) * 24 * time.Hour
		removed := builder.GetIndex().CleanupOldTerms(maxAge)

		fmt.Printf("清理完成，删除了 %d 个旧词汇\n", removed)

		if optimize {
			fmt.Println("正在优化索引...")
			if err := builder.GetIndex().OptimizeIndex(); err != nil {
				cmd.PrintErrln("优化索引失败:", err)
			} else {
				fmt.Println("索引优化完成")
			}
		}
	},
}

// 辅助函数
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func createVocabConfig(cmd *cobra.Command) *vocab.Config {
	config := vocab.CreateDefaultConfig()

	// 从命令行参数更新配置
	if includePatterns, _ := cmd.Flags().GetStringSlice("include"); len(includePatterns) > 0 {
		config.IncludePatterns = includePatterns
	}

	if excludePatterns, _ := cmd.Flags().GetStringSlice("exclude"); len(excludePatterns) > 0 {
		config.ExcludePatterns = excludePatterns
	}

	if dataDir, _ := cmd.Flags().GetString("data-dir"); dataDir != "" {
		config.DataDir = dataDir
		config.IndexFile = filepath.Join(dataDir, "vocabulary.idx")
	}

	if autoUpdate, _ := cmd.Flags().GetBool("auto-update"); cmd.Flags().Changed("auto-update") {
		config.AutoUpdate = autoUpdate
	}

	if updateInterval, _ := cmd.Flags().GetInt("update-interval"); cmd.Flags().Changed("update-interval") {
		config.UpdateInterval = updateInterval
	}

	return config
}

func printUpdateResult(result *vocab.UpdateResult) {
	fmt.Printf("\n=== UPDATE RESULT ===\n")
	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Printf("Added files: %d\n", result.AddedFiles)
	fmt.Printf("Updated files: %d\n", result.UpdatedFiles)
	fmt.Printf("Deleted files: %d\n", result.DeletedFiles)
	fmt.Printf("New terms: %d\n", result.NewTerms)
	fmt.Printf("Removed terms: %d\n", result.RemovedTerms)

	if len(result.Errors) > 0 {
		fmt.Printf("Errors: %d\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}
}

func printDetailedStats(stats map[string]interface{}) {
	globalStats := stats["global_stats"].(*vocab.GlobalStats)
	categoryStats := stats["category_stats"].(map[string]int)
	languageStats := stats["language_stats"].(map[string]int)
	config := stats["config"].(*vocab.Config)
	metadata := stats["metadata"].(*vocab.IndexMetadata)

	fmt.Printf("\n=== DETAILED VOCABULARY STATISTICS ===\n")
	fmt.Printf("Index Version: %s\n", metadata.Version)
	fmt.Printf("Created: %s\n", metadata.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Last Update: %s\n", metadata.LastUpdate.Format(time.RFC3339))
	fmt.Printf("Total Updates: %d\n", metadata.TotalUpdates)
	fmt.Printf("Build Duration: %v\n", metadata.BuildDuration)

	fmt.Printf("\n--- Configuration ---\n")
	fmt.Printf("Data Directory: %s\n", config.DataDir)
	fmt.Printf("Index File: %s\n", config.IndexFile)
	fmt.Printf("Auto Update: %t\n", config.AutoUpdate)
	fmt.Printf("Update Interval: %d minutes\n", config.UpdateInterval)
	fmt.Printf("Include Patterns: %v\n", config.IncludePatterns)
	fmt.Printf("Exclude Patterns: %v\n", config.ExcludePatterns)

	fmt.Printf("\n--- Global Statistics ---\n")
	fmt.Printf("Total Documents: %d\n", globalStats.TotalDocuments)
	fmt.Printf("Unique Terms: %d\n", globalStats.UniqueTerms)
	fmt.Printf("Total Terms: %d\n", globalStats.TotalTerms)
	fmt.Printf("Total Tokens: %d\n", globalStats.TotalTokens)
	fmt.Printf("Vocabulary Size: %d\n", globalStats.VocabularySize)
	fmt.Printf("Index Size: %.2f MB\n", float64(globalStats.IndexSize)/1024/1024)
	fmt.Printf("Average Document Length: %.1f bytes\n", globalStats.AvgDocLength)

	fmt.Printf("\n--- Category Distribution ---\n")
	totalTerms := float64(globalStats.UniqueTerms)
	for category, count := range categoryStats {
		percentage := float64(count) / totalTerms * 100
		fmt.Printf("%-12s: %6d (%5.1f%%)\n", category, count, percentage)
	}

	fmt.Printf("\n--- Language Distribution ---\n")
	totalDocs := float64(globalStats.TotalDocuments)
	for language, count := range languageStats {
		percentage := float64(count) / totalDocs * 100
		fmt.Printf("%-12s: %6d (%5.1f%%)\n", language, count, percentage)
	}
}

func printExpansionResult(result *vocab.QueryExpansionResult) {
	fmt.Printf("\n=== QUERY EXPANSION ===\n")
	fmt.Printf("Original terms: %v\n", result.OriginalTerms)
	fmt.Printf("Expanded terms (%d): %v\n", len(result.ExpandedTerms), result.ExpandedTerms)

	fmt.Printf("\n--- Similar Terms ---\n")
	for original, similar := range result.SimilarTerms {
		fmt.Printf("%s: %v\n", original, similar)
	}

	fmt.Printf("\n--- Category Terms ---\n")
	for category, terms := range result.CategoryTerms {
		fmt.Printf("%s: %v\n", category, terms)
	}

	fmt.Printf("\n--- Weighted Terms (Top 10) ---\n")
	count := 0
	for term, weight := range result.WeightedTerms {
		if count >= 10 {
			break
		}
		fmt.Printf("%-20s: %.6f\n", term, weight)
		count++
	}
}

func printExpansionResultAsJSON(result *vocab.QueryExpansionResult) {
	fmt.Printf("{\n")
	fmt.Printf("  \"original_terms\": %v,\n", result.OriginalTerms)
	fmt.Printf("  \"expanded_terms\": %v,\n", result.ExpandedTerms)
	fmt.Printf("  \"similar_terms\": {\n")

	first := true
	for original, similar := range result.SimilarTerms {
		if !first {
			fmt.Printf(",\n")
		}
		fmt.Printf("    \"%s\": %v", original, similar)
		first = false
	}

	fmt.Printf("\n  },\n")
	fmt.Printf("  \"category_terms\": {\n")

	first = true
	for category, terms := range result.CategoryTerms {
		if !first {
			fmt.Printf(",\n")
		}
		fmt.Printf("    \"%s\": %v", category, terms)
		first = false
	}

	fmt.Printf("\n  },\n")
	fmt.Printf("  \"weighted_terms\": {\n")

	first = true
	count := 0
	for term, weight := range result.WeightedTerms {
		if count >= 10 {
			break
		}
		if !first {
			fmt.Printf(",\n")
		}
		fmt.Printf("    \"%s\": %.6f", term, weight)
		first = false
		count++
	}

	fmt.Printf("\n  }\n")
	fmt.Printf("}\n")
}

func printTermsByCategory(terms []*vocab.TermInfo, category string) {
	fmt.Printf("\n=== TOP %d TERMS IN CATEGORY: %s ===\n", len(terms), category)
	fmt.Printf("%-20s %8s %8s %8s %8s %s\n", "Term", "Weight", "Docs", "Freq", "Category", "LastSeen")
	fmt.Printf("%-20s %8s %8s %8s %8s %s\n", strings.Repeat("-", 20), strings.Repeat("-", 8), strings.Repeat("-", 8), strings.Repeat("-", 8), strings.Repeat("-", 8), strings.Repeat("-", 19))

	for _, termInfo := range terms {
		fmt.Printf("%-20s %8.6f %8d %8d %8s %s\n",
			termInfo.Term,
			termInfo.Weight,
			termInfo.DocumentFreq,
			termInfo.TotalFreq,
			termInfo.Category,
			termInfo.LastSeen.Format("2006-01-02"))
	}
}

func printTermsAsJSON(terms []*vocab.TermInfo, query, context string) {
	fmt.Printf("[\n")
	for i, termInfo := range terms {
		comma := ","
		if i == len(terms)-1 {
			comma = ""
		}

		fmt.Printf("  {\n")
		fmt.Printf("    \"term\": \"%s\",\n", termInfo.Term)
		fmt.Printf("    \"weight\": %.6f,\n", termInfo.Weight)
		fmt.Printf("    \"document_freq\": %d,\n", termInfo.DocumentFreq)
		fmt.Printf("    \"total_freq\": %d,\n", termInfo.TotalFreq)
		fmt.Printf("    \"category\": \"%s\",\n", termInfo.Category)
		fmt.Printf("    \"last_seen\": \"%s\",\n", termInfo.LastSeen.Format(time.RFC3339))
		fmt.Printf("    \"context\": \"%s\",\n", context)
		fmt.Printf("    \"query\": \"%s\"\n", query)
		fmt.Printf("  }%s\n", comma)
	}
	fmt.Printf("]\n")
}

func printVocabSearchResults(terms []*vocab.TermInfo, query string) {
	fmt.Printf("\n=== SEARCH RESULTS FOR: %s ===\n", query)
	fmt.Printf("%-20s %8s %8s %8s %8s %s\n", "Term", "Weight", "Docs", "Freq", "Category", "LastSeen")
	fmt.Printf("%-20s %8s %8s %8s %8s %s\n", strings.Repeat("-", 20), strings.Repeat("-", 8), strings.Repeat("-", 8), strings.Repeat("-", 8), strings.Repeat("-", 8), strings.Repeat("-", 19))

	for _, termInfo := range terms {
		fmt.Printf("%-20s %8.6f %8d %8d %8s %s\n",
			termInfo.Term,
			termInfo.Weight,
			termInfo.DocumentFreq,
			termInfo.TotalFreq,
			termInfo.Category,
			termInfo.LastSeen.Format("2006-01-02"))
	}
}

func init() {
	vocabCmd.AddCommand(buildCmd)
	vocabCmd.AddCommand(updateCmd)
	vocabCmd.AddCommand(statsCmd)
	vocabCmd.AddCommand(expandCmd)
	vocabCmd.AddCommand(vocabSearchCmd)
	vocabCmd.AddCommand(exportCmd)
	vocabCmd.AddCommand(cleanupCmd)

	// 通用标志
	vocabCmd.PersistentFlags().StringSlice("include", []string{}, "包含文件模式")
	vocabCmd.PersistentFlags().StringSlice("exclude", []string{}, "排除文件模式")
	vocabCmd.PersistentFlags().String("data-dir", "", "数据目录路径")
	vocabCmd.PersistentFlags().Bool("auto-update", true, "启用自动更新")
	vocabCmd.PersistentFlags().Int("update-interval", 60, "更新间隔（分钟）")

	// 构建命令标志
	buildCmd.Flags().Bool("recursive", true, "递归扫描子目录")

	// 更新命令标志
	updateCmd.Flags().Bool("recursive", true, "递归扫描子目录")

	// 统计命令标志
	statsCmd.Flags().Bool("detail", false, "显示详细信息")

	// 扩展命令标志
	expandCmd.Flags().Int("limit", 20, "扩展词汇数量限制")
	expandCmd.Flags().String("format", "text", "输出格式 (text, json)")
	expandCmd.Flags().String("category", "", "指定词汇分类")
	expandCmd.Flags().Bool("use-embedding", true, "使用本地嵌入进行扩展")
	expandCmd.Flags().String("embedding-type", "cybertron", "本地嵌入类型 (fast, python, onnx, cybertron)")

	// 搜索命令标志
	searchCmd.Flags().Int("limit", 20, "搜索结果数量限制")
	searchCmd.Flags().String("format", "text", "输出格式 (text, json)")

	// 导出命令标志
	exportCmd.Flags().String("format", "txt", "导出格式 (txt, csv, json)")
	exportCmd.Flags().Int("limit", 1000, "导出词条数量 (例如 1000 或 0 表示全部)")

	// 清理命令标志
	cleanupCmd.Flags().Int("days", 30, "清理多少天前的词汇")
	cleanupCmd.Flags().Bool("optimize", false, "同时优化索引")

	// AddCommand(vocabCmd)  // 已集成到 rag vocab 中
}
