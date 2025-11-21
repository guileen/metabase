package cli

import ("fmt"
    
    "github.com/spf13/cobra")

var rootCmd = &cobra.Command{
	Use:   "metabase",
	Short: "MetaBase - 下一代后端核心",
	Long: `MetaBase 是为一人公司与小团队打造的下一代后端核心，
目标是让 90% 的重复性后端工作消失。它以简单为先、
性能为纲、可观测为标配，让你专注业务表与前端策略。

三层架构:
- Gateway (网关): 统一入口和路由分发 (端口: 7609)
- API (接口): REST API 和业务逻辑 (端口: 7610)
- Admin (管理): 管理后台和监控工具 (端口: 7680)
- Website (官网): 文档和静态网站 (端口: 8080)

推荐使用方式:
- metabase gateway    # 启动所有服务 (推荐)
- metabase api        # 单独启动API服务
- metabase admin      # 单独启动管理后台
- metabase www        # 单独启动官网服务

默认行为: 显示帮助信息`,
	Version: "1.0.0",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(`🚀 MetaBase - 下一代后端核心

三层架构服务:

🌐 Gateway (网关) - 端口: 7609
   统一入口和路由分发，整合所有服务
   命令: metabase gateway

🚀 API (接口) - 端口: 7610
   REST API 和业务逻辑
   命令: metabase api

🔧 Admin (管理) - 端口: 7680
   管理后台和监控工具
   命令: metabase admin

📖 Website (官网) - 端口: 8080
   文档和静态网站服务
   命令: metabase www

使用 "metabase --help" 查看更多命令。`)
	},
}

func init() {
	// 添加新的三层架构命令
	rootCmd.AddCommand(gatewayCmd)
	rootCmd.AddCommand(apiCmd)
	rootCmd.AddCommand(adminCmd)
	rootCmd.AddCommand(wwwCmd)

	// 保持原有命令
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
