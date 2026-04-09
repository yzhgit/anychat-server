#!/bin/bash
#
# Environment check and setup script
# Check and configure AnyChat development environment
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
    echo -e "\n${BLUE}========================================${NC}"
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

# Check if command exists
check_command() {
    local cmd=$1
    local name=$2
    local install_hint=$3

    if command -v $cmd &> /dev/null; then
        local version=$($cmd --version 2>&1 | head -n 1)
        print_success "$name installed: $version"
        return 0
    else
        print_error "$name not installed"
        if [ -n "$install_hint" ]; then
            print_info "Install hint: $install_hint"
        fi
        return 1
    fi
}

# Compare version numbers (semantic versioning, compare major.minor.patch only)
# Returns 0 if $1 >= $2, returns 1 if $1 < $2
version_gte() {
    local actual=$1
    local required=$2

    # Split version into array
    IFS='.' read -r -a actual_parts <<< "$actual"
    IFS='.' read -r -a required_parts <<< "$required"

    for i in 0 1 2; do
        local a=${actual_parts[$i]:-0}
        local r=${required_parts[$i]:-0}
        if [ "$a" -gt "$r" ]; then return 0; fi
        if [ "$a" -lt "$r" ]; then return 1; fi
    done
    return 0
}

# Check Docker version (>= 28.1.1)
check_docker_version() {
    local required="28.1.1"

    if ! command -v docker &> /dev/null; then
        print_error "Docker not installed"
        print_info "Install hint: https://docs.docker.com/get-docker/"
        return 1
    fi

    local raw
    raw=$(docker --version 2>&1)
    # Format: "Docker version 24.0.5, build ced0996"
    local ver
    ver=$(echo "$raw" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)

    if version_gte "$ver" "$required"; then
        print_success "Docker installed: $raw"
        return 0
    else
        print_error "Docker version too low: $ver (required >= $required)"
        print_info "Upgrade Docker: https://docs.docker.com/get-docker/"
        return 1
    fi
}

