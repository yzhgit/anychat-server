#!/bin/bash
#
# 服务健康检查脚本
# 检查所有微服务的健康状态
#

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 服务配置
declare -A SERVICES=(
    ["Auth Service"]="http://localhost:8001/health"
    ["User Service"]="http://localhost:8002/health"
    ["Friend Service"]="http://localhost:8003/health"
    ["Gateway Service"]="http://localhost:8080/health"
)

# gRPC 服务端口
declare -A GRPC_PORTS=(
    ["Auth gRPC"]="9001"
    ["User gRPC"]="9002"
    ["Friend gRPC"]="9003"
)

# 基础设施服务
declare -A INFRA=(
    ["PostgreSQL"]="5432"
    ["Redis"]="6379"
    ["NATS"]="4222"
    ["MinIO API"]="9000"
    ["MinIO Console"]="9091"
)

print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# 检查 HTTP 健康端点
check_http_health() {
    local name=$1
    local url=$2

    local response=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")

    if [ "$response" = "200" ]; then
        print_success "$name - 健康 (HTTP $response)"
        return 0
    else
        print_error "$name - 不健康 (HTTP $response)"
        return 1
    fi
}

# 检查端口
check_port() {
    local name=$1
    local port=$2

    if nc -z localhost $port 2>/dev/null; then
        print_success "$name - 端口 $port 正在监听"
        return 0
    else
        print_error "$name - 端口 $port 未监听"
        return 1
    fi
}

# 检查 Docker 容器
check_container() {
    local name=$1

    if docker ps --format '{{.Names}}' | grep -q "$name"; then
        local status=$(docker ps --filter "name=$name" --format '{{.Status}}')
        print_success "Docker 容器 $name - $status"
        return 0
    else
        print_error "Docker 容器 $name - 未运行"
        return 1
    fi
}

# 主函数
main() {
    echo -e "${GREEN}"
    echo "╔═══════════════════════════════════════════╗"
    echo "║   AnyChat 服务健康检查                    ║"
    echo "╚═══════════════════════════════════════════╝"
    echo -e "${NC}"

    echo "检查时间: $(date '+%Y-%m-%d %H:%M:%S')"

    local total=0
    local healthy=0

    # 1. 检查基础设施
    print_header "基础设施服务"
    for service in "${!INFRA[@]}"; do
        ((total++))
        if check_port "$service" "${INFRA[$service]}"; then
            ((healthy++))
        fi
    done

    # 2. 检查 HTTP 服务
    print_header "HTTP 服务"
    for service in "${!SERVICES[@]}"; do
        ((total++))
        if check_http_health "$service" "${SERVICES[$service]}"; then
            ((healthy++))
        fi
    done

    # 3. 检查 gRPC 服务
    print_header "gRPC 服务"
    for service in "${!GRPC_PORTS[@]}"; do
        ((total++))
        if check_port "$service" "${GRPC_PORTS[$service]}"; then
            ((healthy++))
        fi
    done

    # 4. 检查 Docker 容器
    if command -v docker &> /dev/null; then
        print_header "Docker 容器"
        local containers=("postgres" "redis" "nats" "minio")
        for container in "${containers[@]}"; do
            ((total++))
            if check_container "$container"; then
                ((healthy++))
            fi
        done
    fi

    # 输出总结
    echo ""
    print_header "健康检查总结"

    local percentage=$((healthy * 100 / total))

    echo "健康服务: $healthy / $total ($percentage%)"

    if [ $healthy -eq $total ]; then
        echo -e "${GREEN}✓ 所有服务健康!${NC}"
        exit 0
    elif [ $healthy -ge $((total * 2 / 3)) ]; then
        echo -e "${YELLOW}⚠ 部分服务不健康${NC}"
        exit 1
    else
        echo -e "${RED}✗ 大部分服务不健康${NC}"
        exit 2
    fi
}

# 运行主函数
main "$@"
