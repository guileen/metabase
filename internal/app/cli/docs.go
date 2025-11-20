package cli

import ("fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/guileen/metabase/internal/app/www")

var wwwCmd = &cobra.Command{
	Use:   "www",
	Short: "静态网站服务",
	Long:  `管理静态网站服务，支持Markdown、博客、文档等各种静态网站。`,
}

var wwwServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动静态网站服务",
	Long: `启动静态网站服务，支持Front Matter、自动导航、
搜索功能等，可用于博客、文档、官网等。`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		host, _ := cmd.Flags().GetString("host")
		config, _ := cmd.Flags().GetString("config")
		dev, _ := cmd.Flags().GetBool("dev")

		cfg := &www.Config{
			Port:     port,
			Host:     host,
			Config:   config,
			DevMode:  dev,
			RootDir:  "docs",
			TemplateDir: "templates",
			AssetDir: "web/assets",
		}

		if err := www.Serve(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "启动静态网站服务失败: %v\n", err)
			os.Exit(1)
		}
	},
}

var wwwBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "构建静态网站",
	Long: `将Markdown内容构建为静态HTML网站，
适合部署到CDN或静态托管服务。`,
	Run: func(cmd *cobra.Command, args []string) {
		output, _ := cmd.Flags().GetString("output")
		config, _ := cmd.Flags().GetString("config")

		cfg := &www.BuildConfig{
			OutputDir: output,
			Config:    config,
			RootDir:   "docs",
		}

		if err := www.Build(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "构建静态网站失败: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	// www serve 子命令
	wwwServeCmd.Flags().StringP("port", "p", "8080", "服务端口")
	wwwServeCmd.Flags().StringP("host", "H", "localhost", "绑定主机")
	wwwServeCmd.Flags().StringP("config", "c", "", "配置文件路径")
	wwwServeCmd.Flags().BoolP("dev", "d", false, "开发模式")

	// www build 子命令
	wwwBuildCmd.Flags().StringP("output", "o", "dist", "输出目录")
	wwwBuildCmd.Flags().StringP("config", "c", "", "配置文件路径")

	// 添加到 www 命令
	wwwCmd.AddCommand(wwwServeCmd)
	wwwCmd.AddCommand(wwwBuildCmd)
}