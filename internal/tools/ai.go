package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"mcp-ai-server/internal/config"
	"mcp-ai-server/internal/mcp"
)

// AITools AI相关工具
type AITools struct {
	configManager     *config.AIConfigManager
	databaseConfigMgr *config.DatabaseConfigManager
	providers         []AIProvider
	databaseTools     *DatabaseTools
	systemTools       *SystemTools
	dataTools         *DataTools
	networkTools      *NetworkTools
}

// debugPrintAI 调试输出函数，避免在stdio模式下干扰JSON通信
func debugPrintAI(format string, args ...interface{}) {
	// 在stdio模式下，调试信息输出到stderr
	fmt.Fprintf(os.Stderr, format, args...)
}

// debugPrintAICompatible 兼容fmt.Printf的调试输出函数
func debugPrintAICompatible(format string, args ...interface{}) {
	// 重定向所有调试输出到stderr，避免干扰stdio通信
	fmt.Fprintf(os.Stderr, format, args...)
}

// 初始化日志输出到stderr，避免干扰stdio模式的JSON通信
func init() {
	log.SetOutput(os.Stderr)
}

// NewAITools 创建AI工具实例
func NewAITools(configPath string, databaseTools *DatabaseTools, systemTools *SystemTools, dataTools *DataTools, networkTools *NetworkTools) (*AITools, error) {
	// 创建AI工具实例，即使后续失败也返回一个非nil的实例
	aiTools := &AITools{
		providers:     make([]AIProvider, 0),
		databaseTools: databaseTools,
		systemTools:   systemTools,
		dataTools:     dataTools,
		networkTools:  networkTools,
	}

	// 创建AI配置管理器
	configManager, err := config.NewAIConfigManager(configPath)
	if err != nil {
		return aiTools, fmt.Errorf("创建AI配置管理器失败: %v", err)
	}
	aiTools.configManager = configManager

	// 创建数据库配置管理器
	databaseConfigMgr, err := config.NewDatabaseConfigManager(configPath)
	if err != nil {
		return aiTools, fmt.Errorf("创建数据库配置管理器失败: %v", err)
	}
	aiTools.databaseConfigMgr = databaseConfigMgr

	// 初始化提供商
	if err := aiTools.initializeProviders(); err != nil {
		return aiTools, fmt.Errorf("初始化AI提供商失败: %v", err)
	}

	return aiTools, nil
}

// initializeProviders 初始化AI提供商
func (c *AITools) initializeProviders() error {
	if ollamaConfig, exists := c.configManager.GetProvider("ollama"); exists {
		c.providers = append(c.providers, NewOllamaProvider(ollamaConfig))
	}
	if openaiConfig, exists := c.configManager.GetProvider("openai"); exists {
		c.providers = append(c.providers, NewOpenAIProvider(openaiConfig))
	}
	if anthropicConfig, exists := c.configManager.GetProvider("anthropic"); exists {
		c.providers = append(c.providers, NewAnthropicProvider(anthropicConfig))
	}
	return nil
}

// getProvider 按名称查找AI提供商
func (c *AITools) getProvider(name string) (AIProvider, bool) {
	for _, p := range c.providers {
		// 假设 AIProvider 接口有一个 Name() 方法返回其名称
		if p.Name() == name {
			return p, true
		}
	}
	return nil, false
}

// GetTools 获取AI工具列表 - 按照功能复杂度递增排列
func (c *AITools) GetTools() []mcp.Tool {
	return []mcp.Tool{
		// 1. 基础AI对话 - 纯聊天，不涉及数据库
		{
			Name:        "ai_chat",
			Description: "与AI进行基础对话，回答一般问题",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"prompt": map[string]interface{}{
						"type":        "string",
						"description": "对话内容或问题",
					},
					"provider": map[string]interface{}{
						"type":        "string",
						"description": "AI提供商 (ollama, openai, anthropic)",
						"enum":        c.configManager.GetAvailableProviders(),
						"default":     c.configManager.GetDefaultProvider(),
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "使用的模型名称",
						"default":     c.configManager.GetDefaultModel(),
					},
					"max_tokens": map[string]interface{}{
						"type":        "integer",
						"description": "最大生成token数",
						"default":     c.configManager.GetCommonConfig().MaxTokens,
					},
					"temperature": map[string]interface{}{
						"type":        "number",
						"description": "生成温度参数",
						"default":     c.configManager.GetCommonConfig().Temperature,
					},
				},
				"required": []string{"prompt"},
			},
		},
		// 2. SQL生成 - 根据自然语言生成SQL，但不执行
		{
			Name:        "ai_generate_sql",
			Description: "根据自然语言描述生成SQL查询语句（仅生成，不执行）",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"description": map[string]interface{}{
						"type":        "string",
						"description": "自然语言描述，如'查询所有IT部门的员工'",
					},
					"table_name": map[string]interface{}{
						"type":        "string",
						"description": "目标表名（可选，系统会自动检测）",
					},
					"table_schema": map[string]interface{}{
						"type":        "string",
						"description": "表结构信息（可选）",
					},
					"provider": map[string]interface{}{
						"type":        "string",
						"description": "AI提供商",
						"enum":        c.configManager.GetAvailableProviders(),
						"default":     c.configManager.GetDefaultProvider(),
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "使用的模型名称",
						"default":     c.configManager.GetDefaultModel(),
					},
				},
				"required": []string{"description"},
			},
		},
		// 3. 智能查询 - 统一的智能查询工具（自动检测SQL或自然语言）
		{
			Name:        "ai_smart_query",
			Description: "智能查询：自动检测输入类型（SQL语句或自然语言），生成SQL→执行查询",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"prompt": map[string]interface{}{
						"type":        "string",
						"description": "查询描述（可以是自然语言或SQL语句）",
					},
					"analysis_mode": map[string]interface{}{
						"type":        "string",
						"description": "分析模式: 'full'(生成SQL+执行+分析) 或 'fast'(仅生成SQL+执行)",
						"enum":        []string{"full", "fast"},
						"default":     "fast",
					},
					"alias": map[string]interface{}{
						"type":        "string",
						"description": "数据库连接别名",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "查询结果限制条数",
						"default":     100,
					},
					"table_name": map[string]interface{}{
						"type":        "string",
						"description": "目标表名（可选，系统会自动检测）",
					},
					"provider": map[string]interface{}{
						"type":        "string",
						"description": "AI提供商（仅在使用自然语言时需要）",
						"enum":        c.configManager.GetAvailableProviders(),
						"default":     c.configManager.GetDefaultProvider(),
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "使用的模型名称（仅在使用自然语言时需要）",
						"default":     c.configManager.GetDefaultModel(),
					},
				},
				"required": []string{"prompt"},
			},
		},
		// 4. 直接数据查询 - 通过自然语言直接获取数据库数据
		{
			Name:        "ai_query_data",
			Description: "通过自然语言直接获取数据库数据（生成SQL + 执行，不分析）",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"description": map[string]interface{}{
						"type":        "string",
						"description": "自然语言查询描述",
					},
					"table_name": map[string]interface{}{
						"type":        "string",
						"description": "目标表名（可选，系统会自动检测）",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "查询结果限制条数",
						"default":     100,
					},
					"provider": map[string]interface{}{
						"type":        "string",
						"description": "AI提供商",
						"enum":        c.configManager.GetAvailableProviders(),
						"default":     c.configManager.GetDefaultProvider(),
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "使用的模型名称",
						"default":     c.configManager.GetDefaultModel(),
					},
				},
				"required": []string{"description"},
			},
		},
		// 5. 数据分析 - 对已有数据进行AI分析
		{
			Name:        "ai_analyze_data",
			Description: "使用AI分析已有数据并提供洞察",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"data": map[string]interface{}{
						"type":        "string",
						"description": "要分析的数据（JSON格式）",
					},
					"analysis_type": map[string]interface{}{
						"type":        "string",
						"description": "分析类型：summary, insights, recommendations",
						"enum":        []string{"summary", "insights", "recommendations"},
						"default":     "summary",
					},
					"provider": map[string]interface{}{
						"type":        "string",
						"description": "AI提供商",
						"enum":        c.configManager.GetAvailableProviders(),
						"default":     c.configManager.GetDefaultProvider(),
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "使用的模型名称",
						"default":     c.configManager.GetDefaultModel(),
					},
				},
				"required": []string{"data"},
			},
		},
		// 6. 数据查询+分析 - 查询数据并进行AI分析
		{
			Name:        "ai_query_with_analysis",
			Description: "查询数据并进行AI分析（ai_query_data + ai_analyze_data的组合）",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"description": map[string]interface{}{
						"type":        "string",
						"description": "自然语言查询描述",
					},
					"analysis_type": map[string]interface{}{
						"type":        "string",
						"description": "分析类型：summary, insights, recommendations",
						"enum":        []string{"summary", "insights", "recommendations"},
						"default":     "summary",
					},
					"table_name": map[string]interface{}{
						"type":        "string",
						"description": "目标表名（可选，系统会自动检测）",
					},
					"provider": map[string]interface{}{
						"type":        "string",
						"description": "AI提供商",
						"enum":        c.configManager.GetAvailableProviders(),
						"default":     c.configManager.GetDefaultProvider(),
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "使用的模型名称",
						"default":     c.configManager.GetDefaultModel(),
					},
				},
				"required": []string{"description"},
			},
		},
		// 7. 智能洞察 - 深度智能分析，提供业务洞察和建议
		{
			Name:        "ai_smart_insights",
			Description: "深度智能分析，提供业务洞察和建议",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"prompt": map[string]interface{}{
						"type":        "string",
						"description": "分析需求描述",
					},
					"context": map[string]interface{}{
						"type":        "string",
						"description": "额外的上下文信息",
					},
					"insight_level": map[string]interface{}{
						"type":        "string",
						"description": "洞察深度：basic, advanced, strategic",
						"enum":        []string{"basic", "advanced", "strategic"},
						"default":     "basic",
					},
					"table_name": map[string]interface{}{
						"type":        "string",
						"description": "目标表名（可选，系统会自动检测）",
					},
					"provider": map[string]interface{}{
						"type":        "string",
						"description": "AI提供商",
						"enum":        c.configManager.GetAvailableProviders(),
						"default":     c.configManager.GetDefaultProvider(),
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "使用的模型名称",
						"default":     c.configManager.GetDefaultModel(),
					},
				},
				"required": []string{"prompt"},
			},
		},
		// 7. AI智能文件管理 - 自然语言描述的文件操作
		{
			Name:        "ai_file_manager",
			Description: "AI智能文件管理：使用自然语言描述文件操作需求，AI理解后执行相应的文件系统操作",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"instruction": map[string]interface{}{
						"type":        "string",
						"description": "文件操作指令，如'创建一个项目结构'、'查找包含某内容的文件'等",
					},
					"target_path": map[string]interface{}{
						"type":        "string",
						"description": "目标路径（可选）",
					},
					"operation_mode": map[string]interface{}{
						"type":        "string",
						"description": "操作模式：plan_only（仅分析和规划）或 execute（执行操作）",
						"enum":        []string{"plan_only", "execute"},
						"default":     "plan_only",
					},
					"provider": map[string]interface{}{
						"type":        "string",
						"description": "AI提供商",
						"enum":        c.configManager.GetAvailableProviders(),
						"default":     c.configManager.GetDefaultProvider(),
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "使用的模型名称",
						"default":     c.configManager.GetDefaultModel(),
					},
				},
				"required": []string{"instruction"},
			},
		},
		// 8. AI智能数据处理 - 自然语言描述的数据转换
		{
			Name:        "ai_data_processor",
			Description: "AI智能数据处理：使用自然语言描述数据处理需求，AI理解后执行相应的数据转换和分析",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"instruction": map[string]interface{}{
						"type":        "string",
						"description": "数据处理指令，如'解析这个JSON并提取用户信息'、'验证这些数据的格式'等",
					},
					"input_data": map[string]interface{}{
						"type":        "string",
						"description": "输入数据",
					},
					"data_type": map[string]interface{}{
						"type":        "string",
						"description": "数据类型：json, xml, csv, base64等",
						"enum":        []string{"json", "xml", "csv", "base64", "text", "auto"},
						"default":     "auto",
					},
					"output_format": map[string]interface{}{
						"type":        "string",
						"description": "期望的输出格式",
						"enum":        []string{"json", "table", "summary", "original"},
						"default":     "json",
					},
					"provider": map[string]interface{}{
						"type":        "string",
						"description": "AI提供商",
						"enum":        c.configManager.GetAvailableProviders(),
						"default":     c.configManager.GetDefaultProvider(),
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "使用的模型名称",
						"default":     c.configManager.GetDefaultModel(),
					},
				},
				"required": []string{"instruction", "input_data"},
			},
		},
		// 9. AI智能网络请求 - 自然语言描述的API调用
		{
			Name:        "ai_api_client",
			Description: "AI智能网络请求：使用自然语言描述API调用需求，AI理解后构造和执行HTTP请求",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"instruction": map[string]interface{}{
						"type":        "string",
						"description": "API调用指令，如'获取GitHub用户信息'、'发送POST请求到某API'等",
					},
					"base_url": map[string]interface{}{
						"type":        "string",
						"description": "基础URL（可选，AI会从指令中推断）",
					},
					"auth_info": map[string]interface{}{
						"type":        "string",
						"description": "认证信息（可选）",
					},
					"request_mode": map[string]interface{}{
						"type":        "string",
						"description": "请求模式：plan_only（仅生成请求计划）或 execute（执行请求）",
						"enum":        []string{"plan_only", "execute"},
						"default":     "plan_only",
					},
					"response_analysis": map[string]interface{}{
						"type":        "boolean",
						"description": "是否对响应进行AI分析",
						"default":     true,
					},
					"provider": map[string]interface{}{
						"type":        "string",
						"description": "AI提供商",
						"enum":        c.configManager.GetAvailableProviders(),
						"default":     c.configManager.GetDefaultProvider(),
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "使用的模型名称",
						"default":     c.configManager.GetDefaultModel(),
					},
				},
				"required": []string{"instruction"},
			},
		},
		// 10. AI智能系统管理 - 自然语言描述的系统操作
		{
			Name:        "ai_system_admin",
			Description: "AI智能系统管理：使用自然语言描述系统管理需求，AI理解后执行相应的系统操作",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"instruction": map[string]interface{}{
						"type":        "string",
						"description": "系统管理指令，如'检查系统状态'、'清理临时文件'等",
					},
					"safety_mode": map[string]interface{}{
						"type":        "string",
						"description": "安全模式：safe（安全操作）、moderate（中等风险）、advanced（高级操作）",
						"enum":        []string{"safe", "moderate", "advanced"},
						"default":     "safe",
					},
					"operation_mode": map[string]interface{}{
						"type":        "string",
						"description": "操作模式：plan_only（仅分析和规划）或 execute（执行操作）",
						"enum":        []string{"plan_only", "execute"},
						"default":     "plan_only",
					},
					"provider": map[string]interface{}{
						"type":        "string",
						"description": "AI提供商",
						"enum":        c.configManager.GetAvailableProviders(),
						"default":     c.configManager.GetDefaultProvider(),
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "使用的模型名称",
						"default":     c.configManager.GetDefaultModel(),
					},
				},
				"required": []string{"instruction"},
			},
		},
	}
}

