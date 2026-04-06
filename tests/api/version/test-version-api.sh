#!/bin/bash
#
# Version Service HTTP API 测试脚本
# 用于测试客户端版本升级相关的 HTTP 接口
#

set -e

# 加载公共函数
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PARENT_DIR="$(dirname "$SCRIPT_DIR")"
source "${PARENT_DIR}/common.sh"

# 配置
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
API_BASE="${GATEWAY_URL}/api/v1"

# 测试数据
TIMESTAMP=$(date +%s)
TEST_PLATFORM="android"
TEST_VERSION="1.0.0"
TEST_BUILD_NUMBER=100

# 全局变量
ACCESS_TOKEN=""

# 打印函数
print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
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

# 检查响应
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

# 测试1: 版本检测 - 无需更新
test_check_version_no_update() {
    print_header "测试1: 版本检测 - 无需更新"

    local response=$(http_get "${API_BASE}/versions/check?platform=${TEST_PLATFORM}&version=99.0.0")
    echo "Response: $response"

    local hasUpdate=$(echo "$response" | jq -r '.data.hasUpdate // false')
    if [ "$hasUpdate" = "false" ]; then
        print_success "版本检测成功 - 无需更新"
    else
        print_error "版本检测失败"
        exit 1
    fi
}

# 测试2: 版本检测 - 有更新
test_check_version_has_update() {
    print_header "测试2: 版本检测 - 有更新"

    local response=$(http_get "${API_BASE}/versions/check?platform=${TEST_PLATFORM}&version=1.0.0")
    echo "Response: $response"

    local hasUpdate=$(echo "$response" | jq -r '.data.hasUpdate // false')
    if [ "$hasUpdate" = "true" ]; then
        local version=$(echo "$response" | jq -r '.data.latestVersion // ""')
        print_success "版本检测成功 - 有新版本: $version"
    else
        print_info "当前无新版本，跳过此测试"
    fi
}

# 测试3: 获取最新版本
test_get_latest_version() {
    print_header "测试3: 获取最新版本"

    local response=$(http_get "${API_BASE}/versions/latest?platform=${TEST_PLATFORM}")
    echo "Response: $response"

    local version=$(echo "$response" | jq -r '.data.version.version // ""')
    if [ -n "$version" ]; then
        print_success "获取最新版本成功: $version"
    else
        print_info "当前无发布版本，跳过此测试"
    fi
}

# 测试4: 获取版本列表
test_list_versions() {
    print_header "测试4: 获取版本列表"

    local response=$(http_get "${API_BASE}/versions/list?platform=${TEST_PLATFORM}&page=1&pageSize=10")
    echo "Response: $response"

    local total=$(echo "$response" | jq -r '.data.total // 0')
    print_success "获取版本列表成功 - 总数: $total"
}

# 测试5: 版本上报
test_report_version() {
    print_header "测试5: 版本上报"

    local response=$(http_post "${API_BASE}/versions/report" \
        "{\"platform\":\"${TEST_PLATFORM}\",\"version\":\"1.0.0\",\"buildNumber\":${TEST_BUILD_NUMBER},\"deviceId\":\"test-device-001\"}")
    echo "Response: $response"

    local code=$(echo "$response" | jq -r '.code // -1')
    if [ "$code" = "0" ]; then
        print_success "版本上报成功"
    else
        print_info "版本上报完成 (code: $code)"
    fi
}

# 测试6: 创建版本（管理后台）
test_create_version() {
    print_header "测试6: 创建版本（管理后台）"

    # 管理后台API在单独端口
    ADMIN_API="${ADMIN_URL:-http://localhost:8011}/api/v1"

    # 如果没有admin token，跳过此测试
    if [ -z "$ACCESS_TOKEN" ]; then
        print_info "需要先登录获取token，跳过创建版本测试"
        return 0
    fi

    local response=$(http_post "${ADMIN_API}/versions" \
        "{
            \"platform\": \"${TEST_PLATFORM}\",
            \"version\": \"2.0.0\",
            \"buildNumber\": 200,
            \"minVersion\": \"1.0.0\",
            \"minBuildNumber\": 100,
            \"forceUpdate\": false,
            \"releaseType\": \"stable\",
            \"title\": \"测试版本 v2.0.0\",
            \"content\": \"## 更新内容\\n- 新功能测试\\n- Bug修复\",
            \"downloadUrl\": \"https://example.com/app/android/v2.0.0.apk\",
            \"fileSize\": 52428800,
            \"fileHash\": \"sha256:test-hash\"
        }" "$ACCESS_TOKEN")
    echo "Response: $response"

    local code=$(echo "$response" | jq -r '.code // -1')
    if [ "$code" = "0" ]; then
        local versionId=$(echo "$response" | jq -r '.data.id // 0')
        print_success "创建版本成功 - ID: $versionId"
        echo "$versionId" > /tmp/version_test_id.txt
    else
        print_info "创建版本完成 (code: $code)"
    fi
}

# 测试7: 获取版本详情
test_get_version() {
    print_header "测试7: 获取版本详情"

    ADMIN_API="${ADMIN_URL:-http://localhost:8011}/api/v1"
    local versionId=$(cat /tmp/version_test_id.txt 2>/dev/null || echo "1")

    if [ -z "$ACCESS_TOKEN" ]; then
        print_info "需要先登录获取token，跳过获取版本详情测试"
        return 0
    fi

    local response=$(http_get "${ADMIN_API}/versions/${versionId}" "$ACCESS_TOKEN")
    echo "Response: $response"

    local version=$(echo "$response" | jq -r '.data.version.version // ""')
    if [ -n "$version" ]; then
        print_success "获取版本详情成功: $version"
    else
        print_info "版本详情获取完成"
    fi
}

# 测试8: 强制更新检测
test_force_update() {
    print_header "测试8: 强制更新检测"

    local response=$(http_get "${API_BASE}/versions/check?platform=${TEST_PLATFORM}&version=0.1.0")
    echo "Response: $response"

    local forceUpdate=$(echo "$response" | jq -r '.data.forceUpdate // false')
    if [ "$forceUpdate" = "true" ]; then
        print_success "强制更新检测成功"
    else
        print_info "无需强制更新"
    fi
}

# 清理测试数据
cleanup() {
    print_header "清理测试数据"
    rm -f /tmp/version_test_id.txt
    print_success "清理完成"
}

# 主函数
main() {
    echo -e "${GREEN}"
    echo "╔═══════════════════════════════════════════╗"
    echo "║   Version Service API 测试脚本           ║"
    echo "╚═══════════════════════════════════════════╝"
    echo -e "${NC}"

    check_dependencies

    # 执行测试用例
    test_check_version_no_update
    test_check_version_has_update
    test_get_latest_version
    test_list_versions
    test_report_version
    test_create_version
    test_get_version
    test_force_update

    cleanup

    echo -e "\n${GREEN}✓ 所有测试完成！${NC}\n"
}

main "$@"