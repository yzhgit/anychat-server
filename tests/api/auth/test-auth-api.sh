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
FIXED_CODE="${VERIFY_DEBUG_FIXED_CODE:-123456}"

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

send_register_code() {
    send_code "${TEST_PHONE}" "sms" "register" "${TEST_DEVICE_ID}"
}

make_phone() {
    local seed=${1:-0}
    printf "138%08d" $(((TIMESTAMP + seed) % 100000000))
}

send_code() {
    local target=$1
    local target_type=$2
    local purpose=$3
    local device_id=${4:-$TEST_DEVICE_ID}
    local data=$(cat <<EOF
{
    "target": "${target}",
    "targetType": "${target_type}",
    "purpose": "${purpose}",
    "deviceId": "${device_id}"
}
EOF
)

    http_post "${API_BASE}/auth/send-code" "$data"
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

# 1. 发送短信验证码
test_send_sms_code() {
    print_header "1. 发送短信验证码"

    local phone=$(make_phone 1)
    local response=$(send_code "$phone" "sms" "register" "${TEST_DEVICE_ID}-sms")
    print_info "响应: $response"

    if check_response "$response"; then
        local code_id=$(echo "$response" | jq -r '.data.codeId // empty')
        print_success "短信验证码发送成功"
        print_info "CodeId: ${code_id}"
        return 0
    fi

    return 1
}

# 2. 发送邮箱验证码
test_send_email_code() {
    print_header "2. 发送邮箱验证码"

    local email="send_${TIMESTAMP}@example.com"
    local response=$(send_code "$email" "email" "register" "${TEST_DEVICE_ID}-email")
    print_info "响应: $response"

    if check_response "$response"; then
        local code_id=$(echo "$response" | jq -r '.data.codeId // empty')
        print_success "邮箱验证码发送成功"
        print_info "CodeId: ${code_id}"
        return 0
    fi

    return 1
}

# 3. 目标格式错误
test_invalid_target_format() {
    print_header "3. 目标格式错误"

    local response=$(send_code "invalid-phone" "sms" "register" "${TEST_DEVICE_ID}-invalid")
    print_info "响应: $response"

    local code=$(echo "$response" | jq -r '.code // -1')
    if [ "$code" != "0" ]; then
        print_success "无效目标格式被正确拒绝"
        return 0
    fi

    print_error "无效目标格式不应成功"
    return 1
}

# 4. 发送频率限制
test_rate_limit() {
    print_header "4. 发送频率限制"

    local target="rate_limit_${TIMESTAMP}@example.com"
    local failed_count=0

    for _ in 1 2 3; do
        local response=$(send_code "$target" "email" "register" "${TEST_DEVICE_ID}-rate")
        local code=$(echo "$response" | jq -r '.code // -1')
        if [ "$code" != "0" ]; then
            ((failed_count+=1))
        fi
        sleep 0.5
    done

    if [ $failed_count -gt 0 ]; then
        print_success "频率限制生效 (触发 ${failed_count} 次)"
        return 0
    fi

    print_error "未触发频率限制"
    return 1
}

# 5. 使用错误验证码注册
test_register_with_wrong_code() {
    print_header "5. 使用错误验证码注册"

    local email="wrong_code_${TIMESTAMP}@example.com"
    local device_id="${TEST_DEVICE_ID}-wrong"
    local send_response=$(send_code "$email" "email" "register" "$device_id")
    print_info "发送验证码响应: $send_response"
    if ! check_response "$send_response"; then
        return 1
    fi

    local data=$(cat <<EOF
{
    "email": "${email}",
    "password": "${TEST_PASSWORD}",
    "verifyCode": "000000",
    "nickname": "WrongCodeUser${TIMESTAMP}",
    "deviceType": "Web",
    "deviceId": "${device_id}",
    "clientVersion": "1.0.0"
}
EOF
)

    local response=$(http_post "${API_BASE}/auth/register" "$data")
    print_info "响应: $response"

    local code=$(echo "$response" | jq -r '.code // -1')
    if [ "$code" != "0" ]; then
        print_success "错误验证码被正确拒绝"
        return 0
    fi

    print_error "错误验证码不应注册成功"
    return 1
}

# 6. 使用固定验证码注册
test_register_with_fixed_code() {
    print_header "6. 使用固定验证码注册"

    local email="success_${TIMESTAMP}@example.com"
    local device_id="${TEST_DEVICE_ID}-success"
    local send_response=$(send_code "$email" "email" "register" "$device_id")
    print_info "发送验证码响应: $send_response"
    if ! check_response "$send_response"; then
        return 1
    fi

    local data=$(cat <<EOF
{
    "email": "${email}",
    "password": "${TEST_PASSWORD}",
    "verifyCode": "${FIXED_CODE}",
    "nickname": "VerifyFlowUser${TIMESTAMP}",
    "deviceType": "Web",
    "deviceId": "${device_id}",
    "clientVersion": "1.0.0"
}
EOF
)

    local response=$(http_post "${API_BASE}/auth/register" "$data")
    print_info "响应: $response"

    if check_response "$response"; then
        print_success "固定验证码注册成功"
        return 0
    fi

    return 1
}

# 7. 发送重置密码验证码
test_send_reset_password_code() {
    print_header "7. 发送重置密码验证码"

    local phone=$(make_phone 2)
    local response=$(send_code "$phone" "sms" "reset_password" "${TEST_DEVICE_ID}-reset")
    print_info "响应: $response"

    if check_response "$response"; then
        print_success "重置密码验证码发送成功"
        return 0
    fi

    return 1
}

# 8. 用户注册
test_register() {
    print_header "8. 用户注册"

    local send_response=$(send_register_code)
    print_info "发送验证码响应: $send_response"
    if ! check_response "$send_response"; then
        return 1
    fi

    local data=$(cat <<EOF
{
    "phoneNumber": "${TEST_PHONE}",
    "password": "${TEST_PASSWORD}",
    "verifyCode": "${FIXED_CODE}",
    "nickname": "测试用户${TIMESTAMP}",
    "deviceType": "iOS",
    "deviceId": "${TEST_DEVICE_ID}",
    "clientVersion": "1.0.0"
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

# 9. 用户登录
test_login() {
    print_header "9. 用户登录"

    local data=$(cat <<EOF
{
    "account": "${TEST_PHONE}",
    "password": "${TEST_PASSWORD}",
    "deviceType": "iOS",
    "deviceId": "${TEST_DEVICE_ID}",
    "clientVersion": "1.0.0"
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

# 10. 修改密码
test_change_password() {
    print_header "10. 修改密码"

    local new_password="NewPass@123456"
    local data=$(cat <<EOF
{
    "deviceId": "${TEST_DEVICE_ID}",
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

# 11. 使用新密码登录验证
test_login_with_new_password() {
    print_header "11. 使用新密码登录验证"

    local data=$(cat <<EOF
{
    "account": "${TEST_PHONE}",
    "password": "${TEST_PASSWORD}",
    "deviceType": "iOS",
    "deviceId": "${TEST_DEVICE_ID}_2",
    "clientVersion": "1.0.0"
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

# 12. 刷新Token
test_refresh_token() {
    print_header "12. 刷新Token"

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

# 13. 登出
test_logout() {
    print_header "13. 登出"

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
    sleep 1
    test_send_sms_code || ((failed++))
    sleep 1
    test_send_email_code || ((failed++))
    sleep 1
    test_invalid_target_format || ((failed++))
    sleep 1
    test_rate_limit || ((failed++))
    sleep 1
    test_register_with_wrong_code || ((failed++))
    sleep 1
    test_register_with_fixed_code || ((failed++))
    sleep 1
    test_send_reset_password_code || ((failed++))
    sleep 1
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
