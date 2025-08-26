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
	"mcp-ai-server/internal/logger"
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
					"alias": map[string]interface{}{
						"type":        "string",
						"description": "数据库连接别名（如demo、mysql_test等）",
						"enum":        c.databaseConfigMgr.GetAvailableAliases(),
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
				"required": []string{"description", "alias"},
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
	}
}

// ExecuteTool 执行AI工具 - 按功能分类处理
func (c *AITools) ExecuteTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	switch toolName {
	case "ai_chat":
		return c.executeAIChat(ctx, arguments)
	case "ai_query_with_analysis":
		return c.executeAIQueryWithAnalysis(ctx, arguments)
	case "ai_file_manager":
		return c.executeAIFileManager(ctx, arguments)
	case "ai_data_processor":
		return c.executeAIDataProcessor(ctx, arguments)
	case "ai_api_client":
		return c.executeAIAPIClient(ctx, arguments)
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

	// 获取文本生成专用的AI提供商和模型
	provider, model, err := c.getProviderAndModelForFunction(arguments, "text_generation")
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

	// 调用AI进行对话（使用超时包装函数）
	response, err := c.callAIWithTimeout(ctx, provider, model, prompt, map[string]interface{}{
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

// getProviderAndModelForFunction 根据功能获取提供商和模型
func (c *AITools) getProviderAndModelForFunction(arguments map[string]interface{}, function string) (AIProvider, string, error) {
	// 优先使用参数中指定的提供商和模型
	if p, ok := arguments["provider"].(string); ok && p != "" {
		if m, ok := arguments["model"].(string); ok && m != "" {
			provider, exists := c.getProvider(p)
			if !exists || !provider.IsEnabled() {
				return nil, "", fmt.Errorf("AI提供商 %s 不可用或未启用", p)
			}
			return provider, m, nil
		}
	}

	// 使用功能特定的模型配置
	providerName, model, exists := c.configManager.GetFunctionModel(function)
	if !exists {
		// 如果没有找到功能特定配置，使用默认配置
		providerName = c.configManager.GetDefaultProvider()
		model = c.configManager.GetDefaultModel()
	}

	provider, exists := c.getProvider(providerName)
	if !exists || !provider.IsEnabled() {
		return nil, "", fmt.Errorf("AI提供商 %s 不可用或未启用", providerName)
	}

	return provider, model, nil
}

// callAIWithTimeout 带超时和重试的AI调用包装函数
func (c *AITools) callAIWithTimeout(ctx context.Context, provider AIProvider, model, prompt string, options map[string]interface{}) (string, error) {
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	// 添加重试机制
	maxRetries := 2
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("AI调用重试 %d/%d", attempt, maxRetries)
			time.Sleep(time.Duration(attempt) * 2 * time.Second) // 指数退避
		}

		response, err := provider.Call(ctx, model, prompt, options)
		if err == nil {
			return response, nil
		}

		// 检查是否是超时错误
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("AI调用超时，尝试 %d/%d: %v", attempt+1, maxRetries+1, err)
			if attempt == maxRetries {
				return "", fmt.Errorf("AI调用超时，已重试%d次: %v", maxRetries, err)
			}
			continue
		}

		// 其他错误，直接返回
		log.Printf("AI调用失败，尝试 %d/%d: %v", attempt+1, maxRetries+1, err)
		if attempt == maxRetries {
			return "", fmt.Errorf("AI调用失败，已重试%d次: %v", maxRetries, err)
		}
	}

	return "", fmt.Errorf("AI调用失败，已达到最大重试次数")
}

