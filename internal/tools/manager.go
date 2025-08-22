package tools

import (
	"context"
	"fmt"

	"mcp-ai-server/internal/config"
	"mcp-ai-server/internal/mcp"
)

// ToolExecutor 工具执行器接口
type ToolExecutor interface {
	ExecuteTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.ToolCallResult, error)
	GetTools() []mcp.Tool
}

// ToolManager 工具管理器
type ToolManager struct {
	toolMap         map[string]ToolExecutor // 工具名到执行器的映射
	securityManager *config.SecurityManager
	systemTools     *SystemTools
	networkTools    *NetworkTools
	dataTools       *DataTools
	databaseTools   *DatabaseTools
	aiTools         *AITools
}

// NewToolManager 创建新的工具管理器
func NewToolManager(configPath string) (*ToolManager, error) {
	// 创建安全管理器（共享给所有工具）
	securityManager, err := config.NewSecurityManager(configPath)
	if err != nil {
		return nil, fmt.Errorf("创建安全管理器失败: %v", err)
	}

	// 创建各种工具集合，共享安全管理器
	systemTools, err := NewSystemTools(configPath)
	if err != nil {
		return nil, fmt.Errorf("创建系统工具失败: %v", err)
	}

	networkTools := NewNetworkTools(securityManager)
	dataTools := NewDataTools(securityManager)
	databaseTools := NewDatabaseTools(securityManager)

	// 创建AI工具，传递配置文件路径和数据库工具
	aiTools, err := NewAITools(configPath, databaseTools)
	if err != nil {
		return nil, fmt.Errorf("创建AI工具失败: %v", err)
	}

	// 创建工具映射表，提高查找效率
	toolMap := make(map[string]ToolExecutor)

	// 注册系统工具
	for _, tool := range systemTools.GetTools() {
		toolMap[tool.Name] = systemTools
	}

	// 注册网络工具
	for _, tool := range networkTools.GetTools() {
		toolMap[tool.Name] = networkTools
	}

	// 注册数据处理工具
	for _, tool := range dataTools.GetTools() {
		toolMap[tool.Name] = dataTools
	}

	// 注册数据库工具
	for _, tool := range databaseTools.GetTools() {
		toolMap[tool.Name] = databaseTools
	}

	// 注册AI工具
	for _, tool := range aiTools.GetTools() {
		toolMap[tool.Name] = aiTools
	}

	return &ToolManager{
		toolMap:         toolMap,
		securityManager: securityManager,
		systemTools:     systemTools,
		networkTools:    networkTools,
		dataTools:       dataTools,
		databaseTools:   databaseTools,
		aiTools:         aiTools,
	}, nil
}

// GetTools 获取所有工具
func (tm *ToolManager) GetTools() []mcp.Tool {
	var tools []mcp.Tool

	// 添加系统工具
	tools = append(tools, tm.systemTools.GetTools()...)

	// 添加网络工具
	tools = append(tools, tm.networkTools.GetTools()...)

	// 添加数据处理工具
	tools = append(tools, tm.dataTools.GetTools()...)

	// 添加数据库工具
	tools = append(tools, tm.databaseTools.GetTools()...)

	// 添加AI工具
	tools = append(tools, tm.aiTools.GetTools()...)

	return tools
}

// ExecuteTool 执行工具 - 优化版本，使用工具映射提高效率
func (tm *ToolManager) ExecuteTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	// 直接从映射表查找工具执行器
	executor, exists := tm.toolMap[name]
	if !exists {
		return nil, fmt.Errorf("工具未找到: %s", name)
	}

	// 执行工具
	return executor.ExecuteTool(ctx, name, arguments)
}

// GetTool 获取指定工具
func (tm *ToolManager) GetTool(name string) (mcp.Tool, bool) {
	allTools := tm.GetTools()
	for _, tool := range allTools {
		if tool.Name == name {
			return tool, true
		}
	}
	return mcp.Tool{}, false
}

// GetSecurityManager 获取安全管理器
func (tm *ToolManager) GetSecurityManager() *config.SecurityManager {
	return tm.securityManager
}
