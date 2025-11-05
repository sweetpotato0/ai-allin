.PHONY: help build run test clean docker-build docker-up docker-down lint fmt

# 版本信息
VERSION ?= 1.0.0
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d %H:%M:%S')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 源文件列表
SOURCES := main.go ecommerce_service.go api.go database.go

help:
	@echo "E-Commerce AI Customer Service Platform - Makefile"
	@echo ""
	@echo "可用命令:"
	@echo "  make build           - 构建可执行文件"
	@echo "  make run             - 运行应用"
	@echo "  make test            - 运行测试"
	@echo "  make bench           - 运行基准测试"
	@echo "  make coverage        - 生成覆盖率报告"
	@echo "  make clean           - 清理构建文件"
	@echo "  make lint            - 代码检查"
	@echo "  make fmt             - 代码格式化"
	@echo "  make docker-build    - 构建Docker镜像"
	@echo "  make docker-up       - 启动Docker Compose"
	@echo "  make docker-down     - 停止Docker Compose"
	@echo "  make api-docs        - 生成API文档"
	@echo "  make all             - 完整构建和测试"

# 构建可执行文件
build:
	@echo "构建应用程序..."
	go build -v \
		-ldflags="-w -s -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)" \
		-o ecommerce-platform $(SOURCES)
	@echo "✓ 构建完成: ecommerce-platform"

# 运行应用
run: build
	@echo "启动应用程序..."
	./ecommerce-platform

# 运行测试
test:
	@echo "运行测试..."
	go test -v -race -timeout 30s ./...
	@echo "✓ 测试完成"

# 基准测试
bench:
	@echo "运行基准测试..."
	go test -bench=. -benchmem -run=^$ ./...

# 覆盖率报告
coverage:
	@echo "生成覆盖率报告..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ 覆盖率报告已生成: coverage.html"

# 代码检查
lint:
	@echo "运行代码检查..."
	golint ./...
	go vet ./...
	@echo "✓ 代码检查完成"

# 代码格式化
fmt:
	@echo "格式化代码..."
	go fmt ./...
	goimports -w .
	@echo "✓ 代码格式化完成"

# 清理
clean:
	@echo "清理构建文件..."
	rm -f ecommerce-platform
	rm -f coverage.out coverage.html
	go clean
	@echo "✓ 清理完成"

# Docker构建
docker-build:
	@echo "构建Docker镜像..."
	docker build -t ecommerce-platform:$(VERSION) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME="$(BUILD_TIME)" \
		-f Dockerfile .
	@echo "✓ Docker镜像构建完成"

# Docker启动
docker-up:
	@echo "启动Docker Compose栈..."
	docker-compose up -d
	@echo "✓ Docker Compose已启动"
	@echo ""
	@echo "服务地址:"
	@echo "  应用程序:    http://localhost:8080"
	@echo "  Grafana:     http://localhost:3000"
	@echo "  Prometheus:  http://localhost:9090"
	@echo "  Kibana:      http://localhost:5601"

# Docker停止
docker-down:
	@echo "停止Docker Compose栈..."
	docker-compose down -v
	@echo "✓ Docker Compose已停止"

# 查看日志
docker-logs:
	docker-compose logs -f app

# API文档
api-docs:
	@echo "生成API文档..."
	@echo "API文档可以通过以下curl命令查看:"
	@echo ""
	@echo "# 健康检查"
	@echo "curl http://localhost:8080/api/v1/health"
	@echo ""
	@echo "# 处理客户咨询"
	@echo "curl -X POST http://localhost:8080/api/v1/inquiry \\"
	@echo "  -H 'Content-Type: application/json' \\"
	@echo "  -d '{\"customer_id\":\"CUST001\",\"message\":\"查询订单\"}'"
	@echo ""
	@echo "# 获取所有客户"
	@echo "curl http://localhost:8080/api/v1/customers"
	@echo ""
	@echo "# 获取所有订单"
	@echo "curl http://localhost:8080/api/v1/orders"
	@echo ""
	@echo "# 获取所有工单"
	@echo "curl http://localhost:8080/api/v1/tickets"
	@echo ""
	@echo "# 获取指标"
	@echo "curl http://localhost:8080/api/v1/metrics"

# 完整构建和测试
all: clean fmt lint test build
	@echo ""
	@echo "╔════════════════════════════════════════════════════════════════╗"
	@echo "║ 完整构建和测试完成！✓                                        ║"
	@echo "╚════════════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "运行应用: make run"
	@echo "启动Docker: make docker-up"

# 性能分析
profile:
	@echo "生成性能分析数据..."
	go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./...
	go tool pprof -http=:8081 cpu.prof

# 版本信息
version:
	@echo "应用版本信息:"
	@echo "  版本: $(VERSION)"
	@echo "  构建时间: $(BUILD_TIME)"
	@echo "  Git提交: $(GIT_COMMIT)"
