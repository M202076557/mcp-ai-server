package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	_ "github.com/lib/pq"              // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3"    // SQLite driver

	"mcp-ai-server/internal/config"
	"mcp-ai-server/internal/mcp"
)

// DatabaseTools 数据库操作工具集合
type DatabaseTools struct {
	securityManager *config.SecurityManager
	connections     []*sql.DB          // 使用切片管理连接
	aliasMap        map[string]*sql.DB // 别名到连接的映射
}

// DatabaseResource 数据库资源
type DatabaseResource struct {
	URI         string
	Name        string
	Description string
	MimeType    string
	Alias       string
	TableName   string
}

// NewDatabaseTools 创建新的数据库工具集合，并建立默认连接
func NewDatabaseTools(securityManager *config.SecurityManager) *DatabaseTools {
	dt := &DatabaseTools{
		securityManager: securityManager,
		connections:     make([]*sql.DB, 0),
		aliasMap:        make(map[string]*sql.DB),
	}

	// 尝试建立默认数据库连接
	if err := dt.initializeDefaultConnection(); err != nil {
		// 如果默认连接失败，只记录错误但不阻止创建工具
		fmt.Fprintf(os.Stderr, "[DEBUG] 警告：初始化默认数据库连接失败: %v\n", err)
	}

	return dt
}

// initializeDefaultConnection 初始化默认数据库连接
func (t *DatabaseTools) initializeDefaultConnection() error {
	// 创建数据库配置管理器
	dbConfigMgr, err := config.NewDatabaseConfigManager("configs/config.yaml")
	if err != nil {
		return fmt.Errorf("创建数据库配置管理器失败: %v", err)
	}

	// 获取默认连接配置
	alias, driver, dsn, err := dbConfigMgr.GetDefaultConnection()
	if err != nil {
		return fmt.Errorf("获取默认连接配置失败: %v", err)
	}

	// 建立默认连接
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return fmt.Errorf("连接默认数据库失败: %v", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("默认数据库连接测试失败: %v", err)
	}

	// 设置连接参数
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// 存储连接
	t.aliasMap[alias] = db
	t.connections = append(t.connections, db)

	fmt.Fprintf(os.Stderr, "[DEBUG] 成功建立默认数据库连接: %s (%s)\n", alias, driver)
	return nil
}

// DBConnectTool 数据库连接工具
func (t *DatabaseTools) DBConnectTool() mcp.Tool {
	return mcp.Tool{
		Name:        "db_connect",
		Description: "连接到数据库，并返回连接索引。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"driver": map[string]interface{}{
					"type":        "string",
					"description": "数据库驱动类型 (mysql, postgres, sqlite3)",
					"enum":        []string{"mysql", "postgres", "sqlite3"},
				},
				"dsn": map[string]interface{}{
					"type":        "string",
					"description": "数据库连接字符串",
				},
			},
			"required": []string{"driver", "dsn"},
		},
	}
}

// DBQueryTool 数据库查询工具
func (t *DatabaseTools) DBQueryTool() mcp.Tool {
	return mcp.Tool{
		Name:        "db_query",
		Description: "执行数据库查询",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"alias": map[string]interface{}{
					"type":        "string",
					"description": "数据库连接别名",
				},
				"sql": map[string]interface{}{
					"type":        "string",
					"description": "SQL查询语句",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "查询结果限制数量",
					"default":     100,
				},
			},
			"required": []string{"alias", "sql"},
		},
	}
}

// DBExecuteTool 数据库执行工具
func (t *DatabaseTools) DBExecuteTool() mcp.Tool {
	return mcp.Tool{
		Name:        "db_execute",
		Description: "执行数据库操作 (INSERT, UPDATE, DELETE)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"alias": map[string]interface{}{
					"type":        "string",
					"description": "数据库连接别名",
				},
				"sql": map[string]interface{}{
					"type":        "string",
					"description": "SQL执行语句",
				},
			},
			"required": []string{"alias", "sql"},
		},
	}
}

// GetTools 获取所有数据库工具
func (t *DatabaseTools) GetTools() []mcp.Tool {
	return []mcp.Tool{
		t.DBConnectTool(),
		t.DBQueryTool(),
		t.DBExecuteTool(),
	}
}

