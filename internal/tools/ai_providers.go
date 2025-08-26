package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"mcp-ai-server/internal/config"
)

// AIProvider AI提供商接口
type AIProvider interface {
	Name() string
	IsEnabled() bool
	Call(ctx context.Context, model, prompt string, options map[string]interface{}) (string, error)
}

// BaseProvider 基础提供商
type BaseProvider struct {
	config *config.ProviderConfig
	client *http.Client
}

// NewBaseProvider 创建基础提供商
func NewBaseProvider(config *config.ProviderConfig) *BaseProvider {
	return &BaseProvider{
		config: config,
		client: &http.Client{
			Timeout: time.Duration(120) * time.Second,
		},
	}
}

// IsEnabled 检查是否启用
func (p *BaseProvider) IsEnabled() bool {
	return p.config.Enabled
}

// OllamaProvider Ollama本地服务提供商
type OllamaProvider struct {
	*BaseProvider
}

// NewOllamaProvider 创建Ollama提供商
func NewOllamaProvider(config *config.ProviderConfig) *OllamaProvider {
	return &OllamaProvider{
		BaseProvider: NewBaseProvider(config),
	}
}

// Name 提供商名称
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// Call 调用Ollama API
func (p *OllamaProvider) Call(ctx context.Context, model, prompt string, options map[string]interface{}) (string, error) {
	// Ollama API请求结构
	request := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"num_predict": getIntOption(options, "max_tokens", 1000),
			"temperature": getFloatOption(options, "temperature", 0.7),
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/api/generate", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama API返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var ollamaResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResponse); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	// 提取生成的文本
	response, ok := ollamaResponse["response"].(string)
	if !ok {
		return "", fmt.Errorf("响应中没有找到response字段")
	}

	return response, nil
}

// OpenAIProvider OpenAI云服务提供商
type OpenAIProvider struct {
	*BaseProvider
}

// NewOpenAIProvider 创建OpenAI提供商
func NewOpenAIProvider(config *config.ProviderConfig) *OpenAIProvider {
	return &OpenAIProvider{
		BaseProvider: NewBaseProvider(config),
	}
}

// Name 提供商名称
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// Call 调用OpenAI API
func (p *OpenAIProvider) Call(ctx context.Context, model, prompt string, options map[string]interface{}) (string, error) {
	// OpenAI API请求结构
	request := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens":  getIntOption(options, "max_tokens", 1000),
		"temperature": getFloatOption(options, "temperature", 0.7),
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/chat/completions", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	// 发送请求
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var openaiResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&openaiResponse); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	// 提取生成的文本
	choices, ok := openaiResponse["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("响应中没有找到choices字段")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("choice格式错误")
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("message格式错误")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("content格式错误")
	}

	return content, nil
}

// AnthropicProvider Anthropic Claude云服务提供商
type AnthropicProvider struct {
	*BaseProvider
}

// NewAnthropicProvider 创建Anthropic提供商
func NewAnthropicProvider(config *config.ProviderConfig) *AnthropicProvider {
	return &AnthropicProvider{
		BaseProvider: NewBaseProvider(config),
	}
}

// Name 提供商名称
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// Call 调用Anthropic API
func (p *AnthropicProvider) Call(ctx context.Context, model, prompt string, options map[string]interface{}) (string, error) {
	// Anthropic API请求结构
	request := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens":  getIntOption(options, "max_tokens", 1000),
		"temperature": getFloatOption(options, "temperature", 0.7),
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/v1/messages", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// 发送请求
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Anthropic API返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var anthropicResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResponse); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	// 提取生成的文本
	content, ok := anthropicResponse["content"].([]interface{})
	if !ok || len(content) == 0 {
		return "", fmt.Errorf("响应中没有找到content字段")
	}

	contentItem, ok := content[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("content item格式错误")
	}

	text, ok := contentItem["text"].(string)
	if !ok {
		return "", fmt.Errorf("text格式错误")
	}

	return text, nil
}

// 辅助函数
func getIntOption(options map[string]interface{}, key string, defaultValue int) int {
	if value, ok := options[key]; ok {
		if intValue, ok := value.(float64); ok {
			return int(intValue)
		}
	}
	return defaultValue
}

func getFloatOption(options map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := options[key]; ok {
		if floatValue, ok := value.(float64); ok {
			return floatValue
		}
	}
	return defaultValue
}
