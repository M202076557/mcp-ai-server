package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"mcp-ai-server/internal/mcp"
)

// ImprovedMCPClient 改进的MCP客户端
type ImprovedMCPClient struct {
	serverCmd    *exec.Cmd
	userInput    io.Reader
	mcpOutput    io.Writer
	serverInput  io.WriteCloser
	serverOutput io.ReadCloser
	mu           sync.RWMutex
	running      bool
	responses    map[interface{}]*mcp.Message
	requestID    int64 // 添加请求ID计数器
}

// NewImprovedMCPClient 创建新的改进客户端
func NewImprovedMCPClient() *ImprovedMCPClient {
	return &ImprovedMCPClient{
		userInput: os.Stdin,  // 用户命令从stdin读取
		mcpOutput: os.Stdout, // MCP消息输出到stdout
		responses: make(map[interface{}]*mcp.Message),
		requestID: 1, // 初始化请求ID
	}
}

// StartServer 启动MCP服务器
func (c *ImprovedMCPClient) StartServer() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("服务器已在运行")
	}

	fmt.Println("🚀 启动MCP服务器...")

	// 启动服务器进程
	c.serverCmd = exec.Command("./bin/mcp-server")

	// 创建管道
	serverIn, err := c.serverCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("创建stdin管道失败: %v", err)
	}

	serverOut, err := c.serverCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("创建stdout管道失败: %v", err)
	}

	c.serverInput = serverIn
	c.serverOutput = serverOut

	// 启动服务器
	if err := c.serverCmd.Start(); err != nil {
		return fmt.Errorf("启动服务器失败: %v", err)
	}

	c.running = true
	fmt.Println("✅ MCP服务器已启动")

	// 启动MCP消息处理循环
	go c.handleMCPMessages()

	// 等待服务器初始化
	time.Sleep(500 * time.Millisecond)

	return nil
}

// StopServer 停止MCP服务器
func (c *ImprovedMCPClient) StopServer() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	fmt.Println("🛑 停止MCP服务器...")
	c.running = false

	if c.serverCmd != nil && c.serverCmd.Process != nil {
		return c.serverCmd.Process.Kill()
	}

	return nil
}

// handleMCPMessages 处理来自服务器的MCP消息
func (c *ImprovedMCPClient) handleMCPMessages() {
	scanner := bufio.NewScanner(c.serverOutput)
	for c.isRunning() {
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var msg mcp.Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Printf("解析MCP消息失败: %v", err)
			continue
		}

		// 处理MCP消息
		c.processMCPMessage(&msg)
	}
}

// processMCPMessage 处理单个MCP消息
func (c *ImprovedMCPClient) processMCPMessage(msg *mcp.Message) {
	if msg.IsResponse() {
		// 将ID转换为字符串以确保类型一致性
		idStr := fmt.Sprintf("%v", msg.ID)

		// 存储响应，供后续使用
		c.mu.Lock()
		c.responses[idStr] = msg
		c.mu.Unlock()

		// 显示响应
		if msg.Error != nil {
			fmt.Printf("❌ 服务器错误: %s\n", msg.Error.Message)
		} else {
			fmt.Printf("✅ 服务器响应: %v\n", msg.Result)
		}
	} else if msg.IsNotification() {
		fmt.Printf("📢 服务器通知: %s\n", msg.Method)
	}
}

// isRunning 检查是否正在运行
func (c *ImprovedMCPClient) isRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// generateRequestID 生成唯一的请求ID
func (c *ImprovedMCPClient) generateRequestID() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := c.requestID
	c.requestID++
	return id
}

