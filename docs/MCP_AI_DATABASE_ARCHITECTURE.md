# MCP 服务 AI 查询数据库架构设计

## 1. 架构概述

### 1.1 三层架构模式

```
┌─────────────────────────────────────────────────────────┐
│                    客户端层                              │
│  Claude, GPT, 或其他支持MCP协议的AI助手                 │
└─────────────────────┬───────────────────────────────────┘
                      │ MCP协议 (WebSocket/stdio)
┌─────────────────────▼───────────────────────────────────┐
│                  MCP服务器层                            │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────┐   │
│  │   协议处理   │ │   权限管理   │ │    工具路由     │   │
│  └─────────────┘ └─────────────┘ └─────────────────┘   │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                   业务逻辑层                            │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────┐   │
│  │  AI工具管理  │ │  数据库管理  │ │    安全控制     │   │
│  └─────────────┘ └─────────────┘ └─────────────────┘   │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                   数据访问层                            │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────┐   │
│  │ AI提供商API │ │  数据库连接  │ │    缓存系统     │   │
│  │(Ollama/GPT) │ │ (MySQL/PG)  │ │   (Redis等)     │   │
│  └─────────────┘ └─────────────┘ └─────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

## 2. MCP + AI + 数据库完整流程图解

### 2.1 完整交互流程

```
用户输入自然语言
    ↓
"查询IT部门的所有员工"
    ↓
┌─────────────────────────────────────┐
│         MCP客户端 (Claude等)         │
│  ┌─────────────────────────────────┐ │
│  │   发送MCP调用请求                │ │
│  │   tool: "ai_smart_sql"         │ │
│  │   prompt: "查询IT部门员工"       │ │
│  └─────────────────────────────────┘ │
└─────────────────┬───────────────────┘
                  │ MCP协议 (步骤1)
                  ▼
┌─────────────────────────────────────┐
│           MCP服务器                 │
│  ┌─────────────────────────────────┐ │
│  │        AI工具管理器              │ │
│  │                                 │ │
│  │  2. 接收自然语言输入             │ │
│  │  3. 调用executeAIGenerateSQL     │ │
│  │     AI分析用户意图并生成SQL      │ │
│  │     "SELECT * FROM users        │ │
│  │      WHERE department='IT'"     │ │
│  │  4. SQL安全验证                 │ │
│  └─────────────────────────────────┘ │
│  ┌─────────────────────────────────┐ │
│  │       数据库工具管理器           │ │
│  │                                 │ │
│  │  5. 使用预配置的数据库连接       │ │
│  │  6. 执行安全的SQL查询           │ │
│  │  7. 获取查询结果数据            │ │
│  └─────────────────────────────────┘ │
└─────────────────┬───────────────────┘
                  │ 数据库连接 (步骤5-7)
                  ▼
┌─────────────────────────────────────┐
│            MySQL数据库              │
│  ┌─────────────────────────────────┐ │
│  │  执行: SELECT * FROM users      │ │
│  │        WHERE department='IT'    │ │
│  │                                 │ │
│  │  返回: [                        │ │
│  │    {id:1, name:"张三", dept:"IT"} │ │
│  │    {id:2, name:"李四", dept:"IT"} │ │
│  │  ]                              │ │
│  └─────────────────────────────────┘ │
└─────────────────┬───────────────────┘
                  │ 查询结果
                  ▼
┌─────────────────────────────────────┐
│           MCP服务器                 │
│  ┌─────────────────────────────────┐ │
│  │      AI结果分析器               │ │
│  │                                 │ │
│  │  8. 接收数据库查询结果           │ │
│  │  9. AI分析和格式化结果           │ │
│  │  10. 生成用户友好的回答          │ │
│  └─────────────────────────────────┘ │
└─────────────────┬───────────────────┘
                  │ MCP协议 (步骤8-10)
                  ▼
┌─────────────────────────────────────┐
│         MCP客户端 (Claude等)         │
│  ┌─────────────────────────────────┐ │
│  │   接收格式化结果                 │ │
│  │   "找到2名IT部门员工：           │ │
│  │    张三(开发工程师)、李四(测试)   │ │
│  │    平均薪资15000元..."          │ │
│  └─────────────────────────────────┘ │
└─────────────────────────────────────┘
```

### 2.2 核心安全机制

```
AI角色：理解用户意图 + 生成SQL + 分析结果
    ↓
MCP服务器：安全验证 + 执行SQL + 管理连接 + 权限控制
    ↓
数据库：只接受来自MCP的授权请求，不知道AI存在
```

### 2.3 混合查询模式 (推荐生产使用)

```go
type HybridQueryMode struct {
    Mode string // "ai_only", "db_only", "hybrid"
}

