package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"mcp-ai-server/internal/config"
	"mcp-ai-server/internal/mcp"
)

// SystemTools 系统工具集合
type SystemTools struct {
	securityManager *config.SecurityManager
}

// NewSystemTools 创建新的系统工具集合
func NewSystemTools(configPath string) (*SystemTools, error) {
	securityManager, err := config.NewSecurityManager(configPath)
	if err != nil {
		return nil, fmt.Errorf("创建安全管理器失败: %v", err)
	}

	return &SystemTools{
		securityManager: securityManager,
	}, nil
}

// FileReadTool 文件读取工具
func (t *SystemTools) FileReadTool() mcp.Tool {
	return mcp.Tool{
		Name:        "file_read",
		Description: "读取文件内容",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件路径",
				},
			},
			"required": []string{"path"},
		},
	}
}

// FileWriteTool 文件写入工具
func (t *SystemTools) FileWriteTool() mcp.Tool {
	return mcp.Tool{
		Name:        "file_write",
		Description: "写入文件内容",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件路径",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "文件内容",
				},
			},
			"required": []string{"path", "content"},
		},
	}
}

// CommandExecuteTool 命令执行工具
func (t *SystemTools) CommandExecuteTool() mcp.Tool {
	return mcp.Tool{
		Name:        "command_execute",
		Description: "执行系统命令",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "要执行的命令",
				},
				"args": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "命令参数",
				},
				"working_dir": map[string]interface{}{
					"type":        "string",
					"description": "工作目录",
				},
			},
			"required": []string{"command"},
		},
	}
}

// DirectoryListTool 目录列表工具
func (t *SystemTools) DirectoryListTool() mcp.Tool {
	return mcp.Tool{
		Name:        "directory_list",
		Description: "列出目录内容",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "目录路径",
				},
			},
			"required": []string{"path"},
		},
	}
}

// GetTools 获取所有系统工具
func (t *SystemTools) GetTools() []mcp.Tool {
	return []mcp.Tool{
		t.FileReadTool(),
		t.FileWriteTool(),
		t.CommandExecuteTool(),
		t.DirectoryListTool(),
	}
}

// ExecuteTool 执行工具
func (t *SystemTools) ExecuteTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	switch name {
	case "file_read":
		return t.executeFileRead(ctx, arguments)
	case "file_write":
		return t.executeFileWrite(ctx, arguments)
	case "command_execute":
		return t.executeCommand(ctx, arguments)
	case "directory_list":
		return t.executeDirectoryList(ctx, arguments)
	default:
		return nil, fmt.Errorf("未知工具: %s", name)
	}
}

// executeFileRead 执行文件读取
func (t *SystemTools) executeFileRead(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	path, ok := arguments["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path参数必须是字符串")
	}

	// 使用安全管理器检查路径
	if err := t.securityManager.IsPathAllowed(path); err != nil {
		return nil, fmt.Errorf("安全检查失败: %v", err)
	}

	// 获取文件信息
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	// 检查文件大小
	if err := t.securityManager.CheckFileSize(fileInfo.Size()); err != nil {
		return nil, fmt.Errorf("文件大小检查失败: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: string(content),
			},
		},
	}, nil
}

// executeFileWrite 执行文件写入
func (t *SystemTools) executeFileWrite(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	path, ok := arguments["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path参数必须是字符串")
	}

	content, ok := arguments["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content参数必须是字符串")
	}

	// 使用安全管理器检查路径
	if err := t.securityManager.IsPathAllowed(path); err != nil {
		return nil, fmt.Errorf("安全检查失败: %v", err)
	}

	// 检查内容大小
	if err := t.securityManager.CheckFileSize(int64(len(content))); err != nil {
		return nil, fmt.Errorf("内容大小检查失败: %v", err)
	}

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("写入文件失败: %v", err)
	}

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: fmt.Sprintf("文件 %s 写入成功", path),
			},
		},
	}, nil
}

// executeCommand 执行命令
func (t *SystemTools) executeCommand(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	command, ok := arguments["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command参数必须是字符串")
	}

	var args []string
	if argsInterface, ok := arguments["args"]; ok {
		if argsSlice, ok := argsInterface.([]interface{}); ok {
			for _, arg := range argsSlice {
				if argStr, ok := arg.(string); ok {
					args = append(args, argStr)
				}
			}
		}
	}

	workingDir := ""
	if wd, ok := arguments["working_dir"].(string); ok {
		workingDir = wd
	}

	// 使用安全管理器检查命令
	if err := t.securityManager.IsCommandAllowed(command); err != nil {
		return nil, fmt.Errorf("命令安全检查失败: %v", err)
	}

	cmd := exec.CommandContext(ctx, command, args...)
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &mcp.ToolCallResult{
			Content: []mcp.Content{
				{
					Type: "text",
					Text: fmt.Sprintf("命令执行失败: %v\n输出: %s", err, string(output)),
				},
			},
		}, nil
	}

	// 检查输出大小
	if err := t.securityManager.CheckCommandOutput(int64(len(output))); err != nil {
		return nil, fmt.Errorf("命令输出大小检查失败: %v", err)
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

// executeDirectoryList 执行目录列表
func (t *SystemTools) executeDirectoryList(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	path, ok := arguments["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path参数必须是字符串")
	}

	// 使用安全管理器检查路径
	if err := t.securityManager.IsPathAllowed(path); err != nil {
		return nil, fmt.Errorf("安全检查失败: %v", err)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %v", err)
	}

	// 检查目录项数
	if err := t.securityManager.CheckDirectoryItems(len(entries)); err != nil {
		return nil, fmt.Errorf("目录项数检查失败: %v", err)
	}

	var items []string
	for _, entry := range entries {
		info := entry.Name()
		if entry.IsDir() {
			info += "/"
		}
		items = append(items, info)
	}

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: strings.Join(items, "\n"),
			},
		},
	}, nil
}
