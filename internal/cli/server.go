package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/metabase/metabase/internal/core"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "启动 MetaBase 核心服务器",
	Long: `启动 MetaBase 核心服务器，提供完整的后端服务功能。
包括 NRPC 消息队列、存储引擎、控制台等核心组件。

组件说明:
- NRPC: 基于 NATS 的 RPC 与任务队列
- 存储引擎: SQLite + Pebble 组合存储
- 控制台: 监控、日志、性能统计
- 统一网关: HTTP API、静态文件服务`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		host, _ := cmd.Flags().GetString("host")
		dev, _ := cmd.Flags().GetBool("dev")
		consolePort, _ := cmd.Flags().GetString("console-port")

		// Enable/disable components
		enableNRPC, _ := cmd.Flags().GetBool("enable-nrpc")
		enableStorage, _ := cmd.Flags().GetBool("enable-storage")
		enableConsole, _ := cmd.Flags().GetBool("enable-console")

		// Create core configuration
		config := core.NewConfig()
		config.Host = host
		config.Port = port
		config.DevMode = dev
		config.Console.Port = consolePort

		// Override component settings
		config.EnableNRPC = enableNRPC
		config.EnableStorage = enableStorage
		config.EnableConsole = enableConsole

		// Create and start server
		server, err := core.NewServer(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "创建服务器失败: %v\n", err)
			os.Exit(1)
		}

		// Setup graceful shutdown
		go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			<-sigChan

			fmt.Println("\n🛑 正在优雅关闭服务器...")
			if err := server.Stop(); err != nil {
				fmt.Fprintf(os.Stderr, "关闭服务器时出错: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✅ 服务器已安全关闭")
			os.Exit(0)
		}()

		// Start server
		fmt.Println("🚀 启动 MetaBase 核心服务器...")
		if err := server.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "启动服务器失败: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	serverCmd.Flags().StringP("port", "p", "7609", "核心服务器端口")
	serverCmd.Flags().StringP("host", "H", "0.0.0.0", "绑定主机")
	serverCmd.Flags().BoolP("dev", "d", true, "开发模式")
	serverCmd.Flags().String("console-port", "7610", "控制台端口")

	// Component control flags
	serverCmd.Flags().Bool("enable-nrpc", true, "启用 NRPC 消息队列")
	serverCmd.Flags().Bool("enable-storage", true, "启用存储引擎")
	serverCmd.Flags().Bool("enable-console", true, "启用控制台")
}