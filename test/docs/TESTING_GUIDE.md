# MCP AI 工具完整测试用例

## 测试说明

**重要**: 下面所有的测试命令都需要在 MCP 客户端的交互式界面中手动输入执行，不需要在终端中使用 `./bin/mcp-client` 前缀。

启动客户端后，您会看到类似 `mcp>` 的提示符，直接输入命令即可，例如：

```
mcp> call ai_chat prompt:"你好"
```

## 测试环境准备

### 1. 启动服务

```bash
# 启动 MCP 服务器
cd /Users/ksc/Desktop/study/mcp-ai-server
go build -o bin/mcp-server cmd/server/main.go
./bin/mcp-server

# 启动客户端（另一个终端）
go build -o bin/mcp-client cmd/client/main.go
./bin/mcp-client
```

### 2. 数据库准备

确保数据库连接正常，表结构如下：

```sql
CREATE TABLE mcp_user (
    id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    full_name VARCHAR(100),
    age INT,
    department VARCHAR(50),
    position VARCHAR(50),
    salary DECIMAL(10,2),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

## AI 工具测试用例

### 1. ai_chat - 基础 AI 对话

**测试目标**: 验证基础 AI 对话功能

```bash
# 基础对话测试
call ai_chat prompt:"你好，请介绍一下MCP协议"

# 技术问题测试
call ai_chat prompt:"什么是WebSocket协议？" provider:ollama

# 复杂对话测试
call ai_chat prompt:"请解释一下机器学习中的监督学习和无监督学习的区别" max_tokens:500
```

**预期结果**: 返回相关的 AI 回答，无数据库操作

### 2. ai_generate_sql - SQL 生成

**测试目标**: 验证自然语言转 SQL 功能

```bash
# 基础查询SQL生成
call ai_generate_sql description:"查询所有IT部门的员工"

# 复杂查询SQL生成
call ai_generate_sql description:"查询年龄大于30岁且薪资超过8000的员工，按薪资降序排列" table_name:mcp_user

# 聚合查询SQL生成
call ai_generate_sql description:"统计各部门的员工数量和平均薪资"

# 带条件的SQL生成
call ai_generate_sql description:"查询最近一个月新入职的活跃员工" table_name:mcp_user
```

**预期结果**: 返回生成的 SQL 语句，但不执行

### 3. ai_smart_sql - AI 智能 SQL 查询

**测试目标**: 验证智能 SQL 查询功能（自然语言 + 直接 SQL）

#### 3.1 自然语言模式

```bash
# 自然语言查询（会自动建立数据库连接）
call ai_smart_sql prompt:"查询所有IT部门的员工"

# 复杂自然语言查询
call ai_smart_sql prompt:"查询薪资最高的5名员工" limit:5

# 带条件的查询
call ai_smart_sql prompt:"查询年龄在25-35岁之间的员工信息"
```

#### 3.2 直接 SQL 模式

```bash
# 直接执行SQL
call ai_smart_sql sql:"SELECT * FROM mcp_user WHERE department = 'IT' LIMIT 10"

# 复杂SQL执行
call ai_smart_sql sql:"SELECT department, COUNT(*) as count, AVG(salary) as avg_salary FROM mcp_user GROUP BY department"

# 带参数的SQL
call ai_smart_sql sql:"SELECT * FROM mcp_user WHERE salary > 8000 ORDER BY salary DESC" limit:20
```

**预期结果**: 返回查询结果数据，支持两种输入模式

### 4. ai_query_data - 直接数据查询

**测试目标**: 验证直接数据查询功能

```bash
# 基础数据查询
call ai_query_data description:"查询所有用户信息"

# 条件查询
call ai_query_data description:"查询IT部门的所有员工" table_name:mcp_user

# 限制结果数量
call ai_query_data description:"查询薪资最高的员工" limit:10

# 复杂业务查询
call ai_query_data description:"查询各部门的人员分布情况"
```

**预期结果**: 返回查询数据，不包含 AI 分析

### 5. ai_analyze_data - 数据分析

**测试目标**: 验证数据分析功能

#### 5.1 准备测试数据

```bash
# 先获取一些数据用于分析
call ai_query_data description:"查询所有部门的员工统计信息" > test_data.json
```

#### 5.2 分析测试

```bash
# 摘要分析
call ai_analyze_data data:"{\"departments\":[{\"name\":\"IT\",\"count\":15,\"avg_salary\":9500},{\"name\":\"HR\",\"count\":8,\"avg_salary\":7200}]}" analysis_type:summary

# 洞察分析
call ai_analyze_data data:"{\"employees\":50,\"departments\":[\"IT\",\"HR\",\"Finance\"],\"avg_age\":32}" analysis_type:insights

# 建议分析
call ai_analyze_data data:"{\"turnover_rate\":0.15,\"satisfaction_score\":3.8}" analysis_type:recommendations
```

**预期结果**: 返回 AI 分析结果

### 6. ai_query_with_analysis - 数据查询+分析

**测试目标**: 验证查询和分析组合功能

```bash
# 查询+摘要分析
call ai_query_with_analysis description:"查询所有部门的薪资情况" analysis_type:summary

# 查询+洞察分析
call ai_query_with_analysis description:"查询员工年龄分布" analysis_type:insights

# 查询+建议分析
call ai_query_with_analysis description:"查询各部门的人员流动情况" analysis_type:recommendations

