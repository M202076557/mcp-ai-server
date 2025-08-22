# MCP AI Server

基于 MCP (Model Context Protocol) 的 AI 工具服务器，提供7个递增复杂度的智能工具。

## 🚀 功能特性

### 核心 AI 工具
1. **AI 对话** - 基础聊天和问答功能
2. **SQL 生成** - 根据自然语言生成SQL查询
3. **智能查询** - 统一入口，支持自然语言和SQL的智能查询
4. **数据分析** - 对数据进行智能分析和洞察
5. **查询+分析** - 一键完成数据查询和分析
6. **智能洞察** - 深度业务洞察和建议

### 技术特性
- ✅ **动态表名识别** - 自动检测数据库表结构，智能匹配表名
- ✅ **多AI提供商支持** - Ollama、OpenAI等
- ✅ **WebSocket通信** - 基于MCP协议的实时通信
- ✅ **SQL安全验证** - 防止危险操作
- ✅ **中文优化** - 专门优化的中文分析能力

## 📡 架构说明

```
┌─────────────────┐    WebSocket     ┌─────────────────┐
│  mcp-ai-client  │ ←────────────── │  mcp-ai-server  │
│   (HTTP API)    │     MCP协议      │   (AI工具服务)   │
└─────────────────┘                 └─────────────────┘
        ↑                                    ↓
        │                           ┌─────────────────┐
    REST API                        │   AI Provider   │
        │                           │ (Ollama/OpenAI) │
        ↓                           └─────────────────┘
┌─────────────────┐                          ↓
│   前端应用/     │                 ┌─────────────────┐
│    用户端       │                 │    MySQL DB     │
└─────────────────┘                 └─────────────────┘
```

## 🔧 快速开始

### 环境要求
- Go 1.21+
- MySQL 5.7+
- Ollama (推荐) 或其他AI提供商

### 启动服务

1. **配置数据库**：编辑 `configs/config.yaml`
2. **启动服务**：
   ```bash
   make build && make run
   # 或
   go run cmd/server/main.go
   ```
3. **服务地址**：`ws://localhost:8081`

## 📚 API文档

服务通过 WebSocket 提供 MCP 协议接口，主要工具：

- `ai_chat` - AI对话
- `ai_generate_sql` - SQL生成  
- `ai_smart_query` - 智能查询
- `ai_analyze_data` - 数据分析
- `ai_query_with_analysis` - 查询+分析
- `ai_smart_insights` - 智能洞察

## 🔍 配置说明

主要配置文件：`configs/config.yaml`

```yaml
ai:
  default_provider: "ollama"
  providers:
    ollama:
      enabled: true
      base_url: "http://localhost:11434"
      models:
        - "codellama:7b"

database:
  mysql:
    host: "localhost"
    port: 3306
    username: "root"
    password: "your_password"
    database: "your_database"
```

## 📄 许可证

MIT License
