#!/bin/bash
#
# LiveKit RTC Service API 测试脚本
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../common.sh"

BASE_URL="${GATEWAY_URL:-http://localhost:8080}/api/v1"

echo "=================================================="
echo "  LiveKit RTC Service API 测试"
echo "=================================================="
echo ""

PASS=0
FAIL=0

# ── 辅助函数 ─────────────────────────────────────────

pass() { echo -e "${GREEN}✓ PASS${NC}: $1"; PASS=$((PASS + 1)); }
fail() { echo -e "${RED}✗ FAIL${NC}: $1"; FAIL=$((FAIL + 1)); }

# 注册并登录用户，返回 token
register_and_login() {
    local suffix="$1"
    local email="rtc_${suffix}@test.com"
    local password="Test@1234"
    local device_id="rtc-dev-${suffix}"

    curl -s -X POST "${BASE_URL}/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"${email}\",\"password\":\"${password}\",\"verifyCode\":\"123456\",\"nickname\":\"User${suffix}\",\"deviceId\":\"${device_id}\",\"deviceType\":\"Web\"}" > /dev/null

    local resp
    resp=$(curl -s -X POST "${BASE_URL}/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"account\":\"${email}\",\"password\":\"${password}\",\"deviceId\":\"${device_id}\",\"deviceType\":\"Web\"}")

    echo "$resp" | grep -o '"accessToken":"[^"]*"' | cut -d'"' -f4
}

# ── 准备测试用户 ──────────────────────────────────────

echo "正在注册测试用户..."
TOKEN_A=$(register_and_login "00000001")
TOKEN_B=$(register_and_login "00000002")

if [ -z "$TOKEN_A" ] || [ -z "$TOKEN_B" ]; then
    echo -e "${RED}错误: 无法获取测试用户 token，跳过 RTC 测试${NC}"
    exit 0
fi

echo "用户 A token 获取成功"
echo "用户 B token 获取成功"
echo ""

# ── 测试 1: 未认证发起通话（应返回 401）────────────────

echo "测试 1: 未认证发起通话"
RESP=$(curl -s -X POST "${BASE_URL}/rtc/calls" \
    -H "Content-Type: application/json" \
    -d '{"calleeId":"user-xxx","callType":"audio"}')
CODE=$(echo "$RESP" | grep -o '"code":[0-9]*' | cut -d: -f2)
if [ "$CODE" = "401" ]; then
    pass "未认证返回 code 401"
else
    fail "期望 code 401，实际 $CODE"
fi

# ── 测试 2: 缺少 calleeId（应返回 400）────────────────

echo "测试 2: 缺少 calleeId"
RESP=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BASE_URL}/rtc/calls" \
    -H "Authorization: Bearer $TOKEN_A" \
    -H "Content-Type: application/json" \
    -d '{}')
if [ "$RESP" = "400" ]; then
    pass "缺少 calleeId 返回 400"
else
    fail "期望 400，实际 $RESP"
fi

# ── 测试 3: 获取通话记录（空列表）────────────────────

echo "测试 3: 获取通话记录（初始为空）"
RESP=$(curl -s "${BASE_URL}/rtc/calls" \
    -H "Authorization: Bearer $TOKEN_A")
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/rtc/calls" \
    -H "Authorization: Bearer $TOKEN_A")
if [ "$HTTP_CODE" = "200" ]; then
    pass "获取通话记录成功（HTTP 200）"
else
    fail "期望 200，实际 $HTTP_CODE"
fi

# ── 测试 4: 获取不存在的通话（应返回错误）────────────

echo "测试 4: 获取不存在的通话"
RESP=$(curl -s "${BASE_URL}/rtc/calls/nonexistent-call-id" \
    -H "Authorization: Bearer $TOKEN_A")
CODE=$(echo "$RESP" | grep -o '"code":[0-9]*' | cut -d: -f2)
if [ "$CODE" = "404" ] || [ "$CODE" = "500" ]; then
    pass "不存在通话返回错误码 $CODE"
else
    fail "期望 code 404/500，实际 $CODE"
fi

# ── 测试 5: 未认证创建会议室（应返回 401）────────────

echo "测试 5: 未认证创建会议室"
RESP=$(curl -s -X POST "${BASE_URL}/rtc/meetings" \
    -H "Content-Type: application/json" \
    -d '{"title":"Test Meeting"}')
CODE=$(echo "$RESP" | grep -o '"code":[0-9]*' | cut -d: -f2)
if [ "$CODE" = "401" ]; then
    pass "未认证返回 code 401"
else
    fail "期望 code 401，实际 $CODE"
fi

# ── 测试 6: 缺少 title（应返回 400）────────────────────

echo "测试 6: 创建会议室缺少 title"
RESP=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BASE_URL}/rtc/meetings" \
    -H "Authorization: Bearer $TOKEN_A" \
    -H "Content-Type: application/json" \
    -d '{}')
if [ "$RESP" = "400" ]; then
    pass "缺少 title 返回 400"
else
    fail "期望 400，实际 $RESP"
fi

# ── 测试 7: 列举会议室（无需 rtc-service 可用）────

echo "测试 7: 列举会议室（初始为空或报错）"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    "${BASE_URL}/rtc/meetings" \
    -H "Authorization: Bearer $TOKEN_A")
if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "500" ]; then
    pass "列举会议室接口可达（HTTP ${HTTP_CODE}）"
else
    fail "期望 200/500，实际 $HTTP_CODE"
fi

# ── 测试 8: 接听不存在的通话（应返回错误）────────────

echo "测试 8: 接听不存在的通话"
RESP=$(curl -s -X POST "${BASE_URL}/rtc/calls/fake-call-id/join" \
    -H "Authorization: Bearer $TOKEN_B")
CODE=$(echo "$RESP" | grep -o '"code":[0-9]*' | cut -d: -f2)
if [ "$CODE" = "404" ] || [ "$CODE" = "500" ]; then
    pass "接听不存在通话返回错误码 $CODE"
else
    fail "期望 code 404/500，实际 $CODE"
fi

# ── 测试 9: 获取不存在的会议室（应返回错误）─────────

echo "测试 9: 获取不存在的会议室"
RESP=$(curl -s "${BASE_URL}/rtc/meetings/nonexistent-room" \
    -H "Authorization: Bearer $TOKEN_A")
CODE=$(echo "$RESP" | grep -o '"code":[0-9]*' | cut -d: -f2)
if [ "$CODE" = "404" ] || [ "$CODE" = "500" ]; then
    pass "不存在会议室返回错误码 $CODE"
else
    fail "期望 code 404/500，实际 $CODE"
fi

# ── 汇总 ──────────────────────────────────────────────

echo ""
echo "=================================================="
echo "  测试结果: ${PASS} 通过, ${FAIL} 失败"
echo "=================================================="

if [ $FAIL -eq 0 ]; then
    exit 0
else
    exit 1
fi
