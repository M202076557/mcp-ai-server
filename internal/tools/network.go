package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"mcp-ai-server/internal/config"
	"mcp-ai-server/internal/mcp"
)

// NetworkTools 网络工具集合
type NetworkTools struct {
	securityManager *config.SecurityManager
	httpClient      *http.Client
}

// NewNetworkTools 创建新的网络工具集合
func NewNetworkTools(securityManager *config.SecurityManager) *NetworkTools {
	return &NetworkTools{
		securityManager: securityManager,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// HTTPGetTool HTTP GET请求工具
func (t *NetworkTools) HTTPGetTool() mcp.Tool {
	return mcp.Tool{
		Name:        "http_get",
		Description: "发送HTTP GET请求",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "请求的URL",
				},
				"headers": map[string]interface{}{
					"type":        "object",
					"description": "请求头",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "超时时间（秒）",
				},
			},
			"required": []string{"url"},
		},
	}
}

// HTTPPostTool HTTP POST请求工具
func (t *NetworkTools) HTTPPostTool() mcp.Tool {
	return mcp.Tool{
		Name:        "http_post",
		Description: "发送HTTP POST请求",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "请求的URL",
				},
				"data": map[string]interface{}{
					"type":        "string",
					"description": "POST数据",
				},
				"headers": map[string]interface{}{
					"type":        "object",
					"description": "请求头",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "超时时间（秒）",
				},
			},
			"required": []string{"url"},
		},
	}
}

// PingTool 网络连通性检查工具
func (t *NetworkTools) PingTool() mcp.Tool {
	return mcp.Tool{
		Name:        "ping",
		Description: "检查网络连通性",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"host": map[string]interface{}{
					"type":        "string",
					"description": "要ping的主机",
				},
				"count": map[string]interface{}{
					"type":        "integer",
					"description": "ping次数",
				},
			},
			"required": []string{"host"},
		},
	}
}

// DNSLookupTool DNS查询工具
func (t *NetworkTools) DNSLookupTool() mcp.Tool {
	return mcp.Tool{
		Name:        "dns_lookup",
		Description: "DNS域名解析",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"domain": map[string]interface{}{
					"type":        "string",
					"description": "要查询的域名",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "记录类型（A, AAAA, CNAME, MX等）",
				},
			},
			"required": []string{"domain"},
		},
	}
}

// GetTools 获取所有网络工具
func (t *NetworkTools) GetTools() []mcp.Tool {
	return []mcp.Tool{
		t.HTTPGetTool(),
		t.HTTPPostTool(),
		t.PingTool(),
		t.DNSLookupTool(),
	}
}

// ExecuteTool 执行网络工具
func (t *NetworkTools) ExecuteTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	switch name {
	case "http_get":
		return t.executeHTTPGet(ctx, arguments)
	case "http_post":
		return t.executeHTTPPost(ctx, arguments)
	case "ping":
		return t.executePing(ctx, arguments)
	case "dns_lookup":
		return t.executeDNSLookup(ctx, arguments)
	default:
		return nil, fmt.Errorf("未知的网络工具: %s", name)
	}
}

// executeHTTPGet 执行HTTP GET请求
func (t *NetworkTools) executeHTTPGet(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	urlStr, ok := arguments["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url参数必须是字符串")
	}

	// 安全检查：只允许HTTP和HTTPS
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return nil, fmt.Errorf("只允许HTTP和HTTPS协议")
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 添加请求头
	if headers, ok := arguments["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	// 设置超时
	if timeout, ok := arguments["timeout"].(int); ok && timeout > 0 {
		t.httpClient.Timeout = time.Duration(timeout) * time.Second
	} else {
		// 为GET请求设置合理的默认超时时间
		t.httpClient.Timeout = 60 * time.Second
	}

	// 发送请求
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查响应大小
	if err := t.securityManager.CheckCommandOutput(int64(len(body))); err != nil {
		return nil, fmt.Errorf("响应大小检查失败: %v", err)
	}

	// 构建响应信息
	responseInfo := map[string]interface{}{
		"status_code": resp.StatusCode,
		"status":      resp.Status,
		"headers":     resp.Header,
		"body":        string(body),
		"url":         urlStr,
	}

	responseJSON, _ := json.MarshalIndent(responseInfo, "", "  ")

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: string(responseJSON),
			},
		},
	}, nil
}