# Check Docker Compose version (>= 2.35.1)
check_docker_compose_version() {
    local required="2.35.1"

    # Prioritize detection of docker compose (v2 plugin form)
    if docker compose version &> /dev/null; then
        local raw
        raw=$(docker compose version 2>&1)
        local ver
        ver=$(echo "$raw" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
        if version_gte "$ver" "$required"; then
            print_success "Docker Compose installed: $raw"
            return 0
        else
            print_error "Docker Compose version too low: $ver (required >= $required)"
            return 1
        fi
    fi

    # Fallback to standalone docker-compose command
    if command -v docker-compose &> /dev/null; then
        local raw
        raw=$(docker-compose --version 2>&1)
        local ver
        ver=$(echo "$raw" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
        if version_gte "$ver" "$required"; then
            print_success "Docker Compose installed: $raw"
            return 0
        else
            print_error "Docker Compose version too low: $ver (required >= $required)"
            print_info "Upgrade Docker Compose: https://docs.docker.com/compose/install/"
            return 1
        fi
    fi

    print_error "Docker Compose not installed"
    print_info "Install hint: https://docs.docker.com/compose/install/"
    return 1
}

# Check protoc version (>= 3.12.4, and supports proto3 optional)
check_protoc_version() {
    local required="3.12.4"

    if ! command -v protoc &> /dev/null; then
        print_warning "protoc not installed, cannot generate proto code"
        print_info "Install hint: https://grpc.io/docs/protoc-installation/"
        print_info "Required version: >= $required (supports --experimental_allow_proto3_optional)"
        return 1
    fi

    local raw
    raw=$(protoc --version 2>&1)
    # Format: "libprotoc 3.12.4"
    local ver
    ver=$(echo "$raw" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)

    if version_gte "$ver" "$required"; then
        print_success "protoc installed: $raw"
        return 0
    else
        print_error "protoc version too low: $ver (required >= $required, must support --experimental_allow_proto3_optional)"
        print_info "Upgrade protoc: https://grpc.io/docs/protoc-installation/"
        return 1
    fi
}

# Check service port
check_port() {
    local port=$1
    local service=$2

    if nc -z localhost $port 2>/dev/null; then
        print_success "$service (port $port) is running"
        return 0
    else
        print_warning "$service (port $port) is not running"
        return 1
    fi
}

# Check Docker container
check_docker_container() {
    local container=$1
    local service=$2

    if docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
        print_success "$service container is running"
        return 0
    else
        print_warning "$service container is not running"
        return 1
    fi
}

# ========================================
# Check development tools
# ========================================

check_dev_tools() {
    print_header "Checking Development Tools"

    local failed=0

    # Go
    check_command go "Go" "https://go.dev/doc/install" || ((failed++))

    # Docker (with version check)
    check_docker_version || ((failed++))

    # Docker Compose (with version check)
    check_docker_compose_version || ((failed++))

    # jq (for JSON processing)
    check_command jq "jq" "apt-get install jq or brew install jq" || ((failed++))

    # curl
    check_command curl "curl" "apt-get install curl or brew install curl" || ((failed++))

    # netcat (for port checking)
    if ! command -v nc &> /dev/null; then
        print_warning "nc (netcat) not installed, cannot check ports"
    else
        print_success "nc (netcat) installed"
    fi

    # protoc (with version check)
    check_protoc_version || ((failed++))

    # mage
    if command -v mage &> /dev/null; then
        print_success "Mage installed"
    else
        print_warning "Mage not installed"
        print_info "Install command: go install github.com/magefile/mage@latest"
    fi

    return $failed
}

# ========================================
# Check infrastructure services
# ========================================

check_infrastructure() {
    print_header "Checking Infrastructure Services"

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
# Check microservices
# ========================================

check_microservices() {
    print_header "Checking Microservice Status"

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

    print_info "Running services: $running / $total"

    if [ $running -eq $total ]; then
        return 0
    else
        return 1
    fi
}

# ========================================
# Check database connection
# ========================================

check_database() {
    print_header "Checking Database Connection"

    if ! command -v psql &> /dev/null; then
        print_warning "psql not installed, skipping database check"
        return 0
    fi

    # Check database connection
    if PGPASSWORD=anychat123 psql -h localhost -U anychat -d anychat -c "SELECT 1" &> /dev/null; then
        print_success "Database connection successful"

        # Check if tables exist
        local tables=$(PGPASSWORD=anychat123 psql -h localhost -U anychat -d anychat -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'")
        print_info "Database table count: $(echo $tables | xargs)"

        return 0
    else
        print_error "Database connection failed"
        return 1
    fi
}

# ========================================
# Go module check
# ========================================

check_go_modules() {
    print_header "Checking Go Modules"

    if [ ! -f "go.mod" ]; then
        print_error "go.mod file does not exist"
        return 1
    fi

    print_success "go.mod file exists"

    # Check dependencies
    print_info "Checking Go dependencies..."
    if go mod verify &> /dev/null; then
        print_success "Go module verification passed"
    else
        print_warning "Go module verification failed, try running go mod tidy"
    fi

    return 0
}

# ========================================
# Environment variable check
# ========================================

check_environment_variables() {
    print_header "Checking Environment Variables"

    local vars=(
        "GOPATH:Go workspace path"
        "GOPROXY:Go module proxy"
    )

    for var_info in "${vars[@]}"; do
        local var=$(echo $var_info | cut -d: -f1)
        local desc=$(echo $var_info | cut -d: -f2)

        if [ -n "${!var}" ]; then
            print_success "$var ($desc): ${!var}"
        else
            print_info "$var not set"
        fi
    done

    return 0
}

# ========================================
# Generate environment report
# ========================================

generate_report() {
    print_header "Environment Check Summary"

    echo ""
    echo "System Information:"
    print_info "Operating System: $(uname -s)"
    print_info "Architecture: $(uname -m)"

    if [ -f /etc/os-release ]; then
        source /etc/os-release
        print_info "Distribution: $NAME $VERSION"
    fi

    echo ""
    echo "Check Time: $(date '+%Y-%m-%d %H:%M:%S')"
}

# ========================================
# Quick fix suggestions
# ========================================

suggest_fixes() {
    print_header "Quick Fix Suggestions"

    echo ""
    echo "If infrastructure services are not running, run:"
    echo "  ${GREEN}mage docker:up${NC}"
    echo ""
    echo "If you need to run database migrations, run:"
    echo "  ${GREEN}mage db:up${NC}"
    echo ""
    echo "Start microservices:"
    echo "  ${GREEN}mage dev:auth${NC}      # Terminal 1: auth-service"
    echo "  ${GREEN}mage dev:user${NC}      # Terminal 2: user-service"
    echo "  ${GREEN}mage dev:friend${NC}    # Terminal 3: friend-service"
    echo "  ${GREEN}mage dev:gateway${NC}   # Terminal 4: gateway-service"
    echo ""
    echo "Run full tests:"
    echo "  ${GREEN}./scripts/test-all.sh${NC}"
    echo ""
}

# ========================================
# Main function
# ========================================

main() {
    echo -e "${GREEN}"
    echo "╔═══════════════════════════════════════════╗"
    echo "║   AnyChat Environment Check Script       ║"
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
    # Microservices may not be running, don't count as failure

    generate_report

    if [ $total_failed -eq 0 ]; then
        echo ""
        echo -e "${GREEN}✓ All environment checks passed!${NC}"
        echo ""
        exit 0
    else
        suggest_fixes
        echo ""
        echo -e "${YELLOW}⚠ Found $total_failed issues, please fix according to the suggestions above${NC}"
        echo ""
        exit 1
    fi
}

# Run main function
main "$@"