func (c *AITools) executeHybridQuery(ctx context.Context, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
    prompt := arguments["prompt"].(string)
    mode := arguments["mode"].(string) // 支持动态切换模式

    switch mode {
    case "ai_only":
        // 纯AI模拟，适用于演示、开发、无敏感数据场景
        return c.executeAIOnlyQuery(ctx, prompt)

    case "db_only":
        // 纯数据库查询，适用于传统SQL场景
        return c.executeDatabaseOnlyQuery(ctx, prompt)

    case "hybrid":
        // AI+数据库混合，适用于生产环境
        return c.executeAIWithDatabaseQuery(ctx, prompt)

    default:
        return c.executeAIWithDatabaseQuery(ctx, prompt)
    }
}
```

### 2.2 安全分层设计

```yaml
# 配置文件示例
security:
  database:
    # 数据库访问权限控制
    access_control:
      read_only: true # 只读模式
      allowed_tables: ["users", "orders", "products"]
      blocked_columns: ["password", "ssn", "credit_card"]
      row_limit: 1000 # 单次查询最大行数

    # SQL注入防护
    sql_injection_protection:
      enabled: true
      whitelist_only: true # 只允许白名单SQL模式
      parameterized_queries: true # 强制使用参数化查询

  ai:
    # AI提供商配置
    providers:
      - name: "ollama"
        type: "local" # 本地部署，数据不出网
        models: ["codellama:7b", "llama3:latest"]
      - name: "openai"
        type: "cloud" # 云服务，需要数据脱敏
        api_key_env: "OPENAI_API_KEY"

    # 数据脱敏规则
    data_masking:
      enabled: true
      rules:
        email: "***@***.com"
        phone: "***-****-****"
        name: "用户***"
```

## 3. 作为 MCP 服务提供方的部署策略

### 3.1 SaaS 模式部署

```go
// 多租户架构示例
type TenantMCPServer struct {
    TenantID     string
    DatabasePool map[string]*sql.DB  // 每个租户独立数据库
    AIProviders  []AIProvider        // 共享AI资源
    SecurityConfig *SecurityConfig   // 租户级安全配置
}

func (s *TenantMCPServer) HandleAIQuery(tenantID string, query string) {
    // 1. 租户身份验证
    tenant := s.validateTenant(tenantID)

    // 2. 权限检查
    if !tenant.HasPermission("ai_query") {
        return errors.New("权限不足")
    }

    // 3. 数据隔离查询
    db := s.DatabasePool[tenantID]
    results := s.executeSecureQuery(db, query, tenant.SecurityRules)

    // 4. 数据脱敏
    maskedResults := s.maskSensitiveData(results, tenant.MaskingRules)

    return maskedResults
}
```

### 3.2 私有化部署模式

```dockerfile
# Dockerfile示例
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mcp-server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/mcp-server .
COPY configs/ ./configs/

# 支持环境变量配置
ENV CONFIG_PATH=./configs/config.yaml
ENV DB_HOST=localhost
ENV DB_PORT=3306
ENV AI_PROVIDER=ollama
ENV AI_BASE_URL=http://localhost:11434

EXPOSE 8081
CMD ["./mcp-server"]
```

## 4. 业界最佳实践

### 4.1 企业级实现参考

```go
// 企业级AI数据查询服务架构
type EnterpriseAIQueryService struct {
    // 核心组件
    MCPServer       *mcp.Server
    AIOrchestrator  *AIOrchestrator    // AI编排器
    DataGateway     *DataGateway       // 数据网关
    SecurityManager *SecurityManager   // 安全管理器
    AuditLogger     *AuditLogger      // 审计日志

    // 缓存和优化
    QueryCache      *QueryCache       // 查询缓存
    QueryOptimizer  *QueryOptimizer   // 查询优化器
}

func (s *EnterpriseAIQueryService) ProcessAIQuery(ctx context.Context, req *AIQueryRequest) (*AIQueryResponse, error) {
    // 1. 请求验证和授权
    if err := s.SecurityManager.ValidateRequest(ctx, req); err != nil {
        return nil, err
    }

    // 2. 查询缓存检查
    if cached := s.QueryCache.Get(req.CacheKey()); cached != nil {
        return cached, nil
    }

    // 3. AI查询规划
    plan, err := s.AIOrchestrator.PlanQuery(ctx, req.Prompt)
    if err != nil {
        return nil, err
    }

    // 4. 数据访问执行
    results, err := s.DataGateway.ExecutePlan(ctx, plan)
    if err != nil {
        return nil, err
    }

    // 5. 结果后处理
    response := s.processResults(results, req.OutputFormat)

    // 6. 审计日志记录
    s.AuditLogger.LogQuery(ctx, req, response)

    // 7. 缓存结果
    s.QueryCache.Set(req.CacheKey(), response)

    return response, nil
}
```

### 4.2 数据安全和合规

```go
// 数据安全管理器
type DataSecurityManager struct {
    EncryptionKey    []byte
    MaskingRules     map[string]MaskingRule
    AccessPolicies   []AccessPolicy
    ComplianceRules  []ComplianceRule
}

