#!/bin/bash
#
# 环境检查和设置脚本
# 检查和配置 AnyChat 开发环境
#

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
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

print_info() {
    echo -e "  $1"
}

# 检查命令是否存在
check_command() {
    local cmd=$1
    local name=$2
    local install_hint=$3

    if command -v $cmd &> /dev/null; then
        local version=$($cmd --version 2>&1 | head -n 1)
        print_success "$name 已安装: $version"
        return 0
    else
        print_error "$name 未安装"
        if [ -n "$install_hint" ]; then
            print_info "安装提示: $install_hint"
        fi
        return 1
    fi
}

# 检查服务端口
check_port() {
    local port=$1
    local service=$2

    if nc -z localhost $port 2>/dev/null; then
        print_success "$service (端口 $port) 正在运行"
        return 0
    else
        print_warning "$service (端口 $port) 未运行"
        return 1
    fi
}

# 检查 Docker 容器
check_docker_container() {
    local container=$1
    local service=$2

    if docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
        print_success "$service 容器正在运行"
        return 0
    else
        print_warning "$service 容器未运行"
        return 1
    fi
}

# ========================================
# 检查开发工具
# ========================================

check_dev_tools() {
    print_header "检查开发工具"

    local failed=0

    # Go
    check_command go "Go" "https://go.dev/doc/install" || ((failed++))

    # Docker
    check_command docker "Docker" "https://docs.docker.com/get-docker/" || ((failed++))

    # Docker Compose
    check_command docker-compose "Docker Compose" "https://docs.docker.com/compose/install/" || ((failed++))

    # jq (用于 JSON 处理)
    check_command jq "jq" "apt-get install jq 或 brew install jq" || ((failed++))

    # curl
    check_command curl "curl" "apt-get install curl 或 brew install curl" || ((failed++))

    # netcat (用于端口检查)
    if ! command -v nc &> /dev/null; then
        print_warning "nc (netcat) 未安装，无法检查端口"
    else
        print_success "nc (netcat) 已安装"
    fi

    # protoc
    check_command protoc "Protocol Buffers" "https://grpc.io/docs/protoc-installation/" || print_warning "protoc 未安装，无法生成 proto 代码"

    # mage
    if command -v mage &> /dev/null; then
        print_success "Mage 已安装"
    else
        print_warning "Mage 未安装"
        print_info "安装命令: go install github.com/magefile/mage@latest"
    fi

    return $failed
}

# ========================================
# 检查基础设施服务
# ========================================

check_infrastructure() {
    print_header "检查基础设施服务"

    local failed=0

    # PostgreSQL
    if check_docker_container "postgres" "PostgreSQL"; then
        check_port 5432 "PostgreSQL" || ((failed++))
    else
        ((failed++))
    fi

    # Redis
    if check_docker_container "redis" "Redis"; then
        check_port 6379 "Redis" || ((failed++))
    else
        ((failed++))
    fi

    # NATS
    if check_docker_container "nats" "NATS"; then
        check_port 4222 "NATS" || ((failed++))
    else
        ((failed++))
    fi

    # MinIO
    if check_docker_container "minio" "MinIO"; then
        check_port 9000 "MinIO API" || ((failed++))
        check_port 9091 "MinIO Console" || ((failed++))
    else
        ((failed++))
    fi

    return $failed
}

# ========================================
# 检查微服务
# ========================================

