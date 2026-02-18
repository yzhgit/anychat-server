#!/bin/bash
#
# 启动所有微服务
# 按依赖顺序启动全部 12 个服务
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

print_info() {
    echo -e "${YELLOW}➜ $1${NC}"
}

# 检查端口是否可用
check_port_available() {
    local port=$1
    local result=$(lsof -i :$port 2>/dev/null | grep LISTEN || echo "")
    if [ -z "$result" ]; then
        return 0
    else
        return 1
    fi
}

# 等待服务启动（检测 gRPC 或 HTTP 端口）
wait_for_service() {
    local port=$1
    local service=$2
    local max_attempts=30
    local attempt=0

    print_info "等待 $service 启动 (端口 $port)..."

    while [ $attempt -lt $max_attempts ]; do
        if lsof -i :$port 2>/dev/null | grep LISTEN > /dev/null; then
            print_success "$service 已启动"
            return 0
        fi
        sleep 1
        ((attempt++))
        echo -n "."
    done

    echo ""
    print_error "$service 启动超时"
    return 1
}

# 启动单个服务的通用函数
# 用法: start_service <服务名> <mage目标> <检测端口> <PID文件>
start_service() {
    local name=$1
    local mage_target=$2
    local port=$3
    local pid_file="/tmp/${name}.pid"

    if check_port_available "$port"; then
        print_info "启动 ${name}..."
        nohup mage "$mage_target" > "logs/${name}.log" 2>&1 &
        local pid=$!
        echo $pid > "$pid_file"

        if wait_for_service "$port" "$name"; then
            print_success "${name} 运行在 PID: $pid"
        else
            print_error "${name} 启动失败，查看日志: logs/${name}.log"
            exit 1
        fi
    else
        print_success "${name} 已在运行"
    fi
}

# 检查基础设施
check_infrastructure() {
    print_header "检查基础设施"

    local failed=0

    if ! docker ps | grep anychat-postgres | grep -q "healthy\|Up"; then
        print_error "PostgreSQL 未运行，请先执行: mage docker:up"
        ((failed++))
    else
        print_success "PostgreSQL 正在运行"
    fi

    if ! docker ps | grep anychat-redis | grep -q "healthy\|Up"; then
        print_error "Redis 未运行，请先执行: mage docker:up"
        ((failed++))
    else
        print_success "Redis 正在运行"
    fi

    if ! docker ps | grep anychat-nats | grep -q "Up"; then
        print_error "NATS 未运行，请先执行: mage docker:up"
        ((failed++))
    else
        print_success "NATS 正在运行"
    fi

    if ! docker ps | grep anychat-minio | grep -q "healthy\|Up"; then
        print_error "MinIO 未运行，请先执行: mage docker:up"
        ((failed++))
    else
        print_success "MinIO 正在运行"
    fi

    if ! docker ps | grep anychat-livekit | grep -q "Up"; then
        print_error "LiveKit 未运行，请先执行: mage docker:up"
        ((failed++))
    else
        print_success "LiveKit 正在运行"
    fi

    if [ $failed -gt 0 ]; then
        echo ""
        print_error "基础设施未就绪，请先启动: mage docker:up"
        exit 1
    fi

    echo ""
    print_success "所有基础设施就绪"
}

# 检查数据库迁移
check_migrations() {
    print_header "检查数据库迁移"

    if docker exec anychat-postgres psql -U anychat -d anychat -c "\dt" 2>/dev/null | grep -q users; then
        print_success "数据库迁移已完成"
    else
        print_info "数据库迁移未完成，正在运行迁移..."
        mage db:up
        print_success "数据库迁移完成"
    fi
}

# 启动核心域服务（第一层，无互相依赖）
start_core_services() {
    print_header "启动核心域服务"

    start_service "auth-service"    "dev:auth"    9001
    start_service "user-service"    "dev:user"    9002
    start_service "friend-service"  "dev:friend"  9003
    start_service "group-service"   "dev:group"   9004
    start_service "file-service"    "dev:file"    9007
}

# 启动应用层服务（第二层，消息/会话）
start_app_services() {
    print_header "启动应用层服务"

    start_service "message-service" "dev:message"  9005
    start_service "session-service" "dev:session"  9006
}

# 启动辅助服务（第三层，推送/RTC/同步）
start_auxiliary_services() {
    print_header "启动辅助服务"

    start_service "push-service"    "dev:push"    9008
    start_service "rtc-service" "dev:rtc" 9009
    start_service "sync-service"    "dev:sync"    9010
}

# 启动管理服务
start_admin_service() {
    print_header "启动管理服务"

    start_service "admin-service"   "dev:admin"   9011
}

# 启动网关服务（最后，依赖所有后端服务）
start_gateway() {
    print_header "启动网关服务"

    start_service "gateway-service" "dev:gateway" 8080
}

# 显示服务状态
show_status() {
    print_header "服务状态"

    echo -e "${YELLOW}核心域服务:${NC}"
    echo "  auth-service:     grpc://localhost:9001  logs/auth-service.log"
    echo "  user-service:     grpc://localhost:9002  logs/user-service.log"
    echo "  friend-service:   grpc://localhost:9003  logs/friend-service.log"
    echo "  group-service:    grpc://localhost:9004  logs/group-service.log"
    echo "  file-service:     grpc://localhost:9007  logs/file-service.log"

    echo -e "\n${YELLOW}应用层服务:${NC}"
    echo "  message-service:  grpc://localhost:9005  logs/message-service.log"
    echo "  session-service:  grpc://localhost:9006  logs/session-service.log"

    echo -e "\n${YELLOW}辅助服务:${NC}"
    echo "  push-service:     grpc://localhost:9008  logs/push-service.log"
    echo "  rtc-service:      grpc://localhost:9009  logs/rtc-service.log"
    echo "  sync-service:     grpc://localhost:9010  logs/sync-service.log"

    echo -e "\n${YELLOW}管理服务:${NC}"
    echo "  admin-service:    http://localhost:8011  logs/admin-service.log"
    echo "                    grpc://localhost:9011"

    echo -e "\n${YELLOW}网关服务:${NC}"
    echo "  gateway-service:  http://localhost:8080  logs/gateway-service.log"
    echo "  Swagger UI:       http://localhost:8080/swagger/index.html"

    echo -e "\n${YELLOW}停止所有服务:${NC}"
    echo "  ./scripts/stop-services.sh"

    echo -e "\n${YELLOW}查看实时日志（示例）:${NC}"
    echo "  tail -f logs/gateway-service.log"
    echo "  tail -f logs/message-service.log"
}

# 主函数
main() {
    echo -e "${GREEN}"
    echo "╔═══════════════════════════════════════════╗"
    echo "║   AnyChat 服务启动脚本                    ║"
    echo "╚═══════════════════════════════════════════╝"
    echo -e "${NC}"

    mkdir -p logs

    check_infrastructure
    check_migrations

    start_core_services
    sleep 1
    start_app_services
    sleep 1
    start_auxiliary_services
    sleep 1
    start_admin_service
    sleep 1
    start_gateway

    show_status

    echo -e "\n${GREEN}✓ 所有服务启动成功！${NC}\n"
}

main "$@"