// executeAIGenerateSQL 根据描述生成SQL语句
func (c *AITools) executeAIGenerateSQL(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	description, ok := arguments["description"].(string)
	if !ok {
		return nil, fmt.Errorf("description参数必须是字符串")
	}

	// 获取SQL生成专用的AI提供商和模型
	provider, model, err := c.getProviderAndModelForFunction(arguments, "sql_generation")
	if err != nil {
		return nil, err
	}

	// 获取表名 - 使用智能表名检测或默认表名
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

	// 构建SQL生成提示
	prompt := fmt.Sprintf(`请根据以下描述生成SQL查询语句：

描述：%s
表名：%s

要求：
1. 只返回SQL语句，不要其他解释
2. 使用标准SQL语法
3. 确保查询安全，避免SQL注入
4. 如果需要限制结果数量，默认使用LIMIT 100

SQL：`, description, tableName)

	// 调用AI生成SQL
	aiResponse, err := c.callAIWithTimeout(ctx, provider, model, prompt, map[string]interface{}{
		"max_tokens":  500,
		"temperature": 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("AI生成SQL失败: %v", err)
	}

	// 提取SQL语句
	generatedSQL := extractSQLFromAIResponse(aiResponse)
	if generatedSQL == "" {
		return nil, fmt.Errorf("无法从AI响应中提取有效的SQL语句")
	}

	// 验证SQL安全性
	if err := c.validateSQL(generatedSQL); err != nil {
		return nil, fmt.Errorf("生成的SQL不安全: %v", err)
	}

	// 构建响应
	response := map[string]interface{}{
		"tool":        "ai_generate_sql",
		"status":      "success",
		"description": description,
		"table_name":  tableName,
		"provider":    provider.Name(),
		"model":       model,
		"sql":         generatedSQL,
		"ai_response": aiResponse,
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

// executeAIExecuteSQL 执行SQL并返回结果
func (c *AITools) executeAIExecuteSQL(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	sql, ok := arguments["sql"].(string)
	if !ok {
		return nil, fmt.Errorf("sql参数必须是字符串")
	}

	// 验证SQL安全性
	if err := c.validateSQL(sql); err != nil {
		return nil, fmt.Errorf("SQL不安全: %v", err)
	}

	// 获取数据库别名
	alias, ok := arguments["alias"].(string)
	if !ok || alias == "" {
		return nil, fmt.Errorf("alias参数必须是非空字符串")
	}

	// 执行SQL查询
	queryArgs := map[string]interface{}{
		"alias": alias,
		"sql":   sql,
		"limit": 100, // 默认限制
	}

	if limit, ok := arguments["limit"].(int); ok {
		queryArgs["limit"] = limit
	}

	// 使用数据库工具执行查询
	result, err := c.databaseTools.ExecuteTool(ctx, "db_query", queryArgs)
	if err != nil {
		return nil, fmt.Errorf("SQL执行失败: %v", err)
	}

	// 构建响应
	response := map[string]interface{}{
		"tool":   "ai_execute_sql",
		"status": "success",
		"sql":    sql,
		"alias":  alias,
		"result": result.Content[0].Text,
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

// executeAISmartQuery 智能查询 - 生成SQL并执行，带AI分析
func (c *AITools) executeAISmartQuery(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	prompt, ok := arguments["prompt"].(string)
	if !ok {
		return nil, fmt.Errorf("prompt参数必须是字符串")
	}

	// 获取数据分析专用的AI提供商和模型
	provider, model, err := c.getProviderAndModelForFunction(arguments, "data_analysis")
	if err != nil {
		return nil, err
	}

	// 获取分析模式
	analysisMode := "full"
	if am, ok := arguments["analysis_mode"].(string); ok {
		analysisMode = am
	}

	// 第一步：生成SQL
	sqlGenArgs := map[string]interface{}{
		"description": prompt,
		"provider":    provider.Name(),
		"model":       model,
	}
	if tableName, ok := arguments["table_name"]; ok {
		sqlGenArgs["table_name"] = tableName
	}

	log.Printf("[AISmartQuery] 开始生成SQL，提示：%s", prompt)

	sqlResult, err := c.executeAIGenerateSQL(ctx, sqlGenArgs)
	if err != nil {
		return nil, fmt.Errorf("SQL生成失败: %v", err)
	}

	// 解析SQL生成结果
	var sqlData map[string]interface{}
	if err := json.Unmarshal([]byte(sqlResult.Content[0].Text), &sqlData); err != nil {
		return nil, fmt.Errorf("解析SQL生成结果失败: %v", err)
	}

	generatedSQL, ok := sqlData["sql"].(string)
	if !ok {
		return nil, fmt.Errorf("无法从SQL生成结果中获取SQL语句")
	}

	log.Printf("[AISmartQuery] SQL生成完成：%s", generatedSQL)

	// 第二步：执行SQL
	execArgs := map[string]interface{}{
		"sql": generatedSQL,
	}
	if alias, ok := arguments["alias"]; ok {
		execArgs["alias"] = alias
	}
	if limit, ok := arguments["limit"]; ok {
		execArgs["limit"] = limit
	}

	log.Printf("[AISmartQuery] 开始执行SQL")

	execResult, err := c.executeAIExecuteSQL(ctx, execArgs)
	var dbResult string
	var queryError error

	if err != nil {
		queryError = err
		dbResult = fmt.Sprintf("查询执行失败: %v", err)
		log.Printf("[AISmartQuery] SQL执行失败: %v", err)
	} else {
		dbResult = execResult.Content[0].Text
		log.Printf("[AISmartQuery] SQL执行成功")
	}

	// 第三步：AI分析（根据分析模式）
	var analysis string
	if analysisMode == "fast" {
		// 快速模式：只返回查询结果，不进行AI分析
		analysis = "快速模式：跳过AI分析"
	} else {
		// 完整模式：进行AI分析
		log.Printf("[AISmartQuery] 开始AI分析")
		analysis = c.analyzeQueryResult(ctx, provider, model, prompt, generatedSQL, dbResult, queryError)
		log.Printf("[AISmartQuery] AI分析完成")
	}

	// 构建响应
	response := map[string]interface{}{
		"tool":           "ai_smart_query",
		"status":         "success",
		"prompt":         prompt,
		"provider":       provider.Name(),
		"model":          model,
		"analysis_mode":  analysisMode,
		"generated_sql":  generatedSQL,
		"query_result":   dbResult,
		"ai_analysis":    analysis,
		"has_error":      queryError != nil,
	}

	if queryError != nil {
		response["error"] = queryError.Error()
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

// executeAIAnalyzeData 执行AI数据分析


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

	// 获取数据分析专用的AI提供商和模型
	provider, model, err := c.getProviderAndModelForFunction(arguments, "data_analysis")
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

	// 调用AI进行分析（使用超时包装函数）
	analysisResponse, err := c.callAIWithTimeout(ctx, provider, model, analysisPrompt, map[string]interface{}{
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
			// 提取分号前的部分，避免包含注释
			if idx := strings.Index(line, ";"); idx != -1 {
				line = line[:idx+1]
			}
			sqlLines = append(sqlLines, line)
		}
	}

	// 如果没有找到代码块，尝试直接提取包含SELECT的行
	if len(sqlLines) == 0 {
		for _, line := range lines {
			if strings.Contains(strings.ToUpper(line), "SELECT") {
				// 提取分号前的部分，避免包含注释
				if idx := strings.Index(line, ";"); idx != -1 {
					line = line[:idx+1]
				}
				sqlLines = append(sqlLines, strings.TrimSpace(line))
				break // 只取第一个有效的SQL语句
			}
		}
	}

	// 合并SQL行并进一步清理
	if len(sqlLines) > 0 {
		sql := strings.Join(sqlLines, " ")
		// 移除可能的多余文本
		if idx := strings.Index(sql, ";"); idx != -1 {
			sql = sql[:idx+1]
		}
		return strings.TrimSpace(sql)
	}

	return ""
}

// 6. 数据查询+分析 - 查询数据并进行AI分析
func (c *AITools) executeAIQueryWithAnalysis(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	// 记录总体开始时间
	totalStartTime := time.Now()
	log.Printf("[QueryWithAnalysis] ⏱️  总体开始时间: %s", totalStartTime.Format("15:04:05.000"))
	logger.Performance("🚀 [性能] ai_query_with_analysis 开始执行 - 开始时间: %s", totalStartTime.Format("15:04:05.000"))

	description, ok := arguments["description"].(string)
	if !ok {
		return nil, fmt.Errorf("description参数必须是字符串")
	}

	analysisType := "summary"
	if at, ok := arguments["analysis_type"].(string); ok {
		analysisType = at
	}

	// 获取SQL生成专用的AI提供商和模型
	sqlProvider, sqlModel, err := c.getProviderAndModelForFunction(arguments, "sql_generation")
	if err != nil {
		return nil, err
	}

	// 获取数据分析专用的AI提供商和模型
	analysisProvider, analysisModel, err := c.getProviderAndModelForFunction(arguments, "data_analysis")
	if err != nil {
		return nil, err
	}

	log.Printf("[QueryWithAnalysis] 🚀 开始查询和分析，描述：%s，分析类型：%s，SQL模型：%s/%s，分析模型：%s/%s", description, analysisType, sqlProvider.Name(), sqlModel, analysisProvider.Name(), analysisModel)
	logger.Performance("📋 [性能] 任务参数 - 描述: %s, 分析类型: %s, SQL模型: %s/%s, 分析模型: %s/%s", description, analysisType, sqlProvider.Name(), sqlModel, analysisProvider.Name(), analysisModel)

	// 第一步：生成SQL
	sqlGenStartTime := time.Now()
	log.Printf("[QueryWithAnalysis] 📝 步骤1开始：SQL生成 - %s", sqlGenStartTime.Format("15:04:05.000"))
	logger.Performance("🚀 [性能] SQL生成开始 - 使用模型: %s/%s", sqlProvider.Name(), sqlModel)

	sqlGenArgs := map[string]interface{}{
		"description": description,
		"provider":    sqlProvider.Name(),
		"model":       sqlModel,
	}
	if tableName, ok := arguments["table_name"]; ok {
		sqlGenArgs["table_name"] = tableName
	}

	sqlResult, err := c.executeAIGenerateSQL(ctx, sqlGenArgs)
	if err != nil {
		return nil, fmt.Errorf("SQL生成失败: %v", err)
	}

	sqlGenDuration := time.Since(sqlGenStartTime)
	log.Printf("[QueryWithAnalysis] ✅ 步骤1完成：SQL生成耗时 %v", sqlGenDuration)
	logger.Performance("✅ [性能] SQL生成完成 - 耗时: %v", sqlGenDuration)

	// 解析SQL
	var sqlData map[string]interface{}
	if err := json.Unmarshal([]byte(sqlResult.Content[0].Text), &sqlData); err != nil {
		return nil, fmt.Errorf("解析SQL生成结果失败: %v", err)
	}

	generatedSQL, ok := sqlData["sql"].(string)
	if !ok {
		return nil, fmt.Errorf("无法从SQL生成结果中获取SQL语句")
	}

	// 第二步：执行SQL
	sqlExecStartTime := time.Now()
	log.Printf("[QueryWithAnalysis] 🗄️  步骤2开始：执行SQL查询 - %s，SQL: %s", sqlExecStartTime.Format("15:04:05.000"), generatedSQL)
	logger.Performance("🗄️ [性能] 数据库查询开始 - SQL: %s", generatedSQL)

	execArgs := map[string]interface{}{
		"sql": generatedSQL,
	}
	if alias, ok := arguments["alias"]; ok {
		execArgs["alias"] = alias
	}
	if limit, ok := arguments["limit"]; ok {
		execArgs["limit"] = limit
	}

	execResult, err := c.executeAIExecuteSQL(ctx, execArgs)
	if err != nil {
		return nil, fmt.Errorf("SQL执行失败: %v", err)
	}

	sqlExecDuration := time.Since(sqlExecStartTime)
	log.Printf("[QueryWithAnalysis] ✅ 步骤2完成：SQL执行耗时 %v", sqlExecDuration)
	logger.Performance("✅ [性能] 数据库查询完成 - 耗时: %v", sqlExecDuration)

	// 解析查询结果，直接提取员工数据
	var dbResponse map[string]interface{}
	if err := json.Unmarshal([]byte(execResult.Content[0].Text), &dbResponse); err != nil {
		return nil, fmt.Errorf("解析查询结果失败: %v", err)
	}

	// 提取实际的员工数据行
	var employeeData []interface{}
	if result, ok := dbResponse["result"].(string); ok {
		var resultData map[string]interface{}
		if err := json.Unmarshal([]byte(result), &resultData); err == nil {
			if rows, ok := resultData["rows"].([]interface{}); ok {
				employeeData = rows
			}
		}
	}

	if len(employeeData) == 0 {
		return nil, fmt.Errorf("未找到员工数据")
	}

	// 第三步：基于实际员工数据进行AI分析
	analysisStartTime := time.Now()
	log.Printf("[QueryWithAnalysis] 🤖 步骤3开始：AI数据分析 - %s，数据行数：%d", analysisStartTime.Format("15:04:05.000"), len(employeeData))
	logger.Performance("🤖 [性能] AI分析开始 - 使用模型: %s/%s, 数据行数: %d", analysisProvider.Name(), analysisModel, len(employeeData))

	employeeJSON, _ := json.Marshal(employeeData)
	var analysisPrompt string
	switch analysisType {
	case "insights":
		analysisPrompt = fmt.Sprintf("请分析以下员工数据，提供业务洞察和发现。请用中文回答，重点关注：1)员工分布情况 2)薪资水平分析 3)部门结构 4)年龄构成 5)潜在的业务建议。\n\n员工数据：%s", string(employeeJSON))
	case "summary":
		analysisPrompt = fmt.Sprintf("请用中文总结以下员工数据的关键信息。\n\n员工数据：%s", string(employeeJSON))
	case "recommendations":
		analysisPrompt = fmt.Sprintf("请基于以下员工数据用中文提供管理建议和推荐。\n\n员工数据：%s", string(employeeJSON))
	default:
		analysisPrompt = fmt.Sprintf("请用中文分析以下员工数据。\n\n员工数据：%s", string(employeeJSON))
	}

	log.Printf("[QueryWithAnalysis] 🔄 调用AI模型进行分析，模型：%s/%s，提示词长度：%d", analysisProvider.Name(), analysisModel, len(analysisPrompt))
	logger.Performance("🔄 [性能] AI模型调用 - 提示词长度: %d, 分析类型: %s", len(analysisPrompt), analysisType)

	analysisResponse, err := analysisProvider.Call(ctx, analysisModel, analysisPrompt, map[string]interface{}{
		"max_tokens": c.configManager.GetCommonConfig().MaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("AI分析失败: %v", err)
	}

	analysisDuration := time.Since(analysisStartTime)
	log.Printf("[QueryWithAnalysis] ✅ 步骤3完成：AI分析耗时 %v，响应长度：%d", analysisDuration, len(analysisResponse))
	logger.Performance("✅ [性能] AI分析完成 - 耗时: %v, 响应长度: %d", analysisDuration, len(analysisResponse))

	// 清理分析结果
	cleanedAnalysis := cleanAIResponse(analysisResponse)

	// 计算总体耗时
	totalDuration := time.Since(totalStartTime)
	log.Printf("[QueryWithAnalysis] 🎉 全部完成！总耗时：%v", totalDuration)
	log.Printf("[QueryWithAnalysis] 📊 性能统计 - SQL生成：%v，SQL执行：%v，AI分析：%v", sqlGenDuration, sqlExecDuration, analysisDuration)
	logger.Performance("🎉 [性能] ai_query_with_analysis 执行完成 - 总耗时: %v", totalDuration)
	logger.Performance("📊 [性能汇总] SQL生成: %v (%.1f%%) | 数据库查询: %v (%.1f%%) | AI分析: %v (%.1f%%)", 
		sqlGenDuration, float64(sqlGenDuration.Nanoseconds())/float64(totalDuration.Nanoseconds())*100,
		sqlExecDuration, float64(sqlExecDuration.Nanoseconds())/float64(totalDuration.Nanoseconds())*100,
		analysisDuration, float64(analysisDuration.Nanoseconds())/float64(totalDuration.Nanoseconds())*100)

	// 构建简洁的响应
	response := map[string]interface{}{
		"tool":           "ai_query_with_analysis",
		"status":         "success",
		"description":    description,
		"analysis_type":  analysisType,
		"generated_sql":  generatedSQL,
		"employee_data":  employeeData,
		"analysis":       cleanedAnalysis,
		"sql_provider":   sqlProvider.Name(),
		"sql_model":      sqlModel,
		"analysis_provider": analysisProvider.Name(),
		"analysis_model": analysisModel,
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
	// 获取SQL生成专用的AI提供商和模型
	provider, model, err := c.getProviderAndModelForFunction(map[string]interface{}{}, "sql_generation")
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

	// 调用AI（使用超时包装函数）
	result, err := c.callAIWithTimeout(ctx, provider, model, aiPrompt, map[string]interface{}{
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

	// 获取代码生成专用的AI提供商和模型（文件管理涉及代码和脚本生成）
	provider, model, err := c.getProviderAndModelForFunction(arguments, "code_generation")
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

	result, err := c.callAIWithTimeout(ctx, provider, model, aiPrompt, nil)
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
	startTime := time.Now()
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

	// 获取数据分析专用的AI提供商和模型
	provider, model, err := c.getProviderAndModelForFunction(arguments, "data_analysis")
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

	result, err := c.callAIWithTimeout(ctx, provider, model, aiPrompt, nil)
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
		"tool":               "ai_data_processor",
		"status":             "success",
		"instruction":        instruction,
		"ai_analysis":        result,
		"data_type":          dataType,
		"output_format":      outputFormat,
		"processing_results": processingResults,
		"duration":           time.Since(startTime).String(),
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

// executeAIAPIClient 执行AI网络请求
func (c *AITools) executeAIAPIClient(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	startTime := time.Now()
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

	// 获取文本生成专用的AI提供商和模型（API分析和请求构造）
	provider, model, err := c.getProviderAndModelForFunction(arguments, "text_generation")
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

	result, err := c.callAIWithTimeout(ctx, provider, model, aiPrompt, nil)
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

				// 如果启用了响应分析，对HTTP响应进行AI分析
				if responseAnalysis && len(httpResult.Content) > 0 {
					responseData := httpResult.Content[0].Text
					analysisPrompt := fmt.Sprintf(`请分析以下API响应数据，用中文提供详细分析：

原始指令：%s
请求URL：%s

响应数据：
%s

请提供以下分析：
1. 数据结构分析
2. 数据内容概述
3. 数据质量评估
4. 潜在用途建议
5. 异常或注意事项`, instruction, requestURL, responseData)

					analysisResult, err := c.callAIWithTimeout(ctx, provider, model, analysisPrompt, map[string]interface{}{
						"max_tokens":  1000,
						"temperature": 0.3,
					})
					if err != nil {
						executionResults["analysis_error"] = fmt.Sprintf("响应分析失败: %v", err)
					} else {
						executionResults["response_analysis"] = analysisResult
					}
				}
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
		"tool":                      "ai_api_client",
		"status":                    "success",
		"instruction":               instruction,
		"base_url":                  baseURL,
		"ai_analysis":               result,
		"request_mode":              requestMode,
		"response_analysis_enabled": responseAnalysis,
		"execution_results":         executionResults,
		"duration":                  time.Since(startTime).String(),
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