check_microservices() {
    print_header "检查微服务状态"

    local services=(
        "8001:auth-service"
        "8002:user-service"
        "8003:friend-service"
        "8080:gateway-service"
    )

    local running=0
    local total=${#services[@]}

    for service_info in "${services[@]}"; do
        local port=$(echo $service_info | cut -d: -f1)
        local name=$(echo $service_info | cut -d: -f2)

        if check_port $port "$name"; then
            ((running++))
        fi
    done

    print_info "运行中的服务: $running / $total"

    if [ $running -eq $total ]; then
        return 0
    else
        return 1
    fi
}

# ========================================
# 检查数据库连接
# ========================================

check_database() {
    print_header "检查数据库连接"

    if ! command -v psql &> /dev/null; then
        print_warning "psql 未安装，跳过数据库检查"
        return 0
    fi

    # 检查数据库连接
    if PGPASSWORD=anychat123 psql -h localhost -U anychat -d anychat -c "SELECT 1" &> /dev/null; then
        print_success "数据库连接成功"

        # 检查表是否存在
        local tables=$(PGPASSWORD=anychat123 psql -h localhost -U anychat -d anychat -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'")
        print_info "数据库表数量: $(echo $tables | xargs)"

        return 0
    else
        print_error "数据库连接失败"
        return 1
    fi
}

# ========================================
# Go 模块检查
# ========================================

check_go_modules() {
    print_header "检查 Go 模块"

    if [ ! -f "go.mod" ]; then
        print_error "go.mod 文件不存在"
        return 1
    fi

    print_success "go.mod 文件存在"

    # 检查依赖
    print_info "检查 Go 依赖..."
    if go mod verify &> /dev/null; then
        print_success "Go 模块验证通过"
    else
        print_warning "Go 模块验证失败，尝试运行 go mod tidy"
    fi

    return 0
}

# ========================================
# 环境变量检查
# ========================================

check_environment_variables() {
    print_header "检查环境变量"

    local vars=(
        "GOPATH:Go 工作路径"
        "GOPROXY:Go 模块代理"
    )

    for var_info in "${vars[@]}"; do
        local var=$(echo $var_info | cut -d: -f1)
        local desc=$(echo $var_info | cut -d: -f2)

        if [ -n "${!var}" ]; then
            print_success "$var ($desc): ${!var}"
        else
            print_info "$var 未设置"
        fi
    done

    return 0
}

# ========================================
# 生成环境报告
# ========================================

generate_report() {
    print_header "环境检查总结"

    echo ""
    echo "系统信息:"
    print_info "操作系统: $(uname -s)"
    print_info "架构: $(uname -m)"

    if [ -f /etc/os-release ]; then
        source /etc/os-release
        print_info "发行版: $NAME $VERSION"
    fi

    echo ""
    echo "检查时间: $(date '+%Y-%m-%d %H:%M:%S')"
}

# ========================================
# 快速修复建议
# ========================================

suggest_fixes() {
    print_header "快速修复建议"

    echo ""
    echo "如果基础设施服务未运行，执行:"
    echo "  ${GREEN}mage docker:up${NC}"
    echo ""
    echo "如果需要运行数据库迁移，执行:"
    echo "  ${GREEN}mage db:up${NC}"
    echo ""
    echo "启动微服务:"
    echo "  ${GREEN}mage dev:auth${NC}      # 终端1: auth-service"
    echo "  ${GREEN}mage dev:user${NC}      # 终端2: user-service"
    echo "  ${GREEN}mage dev:friend${NC}    # 终端3: friend-service"
    echo "  ${GREEN}mage dev:gateway${NC}   # 终端4: gateway-service"
    echo ""
    echo "运行完整测试:"
    echo "  ${GREEN}./scripts/test-all.sh${NC}"
    echo ""
}

# ========================================
# 主函数
# ========================================

main() {
    echo -e "${GREEN}"
    echo "╔═══════════════════════════════════════════╗"
    echo "║   AnyChat 环境检查脚本                    ║"
    echo "╚═══════════════════════════════════════════╝"
    echo -e "${NC}"

    local total_failed=0

    check_dev_tools
    total_failed=$((total_failed + $?))

    check_go_modules
    total_failed=$((total_failed + $?))

    check_environment_variables

    check_infrastructure
    total_failed=$((total_failed + $?))

    check_database
    total_failed=$((total_failed + $?))

    check_microservices
    # 微服务可能未运行，不计入失败

    generate_report

    if [ $total_failed -eq 0 ]; then
        echo ""
        echo -e "${GREEN}✓ 环境检查全部通过!${NC}"
        echo ""
        exit 0
    else
        suggest_fixes
        echo ""
        echo -e "${YELLOW}⚠ 发现 $total_failed 个问题，请根据上述建议修复${NC}"
        echo ""
        exit 1
    fi
}

# 运行主函数
main "$@"
