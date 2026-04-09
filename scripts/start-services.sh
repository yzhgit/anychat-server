#!/bin/bash
#
# Start all microservices
# Starts all services in dependency order
#

set -e

# Color output
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

# Check if port is available
check_port_available() {
    local port=$1
    local result=$(lsof -i :$port 2>/dev/null | grep LISTEN || echo "")
    if [ -z "$result" ]; then
        return 0
    else
        return 1
    fi
}

# Wait for service to start (check gRPC or HTTP port)
wait_for_service() {
    local port=$1
    local service=$2
    local max_attempts=30
    local attempt=0

    print_info "Waiting for $service to start (port $port)..."

    while [ $attempt -lt $max_attempts ]; do
        if lsof -i :$port 2>/dev/null | grep LISTEN > /dev/null; then
            print_success "$service started"
            return 0
        fi
        sleep 1
        ((attempt++))
        echo -n "."
    done

    echo ""
    print_error "$service start timeout"
    return 1
}

# Generic function to start a single service
# Usage: start_service <service_name> <mage_target> <check_port> <pid_file>
start_service() {
    local name=$1
    local mage_target=$2
    local port=$3
    local pid_file="/tmp/${name}.pid"

    if check_port_available "$port"; then
        print_info "Starting ${name}..."
        nohup mage "$mage_target" > "logs/${name}.log" 2>&1 &
        local pid=$!
        echo $pid > "$pid_file"

        if wait_for_service "$port" "$name"; then
            print_success "${name} running on PID: $pid"
        else
            print_error "${name} failed to start, check logs: logs/${name}.log"
            exit 1
        fi
    else
        print_success "${name} is already running"
    fi
}

# Check infrastructure
check_infrastructure() {
    print_header "Checking Infrastructure"

    local failed=0

    if ! docker ps | grep anychat-postgres | grep -q "healthy\|Up"; then
        print_error "PostgreSQL not running, please run: mage docker:up"
        ((failed++))
    else
        print_success "PostgreSQL is running"
    fi

    if ! docker ps | grep anychat-redis | grep -q "healthy\|Up"; then
        print_error "Redis not running, please run: mage docker:up"
        ((failed++))
    else
        print_success "Redis is running"
    fi

    if ! docker ps | grep anychat-nats | grep -q "Up"; then
        print_error "NATS not running, please run: mage docker:up"
        ((failed++))
    else
        print_success "NATS is running"
    fi

    if ! docker ps | grep anychat-minio | grep -q "healthy\|Up"; then
        print_error "MinIO not running, please run: mage docker:up"
        ((failed++))
    else
        print_success "MinIO is running"
    fi

    if ! docker ps | grep anychat-livekit | grep -q "Up"; then
        print_error "LiveKit not running, please run: mage docker:up"
        ((failed++))
    else
        print_success "LiveKit is running"
    fi

    if [ $failed -gt 0 ]; then
        echo ""
        print_error "Infrastructure not ready, please start: mage docker:up"
        exit 1
    fi

    echo ""
    print_success "All infrastructure ready"
}

# Check database migrations
check_migrations() {
    print_header "Checking Database Migrations"

    if docker exec anychat-postgres psql -U anychat -d anychat -c "\dt" 2>/dev/null | grep -q users; then
        print_success "Database migrations completed"
    else
        print_info "Database migrations not completed, running migrations..."
        mage db:up
        print_success "Database migrations completed"
    fi
}

# Start core domain services (first layer, no inter-dependencies)
start_core_services() {
    print_header "Starting Core Domain Services"

    start_service "auth-service"    "dev:auth"    9001
    start_service "user-service"    "dev:user"    9002
    start_service "friend-service"  "dev:friend"  9003
    start_service "group-service"   "dev:group"   9004
    start_service "file-service"    "dev:file"    9007
}

# Start application layer services (second layer, message/conversation)
start_app_services() {
    print_header "Starting Application Layer Services"

    start_service "message-service" "dev:message"  9005
    start_service "conversation-service" "dev:conversation"  9006
}

# Start auxiliary services (third layer, push/Calling/sync)
start_auxiliary_services() {
    print_header "Starting Auxiliary Services"

    start_service "push-service"    "dev:push"    9008
    start_service "calling-service" "dev:calling" 9009
    start_service "sync-service"    "dev:sync"    9010
}

# Start admin service
start_admin_service() {
    print_header "Starting Admin Service"

    start_service "admin-service"   "dev:admin"   9011
}

# Start version service
start_version_service() {
    print_header "Starting Version Service"

    start_service "version-service" "dev:version" 9012
}

# Start gateway service (last, depends on all backend services)
start_gateway() {
    print_header "Starting Gateway Service"

    start_service "gateway-service" "dev:gateway" 8080
}

# Show service status
show_status() {
    print_header "Service Status"

    echo -e "${YELLOW}Core Domain Services:${NC}"
    echo "  auth-service:     grpc://localhost:9001  logs/auth-service.log"
    echo "  user-service:     grpc://localhost:9002  logs/user-service.log"
    echo "  friend-service:   grpc://localhost:9003  logs/friend-service.log"
    echo "  group-service:    grpc://localhost:9004  logs/group-service.log"
    echo "  file-service:     grpc://localhost:9007  logs/file-service.log"

    echo -e "\n${YELLOW}Application Layer Services:${NC}"
    echo "  message-service:  grpc://localhost:9005  logs/message-service.log"
    echo "  conversation-service:  grpc://localhost:9006  logs/conversation-service.log"

    echo -e "\n${YELLOW}Auxiliary Services:${NC}"
    echo "  push-service:     grpc://localhost:9008  logs/push-service.log"
    echo "  calling-service: grpc://localhost:9009  logs/calling-service.log"
    echo "  sync-service:     grpc://localhost:9010  logs/sync-service.log"

    echo -e "\n${YELLOW}Admin Services:${NC}"
    echo "  admin-service:    http://localhost:8011  logs/admin-service.log"
    echo "                    grpc://localhost:9011"

    echo -e "\n${YELLOW}Version Services:${NC}"
    echo "  version-service: grpc://localhost:9012  logs/version-service.log"

    echo -e "\n${YELLOW}Gateway Service:${NC}"
    echo "  gateway-service:  http://localhost:8080  logs/gateway-service.log"
    echo "  Swagger UI:       http://localhost:8080/swagger/index.html"

    echo -e "\n${YELLOW}Stop All Services:${NC}"
    echo "  ./scripts/stop-services.sh"

    echo -e "\n${YELLOW}View Real-time Logs (example):${NC}"
    echo "  tail -f logs/gateway-service.log"
    echo "  tail -f logs/message-service.log"
}

# Main function
main() {
    echo -e "${GREEN}"
    echo "╔═══════════════════════════════════════════╗"
    echo "║   AnyChat Service Startup Script          ║"
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
    start_version_service
    sleep 1
    start_gateway

    show_status

    echo -e "\n${GREEN}✓ All services started successfully!${NC}\n"
}

main "$@"