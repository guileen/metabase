package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/guileen/metabase/internal/app/admin"
	"github.com/spf13/cobra"
)

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "启动 MetaBase 管理后台服务器",
	Long: `启动 MetaBase 管理后台服务器，提供系统管理和监控功能。

功能特性:
- 管理界面 (/)
- 系统监控和指标 (/api/admin/*)
- 实时日志和事件处理
- 嵌入式 NATS 和 NRPC 服务
- WebSocket 实时通信

端口: 7680 (默认)
静态文件: web/admin/

与管理控制台 (console) 的区别:
- Admin: 管理后台界面，用于系统管理
- Console: 开发者工具，用于调试和监控`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		host, _ := cmd.Flags().GetString("host")
		dev, _ := cmd.Flags().GetBool("dev")
		staticFiles, _ := cmd.Flags().GetString("static")

		// Create admin configuration
		config := admin.NewConfig()
		config.Host = host
		config.Port = port
		config.DevMode = dev
		if staticFiles != "" {
			config.StaticFiles = staticFiles
		}

		// Service flags
		enableRealtime, _ := cmd.Flags().GetBool("realtime")
		authRequired, _ := cmd.Flags().GetBool("auth")

		config.EnableRealtime = enableRealtime
		config.AuthRequired = authRequired

		// Create and start admin server
		server, err := admin.NewServer(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "创建管理后台服务器失败: %v\n", err)
			os.Exit(1)
		}

		// Setup graceful shutdown
		go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			<-sigChan

			fmt.Println("\n🛑 正在优雅关闭管理后台服务器...")
			if err := server.Stop(); err != nil {
				fmt.Fprintf(os.Stderr, "关闭管理后台服务器时出错: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✅ 管理后台服务器已安全关闭")
			os.Exit(0)
		}()

		// Start server
		fmt.Println("🚀 启动 MetaBase 管理后台服务器...")
		if err := server.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "启动管理后台服务器失败: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	adminCmd.Flags().StringP("port", "p", "7680", "管理后台服务器端口")
	adminCmd.Flags().StringP("host", "H", "localhost", "绑定主机")
	adminCmd.Flags().BoolP("dev", "d", true, "开发模式")
	adminCmd.Flags().String("static", "web/admin", "静态文件目录")

	// Service flags
	adminCmd.Flags().Bool("realtime", true, "启用实时功能")
	adminCmd.Flags().Bool("auth", true, "启用身份验证")
}
