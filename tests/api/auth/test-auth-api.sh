#!/bin/bash
#
# Auth Service HTTP API 测试脚本
# 用于测试认证相关的 HTTP 接口
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
TEST_PHONE="138${TIMESTAMP:(-8)}"
TEST_EMAIL="test${TIMESTAMP}@example.com"
TEST_PASSWORD="Test@123456"
TEST_DEVICE_ID="test-device-${TIMESTAMP}"

# 全局变量
ACCESS_TOKEN=""
REFRESH_TOKEN=""
USER_ID=""

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
# 测试用例
# ========================================

# 0. 健康检查
test_health_check() {
    print_header "0. 健康检查"

    local response=$(http_get "${GATEWAY_URL}/health")
    print_info "响应: $response"

    local status=$(echo "$response" | jq -r '.status // ""')
    if [ "$status" = "ok" ]; then
        print_success "健康检查通过"
        return 0
    else
        print_error "健康检查失败"
        return 1
    fi
}

# 1. 用户注册
test_register() {
    print_header "1. 用户注册"

    local data=$(cat <<EOF
{
    "phoneNumber": "${TEST_PHONE}",
    "password": "${TEST_PASSWORD}",
    "verifyCode": "123456",
    "nickname": "测试用户${TIMESTAMP}",
    "deviceType": "iOS",
    "deviceId": "${TEST_DEVICE_ID}"
}
EOF
)

    print_info "注册信息: 手机号=${TEST_PHONE}"

    local response=$(http_post "${API_BASE}/auth/register" "$data")
    print_info "响应: $response"

    if check_response "$response"; then
        USER_ID=$(echo "$response" | jq -r '.data.userId // .data.user_id // empty')
        ACCESS_TOKEN=$(echo "$response" | jq -r '.data.accessToken // .data.access_token // empty')
        REFRESH_TOKEN=$(echo "$response" | jq -r '.data.refreshToken // .data.refresh_token // empty')

        if [ -z "$USER_ID" ] || [ "$USER_ID" = "null" ]; then
            print_error "无法获取用户ID"
            return 1
        fi

        print_success "注册成功"
        print_info "用户ID: ${USER_ID}"
        print_info "AccessToken: ${ACCESS_TOKEN:0:20}..."
        return 0
    else
        return 1
    fi
}

# 2. 用户登录
test_login() {
    print_header "2. 用户登录"

    local data=$(cat <<EOF
{
    "account": "${TEST_PHONE}",
    "password": "${TEST_PASSWORD}",
    "deviceType": "iOS",
    "deviceId": "${TEST_DEVICE_ID}"
}
EOF
)

    print_info "登录信息: 账号=${TEST_PHONE}"

    local response=$(http_post "${API_BASE}/auth/login" "$data")
    print_info "响应: $response"

    if check_response "$response"; then
        USER_ID=$(echo "$response" | jq -r '.data.userId // .data.user_id // empty')
        ACCESS_TOKEN=$(echo "$response" | jq -r '.data.accessToken // .data.access_token // empty')
        REFRESH_TOKEN=$(echo "$response" | jq -r '.data.refreshToken // .data.refresh_token // empty')

        if [ -z "$USER_ID" ] || [ "$USER_ID" = "null" ]; then
            print_error "无法获取用户ID"
            return 1
        fi

        print_success "登录成功"
        print_info "用户ID: ${USER_ID}"
        return 0
    else
        return 1
    fi
}

# 3. 修改密码
test_change_password() {
    print_header "3. 修改密码"

    local new_password="NewPass@123456"
    local data=$(cat <<EOF
{
    "oldPassword": "${TEST_PASSWORD}",
    "newPassword": "${new_password}"
}
EOF
)

    print_info "修改密码"

    local response=$(http_post "${API_BASE}/auth/password/change" "$data" "$ACCESS_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        print_success "修改密码成功"
        TEST_PASSWORD="${new_password}"
        return 0
    else
        return 1
    fi
}

# 4. 使用新密码登录验证
test_login_with_new_password() {
    print_header "4. 使用新密码登录验证"

    local data=$(cat <<EOF
{
    "account": "${TEST_PHONE}",
    "password": "${TEST_PASSWORD}",
    "deviceType": "iOS",
    "deviceId": "${TEST_DEVICE_ID}_2"
}
EOF
)

    print_info "使用新密码登录"

    local response=$(http_post "${API_BASE}/auth/login" "$data")
    print_info "响应: $response"

    if check_response "$response"; then
        ACCESS_TOKEN=$(echo "$response" | jq -r '.data.accessToken // .data.access_token // empty')
        REFRESH_TOKEN=$(echo "$response" | jq -r '.data.refreshToken // .data.refresh_token // empty')

        print_success "新密码登录成功"
        return 0
    else
        return 1
    fi
}

# 5. 刷新Token
test_refresh_token() {
    print_header "5. 刷新Token"

    local data=$(cat <<EOF
{
    "refreshToken": "${REFRESH_TOKEN}"
}
EOF
)

    print_info "使用 RefreshToken 刷新"

    local response=$(http_post "${API_BASE}/auth/refresh" "$data")
    print_info "响应: $response"

    if check_response "$response"; then
        local new_access=$(echo "$response" | jq -r '.data.accessToken // .data.access_token // empty')
        local new_refresh=$(echo "$response" | jq -r '.data.refreshToken // .data.refresh_token // empty')

        if [ -z "$new_access" ] || [ "$new_access" = "null" ]; then
            print_error "无法获取新的AccessToken"
            return 1
        fi

        ACCESS_TOKEN="$new_access"
        REFRESH_TOKEN="$new_refresh"

        print_success "刷新Token成功"
        print_info "新AccessToken: ${ACCESS_TOKEN:0:20}..."
        return 0
    else
        return 1
    fi
}

# 6. 登出
test_logout() {
    print_header "6. 登出"

    local data=$(cat <<EOF
{
    "deviceId": "${TEST_DEVICE_ID}"
}
EOF
)

    local response=$(http_post "${API_BASE}/auth/logout" "$data" "$ACCESS_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        print_success "登出成功"
        return 0
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
    echo "║   Auth Service API 测试脚本               ║"
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

    # 执行测试
    local failed=0

    test_health_check || ((failed++))
    test_register || ((failed++))
    sleep 1
    test_login || ((failed++))
    test_change_password || ((failed++))
    sleep 1
    test_login_with_new_password || ((failed++))
    test_refresh_token || ((failed++))
    sleep 1
    test_logout || ((failed++))

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
