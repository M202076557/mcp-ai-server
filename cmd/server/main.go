package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"mcp-ai-server/internal/mcp"
	"mcp-ai-server/internal/tools"
)

func main() {
	// 解析命令行参数
	var (
		help = flag.Bool("help", false, "显示帮助信息")
		port = flag.Int("port", 8081, "WebSocket端口（如果使用WebSocket模式）")
		mode = flag.String("mode", "stdio", "运行模式：stdio 或 websocket")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// 创建工具管理器
	toolManager, err := tools.NewToolManager("configs/config.yaml")
	if err != nil {
		log.Fatalf("创建工具管理器失败: %v", err)
	}

	// 根据模式创建服务器
	var server mcp.Server

	switch *mode {
	case "stdio":
		// 在stdio模式下，将日志输出到stderr避免干扰JSON通信
		log.SetOutput(os.Stderr)
		server, err = createStdioServer(toolManager)
	case "websocket":
		server, err = createWebSocketServer(toolManager, *port)
	default:
		log.Fatalf("不支持的运行模式: %s", *mode)
	}

	if err != nil {
		log.Fatalf("创建服务器失败: %v", err)
	}

	// 启动服务器
	if err := server.Start(); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}

	// 等待中断信号
	waitForInterrupt()

	// 停止服务器
	if err := server.Stop(); err != nil {
		log.Printf("停止服务器失败: %v", err)
	}

	log.Println("服务器已停止")
}

// createStdioServer 创建stdio服务器
func createStdioServer(toolManager *tools.ToolManager) (mcp.Server, error) {
	stdioServer := mcp.NewStdioServer(os.Stdin, os.Stdout)

	// 注册所有工具
	for _, tool := range toolManager.GetTools() {
		if err := stdioServer.RegisterTool(tool); err != nil {
			return nil, fmt.Errorf("注册工具失败: %v", err)
		}
		log.Printf("已注册工具: %s", tool.Name)
	}

	// 设置工具执行器
	stdioServer.SetToolExecutor(toolManager)
	log.Println("已设置工具执行器")

	return stdioServer, nil
}

// createWebSocketServer 创建WebSocket服务器
func createWebSocketServer(toolManager *tools.ToolManager, port int) (mcp.Server, error) {
	websocketServer := mcp.NewWebSocketServer(port)
	if websocketServer == nil {
		return nil, fmt.Errorf("创建WebSocket服务器失败")
	}

	// 注册所有工具
	for _, tool := range toolManager.GetTools() {
		if err := websocketServer.RegisterTool(tool); err != nil {
			return nil, fmt.Errorf("注册工具失败: %v", err)
		}
		log.Printf("已注册工具: %s", tool.Name)
	}

	// 设置工具执行器
	websocketServer.SetToolExecutor(toolManager)
	log.Println("已设置工具执行器")

	return websocketServer, nil
}

// showHelp 显示帮助信息
func showHelp() {
	fmt.Println(`
	╔══════════════════════════════════════════════════════════════════════════════╗
	║                         MCP AI Server                                       ║
	║                        Model Context Protocol                               ║
	╚══════════════════════════════════════════════════════════════════════════════╝

	📖 用法:
	mcp-server [选项]

	🔧 选项:
	-help    显示此帮助信息
	-mode    运行模式 (stdio|websocket) [默认: stdio]
	-port    WebSocket端口 [默认: 8081]
	-config  配置文件路径 [默认: configs/config.yaml]

	🌐 运行模式:
	stdio     通过标准输入输出通信，适合本地集成和CLI工具
	websocket 通过网络WebSocket通信，支持远程访问和API集成

	💡 使用示例:
	mcp-server                           # 使用默认stdio模式
	mcp-server -mode websocket          # 使用WebSocket模式
	mcp-server -mode websocket -port 9000  # 指定WebSocket端口
	mcp-server -config ./my-config.yaml    # 使用自定义配置文件

	🛠️ 可用工具列表:

	🔧 系统工具:
	• file_read        读取文件内容
	• file_write       写入文件内容
	• command_execute  执行系统命令
	• directory_list   列出目录内容

	🌐 网络工具:
	• http_get        发送HTTP GET请求
	• http_post       发送HTTP POST请求
	• ping            检查网络连通性
	• dns_lookup      DNS域名解析

	📊 数据处理工具:
	• json_parse      JSON解析和格式化
	• json_validate   JSON格式验证
	• base64_encode   Base64编码
	• base64_decode   Base64解码
	• hash            计算哈希值
	• text_transform  文本格式转换

	🗄️ 数据库工具:
	• db_connect      连接到数据库
	• db_query        执行数据库查询
	• db_execute      执行数据库操作

	🤖 AI工具 (Ollama集成):
	• ai_query        使用AI进行智能查询和回答
	• ai_analyze_data 使用AI分析数据并提供洞察
	• ai_generate_query 根据自然语言描述生成SQL查询

	⚙️ 配置说明:
	配置文件: configs/config.yaml
	支持热重载: 否 (需要重启服务器)
	默认端口: 8081 (WebSocket模式)
	健康检查: http://localhost:8081/health (WebSocket模式)

	🔒 安全特性:
	• 路径访问控制
	• 命令执行白名单
	• 文件类型过滤
	• 资源使用限制
	• 超时保护

	📈 监控功能:
	• 健康检查端点
	• 性能指标收集
	• 连接状态监控
	• 错误日志记录

	🌍 环境支持:
	• 开发环境: 调试模式、详细日志
	• 生产环境: 安全严格、性能优化

	📚 更多信息:
	• 项目文档: README.md
	• MCP协议: https://modelcontextprotocol.io/
	• Ollama: https://ollama.ai/

	💡 提示:
	• 首次使用建议先启动stdio模式测试
	• WebSocket模式需要确保端口未被占用
	• AI功能需要本地运行Ollama服务
	• 生产环境请仔细配置安全策略
	`)
}

// waitForInterrupt 等待中断信号
func waitForInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Println("收到中断信号，正在停止服务器...")
}
