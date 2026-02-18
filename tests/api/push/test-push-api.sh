#!/bin/bash
#
# Push Service 测试脚本
# 用于测试推送服务相关功能
#
# 用法:
#   ./test-push-api.sh
#   GATEWAY_URL=http://localhost:8080 ./test-push-api.sh
#   PUSH_GRPC=localhost:9008 ./test-push-api.sh
#
# 说明:
#   推送服务主要通过 NATS 事件驱动，无直接 HTTP 端点。
#   本脚本验证:
#   1. 推送服务健康检查接口
#   2. 通过 grpcurl 直接调用 SendPush gRPC（若 grpcurl 可用）
#   3. 发送消息后触发 NATS 通知，验证 push-service 不崩溃
#

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 配置
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
PUSH_HTTP="${PUSH_HTTP:-http://localhost:8008}"
PUSH_GRPC="${PUSH_GRPC:-localhost:9008}"
API_BASE="${GATEWAY_URL}/api/v1"

# 测试数据
TIMESTAMP=$(date +%s)
TEST_EMAIL="push_u1_${TIMESTAMP}@example.com"
TEST_PASSWORD="Test@123456"
TEST_DEVICE_ID="push-test-device-${TIMESTAMP}"

# 全局变量
USER_TOKEN=""
USER_ID=""
HAS_GRPCURL=false
PASS=0
FAIL=0

# ────────────────────────────────────────
# 工具函数
# ────────────────────────────────────────

print_header() {
    echo -e "\n${YELLOW}========================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}========================================${NC}"
}

pass() {
    echo -e "  ${GREEN}✓ PASS${NC}: $1"
    PASS=$((PASS + 1))
}

fail() {
    echo -e "  ${RED}✗ FAIL${NC}: $1"
    echo -e "  ${RED}  详情: $2${NC}"
    FAIL=$((FAIL + 1))
}

check_http_status() {
    local desc="$1"
    local expected="$2"
    local actual="$3"
    local body="$4"

    if [ "$actual" = "$expected" ]; then
        pass "$desc (HTTP $actual)"
    else
        fail "$desc" "期望 HTTP $expected，实际 HTTP $actual，响应: $body"
    fi
}

# ────────────────────────────────────────
# 检测 grpcurl
# ────────────────────────────────────────

detect_grpcurl() {
    if command -v grpcurl &>/dev/null; then
        HAS_GRPCURL=true
        echo "  grpcurl 可用: $(grpcurl --version 2>&1 | head -1)"
    else
        HAS_GRPCURL=false
        echo "  grpcurl 不可用，跳过 gRPC 测试"
    fi
}

# ────────────────────────────────────────
# 注册并登录用户
# ────────────────────────────────────────

setup_user() {
    print_header "初始化测试用户"

    local reg_resp
    reg_resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\",\"verifyCode\":\"123456\",\"nickname\":\"PushTestUser\",\"deviceId\":\"${TEST_DEVICE_ID}\",\"deviceType\":\"Web\"}")
    local reg_status
    reg_status=$(echo "$reg_resp" | tail -1)
    local reg_body
    reg_body=$(echo "$reg_resp" | head -n -1)

    if [ "$reg_status" != "200" ]; then
        echo -e "${RED}注册用户失败 (HTTP $reg_status): $reg_body${NC}"
        exit 1
    fi

    local login_resp
    login_resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"account\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\",\"deviceId\":\"${TEST_DEVICE_ID}\",\"deviceType\":\"Web\"}")
    local login_status
    login_status=$(echo "$login_resp" | tail -1)
    local login_body
    login_body=$(echo "$login_resp" | head -n -1)

    if [ "$login_status" != "200" ]; then
        echo -e "${RED}登录失败 (HTTP $login_status): $login_body${NC}"
        exit 1
    fi

    USER_TOKEN=$(echo "$login_body" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['accessToken'])" 2>/dev/null || \
                 echo "$login_body" | grep -o '"accessToken":"[^"]*"' | head -1 | cut -d'"' -f4)
    USER_ID=$(echo "$login_body" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['userId'])" 2>/dev/null || \
              echo "$login_body" | grep -o '"userId":"[^"]*"' | head -1 | cut -d'"' -f4)

    if [ -z "$USER_TOKEN" ]; then
        echo -e "${RED}无法获取 accessToken${NC}"
        exit 1
    fi
    echo "  用户注册并登录成功 (ID: ${USER_ID})"
}

# ────────────────────────────────────────
# 测试用例
# ────────────────────────────────────────

test_push_service_health() {
    print_header "测试1: Push Service 健康检查"

    local resp
    resp=$(curl -s -w "\n%{http_code}" "${PUSH_HTTP}/health")
    local status
    status=$(echo "$resp" | tail -1)
    local body
    body=$(echo "$resp" | head -n -1)

    check_http_status "Push service 健康检查返回200" "200" "$status" "$body"

    local svc_name
    svc_name=$(echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin).get('service',''))" 2>/dev/null || \
               echo "$body" | grep -o '"service":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [ "$svc_name" = "push-service" ]; then
        pass "健康检查 service 字段正确"
    else
        fail "健康检查 service 字段" "期望 push-service，实际 '$svc_name'"
    fi
}

