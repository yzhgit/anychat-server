#!/bin/bash
#
# 运行所有 HTTP API 测试的入口脚本
#

set -e

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}╔═══════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   AnyChat HTTP API 测试套件               ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════╝${NC}"
echo ""
echo "测试环境: ${GATEWAY_URL:-http://localhost:8080}"
echo "开始时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo ""

FAILED=0

# 运行Auth Service测试
echo -e "${YELLOW}[1/10] 运行 Auth Service API 测试...${NC}"
if "${SCRIPT_DIR}/auth/test-auth-api.sh"; then
    echo -e "${GREEN}✓ Auth Service 测试通过${NC}"
else
    echo -e "${RED}✗ Auth Service 测试失败${NC}"
    ((FAILED++))
fi
echo ""

# 运行User Service测试
echo -e "${YELLOW}[2/10] 运行 User Service API 测试...${NC}"
if "${SCRIPT_DIR}/user/test-user-api.sh"; then
    echo -e "${GREEN}✓ User Service 测试通过${NC}"
else
    echo -e "${RED}✗ User Service 测试失败${NC}"
    ((FAILED++))
fi
echo ""

# 运行Friend Service测试
echo -e "${YELLOW}[3/10] 运行 Friend Service API 测试...${NC}"
if "${SCRIPT_DIR}/friend/test-friend-api.sh"; then
    echo -e "${GREEN}✓ Friend Service 测试通过${NC}"
else
    echo -e "${RED}✗ Friend Service 测试失败${NC}"
    ((FAILED++))
fi
echo ""

# 运行Group Service测试
echo -e "${YELLOW}[4/10] 运行 Group Service API 测试...${NC}"
if "${SCRIPT_DIR}/group/test-group-api.sh"; then
    echo -e "${GREEN}✓ Group Service 测试通过${NC}"
else
    echo -e "${RED}✗ Group Service 测试失败${NC}"
    ((FAILED++))
fi
echo ""

# 运行File Service测试
echo -e "${YELLOW}[5/10] 运行 File Service API 测试...${NC}"
if "${SCRIPT_DIR}/file/test-file-api.sh"; then
    echo -e "${GREEN}✓ File Service 测试通过${NC}"
else
    echo -e "${RED}✗ File Service 测试失败${NC}"
    ((FAILED++))
fi
echo ""

# 运行Session Service测试
echo -e "${YELLOW}[6/10] 运行 Session Service API 测试...${NC}"
if "${SCRIPT_DIR}/session/test-session-api.sh"; then
    echo -e "${GREEN}✓ Session Service 测试通过${NC}"
else
    echo -e "${RED}✗ Session Service 测试失败${NC}"
    ((FAILED++))
fi
echo ""

# 运行Sync Service测试
echo -e "${YELLOW}[7/10] 运行 Sync Service API 测试...${NC}"
if "${SCRIPT_DIR}/sync/test-sync-api.sh"; then
    echo -e "${GREEN}✓ Sync Service 测试通过${NC}"
else
    echo -e "${RED}✗ Sync Service 测试失败${NC}"
    ((FAILED++))
fi
echo ""

# # 运行Push Service测试
# echo -e "${YELLOW}[8/10] 运行 Push Service API 测试...${NC}"
# if "${SCRIPT_DIR}/push/test-push-api.sh"; then
#     echo -e "${GREEN}✓ Push Service 测试通过${NC}"
# else
#     echo -e "${RED}✗ Push Service 测试失败${NC}"
#     ((FAILED++))
# fi
# echo ""

# 运行RTC Service测试
echo -e "${YELLOW}[9/10] 运行 RTC Service API 测试...${NC}"
if "${SCRIPT_DIR}/rtc/test-rtc-api.sh"; then
    echo -e "${GREEN}✓ RTC Service 测试通过${NC}"
else
    echo -e "${RED}✗ RTC Service 测试失败${NC}"
    ((FAILED++))
fi
echo ""

# 运行Admin Service测试
echo -e "${YELLOW}[10/10] 运行 Admin Service API 测试...${NC}"
if ADMIN_URL="${ADMIN_URL:-http://localhost:8011}" "${SCRIPT_DIR}/admin/test-admin-api.sh"; then
    echo -e "${GREEN}✓ Admin Service 测试通过${NC}"
else
    echo -e "${RED}✗ Admin Service 测试失败${NC}"
    ((FAILED++))
fi
echo ""

# 输出总结
echo -e "${YELLOW}═══════════════════════════════════════════${NC}"
echo "结束时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}╔═══════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║   所有测试通过! ✓                          ║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════╝${NC}"
    exit 0
else
    echo -e "${RED}╔═══════════════════════════════════════════╗${NC}"
    echo -e "${RED}║   失败测试数: ${FAILED}                          ║${NC}"
    echo -e "${RED}╚═══════════════════════════════════════════╝${NC}"
    exit 1
fi
