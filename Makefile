.PHONY: help build clean test run-server run-client run-example

# 默认目标
help:
	@echo "MCP本地学习项目 - 可用命令:"
	@echo "  build        - 构建项目"
	@echo "  clean        - 清理构建文件"
	@echo "  test         - 运行测试"
	@echo "  run-server   - 运行MCP服务器 (stdio模式)"
	@echo "  run-websocket - 运行MCP服务器 (WebSocket模式, 端口8081)"
	@echo "  run-websocket-port PORT=端口号 - 运行MCP服务器 (WebSocket模式, 指定端口)"
	@echo "  run-client   - 运行MCP客户端"

	@echo "  install      - 安装依赖"
	@echo "  docker-build - 构建Docker镜像"
	@echo "  docker-run   - 运行Docker容器"

# 安装依赖
install:
	go mod tidy
	go mod download

# 构建项目
build: install
	@echo "构建MCP服务器..."
	go build -o bin/mcp-server cmd/server/main.go
	@echo "构建MCP客户端..."
	go build -o bin/mcp-client cmd/client/main.go
	@echo "构建完成！"

# 清理构建文件
clean:
	@echo "清理构建文件..."
	rm -rf bin/
	@echo "清理完成！"

# 运行测试
test:
	@echo "运行测试..."
	go test ./...

# 运行MCP服务器
run-server: build
	@echo "启动MCP服务器..."
	./bin/mcp-server

# 运行WebSocket模式的MCP服务器
run-websocket: build
	@echo "启动WebSocket模式的MCP服务器..."
	./bin/mcp-server -mode websocket -port 8081

# 运行指定端口的WebSocket MCP服务器
run-websocket-port: build
	@echo "启动WebSocket模式的MCP服务器..."
	./bin/mcp-server -mode websocket -port $(PORT)

# 运行MCP客户端
run-client: build
	@echo "启动MCP客户端..."
	./bin/mcp-client



# 构建Docker镜像
docker-build:
	@echo "构建Docker镜像..."
	docker build -t mcp-ai-server:latest .

# 运行Docker容器
docker-run: docker-build
	@echo "运行Docker容器..."
	docker run -it --rm -p 8080:8081 mcp-ai-server:latest

# 开发模式：同时运行服务器和客户端
dev: build
	@echo "开发模式：启动服务器和客户端..."
	@echo "在另一个终端中运行: make run-client"
	./bin/mcp-server

# 检查代码质量
lint:
	@echo "检查代码质量..."
	golangci-lint run

# 格式化代码
fmt:
	@echo "格式化代码..."
	go fmt ./...
	go vet ./...

# 生成文档
docs:
	@echo "生成文档..."
	@echo "文档已生成在 docs/ 目录中"

# 快速测试：构建并运行客户端
quick-test: build
	@echo "快速测试：运行客户端..."
	@echo "注意：这需要在一个终端中运行服务器，另一个终端运行客户端"
	@echo "终端1: make run-server"
	@echo "终端2: make run-client"

# 添加WebSocket调试模式
.PHONY: debug-websocket
debug-websocket:
	@echo "🔍 启动WebSocket调试模式..."
	DEBUG=true LOG_LEVEL=debug go run cmd/server/main.go -mode websocket -port 8081 -debug