// ExecuteTool 执行AI工具 - 按功能分类处理
func (c *AITools) ExecuteTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	switch toolName {
	case "ai_chat":
		return c.executeAIChat(ctx, arguments)
	case "ai_generate_sql":
		return c.executeAIGenerateSQL(ctx, arguments)
	case "ai_smart_query":
		return c.executeAISmartQuery(ctx, arguments)
	case "ai_query_data":
		return c.executeAIQueryData(ctx, arguments)
	case "ai_analyze_data":
		return c.executeAIAnalyzeData(ctx, arguments)
	case "ai_query_with_analysis":
		return c.executeAIQueryWithAnalysis(ctx, arguments)
	case "ai_smart_insights":
		return c.executeAISmartInsights(ctx, arguments)
	case "ai_file_manager":
		return c.executeAIFileManager(ctx, arguments)
	case "ai_data_processor":
		return c.executeAIDataProcessor(ctx, arguments)
	case "ai_api_client":
		return c.executeAIAPIClient(ctx, arguments)
	case "ai_system_admin":
		return c.executeAISystemAdmin(ctx, arguments)
	default:
		return nil, fmt.Errorf("未知的AI工具: %s", toolName)
	}
}

// 1. 基础AI对话 - 纯聊天功能
func (c *AITools) executeAIChat(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	prompt, ok := arguments["prompt"].(string)
	if !ok {
		return nil, fmt.Errorf("prompt参数必须是字符串")
	}

	// 获取AI提供商
	provider, model, err := c.getProviderAndModel(arguments)
	if err != nil {
		return nil, err
	}

	// 获取参数
	maxTokens := c.configManager.GetCommonConfig().MaxTokens
	if mt, ok := arguments["max_tokens"].(float64); ok {
		maxTokens = int(mt)
	}

	temperature := c.configManager.GetCommonConfig().Temperature
	if temp, ok := arguments["temperature"].(float64); ok {
		temperature = temp
	}

	// 调用AI进行对话
	response, err := provider.Call(ctx, model, prompt, map[string]interface{}{
		"max_tokens":  maxTokens,
		"temperature": temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("AI对话失败: %v", err)
	}

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: response,
			},
		},
	}, nil
}

// 2. SQL生成 - 仅生成SQL，不执行（支持自动检测SQL语句）
func (c *AITools) executeAIGenerateSQL(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	description, ok := arguments["description"].(string)
	if !ok {
		return nil, fmt.Errorf("description参数必须是字符串")
	}

	// 检测输入是否已经是SQL语句
	if isSQL := c.detectSQL(description); isSQL {
		// 如果输入已经是SQL，直接返回
		result := map[string]interface{}{
			"tool":          "ai_generate_sql",
			"status":        "success",
			"description":   description,
			"generated_sql": description,
			"input_type":    "direct_sql",
			"message":       "检测到输入已是SQL语句，直接返回",
		}

		jsonResponse, _ := json.MarshalIndent(result, "", "  ")
		return &mcp.ToolCallResult{
			Content: []mcp.Content{
				{
					Type: "text",
					Text: string(jsonResponse),
				},
			},
		}, nil
	}

	// 获取AI提供商（仅在需要生成SQL时）
	provider, model, err := c.getProviderAndModel(arguments)
	if err != nil {
		return nil, err
	}

	// 获取表信息 - 使用智能表名检测或默认表名
	var tableName string
	if tn, ok := arguments["table_name"].(string); ok && tn != "" {
		tableName = tn
	} else {
		// 使用智能表名检测
		defaultTable := c.getDefaultTableName(ctx)
		detectedTable, err := c.intelligentTableDetection(ctx, description, defaultTable)
		if err != nil {
			debugPrintAI("[DEBUG] 智能表名检测失败: %v，使用默认表名: %s\n", err, defaultTable)
			tableName = defaultTable
		} else {
			tableName = detectedTable
			debugPrintAI("[DEBUG] 智能表名检测成功，使用表名: %s\n", tableName)
		}
	}

	tableSchema := ""
	if ts, ok := arguments["table_schema"].(string); ok {
		tableSchema = ts
	}

	// 构建SQL生成提示词
	var prompt string
	if tableSchema != "" {
		prompt = fmt.Sprintf(`表结构信息：
表名：%s
字段：%s

请根据以下需求生成SQL查询语句：%s

要求：
1. 只返回SQL语句，不要任何解释
2. 确保SQL语法正确
3. 如果需求不明确，生成一个合理的查询`, tableName, tableSchema, description)
	} else {
		// 使用默认字段信息
		defaultFields := "id, username, email, full_name, age, department, position, salary, is_active, created_at, updated_at"
		prompt = fmt.Sprintf(`表信息：
表名：%s
字段：%s

请根据以下需求生成SQL查询语句：%s

要求：
1. 只返回SQL语句，不要任何解释
2. 确保SQL语法正确
3. 如果需求不明确，生成一个合理的查询`, tableName, defaultFields, description)
	}

	// 调用AI生成SQL
	response, err := provider.Call(ctx, model, prompt, map[string]interface{}{
		"max_tokens":  500,
		"temperature": 0.3, // 低温度确保SQL准确性
	})
	if err != nil {
		return nil, fmt.Errorf("SQL生成失败: %v", err)
	}

	debugPrintAI("[GenerateSQL] AI响应: %s\n", response)

	// 提取SQL语句
	sql := extractSQLFromAIResponse(response)
	if sql == "" {
		// 如果提取失败，尝试直接使用响应作为SQL
		cleanedResponse := strings.TrimSpace(response)
		cleanedResponse = strings.ReplaceAll(cleanedResponse, "\n", " ")
		cleanedResponse = strings.ReplaceAll(cleanedResponse, "  ", " ")

		// 如果看起来像SQL，就使用它
		if strings.Contains(strings.ToUpper(cleanedResponse), "SELECT") &&
			strings.Contains(strings.ToUpper(cleanedResponse), "FROM") {
			sql = cleanedResponse
		}
	}

	debugPrintAI("[GenerateSQL] 提取的SQL: %s\n", sql)

	if sql == "" {
		return &mcp.ToolCallResult{
			Content: []mcp.Content{
				{
					Type: "text",
					Text: fmt.Sprintf("AI无法理解需求描述，原始响应：%s", response),
				},
			},
		}, nil
	}

	// 返回结构化结果
	result := map[string]interface{}{
		"tool":          "ai_generate_sql",
		"status":        "success",
		"description":   description,
		"table_name":    tableName,
		"generated_sql": sql,
		"provider":      provider.Name(),
		"model":         model,
	}

	jsonResponse, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: string(jsonResponse),
			},
		},
	}, nil
}

