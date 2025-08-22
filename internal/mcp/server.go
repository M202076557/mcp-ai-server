package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
)

// Server MCP服务器接口
type Server interface {
	// 启动服务器
	Start() error

	// 停止服务器
	Stop() error

	// 注册工具
	RegisterTool(tool Tool) error

	// 注册资源处理器
	RegisterResourceHandler(scheme string, handler ResourceHandler) error
}

// ResourceHandler 资源处理器接口
type ResourceHandler interface {
	Read(ctx context.Context, uri string) (*ResourceReadResult, error)
	Write(ctx context.Context, uri string, content []Content) error
	List(ctx context.Context, uri string) ([]string, error)
}

// ToolExecutor 工具执行器接口
type ToolExecutor interface {
	ExecuteTool(ctx context.Context, name string, arguments map[string]interface{}) (*ToolCallResult, error)
}

// BaseServer MCP服务器基础实现
type BaseServer struct {
	tools            map[string]Tool
	resourceHandlers map[string]ResourceHandler
	toolExecutor     ToolExecutor
	mu               sync.RWMutex
	initialized      bool
	clientInfo       *ClientInfo
	capabilities     map[string]interface{}
}

// NewBaseServer 创建新的基础服务器
func NewBaseServer() *BaseServer {
	return &BaseServer{
		tools:            make(map[string]Tool),
		resourceHandlers: make(map[string]ResourceHandler),
		capabilities:     make(map[string]interface{}),
	}
}

// RegisterTool 注册工具
func (s *BaseServer) RegisterTool(tool Tool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	s.tools[tool.Name] = tool
	return nil
}

// RegisterResourceHandler 注册资源处理器
func (s *BaseServer) RegisterResourceHandler(scheme string, handler ResourceHandler) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if scheme == "" {
		return fmt.Errorf("scheme cannot be empty")
	}

	s.resourceHandlers[scheme] = handler
	return nil
}

// GetTool 获取工具
func (s *BaseServer) GetTool(name string) (Tool, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tool, exists := s.tools[name]
	return tool, exists
}

// GetTools 获取所有工具
func (s *BaseServer) GetTools() []Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetResourceHandler 获取资源处理器
func (s *BaseServer) GetResourceHandler(scheme string) (ResourceHandler, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	handler, exists := s.resourceHandlers[scheme]
	return handler, exists
}

// SetCapabilities 设置服务器能力
func (s *BaseServer) SetCapabilities(capabilities map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.capabilities = capabilities
}

// SetToolExecutor 设置工具执行器
func (s *BaseServer) SetToolExecutor(executor ToolExecutor) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.toolExecutor = executor
}

// GetCapabilities 获取服务器能力
func (s *BaseServer) GetCapabilities() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.capabilities
}

// StdioServer 基于标准输入输出的MCP服务器
type StdioServer struct {
	*BaseServer
	reader io.Reader
	writer io.Writer
	ctx    context.Context
	cancel context.CancelFunc
}

// NewStdioServer 创建新的stdio服务器
func NewStdioServer(reader io.Reader, writer io.Writer) *StdioServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &StdioServer{
		BaseServer: NewBaseServer(),
		reader:     reader,
		writer:     writer,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// SetToolExecutor 设置工具执行器
func (s *StdioServer) SetToolExecutor(executor ToolExecutor) {
	s.BaseServer.SetToolExecutor(executor)
}

// Start 启动stdio服务器
func (s *StdioServer) Start() error {
	// 注意：在stdio模式下，日志应该输出到stderr，避免干扰JSON通信

	// 设置默认能力
	s.SetCapabilities(map[string]interface{}{
		"tools": map[string]interface{}{
			"listChanged": true,
		},
		"resources": map[string]interface{}{
			"listChanged": true,
		},
	})

	// 启动消息处理循环
	go s.handleMessages()

	return nil
}

// Stop 停止stdio服务器
func (s *StdioServer) Stop() error {
	// 注意：在stdio模式下，日志应该输出到stderr
	s.cancel()
	return nil
}

// handleMessages 处理消息循环
func (s *StdioServer) handleMessages() {
	decoder := json.NewDecoder(s.reader)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			var msg Message
			if err := decoder.Decode(&msg); err != nil {
				if err == io.EOF {
					log.Println("客户端断开连接")
					return
				}
				log.Printf("解析消息错误: %v", err)
				continue
			}

			// 处理消息
			if err := s.handleMessage(&msg); err != nil {
				log.Printf("处理消息错误: %v", err)
			}
		}
	}
}

// handleMessage 处理单个消息
func (s *StdioServer) handleMessage(msg *Message) error {
	// 验证消息
	if err := msg.Validate(); err != nil {
		return s.sendError(msg.ID, ParseErrorCode, err.Error(), nil)
	}

	// 根据消息类型处理
	switch {
	case msg.IsRequest():
		return s.handleRequest(msg)
	case msg.IsResponse():
		// 服务器通常不处理响应
		return nil
	case msg.IsNotification():
		return s.handleNotification(msg)
	default:
		return s.sendError(msg.ID, InvalidRequestCode, "invalid message", nil)
	}
}