type MaskingRule struct {
    ColumnPattern string            // 字段匹配模式
    MaskType      string            // 脱敏类型: hash, replace, truncate
    MaskValue     string            // 脱敏值
}

func (dm *DataSecurityManager) ApplyDataGovernance(data []map[string]interface{}) []map[string]interface{} {
    // 1. 数据分类
    classified := dm.classifyData(data)

    // 2. 敏感数据脱敏
    masked := dm.maskSensitiveData(classified)

    // 3. 合规性检查
    compliant := dm.checkCompliance(masked)

    return compliant
}
```

## 5. 部署建议

### 5.1 云原生部署

```yaml
# kubernetes部署示例
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mcp-ai-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: mcp-ai-server
  template:
    metadata:
      labels:
        app: mcp-ai-server
    spec:
      containers:
        - name: mcp-server
          image: your-registry/mcp-ai-server:latest
          ports:
            - containerPort: 8081
          env:
            - name: DB_HOST
              valueFrom:
                secretKeyRef:
                  name: db-secret
                  key: host
            - name: AI_API_KEY
              valueFrom:
                secretKeyRef:
                  name: ai-secret
                  key: api-key
          resources:
            requests:
              memory: "256Mi"
              cpu: "250m"
            limits:
              memory: "512Mi"
              cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: mcp-ai-service
spec:
  selector:
    app: mcp-ai-server
  ports:
    - port: 8081
      targetPort: 8081
  type: LoadBalancer
```

### 5.2 监控和可观测性

```go
// 监控指标
type MCPMetrics struct {
    QueryCount       prometheus.Counter    // 查询总数
    QueryDuration    prometheus.Histogram // 查询耗时
    ErrorRate        prometheus.Counter    // 错误率
    ActiveConnections prometheus.Gauge     // 活跃连接数
    DatabasePoolSize prometheus.Gauge     // 数据库连接池大小
    AIProviderLatency prometheus.Histogram // AI提供商延迟
}

func (m *MCPMetrics) RecordQuery(ctx context.Context, duration time.Duration, err error) {
    m.QueryCount.Inc()
    m.QueryDuration.Observe(duration.Seconds())
    if err != nil {
        m.ErrorRate.Inc()
    }
}
```

## 6. 成本优化策略

### 6.1 智能缓存策略

```go
type SmartCache struct {
    L1Cache *lru.Cache      // 内存缓存 (热点数据)
    L2Cache redis.Client    // Redis缓存 (温数据)
    L3Cache *sql.DB        // 数据库缓存 (冷数据)
}

func (c *SmartCache) Get(key string) (interface{}, bool) {
    // 1. 检查L1缓存
    if val, ok := c.L1Cache.Get(key); ok {
        return val, true
    }

    // 2. 检查L2缓存
    if val := c.L2Cache.Get(ctx, key).Val(); val != "" {
        c.L1Cache.Add(key, val) // 提升到L1
        return val, true
    }

    // 3. 检查L3缓存
    if val := c.queryFromDatabase(key); val != nil {
        c.L2Cache.Set(ctx, key, val, time.Hour) // 缓存到L2
        return val, true
    }

    return nil, false
}
```

### 6.2 AI 成本控制

```go
type AIBudgetController struct {
    DailyLimit   int64   // 每日token限制
    UserQuotas   map[string]int64  // 用户配额
    ModelCosts   map[string]float64 // 模型成本
}

func (bc *AIBudgetController) CanExecuteQuery(userID string, estimatedTokens int64) bool {
    userUsage := bc.getUserDailyUsage(userID)
    userQuota := bc.UserQuotas[userID]

    return userUsage+estimatedTokens <= userQuota
}
```

## 7. 总结

作为 MCP 服务提供方，成功部署 AI 查询数据库服务需要：

1. **技术架构**：选择合适的混合查询模式
2. **安全设计**：实施多层安全防护和数据治理
3. **运维保障**：建立完善的监控和运维体系
4. **成本控制**：实施智能缓存和预算控制
5. **合规管理**：满足数据保护和行业合规要求

这样可以为用户提供安全、高效、可扩展的 AI 数据查询服务。
