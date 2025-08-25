# MCP AI Server 核心功能演示

这是一个简洁的演示脚本，展示 MCP AI Server 的关键工具功能，适合向他人介绍使用。

## 准备工作

```bash
# 1. 启动 MCP Server (stdio模式)
./bin/mcp-server -mode=stdio

# 2. 在另一个终端启动客户端
./bin/mcp-client
```

## 1. 系统操作演示

```bash
# 创建演示文件（在当前目录）
call file_write path:"./demo.txt" content:"MCP Server 演示文件"

# 读取文件内容
call file_read path:"./demo.txt"

# 执行系统命令
call command_execute command:"date"

# 列出当前目录内容
call directory_list path:"."
```

**演示效果**: 展示文件创建、读取、目录列表和系统命令执行能力

## 2. 数据处理演示

```bash
# JSON 数据处理
call json_parse json_string:'{"name":"MCP Demo","version":"1.0","features":["system","database","network"]}' pretty:true

# 文本编码处理
call base64_encode text:"Hello MCP Server"

# 数据哈希计算
call hash text:"MCP Server Demo" algorithm:"sha256"
```

**演示效果**: 展示数据解析、编码和哈希计算能力

## 3. 网络操作演示

```bash
# HTTP 请求测试
call http_get url:"https://httpbin.org/json"

# 网络连通性检查
call ping host:"8.8.8.8" count:3

# DNS 域名解析
call dns_lookup domain:"github.com"
```

**演示效果**: 展示 HTTP 请求、网络检测和 DNS 解析能力

## 4. 数据库操作演示

```bash
# 连接MySQL数据库
call db_connect driver:"mysql" dsn:"root:root@tcp(127.0.0.1:3306)/mcp_test" alias:"demo"

# 创建演示表
call db_execute alias:"demo" sql:"CREATE TABLE IF NOT EXISTS demo_user (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(50), email VARCHAR(100))"

# 插入数据（增）
call db_execute alias:"demo" sql:"INSERT INTO demo_user (name, email) VALUES ('张三', 'zhangsan@example.com')"

# 查询数据（查）
call db_query alias:"demo" sql:"SELECT * FROM demo_user"

# 更新数据（改）
call db_execute alias:"demo" sql:"UPDATE demo_user SET email = 'zhangsan_new@example.com' WHERE name = '张三'"

# 验证更新
call db_query alias:"demo" sql:"SELECT * FROM demo_user"

# 删除数据（删）
call db_execute alias:"demo" sql:"DELETE FROM demo_user WHERE name = '张三'"

# 验证删除
call db_query alias:"demo" sql:"SELECT COUNT(*) as count FROM demo_user"

# 清理：删除演示表
call db_execute alias:"demo" sql:"DROP TABLE IF EXISTS demo_user"
```

**演示效果**: 展示 MySQL 数据库连接和完整的 CRUD 操作能力（增查改删）

## 5. AI 工具功能演示

> **注意**: AI 工具演示需要先配置 AI 提供商（Ollama/OpenAI/Anthropic）并在配置文件中启用相应服务

### 5.1 基础 AI 对话

```bash
# 基础AI聊天 - 使用默认提供商
call ai_chat prompt:"你好，请介绍一下MCP协议是什么？"

# 指定提供商和模型
call ai_chat prompt:"解释一下Go语言的并发特性" provider:"ollama" model:"llama2:7b"

# 调整生成参数
call ai_chat prompt:"写一个简单的Go语言Hello World程序" provider:"ollama" max_tokens:500 temperature:0.3
```

### 5.2 SQL 生成功能

```bash
# 根据自然语言生成SQL（仅生成，不执行）
call ai_generate_sql description:"查询所有年龄大于25岁的用户信息"

# 指定表名和结构
call ai_generate_sql description:"统计每个部门的员工数量" table_name:"employees" table_schema:"id INT, name VARCHAR(50), department VARCHAR(30), age INT"

# 复杂查询生成
call ai_generate_sql description:"查询最近30天内注册的用户，按注册时间降序排列，只显示前10个"
```

### 5.3 智能查询（自动检测 SQL 或自然语言）

```bash
# 前提：先连接数据库
call db_connect driver:"mysql" dsn:"root:root@tcp(127.0.0.1:3306)/mcp_test" alias:"demo"

# 创建演示数据
call db_execute alias:"demo" sql:"CREATE TABLE IF NOT EXISTS employees (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(50), department VARCHAR(30), age INT, salary DECIMAL(10,2))"
call db_execute alias:"demo" sql:"INSERT INTO employees (name, department, age, salary) VALUES ('张三', 'IT', 28, 8000), ('李四', 'HR', 32, 6500), ('王五', 'IT', 25, 7500), ('赵六', 'Finance', 30, 7000)"

# 自然语言智能查询
call ai_smart_query prompt:"查询IT部门的所有员工" alias:"demo" analysis_mode:"fast"

# SQL语句智能查询（自动检测为SQL）
call ai_smart_query prompt:"SELECT department, AVG(salary) as avg_salary FROM employees GROUP BY department" alias:"demo" analysis_mode:"full"

# 复杂自然语言查询
call ai_smart_query prompt:"找出薪资最高的3个员工及其部门信息" alias:"demo" limit:3
```

