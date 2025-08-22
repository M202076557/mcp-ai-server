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

## 预期结果

- **系统操作**: 文件正常创建和读取，系统命令返回当前时间，目录列表显示当前目录内容
- **数据处理**: JSON 格式化输出，Base64 编码结果，SHA256 哈希值
- **网络操作**: 返回 JSON 响应数据，ping 统计信息，IP 地址列表
- **数据库操作**: 成功连接 MySQL，完成表创建、数据插入、查询、更新、删除和表清理的完整流程

## 注意事项

- 所有演示文件都在当前目录下创建，每个操作完成后会自动清理相关文件
- 数据库演示会在 mcp_test 数据库中创建 demo_user 表，完成 CRUD 操作展示，最后会自动删除演示表
- 演示完成后数据库将恢复到初始状态，不会留下测试数据

---

**总结**: 通过这些简单命令，可以快速展示 MCP AI Server 的核心功能，包括系统操作、数据处理、网络请求和数据库交互能力。
