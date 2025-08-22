package mcp

import (
	"encoding/json"
	"fmt"
)

// JSONRPC 2.0 基础消息结构
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// 错误结构
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 初始化请求参数
type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      *ClientInfo            `json:"clientInfo,omitempty"`
}

// 客户端信息
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// 初始化响应结果
type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      *ServerInfo            `json:"serverInfo,omitempty"`
}

// 服务器信息
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// 工具定义
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// 工具调用参数
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// 工具调用结果
type ToolCallResult struct {
	Content []Content `json:"content"`
}

// 内容结构
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// 资源读取参数
type ResourceReadParams struct {
	URI string `json:"uri"`
}

// 资源读取结果
type ResourceReadResult struct {
	Contents []Content `json:"contents"`
	MimeType string    `json:"mimeType"`
}

// 常量定义
const (
	JSONRPCVersion  = "2.0"
	ProtocolVersion = "2024-11-05"
)

// 错误代码
const (
	ParseErrorCode     = -32700
	InvalidRequestCode = -32600
	MethodNotFoundCode = -32601
	InvalidParamsCode  = -32602
	InternalErrorCode  = -32603
	ServerErrorCode    = -32000
)

// 创建请求消息
func NewRequest(id interface{}, method string, params interface{}) *Message {
	paramsBytes, _ := json.Marshal(params)
	return &Message{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  method,
		Params:  paramsBytes,
	}
}

// 创建响应消息
func NewResponse(id interface{}, result interface{}) *Message {
	return &Message{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  result,
	}
}

// 创建错误响应
func NewErrorResponse(id interface{}, code int, message string, data interface{}) *Message {
	return &Message{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// 创建通知消息
func NewNotification(method string, params interface{}) *Message {
	paramsBytes, _ := json.Marshal(params)
	return &Message{
		JSONRPC: JSONRPCVersion,
		Method:  method,
		Params:  paramsBytes,
	}
}

// 验证消息
func (m *Message) Validate() error {
	if m.JSONRPC != JSONRPCVersion {
		return fmt.Errorf("invalid JSON-RPC version: %s", m.JSONRPC)
	}

	if m.Method == "" && m.Result == nil && m.Error == nil {
		return fmt.Errorf("message must have method, result, or error")
	}

	return nil
}

// 检查是否为请求
func (m *Message) IsRequest() bool {
	return m.Method != "" && m.ID != nil
}

// 检查是否为响应
func (m *Message) IsResponse() bool {
	return (m.Result != nil || m.Error != nil) && m.ID != nil
}

// 检查是否为通知
func (m *Message) IsNotification() bool {
	return m.Method != "" && m.ID == nil
}