# 复杂业务分析
call ai_query_with_analysis description:"分析公司的薪资结构和员工满意度关系" analysis_type:insights
```

**预期结果**: 返回查询数据和对应分析

### 7. ai_smart_insights - 智能洞察

**测试目标**: 验证深度智能分析功能

#### 7.1 基础洞察

```bash
# 基础洞察
call ai_smart_insights prompt:"分析公司的人力资源现状" insight_level:basic

# 高级洞察
call ai_smart_insights prompt:"分析各部门的工作效率和成本效益" insight_level:advanced

# 战略洞察
call ai_smart_insights prompt:"制定公司未来3年的人才发展战略" insight_level:strategic
```

#### 7.2 带上下文的洞察

```bash
# 带业务上下文
call ai_smart_insights prompt:"优化部门人员配置" context:"公司正在进行数字化转型，需要更多技术人才" insight_level:advanced

# 带行业上下文
call ai_smart_insights prompt:"制定薪酬竞争策略" context:"科技行业人才竞争激烈，离职率较高" insight_level:strategic
```

**预期结果**: 返回深度业务分析和建议

### 8. ai_smart_query - 智能查询（兼容性测试）

**测试目标**: 验证原有智能查询功能的兼容性

```bash
# 完整模式测试
call ai_smart_query prompt:"分析公司的人力成本结构" analysis_mode:full

# 快速模式测试
call ai_smart_query prompt:"查询IT部门员工信息" analysis_mode:fast

# 自定义表测试
call ai_smart_query prompt:"查询用户活跃度" table_name:mcp_user analysis_mode:full
```

**预期结果**: 返回完整的智能查询结果

## 性能测试

### 响应时间测试

在客户端中依次执行以下命令来测试各工具的响应时间：

```bash
# 1. ai_chat 响应时间
call ai_chat prompt:"测试响应时间"

# 2. ai_generate_sql 响应时间
call ai_generate_sql description:"查询测试"

# 3. ai_smart_sql 响应时间
call ai_smart_sql prompt:"查询mcp_user表的第一条记录" limit:1

# 4. ai_query_data 响应时间
call ai_query_data description:"测试查询" limit:1

# 5. ai_analyze_data 响应时间
call ai_analyze_data data:"{\"test\":\"data\"}" analysis_type:summary

# 6. ai_query_with_analysis 响应时间
call ai_query_with_analysis description:"测试分析" analysis_type:summary

# 7. ai_smart_insights 响应时间
call ai_smart_insights prompt:"快速测试" insight_level:basic
```

### 并发测试

在客户端中快速执行多个相同命令来测试并发性能：

```bash
# 多次执行相同的查询命令
call ai_smart_sql sql:"SELECT COUNT(*) FROM mcp_user"
call ai_smart_sql sql:"SELECT COUNT(*) FROM mcp_user"
call ai_smart_sql sql:"SELECT COUNT(*) FROM mcp_user"
call ai_smart_sql sql:"SELECT COUNT(*) FROM mcp_user"
call ai_smart_sql sql:"SELECT COUNT(*) FROM mcp_user"
```

## 错误处理测试

### 1. 参数错误测试

```bash
# 缺少必需参数
call ai_chat

# 错误的参数类型
call ai_generate_sql description:123

# 无效的枚举值
call ai_analyze_data data:"{}" analysis_type:invalid
```

### 2. 数据库错误测试

```bash
# 无效的SQL
call ai_smart_sql sql:"INVALID SQL STATEMENT"

# 不存在的表
call ai_query_data description:"查询不存在的表" table_name:nonexistent_table
```

### 3. AI 提供商错误测试

```bash
# 无效的提供商
call ai_chat prompt:"测试" provider:invalid_provider

# 无效的模型
call ai_chat prompt:"测试" model:invalid_model
```

## 集成测试

### 完整业务流程测试

在客户端中依次执行以下命令，测试完整的业务流程：

```bash
# 1. 生成SQL
call ai_generate_sql description:"查询IT部门员工薪资情况"

# 2. 执行查询
call ai_smart_sql prompt:"查询IT部门员工薪资情况"

# 3. 分析数据
call ai_query_with_analysis description:"分析IT部门薪资结构" analysis_type:insights

# 4. 深度洞察
call ai_smart_insights prompt:"基于IT部门数据制定薪酬优化建议" insight_level:advanced
```

## 预期结果验证

### 成功案例的响应格式

1. **ai_chat 成功响应**:

```json
{
  "content": [
    {
      "type": "text",
      "text": "AI回答内容..."
    }
  ]
}
```

2. **ai_smart_sql 成功响应**:

```json
{
  "content": [
    {
      "type": "text",
      "text": "{\"tool\":\"ai_smart_sql\",\"status\":\"success\",\"sql\":\"...\",\"result\":{...}}"
    }
  ]
}
```

### 常见错误的响应格式

1. **参数错误**:

```json
{
  "error": "参数错误信息"
}
```

2. **数据库错误**:

```json
{
  "error": "SQL执行失败: ..."
}
```

## 测试通过标准

- ✅ 所有工具都能正常调用
- ✅ 参数验证正确
- ✅ 错误处理得当
- ✅ 响应格式统一
- ✅ 性能符合预期
- ✅ 集成流程顺畅

## 故障排除

### 常见问题

1. **数据库连接失败**: 检查配置文件和数据库状态
2. **AI 提供商不可用**: 检查 AI 服务配置
3. **工具调用失败**: 检查参数格式和必需字段
4. **性能问题**: 检查数据库索引和查询优化

### 调试建议

1. 开启详细日志模式
2. 检查网络连接
3. 验证配置文件
4. 测试数据库连接
5. 验证 AI 提供商状态