// ExecuteTool 执行数据库工具
func (t *DatabaseTools) ExecuteTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	switch name {
	case "db_connect":
		return t.executeDBConnect(ctx, arguments)
	case "db_query":
		return t.executeDBQuery(ctx, arguments)
	case "db_execute":
		return t.executeDBExecute(ctx, arguments)
	default:
		return nil, fmt.Errorf("未知的数据库工具: %s", name)
	}
}

// executeDBConnect 执行数据库连接
func (t *DatabaseTools) executeDBConnect(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	driver, ok := arguments["driver"].(string)
	if !ok {
		return nil, fmt.Errorf("driver参数必须是字符串")
	}
	dsn, ok := arguments["dsn"].(string)
	if !ok {
		return nil, fmt.Errorf("dsn参数必须是字符串")
	}
	alias, ok := arguments["alias"].(string)
	if !ok {
		return nil, fmt.Errorf("alias参数必须是字符串")
	}

	// 如果别名已存在，关闭旧连接
	if existingDB, exists := t.aliasMap[alias]; exists {
		existingDB.Close()
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %v", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库连接测试失败: %v", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// 将连接存储到别名映射中
	t.aliasMap[alias] = db
	t.connections = append(t.connections, db)
	newIndex := len(t.connections) - 1

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: fmt.Sprintf("成功连接到数据库 %s，别名: %s，连接索引为: %d", driver, alias, newIndex),
			},
		},
	}, nil
}

// executeDBQuery 执行数据库查询
func (t *DatabaseTools) executeDBQuery(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	alias, ok := arguments["alias"].(string)
	if !ok {
		return nil, fmt.Errorf("alias参数必须是字符串")
	}

	db, exists := t.aliasMap[alias]
	if !exists {
		return nil, fmt.Errorf("未找到别名为 %s 的数据库连接", alias)
	}

	sqlQuery, ok := arguments["sql"].(string)
	if !ok {
		return nil, fmt.Errorf("sql参数必须是字符串")
	}

	limit := 100
	if limitVal, ok := arguments["limit"].(float64); ok {
		limit = int(limitVal)
	}

	sqlLower := strings.ToLower(strings.TrimSpace(sqlQuery))
	if !strings.HasPrefix(sqlLower, "select") {
		return nil, fmt.Errorf("db_query只允许SELECT查询语句")
	}

	rows, err := db.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("查询执行失败: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取列信息失败: %v", err)
	}

	var results []map[string]interface{}
	rowCount := 0
	for rows.Next() && rowCount < limit {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("扫描行数据失败: %v", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历结果集失败: %v", err)
	}

	output := map[string]interface{}{
		"columns":   columns,
		"rows":      results,
		"row_count": rowCount,
		"limited":   rowCount >= limit,
	}

	outputJSON, _ := json.MarshalIndent(output, "", "  ")

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: string(outputJSON),
			},
		},
	}, nil
}

// executeDBExecute 执行数据库操作
func (t *DatabaseTools) executeDBExecute(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	alias, ok := arguments["alias"].(string)
	if !ok {
		return nil, fmt.Errorf("alias参数必须是字符串")
	}

	db, exists := t.aliasMap[alias]
	if !exists {
		return nil, fmt.Errorf("未找到别名为 %s 的数据库连接", alias)
	}

	sqlQuery, ok := arguments["sql"].(string)
	if !ok {
		return nil, fmt.Errorf("sql参数必须是字符串")
	}

	sqlLower := strings.ToLower(strings.TrimSpace(sqlQuery))
	dangerousKeywords := []string{"truncate", "alter"}
	for _, keyword := range dangerousKeywords {
		if strings.HasPrefix(sqlLower, keyword) {
			return nil, fmt.Errorf("不允许执行 %s 语句", strings.ToUpper(keyword))
		}
	}

	result, err := db.ExecContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("执行SQL失败: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertId, _ := result.LastInsertId()

	output := map[string]interface{}{
		"rows_affected":  rowsAffected,
		"last_insert_id": lastInsertId,
		"status":         "success",
	}

	outputJSON, _ := json.MarshalIndent(output, "", "  ")

	return &mcp.ToolCallResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: string(outputJSON),
			},
		},
	}, nil
}
