#!/bin/bash
#
# Group Service HTTP API 测试脚本
# 用于测试群组管理相关的 HTTP 接口
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
TEST_PHONE_3="137${TIMESTAMP:(-8)}"
TEST_EMAIL_1="user1_${TIMESTAMP}@example.com"
TEST_EMAIL_2="user2_${TIMESTAMP}@example.com"
TEST_EMAIL_3="user3_${TIMESTAMP}@example.com"
TEST_PASSWORD="Test@123456"
TEST_DEVICE_ID="test-device-${TIMESTAMP}"
GROUP_NAME="测试群组_${TIMESTAMP}"

# 全局变量
USER1_TOKEN=""
USER2_TOKEN=""
USER3_TOKEN=""
USER1_ID=""
USER2_ID=""
USER3_ID=""
GROUP_ID=""
JOIN_REQUEST_ID=""
TEST_PASSED=0
TEST_FAILED=0

# 打印函数
print_header() {
    echo -e "\n${YELLOW}========================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
    TEST_PASSED=$((TEST_PASSED + 1))
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
    TEST_FAILED=$((TEST_FAILED + 1))
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

# 检查 JSON 响应状态
check_response() {
    local response=$1
    local expected_code=$2
    local test_name=$3

    local code=$(echo "$response" | jq -r '.code // empty')

    if [ "$code" = "$expected_code" ] || [ "$code" = "0" ]; then
        print_success "$test_name"
        return 0
    else
        print_error "$test_name - Expected code $expected_code, got: $code"
        print_info "Response: $response"
        return 1
    fi
}

# 前置准备：创建测试用户
setup_test_users() {
    print_header "前置准备：创建测试用户"

    # 注册用户1（注册成功后会自动登录返回token）
    print_info "注册用户1: ${TEST_EMAIL_1}"
    local data1="{\"email\":\"${TEST_EMAIL_1}\",\"password\":\"${TEST_PASSWORD}\",\"verifyCode\":\"123456\",\"nickname\":\"测试用户1_${TIMESTAMP}\",\"deviceType\":\"Web\",\"deviceId\":\"${TEST_DEVICE_ID}_1\"}"
    local response1=$(http_post "${API_BASE}/auth/register" "$data1")

    USER1_ID=$(echo "$response1" | jq -r '.data.userId // empty')
    USER1_TOKEN=$(echo "$response1" | jq -r '.data.accessToken // empty')

    if [ -z "$USER1_TOKEN" ] || [ "$USER1_TOKEN" = "null" ]; then
        print_error "用户1注册失败"
        print_info "Response: $response1"
        exit 1
    fi
    print_success "用户1注册成功 (ID: ${USER1_ID})"

    # 注册用户2
    print_info "注册用户2: ${TEST_EMAIL_2}"
    local data2="{\"email\":\"${TEST_EMAIL_2}\",\"password\":\"${TEST_PASSWORD}\",\"verifyCode\":\"123456\",\"nickname\":\"测试用户2_${TIMESTAMP}\",\"deviceType\":\"Web\",\"deviceId\":\"${TEST_DEVICE_ID}_2\"}"
    local response2=$(http_post "${API_BASE}/auth/register" "$data2")

    USER2_ID=$(echo "$response2" | jq -r '.data.userId // empty')
    USER2_TOKEN=$(echo "$response2" | jq -r '.data.accessToken // empty')

    if [ -z "$USER2_TOKEN" ] || [ "$USER2_TOKEN" = "null" ]; then
        print_error "用户2注册失败"
        print_info "Response: $response2"
        exit 1
    fi
    print_success "用户2注册成功 (ID: ${USER2_ID})"

    # 注册用户3
    print_info "注册用户3: ${TEST_EMAIL_3}"
    local data3="{\"email\":\"${TEST_EMAIL_3}\",\"password\":\"${TEST_PASSWORD}\",\"verifyCode\":\"123456\",\"nickname\":\"测试用户3_${TIMESTAMP}\",\"deviceType\":\"Web\",\"deviceId\":\"${TEST_DEVICE_ID}_3\"}"
    local response3=$(http_post "${API_BASE}/auth/register" "$data3")

    USER3_ID=$(echo "$response3" | jq -r '.data.userId // empty')
    USER3_TOKEN=$(echo "$response3" | jq -r '.data.accessToken // empty')

    if [ -z "$USER3_TOKEN" ] || [ "$USER3_TOKEN" = "null" ]; then
        print_error "用户3注册失败"
        print_info "Response: $response3"
        exit 1
    fi
    print_success "用户3注册成功 (ID: ${USER3_ID})"

    print_success "3个测试用户创建成功"
    print_info "User1 ID: $USER1_ID"
    print_info "User2 ID: $USER2_ID"
    print_info "User3 ID: $USER3_ID"
}

# 测试1：健康检查
test_health_check() {
    print_header "测试1：健康检查"

    local response=$(http_get "${GATEWAY_URL}/health")
    local status=$(echo "$response" | jq -r '.status // empty')

    if [ "$status" = "ok" ]; then
        print_success "健康检查通过"
    else
        print_error "健康检查失败"
    fi
}

# 测试2：创建群组
test_create_group() {
    print_header "测试2：创建群组"

    local data="{\"name\":\"${GROUP_NAME}\",\"memberIds\":[\"${USER2_ID}\"],\"joinVerify\":true}"
    local response=$(http_post "${API_BASE}/groups" "$data" "$USER1_TOKEN")

    GROUP_ID=$(echo "$response" | jq -r '.data.groupId // empty')

    if [ -n "$GROUP_ID" ] && [ "$GROUP_ID" != "null" ]; then
        check_response "$response" "0" "创建群组"
        print_info "群组ID: $GROUP_ID"
    else
        print_error "创建群组失败 - 未获取到群组ID"
        print_info "Response: $response"
    fi
}

# 测试3：获取群组信息
test_get_group_info() {
    print_header "测试3：获取群组信息"

    if [ -z "$GROUP_ID" ]; then
        print_error "跳过测试 - 群组ID为空"
        return 1
    fi

    local response=$(http_get "${API_BASE}/groups/${GROUP_ID}" "$USER1_TOKEN")
    local name=$(echo "$response" | jq -r '.data.name // empty')

    if [ "$name" = "$GROUP_NAME" ]; then
        check_response "$response" "0" "获取群组信息"
        print_info "群名称: $name"
        print_info "群主ID: $(echo "$response" | jq -r '.data.ownerId')"
        print_info "成员数: $(echo "$response" | jq -r '.data.memberCount')"
        print_info "我的角色: $(echo "$response" | jq -r '.data.myRole')"
    else
        print_error "获取群组信息失败"
    fi
}

# 测试4：获取群成员列表
test_get_group_members() {
    print_header "测试4：获取群成员列表"

    if [ -z "$GROUP_ID" ]; then
        print_error "跳过测试 - 群组ID为空"
        return 1
    fi

    local response=$(http_get "${API_BASE}/groups/${GROUP_ID}/members" "$USER1_TOKEN")
    local total=$(echo "$response" | jq -r '.data.total // 0')

    if [ "$total" -ge "2" ]; then
        check_response "$response" "0" "获取群成员列表"
        print_info "成员总数: $total"
    else
        print_error "获取群成员列表失败 - 成员数不正确"
    fi
}

# 测试5：更新群信息
test_update_group() {
    print_header "测试5：更新群信息"

    if [ -z "$GROUP_ID" ]; then
        print_error "跳过测试 - 群组ID为空"
        return 1
    fi

    local new_name="${GROUP_NAME}_updated"
    local data="{\"name\":\"${new_name}\",\"announcement\":\"这是测试公告\"}"
    local response=$(http_put "${API_BASE}/groups/${GROUP_ID}" "$data" "$USER1_TOKEN")

    check_response "$response" "0" "更新群信息"
}

# 测试6：邀请成员（需要验证）
test_invite_members() {
    print_header "测试6：邀请成员加入（需验证）"

    if [ -z "$GROUP_ID" ]; then
        print_error "跳过测试 - 群组ID为空"
        return 1
    fi

    local data="{\"userIds\":[\"${USER3_ID}\"]}"
    local response=$(http_post "${API_BASE}/groups/${GROUP_ID}/members" "$data" "$USER1_TOKEN")

    check_response "$response" "0" "邀请成员"
    print_info "已邀请用户3加入群组（需要验证）"
}

# 测试7：获取入群申请列表
test_get_join_requests() {
    print_header "测试7：获取入群申请列表"

    if [ -z "$GROUP_ID" ]; then
        print_error "跳过测试 - 群组ID为空"
        return 1
    fi

    # 稍等片刻确保申请已创建
    sleep 1

    local response=$(http_get "${API_BASE}/groups/${GROUP_ID}/requests?status=pending" "$USER1_TOKEN")
    local total=$(echo "$response" | jq -r '.data.total // 0')

    if [ "$total" -gt "0" ]; then
        check_response "$response" "0" "获取入群申请列表"
        JOIN_REQUEST_ID=$(echo "$response" | jq -r '.data.requests[0].id // empty')
        print_info "待处理申请数: $total"
        print_info "申请ID: $JOIN_REQUEST_ID"
    else
        print_error "获取入群申请列表失败 - 没有待处理的申请"
    fi
}

# 测试8：处理入群申请（接受）
test_accept_join_request() {
    print_header "测试8：处理入群申请（接受）"

    if [ -z "$GROUP_ID" ] || [ -z "$JOIN_REQUEST_ID" ]; then
        print_error "跳过测试 - 群组ID或申请ID为空"
        return 1
    fi

    local data="{\"accept\":true}"
    local response=$(http_put "${API_BASE}/groups/${GROUP_ID}/requests/${JOIN_REQUEST_ID}" "$data" "$USER1_TOKEN")

    check_response "$response" "0" "接受入群申请"
}

# 测试9：验证成员已加入
test_verify_member_joined() {
    print_header "测试9：验证成员已加入"

    if [ -z "$GROUP_ID" ]; then
        print_error "跳过测试 - 群组ID为空"
        return 1
    fi

    # 稍等片刻确保成员已添加
    sleep 1

    local response=$(http_get "${API_BASE}/groups/${GROUP_ID}/members" "$USER1_TOKEN")
    local total=$(echo "$response" | jq -r '.data.total // 0')

    if [ "$total" -ge "3" ]; then
        check_response "$response" "0" "验证成员已加入"
        print_info "当前成员总数: $total"
    else
        print_error "验证失败 - 成员数不正确（期望>=3，实际: $total）"
    fi
}

# 测试10：更新成员角色
test_update_member_role() {
    print_header "测试10：更新成员角色为管理员"

    if [ -z "$GROUP_ID" ] || [ -z "$USER2_ID" ]; then
        print_error "跳过测试 - 群组ID或用户ID为空"
        return 1
    fi

    local data="{\"role\":\"admin\"}"
    local response=$(http_put "${API_BASE}/groups/${GROUP_ID}/members/${USER2_ID}/role" "$data" "$USER1_TOKEN")

    check_response "$response" "0" "设置成员为管理员"
}

# 测试11：更新群昵称
test_update_member_nickname() {
    print_header "测试11：更新群昵称"

    if [ -z "$GROUP_ID" ]; then
        print_error "跳过测试 - 群组ID为空"
        return 1
    fi

    local data="{\"nickname\":\"我的群昵称\"}"
    local response=$(http_put "${API_BASE}/groups/${GROUP_ID}/nickname" "$data" "$USER1_TOKEN")

    check_response "$response" "0" "更新群昵称"
}

# 测试12：移除群成员
test_remove_member() {
    print_header "测试12：移除群成员"

    if [ -z "$GROUP_ID" ] || [ -z "$USER3_ID" ]; then
        print_error "跳过测试 - 群组ID或用户ID为空"
        return 1
    fi

    local response=$(http_delete "${API_BASE}/groups/${GROUP_ID}/members/${USER3_ID}" "$USER1_TOKEN")

    check_response "$response" "0" "移除群成员"
}

# 测试13：退出群组
test_quit_group() {
    print_header "测试13：用户2退出群组"

    if [ -z "$GROUP_ID" ]; then
        print_error "跳过测试 - 群组ID为空"
        return 1
    fi

    local response=$(http_post "${API_BASE}/groups/${GROUP_ID}/quit" "{}" "$USER2_TOKEN")

    check_response "$response" "0" "退出群组"
}

# 测试14：获取我的群组列表
test_get_my_groups() {
    print_header "测试14：获取我的群组列表"

    local response=$(http_get "${API_BASE}/groups" "$USER1_TOKEN")
    local total=$(echo "$response" | jq -r '.data.total // 0')

    if [ "$total" -ge "1" ]; then
        check_response "$response" "0" "获取我的群组列表"
        print_info "我加入的群组数: $total"
    else
        print_error "获取群组列表失败"
    fi
}

# 测试15：解散群组
test_dissolve_group() {
    print_header "测试15：解散群组"

    if [ -z "$GROUP_ID" ]; then
        print_error "跳过测试 - 群组ID为空"
        return 1
    fi

    local response=$(http_delete "${API_BASE}/groups/${GROUP_ID}" "$USER1_TOKEN")

    check_response "$response" "0" "解散群组"
}

# 打印测试结果摘要
print_summary() {
    print_header "测试结果汇总"

    local total=$((TEST_PASSED + TEST_FAILED))
    echo -e "总测试数: $total"
    echo -e "${GREEN}通过: $TEST_PASSED${NC}"
    echo -e "${RED}失败: $TEST_FAILED${NC}"

    if [ $TEST_FAILED -eq 0 ]; then
        echo -e "\n${GREEN}所有测试通过! ✓${NC}"
        return 0
    else
        echo -e "\n${RED}部分测试失败 ✗${NC}"
        return 1
    fi
}

# 主函数
main() {
    echo "======================================"
    echo "Group Service HTTP API 测试"
    echo "======================================"
    echo "开始时间: $(date '+%Y-%m-%d %H:%M:%S')"
    echo "Gateway URL: $GATEWAY_URL"
    echo ""

    # 检查必要工具
    if ! command -v jq &> /dev/null; then
        print_error "需要安装 jq 工具: sudo apt-get install jq"
        exit 1
    fi

    if ! command -v curl &> /dev/null; then
        print_error "需要安装 curl 工具"
        exit 1
    fi

    # 前置准备
    setup_test_users || exit 1

    # 执行测试
    test_health_check
    test_create_group
    test_get_group_info
    test_get_group_members
    test_update_group
    test_invite_members
    test_get_join_requests
    test_accept_join_request
    test_verify_member_joined
    test_update_member_role
    test_update_member_nickname
    test_remove_member
    test_quit_group
    test_get_my_groups
    test_dissolve_group

    # 打印结果
    echo ""
    echo "结束时间: $(date '+%Y-%m-%d %H:%M:%S')"
    print_summary
}

# 执行主函数
main "$@"
