package tools

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"mcp-ai-server/internal/config"
	"mcp-ai-server/internal/mcp"
)

// DataTools 数据处理工具集合
type DataTools struct {
	securityManager *config.SecurityManager
}

// NewDataTools 创建新的数据处理工具集合
func NewDataTools(securityManager *config.SecurityManager) *DataTools {
	return &DataTools{
		securityManager: securityManager,
	}
}

// JSONParseTool JSON解析工具
func (t *DataTools) JSONParseTool() mcp.Tool {
	return mcp.Tool{
		Name:        "json_parse",
		Description: "解析JSON字符串",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"json_string": map[string]interface{}{
					"type":        "string",
					"description": "要解析的JSON字符串",
				},
				"pretty": map[string]interface{}{
					"type":        "boolean",
					"description": "是否格式化输出",
				},
			},
			"required": []string{"json_string"},
		},
	}
}

// JSONValidateTool JSON验证工具
func (t *DataTools) JSONValidateTool() mcp.Tool {
	return mcp.Tool{
		Name:        "json_validate",
		Description: "验证JSON字符串格式",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"json_string": map[string]interface{}{
					"type":        "string",
					"description": "要验证的JSON字符串",
				},
			},
			"required": []string{"json_string"},
		},
	}
}

// Base64EncodeTool Base64编码工具
func (t *DataTools) Base64EncodeTool() mcp.Tool {
	return mcp.Tool{
		Name:        "base64_encode",
		Description: "Base64编码",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "string",
					"description": "要编码的文本",
				},
			},
			"required": []string{"text"},
		},
	}
}

// Base64DecodeTool Base64解码工具
func (t *DataTools) Base64DecodeTool() mcp.Tool {
	return mcp.Tool{
		Name:        "base64_decode",
		Description: "Base64解码",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "string",
					"description": "要解码的Base64文本",
				},
			},
			"required": []string{"text"},
		},
	}
}

// HashTool 哈希计算工具
func (t *DataTools) HashTool() mcp.Tool {
	return mcp.Tool{
		Name:        "hash",
		Description: "计算文本的哈希值",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "string",
					"description": "要计算哈希的文本",
				},
				"algorithm": map[string]interface{}{
					"type":        "string",
					"description": "哈希算法 (md5, sha1, sha256)",
				},
			},
			"required": []string{"text"},
		},
	}
}

// TextTransformTool 文本转换工具
func (t *DataTools) TextTransformTool() mcp.Tool {
	return mcp.Tool{
		Name:        "text_transform",
		Description: "文本格式转换",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "string",
					"description": "要转换的文本",
				},
				"operation": map[string]interface{}{
					"type":        "string",
					"description": "转换操作 (uppercase, lowercase, title, reverse, trim)",
				},
			},
			"required": []string{"text", "operation"},
		},
	}
}

// GetTools 获取所有数据处理工具
func (t *DataTools) GetTools() []mcp.Tool {
	return []mcp.Tool{
		t.JSONParseTool(),
		t.JSONValidateTool(),
		t.Base64EncodeTool(),
		t.Base64DecodeTool(),
		t.HashTool(),
		t.TextTransformTool(),
	}
}

// ExecuteTool 执行数据处理工具
func (t *DataTools) ExecuteTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	switch name {
	case "json_parse":
		return t.executeJSONParse(ctx, arguments)
	case "json_validate":
		return t.executeJSONValidate(ctx, arguments)
	case "base64_encode":
		return t.executeBase64Encode(ctx, arguments)
	case "base64_decode":
		return t.executeBase64Decode(ctx, arguments)
	case "hash":
		return t.executeHash(ctx, arguments)
	case "text_transform":
		return t.executeTextTransform(ctx, arguments)
	default:
		return nil, fmt.Errorf("未知的数据处理工具: %s", name)
	}
}

// executeJSONParse 执行JSON解析
func (t *DataTools) executeJSONParse(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	jsonString, ok := arguments["json_string"].(string)
	if !ok {
		return nil, fmt.Errorf("json_string参数必须是字符串")
	}

	pretty := false
	if prettyVal, ok := arguments["pretty"].(bool); ok {
		pretty = prettyVal
	}

	// 解析JSON
	var data interface{}
	if err := json.Unmarshal([]byte(jsonString), &data); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %v", err)
	}

	// 格式化输出
	var output []byte
	var err error
	if pretty {
		output, err = json.MarshalIndent(data, "", "  ")
	} else {
		output, err = json.Marshal(data)
	}

	if err != nil {
		return nil, fmt.Errorf("JSON格式化失败: %v", err)
	}

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: string(output),
			},
		},
	}, nil
}

// executeJSONValidate 执行JSON验证
func (t *DataTools) executeJSONValidate(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	jsonString, ok := arguments["json_string"].(string)
	if !ok {
		return nil, fmt.Errorf("json_string参数必须是字符串")
	}

	// 验证JSON
	var data interface{}
	if err := json.Unmarshal([]byte(jsonString), &data); err != nil {
		return &mcp.ToolCallResult{
			Content: []mcp.Content{
				{
					Type: "text",
					Text: fmt.Sprintf("JSON验证失败: %v", err),
				},
			},
		}, nil
	}

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: "JSON格式有效",
			},
		},
	}, nil
}

// executeBase64Encode 执行Base64编码
func (t *DataTools) executeBase64Encode(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	text, ok := arguments["text"].(string)
	if !ok {
		return nil, fmt.Errorf("text参数必须是字符串")
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(text))

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: encoded,
			},
		},
	}, nil
}

// executeBase64Decode 执行Base64解码
func (t *DataTools) executeBase64Decode(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	text, ok := arguments["text"].(string)
	if !ok {
		return nil, fmt.Errorf("text参数必须是字符串")
	}

	decoded, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return nil, fmt.Errorf("Base64解码失败: %v", err)
	}

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: string(decoded),
			},
		},
	}, nil
}

// executeHash 执行哈希计算
func (t *DataTools) executeHash(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	text, ok := arguments["text"].(string)
	if !ok {
		return nil, fmt.Errorf("text参数必须是字符串")
	}

	algorithm := "md5"
	if algo, ok := arguments["algorithm"].(string); ok && algo != "" {
		algorithm = strings.ToLower(algo)
	}

	var hash string
	switch algorithm {
	case "md5":
		hashBytes := md5.Sum([]byte(text))
		hash = hex.EncodeToString(hashBytes[:])
	case "sha1":
		hashBytes := sha1.Sum([]byte(text))
		hash = hex.EncodeToString(hashBytes[:])
	case "sha256":
		hashBytes := sha256.Sum256([]byte(text))
		hash = hex.EncodeToString(hashBytes[:])
	default:
		return nil, fmt.Errorf("不支持的哈希算法: %s", algorithm)
	}

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: fmt.Sprintf("%s: %s", algorithm, hash),
			},
		},
	}, nil
}

// executeTextTransform 执行文本转换
func (t *DataTools) executeTextTransform(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	text, ok := arguments["text"].(string)
	if !ok {
		return nil, fmt.Errorf("text参数必须是字符串")
	}

	operation, ok := arguments["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation参数必须是字符串")
	}

	var result string
	switch strings.ToLower(operation) {
	case "uppercase":
		result = strings.ToUpper(text)
	case "lowercase":
		result = strings.ToLower(text)
	case "title":
		result = strings.Title(strings.ToLower(text))
	case "reverse":
		runes := []rune(text)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		result = string(runes)
	case "trim":
		result = strings.TrimSpace(text)
	default:
		return nil, fmt.Errorf("不支持的转换操作: %s", operation)
	}

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}
