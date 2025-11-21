package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/guileen/metabase/internal/app/gateway"
	"github.com/guileen/metabase/internal/pkg/banner"
	"github.com/spf13/cobra"
)

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "启动 MetaBase 统一网关服务器",
	Long: `启动 MetaBase 统一网关服务器，整合 API、管理后台和官网服务。
这是主要的入口点，通过反向代理提供所有服务。

三层架构:
- 网关层 (7609): 统一入口和路由分发
- API层 (7610): REST API 和业务逻辑
- 管理层 (7680): 管理后台和监控工具
- 网站层 (8080): 官网和文档服务

服务说明:
- Gateway: 反向代理，统一入口点
- API: 完整的 REST API 接口
- Admin: 管理界面和监控工具
- Website: 官网和文档站点`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		host, _ := cmd.Flags().GetString("host")
		dev, _ := cmd.Flags().GetBool("dev")

		// Service ports
		apiPort, _ := cmd.Flags().GetString("api-port")
		adminPort, _ := cmd.Flags().GetString("admin-port")
		webPort, _ := cmd.Flags().GetString("web-port")

		// Service flags
		enableAPI, _ := cmd.Flags().GetBool("enable-api")
		enableAdmin, _ := cmd.Flags().GetBool("enable-admin")
		enableWeb, _ := cmd.Flags().GetBool("enable-web")

		// Create gateway configuration
		config := gateway.NewConfig()
		config.Host = host
		config.Port = port
		config.DevMode = dev
		config.APIPort = apiPort
		config.AdminPort = adminPort
		config.WebPort = webPort
		config.EnableAPI = enableAPI
		config.EnableAdmin = enableAdmin
		config.EnableWeb = enableWeb

		// Create and start gateway server
		server, err := gateway.NewServer(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "创建网关服务器失败: %v\n", err)
			os.Exit(1)
		}

		// Print startup banner
		banner.PrintBanner()

		// Setup graceful shutdown
		go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			<-sigChan

			banner.PrintShutdown()
			if err := server.Stop(); err != nil {
				banner.PrintError(fmt.Sprintf("关闭网关服务器时出错: %v", err))
				os.Exit(1)
			}
			os.Exit(0)
		}()

		// Start server
		banner.PrintServiceStartup("统一网关", port)
		if err := server.Start(); err != nil {
			banner.PrintError(fmt.Sprintf("启动网关服务器失败: %v", err))
			os.Exit(1)
		}
	},
}

func init() {
	gatewayCmd.Flags().StringP("port", "p", "7609", "网关服务器端口")
	gatewayCmd.Flags().StringP("host", "H", "0.0.0.0", "绑定主机")
	gatewayCmd.Flags().BoolP("dev", "d", true, "开发模式")

	// Service ports
	gatewayCmd.Flags().String("api-port", "7610", "API服务端口")
	gatewayCmd.Flags().String("admin-port", "7680", "管理后台端口")
	gatewayCmd.Flags().String("web-port", "8080", "官网服务端口")

	// Service control flags
	gatewayCmd.Flags().Bool("enable-api", true, "启用API服务")
	gatewayCmd.Flags().Bool("enable-admin", true, "启用管理后台服务")
	gatewayCmd.Flags().Bool("enable-web", true, "启用官网服务")
}