func (c *AITools) executeAIExecuteSQL(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	// 步骤1: 接收用户自然语言输入
	prompt, hasPrompt := arguments["prompt"].(string)
	if !hasPrompt || prompt == "" {
		return nil, fmt.Errorf("必须提供prompt参数")
	}

	// 获取AI提供商（必须使用AI）
	provider, model, err := c.getProviderAndModel(arguments)
	if err != nil {
		debugPrintAI("[DEBUG] 获取AI提供商失败: %v\n", err)
		return nil, fmt.Errorf("AI不可用: %v", err)
	}

	debugPrintAI("[DEBUG] 步骤1-2: 接收用户输入: %s\n", prompt)

	// 获取数据库连接别名
	alias := ""
	if a, ok := arguments["alias"].(string); ok {
		alias = a
	}

	// 获取查询限制
	limit := 100
	if l, ok := arguments["limit"].(float64); ok {
		limit = int(l)
	}

	// 步骤3: 调用AI生成SQL（使用executeAIGenerateSQL）
	debugPrintAI("[DEBUG] 步骤3: 开始AI生成SQL...\n")
	sqlGenArgs := map[string]interface{}{
		"description": prompt,
		"provider":    provider.Name(),
		"model":       model,
	}

	// 传递表名信息（如果有）
	if tableName, ok := arguments["table_name"]; ok {
		sqlGenArgs["table_name"] = tableName
	}
	if tableSchema, ok := arguments["table_schema"]; ok {
		sqlGenArgs["table_schema"] = tableSchema
	}

	sqlResult, err := c.executeAIGenerateSQL(ctx, sqlGenArgs)
	if err != nil {
		return nil, fmt.Errorf("步骤3失败-SQL生成失败: %v", err)
	}

	// 解析生成的SQL
	var sqlData map[string]interface{}
	if err := json.Unmarshal([]byte(sqlResult.Content[0].Text), &sqlData); err != nil {
		return nil, fmt.Errorf("步骤3失败-解析SQL生成结果失败: %v", err)
	}

	generatedSQL, ok := sqlData["generated_sql"].(string)
	if !ok {
		return nil, fmt.Errorf("步骤3失败-未能从SQL生成结果中提取SQL语句")
	}

	debugPrintAI("[DEBUG] 步骤3完成: 生成SQL: %s\n", generatedSQL)

	// 步骤4: SQL安全验证
	debugPrintAI("[DEBUG] 步骤4: 开始SQL安全验证...\n")
	if err := c.validateSQL(generatedSQL); err != nil {
		return nil, fmt.Errorf("步骤4失败-SQL安全验证失败: %v", err)
	}

	// 步骤5-7: 执行数据库查询
	debugPrintAI("[DEBUG] 步骤5-7: 开始执行数据库查询...\n")
	var queryResult *mcp.ToolCallResult
	var queryError error

	if c.databaseTools != nil {
		// 构建数据库查询参数
		dbArgs := map[string]interface{}{
			"sql":   generatedSQL,
			"limit": limit,
		}

		// 如果提供了别名，使用指定的数据库连接
		if alias != "" {
			dbArgs["alias"] = alias
		}

		// 调用数据库工具执行查询
		queryResult, queryError = c.databaseTools.ExecuteTool(ctx, "db_query", dbArgs)

		if queryError != nil {
			debugPrintAI("[DEBUG] 步骤5-7失败: 数据库查询错误: %v\n", queryError)
		} else {
			debugPrintAI("[DEBUG] 步骤5-7完成: 数据库查询成功\n")
		}
	} else {
		queryError = fmt.Errorf("数据库工具不可用")
		debugPrintAI("[DEBUG] 步骤5-7失败: %v\n", queryError)
	}

	// 步骤8: 处理查询结果 - 直接返回数据或错误
	debugPrintAI("[DEBUG] 步骤8: 处理查询结果...\n")
	var queryData interface{}

	// 如果查询失败，直接返回错误
	if queryError != nil {
		debugPrintAI("[DEBUG] 查询失败，直接返回错误: %v\n", queryError)
		return nil, fmt.Errorf("数据库查询失败: %v", queryError)
	}

	// 查询成功，解析并返回原始数据
	if queryResult != nil && len(queryResult.Content) > 0 {
		// 解析查询结果
		var dbResponse map[string]interface{}
		if err := json.Unmarshal([]byte(queryResult.Content[0].Text), &dbResponse); err == nil {
			queryData = dbResponse
		} else {
			queryData = queryResult.Content[0].Text
		}
		debugPrintAI("[DEBUG] 查询成功，准备返回原始数据\n")
	}

	debugPrintAI("[DEBUG] 步骤8完成: 查询结果处理完成\n")

	// 构建最终响应
	response := map[string]interface{}{
		"tool":          "ai_smart_sql",
		"status":        "success",
		"prompt":        prompt,
		"generated_sql": generatedSQL,
		"provider":      provider.Name(),
		"model":         model,
		"execution_flow": []string{
			"1. 接收用户自然语言输入",
			"2. AI理解用户意图",
			"3. AI生成SQL查询语句",
			"4. SQL安全验证",
			"5. 使用预配置数据库连接",
			"6. 执行SQL查询",
			"7. 获取查询结果",
			"8. 返回原始查询数据",
		},
	}

	// 添加查询执行信息和原始数据
	response["query_execution"] = map[string]interface{}{
		"success": true,
	}
	if queryData != nil {
		response["query_data"] = queryData
	}

	// 添加技术细节（用于调试）
	response["technical_details"] = map[string]interface{}{
		"sql_validation": "passed",
		"database_alias": alias,
		"query_limit":    limit,
		"ai_provider":    provider.Name(),
		"ai_model":       model,
	}

	jsonResponse, _ := json.MarshalIndent(response, "", "  ")
	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: string(jsonResponse),
			},
		},
	}, nil
}

// validateSQL SQL安全验证方法 - 本地调试版本（宽松验证）
func (c *AITools) validateSQL(sql string) error {
	// 基础安全检查
	upperSQL := strings.ToUpper(sql)

	debugPrintAI("[DEBUG] SQL安全验证 - 输入SQL: %s\n", sql)

	// 只检查最危险的操作 - 防止误删数据
	dangerousKeywords := []string{"DROP", "DELETE", "TRUNCATE"}
	for _, keyword := range dangerousKeywords {
		if strings.Contains(upperSQL, keyword) {
			debugPrintAI("[DEBUG] 发现危险操作: %s\n", keyword)
			return fmt.Errorf("为了安全，不允许 %s 操作", keyword)
		}
	}

	// 检查是否是查询语句（允许CREATE、INSERT、UPDATE用于本地调试）
	if !strings.Contains(upperSQL, "SELECT") &&
		!strings.Contains(upperSQL, "CREATE") &&
		!strings.Contains(upperSQL, "INSERT") &&
		!strings.Contains(upperSQL, "UPDATE") {
		debugPrintAI("[DEBUG] 未识别的SQL类型\n")
		return fmt.Errorf("只支持 SELECT, CREATE, INSERT, UPDATE 操作")
	}

	// 本地调试环境：跳过复杂的SQL注入检查
	debugPrintAI("[DEBUG] SQL安全验证通过（本地调试模式）\n")
	return nil
}

// 3. 智能查询 - 统一的智能查询工具（自动检测SQL或自然语言）
func (c *AITools) executeAISmartQuery(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	debugPrintAI("[DEBUG] ====== executeAISmartQuery 开始 ======\n")
	debugPrintAI("[DEBUG] 接收到的参数: %+v\n", arguments)

	// 步骤1: 接收用户输入
	prompt, hasPrompt := arguments["prompt"].(string)
	if !hasPrompt || prompt == "" {
		debugPrintAI("[DEBUG] ERROR: prompt参数缺失或为空\n")
		return nil, fmt.Errorf("必须提供prompt参数")
	}

	debugPrintAI("[DEBUG] 步骤1完成 - 输入prompt: '%s'\n", prompt)

	// 步骤2: 自动检测输入类型
	isDirectSQL := c.detectSQL(prompt)
	debugPrintAI("[DEBUG] 步骤2完成 - SQL检测结果: %v\n", isDirectSQL)

	// 获取通用参数
	alias := ""
	if a, ok := arguments["alias"].(string); ok {
		alias = a
	}

	limit := 100
	if l, ok := arguments["limit"].(float64); ok {
		limit = int(l)
	}

	analysisMode := "fast"
	if mode, ok := arguments["analysis_mode"].(string); ok {
		analysisMode = mode
	}

	tableName := "mcp_user"
	if tn, ok := arguments["table_name"].(string); ok && tn != "" {
		tableName = tn
	} else {
		// 如果没有指定表名，使用智能表名识别
		debugPrintAI("[DEBUG] 未指定表名，启动智能表名识别\n")
		detectedTable, err := c.intelligentTableDetection(ctx, prompt, tableName)
		if err != nil {
			debugPrintAI("[DEBUG] 智能表名识别失败: %v，使用默认表名: %s\n", err, tableName)
		} else {
			tableName = detectedTable
			debugPrintAI("[DEBUG] 智能表名识别成功，使用表名: %s\n", tableName)
		}
	}

	debugPrintAI("[DEBUG] 参数解析完成 - alias: '%s', limit: %d, analysisMode: '%s', tableName: '%s'\n",
		alias, limit, analysisMode, tableName)

	var finalSQL string
	var inputType string

	// 步骤3: 根据输入类型处理
	if isDirectSQL {
		// 场景1：直接SQL执行
		debugPrintAI("[DEBUG] 步骤3 - 进入直接SQL模式\n")
		finalSQL = prompt
		inputType = "direct_sql"
		debugPrintAI("[DEBUG] 步骤3完成 - 直接使用SQL: '%s'\n", finalSQL)
	} else {
		// 场景2：自然语言查询（需要AI生成SQL）
		debugPrintAI("[DEBUG] 步骤3 - 进入自然语言模式，需要AI生成SQL\n")

		// 获取AI提供商（仅在需要生成SQL时）
		provider, model, err := c.getProviderAndModel(arguments)
		if err != nil {
			debugPrintAI("[DEBUG] ERROR: 获取AI提供商失败: %v\n", err)
			return nil, fmt.Errorf("AI不可用（自然语言查询需要AI支持）: %v", err)
		}
		debugPrintAI("[DEBUG] AI提供商获取成功 - provider: %s, model: %s\n", provider.Name(), model)

		// 调用AI生成SQL
		sqlGenArgs := map[string]interface{}{
			"description": prompt,
			"table_name":  tableName,
			"provider":    provider.Name(),
			"model":       model,
		}
		debugPrintAICompatible("[DEBUG] 准备调用executeAIGenerateSQL，参数: %+v\n", sqlGenArgs)

		sqlResult, err := c.executeAIGenerateSQL(ctx, sqlGenArgs)
		if err != nil {
			debugPrintAICompatible("[DEBUG] ERROR: SQL生成失败: %v\n", err)
			return nil, fmt.Errorf("SQL生成失败: %v", err)
		}
		debugPrintAICompatible("[DEBUG] executeAIGenerateSQL调用成功，返回内容长度: %d\n", len(sqlResult.Content[0].Text))

		// 解析生成的SQL
		var sqlData map[string]interface{}
		if err := json.Unmarshal([]byte(sqlResult.Content[0].Text), &sqlData); err != nil {
			debugPrintAICompatible("[DEBUG] ERROR: 解析SQL生成结果失败: %v\n", err)
			debugPrintAICompatible("[DEBUG] 原始SQL生成结果: %s\n", sqlResult.Content[0].Text)
			return nil, fmt.Errorf("解析SQL生成结果失败: %v", err)
		}
		debugPrintAICompatible("[DEBUG] SQL生成结果解析成功: %+v\n", sqlData)

		generatedSQL, ok := sqlData["generated_sql"].(string)
		if !ok {
			debugPrintAICompatible("[DEBUG] ERROR: 无法提取generated_sql字段\n")
			debugPrintAICompatible("[DEBUG] sqlData内容: %+v\n", sqlData)
			return nil, fmt.Errorf("未能从SQL生成结果中提取SQL语句")
		}

		if generatedSQL == "" {
			debugPrintAICompatible("[DEBUG] ERROR: generated_sql字段为空\n")
			debugPrintAICompatible("[DEBUG] sqlData内容: %+v\n", sqlData)
			return nil, fmt.Errorf("生成的SQL语句为空")
		}

		finalSQL = generatedSQL
		inputType = "natural_language"
		debugPrintAICompatible("[DEBUG] 步骤3完成 - AI生成SQL: '%s'\n", finalSQL)
	}

	// 步骤4: SQL安全验证
	debugPrintAICompatible("[DEBUG] 步骤4 - 开始SQL安全验证，SQL: '%s'\n", finalSQL)
	if err := c.validateSQL(finalSQL); err != nil {
		debugPrintAICompatible("[DEBUG] ERROR: SQL安全验证失败: %v\n", err)
		return nil, fmt.Errorf("SQL安全验证失败: %v", err)
	}
	debugPrintAICompatible("[DEBUG] 步骤4完成 - SQL安全验证通过\n")

	// 步骤5: 执行数据库查询
	debugPrintAICompatible("[DEBUG] 步骤5 - 开始执行数据库查询\n")
	var queryResult *mcp.ToolCallResult
	var queryError error

	if c.databaseTools != nil {
		debugPrintAICompatible("[DEBUG] 数据库工具可用，准备执行查询\n")
		dbArgs := map[string]interface{}{
			"sql":   finalSQL,
			"limit": limit,
		}

		// 设置数据库别名，如果没有提供则使用默认值
		if alias != "" {
			dbArgs["alias"] = alias
			debugPrintAICompatible("[DEBUG] 使用指定的数据库别名: %s\n", alias)
		} else {
			// 使用配置文件中的默认数据库别名
			dbArgs["alias"] = "mysql_test"
			debugPrintAICompatible("[DEBUG] 使用默认数据库别名: mysql_test\n")
		}

		debugPrintAICompatible("[DEBUG] 数据库查询参数: %+v\n", dbArgs)
		queryResult, queryError = c.databaseTools.ExecuteTool(ctx, "db_query", dbArgs)

		if queryError != nil {
			debugPrintAICompatible("[DEBUG] ERROR: 数据库查询失败: %v\n", queryError)
		} else {
			debugPrintAICompatible("[DEBUG] 数据库查询成功，结果长度: %d\n", len(queryResult.Content[0].Text))
		}
	} else {
		queryError = fmt.Errorf("数据库工具不可用")
		debugPrintAICompatible("[DEBUG] ERROR: 数据库工具不可用\n")
	}

	// 步骤6: 构建响应
	debugPrintAICompatible("[DEBUG] 步骤6 - 开始构建响应\n")
	result := map[string]interface{}{
		"tool":          "ai_smart_query",
		"status":        "success",
		"input_type":    inputType,
		"prompt":        prompt,
		"sql":           finalSQL,
		"analysis_mode": analysisMode,
		"limit":         limit,
		"row_count":     0,
	}

	if alias != "" {
		result["alias"] = alias
	}

	if inputType == "natural_language" {
		result["table_name"] = tableName
	}

	debugPrintAICompatible("[DEBUG] 基础响应结构创建完成\n")

	// 处理查询结果
	if queryError != nil {
		result["status"] = "error"
		result["error"] = queryError.Error()
		debugPrintAICompatible("[DEBUG] ERROR: 设置错误状态，错误信息: %v\n", queryError)
	} else if queryResult != nil {
		debugPrintAICompatible("[DEBUG] 开始解析数据库查询结果\n")
		// 解析数据库查询结果
		var dbResponse map[string]interface{}
		if err := json.Unmarshal([]byte(queryResult.Content[0].Text), &dbResponse); err == nil {
			debugPrintAICompatible("[DEBUG] 数据库结果解析成功: %+v\n", dbResponse)

			if dbResult, ok := dbResponse["result"].(map[string]interface{}); ok {
				result["result"] = dbResult
				if rowCount, ok := dbResult["row_count"]; ok {
					result["row_count"] = rowCount
					debugPrintAICompatible("[DEBUG] 设置行数: %v\n", rowCount)
				}
				if columns, ok := dbResult["columns"]; ok {
					result["columns"] = columns
					debugPrintAICompatible("[DEBUG] 设置列信息，列数: %d\n", len(columns.([]interface{})))
				}
				if rows, ok := dbResult["rows"]; ok {
					result["rows"] = rows
					debugPrintAICompatible("[DEBUG] 设置行数据，行数: %d\n", len(rows.([]interface{})))
				}
				if limited, ok := dbResult["limited"]; ok {
					result["limited"] = limited
				}
			} else {
				result["raw_result"] = dbResponse
				debugPrintAICompatible("[DEBUG] 使用原始结果格式\n")
			}

			// 步骤7: 可选AI分析
			if analysisMode == "full" && inputType == "natural_language" {
				debugPrintAICompatible("[DEBUG] 步骤7 - 开始执行AI分析\n")

				// 获取AI提供商进行分析
				if provider, model, err := c.getProviderAndModel(arguments); err == nil {
					analysisResult := c.analyzeQueryResult(ctx, provider, model, prompt, finalSQL, queryResult.Content[0].Text, nil)
					result["ai_analysis"] = analysisResult
					debugPrintAICompatible("[DEBUG] AI分析完成，结果长度: %d\n", len(analysisResult))
				} else {
					debugPrintAICompatible("[DEBUG] WARNING: AI分析跳过，无法获取AI提供商: %v\n", err)
				}
			} else {
				debugPrintAICompatible("[DEBUG] 跳过AI分析 - analysisMode: %s, inputType: %s\n", analysisMode, inputType)
			}
		} else {
			result["status"] = "error"
			result["error"] = fmt.Sprintf("解析数据库结果失败: %v", err)
			debugPrintAICompatible("[DEBUG] ERROR: 解析数据库结果失败: %v\n", err)
			debugPrintAICompatible("[DEBUG] 原始数据库结果: %s\n", queryResult.Content[0].Text)
		}
	} else {
		debugPrintAICompatible("[DEBUG] WARNING: queryResult为nil\n")
	}

	debugPrintAICompatible("[DEBUG] 响应构建完成，最终结果状态: %s\n", result["status"])
	jsonResponse, _ := json.MarshalIndent(result, "", "  ")
	debugPrintAICompatible("[DEBUG] ====== executeAISmartQuery 结束 ======\n")
	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: string(jsonResponse),
			},
		},
	}, nil
}