// SendMCPMessage 发送MCP消息到服务器
func (c *ImprovedMCPClient) SendMCPMessage(msg *mcp.Message) error {
	if !c.isRunning() {
		return fmt.Errorf("服务器未运行")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	data = append(data, '\n')
	_, err = c.serverInput.Write(data)
	if err != nil {
		return fmt.Errorf("发送消息失败: %v", err)
	}

	return nil
}

// WaitForResponse 等待指定ID的响应
func (c *ImprovedMCPClient) WaitForResponse(id interface{}, timeout time.Duration) *mcp.Message {
	start := time.Now()
	for time.Since(start) < timeout {
		c.mu.RLock()

		// 将ID转换为字符串以确保类型一致性
		idStr := fmt.Sprintf("%v", id)

		if response, exists := c.responses[idStr]; exists && response != nil {
			c.mu.RUnlock()

			// 清理响应
			c.mu.Lock()
			delete(c.responses, idStr)
			c.mu.Unlock()

			return response
		}
		c.mu.RUnlock()

		time.Sleep(10 * time.Millisecond)
	}

	return nil
}

// Initialize 初始化客户端
func (c *ImprovedMCPClient) Initialize() error {
	fmt.Println("📋 发送初始化请求...")

	params := mcp.InitializeParams{
		ProtocolVersion: mcp.ProtocolVersion,
		Capabilities: map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": true,
			},
			"resources": map[string]interface{}{
				"listChanged": true,
			},
		},
		ClientInfo: &mcp.ClientInfo{
			Name:    "improved-mcp-client",
			Version: "1.0.0",
		},
	}

	requestID := c.generateRequestID()
	msg := mcp.NewRequest(requestID, "initialize", params)

	if err := c.SendMCPMessage(msg); err != nil {
		return fmt.Errorf("发送初始化消息失败: %v", err)
	}

	fmt.Printf("✓ 初始化请求已发送 (ID: %d)\n", requestID)

	// 等待响应
	response := c.WaitForResponse(requestID, 5*time.Second)
	if response == nil {
		return fmt.Errorf("初始化超时")
	}

	if response.Error != nil {
		return fmt.Errorf("初始化失败: %s", response.Error.Message)
	}

	fmt.Println("✓ 初始化成功")
	return nil
}

// ListTools 获取工具列表
func (c *ImprovedMCPClient) ListTools() error {
	fmt.Println("🔧 获取工具列表...")

	requestID := c.generateRequestID()
	msg := mcp.NewRequest(requestID, "tools/list", nil)

	if err := c.SendMCPMessage(msg); err != nil {
		return fmt.Errorf("发送工具列表请求失败: %v", err)
	}

	fmt.Printf("✓ 工具列表请求已发送 (ID: %d)\n", requestID)

	// 等待响应
	response := c.WaitForResponse(requestID, 5*time.Second)
	if response == nil {
		return fmt.Errorf("获取工具列表超时")
	}

	if response.Error != nil {
		return fmt.Errorf("获取工具列表失败: %s", response.Error.Message)
	}

	fmt.Println("✓ 工具列表获取成功")

	// 解析工具列表
	if response.Result != nil {
		resultBytes, _ := json.Marshal(response.Result)
		var result struct {
			Tools []mcp.Tool `json:"tools"`
		}

		if err := json.Unmarshal(resultBytes, &result); err == nil {
			fmt.Printf("   发现 %d 个工具:\n", len(result.Tools))
			for _, tool := range result.Tools {
				fmt.Printf("   - %s: %s\n", tool.Name, tool.Description)
			}
		}
	}

	return nil
}

