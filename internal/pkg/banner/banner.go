package banner

import (
	"fmt"
	"strings"
	"time"
)

// ANSI 颜色码
const (
	Reset         = "\033[0m"
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
	White         = "\033[97m"
	Dim           = "\033[2m"
	Bold          = "\033[1m"
)

// MetaBase ASCII Art Banner
var asciiArt = fmt.Sprintf(`
%s%s%s%s%s
%s██╗  ██╗███████╗████████╗ █████╗ ██╗     ██╗     %s
%s██║ ██╔╝██╔════╝╚══██╔══╝██╔══██╗██║     ██║     %s
%s█████╔╝ █████╗     ██║   ███████║██║     ██║     %s
%s██╔═██╗ ██╔══╝     ██║   ██╔══██║██║     ██║     %s
%s██║  ██╗███████╗   ██║   ██║  ██║███████╗███████╗%s
%s╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚══════╝%s
%s%s   %sM E T A B A S E%s   %s%s%s
`,
	BrightCyan, Bold, strings.Repeat(" ", 32), Reset, BrightCyan,
	BrightCyan, White, Reset,
	BrightCyan, White, Reset,
	BrightCyan, White, Reset,
	BrightCyan, White, Reset,
	BrightCyan, White, Reset,
	BrightCyan, White, Reset,
	Reset, strings.Repeat(" ", 25), BrightMagenta, Bold, Reset, strings.Repeat(" ", 25), Reset,
)

// StartupInfo 包含启动信息
type StartupInfo struct {
	Services    []ServiceInfo
	AccessLinks []AccessLink
	DevMode     bool
	StartTime   time.Time
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name   string
	Status string
	Port   string
	Color  string
}

// AccessLink 访问链接
type AccessLink struct {
	Name  string
	URL   string
	Desc  string
	Color string
}

// PrintBanner 打印 MetaBase Banner
func PrintBanner() {
	fmt.Println(asciiArt)
	fmt.Printf("%s%s🚀 MetaBase - 下一代后端核心%s\n", BrightGreen, Bold, Reset)
	fmt.Printf("%s%sVersion: 1.0.0 | Built with Go%s\n\n", Dim, BrightBlue, Reset)
}

// PrintStartupInfo 打印启动信息
func PrintStartupInfo(info *StartupInfo) {
	fmt.Printf("%s%s═══════════════════════════════════════════════════════════════%s\n", BrightCyan, Bold, Reset)
	fmt.Printf("%s%s🌟 服务状态%s\n", BrightYellow, Bold, Reset)
	fmt.Printf("%s%s═══════════════════════════════════════════════════════════════%s\n\n", BrightCyan, Bold, Reset)

	// 打印服务状态
	for _, service := range info.Services {
		status := "✅ 运行中"
		if service.Status != "running" {
			status = "❌ 停止"
		}
		fmt.Printf("  %s%-12s%s %s%-8s%s %s%s:%s%s\n",
			service.Color, service.Name, Reset,
			BrightGreen, status, Reset,
			BrightBlue, service.Port, Reset, Reset)
	}

	fmt.Printf("\n%s%s═══════════════════════════════════════════════════════════════%s\n", BrightCyan, Bold, Reset)
	fmt.Printf("%s%s🔗 访问地址%s\n", BrightMagenta, Bold, Reset)
	fmt.Printf("%s%s═══════════════════════════════════════════════════════════════%s\n\n", BrightCyan, Bold, Reset)

	// 打印访问链接
	for _, link := range info.AccessLinks {
		fmt.Printf("  %s%-16s%s %s%-40s%s %s%s%s\n",
			link.Color, link.Name, Reset,
			BrightCyan, link.URL, Reset,
			Dim, link.Desc, Reset)
	}

	if info.DevMode {
		fmt.Printf("\n%s%s═══════════════════════════════════════════════════════════════%s\n", BrightCyan, Bold, Reset)
		fmt.Printf("%s%s🛠️  开发模式%s %s已启用%s\n", BrightYellow, Bold, Reset, BrightGreen, Reset)
		fmt.Printf("%s%s═══════════════════════════════════════════════════════════════%s\n", BrightCyan, Bold, Reset)
	}

	// 启动时间
	duration := time.Since(info.StartTime)
	fmt.Printf("\n%s⏱️  启动耗时: %v%s\n", Dim, duration.Round(time.Millisecond), Reset)
	fmt.Printf("%s🎉 所有服务已就绪，开始您的 MetaBase 之旅！%s\n\n", BrightGreen, Bold, Reset)
}

// PrintServiceStartup 打印单个服务启动信息
func PrintServiceStartup(serviceName, port string) {
	fmt.Printf("  %s►%s 启动 %s%s%s 服务 (端口 %s%s%s)\n",
		BrightCyan, Reset,
		BrightYellow, serviceName, Reset,
		BrightBlue, port, Reset)
}

// PrintShutdown 打印关闭信息
func PrintShutdown() {
	fmt.Printf("\n%s%s🛑 正在优雅关闭 MetaBase...%s\n", BrightYellow, Bold, Reset)
	fmt.Printf("%s%s✅ MetaBase 已安全关闭%s\n", BrightGreen, Bold, Reset)
}

// PrintError 打印错误信息
func PrintError(message string) {
	fmt.Printf("%s%s❌ 错误: %s%s\n", BrightRed, Bold, message, Reset)
}

// PrintSuccess 打印成功信息
func PrintSuccess(message string) {
	fmt.Printf("%s%s✅ %s%s\n", BrightGreen, Bold, message, Reset)
}

// PrintWarning 打印警告信息
func PrintWarning(message string) {
	fmt.Printf("%s%s⚠️  %s%s\n", BrightYellow, Bold, message, Reset)
}