// 辅助方法：获取AI提供商和模型
func (c *AITools) getProviderAndModel(arguments map[string]interface{}) (AIProvider, string, error) {
	// 获取提供商
	providerName := c.configManager.GetDefaultProvider()
	if p, ok := arguments["provider"].(string); ok && p != "" {
		providerName = p
	}

	provider, exists := c.getProvider(providerName)
	if !exists || !provider.IsEnabled() {
		return nil, "", fmt.Errorf("AI提供商 %s 不可用或未启用", providerName)
	}

	// 获取模型
	model := c.configManager.GetDefaultModel()
	if m, ok := arguments["model"].(string); ok && m != "" {
		model = m
	}

	return provider, model, nil
}

// executeAIAnalyzeData 执行AI数据分析
func (c *AITools) executeAIAnalyzeData(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	data, ok := arguments["data"].(string)
	if !ok {
		return nil, fmt.Errorf("data参数必须是字符串")
	}

	analysisType := "summary"
	if at, ok := arguments["analysis_type"].(string); ok {
		analysisType = at
	}

	// 获取AI提供商
	provider, model, err := c.getProviderAndModel(arguments)
	if err != nil {
		return nil, err
	}

	// 构建中文分析提示词
	var prompt string
	switch analysisType {
	case "summary":
		prompt = fmt.Sprintf("请用中文分析以下数据并提供摘要。请直接返回分析结果，不要包含额外的格式化字符：\n%s", data)
	case "insights":
		prompt = fmt.Sprintf("请用中文分析以下数据并提供洞察和发现。请直接返回分析结果，不要包含额外的格式化字符：\n%s", data)
	case "recommendations":
		prompt = fmt.Sprintf("请用中文分析以下数据并提供建议和推荐。请直接返回分析结果，不要包含额外的格式化字符：\n%s", data)
	case "detailed":
		prompt = fmt.Sprintf("请用中文对以下数据进行详细分析。请直接返回分析结果，不要包含额外的格式化字符：\n%s", data)
	default:
		prompt = fmt.Sprintf("请用中文分析以下数据。请直接返回分析结果，不要包含额外的格式化字符：\n%s", data)
	}

	// 调用AI提供商
	response, err := provider.Call(ctx, model, prompt, map[string]interface{}{
		"max_tokens": c.configManager.GetCommonConfig().MaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("数据分析失败: %v", err)
	}

	// 增强的清理AI响应格式
	cleanedResponse := cleanAIResponse(response)

	// 进一步清理可能的JSON转义字符
	cleanedResponse = strings.ReplaceAll(cleanedResponse, "\\n", "\n")
	cleanedResponse = strings.ReplaceAll(cleanedResponse, "\\\"", "\"")
	cleanedResponse = strings.ReplaceAll(cleanedResponse, "\\\\", "\\")

	// 移除开头和结尾的引号（如果存在）
	cleanedResponse = strings.Trim(cleanedResponse, "\"")

	// 构建结构化响应
	result := map[string]interface{}{
		"tool":          "ai_analyze_data",
		"status":        "success",
		"analysis_type": analysisType,
		"provider":      provider.Name(),
		"model":         model,
		"analysis":      cleanedResponse,
	}

	jsonResponse, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: string(jsonResponse),
			},
		},
	}, nil
}

// executeAIAnalyzeDataWithChinesePrompt 专门用于中文分析的方法
func (c *AITools) executeAIAnalyzeDataWithChinesePrompt(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	data, ok := arguments["data"].(string)
	if !ok {
		return nil, fmt.Errorf("data参数必须是字符串")
	}

	analysisType := "summary"
	if at, ok := arguments["analysis_type"].(string); ok {
		analysisType = at
	}

	// 获取AI提供商
	provider, model, err := c.getProviderAndModel(arguments)
	if err != nil {
		return nil, err
	}

	// 构建中文分析提示词，更加明确要求使用中文
	var prompt string
	switch analysisType {
	case "summary":
		prompt = fmt.Sprintf("请严格用中文分析以下数据并提供摘要。必须用中文回答，不要使用英文。请直接返回分析结果：\n\n数据：%s\n\n要求：用中文分析并提供摘要", data)
	case "insights":
		prompt = fmt.Sprintf("请严格用中文分析以下数据并提供洞察和发现。必须用中文回答，不要使用英文。请直接返回分析结果：\n\n数据：%s\n\n要求：用中文分析并提供洞察", data)
	case "recommendations":
		prompt = fmt.Sprintf("请严格用中文分析以下数据并提供建议和推荐。必须用中文回答，不要使用英文。请直接返回分析结果：\n\n数据：%s\n\n要求：用中文分析并提供建议", data)
	case "detailed":
		prompt = fmt.Sprintf("请严格用中文对以下数据进行详细分析。必须用中文回答，不要使用英文。请直接返回分析结果：\n\n数据：%s\n\n要求：用中文进行详细分析，包括数据统计、趋势分析和业务洞察", data)
	default:
		prompt = fmt.Sprintf("请严格用中文分析以下数据。必须用中文回答，不要使用英文。请直接返回分析结果：\n\n数据：%s\n\n要求：用中文分析数据", data)
	}

	// 调用AI提供商
	response, err := provider.Call(ctx, model, prompt, map[string]interface{}{
		"max_tokens": c.configManager.GetCommonConfig().MaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("数据分析失败: %v", err)
	}

	// 增强的清理AI响应格式
	cleanedResponse := cleanAIResponse(response)

	// 进一步清理可能的JSON转义字符
	cleanedResponse = strings.ReplaceAll(cleanedResponse, "\\n", "\n")
	cleanedResponse = strings.ReplaceAll(cleanedResponse, "\\\"", "\"")
	cleanedResponse = strings.ReplaceAll(cleanedResponse, "\\\\", "\\")

	// 移除开头和结尾的引号（如果存在）
	cleanedResponse = strings.Trim(cleanedResponse, "\"")

	// 构建结构化响应
	result := map[string]interface{}{
		"tool":          "ai_analyze_data",
		"status":        "success",
		"analysis_type": analysisType,
		"provider":      provider.Name(),
		"model":         model,
		"analysis":      cleanedResponse,
	}

	jsonResponse, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: string(jsonResponse),
			},
		},
	}, nil
}

// 辅助函数

// detectSQL 检测输入字符串是否是SQL语句
func (c *AITools) detectSQL(input string) bool {
	// 清理输入
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return false
	}

	upper := strings.ToUpper(trimmed)

	// 检查是否以SQL关键字开头
	sqlKeywords := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "WITH", "CREATE", "ALTER", "DROP"}
	for _, keyword := range sqlKeywords {
		if strings.HasPrefix(upper, keyword) {
			return true
		}
	}

	// 检查是否包含典型的SQL结构
	sqlPatterns := []string{
		"SELECT.*FROM",
		"INSERT.*INTO",
		"UPDATE.*SET",
		"DELETE.*FROM",
	}

	for _, pattern := range sqlPatterns {
		if matched, _ := regexp.MatchString(pattern, upper); matched {
			return true
		}
	}

	return false
}