// executeHTTPPost 执行HTTP POST请求
func (t *NetworkTools) executeHTTPPost(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	urlStr, ok := arguments["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url参数必须是字符串")
	}

	// 安全检查：只允许HTTP和HTTPS
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return nil, fmt.Errorf("只允许HTTP和HTTPS协议")
	}

	data := ""
	if dataStr, ok := arguments["data"].(string); ok {
		data = dataStr
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, strings.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, t.httpClient.Timeout)
	defer cancel()
	req = req.WithContext(timeoutCtx)

	// 添加请求头
	if headers, ok := arguments["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	// 设置默认Content-Type
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// 设置超时
	if timeout, ok := arguments["timeout"].(int); ok && timeout > 0 {
		t.httpClient.Timeout = time.Duration(timeout) * time.Second
	} else {
		// 为POST请求设置更长的默认超时时间（5分钟）
		t.httpClient.Timeout = 300 * time.Second
	}

	// 记录请求开始时间和超时设置
	startTime := time.Now()
	timeoutDuration := t.httpClient.Timeout
	fmt.Printf("🚀 开始POST请求: %s\n", urlStr)
	fmt.Printf("⏰ 超时设置: %v\n", timeoutDuration)
	fmt.Printf("📊 请求数据大小: %d 字节\n", len(data))

	// 发送请求（带重试机制）
	var resp *http.Response
	var requestErr error
	maxRetries := 1

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			fmt.Printf("🔄 重试请求 (第%d次尝试)\n", attempt)
		}

		resp, requestErr = t.httpClient.Do(req)
		requestDuration := time.Since(startTime)

		if requestErr != nil {
			if strings.Contains(requestErr.Error(), "timeout") && attempt < maxRetries {
				fmt.Printf("⏰ 请求超时，准备重试 (耗时: %v)\n", requestDuration)
				continue
			}
			if strings.Contains(requestErr.Error(), "timeout") {
				return nil, fmt.Errorf("POST请求超时 (耗时: %v, 超时时间: %v, 已重试%d次): %v", requestDuration, timeoutDuration, attempt, requestErr)
			}
			return nil, fmt.Errorf("请求失败 (耗时: %v): %v", requestDuration, requestErr)
		}

		fmt.Printf("✅ 请求完成，耗时: %v\n", requestDuration)
		break
	}

	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查响应大小
	if err := t.securityManager.CheckCommandOutput(int64(len(body))); err != nil {
		return nil, fmt.Errorf("响应大小检查失败: %v", err)
	}

	// 构建响应信息
	responseInfo := map[string]interface{}{
		"status_code": resp.StatusCode,
		"status":      resp.Status,
		"headers":     resp.Header,
		"body":        string(body),
		"url":         urlStr,
		"data":        data,
	}

	responseJSON, _ := json.MarshalIndent(responseInfo, "", "  ")

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: string(responseJSON),
			},
		},
	}, nil
}

// executePing 执行ping命令
func (t *NetworkTools) executePing(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	host, ok := arguments["host"].(string)
	if !ok {
		return nil, fmt.Errorf("host参数必须是字符串")
	}

	count := 4
	if countVal, ok := arguments["count"].(int); ok && countVal > 0 {
		count = countVal
	}

	// 使用系统ping命令
	cmd := exec.CommandContext(ctx, "ping", "-c", fmt.Sprintf("%d", count), host)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return &mcp.ToolCallResult{
			Content: []mcp.Content{
				{
					Type: "text",
					Text: fmt.Sprintf("Ping失败: %v\n输出: %s", err, string(output)),
				},
			},
		}, nil
	}

	// 检查输出大小
	if err := t.securityManager.CheckCommandOutput(int64(len(output))); err != nil {
		return nil, fmt.Errorf("输出大小检查失败: %v", err)
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

// executeDNSLookup 执行DNS查询
func (t *NetworkTools) executeDNSLookup(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	domain, ok := arguments["domain"].(string)
	if !ok {
		return nil, fmt.Errorf("domain参数必须是字符串")
	}

	recordType := "A"
	if typeVal, ok := arguments["type"].(string); ok && typeVal != "" {
		recordType = strings.ToUpper(typeVal)
	}

	// 使用系统nslookup命令
	cmd := exec.CommandContext(ctx, "nslookup", "-type="+recordType, domain)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return &mcp.ToolCallResult{
			Content: []mcp.Content{
				{
					Type: "text",
					Text: fmt.Sprintf("DNS查询失败: %v\n输出: %s", err, string(output)),
				},
			},
		}, nil
	}

	// 检查输出大小
	if err := t.securityManager.CheckCommandOutput(int64(len(output))); err != nil {
		return nil, fmt.Errorf("输出大小检查失败: %v", err)
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
// GetTools and ExecuteTool methods
