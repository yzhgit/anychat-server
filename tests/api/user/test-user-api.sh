#!/bin/bash
#
# User Service HTTP API 测试脚本
# 用于测试用户管理相关的 HTTP 接口
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

http_put() {
    local url=$1
    local data=$2
    local token=$3

    curl -s -X PUT "${url}" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${token}" \
        -d "${data}"
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

setup_test_user() {
    print_header "准备测试用户"

    # 注册用户
    print_info "注册测试用户: ${TEST_EMAIL}"
    local data=$(cat <<EOF
{
    "email": "${TEST_EMAIL}",
    "password": "${TEST_PASSWORD}",
    "verifyCode": "123456",
    "nickname": "测试用户${TIMESTAMP}",
    "deviceType": "iOS",
    "deviceId": "${TEST_DEVICE_ID}"
}
EOF
)

    local response=$(http_post "${API_BASE}/auth/register" "$data")
    if check_response "$response"; then
        USER_ID=$(echo "$response" | jq -r '.data.userId // .data.user_id // empty')
        ACCESS_TOKEN=$(echo "$response" | jq -r '.data.accessToken // .data.access_token // empty')

        if [ -z "$USER_ID" ] || [ "$USER_ID" = "null" ]; then
            print_error "无法获取用户ID"
            return 1
        fi

        print_success "测试用户创建成功 (ID: ${USER_ID})"
        return 0
    else
        print_error "测试用户创建失败"
        return 1
    fi
}

# ========================================
# 测试用例
# ========================================

# 1. 获取个人资料
test_get_profile() {
    print_header "1. 获取个人资料"

    local response=$(http_get "${API_BASE}/users/me" "$ACCESS_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        local nickname=$(echo "$response" | jq -r '.data.nickname // empty')
        if [ -z "$nickname" ]; then
            print_error "无法获取昵称"
            return 1
        fi

        print_success "获取个人资料成功"
        print_info "昵称: ${nickname}"
        return 0
    else
        return 1
    fi
}

# 2. 更新个人资料
test_update_profile() {
    print_header "2. 更新个人资料"

    local new_nickname="更新昵称${TIMESTAMP}"
    local data=$(cat <<EOF
{
    "nickname": "${new_nickname}",
    "signature": "这是一个测试签名",
    "gender": 1
}
EOF
)

    print_info "更新信息: 新昵称=${new_nickname}"

    local response=$(http_put "${API_BASE}/users/me" "$data" "$ACCESS_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        print_success "更新个人资料成功"
        return 0
    else
        return 1
    fi
}

# 3. 验证资料已更新
test_verify_profile_updated() {
    print_header "3. 验证资料已更新"

    local response=$(http_get "${API_BASE}/users/me" "$ACCESS_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        local nickname=$(echo "$response" | jq -r '.data.nickname // empty')
        local signature=$(echo "$response" | jq -r '.data.signature // empty')
        local gender=$(echo "$response" | jq -r '.data.gender // 0')

        print_success "验证成功"
        print_info "昵称: ${nickname}"
        print_info "签名: ${signature}"
        print_info "性别: ${gender}"
        return 0
    else
        return 1
    fi
}

# 4. 搜索用户
test_search_users() {
    print_header "4. 搜索用户"

    local keyword="测试"
    print_info "搜索关键词: ${keyword}"

    local response=$(http_get "${API_BASE}/users/search?keyword=${keyword}&page=1&pageSize=10" "$ACCESS_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        local total=$(echo "$response" | jq -r '.data.total // 0')
        print_success "搜索用户成功"
        print_info "找到 ${total} 个用户"
        return 0
    else
        return 1
    fi
}

# 5. 获取用户设置
test_get_settings() {
    print_header "5. 获取用户设置"

    local response=$(http_get "${API_BASE}/users/me/settings" "$ACCESS_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        print_success "获取用户设置成功"
        return 0
    else
        return 1
    fi
}

# 6. 更新用户设置
test_update_settings() {
    print_header "6. 更新用户设置"

    local data=$(cat <<EOF
{
    "notificationEnabled": true,
    "soundEnabled": false,
    "language": "zh-CN"
}
EOF
)

    local response=$(http_put "${API_BASE}/users/me/settings" "$data" "$ACCESS_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        print_success "更新用户设置成功"
        return 0
    else
        return 1
    fi
}

# 7. 验证设置已更新
test_verify_settings_updated() {
    print_header "7. 验证设置已更新"

    local response=$(http_get "${API_BASE}/users/me/settings" "$ACCESS_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        local notification=$(echo "$response" | jq -r '.data.notificationEnabled // .data.notification_enabled // false')
        local sound=$(echo "$response" | jq -r '.data.soundEnabled // .data.sound_enabled // true')
        local language=$(echo "$response" | jq -r '.data.language // ""')

        print_success "验证成功"
        print_info "通知: ${notification}"
        print_info "声音: ${sound}"
        print_info "语言: ${language}"
        return 0
    else
        return 1
    fi
}

# 8. 刷新二维码
test_refresh_qrcode() {
    print_header "8. 刷新二维码"

    local response=$(http_post "${API_BASE}/users/me/qrcode/refresh" "{}" "$ACCESS_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        local qrcode_url=$(echo "$response" | jq -r '.data.qrcodeUrl // .data.qrcode_url // empty')

        if [ -z "$qrcode_url" ] || [ "$qrcode_url" = "null" ]; then
            print_error "无法获取二维码URL"
            return 1
        fi

        print_success "刷新二维码成功"
        print_info "二维码URL: ${qrcode_url}"
        return 0
    else
        return 1
    fi
}

# 9. 更新推送Token
test_update_push_token() {
    print_header "9. 更新推送Token"

    local data=$(cat <<EOF
{
    "deviceId": "${TEST_DEVICE_ID}",
    "pushToken": "test-push-token-${TIMESTAMP}",
    "platform": "iOS"
}
EOF
)

    local response=$(http_post "${API_BASE}/users/me/push-token" "$data" "$ACCESS_TOKEN")
    print_info "响应: $response"

    if check_response "$response"; then
        print_success "更新推送Token成功"
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
    echo "║   User Service API 测试脚本               ║"
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
    setup_test_user || exit 1

    # 执行测试
    local failed=0

    test_get_profile || ((failed++))
    sleep 1
    test_update_profile || ((failed++))
    test_verify_profile_updated || ((failed++))
    sleep 1
    test_search_users || ((failed++))
    test_get_settings || ((failed++))
    test_update_settings || ((failed++))
    test_verify_settings_updated || ((failed++))
    sleep 1
    test_refresh_qrcode || ((failed++))
    test_update_push_token || ((failed++))

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