// cleanAIResponse 清理AI响应中的格式问题
func cleanAIResponse(response string) string {
	// 移除开头和结尾的换行符
	cleaned := strings.TrimSpace(response)

	// 移除多余的开头换行符
	for strings.HasPrefix(cleaned, "\n") {
		cleaned = strings.TrimPrefix(cleaned, "\n")
	}

	// 移除多余的结尾换行符
	for strings.HasSuffix(cleaned, "\n") {
		cleaned = strings.TrimSuffix(cleaned, "\n")
	}

	// 清理连续的多个换行符，替换为单个换行符
	cleaned = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(cleaned, "\n\n")

	return strings.TrimSpace(cleaned)
}

// analyzeQueryResult 分析查询结果（用于智能查询的AI分析）
func (c *AITools) analyzeQueryResult(ctx context.Context, provider AIProvider, model, prompt, generatedSQL, dbResult string, queryError error) string {
	var analysisPrompt string

	if queryError != nil {
		analysisPrompt = fmt.Sprintf(`用户需求：%s

AI生成的SQL：%s

执行结果：查询失败，错误信息：%v

请分析失败原因并给出改进建议。不要重复SQL语句和错误信息，只需要提供分析和建议。`, prompt, generatedSQL, queryError)
	} else {
		analysisPrompt = fmt.Sprintf(`用户需求：%s

AI生成的SQL：%s

我已经获得了查询结果数据。请基于这个查询提供高层次的业务分析洞察，用中文回答。

重要要求：
- 绝对不要重复或复述任何具体的数据值（如姓名、邮箱、部门名称等）
- 不要说"从查询结果中我们可以看到..."这样的表述
- 专注于提供宏观的业务洞察和建议
- 基于查询类型和结构进行分析，而非具体数据内容

请提供以下方面的分析：
1. 查询类型评估：这个查询主要关注什么业务问题？
2. 数据结构洞察：表结构反映了什么业务模式？
3. 潜在应用场景：这类查询通常用于什么业务决策？
4. 优化建议：如何改进查询效率或扩展分析维度？`, prompt, generatedSQL)
	}

	// 调用AI进行分析
	analysisResponse, err := provider.Call(ctx, model, analysisPrompt, map[string]interface{}{
		"max_tokens":  1000,
		"temperature": 0.5,
	})
	if err != nil {
		return fmt.Sprintf("AI分析失败: %v", err)
	}

	return analysisResponse
}

// extractSQLFromAIResponse 从AI响应中提取SQL语句
func extractSQLFromAIResponse(aiResponse string) string {
	// 清理AI响应，提取SQL语句
	lines := strings.Split(aiResponse, "\n")
	var sqlLines []string
	var inSQLBlock bool

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 检测SQL代码块开始
		if strings.Contains(line, "```sql") || strings.Contains(line, "```") {
			inSQLBlock = !inSQLBlock
			continue
		}

		// 如果在SQL代码块内，收集SQL行
		if inSQLBlock && line != "" {
			sqlLines = append(sqlLines, line)
		}

		// 检测是否包含SELECT语句（不在代码块内）
		if !inSQLBlock && strings.Contains(strings.ToUpper(line), "SELECT") {
			sqlLines = append(sqlLines, line)
		}
	}

	// 如果没有找到代码块，尝试直接提取包含SELECT的行
	if len(sqlLines) == 0 {
		for _, line := range lines {
			if strings.Contains(strings.ToUpper(line), "SELECT") {
				sqlLines = append(sqlLines, strings.TrimSpace(line))
			}
		}
	}

	// 合并SQL行
	if len(sqlLines) > 0 {
		return strings.Join(sqlLines, " ")
	}

	return ""
}

// 4. 直接数据查询 - 通过自然语言直接获取数据库数据
func (c *AITools) executeAIQueryData(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	description, ok := arguments["description"].(string)
	if !ok {
		return nil, fmt.Errorf("description参数必须是字符串")
	}

	// 获取AI提供商
	provider, model, err := c.getProviderAndModel(arguments)
	if err != nil {
		return nil, err
	}

	// 获取表信息 - 使用智能表名检测或默认表名
	var tableName string
	if tn, ok := arguments["table_name"].(string); ok && tn != "" {
		tableName = tn
	} else {
		// 使用智能表名检测
		defaultTable := c.getDefaultTableName(ctx)
		detectedTable, err := c.intelligentTableDetection(ctx, description, defaultTable)
		if err != nil {
			debugPrintAICompatible("[DEBUG] 智能表名检测失败: %v，使用默认表名: %s\n", err, defaultTable)
			tableName = defaultTable
		} else {
			tableName = detectedTable
			debugPrintAICompatible("[DEBUG] 智能表名检测成功，使用表名: %s\n", tableName)
		}
	}

	// 第一步：生成SQL
	sqlGenArgs := map[string]interface{}{
		"description": description,
		"table_name":  tableName,
		"provider":    provider.Name(),
		"model":       model,
	}
	sqlResult, err := c.executeAIGenerateSQL(ctx, sqlGenArgs)
	if err != nil {
		return nil, fmt.Errorf("SQL生成失败: %v", err)
	}

	// 解析生成的SQL
	var sqlData map[string]interface{}
	if err := json.Unmarshal([]byte(sqlResult.Content[0].Text), &sqlData); err != nil {
		return nil, fmt.Errorf("解析SQL生成结果失败: %v", err)
	}

	generatedSQL, ok := sqlData["generated_sql"].(string)
	if !ok {
		return nil, fmt.Errorf("未能从SQL生成结果中提取SQL语句")
	}

	// 第二步：执行SQL（调用AI智能查询）
	execArgs := map[string]interface{}{
		"prompt":     description,
		"table_name": tableName,
		"provider":   provider.Name(),
		"model":      model,
	}

	// 传递limit参数（如果有）
	if limit, ok := arguments["limit"]; ok {
		execArgs["limit"] = limit
	}

	// 传递alias参数（如果有）
	if alias, ok := arguments["alias"]; ok {
		execArgs["alias"] = alias
	}

	execResult, err := c.executeAIExecuteSQL(ctx, execArgs)

	// 构建响应
	response := map[string]interface{}{
		"tool":          "ai_query_data",
		"status":        "success",
		"description":   description,
		"table_name":    tableName,
		"generated_sql": generatedSQL,
		"provider":      provider.Name(),
		"model":         model,
	}

	// 处理SQL执行结果
	if err != nil {
		response["status"] = "error"
		response["execution"] = map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	} else {
		response["execution"] = map[string]interface{}{
			"success": true,
		}

		// 解析执行结果
		if execResult != nil && len(execResult.Content) > 0 {
			var execData map[string]interface{}
			if err := json.Unmarshal([]byte(execResult.Content[0].Text), &execData); err == nil {
				if result, ok := execData["result"]; ok {
					response["data"] = result
				}
				if rowCount, ok := execData["row_count"]; ok {
					response["execution"].(map[string]interface{})["row_count"] = rowCount
				}
			}
		}
	}

	jsonResponse, _ := json.MarshalIndent(response, "", "  ")
	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: string(jsonResponse),
			},
		},
	}, nil
}

// 6. 数据查询+分析 - 查询数据并进行AI分析
func (c *AITools) executeAIQueryWithAnalysis(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	description, ok := arguments["description"].(string)
	if !ok {
		return nil, fmt.Errorf("description参数必须是字符串")
	}

	analysisType := "summary"
	if at, ok := arguments["analysis_type"].(string); ok {
		analysisType = at
	}

	// 第一步：使用智能查询获取数据
	queryArgs := map[string]interface{}{
		"prompt":        description,
		"analysis_mode": "fast", // 只需要查询数据，不需要在这里分析
	}
	if tableName, ok := arguments["table_name"]; ok {
		queryArgs["table_name"] = tableName
	}
	if provider, ok := arguments["provider"]; ok {
		queryArgs["provider"] = provider
	}
	if model, ok := arguments["model"]; ok {
		queryArgs["model"] = model
	}

	log.Printf("[QueryWithAnalysis] 开始查询数据，描述：%s", description)

	queryResult, err := c.executeAISmartQuery(ctx, queryArgs)
	if err != nil {
		return nil, fmt.Errorf("数据查询失败: %v", err)
	}

	log.Printf("[QueryWithAnalysis] 数据查询完成")

	// 第二步：分析数据 - 使用中文提示词
	log.Printf("[QueryWithAnalysis] 开始数据分析，analysis_type: %s", analysisType)

	analysisArgs := map[string]interface{}{
		"data":          queryResult.Content[0].Text,
		"analysis_type": analysisType,
	}
	if provider, ok := arguments["provider"]; ok {
		analysisArgs["provider"] = provider
	}
	if model, ok := arguments["model"]; ok {
		analysisArgs["model"] = model
	}

	log.Printf("[QueryWithAnalysis] 准备调用AI分析，数据长度: %d", len(queryResult.Content[0].Text))
	log.Printf("[QueryWithAnalysis] 查询结果内容: %s", queryResult.Content[0].Text)

	// 解析查询结果以获取实际数据
	var queryDataForAnalysis map[string]interface{}
	actualData := ""
	if err := json.Unmarshal([]byte(queryResult.Content[0].Text), &queryDataForAnalysis); err == nil {
		// 优先提取 raw_result.rows 数据
		if rawResult, ok := queryDataForAnalysis["raw_result"].(map[string]interface{}); ok {
			if rows, ok := rawResult["rows"].([]interface{}); ok {
				if jsonBytes, err := json.Marshal(rows); err == nil {
					actualData = string(jsonBytes)
				}
			}
		} else if data, ok := queryDataForAnalysis["data"].(map[string]interface{}); ok {
			if rows, ok := data["rows"].([]interface{}); ok {
				if jsonBytes, err := json.Marshal(rows); err == nil {
					actualData = string(jsonBytes)
				}
			}
		} else if rows, ok := queryDataForAnalysis["rows"].([]interface{}); ok {
			if jsonBytes, err := json.Marshal(rows); err == nil {
				actualData = string(jsonBytes)
			}
		}

		// 如果还是没有提取到数据，使用原始结果
		if actualData == "" {
			actualData = queryResult.Content[0].Text
		}
	} else {
		actualData = queryResult.Content[0].Text
	}

	log.Printf("[QueryWithAnalysis] 提取的实际数据: %s", actualData)

	// 更新分析参数使用提取的数据
	analysisArgs["data"] = actualData

	// 使用改进的分析方法，包含中文提示
	analysisResult, err := c.executeAIAnalyzeDataWithChinesePrompt(ctx, analysisArgs)
	if err != nil {
		log.Printf("[QueryWithAnalysis] 数据分析失败: %v", err)
		return nil, fmt.Errorf("数据分析失败: %v", err)
	}

	log.Printf("[QueryWithAnalysis] 数据分析完成")

	// 组合结果
	var queryData, analysisData map[string]interface{}
	json.Unmarshal([]byte(queryResult.Content[0].Text), &queryData)
	json.Unmarshal([]byte(analysisResult.Content[0].Text), &analysisData)

	response := map[string]interface{}{
		"tool":          "ai_query_with_analysis",
		"status":        "success",
		"description":   description,
		"analysis_type": analysisType,
		"query_result":  queryData,
		"analysis":      analysisData,
	}

	jsonResponse, _ := json.MarshalIndent(response, "", "  ")
	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: string(jsonResponse),
			},
		},
	}, nil
}

