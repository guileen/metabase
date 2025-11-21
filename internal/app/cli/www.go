package cli

import (
	"fmt"
	"os"

	"github.com/guileen/metabase/internal/app/www"
	"github.com/spf13/cobra"
)

var wwwCmd = &cobra.Command{
	Use:   "www",
	Short: "启动 MetaBase 官网服务器",
	Long: `启动 MetaBase 官网服务器，提供文档和静态网站服务。

功能特性:
- 文档站点 (/docs/*)
- 搜索功能 (/search)
- 静态资源服务 (/assets/*)
- Markdown 渲染和 Front Matter 支持
- 响应式设计和主题支持

端口: 8080 (默认)
根目录: docs/
资源目录: web/assets/`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		host, _ := cmd.Flags().GetString("host")
		dev, _ := cmd.Flags().GetBool("dev")
		rootDir, _ := cmd.Flags().GetString("root")

		// Create www configuration
		config := &www.Config{
			Host:        host,
			Port:        port,
			DevMode:     dev,
			RootDir:     rootDir,
			TemplateDir: "web/templates",
			AssetDir:    "web/assets",
		}

		// Start server directly (www server doesn't have Stop method)
		fmt.Println("🚀 启动 MetaBase 官网服务器...")
		if err := www.Serve(config); err != nil {
			fmt.Fprintf(os.Stderr, "启动官网服务器失败: %v\n", err)
			os.Exit(1)
		}
	},
}

var wwwBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "构建静态网站",
	Long: `构建 MetaBase 静态网站，生成可部署的静态文件。

输出目录: dist/
输入目录: docs/`,
	Run: func(cmd *cobra.Command, args []string) {
		outputDir, _ := cmd.Flags().GetString("output")
		rootDir, _ := cmd.Flags().GetString("root")

		buildConfig := &www.BuildConfig{
			OutputDir: outputDir,
			RootDir:   rootDir,
		}

		fmt.Println("🏗️  构建静态网站...")
		if err := www.Build(buildConfig); err != nil {
			fmt.Fprintf(os.Stderr, "构建静态网站失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ 静态网站构建完成")
	},
}

func init() {
	wwwCmd.Flags().StringP("port", "p", "8080", "官网服务器端口")
	wwwCmd.Flags().StringP("host", "H", "localhost", "绑定主机")
	wwwCmd.Flags().BoolP("dev", "d", true, "开发模式")
	wwwCmd.Flags().String("root", "docs", "文档根目录")

	wwwBuildCmd.Flags().StringP("output", "o", "dist", "输出目录")
	wwwBuildCmd.Flags().String("root", "docs", "文档根目录")

	wwwCmd.AddCommand(wwwBuildCmd)
}