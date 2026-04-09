#!/bin/bash
#
# Port check and cleanup script
# Check port usage before starting services
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

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Check single port
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

# Check infrastructure ports
check_infrastructure_ports() {
    print_header "Checking Infrastructure Ports"

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

# Check microservice ports
check_microservice_ports() {
    print_header "Checking Microservice Ports"

    local failed=0

    # Core services
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
    check_port 8009 "calling-service HTTP" || ((failed++))
    check_port 9009 "calling-service gRPC" || ((failed++))
    check_port 9010 "sync-service gRPC" || ((failed++))
    check_port 8011 "admin-service HTTP" || ((failed++))
    check_port 9011 "admin-service gRPC" || ((failed++))

    return $failed
}

# Kill process on port
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

# Cleanup microservice processes
cleanup_microservices() {
    print_header "Cleaning Up Microservice Processes"

    echo "Stopping all microservices..."
    pkill -f "auth-service|user-service|gateway-service|friend-service|group-service|file-service|message-service" 2>/dev/null || true
    sleep 2

    # Check if any processes remain
    local remaining=$(ps aux | grep -E "auth-service|user-service|friend-service|group-service|file-service|gateway-service" | grep -v grep || echo "")
    if [ -z "$remaining" ]; then
        print_success "All microservices stopped"
    else
        print_warning "Some processes may still be running"
        echo "$remaining"
    fi
}

# Show port usage
show_port_usage() {
    print_header "Current Port Usage"

    echo -e "\n${YELLOW}Infrastructure Ports:${NC}"
    for port in 5432 6379 4222 8222 9000 9091 7880; do
        local result=$(lsof -i :$port 2>/dev/null | grep LISTEN || echo "")
        if [ -n "$result" ]; then
            echo "  $port: $(echo $result | awk '{print $1, "(PID:", $2")"}')"
        fi
    done

    echo -e "\n${YELLOW}Microservice Ports:${NC}"
    for port in 8080 8001 9001 8002 9002 8003 9003 8004 9004 8007 9007 8008 9008 8009 9009 9010 8011 9011; do
        local result=$(lsof -i :$port 2>/dev/null | grep LISTEN || echo "")
        if [ -n "$result" ]; then
            echo "  $port: $(echo $result | awk '{print $1, "(PID:", $2")"}')"
        fi
    done
}

# Main menu
show_menu() {
    echo -e "\n${BLUE}AnyChat Port Management Tool${NC}"
    echo "1) Check all ports"
    echo "2) Cleanup microservice processes"
    echo "3) Show port usage"
    echo "4) Stop process on specific port"
    echo "5) Full cleanup (stop microservices + Docker)"
    echo "0) Exit"
    echo -n "Select operation: "
}

# Main function
main() {
    if [ "$1" = "--check" ]; then
        # Check-only mode
        local failed=0
        check_infrastructure_ports || ((failed+=$?))
        check_microservice_ports || ((failed+=$?))

        if [ $failed -eq 0 ]; then
            echo -e "\n${GREEN}All ports available!${NC}"
            exit 0
        else
            echo -e "\n${RED}Found $failed port conflicts${NC}"
            exit 1
        fi
    elif [ "$1" = "--clean" ]; then
        # Cleanup mode
        cleanup_microservices
        exit 0
    elif [ "$1" = "--kill" ] && [ -n "$2" ]; then
        # Stop specific port
        kill_process_on_port $2
        exit 0
    elif [ "$1" = "--full-clean" ]; then
        # Full cleanup
        cleanup_microservices
        echo ""
        mage docker:down
        exit 0
    elif [ -z "$1" ]; then
        # Interactive mode
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
                    echo -n "Enter port number: "
                    read port
                    kill_process_on_port $port
                    ;;
                5)
                    cleanup_microservices
                    echo ""
                    mage docker:down
                    ;;
                0)
                    echo "Exiting"
                    exit 0
                    ;;
                *)
                    echo "Invalid choice"
                    ;;
            esac
        done
    else
        # Show help
        echo "Usage:"
        echo "  $0                    # Interactive mode"
        echo "  $0 --check            # Check ports only"
        echo "  $0 --clean            # Cleanup microservice processes"
        echo "  $0 --kill <port>      # Stop process on specific port"
        echo "  $0 --full-clean       # Full cleanup (microservices + Docker)"
        exit 0
    fi
}

# Run main function
main "$@"