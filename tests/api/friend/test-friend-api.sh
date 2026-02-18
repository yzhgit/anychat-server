#!/bin/bash
#
# Friend Service HTTP API 测试脚本
# 用于测试好友管理相关的 HTTP 接口
#

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 配置
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
API_BASE="${GATEWAY_URL}/api/v1"

# 测试数据
TIMESTAMP=$(date +%s)
TEST_PHONE_1="138${TIMESTAMP:(-8)}"
TEST_PHONE_2="139${TIMESTAMP:(-8)}"
TEST_EMAIL_1="user1_${TIMESTAMP}@example.com"
TEST_EMAIL_2="user2_${TIMESTAMP}@example.com"
TEST_PASSWORD="Test@123456"
TEST_DEVICE_ID="test-device-${TIMESTAMP}"

# 全局变量
USER1_TOKEN=""
USER2_TOKEN=""
USER1_ID=""
USER2_ID=""
FRIEND_REQUEST_ID=""

# 打印函数
print_header() {
    echo -e "\n${YELLOW}========================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "  $1"
}

# HTTP 请求函数
http_post() {
    local url=$1
    local data=$2
    local token=$3

    if [ -n "$token" ]; then
        curl -s -X POST "${url}" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer ${token}" \
            -d "${data}"
    else
        curl -s -X POST "${url}" \
            -H "Content-Type: application/json" \
            -d "${data}"
    fi
}

http_get() {
    local url=$1
    local token=$2

    if [ -n "$token" ]; then
        curl -s -X GET "${url}" \
            -H "Authorization: Bearer ${token}"
    else
        curl -s -X GET "${url}"
    fi
}

http_put() {
    local url=$1
    local data=$2
    local token=$3

    curl -s -X PUT "${url}" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${token}" \
        -d "${data}"
}

http_delete() {
    local url=$1
    local token=$2

    curl -s -X DELETE "${url}" \
        -H "Authorization: Bearer ${token}"
}

# 检查 JSON 响应中的 code 字段
check_response() {
    local response=$1
    local code=$(echo "$response" | jq -r '.code // -1')

    if [ "$code" = "0" ]; then
        return 0
    else
        local message=$(echo "$response" | jq -r '.message // "Unknown error"')
        print_error "API Error: $message (code: $code)"
        return 1
    fi
}

# ========================================
# 准备工作：创建测试用户
# ========================================

setup_test_users() {
    print_header "准备测试用户"

    # 注册用户1
    print_info "注册用户1: ${TEST_EMAIL_1}"
    local data1=$(cat <<EOF
{
    "email": "${TEST_EMAIL_1}",
    "password": "${TEST_PASSWORD}",
    "verifyCode": "123456",
    "nickname": "测试用户1_${TIMESTAMP}",
    "deviceType": "iOS",
    "deviceId": "${TEST_DEVICE_ID}_1"
}
EOF
)
    local response1=$(http_post "${API_BASE}/auth/register" "$data1")
    if check_response "$response1"; then
        USER1_ID=$(echo "$response1" | jq -r '.data.userId')
        USER1_TOKEN=$(echo "$response1" | jq -r '.data.accessToken')
        print_success "用户1注册成功 (ID: ${USER1_ID})"
    else
        print_error "用户1注册失败"
        return 1
    fi

    sleep 1

    # 注册用户2
    print_info "注册用户2: ${TEST_EMAIL_2}"
    local data2=$(cat <<EOF
{
    "email": "${TEST_EMAIL_2}",
    "password": "${TEST_PASSWORD}",
    "verifyCode": "123456",
    "nickname": "测试用户2_${TIMESTAMP}",
    "deviceType": "iOS",
    "deviceId": "${TEST_DEVICE_ID}_2"
}
EOF
)
    local response2=$(http_post "${API_BASE}/auth/register" "$data2")
    if check_response "$response2"; then
        USER2_ID=$(echo "$response2" | jq -r '.data.userId')
        USER2_TOKEN=$(echo "$response2" | jq -r '.data.accessToken')
        print_success "用户2注册成功 (ID: ${USER2_ID})"
    else
        print_error "用户2注册失败"
        return 1
    fi
}

# ========================================
# 测试用例
# ========================================

