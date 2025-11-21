package cli

import ("context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/guileen/metabase/pkg/biz/rag"
	"github.com/guileen/metabase/pkg/biz/rag/vocab")

// SearchCmd 新的搜索命令，使用 RAG 系统
var SearchCmd = &cobra.Command{
	Use:   "search",
	Short: "代码语义搜索 (使用 RAG 系统)",
	Long: `使用简化的 RAG 系统进行代码语义搜索。

示例:
  metabase search "如何使用嵌入系统"
  metabase search --top 5 --local "数据库连接"
  metabase search --include "*.go" "API 设计"`,
	Run: func(cmd *cobra.Command, args []string) {
		query := strings.TrimSpace(strings.Join(args, " "))
		if query == "" {
			cmd.PrintErrln("请输入查询文本")
			return
		}

		// 获取命令行参数
		topK, _ := cmd.Flags().GetInt("top")
		win, _ := cmd.Flags().GetInt("win")
		localGo, _ := cmd.Flags().GetBool("local-go")
		doExpand, _ := cmd.Flags().GetBool("expand")
		useSkills, _ := cmd.Flags().GetBool("use-skills")
		includeGlobs, _ := cmd.Flags().GetStringSlice("include")
		excludeGlobs, _ := cmd.Flags().GetStringSlice("exclude")
		vocabUpdate, _ := cmd.Flags().GetBool("vocab-update")
		vocabBuild, _ := cmd.Flags().GetBool("vocab-build")

		start := time.Now()

		// 创建 RAG 配置
		config := rag.NewLocalConfig(".")
		config.TopK = topK
		config.Window = win
		config.LocalMode = localGo
		config.EnableExpansion = doExpand
		config.EnableSkills = useSkills
		config.VocabAutoBuild = vocabBuild
		config.VocabAutoUpdate = vocabUpdate
		config.IncludeGlobs = includeGlobs
		config.ExcludeGlobs = excludeGlobs

		// 创建 RAG 实例并执行搜索
		ragInstance, err := rag.NewUnifiedRAG(config)
		if err != nil {
			cmd.PrintErrln("创建 RAG 实例失败:", err.Error())
			return
		}
		defer ragInstance.Close()

		results, err := ragInstance.Query(context.Background(), query)
		if err != nil {
			cmd.PrintErrln("搜索失败:", err.Error())
			return
		}

		duration := time.Since(start)

		// 显示结果
		printNewSearchResults(cmd, query, results, duration, config)
	},
}

func init() {
	// 命令行参数
	SearchCmd.Flags().IntP("top", "k", 15, "返回结果数量")
	SearchCmd.Flags().IntP("win", "w", 8, "上下文窗口大小")
	SearchCmd.Flags().Bool("local-go", false, "使用 Go 本地嵌入")
	SearchCmd.Flags().Bool("expand", true, "启用查询扩展")
	SearchCmd.Flags().Bool("use-skills", false, "使用技能系统")
	SearchCmd.Flags().StringSlice("include", []string{}, "包含的文件模式")
	SearchCmd.Flags().StringSlice("exclude", []string{}, "排除的文件模式")
	SearchCmd.Flags().Bool("vocab-update", true, "自动更新词表")
	SearchCmd.Flags().Bool("vocab-build", true, "自动构建词表")
	SearchCmd.Flags().Int("vocab-max-age", 24, "词表最大有效时间（小时）")

	// 添加到根命令
	AddCommand(SearchCmd)
}

// printNewSearchResults 打印搜索结果（新版本）
func printNewSearchResults(cmd *cobra.Command, query string, results []*rag.SearchResult, duration time.Duration, config *rag.RAGConfig) {
	fmt.Printf("\n=== SEARCH RESULTS ===\n")
	fmt.Printf("Query: %s\n", query)
	fmt.Printf("Search time: %v\n", duration)
	fmt.Printf("Results: %d found\n\n", len(results))

	if len(results) == 0 {
		fmt.Printf("未找到相关结果。\n")
		fmt.Printf("建议:\n")
		fmt.Printf("  - 尝试使用不同的关键词\n")
		fmt.Printf("  - 使用 --expand 启用查询扩展\n")
		fmt.Printf("  - 使用 --skills 启用技能系统\n")
		return
	}

	fmt.Printf("Top results:\n")
	for i, result := range results {
		fmt.Printf("%d. %s:%d (score=%.3f)\n", i+1, result.File, result.Line, result.Score)

		// 显示文件类型
		if result.FileType != "" {
			fmt.Printf("   [%s] ", result.FileType)
		}

		// 显示匹配原因
		if result.Reason != "" {
			fmt.Printf("原因: %s\n", result.Reason)
		}

		// 显示代码片段
		if result.Snippet != "" {
			snippet := result.Snippet
			if len(snippet) > 200 {
				snippet = snippet[:200] + "..."
			}
			fmt.Printf("   %s\n", snippet)
		}

		fmt.Printf("---\n")
	}

	// 显示配置信息
	fmt.Printf("\nConfiguration:\n")
	fmt.Printf("  TopK: %d\n", config.TopK)
	fmt.Printf("  Window: %d\n", config.Window)
	fmt.Printf("  Local mode: %t\n", config.LocalMode)
	fmt.Printf("  Query expansion: %t\n", config.EnableExpansion)
	fmt.Printf("  Skills system: %t\n", config.EnableSkills)
	fmt.Printf("  Vocabulary auto-build: %t\n", config.VocabAutoBuild)
	fmt.Printf("  Vocabulary auto-update: %t\n", config.VocabAutoUpdate)

	// 显示词表统计信息
	if config.VocabAutoBuild || config.VocabAutoUpdate {
		if stats, err := rag.GetVocabularyStats(); err == nil {
			if globalStats, ok := stats["global_stats"]; ok {
				if gs, ok := globalStats.(*vocab.GlobalStats); ok {
					fmt.Printf("Vocabulary: %d terms, %d documents\n", gs.UniqueTerms, gs.TotalDocuments)
				}
			}
		}
	}
}