// CallTool 调用工具
func (c *ImprovedMCPClient) CallTool(name string, arguments map[string]interface{}) error {
	fmt.Printf("⚡ 调用工具: %s\n", name)

	params := mcp.ToolCallParams{
		Name:      name,
		Arguments: arguments,
	}

	requestID := c.generateRequestID()
	msg := mcp.NewRequest(requestID, "tools/call", params)

	if err := c.SendMCPMessage(msg); err != nil {
		return fmt.Errorf("发送工具调用请求失败: %v", err)
	}

	fmt.Printf("✓ 工具调用请求已发送: %s (ID: %d)\n", name, requestID)

	// 等待响应 - AI工具需要更长的处理时间
	timeout := 10 * time.Second
	if strings.HasPrefix(name, "ai_") {
		timeout = 60 * time.Second // AI工具给60秒超时
	}

	response := c.WaitForResponse(requestID, timeout)
	if response == nil {
		return fmt.Errorf("工具调用超时（%v）", timeout)
	}

	if response.Error != nil {
		return fmt.Errorf("工具调用失败: %s", response.Error.Message)
	}

	fmt.Printf("✓ 工具调用成功: %s\n", name)

	// 显示结果
	if response.Result != nil {
		resultBytes, _ := json.Marshal(response.Result)
		var result mcp.ToolCallResult

		if err := json.Unmarshal(resultBytes, &result); err == nil {
			for _, content := range result.Content {
				fmt.Printf("   结果: %s\n", content.Text)
			}
		}
	}

	return nil
}

// ReadResource 读取资源
func (c *ImprovedMCPClient) ReadResource(uri string) error {
	fmt.Printf("📖 读取资源: %s\n", uri)

	params := mcp.ResourceReadParams{
		URI: uri,
	}

	requestID := c.generateRequestID()
	msg := mcp.NewRequest(requestID, "resources/read", params)

	if err := c.SendMCPMessage(msg); err != nil {
		return fmt.Errorf("发送资源读取请求失败: %v", err)
	}

	fmt.Printf("✓ 资源读取请求已发送: %s (ID: %d)\n", uri, requestID)

	// 等待响应
	response := c.WaitForResponse(requestID, 5*time.Second)
	if response == nil {
		return fmt.Errorf("资源读取超时")
	}

	if response.Error != nil {
		return fmt.Errorf("资源读取失败: %s", response.Error.Message)
	}

	fmt.Printf("✓ 资源读取成功: %s\n", uri)

	// 显示结果
	if response.Result != nil {
		resultBytes, _ := json.Marshal(response.Result)
		var result mcp.ResourceReadResult

		if err := json.Unmarshal(resultBytes, &result); err == nil {
			fmt.Printf("   MIME类型: %s\n", result.MimeType)
			for _, content := range result.Contents {
				fmt.Printf("   内容: %s\n", content.Text)
			}
		}
	}

	return nil
}

// Shutdown 关闭客户端
func (c *ImprovedMCPClient) Shutdown() error {
	fmt.Println("🔌 关闭客户端...")

	requestID := c.generateRequestID()
	msg := mcp.NewRequest(requestID, "shutdown", nil)

	if err := c.SendMCPMessage(msg); err != nil {
		return fmt.Errorf("发送关闭请求失败: %v", err)
	}

	fmt.Printf("✓ 关闭请求已发送 (ID: %d)\n", requestID)

	// 等待一下让服务器处理
	time.Sleep(200 * time.Millisecond)

	return nil
}