# 1. 发送好友申请
test_send_friend_request() {
    print_header "1. 发送好友申请"

    local data=$(cat <<EOF
{
    "userId": "${USER2_ID}",
    "message": "你好，我想加你为好友",
    "source": "search"
}
EOF
)

    print_info "用户1向用户2发送好友申请"

    local response=$(http_post "${API_BASE}/friends/requests" "$data" "$USER1_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        # 注意：protobuf 的 request_id 在 JSON 中是 requestId (驼峰)
        FRIEND_REQUEST_ID=$(echo "$response" | jq -r '.data.requestId // .data.request_id // empty')
        local auto_accepted=$(echo "$response" | jq -r '.data.autoAccepted // .data.auto_accepted // false')

        if [ -z "$FRIEND_REQUEST_ID" ] || [ "$FRIEND_REQUEST_ID" = "null" ]; then
            print_error "无法获取申请ID，响应数据: $(echo "$response" | jq -r '.data')"
            return 1
        fi

        print_success "发送好友申请成功"
        print_info "申请ID: ${FRIEND_REQUEST_ID}"
        print_info "自动接受: ${auto_accepted}"
        return 0
    else
        return 1
    fi
}

# 2. 获取收到的好友申请
test_get_received_requests() {
    print_header "2. 获取收到的好友申请"

    print_info "用户2获取收到的好友申请"

    local response=$(http_get "${API_BASE}/friends/requests?type=received" "$USER2_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        local total=$(echo "$response" | jq -r '.data.total // 0')
        print_success "获取好友申请成功"
        print_info "收到 ${total} 个好友申请"

        # 备用方案：如果之前没有获取到 FRIEND_REQUEST_ID，从列表中提取
        if [ -z "$FRIEND_REQUEST_ID" ] || [ "$FRIEND_REQUEST_ID" = "null" ]; then
            FRIEND_REQUEST_ID=$(echo "$response" | jq -r '.data.requests[0].id // empty')
            if [ -n "$FRIEND_REQUEST_ID" ] && [ "$FRIEND_REQUEST_ID" != "null" ]; then
                print_info "从申请列表中获取到申请ID: ${FRIEND_REQUEST_ID}"
            fi
        fi

        return 0
    else
        return 1
    fi
}

# 3. 获取发送的好友申请
test_get_sent_requests() {
    print_header "3. 获取发送的好友申请"

    print_info "用户1获取发送的好友申请"

    local response=$(http_get "${API_BASE}/friends/requests?type=sent" "$USER1_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        local total=$(echo "$response" | jq -r '.data.total // 0')
        print_success "获取发送的好友申请成功"
        print_info "发送了 ${total} 个好友申请"
        return 0
    else
        return 1
    fi
}

# 4. 接受好友申请
test_accept_friend_request() {
    print_header "4. 接受好友申请"

    # 检查 FRIEND_REQUEST_ID 是否有效
    if [ -z "$FRIEND_REQUEST_ID" ] || [ "$FRIEND_REQUEST_ID" = "null" ]; then
        print_error "申请ID无效，跳过此测试"
        print_info "提示：请确保前面的测试成功执行"
        return 1
    fi

    local data=$(cat <<EOF
{
    "action": "accept"
}
EOF
)

    print_info "用户2接受好友申请 (ID: ${FRIEND_REQUEST_ID})"

    local response=$(http_put "${API_BASE}/friends/requests/${FRIEND_REQUEST_ID}" "$data" "$USER2_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        print_success "接受好友申请成功"
        return 0
    else
        return 1
    fi
}

# 5. 获取好友列表
test_get_friend_list() {
    print_header "5. 获取好友列表"

    # 用户1获取好友列表
    print_info "用户1获取好友列表"
    local response1=$(http_get "${API_BASE}/friends" "$USER1_TOKEN")
    print_info "响应: $response1"

    if check_response "$response1"; then
        local total1=$(echo "$response1" | jq -r '.data.total // 0')
        print_success "用户1获取好友列表成功 (共 ${total1} 个好友)"
    else
        return 1
    fi

    # 用户2获取好友列表
    print_info "用户2获取好友列表"
    local response2=$(http_get "${API_BASE}/friends" "$USER2_TOKEN")
    print_info "响应: $response2"

    if check_response "$response2"; then
        local total2=$(echo "$response2" | jq -r '.data.total // 0')
        print_success "用户2获取好友列表成功 (共 ${total2} 个好友)"
        return 0
    else
        return 1
    fi
}

# 6. 更新好友备注
test_update_friend_remark() {
    print_header "6. 更新好友备注"

    local data=$(cat <<EOF
{
    "remark": "我的好朋友"
}
EOF
)

    print_info "用户1更新用户2的备注"

    local response=$(http_put "${API_BASE}/friends/${USER2_ID}/remark" "$data" "$USER1_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        print_success "更新好友备注成功"
        return 0
    else
        return 1
    fi
}

# 7. 增量同步好友列表
test_incremental_sync() {
    print_header "7. 增量同步好友列表"

    # 使用过去的时间戳（5分钟前）来测试增量同步
    # 这样可以捕获刚才创建的好友关系
    local last_time=$(($(date +%s) - 300))
    print_info "使用时间戳进行增量同步: ${last_time}"

    local response=$(http_get "${API_BASE}/friends?lastUpdateTime=${last_time}" "$USER1_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        local total=$(echo "$response" | jq -r '.data.total // 0')
        print_success "增量同步成功"
        print_info "更新了 ${total} 个好友"
        return 0
    else
        return 1
    fi
}

# 8. 添加到黑名单
test_add_to_blacklist() {
    print_header "8. 添加到黑名单"

    local data=$(cat <<EOF
{
    "userId": "${USER2_ID}"
}
EOF
)

    print_info "用户1将用户2添加到黑名单"

    local response=$(http_post "${API_BASE}/friends/blacklist" "$data" "$USER1_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        print_success "添加到黑名单成功"
        return 0
    else
        return 1
    fi
}

# 9. 获取黑名单
test_get_blacklist() {
    print_header "9. 获取黑名单"

    print_info "用户1获取黑名单"

    local response=$(http_get "${API_BASE}/friends/blacklist" "$USER1_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        local total=$(echo "$response" | jq -r '.data.total // 0')
        print_success "获取黑名单成功"
        print_info "黑名单中有 ${total} 个用户"
        return 0
    else
        return 1
    fi
}

# 10. 从黑名单移除
test_remove_from_blacklist() {
    print_header "10. 从黑名单移除"

    print_info "用户1将用户2从黑名单移除"

    local response=$(http_delete "${API_BASE}/friends/blacklist/${USER2_ID}" "$USER1_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        print_success "从黑名单移除成功"
        return 0
    else
        return 1
    fi
}

# 11. 删除好友
test_delete_friend() {
    print_header "11. 删除好友"

    print_info "用户1删除用户2"

    local response=$(http_delete "${API_BASE}/friends/${USER2_ID}" "$USER1_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        print_success "删除好友成功"
        return 0
    else
        return 1
    fi
}

# 12. 验证好友已删除
test_verify_friend_deleted() {
    print_header "12. 验证好友已删除"

    print_info "用户1获取好友列表（应为空）"

    local response=$(http_get "${API_BASE}/friends" "$USER1_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        local total=$(echo "$response" | jq -r '.data.total // 0')
        if [ "$total" -eq 0 ]; then
            print_success "验证成功：好友列表为空"
            return 0
        else
            print_error "验证失败：好友列表不为空 (total: ${total})"
            return 1
        fi
    else
        return 1
    fi
}

# ========================================
# 主函数
# ========================================

main() {
    echo -e "${GREEN}"
    echo "╔═══════════════════════════════════════════╗"
    echo "║   Friend Service API 测试脚本             ║"
    echo "╚═══════════════════════════════════════════╝"
    echo -e "${NC}"

    echo "测试环境: ${GATEWAY_URL}"
    echo "开始时间: $(date '+%Y-%m-%d %H:%M:%S')"
    echo ""

    # 检查依赖
    if ! command -v jq &> /dev/null; then
        print_error "需要安装 jq 工具: apt-get install jq 或 brew install jq"
        exit 1
    fi

    # 准备测试用户
    setup_test_users || exit 1

    # 执行测试
    local failed=0

    test_send_friend_request || ((failed++))
    sleep 1
    test_get_received_requests || ((failed++))
    test_get_sent_requests || ((failed++))
    test_accept_friend_request || ((failed++))
    sleep 1
    test_get_friend_list || ((failed++))
    test_update_friend_remark || ((failed++))
    sleep 1
    test_incremental_sync || ((failed++))
    test_add_to_blacklist || ((failed++))
    test_get_blacklist || ((failed++))
    test_remove_from_blacklist || ((failed++))
    sleep 1
    test_delete_friend || ((failed++))
    test_verify_friend_deleted || ((failed++))

    # 输出测试结果
    echo ""
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}测试结果${NC}"
    echo -e "${YELLOW}========================================${NC}"
    echo "结束时间: $(date '+%Y-%m-%d %H:%M:%S')"

    if [ $failed -eq 0 ]; then
        echo -e "${GREEN}所有测试通过! ✓${NC}"
        exit 0
    else
        echo -e "${RED}失败测试数: ${failed} ✗${NC}"
        exit 1
    fi
}

# 运行主函数
main "$@"
