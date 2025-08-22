package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketServer WebSocket MCP服务器
type WebSocketServer struct {
	*BaseServer
	port     int
	upgrader websocket.Upgrader
	server   *http.Server
	conns    map[*websocket.Conn]bool
	connMu   sync.RWMutex
}

// NewWebSocketServer 创建新的WebSocket服务器
func NewWebSocketServer(port int) *WebSocketServer {
	return &WebSocketServer{
		BaseServer: NewBaseServer(),
		port:       port,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// 允许所有来源，生产环境中应该限制
				return true
			},
		},
		conns: make(map[*websocket.Conn]bool),
	}
}

// Start 启动WebSocket服务器
func (s *WebSocketServer) Start() error {
	// 创建自定义HTTP处理器
	mux := http.NewServeMux()

	// 设置WebSocket处理器
	mux.HandleFunc("/", s.handleWebSocket)

	// 设置健康检查端点
	mux.HandleFunc("/health", s.handleHealth)

	// 创建HTTP服务器
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	log.Printf("WebSocket MCP服务器启动在端口 %d", s.port)
	log.Printf("WebSocket地址: ws://localhost:%d", s.port)
	log.Printf("健康检查: http://localhost:%d/health", s.port)

	// 启动HTTP服务器
	return s.server.ListenAndServe()
}

// Stop 停止WebSocket服务器
func (s *WebSocketServer) Stop() error {
	log.Println("正在停止WebSocket MCP服务器...")

	// 关闭所有WebSocket连接
	s.connMu.Lock()
	for conn := range s.conns {
		conn.Close()
		delete(s.conns, conn)
	}
	s.connMu.Unlock()

	// 关闭HTTP服务器
	if s.server != nil {
		return s.server.Shutdown(context.Background())
	}

	return nil
}

// handleHealth 处理健康检查请求
func (s *WebSocketServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "mcp-ai-websocket",
		"timestamp": time.Now().Format(time.RFC3339),
		"connections": func() int {
			s.connMu.RLock()
			defer s.connMu.RUnlock()
			return len(s.conns)
		}(),
	}

	json.NewEncoder(w).Encode(response)
}

// handleWebSocket 处理WebSocket连接
func (s *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 检查是否是WebSocket升级请求
	if !websocket.IsWebSocketUpgrade(r) {
		// 如果不是WebSocket请求，返回错误信息
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		response := map[string]interface{}{
			"error":   "此端点仅支持WebSocket连接",
			"usage":   "请使用WebSocket客户端连接此端点",
			"example": "ws://localhost:8081/",
		}

		json.NewEncoder(w).Encode(response)
		return
	}

	// 升级HTTP连接为WebSocket连接
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}

	// 注册连接
	s.connMu.Lock()
	s.conns[conn] = true
	s.connMu.Unlock()

	log.Printf("新的WebSocket连接: %s", conn.RemoteAddr())

	// 启动消息处理协程
	go s.handleConnection(conn)
}

// handleConnection 处理单个WebSocket连接
func (s *WebSocketServer) handleConnection(conn *websocket.Conn) {
	defer func() {
		// 清理连接
		s.connMu.Lock()
		delete(s.conns, conn)
		s.connMu.Unlock()
		conn.Close()
		log.Printf("WebSocket连接已关闭: %s", conn.RemoteAddr())
	}()

	// 消息循环
	for {
		// 读取消息
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket读取错误: %v", err)
			}
			break
		}

		// 处理消息
		response, err := s.handleMessage(message)
		if err != nil {
			log.Printf("处理消息失败: %v", err)
			// 发送错误响应
			errorResponse := Message{
				JSONRPC: "2.0",
				ID:      nil,
				Error: &Error{
					Code:    -32603,
					Message: "Internal error",
					Data:    err.Error(),
				},
			}
			if err := conn.WriteJSON(errorResponse); err != nil {
				log.Printf("发送错误响应失败: %v", err)
			}
			continue
		}

		// 发送响应
		if response != nil {
			if err := conn.WriteJSON(response); err != nil {
				log.Printf("发送响应失败: %v", err)
				break
			}
		}
	}
}

// handleMessage 处理单个消息
func (s *WebSocketServer) handleMessage(message []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(message, &msg); err != nil {
		return nil, fmt.Errorf("解析消息失败: %v", err)
	}

	// 根据方法类型处理
	switch msg.Method {
	case "initialize":
		return s.handleInitialize(&msg)
	case "tools/list":
		return s.handleToolsList(&msg)
	case "tools/call":
		return s.handleToolCall(&msg)
	case "resources/read":
		return s.handleResourceRead(&msg)
	case "shutdown":
		return s.handleShutdown(&msg)
	default:
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &Error{
				Code:    -32601,
				Message: "Method not found",
				Data:    fmt.Sprintf("Unknown method: %s", msg.Method),
			},
		}, nil
	}
}

// handleInitialize 处理初始化请求
func (s *WebSocketServer) handleInitialize(msg *Message) (*Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &Error{
				Code:    -32000,
				Message: "Already initialized",
			},
		}, nil
	}

	// 解析客户端信息
	var params map[string]interface{}
	if err := json.Unmarshal(msg.Params, &params); err == nil {
		if clientInfo, ok := params["clientInfo"].(map[string]interface{}); ok {
			if name, ok := clientInfo["name"].(string); ok {
				s.clientInfo = &ClientInfo{Name: name}
			}
		}
	}

	s.initialized = true

	return &Message{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "mcp-ai-websocket",
				"version": "1.0.0",
			},
		},
	}, nil
}

// handleToolsList 处理工具列表请求
func (s *WebSocketServer) handleToolsList(msg *Message) (*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &Error{
				Code:    -32002,
				Message: "Not initialized",
			},
		}, nil
	}

	tools := make([]Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool)
	}

	return &Message{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}, nil
}

// handleToolCall 处理工具调用请求
func (s *WebSocketServer) handleToolCall(msg *Message) (*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &Error{
				Code:    -32002,
				Message: "Not initialized",
			},
		}, nil
	}

	if s.toolExecutor == nil {
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &Error{
				Code:    -32603,
				Message: "Tool executor not available",
			},
		}, nil
	}

	// 解析工具调用参数
	var params map[string]interface{}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &Error{
				Code:    -32602,
				Message: "Invalid params",
			},
		}, nil
	}

	toolName, ok := params["name"].(string)
	if !ok {
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &Error{
				Code:    -32602,
				Message: "Tool name is required",
			},
		}, nil
	}

	arguments, _ := params["arguments"].(map[string]interface{})

	// 执行工具
	ctx := context.Background()
	result, err := s.toolExecutor.ExecuteTool(ctx, toolName, arguments)
	if err != nil {
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &Error{
				Code:    -32603,
				Message: "Tool execution failed",
				Data:    err.Error(),
			},
		}, nil
	}

	return &Message{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result:  result,
	}, nil
}

// handleResourceRead 处理资源读取请求
func (s *WebSocketServer) handleResourceRead(msg *Message) (*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &Error{
				Code:    -32002,
				Message: "Not initialized",
			},
		}, nil
	}

	// 这里可以实现资源读取逻辑
	return &Message{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Error: &Error{
			Code:    -32601,
			Message: "Resource reading not implemented",
		},
	}, nil
}

// handleShutdown 处理关闭请求
func (s *WebSocketServer) handleShutdown(msg *Message) (*Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initialized = false

	return &Message{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result:  nil,
	}, nil
}

// SetToolExecutor 设置工具执行器
func (s *WebSocketServer) SetToolExecutor(executor ToolExecutor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.toolExecutor = executor
}