### 5.4 直接数据查询

```bash
# 直接通过自然语言获取数据
call ai_query_data description:"显示所有员工的姓名和薪资" table_name:"employees"

# 带条件的查询
call ai_query_data description:"查询年龄超过30岁的员工信息" limit:10

# 统计类查询
call ai_query_data description:"计算每个部门的平均薪资"
```

### 5.5 数据分析功能

```bash
# 首先获取一些数据进行分析
call db_query alias:"demo" sql:"SELECT department, COUNT(*) as employee_count, AVG(salary) as avg_salary, MAX(salary) as max_salary, MIN(salary) as min_salary FROM employees GROUP BY department"

# 对查询结果进行AI分析（需要将上面的结果复制为JSON格式）
call ai_analyze_data data:'[{"department":"IT","employee_count":2,"avg_salary":7750.00,"max_salary":8000.00,"min_salary":7500.00},{"department":"HR","employee_count":1,"avg_salary":6500.00,"max_salary":6500.00,"min_salary":6500.00},{"department":"Finance","employee_count":1,"avg_salary":7000.00,"max_salary":7000.00,"min_salary":7000.00}]' analysis_type:"insights"

# 获取推荐建议
call ai_analyze_data data:'[{"department":"IT","employee_count":2,"avg_salary":7750.00},{"department":"HR","employee_count":1,"avg_salary":6500.00}]' analysis_type:"recommendations"

# 生成数据摘要
call ai_analyze_data data:'[{"name":"张三","department":"IT","age":28,"salary":8000},{"name":"李四","department":"HR","age":32,"salary":6500}]' analysis_type:"summary"
```

### 5.6 查询+分析组合功能

```bash
# 一次性完成数据查询和AI分析
call ai_query_with_analysis description:"分析各部门的员工薪资分布情况" analysis_type:"insights" table_name:"employees"

# 获取业务推荐
call ai_query_with_analysis description:"查看员工年龄和薪资的关系" analysis_type:"recommendations"

# 生成数据报告摘要
call ai_query_with_analysis description:"统计公司整体员工情况" analysis_type:"summary"
```

### 5.7 清理演示数据

```bash
# 删除演示表
call db_execute alias:"demo" sql:"DROP TABLE IF EXISTS employees"
```

**演示效果**: 展示 AI 工具的完整功能链路，从基础对话到复杂的数据查询分析

## 预期结果

- **系统操作**: 文件正常创建和读取，系统命令返回当前时间，目录列表显示当前目录内容
- **数据处理**: JSON 格式化输出，Base64 编码结果，SHA256 哈希值
- **网络操作**: 返回 JSON 响应数据，ping 统计信息，IP 地址列表
- **数据库操作**: 成功连接 MySQL，完成表创建、数据插入、查询、更新、删除和表清理的完整流程
- **AI 工具功能**:
  - **基础对话**: AI 返回关于 MCP 协议、Go 语言等问题的专业回答
  - **SQL 生成**: 根据自然语言生成准确的 SQL 查询语句
  - **智能查询**: 自动识别输入类型并执行相应的查询或生成操作
  - **数据查询**: 直接通过自然语言获取数据库数据
  - **数据分析**: 对数据提供深度洞察、趋势分析和业务建议
  - **组合功能**: 一次性完成查询和分析，生成完整的数据报告

## 注意事项

### AI 工具使用前提

- **环境变量配置**: 需要设置相应的 API 密钥环境变量
  ```bash
  export OPENAI_API_KEY="sk-your-openai-key"        # OpenAI
  export ANTHROPIC_API_KEY="sk-ant-your-key"        # Anthropic
  # Ollama本地服务无需API密钥，但需要启动Ollama服务
  ```
- **配置文件修改**: 在 `configs/config.yaml` 中启用相应的 AI 提供商
  ```yaml
  tools:
    ai:
      ollama:
        enabled: true # 启用Ollama（本地）
      openai:
        enabled: true # 启用OpenAI（需要API密钥）
      anthropic:
        enabled: true # 启用Anthropic（需要API密钥）
  ```
- **服务依赖**: 如使用 Ollama，需要先启动 Ollama 服务并下载相应模型

### 其他注意事项

- 所有演示文件都在当前目录下创建，每个操作完成后会自动清理相关文件
- 数据库演示会在 mcp_test 数据库中创建演示表，完成演示后会自动删除
- AI 工具演示中的员工数据仅为演示用途，完成后会清理所有测试数据
- 演示完成后数据库将恢复到初始状态，不会留下测试数据

---

**总结**: 通过这些简单命令，可以快速展示 MCP AI Server 的核心功能，包括系统操作、数据处理、网络请求、数据库交互和强大的 AI 辅助能力。AI 工具提供了从基础对话到复杂数据分析的完整解决方案，大大提升了数据处理和分析的效率。