// handleRequest 处理请求
func (s *StdioServer) handleRequest(msg *Message) error {
	switch msg.Method {
	case "initialize":
		return s.handleInitialize(msg)
	case "tools/list":
		return s.handleToolsList(msg)
	case "tools/call":
		return s.handleToolCall(msg)
	case "resources/read":
		return s.handleResourceRead(msg)
	case "shutdown":
		return s.handleShutdown(msg)
	default:
		return s.sendError(msg.ID, MethodNotFoundCode, "method not found: "+msg.Method, nil)
	}
}

// handleInitialize 处理初始化请求
func (s *StdioServer) handleInitialize(msg *Message) error {
	if s.initialized {
		return s.sendError(msg.ID, InvalidRequestCode, "already initialized", nil)
	}

	var params InitializeParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.sendError(msg.ID, InvalidParamsCode, "invalid initialize params", nil)
	}

	// 保存客户端信息
	s.clientInfo = params.ClientInfo
	s.initialized = true

	// 发送初始化响应
	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities:    s.GetCapabilities(),
		ServerInfo: &ServerInfo{
			Name:    "mcp-ai-server",
			Version: "1.0.0",
		},
	}

	return s.sendResponse(msg.ID, result)
}

// handleToolsList 处理工具列表请求
func (s *StdioServer) handleToolsList(msg *Message) error {
	if !s.initialized {
		return s.sendError(msg.ID, InvalidRequestCode, "not initialized", nil)
	}

	tools := s.GetTools()
	return s.sendResponse(msg.ID, map[string]interface{}{
		"tools": tools,
	})
}

// handleToolCall 处理工具调用请求
func (s *StdioServer) handleToolCall(msg *Message) error {
	if !s.initialized {
		return s.sendError(msg.ID, InvalidRequestCode, "not initialized", nil)
	}

	var params ToolCallParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.sendError(msg.ID, InvalidParamsCode, "invalid tool call params", nil)
	}

	// 查找工具
	tool, exists := s.GetTool(params.Name)
	if !exists {
		return s.sendError(msg.ID, MethodNotFoundCode, "tool not found: "+params.Name, nil)
	}

	// 调用实际的工具实现
	if s.toolExecutor != nil {
		result, err := s.toolExecutor.ExecuteTool(context.Background(), params.Name, params.Arguments)
		if err != nil {
			return s.sendError(msg.ID, InternalErrorCode, fmt.Sprintf("工具执行失败: %v", err), nil)
		}
		return s.sendResponse(msg.ID, result)
	}

	// 如果没有工具处理器，返回默认响应
	result := ToolCallResult{
		Content: []Content{
			{
				Type: "text",
				Text: fmt.Sprintf("工具 %s 被调用，参数: %v", tool.Name, params.Arguments),
			},
		},
	}

	return s.sendResponse(msg.ID, result)
}

// handleResourceRead 处理资源读取请求
func (s *StdioServer) handleResourceRead(msg *Message) error {
	if !s.initialized {
		return s.sendError(msg.ID, InvalidRequestCode, "not initialized", nil)
	}

	var params ResourceReadParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.sendError(msg.ID, InvalidParamsCode, "invalid resource read params", nil)
	}

	// 这里应该调用实际的资源处理器
	// 暂时返回一个简单的响应
	result := ResourceReadResult{
		Contents: []Content{
			{
				Type: "text",
				Text: fmt.Sprintf("读取资源: %s", params.URI),
			},
		},
		MimeType: "text/plain",
	}

	return s.sendResponse(msg.ID, result)
}

// handleShutdown 处理关闭请求
func (s *StdioServer) handleShutdown(msg *Message) error {
	// 发送关闭响应
	if err := s.sendResponse(msg.ID, nil); err != nil {
		return err
	}

	// 停止服务器
	go s.Stop()
	return nil
}

// handleNotification 处理通知
func (s *StdioServer) handleNotification(msg *Message) error {
	// 服务器通常不处理通知
	log.Printf("收到通知: %s", msg.Method)
	return nil
}

// sendResponse 发送响应
func (s *StdioServer) sendResponse(id interface{}, result interface{}) error {
	response := NewResponse(id, result)
	return s.sendMessage(response)
}

// sendError 发送错误响应
func (s *StdioServer) sendError(id interface{}, code int, message string, data interface{}) error {
	response := NewErrorResponse(id, code, message, data)
	return s.sendMessage(response)
}

// sendMessage 发送消息
func (s *StdioServer) sendMessage(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	data = append(data, '\n')

	_, err = s.writer.Write(data)
	if err != nil {
		return fmt.Errorf("发送消息失败: %v", err)
	}

	return nil
}
