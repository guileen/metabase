package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Long:  `显示 MetaBase 的版本信息、构建信息等。`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("MetaBase v1.0.0\n")
		fmt.Printf("Go Version: %s\n", "1.21+")
		fmt.Printf("Build: %s\n", "development")
		fmt.Printf("Commit: %s\n", "unknown")
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "配置管理",
	Long:  `管理 MetaBase 的配置文件、环境变量等。`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化配置文件",
	Long:  `创建默认的配置文件到当前目录。`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("初始化配置文件...")
		// TODO: 实现配置文件初始化
	},
}

func init() {
	configCmd.AddCommand(configInitCmd)
}