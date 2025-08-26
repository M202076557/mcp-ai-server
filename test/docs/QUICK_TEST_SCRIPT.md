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

# 前提：先连接数据库并准备演示数据
call db_connect driver:"mysql" dsn:"root:root@tcp(127.0.0.1:3306)/mcp_test" alias:"demo"

# 创建演示表
call db_execute alias:"demo" sql:"CREATE TABLE IF NOT EXISTS mcp_user (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(50), email VARCHAR(100), department VARCHAR(30), age INT, salary DECIMAL(10,2))"

# 插入数据（增）
call db_execute alias:"demo" sql:"INSERT INTO mcp_user (name, email, department, age, salary) VALUES ('张三', 'zhangsan@company.com', 'IT', 28, 8000), ('李四', 'lisi@company.com', 'HR', 32, 6500), ('王五', 'wangwu@company.com', 'IT', 25, 7500), ('赵六', 'zhaoliu@company.com', 'Finance', 30, 7000)"

# 查询数据（查）
call db_query alias:"demo" sql:"SELECT * FROM mcp_user"

# 更新数据（改）
call db_execute alias:"demo" sql:"UPDATE mcp_user SET email = 'zhangsan_new@example.com' WHERE name = '张三'"

# 验证更新
call db_query alias:"demo" sql:"SELECT * FROM mcp_user"

# 删除数据（删）
call db_execute alias:"demo" sql:"DELETE FROM mcp_user WHERE name = '张三'"

# 验证删除
call db_query alias:"demo" sql:"SELECT COUNT(*) as count FROM mcp_user"

# 清理：删除演示表
call db_execute alias:"demo" sql:"DROP TABLE IF EXISTS mcp_user"
```

**演示效果**: 展示 MySQL 数据库连接和完整的 CRUD 操作能力（增查改删）

## 5. AI 工具功能演示

> **注意**: AI 工具演示需要先配置 AI 提供商（Ollama/OpenAI/Anthropic）并在配置文件中启用相应服务

### 5.1 基础 AI 对话

```bash
# 基础AI聊天 - 使用默认提供商
call ai_chat prompt:"你好，请介绍一下MCP协议是什么？50字以内"

# 指定提供商和模型
call ai_chat prompt:"解释一下Go语言的并发特性" provider:"ollama" model:"llama2:7b"
```

### 5.2 AI 智能文件管理

```bash
# AI文件管理 - 创建Go项目结构（可执行模式）
call ai_file_manager instruction:"创建一个Go项目的标准目录结构" target_path:"./demo-go-project" operation_mode:"execute"

# AI文件管理 - 修改现有项目文件（可执行模式）
call ai_file_manager instruction:"在demo-go-project目录中添加一个HTTP服务器和配置文件" target_path:"./demo-go-project" operation_mode:"execute"
```

### 5.3 AI 智能数据处理

```bash
# AI数据处理 - JSON解析和分析（可执行模式）
call ai_data_processor instruction:"解析这个JSON数据并提取所有用户的邮箱地址" input_data:'{"users":[{"name":"张三","email":"zhangsan@example.com","age":25},{"name":"李四","email":"lisi@example.com","age":30}]}' data_type:"json" output_format:"table" operation_mode:"execute"

# AI数据处理 - CSV数据转换（可执行模式）
call ai_data_processor instruction:"将CSV格式数据转换为JSON格式" input_data:"name,age,city\n张三,25,北京\n李四,30,上海" data_type:"csv" output_format:"json" operation_mode:"execute"
```

### 5.4 AI 智能网络请求

```bash
# AI网络请求 - 获取示例用户数据（简单演示）
call ai_api_client instruction:"获取用户数据" base_url:"https://jsonplaceholder.typicode.com" request_mode:"execute" response_analysis:true

# AI网络请求 - 获取测试数据（简单演示）
call ai_api_client instruction:"获取测试数据" base_url:"https://httpbin.org" request_mode:"execute" response_analysis:true
```

### 5.5 AI 智能数据库查询

```bash
# AI自然语言数据查询
call ai_query_with_analysis description:"查询所有员工信息" analysis_type:"insights" table_name:"mcp_user"

# AI数据摘要报告
call ai_query_with_analysis description:"生成公司员工整体情况报告" analysis_type:"summary"
```

### 5.6 清理演示数据

```bash
# 删除演示表
call db_execute alias:"demo" sql:"DROP TABLE IF EXISTS mcp_user"

# 清理演示文件和目录（可选）
call command_execute command:"rm" args:"-rf" args:"./demo-go-project" args:"./demo.txt"
```

**演示效果**: 展示 AI 工具的完整功能链路，从基础对话到复杂的数据查询分析

## 预期结果

- **系统操作**: 文件正常创建和读取，系统命令返回当前时间，目录列表显示当前目录内容
- **数据处理**: JSON 格式化输出，Base64 编码结果，SHA256 哈希值
- **网络操作**: 返回 JSON 响应数据，ping 统计信息，IP 地址列表
- **数据库操作**: 成功连接 MySQL，完成表创建、数据插入、查询、更新、删除和表清理的完整流程
- **AI 工具功能**:
  - **基础对话**: AI 返回关于 MCP 协议、Go 语言等问题的专业回答
  - **智能数据库**: 通过自然语言查询数据库，AI 自动生成 SQL 并提供数据分析和业务洞察
  - **AI 文件管理**: 智能理解文件操作需求，生成项目结构规划和文件管理方案
  - **AI 数据处理**: 自动识别数据格式，智能解析、验证和转换各种数据类型
  - **AI 网络请求**: 理解 API 调用意图，自动构造 HTTP 请求参数和分析响应

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
