package cli

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/metabase/metabase/internal/core"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "metabase",
	Short: "MetaBase - 下一代后端核心",
	Long: `MetaBase 是为一人公司与小团队打造的下一代后端核心，
目标是让 90% 的重复性后端工作消失。它以简单为先、
性能为纲、可观测为标配，让你专注业务表与前端策略。

默认启动静态网站服务，可用于文档、博客、官网等。`,
	Version: "1.0.0",
	Run: func(cmd *cobra.Command, args []string) {
		// 默认启动核心服务器
		config := core.NewConfig()
		config.Port = "7609"
		config.Host = "localhost"
		config.DevMode = true

		server, err := core.NewServer(config)
		if err != nil {
			cmd.PrintErrf("创建核心服务器失败: %v\n", err)
			return
		}

		// Setup graceful shutdown
		go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			<-sigChan

			cmd.PrintErrln("\n🛑 正在关闭服务器...")
			if err := server.Stop(); err != nil {
				cmd.PrintErrf("关闭服务器时出错: %v\n", err)
			}
			cmd.PrintErrln("✅ 服务器已关闭")
		}()

		// Start server
		if err := server.Start(); err != nil {
			cmd.PrintErrf("启动核心服务器失败: %v\n", err)
		}
	},
}

func init() {
	// 添加子命令
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(wwwCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
}

// Run 执行CLI
func Run() error {
	return rootCmd.Execute()
}

// AddCommand 添加命令
func AddCommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}
