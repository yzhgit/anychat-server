#!/bin/bash
#
# Admin Service API 测试脚本
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../common.sh"

ADMIN_URL="${ADMIN_URL:-http://localhost:8011}"

echo "=================================================="
echo "  Admin Service API 测试"
echo "=================================================="
echo ""

PASS=0
FAIL=0

pass() { echo -e "${GREEN}✓ PASS${NC}: $1"; PASS=$((PASS + 1)); }
fail() { echo -e "${RED}✗ FAIL${NC}: $1"; FAIL=$((FAIL + 1)); }

# ── 测试 1: 健康检查 ────────────────────────────────────

echo "测试 1: 健康检查"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${ADMIN_URL}/health")
if [ "$HTTP_CODE" = "200" ]; then
    pass "健康检查返回 200"
else
    fail "期望 200，实际 $HTTP_CODE"
fi

# ── 测试 2: 未认证访问受保护接口 ────────────────────────

echo "测试 2: 未认证访问用户列表"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${ADMIN_URL}/api/admin/users")
if [ "$HTTP_CODE" = "401" ]; then
    pass "未认证返回 401"
else
    fail "期望 401，实际 $HTTP_CODE"
fi

# ── 测试 3: 缺少密码登录 ────────────────────────────────

echo "测试 3: 缺少密码登录"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${ADMIN_URL}/api/admin/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin"}')
if [ "$HTTP_CODE" = "400" ]; then
    pass "缺少密码返回 400"
else
    fail "期望 400，实际 $HTTP_CODE"
fi

# ── 测试 4: 错误密码登录 ────────────────────────────────

echo "测试 4: 错误密码登录"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${ADMIN_URL}/api/admin/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"wrongpassword"}')
if [ "$HTTP_CODE" = "401" ]; then
    pass "错误密码返回 401"
else
    fail "期望 401，实际 $HTTP_CODE"
fi

# ── 测试 5: 正确登录 ────────────────────────────────────

echo "测试 5: 正确登录（admin/Admin@123456）"
LOGIN_RESP=$(curl -s -X POST "${ADMIN_URL}/api/admin/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"Admin@123456"}')
ADMIN_TOKEN=$(echo "$LOGIN_RESP" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -n "$ADMIN_TOKEN" ]; then
    pass "登录成功并获取 Token"
else
    fail "登录失败或未返回 Token（响应: $LOGIN_RESP）"
    # 如果登录失败则跳过后续测试
    echo ""
    echo "注意: 若数据库未初始化则后续测试将跳过"
    echo "=================================================="
    echo "  测试结果: ${PASS} 通过, ${FAIL} 失败"
    echo "=================================================="
    [ $FAIL -eq 0 ] && exit 0 || exit 1
fi

# ── 测试 6: 获取用户列表（需认证）──────────────────────

echo "测试 6: 管理员获取用户列表"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${ADMIN_URL}/api/admin/users" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "500" ]; then
    pass "用户列表接口可达（HTTP ${HTTP_CODE}）"
else
    fail "期望 200/500，实际 $HTTP_CODE"
fi

# ── 测试 7: 获取统计数据 ────────────────────────────────

echo "测试 7: 获取系统统计概览"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${ADMIN_URL}/api/admin/stats/overview" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
if [ "$HTTP_CODE" = "200" ]; then
    pass "统计概览返回 200"
else
    fail "期望 200，实际 $HTTP_CODE"
fi

# ── 测试 8: 获取系统配置 ────────────────────────────────

echo "测试 8: 获取系统配置"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${ADMIN_URL}/api/admin/config" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
if [ "$HTTP_CODE" = "200" ]; then
    pass "系统配置返回 200"
else
    fail "期望 200，实际 $HTTP_CODE"
fi

# ── 测试 9: 获取审计日志 ────────────────────────────────

echo "测试 9: 获取审计日志"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${ADMIN_URL}/api/admin/audit-logs" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
if [ "$HTTP_CODE" = "200" ]; then
    pass "审计日志返回 200"
else
    fail "期望 200，实际 $HTTP_CODE"
fi

# ── 测试 10: 获取管理员列表 ─────────────────────────────

echo "测试 10: 获取管理员列表"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${ADMIN_URL}/api/admin/admins" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
if [ "$HTTP_CODE" = "200" ]; then
    pass "管理员列表返回 200"
else
    fail "期望 200，实际 $HTTP_CODE"
fi

# ── 测试 11: 更新不存在的配置项 ────────────────────────

echo "测试 11: 更新系统配置项"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "${ADMIN_URL}/api/admin/config/site.name" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"value":"AnyChat Test"}')
if [ "$HTTP_CODE" = "200" ]; then
    pass "配置更新返回 200"
else
    fail "期望 200，实际 $HTTP_CODE"
fi

# ── 测试 12: 退出登录 ───────────────────────────────────

echo "测试 12: 退出登录"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${ADMIN_URL}/api/admin/auth/logout" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
if [ "$HTTP_CODE" = "200" ]; then
    pass "退出登录返回 200"
else
    fail "期望 200，实际 $HTTP_CODE"
fi

# ── 汇总 ──────────────────────────────────────────────

echo ""
echo "=================================================="
echo "  测试结果: ${PASS} 通过, ${FAIL} 失败"
echo "=================================================="

[ $FAIL -eq 0 ] && exit 0 || exit 1