// 7. 智能洞察 - 深度智能分析，提供业务洞察和建议
func (c *AITools) executeAISmartInsights(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	prompt, ok := arguments["prompt"].(string)
	if !ok {
		return nil, fmt.Errorf("prompt参数必须是字符串")
	}

	// 获取AI提供商
	provider, model, err := c.getProviderAndModel(arguments)
	if err != nil {
		return nil, err
	}

	// 获取参数
	context := ""
	if ctx, ok := arguments["context"].(string); ok {
		context = ctx
	}

	insightLevel := "basic"
	if il, ok := arguments["insight_level"].(string); ok {
		insightLevel = il
	}

	// 获取表名 - 使用智能表名检测或默认表名
	var tableName string
	if tn, ok := arguments["table_name"].(string); ok && tn != "" {
		tableName = tn
	} else {
		// 使用智能表名检测
		defaultTable := c.getDefaultTableName(ctx)
		detectedTable, err := c.intelligentTableDetection(ctx, prompt, defaultTable)
		if err != nil {
			debugPrintAICompatible("[DEBUG] 智能表名检测失败: %v，使用默认表名: %s\n", err, defaultTable)
			tableName = defaultTable
		} else {
			tableName = detectedTable
			debugPrintAICompatible("[DEBUG] 智能表名检测成功，使用表名: %s\n", tableName)
		}
	}

	// 第一步：使用智能查询获取相关数据
	dataQuery := fmt.Sprintf("查询%s表中与以下分析需求相关的数据：%s", tableName, prompt)
	queryArgs := map[string]interface{}{
		"prompt":   dataQuery,
		"provider": provider.Name(),
		"model":    model,
	}

	log.Printf("[AISmartInsights] 开始查询数据，查询需求：%s", dataQuery)

	queryResult, err := c.executeAISmartQuery(ctx, queryArgs)
	if err != nil {
		return nil, fmt.Errorf("数据查询失败: %v", err)
	}

	log.Printf("[AISmartInsights] 数据查询完成")

	// 解析查询结果以获取实际数据
	var queryData map[string]interface{}
	actualData := ""
	if err := json.Unmarshal([]byte(queryResult.Content[0].Text), &queryData); err == nil {
		if queryResultData, ok := queryData["query_result"].(string); ok {
			actualData = queryResultData
		} else if data, ok := queryData["data"].(string); ok {
			actualData = data
		} else {
			actualData = queryResult.Content[0].Text
		}
	} else {
		actualData = queryResult.Content[0].Text
	}

	log.Printf("[AISmartInsights] 准备进行%s级别的分析", insightLevel)

	// 第二步：深度分析
	var analysisPrompt string
	switch insightLevel {
	case "strategic":
		analysisPrompt = fmt.Sprintf(`作为高级业务分析师，请基于以下数据和需求进行战略级分析：

分析需求：%s
上下文信息：%s
相关数据：%s

请提供战略级分析：
1. 关键业务指标洞察
2. 市场趋势分析
3. 竞争优势评估
4. 风险识别与管控
5. 战略建议与路线图
6. ROI预期分析

请用中文回答，提供具体可执行的建议。`, prompt, context, actualData)

	case "advanced":
		analysisPrompt = fmt.Sprintf(`作为业务分析专家，请基于以下数据和需求进行深度分析：

分析需求：%s
上下文信息：%s
相关数据：%s

请提供深度分析：
1. 数据模式识别
2. 趋势预测
3. 异常检测
4. 相关性分析
5. 改进建议
6. 预期效果评估

请用中文回答，提供具体的改进方案。`, prompt, context, actualData)

	default: // basic
		analysisPrompt = fmt.Sprintf(`请基于以下数据和需求进行基础分析：

分析需求：%s
上下文信息：%s
相关数据：%s

请提供基础分析：
1. 数据概况总结
2. 主要发现
3. 基本建议

请用中文回答。`, prompt, context, actualData)
	}

	log.Printf("[AISmartInsights] 开始AI分析")

	// 调用AI进行深度分析
	insights, err := provider.Call(ctx, model, analysisPrompt, map[string]interface{}{
		"max_tokens":  2000,
		"temperature": 0.6,
	})
	if err != nil {
		return nil, fmt.Errorf("智能洞察分析失败: %v", err)
	}

	log.Printf("[AISmartInsights] AI分析完成")

	// 清理AI响应格式
	cleanedInsights := cleanAIResponse(insights)

	// 构建响应
	response := map[string]interface{}{
		"tool":          "ai_smart_insights",
		"status":        "success",
		"prompt":        prompt,
		"insight_level": insightLevel,
		"table_name":    tableName,
		"context":       context,
		"provider":      provider.Name(),
		"model":         model,
		"query_data":    actualData,
		"insights":      cleanedInsights,
	}

	jsonResponse, _ := json.MarshalIndent(response, "", "  ")
	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: string(jsonResponse),
			},
		},
	}, nil
}

// 智能表名识别和获取功能
func (c *AITools) intelligentTableDetection(ctx context.Context, prompt string, defaultTable string) (string, error) {
	debugPrintAICompatible("[DEBUG] ====== 智能表名识别开始 ======\n")
	debugPrintAICompatible("[DEBUG] 输入prompt: '%s', 默认表名: '%s'\n", prompt, defaultTable)

	// 首先获取数据库中的所有表
	availableTables, err := c.getAvailableTables(ctx)
	if err != nil {
		debugPrintAICompatible("[DEBUG] 获取表列表失败，使用默认表名: %v\n", err)
		return defaultTable, nil
	}

	debugPrintAICompatible("[DEBUG] 可用表列表: %v\n", availableTables)

	// 如果只有一个表，直接使用
	if len(availableTables) == 1 {
		debugPrintAICompatible("[DEBUG] 只有一个表，直接使用: %s\n", availableTables[0])
		return availableTables[0], nil
	}

	// 尝试从自然语言中识别表名关键词
	detectedTable := c.extractTableFromPrompt(prompt, availableTables)
	if detectedTable != "" {
		debugPrintAICompatible("[DEBUG] 从自然语言中识别到表名: %s\n", detectedTable)
		return detectedTable, nil
	}

	// 如果无法识别，使用AI来智能匹配
	aiMatchedTable, err := c.aiMatchTable(ctx, prompt, availableTables)
	if err == nil && aiMatchedTable != "" {
		debugPrintAICompatible("[DEBUG] AI匹配到表名: %s\n", aiMatchedTable)
		return aiMatchedTable, nil
	}

	// 最后回退到默认表名
	debugPrintAICompatible("[DEBUG] 无法智能识别，使用默认表名: %s\n", defaultTable)
	return defaultTable, nil
}

// 获取数据库中的所有表
func (c *AITools) getAvailableTables(ctx context.Context) ([]string, error) {
	if c.databaseTools == nil {
		return nil, fmt.Errorf("数据库工具不可用")
	}

	// 执行 SHOW TABLES 查询
	dbArgs := map[string]interface{}{
		"sql":   "SHOW TABLES",
		"alias": "mysql_test", // 使用默认数据库别名
		"limit": 100,
	}

	result, err := c.databaseTools.ExecuteTool(ctx, "db_query", dbArgs)
	if err != nil {
		return nil, fmt.Errorf("查询表列表失败: %v", err)
	}

	// 解析查询结果
	var dbResponse map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &dbResponse); err != nil {
		return nil, fmt.Errorf("解析表列表结果失败: %v", err)
	}

	// 提取表名
	var tables []string
	if rawResult, ok := dbResponse["raw_result"].(map[string]interface{}); ok {
		if rows, ok := rawResult["rows"].([]interface{}); ok {
			for _, row := range rows {
				if rowMap, ok := row.(map[string]interface{}); ok {
					// SHOW TABLES 返回的列名可能是 "Tables_in_database_name"
					for _, value := range rowMap {
						if tableName, ok := value.(string); ok {
							tables = append(tables, tableName)
						}
					}
				}
			}
		}
	}

	return tables, nil
}

// 从自然语言中提取表名关键词
func (c *AITools) extractTableFromPrompt(prompt string, availableTables []string) string {
	prompt = strings.ToLower(prompt)

	// 表名关键词映射
	tableKeywords := map[string][]string{
		"user":    {"用户", "员工", "人员", "user", "users", "employee", "staff"},
		"order":   {"订单", "购买", "交易", "order", "orders", "purchase"},
		"product": {"产品", "商品", "货物", "product", "products", "goods"},
		"log":     {"日志", "记录", "log", "logs", "record"},
		"config":  {"配置", "设置", "config", "configuration", "setting"},
	}

	// 遍历可用表，看是否能匹配到关键词
	for _, table := range availableTables {
		tableNameLower := strings.ToLower(table)

		// 直接包含表名
		if strings.Contains(prompt, tableNameLower) {
			return table
		}

		// 检查关键词映射
		for baseTable, keywords := range tableKeywords {
			if strings.Contains(tableNameLower, baseTable) {
				for _, keyword := range keywords {
					if strings.Contains(prompt, keyword) {
						return table
					}
				}
			}
		}
	}

	return ""
}

// 使用AI智能匹配表名
func (c *AITools) aiMatchTable(ctx context.Context, prompt string, availableTables []string) (string, error) {
	// 获取AI提供商
	provider, model, err := c.getProviderAndModel(map[string]interface{}{})
	if err != nil {
		return "", err
	}

	// 构建AI提示词
	tablesStr := strings.Join(availableTables, ", ")
	aiPrompt := fmt.Sprintf(`根据用户的查询需求，从可用的数据库表中选择最合适的表。

用户查询：%s

可用表名：%s

请只返回最合适的一个表名，不要包含其他解释。如果无法确定，返回第一个表名。`, prompt, tablesStr)

	debugPrintAICompatible("[DEBUG] AI表名匹配提示词: %s\n", aiPrompt)

	// 调用AI
	result, err := provider.Call(ctx, model, aiPrompt, map[string]interface{}{
		"temperature": 0.3,
	})
	if err != nil {
		return "", fmt.Errorf("AI表名匹配失败: %v", err)
	}

	// 提取AI返回的表名
	aiResponse := strings.TrimSpace(result)
	aiResponse = strings.ToLower(aiResponse)

	// 验证AI返回的表名是否在可用表列表中
	for _, table := range availableTables {
		if strings.ToLower(table) == aiResponse || strings.Contains(aiResponse, strings.ToLower(table)) {
			return table, nil
		}
	}

	return "", fmt.Errorf("AI返回的表名不在可用列表中")
}

// 获取默认表名（从数据库中的第一个可用表）
func (c *AITools) getDefaultTableName(ctx context.Context) string {
	availableTables, err := c.getAvailableTables(ctx)
	if err != nil || len(availableTables) == 0 {
		return "users" // 最后的回退值
	}
	return availableTables[0]
}

