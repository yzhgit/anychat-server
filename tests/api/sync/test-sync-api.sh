#!/bin/bash
#
# Sync Service HTTP API 测试脚本
# 用于测试数据同步相关的 HTTP 接口
#
# 用法:
#   ./test-sync-api.sh
#   GATEWAY_URL=http://localhost:8080 ./test-sync-api.sh
#
# 说明:
#   同步服务是无状态聚合层，通过调用 friend/group/session/message 服务
#   获取增量数据。本脚本验证 API 的响应结构及错误处理，空账号状态下
#   全量同步应当返回空集合而非错误。
#

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 配置
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
API_BASE="${GATEWAY_URL}/api/v1"

# 测试数据
TIMESTAMP=$(date +%s)
TEST_EMAIL="sync_u1_${TIMESTAMP}@example.com"
TEST_PASSWORD="Test@123456"
TEST_DEVICE_ID="sync-test-device-${TIMESTAMP}"

# 全局变量
USER_TOKEN=""
USER_ID=""
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

check_json_field() {
    local desc="$1"
    local value="$2"
    local expected="$3"

    if [ "$value" = "$expected" ]; then
        pass "$desc"
    else
        fail "$desc" "期望 '$expected'，实际 '$value'"
    fi
}

# ────────────────────────────────────────
# 注册并登录用户
# ────────────────────────────────────────

setup_user() {
    print_header "初始化测试用户"

    # 注册
    local reg_resp
    reg_resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\",\"verifyCode\":\"123456\",\"nickname\":\"SyncTestUser\",\"deviceId\":\"${TEST_DEVICE_ID}\",\"deviceType\":\"Web\"}")
    local reg_status
    reg_status=$(echo "$reg_resp" | tail -1)
    local reg_body
    reg_body=$(echo "$reg_resp" | head -n -1)

    if [ "$reg_status" != "200" ]; then
        echo -e "${RED}注册用户失败 (HTTP $reg_status): $reg_body${NC}"
        exit 1
    fi

    # 登录
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

test_full_sync_empty_account() {
    print_header "测试1: 全量同步（空账号）"

    local resp
    resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/sync" \
        -H "Authorization: Bearer ${USER_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{"lastSyncTime":0,"conversationSeqs":[]}')
    local status
    status=$(echo "$resp" | tail -1)
    local body
    body=$(echo "$resp" | head -n -1)

    check_http_status "全量同步返回200" "200" "$status" "$body"

    local code
    code=$(echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code','-1'))" 2>/dev/null || \
           echo "$body" | grep -o '"code":[0-9]*' | head -1 | cut -d: -f2)
    check_json_field "响应code为0" "$code" "0"

    local sync_time
    sync_time=$(echo "$body" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('sync_time',''))" 2>/dev/null || \
                echo "$body" | grep -o '"sync_time":[0-9]*' | head -1 | cut -d: -f2)
    if [ -n "$sync_time" ] && [ "$sync_time" != "0" ]; then
        pass "响应包含 sync_time"
    else
        fail "响应包含 sync_time" "sync_time 为空或0，body: $body"
    fi
}

test_incremental_sync() {
    print_header "测试2: 增量同步（带 lastSyncTime）"

    local last_sync=$((TIMESTAMP - 3600))
    local resp
    resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/sync" \
        -H "Authorization: Bearer ${USER_TOKEN}" \
        -H "Content-Type: application/json" \
        -d "{\"lastSyncTime\":${last_sync},\"conversationSeqs\":[]}")
    local status
    status=$(echo "$resp" | tail -1)
    local body
    body=$(echo "$resp" | head -n -1)

    check_http_status "增量同步返回200" "200" "$status" "$body"

    local code
    code=$(echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code','-1'))" 2>/dev/null || \
           echo "$body" | grep -o '"code":[0-9]*' | head -1 | cut -d: -f2)
    check_json_field "响应code为0" "$code" "0"
}

test_sync_without_auth() {
    print_header "测试3: 未认证同步"

    local body
    body=$(curl -s -X POST "${API_BASE}/sync" \
        -H "Content-Type: application/json" \
        -d '{"lastSyncTime":0}')
    local code
    code=$(echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code','-1'))" 2>/dev/null || \
           echo "$body" | grep -o '"code":[0-9]*' | head -1 | cut -d: -f2)

    check_json_field "未认证返回code 401" "$code" "401"
}

test_sync_messages_empty() {
    print_header "测试4: 消息补齐（空会话列表）"

    local resp
    resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/sync/messages" \
        -H "Authorization: Bearer ${USER_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{"conversationSeqs":[],"limitPerConversation":20}')
    local status
    status=$(echo "$resp" | tail -1)
    local body
    body=$(echo "$resp" | head -n -1)

    check_http_status "消息补齐（空列表）返回200" "200" "$status" "$body"

    local code
    code=$(echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code','-1'))" 2>/dev/null || \
           echo "$body" | grep -o '"code":[0-9]*' | head -1 | cut -d: -f2)
    check_json_field "响应code为0" "$code" "0"
}

test_sync_messages_without_auth() {
    print_header "测试5: 未认证消息补齐"

    local body
    body=$(curl -s -X POST "${API_BASE}/sync/messages" \
        -H "Content-Type: application/json" \
        -d '{"conversationSeqs":[]}')
    local code
    code=$(echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code','-1'))" 2>/dev/null || \
           echo "$body" | grep -o '"code":[0-9]*' | head -1 | cut -d: -f2)

    check_json_field "未认证返回code 401" "$code" "401"
}

test_sync_with_nonexistent_conversation() {
    print_header "测试6: 消息补齐（不存在的会话）"

    local resp
    resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/sync/messages" \
        -H "Authorization: Bearer ${USER_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{"conversationSeqs":[{"conversationId":"nonexistent-conv-id","conversationType":"single","lastSeq":0}],"limitPerConversation":20}')
    local status
    status=$(echo "$resp" | tail -1)
    local body
    body=$(echo "$resp" | head -n -1)

    # 不存在的会话：同步服务会跳过并返回空列表，不应报错
    check_http_status "不存在会话时仍返回200" "200" "$status" "$body"
}

test_sync_with_limit_query_param() {
    print_header "测试7: 消息补齐（通过 query param 指定 limit）"

    local resp
    resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/sync/messages?limit=10" \
        -H "Authorization: Bearer ${USER_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{"conversationSeqs":[]}')
    local status
    status=$(echo "$resp" | tail -1)
    local body
    body=$(echo "$resp" | head -n -1)

    check_http_status "带 limit 参数的消息补齐返回200" "200" "$status" "$body"
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
        echo -e "${GREEN}所有同步服务测试通过!${NC}"
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
echo -e "${GREEN}║   Sync Service HTTP API 测试              ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}"
echo "Gateway: ${GATEWAY_URL}"
echo "时间: $(date '+%Y-%m-%d %H:%M:%S')"

setup_user

test_full_sync_empty_account
test_incremental_sync
test_sync_without_auth
test_sync_messages_empty
test_sync_messages_without_auth
test_sync_with_nonexistent_conversation
test_sync_with_limit_query_param

print_summary