// InteractiveMode 交互模式
func (c *ImprovedMCPClient) InteractiveMode() {
	fmt.Println("MCP客户端交互模式")
	fmt.Println("输入 'help' 查看可用命令")
	fmt.Println("输入 'quit' 退出")

	scanner := bufio.NewScanner(c.userInput)
	for {
		fmt.Print("mcp> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}

		command := parts[0]
		args := parts[1:]

		switch command {
		case "help":
			c.showHelp()
		case "init":
			if err := c.Initialize(); err != nil {
				log.Printf("初始化失败: %v", err)
			}
		case "tools":
			if err := c.ListTools(); err != nil {
				log.Printf("获取工具列表失败: %v", err)
			}
		case "call":
			if len(args) < 1 {
				fmt.Println("用法: call <工具名> [参数名:值]...")
				fmt.Println("示例: call file_read path:README.md")
				fmt.Println("      call command_execute command:ls args:-la")
				fmt.Println("      call file_write path:test.txt content:\"Hello World\"")
				fmt.Println("注意: 包含空格或换行的内容请用引号包围")
				fmt.Println("对于多行内容，请使用单行并用\\n表示换行")
				continue
			}
			toolName := args[0]
			arguments := make(map[string]interface{})

			// 分类处理不同操作类型的参数解析
			inputAfterTool := input[strings.Index(input, toolName)+len(toolName):]
			inputAfterTool = strings.TrimSpace(inputAfterTool)

			// 简化的参数解析：添加description支持
			// 已知的参数名列表
			knownParams := []string{"driver", "dsn", "alias", "sql", "limit", "path", "command", "args", "url", "host", "count", "prompt", "model", "provider", "description", "data", "analysis_type"}

			// 使用正则表达式匹配 key:value key:"value" key:'value' 模式
			paramPattern := `(\w+):((?:"[^"]*"|'[^']*'|[^\s]+))`
			re, _ := regexp.Compile(paramPattern)
			matches := re.FindAllStringSubmatch(inputAfterTool, -1)

			for _, match := range matches {
				if len(match) == 3 {
					key := match[1]
					value := match[2]

					// 检查是否是已知参数
					isKnownParam := false
					for _, param := range knownParams {
						if key == param {
							isKnownParam = true
							break
						}
					}

					if isKnownParam {
						// 去掉引号
						if (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) ||
							(strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) {
							value = value[1 : len(value)-1]
						}

						// 处理数值类型
						numRe, _ := regexp.Compile(`^-?\d+$`)
						if numRe.MatchString(value) {
							var n int
							fmt.Sscanf(value, "%d", &n)
							arguments[key] = n
						} else {
							arguments[key] = value
						}
					}
				}
			}

			// 调试信息：显示最终参数
			fmt.Printf("🔍 调试: 最终参数: %v\n", arguments)

			if err := c.CallTool(toolName, arguments); err != nil {
				log.Printf("调用工具失败: %v", err)
			}
		case "read":
			if len(args) < 1 {
				fmt.Println("用法: read <资源URI>")
				continue
			}
			uri := args[0]
			if err := c.ReadResource(uri); err != nil {
				log.Printf("读取资源失败: %v", err)
			}
		case "quit", "exit":
			fmt.Println("正在退出...")
			if err := c.Shutdown(); err != nil {
				log.Printf("关闭失败: %v", err)
			}
			return
		default:
			fmt.Printf("未知命令: %s\n", command)
			fmt.Println("输入 'help' 查看可用命令")
		}
	}
}

// showHelp 显示帮助信息
func (c *ImprovedMCPClient) showHelp() {
	fmt.Println(`可用命令:
	help             显示此帮助信息
	init             初始化客户端
	tools            获取可用工具列表
	call <工具名>    调用指定工具
	read <URI>       读取指定资源
	quit/exit        退出客户端

	工具分类:
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

	🔢 数据处理工具:
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

	🤖 AI工具:
		• ai_query        使用AI进行智能查询和回答
		• ai_analyze_data 使用AI分析数据并提供洞察
		• ai_generate_query 根据自然语言描述生成SQL查询

	使用示例:
	init                    # 初始化客户端
	tools                   # 获取工具列表

	# 系统工具
	call file_read path:README.md
	call directory_list path:.
	call command_execute command:ls args:-la

	# 网络工具
	call http_get url:https://httpbin.org/get
	call ping host:google.com count:3

	# 数据处理工具
	call json_parse json_string:'{"test": "value"}' pretty:true
	call base64_encode text:"Hello World"

	# AI工具
	call ai_query prompt:"解释什么是MCP协议"
	call ai_analyze_data data:'{"data": [1,2,3]}' analysis_type:summary

	# 资源读取
	read file://README.md

	详细测试命令请参考: demo/test/mcp_tools_test_commands.md
	`)
}

func main() {
	client := NewImprovedMCPClient()

	// 确保在程序结束时停止服务器
	defer func() {
		if err := client.StopServer(); err != nil {
			log.Printf("停止服务器失败: %v", err)
		}
	}()

	// 启动服务器
	if err := client.StartServer(); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}

	// 启动交互模式
	client.InteractiveMode()
}
