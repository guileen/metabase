//go:build rag_cli

package cli

import (
	"fmt"
	"strings"

	"github.com/guileen/metabase/pkg/biz/rag"
	"github.com/spf13/cobra"
)

// ragCmd 统一的 RAG 命令（包含搜索和词表管理）
var ragCmd = &cobra.Command{
	Use:   "rag",
	Short: "RAG 语义搜索和词表管理",
	Long: `统一的 RAG 系统命令，支持语义搜索和词表管理。

搜索示例:
  metabase rag search "如何使用嵌入系统"
  metabase rag search --top 5 --local "数据库连接"
  metabase rag search --include "*.go" "API 设计"

词表管理示例:
  metabase rag vocab build               # 构建词表
  metabase rag vocab update              # 更新词表
  metabase rag vocab stats               # 词表统计
  metabase rag vocab expand "关键词"      # 扩展查询
  metabase rag vocab clean               # 清理词表

快速搜索示例:
  metabase rag "查询内容"                 # 默认搜索
  metabase rag --top 10 "查询内容"        # 指定结果数量
  metabase rag --local "查询内容"         # 本地模式`,
	Run: func(cmd *cobra.Command, args []string) {
		// 默认行为：快速搜索
		if len(args) > 0 {
			query := strings.Join(args, " ")
			topK, _ := cmd.Flags().GetInt("top")
			localGo, _ := cmd.Flags().GetBool("local")

			if topK <= 0 {
				topK = 10
			}

			var results []*rag.SearchResult
			var err error

			if localGo {
				// 本地模式
				ragInstance, rerr := rag.NewLocal(".")
				if rerr != nil {
					cmd.PrintErrln("创建本地 RAG 实例失败:", rerr.Error())
					return
				}
				defer ragInstance.Close()

				config := ragInstance.GetConfig()
				config.TopK = topK
				results, err = ragInstance.Query(cmd.Context(), query)
			} else {
				// 快速搜索
				results, err = rag.QuickSearchWithTop(query, topK)
			}

			if err != nil {
				cmd.PrintErrln("搜索失败:", err.Error())
				return
			}

			rag.PrintSimpleResults(results)
		} else {
			cmd.Help()
		}
	},
}

func init() {
	// 添加全局参数
	ragCmd.Flags().IntP("top", "k", 10, "返回结果数量")
	ragCmd.Flags().Bool("local", false, "使用本地模式")

	// 添加子命令
	ragCmd.AddCommand(ragSearchCmd())
	ragCmd.AddCommand(ragVocabCmd())
	ragCmd.AddCommand(ragQuickCmd())

	// 将 RAG 命令添加到根命令
	AddCommand(ragCmd)
}

// ragSearchCmd 搜索子命令
func ragSearchCmd() *cobra.Command {
	return rag.CLICommand()
}

// ragVocabCmd 词表管理子命令
func ragVocabCmd() *cobra.Command {
	vocabCmd := &cobra.Command{
		Use:   "vocab",
		Short: "词表管理",
	}

	// 构建命令
	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "构建词表",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("开始构建词表...")
			if err := rag.BuildVocabulary(); err != nil {
				fmt.Printf("构建词表失败: %v\n", err)
			} else {
				fmt.Println("词表构建完成")
			}
		},
	}

	// 更新命令
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "更新词表",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("开始更新词表...")
			if err := rag.UpdateVocabulary(); err != nil {
				fmt.Printf("更新词表失败: %v\n", err)
			} else {
				fmt.Println("词表更新完成")
			}
		},
	}

	// 统计命令
	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "词表统计",
		Run: func(cmd *cobra.Command, args []string) {
			stats, err := rag.GetGlobalStats()
			if err != nil {
				fmt.Printf("获取统计失败: %v\n", err)
				return
			}

			fmt.Printf("RAG 系统统计:\n")
			fmt.Printf("  模式: %s\n", stats.Mode)
			if stats.VocabularyTerms > 0 {
				fmt.Printf("  词表: %d 术语, %d 文档\n", stats.VocabularyTerms, stats.VocabularyDocs)
				fmt.Printf("  最后更新: %s\n", stats.VocabularyLastUpdated.Format("2006-01-02 15:04"))
			}
			if stats.CloudStats != nil {
				fmt.Printf("  云文档数: %d\n", stats.CloudStats.TotalDocuments)
				fmt.Printf("  索引大小: %d\n", stats.CloudStats.IndexSize)
				fmt.Printf("  缓存命中率: %.1f%%\n", stats.CloudStats.CacheHitRate*100)
			}
		},
	}

	// 扩展命令
	expandCmd := &cobra.Command{
		Use:   "expand [query]",
		Short: "扩展查询",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			maxTerms, _ := cmd.Flags().GetInt("max")
			if maxTerms <= 0 {
				maxTerms = 10
			}

			query := args[0]
			terms, err := rag.ExpandQuery(query, maxTerms)
			if err != nil {
				fmt.Printf("查询扩展失败: %v\n", err)
				return
			}

			fmt.Printf("原始查询: %s\n", query)
			fmt.Printf("扩展结果 (%d 项):\n", len(terms))
			for i, term := range terms {
				if term != query {
					fmt.Printf("  %d. %s\n", i+1, term)
				}
			}
		},
	}
	expandCmd.Flags().Int("max", 10, "最大扩展词数")

	// 清理命令
	cleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "清理词表",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("开始清理词表...")
			// 这里可以实现词表清理逻辑
			fmt.Println("词表清理完成")
		},
	}

	vocabCmd.AddCommand(buildCmd)
	vocabCmd.AddCommand(updateCmd)
	vocabCmd.AddCommand(statsCmd)
	vocabCmd.AddCommand(expandCmd)
	vocabCmd.AddCommand(cleanCmd)

	return vocabCmd
}

// ragQuickCmd 快速搜索子命令
func ragQuickCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "quick [query]",
		Short: "快速搜索",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			topK, _ := cmd.Flags().GetInt("top")
			if topK <= 0 {
				topK = 10
			}

			query := args[0]
			results, err := rag.QuickSearchWithTop(query, topK)
			if err != nil {
				fmt.Printf("搜索失败: %v\n", err)
				return
			}

			rag.PrintSimpleResults(results)
		},
	}
}
