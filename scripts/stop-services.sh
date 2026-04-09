#!/bin/bash
#
# Stop all microservices
#

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_info() {
    echo -e "${YELLOW}➜ $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

echo -e "${YELLOW}Stopping all microservices...${NC}\n"

# Stop all microservice processes
pkill -f "auth-service|user-service|gateway-service|friend-service|group-service|file-service|message-service|conversation-service|push-service|calling-service|sync-service|admin-service|version-service" 2>/dev/null || true

# Wait for processes to end
sleep 2

# Check if any processes remain
remaining=$(ps aux | grep -E "auth-service|user-service|friend-service|group-service|file-service|gateway-service|message-service|conversation-service|push-service|calling-service|sync-service|admin-service|version-service" | grep -v grep || echo "")

if [ -z "$remaining" ]; then
    print_success "All microservices stopped"
else
    print_info "Some processes still running, forcing stop..."
    pkill -9 -f "auth-service|user-service|friend-service|group-service|file-service|gateway-service|message-service|conversation-service|push-service|calling-service|sync-service|admin-service|version-service" 2>/dev/null || true
    sleep 1
    print_success "Force stop completed"
fi

# Clean up PID files
rm -f \
    /tmp/auth-service.pid \
    /tmp/user-service.pid \
    /tmp/friend-service.pid \
    /tmp/group-service.pid \
    /tmp/file-service.pid \
    /tmp/message-service.pid \
    /tmp/conversation-service.pid \
    /tmp/push-service.pid \
    /tmp/calling-service.pid \
    /tmp/sync-service.pid \
    /tmp/admin-service.pid \
    /tmp/version-service.pid \
    /tmp/gateway-service.pid \
    2>/dev/null

# Show port status
echo ""
print_info "Port status check:"
for port in \
    9001 9002 9003 9004 9005 9006 9007 9008 9009 9010 9011 9012 \
    8080; do
    if lsof -i :$port 2>/dev/null | grep LISTEN > /dev/null; then
        echo "  Port $port: still in use"
    else
        echo "  Port $port: free ✓"
    fi
done

echo -e "\n${GREEN}Done!${NC}"