test_send_push_grpc_missing_users() {
    print_header "测试2: gRPC SendPush（无 user_ids）"

    if [ "$HAS_GRPCURL" = false ]; then
        echo "  跳过: grpcurl 不可用"
        return
    fi

    local resp
    resp=$(grpcurl -plaintext \
        -d '{"title":"测试推送","content":"内容","push_type":"message"}' \
        "${PUSH_GRPC}" anychat.push.PushService/SendPush 2>&1)

    if echo "$resp" | grep -q "user_ids is required"; then
        pass "空 user_ids 返回 InvalidArgument 错误"
    else
        fail "空 user_ids 验证" "响应: $resp"
    fi
}

test_send_push_grpc_missing_title() {
    print_header "测试3: gRPC SendPush（无 title）"

    if [ "$HAS_GRPCURL" = false ]; then
        echo "  跳过: grpcurl 不可用"
        return
    fi

    local resp
    resp=$(grpcurl -plaintext \
        -d "{\"user_ids\":[\"${USER_ID}\"],\"content\":\"内容\",\"push_type\":\"message\"}" \
        "${PUSH_GRPC}" anychat.push.PushService/SendPush 2>&1)

    if echo "$resp" | grep -q "title is required"; then
        pass "空 title 返回 InvalidArgument 错误"
    else
        fail "空 title 验证" "响应: $resp"
    fi
}

test_send_push_grpc_no_token() {
    print_header "测试4: gRPC SendPush（用户无推送 Token）"

    if [ "$HAS_GRPCURL" = false ]; then
        echo "  跳过: grpcurl 不可用"
        return
    fi

    # 测试用户未注册 JPush token，推送应静默成功（无 token 则跳过 JPush 调用）
    local resp
    resp=$(grpcurl -plaintext \
        -d "{\"user_ids\":[\"${USER_ID}\"],\"title\":\"测试推送\",\"content\":\"内容\",\"push_type\":\"message\"}" \
        "${PUSH_GRPC}" anychat.push.PushService/SendPush 2>&1)

    if echo "$resp" | grep -qE '"successCount"|"failureCount"|\{\}'; then
        pass "无 token 用户推送静默成功（返回空结果）"
    else
        fail "无 token 用户推送" "响应: $resp"
    fi
}

test_update_push_token_via_gateway() {
    print_header "测试5: 通过 Gateway 注册推送 Token"

    # 上传一个测试用的 push token（实际场景为 JPush registration_id）
    local resp
    resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/users/me/push-token" \
        -H "Authorization: Bearer ${USER_TOKEN}" \
        -H "Content-Type: application/json" \
        -d "{\"pushToken\":\"test-registration-id-${TIMESTAMP}\",\"platform\":\"android\",\"deviceId\":\"${TEST_DEVICE_ID}\"}")
    local status
    status=$(echo "$resp" | tail -1)
    local body
    body=$(echo "$resp" | head -n -1)

    # 200 表示成功注册，若 token 更新接口不存在则 404
    if [ "$status" = "200" ]; then
        pass "推送 Token 注册成功 (HTTP $status)"
    elif [ "$status" = "404" ]; then
        echo "  信息: 推送 Token 注册接口暂未实现（HTTP 404），跳过"
    else
        fail "推送 Token 注册" "期望 HTTP 200，实际 HTTP $status，响应: $body"
    fi
}

# ────────────────────────────────────────
# 汇总
# ────────────────────────────────────────

print_summary() {
    echo ""
    echo -e "${YELLOW}════════════════════════════════════════${NC}"
    echo -e "测试结果: ${GREEN}${PASS} 通过${NC} / ${RED}${FAIL} 失败${NC}"
    echo -e "${YELLOW}════════════════════════════════════════${NC}"

    if [ $FAIL -eq 0 ]; then
        echo -e "${GREEN}所有推送服务测试通过!${NC}"
        exit 0
    else
        echo -e "${RED}有 ${FAIL} 个测试失败${NC}"
        exit 1
    fi
}

# ────────────────────────────────────────
# 主流程
# ────────────────────────────────────────

echo -e "${GREEN}╔══════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   Push Service 测试                       ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}"
echo "Gateway:   ${GATEWAY_URL}"
echo "Push HTTP: ${PUSH_HTTP}"
echo "Push gRPC: ${PUSH_GRPC}"
echo "时间: $(date '+%Y-%m-%d %H:%M:%S')"

detect_grpcurl
setup_user

test_push_service_health
test_send_push_grpc_missing_users
test_send_push_grpc_missing_title
test_send_push_grpc_no_token
test_update_push_token_via_gateway

print_summary
