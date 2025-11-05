#!/bin/bash

# E-Commerce AI Customer Service Platform - Initialization Script
# 初始化脚本，用于设置开发和生产环境

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_NAME="ecommerce-platform"
PROJECT_VERSION="1.0.0"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印信息函数
print_header() {
    echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# 检查前置条件
check_requirements() {
    print_header "检查前置条件"

    # 检查Go
    if ! command -v go &> /dev/null; then
        print_error "Go未安装。请访问 https://golang.org/doc/install"
        exit 1
    fi
    print_success "Go已安装: $(go version)"

    # 检查Docker
    if ! command -v docker &> /dev/null; then
        print_warning "Docker未安装（可选）"
    else
        print_success "Docker已安装: $(docker --version)"
    fi

    # 检查Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        print_warning "Docker Compose未安装（可选）"
    else
        print_success "Docker Compose已安装: $(docker-compose --version)"
    fi

    # 检查Git
    if ! command -v git &> /dev/null; then
        print_warning "Git未安装"
    else
        print_success "Git已安装: $(git --version)"
    fi
}

# 初始化Go模块
init_go_module() {
    print_header "初始化Go模块"

    if [ ! -f "go.mod" ]; then
        print_warning "go.mod不存在，创建新的Go模块..."
        # 这里应该从上级目录继承
        print_warning "请确保在正确的项目目录中运行此脚本"
    else
        print_success "go.mod已存在"
    fi

    # 下载依赖
    print_success "下载Go依赖..."
    go mod download
    go mod tidy
}

# 生成环境文件
generate_env_file() {
    print_header "生成环境文件"

    ENV_FILE=".env"

    if [ -f "$ENV_FILE" ]; then
        print_warning ".env文件已存在，跳过生成"
        return
    fi

    cat > "$ENV_FILE" << 'EOF'
# LLM配置
OPENAI_API_KEY=your-api-key-here
LLM_PROVIDER=openai
LLM_MODEL=gpt-4

# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=ecommerce_service

# Redis配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# Elasticsearch配置
ELASTICSEARCH_HOST=localhost:9200

# 应用配置
APP_PORT=8080
APP_ENV=development
LOG_LEVEL=info
EOF

    print_success "生成.env文件"
    print_warning "请编辑.env文件并填入真实的API密钥和密码"
}

# 初始化数据库
init_database() {
    print_header "初始化数据库"

    # 检查PostgreSQL是否运行
    if command -v psql &> /dev/null; then
        print_success "PostgreSQL客户端已安装"

        # 创建数据库脚本
        cat > init.sql << 'EOF'
-- 创建数据库
CREATE DATABASE IF NOT EXISTS ecommerce_service;

-- 使用数据库
\c ecommerce_service;

-- 创建customers表
CREATE TABLE IF NOT EXISTS customers (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20),
    vip_level VARCHAR(20),
    registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    total_spent DECIMAL(10,2) DEFAULT 0,
    order_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建orders表
CREATE TABLE IF NOT EXISTS orders (
    order_id VARCHAR(50) PRIMARY KEY,
    customer_id VARCHAR(50) NOT NULL,
    total_amount DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    tracking_url VARCHAR(500),
    FOREIGN KEY (customer_id) REFERENCES customers(id)
);

-- 创建tickets表
CREATE TABLE IF NOT EXISTS tickets (
    ticket_id VARCHAR(50) PRIMARY KEY,
    customer_id VARCHAR(50) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    priority VARCHAR(20) DEFAULT 'medium',
    status VARCHAR(20) DEFAULT 'open',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    assigned_to VARCHAR(50),
    description TEXT,
    solution TEXT,
    FOREIGN KEY (customer_id) REFERENCES customers(id)
);

-- 创建索引
CREATE INDEX idx_customers_vip ON customers(vip_level);
CREATE INDEX idx_orders_customer ON orders(customer_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_tickets_customer ON tickets(customer_id);
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_priority ON tickets(priority);
EOF

        print_success "生成init.sql脚本"
    else
        print_warning "PostgreSQL客户端未安装，跳过数据库初始化"
    fi
}

# 构建应用
build_application() {
    print_header "构建应用程序"

    print_success "运行 make build..."
    make build

    if [ -f "ecommerce-platform" ]; then
        print_success "构建成功：ecommerce-platform"
    else
        print_error "构建失败"
        exit 1
    fi
}

# 运行测试
run_tests() {
    print_header "运行测试"

    print_success "运行单元测试..."
    go test -v -race ./...
}

# 显示下一步步骤
show_next_steps() {
    print_header "初始化完成！"

    echo ""
    echo -e "${GREEN}✓ 初始化完成${NC}"
    echo ""
    echo "下一步步骤:"
    echo ""
    echo "1. 编辑环境变量:"
    echo "   vim .env"
    echo ""
    echo "2. 启动开发环境（选项A：本地）:"
    echo "   make run"
    echo ""
    echo "3. 启动生产环境（选项B：Docker）:"
    echo "   make docker-up"
    echo ""
    echo "4. 运行完整测试:"
    echo "   make test"
    echo ""
    echo "5. 查看API文档:"
    echo "   make api-docs"
    echo ""
    echo "6. 访问应用:"
    echo "   http://localhost:8080"
    echo ""
    echo "详细信息请查看 README.md"
}

# 主函数
main() {
    echo ""
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║ $PROJECT_NAME 初始化脚本                                    ║${NC}"
    echo -e "${BLUE}║ Production E-Commerce AI Customer Service Platform             ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    check_requirements
    echo ""

    init_go_module
    echo ""

    generate_env_file
    echo ""

    init_database
    echo ""

    build_application
    echo ""

    echo "是否运行测试? (y/n)"
    read -r run_tests_choice
    if [ "$run_tests_choice" = "y" ] || [ "$run_tests_choice" = "Y" ]; then
        run_tests
        echo ""
    fi

    show_next_steps
}

# 运行主函数
main
