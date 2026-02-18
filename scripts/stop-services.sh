#!/bin/bash
#
# 停止所有微服务
#

# 颜色输出
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

echo -e "${YELLOW}正在停止所有微服务...${NC}\n"

# 停止所有微服务进程
pkill -f "auth-service|user-service|gateway-service|friend-service|group-service|file-service|message-service|session-service|push-service|rtc-service|sync-service|admin-service" 2>/dev/null || true

# 等待进程结束
sleep 2

# 检查是否还有进程
remaining=$(ps aux | grep -E "auth-service|user-service|friend-service|group-service|file-service|gateway-service|message-service|session-service|push-service|rtc-service|sync-service|admin-service" | grep -v grep || echo "")

if [ -z "$remaining" ]; then
    print_success "所有微服务已停止"
else
    print_info "仍有进程运行，强制停止..."
    pkill -9 -f "auth-service|user-service|friend-service|group-service|file-service|gateway-service|message-service|session-service|push-service|rtc-service|sync-service|admin-service" 2>/dev/null || true
    sleep 1
    print_success "强制停止完成"
fi

# 清理 PID 文件
rm -f \
    /tmp/auth-service.pid \
    /tmp/user-service.pid \
    /tmp/friend-service.pid \
    /tmp/group-service.pid \
    /tmp/file-service.pid \
    /tmp/message-service.pid \
    /tmp/session-service.pid \
    /tmp/push-service.pid \
    /tmp/rtc-service.pid \
    /tmp/sync-service.pid \
    /tmp/admin-service.pid \
    /tmp/gateway-service.pid \
    2>/dev/null

# 显示端口状态
echo ""
print_info "端口状态检查:"
for port in \
    9001 9002 9003 9004 9005 9006 9007 9008 9009 9010 9011 \
    8011 8080; do
    if lsof -i :$port 2>/dev/null | grep LISTEN > /dev/null; then
        echo "  端口 $port: 仍被占用"
    else
        echo "  端口 $port: 空闲 ✓"
    fi
done

echo -e "\n${GREEN}完成！${NC}"
