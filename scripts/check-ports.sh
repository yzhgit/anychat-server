#!/bin/bash
#
# 端口检查和清理脚本
# 在启动服务前检查端口占用情况
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

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# 检查单个端口
check_port() {
    local port=$1
    local service=$2
    local result=$(lsof -i :$port 2>/dev/null | grep LISTEN || echo "")

    if [ -z "$result" ]; then
        print_success "Port $port ($service) is available"
        return 0
    else
        local pid=$(echo "$result" | awk '{print $2}' | head -1)
        local cmd=$(echo "$result" | awk '{print $1}' | head -1)
        print_error "Port $port ($service) is in use by $cmd (PID: $pid)"
        return 1
    fi
}

# 检查基础设施端口
check_infrastructure_ports() {
    print_header "检查基础设施端口"

    local failed=0

    check_port 5432 "PostgreSQL" || ((failed++))
    check_port 6379 "Redis" || ((failed++))
    check_port 4222 "NATS Client" || ((failed++))
    check_port 8222 "NATS Monitoring" || ((failed++))
    check_port 9000 "MinIO API" || ((failed++))
    check_port 9091 "MinIO Console" || ((failed++))
    check_port 7880 "LiveKit WebSocket/API" || ((failed++))

    return $failed
}

# 检查微服务端口
check_microservice_ports() {
    print_header "检查微服务端口"

    local failed=0

    # 核心服务
    check_port 8080 "gateway-service HTTP" || ((failed++))
    check_port 8001 "auth-service HTTP" || ((failed++))
    check_port 9001 "auth-service gRPC" || ((failed++))
    check_port 8002 "user-service HTTP" || ((failed++))
    check_port 9002 "user-service gRPC" || ((failed++))
    check_port 8003 "friend-service HTTP" || ((failed++))
    check_port 9003 "friend-service gRPC" || ((failed++))
    check_port 8004 "group-service HTTP" || ((failed++))
    check_port 9004 "group-service gRPC" || ((failed++))
    check_port 8007 "file-service HTTP" || ((failed++))
    check_port 9007 "file-service gRPC" || ((failed++))
    check_port 8008 "push-service HTTP" || ((failed++))
    check_port 9008 "push-service gRPC" || ((failed++))
    check_port 8009 "rtc-service HTTP" || ((failed++))
    check_port 9009 "rtc-service gRPC" || ((failed++))
    check_port 9010 "sync-service gRPC" || ((failed++))
    check_port 8011 "admin-service HTTP" || ((failed++))
    check_port 9011 "admin-service gRPC" || ((failed++))

    return $failed
}

# 停止占用端口的进程
kill_process_on_port() {
    local port=$1
    local pids=$(lsof -ti :$port 2>/dev/null || echo "")

    if [ -n "$pids" ]; then
        echo "Killing processes on port $port: $pids"
        kill -9 $pids 2>/dev/null
        sleep 1
        return 0
    else
        echo "No process found on port $port"
        return 1
    fi
}

# 清理微服务进程
cleanup_microservices() {
    print_header "清理微服务进程"

    echo "Stopping all microservices..."
    pkill -f "auth-service|user-service|gateway-service|friend-service|group-service|file-service|message-service" 2>/dev/null || true
    sleep 2

    # 检查是否还有进程
    local remaining=$(ps aux | grep -E "auth-service|user-service|friend-service|group-service|file-service|gateway-service" | grep -v grep || echo "")
    if [ -z "$remaining" ]; then
        print_success "All microservices stopped"
    else
        print_warning "Some processes may still be running"
        echo "$remaining"
    fi
}

# 显示端口使用情况
show_port_usage() {
    print_header "当前端口使用情况"

    echo -e "\n${YELLOW}基础设施端口:${NC}"
    for port in 5432 6379 4222 8222 9000 9091 7880; do
        local result=$(lsof -i :$port 2>/dev/null | grep LISTEN || echo "")
        if [ -n "$result" ]; then
            echo "  $port: $(echo $result | awk '{print $1, "(PID:", $2")"}')"
        fi
    done

    echo -e "\n${YELLOW}微服务端口:${NC}"
    for port in 8080 8001 9001 8002 9002 8003 9003 8004 9004 8007 9007 8008 9008 8009 9009 9010 8011 9011; do
        local result=$(lsof -i :$port 2>/dev/null | grep LISTEN || echo "")
        if [ -n "$result" ]; then
            echo "  $port: $(echo $result | awk '{print $1, "(PID:", $2")"}')"
        fi
    done
}

# 主菜单
show_menu() {
    echo -e "\n${BLUE}AnyChat 端口管理工具${NC}"
    echo "1) 检查所有端口"
    echo "2) 清理微服务进程"
    echo "3) 显示端口使用情况"
    echo "4) 停止特定端口的进程"
    echo "5) 完整清理 (停止微服务 + Docker)"
    echo "0) 退出"
    echo -n "选择操作: "
}

# 主函数
main() {
    if [ "$1" = "--check" ]; then
        # 仅检查模式
        local failed=0
        check_infrastructure_ports || ((failed+=$?))
        check_microservice_ports || ((failed+=$?))

        if [ $failed -eq 0 ]; then
            echo -e "\n${GREEN}所有端口可用！${NC}"
            exit 0
        else
            echo -e "\n${RED}发现 $failed 个端口冲突${NC}"
            exit 1
        fi
    elif [ "$1" = "--clean" ]; then
        # 清理模式
        cleanup_microservices
        exit 0
    elif [ "$1" = "--kill" ] && [ -n "$2" ]; then
        # 停止特定端口
        kill_process_on_port $2
        exit 0
    elif [ "$1" = "--full-clean" ]; then
        # 完整清理
        cleanup_microservices
        echo ""
        mage docker:down
        exit 0
    elif [ -z "$1" ]; then
        # 交互模式
        while true; do
            show_menu
            read choice

            case $choice in
                1)
                    check_infrastructure_ports
                    check_microservice_ports
                    ;;
                2)
                    cleanup_microservices
                    ;;
                3)
                    show_port_usage
                    ;;
                4)
                    echo -n "输入端口号: "
                    read port
                    kill_process_on_port $port
                    ;;
                5)
                    cleanup_microservices
                    echo ""
                    mage docker:down
                    ;;
                0)
                    echo "退出"
                    exit 0
                    ;;
                *)
                    echo "无效选择"
                    ;;
            esac
        done
    else
        # 显示帮助
        echo "用法:"
        echo "  $0                    # 交互模式"
        echo "  $0 --check            # 仅检查端口"
        echo "  $0 --clean            # 清理微服务进程"
        echo "  $0 --kill <port>      # 停止特定端口的进程"
        echo "  $0 --full-clean       # 完整清理 (微服务 + Docker)"
        exit 0
    fi
}

# 运行主函数
main "$@"
