package rag

import ("context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra")

// CLICommand 创建 RAG CLI 命令
func CLICommand() *cobra.Command {
	var (
		topK            int
		window          int
		localMode       bool
		enableExpansion bool
		enableSkills    bool
		includeGlobs    []string
		excludeGlobs    []string
		forceReindex    bool
	)

	cmd := &cobra.Command{
		Use:   "rag",
		Short: "简化的 RAG 语义搜索",
		Long: `RAG 提供简单易用的语义搜索功能。

示例:
  metabase rag "如何使用嵌入系统"
  metabase rag --top 5 --local "数据库连接"
  metabase rag --include "*.go" --include "*.md" "API 设计"
  metabase rag --exclude "*_test.go" "核心业务逻辑"
  metabase rag --skills "设计模式的使用"`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.PrintErrln("请输入查询内容")
				return
			}

			query := args[0]

			// 创建搜索选项
			opts := &SearchOptions{
				TopK:            topK,
				Window:          window,
				LocalMode:       localMode,
				EnableExpansion: enableExpansion,
				EnableSkills:    enableSkills,
				ForceReindex:    forceReindex,
				IncludeGlobs:    includeGlobs,
				ExcludeGlobs:    excludeGlobs,
			}

			start := time.Now()
			// 创建 RAG 实例并执行查询
			rag := NewWithOptions(opts)
			results, err := rag.Query(context.Background(), query)
			if err != nil {
				cmd.PrintErrln("搜索失败:", err.Error())
				return
			}

			duration := time.Since(start)

			// 显示结果
			printResults(cmd, query, results, duration, opts)
		},
	}

	// 命令行参数
	cmd.Flags().IntVarP(&topK, "top", "k", 10, "返回结果数量")
	cmd.Flags().IntVarP(&window, "window", "w", 8, "上下文窗口大小（行数）")
	cmd.Flags().BoolVar(&localMode, "local", false, "使用本地嵌入模式")
	cmd.Flags().BoolVar(&enableExpansion, "expand", true, "启用查询扩展")
	cmd.Flags().BoolVar(&enableSkills, "skills", false, "启用技能系统")
	cmd.Flags().BoolVar(&forceReindex, "reindex", false, "强制重新索引")
	cmd.Flags().StringSliceVar(&includeGlobs, "include", []string{}, "包含的文件模式 (可多次使用)")
	cmd.Flags().StringSliceVar(&excludeGlobs, "exclude", []string{}, "排除的文件模式 (可多次使用)")

	return cmd
}

// printResults 打印搜索结果
func printResults(cmd *cobra.Command, query string, results []*SearchResult, duration time.Duration, opts *SearchOptions) {
	fmt.Printf("\n=== RAG 搜索结果 ===\n")
	fmt.Printf("查询: %s\n", query)
	fmt.Printf("耗时: %v\n", duration)
	fmt.Printf("配置: TopK=%d, Window=%d, Local=%t, Expand=%t, Skills=%t\n",
		opts.TopK, opts.Window, opts.LocalMode, opts.EnableExpansion, opts.EnableSkills)
	fmt.Printf("\n找到 %d 个结果:\n\n", len(results))

	if len(results) == 0 {
		fmt.Printf("未找到相关结果。建议:\n")
		fmt.Printf("  - 尝试使用不同的关键词\n")
		fmt.Printf("  - 检查文件路径是否正确\n")
		fmt.Printf("  - 使用 --expand 启用查询扩展\n")
		fmt.Printf("  - 使用 --skills 启用技能系统\n")
		return
	}

	for i, result := range results {
		fmt.Printf("%d. %s:%d (score=%.3f)\n", i+1, result.File, result.Line, result.Score)

		// 显示文件类型标签
		if result.FileType != "" {
			fmt.Printf("   [%s] ", result.FileType)
		}

		// 显示匹配原因
		if result.Reason != "" {
			fmt.Printf("原因: %s\n", result.Reason)
		}

		fmt.Printf("\n")

		// 显示代码片段
		if result.Snippet != "" {
			lines := formatSnippet(result.Snippet, result.Line, opts.Window)
			for _, line := range lines {
				fmt.Printf("   %s\n", line)
			}
		}

		fmt.Printf("---\n\n")
	}

	// 显示使用提示
	fmt.Printf("💡 提示:\n")
	if len(results) < opts.TopK {
		fmt.Printf("  - 结果较少，可以尝试 --expand 扩展查询\n")
	}
	fmt.Printf("  - 使用 --local 本地模式可能更快\n")
	fmt.Printf("  - 使用 --include/--exclude 精确控制搜索范围\n")
}

// formatSnippet 格式化代码片段显示
func formatSnippet(snippet string, centerLine, window int) []string {
	lines := []string{"   ..."}
	snippetLines := splitLines(snippet)

	// 找到目标行附近的上下文
	for i, line := range snippetLines {
		prefix := "   "
		if i == window {
			prefix = ">> " // 标记目标行
		}
		lines = append(lines, prefix+line)
	}

	lines = append(lines, "   ...")
	return lines
}

// splitLines 分割文本为行
func splitLines(text string) []string {
	if text == "" {
		return []string{}
	}

	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