// executeAIFileManager 执行AI文件管理
func (c *AITools) executeAIFileManager(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	instruction, ok := arguments["instruction"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少instruction参数")
	}

	targetPath := ""
	if path, exists := arguments["target_path"].(string); exists {
		targetPath = path
	}

	operationMode := "plan_only"
	if mode, exists := arguments["operation_mode"].(string); exists {
		operationMode = mode
	}

	provider, model, err := c.getProviderAndModel(arguments)
	if err != nil {
		return nil, fmt.Errorf("获取AI提供商失败: %v", err)
	}

	// 构建AI提示，让AI理解文件操作需求
	aiPrompt := fmt.Sprintf(`你是一个智能文件管理助手。用户的指令是："%s"

当前目标路径：%s

请分析这个指令并：
1. 理解用户想要进行的文件操作类型
2. 确定具体需要执行的操作步骤
3. 如果需要，生成相应的文件操作命令

操作模式：%s
- 如果是 plan_only，只输出分析和计划，不执行实际操作
- 如果是 execute，除了分析还要说明具体执行的操作

请用JSON格式回复，包含以下字段：
{
  "analysis": "对指令的分析",
  "operation_type": "操作类型(read/write/create/delete/list/search等)",
  "action_plan": ["具体的操作步骤"],
  "commands": ["如果需要执行，具体的命令或操作"],
  "warnings": ["任何安全警告或注意事项"]
}`, instruction, targetPath, operationMode)

	result, err := provider.Call(ctx, model, aiPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("AI文件管理分析失败: %v", err)
	}

	// 如果是execute模式且有systemTools，尝试执行文件操作
	var executionResults []string
	if operationMode == "execute" && c.systemTools != nil {
		instructionLower := strings.ToLower(instruction)

		if strings.Contains(instructionLower, "创建") || strings.Contains(instructionLower, "新建") {
			if targetPath != "" {
				// 首先确保目标目录存在 - 使用Go的os.MkdirAll，更简洁可靠
				err := os.MkdirAll(targetPath, 0755)
				if err != nil {
					executionResults = append(executionResults, fmt.Sprintf("创建目录失败: %v", err))
				} else {
					executionResults = append(executionResults, "✅ 目录创建成功")

					// 根据指令内容判断创建类型并创建相应文件
					if strings.Contains(instructionLower, "go") && strings.Contains(instructionLower, "项目") {
						// 创建Go项目文件
						files := map[string]string{
							"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, Go!")
}`,
							"go.mod": `module example

go 1.19`,
							"README.md": "# Go Project\n\nThis is a Go project.",
						}

						for filename, content := range files {
							filePath := filepath.Join(targetPath, filename)
							writeArgs := map[string]interface{}{
								"path":    filePath,
								"content": content,
							}
							_, err := c.systemTools.ExecuteTool(ctx, "file_write", writeArgs)
							if err != nil {
								executionResults = append(executionResults, fmt.Sprintf("创建文件 %s 失败: %v", filename, err))
							} else {
								executionResults = append(executionResults, fmt.Sprintf("✅ 文件 %s 创建成功", filename))
							}
						}
					} else if strings.Contains(instructionLower, "nodejs") || strings.Contains(instructionLower, "node.js") {
						// 创建Node.js项目文件
						packageJSON := `{
  "name": "nodejs-project",
  "version": "1.0.0",
  "description": "A Node.js project",
  "main": "index.js",
  "scripts": {
    "start": "node index.js",
    "test": "echo \"Error: no test specified\" && exit 1"
  },
  "dependencies": {},
  "devDependencies": {}
}`
						indexJS := `console.log('Hello, Node.js!');`
						readme := "# Node.js Project\n\nThis is a Node.js project."

						files := map[string]string{
							"package.json": packageJSON,
							"index.js":     indexJS,
							"README.md":    readme,
						}

						for filename, content := range files {
							filePath := filepath.Join(targetPath, filename)
							writeArgs := map[string]interface{}{
								"path":    filePath,
								"content": content,
							}
							_, err := c.systemTools.ExecuteTool(ctx, "file_write", writeArgs)
							if err != nil {
								executionResults = append(executionResults, fmt.Sprintf("创建文件 %s 失败: %v", filename, err))
							} else {
								executionResults = append(executionResults, fmt.Sprintf("✅ 文件 %s 创建成功", filename))
							}
						}
					} else if strings.Contains(instructionLower, "文档") || strings.Contains(instructionLower, "docs") {
						// 创建文档项目文件
						files := map[string]string{
							"README.md":     "# 文档项目\n\n这是一个文档项目。",
							"docs/index.md": "# 首页\n\n欢迎来到文档站点。",
							"docs/guide.md": "# 使用指南\n\n这里是使用指南。",
							"docs/api.md":   "# API 文档\n\n这里是API文档。",
						}

						// 创建docs子目录
						docsDir := filepath.Join(targetPath, "docs")
						mkdirArgs := map[string]interface{}{
							"command": "mkdir",
							"args":    []string{"-p", docsDir},
						}
						c.systemTools.ExecuteTool(ctx, "command_execute", mkdirArgs)

						for filename, content := range files {
							filePath := filepath.Join(targetPath, filename)
							writeArgs := map[string]interface{}{
								"path":    filePath,
								"content": content,
							}
							_, err := c.systemTools.ExecuteTool(ctx, "file_write", writeArgs)
							if err != nil {
								executionResults = append(executionResults, fmt.Sprintf("创建文件 %s 失败: %v", filename, err))
							} else {
								executionResults = append(executionResults, fmt.Sprintf("✅ 文件 %s 创建成功", filename))
							}
						}
					} else if strings.Contains(instructionLower, "json") || strings.Contains(instructionLower, "配置") {
						// 创建JSON配置文件
						configContent := `{
  "name": "project-config",
  "version": "1.0.0",
  "environment": "development",
  "database": {
    "host": "localhost",
    "port": 3306,
    "name": "mydb"
  },
  "server": {
    "host": "localhost",
    "port": 8080
  },
  "features": {
    "debug": true,
    "cache": false
  }
}`

						filename := "config.json"
						if strings.Contains(instructionLower, "package") {
							filename = "package.json"
							configContent = `{
  "name": "my-project",
  "version": "1.0.0",
  "description": "My project description",
  "main": "index.js",
  "scripts": {
    "start": "node index.js"
  }
}`
						}

						filePath := filepath.Join(targetPath, filename)
						writeArgs := map[string]interface{}{
							"path":    filePath,
							"content": configContent,
						}
						_, err := c.systemTools.ExecuteTool(ctx, "file_write", writeArgs)
						if err != nil {
							executionResults = append(executionResults, fmt.Sprintf("创建配置文件失败: %v", err))
						} else {
							executionResults = append(executionResults, fmt.Sprintf("✅ 配置文件 %s 创建成功", filename))
						}
					} else {
						// 创建默认文件
						defaultContent := fmt.Sprintf("# 文件\n\n创建时间: %s\n指令: %s\n",
							time.Now().Format("2006-01-02 15:04:05"), instruction)

						filename := "README.md"
						if strings.Contains(instructionLower, ".txt") {
							filename = "file.txt"
							defaultContent = fmt.Sprintf("文件创建时间: %s\n指令: %s\n",
								time.Now().Format("2006-01-02 15:04:05"), instruction)
						}

						filePath := filepath.Join(targetPath, filename)
						writeArgs := map[string]interface{}{
							"path":    filePath,
							"content": defaultContent,
						}
						_, err := c.systemTools.ExecuteTool(ctx, "file_write", writeArgs)
						if err != nil {
							executionResults = append(executionResults, fmt.Sprintf("创建文件失败: %v", err))
						} else {
							executionResults = append(executionResults, fmt.Sprintf("✅ 文件 %s 创建成功", filename))
						}
					}
				}
			}
		} else if strings.Contains(instructionLower, "修改") || strings.Contains(instructionLower, "添加") || strings.Contains(instructionLower, "更新") {
			// 修改或添加文件操作
			if targetPath != "" {
				// 检查目标目录是否存在
				if _, err := os.Stat(targetPath); os.IsNotExist(err) {
					executionResults = append(executionResults, "⚠️ 目标目录不存在，请先创建目录")
				} else {
					executionResults = append(executionResults, "✅ 找到目标目录")

					// 根据指令内容判断要添加的文件类型
					if strings.Contains(instructionLower, "http") || strings.Contains(instructionLower, "服务器") || strings.Contains(instructionLower, "server") {
						// 添加HTTP服务器文件
						serverContent := `package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, HTTP Server!")
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "{\"status\": \"ok\", \"message\": \"Server is running\"}")
	})

	fmt.Println("HTTP Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}`

						filePath := filepath.Join(targetPath, "server.go")
						writeArgs := map[string]interface{}{
							"path":    filePath,
							"content": serverContent,
						}
						_, err := c.systemTools.ExecuteTool(ctx, "file_write", writeArgs)
						if err != nil {
							executionResults = append(executionResults, fmt.Sprintf("添加服务器文件失败: %v", err))
						} else {
							executionResults = append(executionResults, "✅ HTTP服务器文件 server.go 添加成功")
						}
					}

					if strings.Contains(instructionLower, "配置") || strings.Contains(instructionLower, "config") {
						// 添加配置文件
						configContent := `# 应用配置
server:
  host: localhost
  port: 8080

database:
  driver: mysql
  host: localhost
  port: 3306
  name: myapp

logging:
  level: info
  file: app.log

features:
  debug: true
  cache: false`

						filePath := filepath.Join(targetPath, "config.yaml")
						writeArgs := map[string]interface{}{
							"path":    filePath,
							"content": configContent,
						}
						_, err := c.systemTools.ExecuteTool(ctx, "file_write", writeArgs)
						if err != nil {
							executionResults = append(executionResults, fmt.Sprintf("添加配置文件失败: %v", err))
						} else {
							executionResults = append(executionResults, "✅ 配置文件 config.yaml 添加成功")
						}
					}
				}
			}
		} else if strings.Contains(instructionLower, "查找") || strings.Contains(instructionLower, "列出") {
			// 文件查找操作
			listArgs := map[string]interface{}{
				"path": targetPath,
			}
			_, err := c.systemTools.ExecuteTool(ctx, "directory_list", listArgs)
			if err != nil {
				executionResults = append(executionResults, fmt.Sprintf("列出目录失败: %v", err))
			} else {
				executionResults = append(executionResults, "✅ 目录列表获取成功")
			}
		} else {
			executionResults = append(executionResults, "⚠️ 当前操作类型暂不支持自动执行，仅提供分析结果")
		}
	}

	response := map[string]interface{}{
		"ai_analysis":       result,
		"operation_mode":    operationMode,
		"target_path":       targetPath,
		"execution_results": executionResults,
	}

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: fmt.Sprintf("AI文件管理结果：\n%s", formatJSONResponse(response)),
			},
		},
	}, nil
}

