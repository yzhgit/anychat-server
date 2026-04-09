#!/bin/bash
#
# Service health check script
# Check health status of all microservices
#

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Service configuration
declare -A SERVICES=(
    ["Auth Service"]="http://localhost:8001/health"
    ["User Service"]="http://localhost:8002/health"
    ["Friend Service"]="http://localhost:8003/health"
    ["Gateway Service"]="http://localhost:8080/health"
)

# gRPC service ports
declare -A GRPC_PORTS=(
    ["Auth gRPC"]="9001"
    ["User gRPC"]="9002"
    ["Friend gRPC"]="9003"
)

# Infrastructure services
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
    echo -e "\n${BLUE}========================================${NC}"
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

# Check HTTP health endpoint
check_http_health() {
    local name=$1
    local url=$2

    local response=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")

    if [ "$response" = "200" ]; then
        print_success "$name - healthy (HTTP $response)"
        return 0
    else
        print_error "$name - unhealthy (HTTP $response)"
        return 1
    fi
}

# Check port
check_port() {
    local name=$1
    local port=$2

    if nc -z localhost $port 2>/dev/null; then
        print_success "$name - port $port listening"
        return 0
    else
        print_error "$name - port $port not listening"
        return 1
    fi
}

# Check Docker container
check_container() {
    local name=$1

    if docker ps --format '{{.Names}}' | grep -q "$name"; then
        local status=$(docker ps --filter "name=$name" --format '{{.Status}}')
        print_success "Docker container $name - $status"
        return 0
    else
        print_error "Docker container $name - not running"
        return 1
    fi
}

# Main function
main() {
    echo -e "${GREEN}"
    echo "╔═══════════════════════════════════════════╗"
    echo "║   AnyChat Service Health Check           ║"
    echo "╚═══════════════════════════════════════════╝"
    echo -e "${NC}"

    echo "Check time: $(date '+%Y-%m-%d %H:%M:%S')"

    local total=0
    local healthy=0

    # 1. Check infrastructure
    print_header "Infrastructure Services"
    for service in "${!INFRA[@]}"; do
        ((total++))
        if check_port "$service" "${INFRA[$service]}"; then
            ((healthy++))
        fi
    done

    # 2. Check HTTP services
    print_header "HTTP Services"
    for service in "${!SERVICES[@]}"; do
        ((total++))
        if check_http_health "$service" "${SERVICES[$service]}"; then
            ((healthy++))
        fi
    done

    # 3. Check gRPC services
    print_header "gRPC Services"
    for service in "${!GRPC_PORTS[@]}"; do
        ((total++))
        if check_port "$service" "${GRPC_PORTS[$service]}"; then
            ((healthy++))
        fi
    done

    # 4. Check Docker containers
    if command -v docker &> /dev/null; then
        print_header "Docker Containers"
        local containers=("postgres" "redis" "nats" "minio")
        for container in "${containers[@]}"; do
            ((total++))
            if check_container "$container"; then
                ((healthy++))
            fi
        done
    fi

    # Output summary
    echo ""
    print_header "Health Check Summary"

    local percentage=$((healthy * 100 / total))

    echo "Healthy services: $healthy / $total ($percentage%)"

    if [ $healthy -eq $total ]; then
        echo -e "${GREEN}✓ All services healthy!${NC}"
        exit 0
    elif [ $healthy -ge $((total * 2 / 3)) ]; then
        echo -e "${YELLOW}⚠ Some services unhealthy${NC}"
        exit 1
    else
        echo -e "${RED}✗ Most services unhealthy${NC}"
        exit 2
    fi
}

# Run main function
main "$@"