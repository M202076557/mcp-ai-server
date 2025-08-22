package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// Client MCP客户端接口
type Client interface {
	// 连接管理
	Connect() error
	Disconnect() error
	IsConnected() bool

	// 基础操作
	Initialize() error
	Shutdown() error

	// 工具操作
	ListTools() ([]Tool, error)
	CallTool(name string, arguments map[string]interface{}) (*ToolCallResult, error)

	// 资源操作
	ReadResource(uri string) (*ResourceReadResult, error)
	WriteResource(uri string, content []Content) error
	ListResources(uri string) ([]string, error)

	// 事件处理
	OnMessage(handler func(*Message))
	OnError(handler func(error))
}

// StdioClient 基于stdio的MCP客户端
type StdioClient struct {
	reader   io.Reader
	writer   io.Writer
	conn     *stdioConnection
	mu       sync.RWMutex
	handlers struct {
		message func(*Message)
		error   func(error)
	}
}

// NewStdioClient 创建新的stdio客户端
func NewStdioClient(reader io.Reader, writer io.Writer) *StdioClient {
	return &StdioClient{
		reader: reader,
		writer: writer,
	}
}

// Connect 连接到MCP服务器
func (c *StdioClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.conn = newStdioConnection(c.reader, c.writer)
	return c.conn.Start()
}

// Disconnect 断开连接
func (c *StdioClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return c.conn.Stop()
	}
	return nil
}

// IsConnected 检查是否已连接
func (c *StdioClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.conn != nil && c.conn.IsRunning()
}

// Initialize 初始化客户端
func (c *StdioClient) Initialize() error {
	if !c.IsConnected() {
		return fmt.Errorf("客户端未连接")
	}

	params := InitializeParams{
		ProtocolVersion: ProtocolVersion,
		Capabilities: map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": true,
			},
			"resources": map[string]interface{}{
				"listChanged": true,
			},
		},
		ClientInfo: &ClientInfo{
			Name:    "stdio-mcp-client",
			Version: "1.0.0",
		},
	}

	paramsBytes, _ := json.Marshal(params)
	msg := Message{
		JSONRPC: JSONRPCVersion,
		ID:      "init-1",
		Method:  "initialize",
		Params:  paramsBytes,
	}

	return c.conn.SendMessage(&msg)
}

// Shutdown 关闭客户端
func (c *StdioClient) Shutdown() error {
	if !c.IsConnected() {
		return nil
	}

	msg := Message{
		JSONRPC: JSONRPCVersion,
		ID:      "shutdown-1",
		Method:  "shutdown",
	}

	err := c.conn.SendMessage(&msg)
	if err != nil {
		return err
	}

	// 等待一下让服务器处理关闭请求
	time.Sleep(100 * time.Millisecond)

	return c.Disconnect()
}

// ListTools 获取工具列表
func (c *StdioClient) ListTools() ([]Tool, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("客户端未连接")
	}

	msg := Message{
		JSONRPC: JSONRPCVersion,
		ID:      "tools-1",
		Method:  "tools/list",
	}

	// 发送请求
	if err := c.conn.SendMessage(&msg); err != nil {
		return nil, err
	}

	// 等待响应
	response := c.conn.WaitForResponse(msg.ID, 5*time.Second)
	if response == nil {
		return nil, fmt.Errorf("获取工具列表超时")
	}

	if response.Error != nil {
		return nil, fmt.Errorf("获取工具列表失败: %s", response.Error.Message)
	}

	// 解析响应
	var result struct {
		Tools []Tool `json:"tools"`
	}

	if response.Result != nil {
		resultBytes, _ := json.Marshal(response.Result)
		if err := json.Unmarshal(resultBytes, &result); err != nil {
			return nil, fmt.Errorf("解析工具列表失败: %v", err)
		}
	}

	return result.Tools, nil
}

// CallTool 调用工具
func (c *StdioClient) CallTool(name string, arguments map[string]interface{}) (*ToolCallResult, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("客户端未连接")
	}

	params := ToolCallParams{
		Name:      name,
		Arguments: arguments,
	}

	paramsBytes, _ := json.Marshal(params)
	msg := Message{
		JSONRPC: JSONRPCVersion,
		ID:      fmt.Sprintf("call-%d", time.Now().Unix()),
		Method:  "tools/call",
		Params:  paramsBytes,
	}

	// 发送请求
	if err := c.conn.SendMessage(&msg); err != nil {
		return nil, err
	}

	// 等待响应
	response := c.conn.WaitForResponse(msg.ID, 10*time.Second)
	if response == nil {
		return nil, fmt.Errorf("调用工具超时")
	}

	if response.Error != nil {
		return nil, fmt.Errorf("调用工具失败: %s", response.Error.Message)
	}

	// 解析响应
	var result ToolCallResult
	if response.Result != nil {
		resultBytes, _ := json.Marshal(response.Result)
		if err := json.Unmarshal(resultBytes, &result); err != nil {
			return nil, fmt.Errorf("解析工具调用结果失败: %v", err)
		}
	}

	return &result, nil
}

// ReadResource 读取资源
func (c *StdioClient) ReadResource(uri string) (*ResourceReadResult, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("客户端未连接")
	}

	params := ResourceReadParams{
		URI: uri,
	}

	paramsBytes, _ := json.Marshal(params)
	msg := Message{
		JSONRPC: JSONRPCVersion,
		ID:      fmt.Sprintf("read-%d", time.Now().Unix()),
		Method:  "resources/read",
		Params:  paramsBytes,
	}

	// 发送请求
	if err := c.conn.SendMessage(&msg); err != nil {
		return nil, err
	}

	// 等待响应
	response := c.conn.WaitForResponse(msg.ID, 5*time.Second)
	if response == nil {
		return nil, fmt.Errorf("读取资源超时")
	}

	if response.Error != nil {
		return nil, fmt.Errorf("读取资源失败: %s", response.Error.Message)
	}

	// 解析响应
	var result ResourceReadResult
	if response.Result != nil {
		resultBytes, _ := json.Marshal(response.Result)
		if err := json.Unmarshal(resultBytes, &result); err != nil {
			return nil, fmt.Errorf("解析资源读取结果失败: %v", err)
		}
	}

	return &result, nil
}

// WriteResource 写入资源
func (c *StdioClient) WriteResource(uri string, content []Content) error {
	if !c.IsConnected() {
		return fmt.Errorf("客户端未连接")
	}

	// 这里需要实现resources/write方法
	// 暂时返回未实现错误
	return fmt.Errorf("写入资源功能尚未实现")
}

// ListResources 列出资源
func (c *StdioClient) ListResources(uri string) ([]string, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("客户端未连接")
	}

	// 这里需要实现resources/list方法
	// 暂时返回未实现错误
	return nil, fmt.Errorf("列出资源功能尚未实现")
}

// OnMessage 设置消息处理器
func (c *StdioClient) OnMessage(handler func(*Message)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handlers.message = handler
}

// OnError 设置错误处理器
func (c *StdioClient) OnError(handler func(error)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handlers.error = handler
}

// 触发消息处理器
func (c *StdioClient) triggerMessageHandler(msg *Message) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.handlers.message != nil {
		c.handlers.message(msg)
	}
}

// 触发错误处理器
func (c *StdioClient) triggerErrorHandler(err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.handlers.error != nil {
		c.handlers.error(err)
	}
}