// executeAIDataProcessor 执行AI数据处理
func (c *AITools) executeAIDataProcessor(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	instruction, ok := arguments["instruction"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少instruction参数")
	}

	inputData, ok := arguments["input_data"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少input_data参数")
	}

	dataType := "auto"
	if dt, exists := arguments["data_type"].(string); exists {
		dataType = dt
	}

	outputFormat := "json"
	if of, exists := arguments["output_format"].(string); exists {
		outputFormat = of
	}

	provider, model, err := c.getProviderAndModel(arguments)
	if err != nil {
		return nil, fmt.Errorf("获取AI提供商失败: %v", err)
	}

	// 构建AI提示，让AI理解数据处理需求
	aiPrompt := fmt.Sprintf(`你是一个智能数据处理助手。用户的指令是："%s"

输入数据：
%s

数据类型：%s
期望输出格式：%s

请分析数据并执行用户的指令：
1. 如果数据类型是auto，请先识别实际数据类型
2. 根据指令处理数据
3. 按照期望格式输出结果

请用JSON格式回复，包含以下字段：
{
  "detected_type": "识别的数据类型",
  "processing_steps": ["处理步骤"],
  "result": "处理后的数据",
  "analysis": "数据分析结果",
  "errors": ["任何错误或警告"]
}`, instruction, inputData, dataType, outputFormat)

	result, err := provider.Call(ctx, model, aiPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("AI数据处理失败: %v", err)
	}

	// 如果有dataTools，尝试执行一些基础的数据处理操作
	var processingResults map[string]interface{}
	operationMode := "plan_only"
	if mode, exists := arguments["operation_mode"].(string); exists {
		operationMode = mode
	}

	if operationMode == "execute" && c.dataTools != nil {
		// 实际执行数据处理
		processingResults = make(map[string]interface{})

		// 尝试解析JSON数据并根据指令处理
		if dataType == "json" || strings.Contains(strings.ToLower(inputData), "{") {
			var jsonData interface{}
			err := json.Unmarshal([]byte(inputData), &jsonData)
			if err != nil {
				processingResults["error"] = fmt.Sprintf("JSON解析失败: %v", err)
			} else {
				processingResults["parsed_json"] = true

				// 根据指令类型进行特定处理
				instructionLower := strings.ToLower(instruction)
				if strings.Contains(instructionLower, "邮箱") || strings.Contains(instructionLower, "email") {
					emails := extractEmailsFromJSON(jsonData)
					processingResults["extracted_emails"] = emails
					processingResults["email_count"] = len(emails)

					// 根据输出格式格式化结果
					if outputFormat == "table" {
						tableResult := "邮箱地址列表:\n"
						tableResult += "序号 | 邮箱地址\n"
						tableResult += "-----|----------\n"
						for i, email := range emails {
							tableResult += fmt.Sprintf("%d    | %s\n", i+1, email)
						}
						processingResults["formatted_output"] = tableResult
					} else {
						processingResults["formatted_output"] = emails
					}
				} else if strings.Contains(instructionLower, "用户") || strings.Contains(instructionLower, "user") {
					users := extractUsersFromJSON(jsonData)
					processingResults["extracted_users"] = users
					processingResults["user_count"] = len(users)

					if outputFormat == "table" {
						tableResult := "用户信息列表:\n"
						tableResult += "姓名 | 邮箱 | 年龄\n"
						tableResult += "-----|------|-----\n"
						for _, user := range users {
							tableResult += fmt.Sprintf("%s | %s | %v\n",
								getFieldFromMap(user, "name"),
								getFieldFromMap(user, "email"),
								getFieldFromMap(user, "age"))
						}
						processingResults["formatted_output"] = tableResult
					} else {
						processingResults["formatted_output"] = users
					}
				}
			}
		} else if dataType == "csv" {
			// 处理CSV数据
			lines := strings.Split(inputData, "\n")
			if len(lines) > 1 {
				headers := strings.Split(lines[0], ",")
				var csvData []map[string]string

				for i := 1; i < len(lines); i++ {
					if strings.TrimSpace(lines[i]) == "" {
						continue
					}
					values := strings.Split(lines[i], ",")
					row := make(map[string]string)
					for j, header := range headers {
						if j < len(values) {
							row[strings.TrimSpace(header)] = strings.TrimSpace(values[j])
						}
					}
					csvData = append(csvData, row)
				}

				processingResults["parsed_csv"] = true
				processingResults["csv_data"] = csvData
				processingResults["row_count"] = len(csvData)

				if outputFormat == "json" {
					jsonBytes, _ := json.MarshalIndent(csvData, "", "  ")
					processingResults["formatted_output"] = string(jsonBytes)
				}
			}
		}

		processingResults["execution_mode"] = "实际执行"
		processingResults["status"] = "数据处理完成"
	} else {
		// 基础数据验证和处理
		processingResults = map[string]interface{}{
			"validation_attempted": true,
			"execution_mode":       "仅规划模式",
			"note":                 "设置operation_mode为execute以执行实际数据处理",
		}
	}

	response := map[string]interface{}{
		"ai_analysis":        result,
		"data_type":          dataType,
		"output_format":      outputFormat,
		"processing_results": processingResults,
	}

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: fmt.Sprintf("AI数据处理结果：\n%s", formatJSONResponse(response)),
			},
		},
	}, nil
}

// executeAIAPIClient 执行AI网络请求
func (c *AITools) executeAIAPIClient(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	instruction, ok := arguments["instruction"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少instruction参数")
	}

	baseURL := ""
	if url, exists := arguments["base_url"].(string); exists {
		baseURL = url
	}

	authInfo := ""
	if auth, exists := arguments["auth_info"].(string); exists {
		authInfo = auth
	}

	requestMode := "plan_only"
	if mode, exists := arguments["request_mode"].(string); exists {
		requestMode = mode
	}

	responseAnalysis := true
	if ra, exists := arguments["response_analysis"].(bool); exists {
		responseAnalysis = ra
	}

	provider, model, err := c.getProviderAndModel(arguments)
	if err != nil {
		return nil, fmt.Errorf("获取AI提供商失败: %v", err)
	}

	// 构建AI提示，让AI理解API调用需求
	aiPrompt := fmt.Sprintf(`你是一个智能API客户端助手。用户的指令是："%s"

基础URL：%s
认证信息：%s
请求模式：%s

请分析这个指令并：
1. 理解用户想要调用的API类型和目的
2. 确定HTTP方法(GET/POST/PUT/DELETE等)
3. 构造请求URL、头部和数据
4. 评估请求的安全性和有效性

请用JSON格式回复，包含以下字段：
{
  "analysis": "对API调用指令的分析",
  "http_method": "HTTP方法",
  "full_url": "完整的请求URL",
  "headers": {"建议的请求头"},
  "body": "请求体数据(如果需要)",
  "security_notes": ["安全注意事项"],
  "expected_response": "预期的响应格式"
}`, instruction, baseURL, authInfo, requestMode)

	result, err := provider.Call(ctx, model, aiPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("AI API分析失败: %v", err)
	}

	// 如果是execute模式且有networkTools，尝试执行请求
	var executionResults map[string]interface{}
	if requestMode == "execute" && c.networkTools != nil {
		// 尝试从AI分析结果中解析API调用信息
		executionResults = make(map[string]interface{})
		executionResults["execution_attempted"] = true

		// 简化URL构造逻辑，使用最可靠的端点
		instructionLower := strings.ToLower(instruction)
		var requestURL string

		if strings.Contains(instructionLower, "httpbin") || baseURL == "https://httpbin.org" {
			// httpbin.org - 使用最简单的get端点
			requestURL = baseURL + "/get"
		} else if strings.Contains(instructionLower, "jsonplaceholder") || baseURL == "https://jsonplaceholder.typicode.com" {
			// JSONPlaceholder - 获取用户数据（限制数量）
			requestURL = baseURL + "/users?_limit=3"
		} else {
			// 默认情况：尝试基础URL或添加常见端点
			if strings.HasSuffix(baseURL, "/") {
				requestURL = strings.TrimSuffix(baseURL, "/")
			} else {
				requestURL = baseURL
			}
		} // 执行实际的HTTP请求
		if requestURL != "" {
			httpArgs := map[string]interface{}{
				"url": requestURL,
				"headers": map[string]interface{}{
					"User-Agent": "MCP-AI-Client/1.0",
					"Accept":     "application/json",
				},
				"timeout": 30,
			}

			httpResult, err := c.networkTools.ExecuteTool(ctx, "http_get", httpArgs)
			if err != nil {
				executionResults["error"] = fmt.Sprintf("HTTP请求失败: %v", err)
				executionResults["success"] = false
			} else {
				executionResults["success"] = true
				executionResults["http_response"] = httpResult.Content
				executionResults["url"] = requestURL
			}
		} else {
			executionResults["error"] = "无法确定请求URL"
			executionResults["success"] = false
		}
	} else {
		executionResults = map[string]interface{}{
			"execution_attempted": false,
			"note":                "需要execute模式和网络工具支持才能执行实际请求",
		}
	}

	response := map[string]interface{}{
		"ai_analysis":               result,
		"request_mode":              requestMode,
		"response_analysis_enabled": responseAnalysis,
		"execution_results":         executionResults,
	}

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: fmt.Sprintf("AI API客户端结果：\n%s", formatJSONResponse(response)),
			},
		},
	}, nil
}

// executeAISystemAdmin 执行AI系统管理
func (c *AITools) executeAISystemAdmin(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	instruction, ok := arguments["instruction"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少instruction参数")
	}

	safetyMode := "safe"
	if sm, exists := arguments["safety_mode"].(string); exists {
		safetyMode = sm
	}

	operationMode := "plan_only"
	if om, exists := arguments["operation_mode"].(string); exists {
		operationMode = om
	}

	provider, model, err := c.getProviderAndModel(arguments)
	if err != nil {
		return nil, fmt.Errorf("获取AI提供商失败: %v", err)
	}

	// 构建AI提示，让AI理解系统管理需求
	aiPrompt := fmt.Sprintf(`你是一个智能系统管理助手。用户的指令是："%s"

安全模式：%s
- safe: 只进行安全的只读操作
- moderate: 允许中等风险的操作
- advanced: 允许高级系统操作(需要特别谨慎)

操作模式：%s

请分析这个指令并：
1. 理解用户想要进行的系统管理操作
2. 评估操作的风险级别
3. 确定具体的执行步骤
4. 提供安全建议

请用JSON格式回复，包含以下字段：
{
  "analysis": "对系统管理指令的分析",
  "risk_level": "风险级别(low/medium/high/critical)",
  "operation_category": "操作类别(monitor/maintenance/config/security等)",
  "action_plan": ["具体的操作步骤"],
  "commands": ["建议的系统命令"],
  "safety_warnings": ["安全警告"],
  "prerequisites": ["执行前置条件"]
}`, instruction, safetyMode, operationMode)

	result, err := provider.Call(ctx, model, aiPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("AI系统管理分析失败: %v", err)
	}

	// 如果是execute模式且有systemTools，尝试执行一些安全的操作
	var executionResults map[string]interface{}
	if operationMode == "execute" && c.systemTools != nil && safetyMode == "safe" {
		executionResults = map[string]interface{}{
			"execution_attempted": true,
			"safety_mode":         safetyMode,
			"note":                "仅在安全模式下执行只读操作",
		}
	}

	response := map[string]interface{}{
		"ai_analysis":       result,
		"safety_mode":       safetyMode,
		"operation_mode":    operationMode,
		"execution_results": executionResults,
	}

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: fmt.Sprintf("AI系统管理结果：\n%s", formatJSONResponse(response)),
			},
		},
	}, nil
}

// formatJSONResponse 格式化JSON响应
func formatJSONResponse(data interface{}) string {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("格式化错误: %v", err)
	}
	return string(jsonBytes)
}

// extractEmailsFromJSON 从JSON数据中提取邮箱地址
func extractEmailsFromJSON(data interface{}) []string {
	var emails []string

	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if strings.Contains(strings.ToLower(key), "email") {
				if email, ok := value.(string); ok && isValidEmail(email) {
					emails = append(emails, email)
				}
			} else {
				emails = append(emails, extractEmailsFromJSON(value)...)
			}
		}
	case []interface{}:
		for _, item := range v {
			emails = append(emails, extractEmailsFromJSON(item)...)
		}
	case string:
		if isValidEmail(v) {
			emails = append(emails, v)
		}
	}

	return emails
}

// extractUsersFromJSON 从JSON数据中提取用户信息
func extractUsersFromJSON(data interface{}) []map[string]interface{} {
	var users []map[string]interface{}

	switch v := data.(type) {
	case map[string]interface{}:
		// 检查是否是单个用户对象
		if hasUserFields(v) {
			users = append(users, v)
		} else {
			// 递归查找用户数组
			for _, value := range v {
				users = append(users, extractUsersFromJSON(value)...)
			}
		}
	case []interface{}:
		for _, item := range v {
			if userMap, ok := item.(map[string]interface{}); ok && hasUserFields(userMap) {
				users = append(users, userMap)
			} else {
				users = append(users, extractUsersFromJSON(item)...)
			}
		}
	}

	return users
}

// hasUserFields 检查对象是否包含用户字段
func hasUserFields(obj map[string]interface{}) bool {
	userFields := []string{"name", "email", "user", "username", "id"}
	for _, field := range userFields {
		if _, exists := obj[field]; exists {
			return true
		}
	}
	return false
}

// getFieldFromMap 从map中获取字段值
func getFieldFromMap(data map[string]interface{}, field string) string {
	if value, exists := data[field]; exists {
		return fmt.Sprintf("%v", value)
	}
	return ""
}

// isValidEmail 简单的邮箱格式验证
func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}
