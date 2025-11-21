package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/guileen/metabase/internal/app/api"
	"github.com/spf13/cobra"
)

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "启动 MetaBase API 服务器",
	Long: `启动 MetaBase API 服务器，提供完整的 REST API 接口。

功能特性:
- RESTful API 接口 (/api/v1/*)
- JWT 认证和授权
- 数据存储和检索
- 用户和租户管理
- 文件上传和管理
- 搜索和查询功能

端口: 7610 (默认)
API版本: v1`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		host, _ := cmd.Flags().GetString("host")
		dev, _ := cmd.Flags().GetBool("dev")
		dbPath, _ := cmd.Flags().GetString("db")

		// Create API configuration
		config := api.NewConfig()
		config.Host = host
		config.Port = port
		config.DevMode = dev
		config.DatabasePath = dbPath

		// Create and start API server
		server, err := api.NewServer(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "创建API服务器失败: %v\n", err)
			os.Exit(1)
		}

		// Setup graceful shutdown
		go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			<-sigChan

			fmt.Println("\n🛑 正在优雅关闭API服务器...")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := server.Stop(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "关闭API服务器时出错: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✅ API服务器已安全关闭")
			os.Exit(0)
		}()

		// Start server
		fmt.Println("🚀 启动 MetaBase API 服务器...")
		if err := server.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "启动API服务器失败: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	apiCmd.Flags().StringP("port", "p", "7610", "API服务器端口")
	apiCmd.Flags().StringP("host", "H", "localhost", "绑定主机")
	apiCmd.Flags().BoolP("dev", "d", true, "开发模式")
	apiCmd.Flags().String("db", "./data/metabase.db", "数据库文件路径")
}
