#!/bin/bash
#
# File Service HTTP API 测试脚本
# 用于测试文件管理相关的 HTTP 接口
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
TEST_EMAIL="filetest_${TIMESTAMP}@example.com"
TEST_PASSWORD="Test@123456"
TEST_DEVICE_ID="test-device-${TIMESTAMP}"

# 全局变量
USER_TOKEN=""
USER_ID=""
FILE_ID=""
UPLOAD_URL=""
DOWNLOAD_URL=""

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

setup_test_user() {
    print_header "准备测试用户"

    print_info "注册用户: ${TEST_EMAIL}"
    local data=$(cat <<EOF
{
    "email": "${TEST_EMAIL}",
    "password": "${TEST_PASSWORD}",
    "verifyCode": "123456",
    "nickname": "文件测试用户_${TIMESTAMP}",
    "deviceType": "iOS",
    "deviceId": "${TEST_DEVICE_ID}"
}
EOF
)
    local response=$(http_post "${API_BASE}/auth/register" "$data")
    if check_response "$response"; then
        USER_ID=$(echo "$response" | jq -r '.data.userId')
        USER_TOKEN=$(echo "$response" | jq -r '.data.accessToken')
        print_success "用户注册成功 (ID: ${USER_ID})"
    else
        print_error "用户注册失败"
        return 1
    fi
}

# ========================================
# 测试用例
# ========================================

# 测试1: 健康检查
test_health_check() {
    print_header "测试1: 健康检查"

    local response=$(http_get "${GATEWAY_URL}/health")
    if echo "$response" | jq -e '.status == "ok"' > /dev/null; then
        print_success "健康检查通过"
        return 0
    else
        print_error "健康检查失败"
        return 1
    fi
}

# 测试2: 生成上传凭证
test_generate_upload_token() {
    print_header "测试2: 生成上传凭证"

    local data=$(cat <<EOF
{
    "fileName": "test-image-${TIMESTAMP}.jpg",
    "fileSize": 102400,
    "mimeType": "image/jpeg",
    "fileType": "image",
    "expiresHours": 0
}
EOF
)

    print_info "请求生成上传凭证..."
    local response=$(http_post "${API_BASE}/files/upload-token" "$data" "$USER_TOKEN")

    if check_response "$response"; then
        FILE_ID=$(echo "$response" | jq -r '.data.file_id')
        UPLOAD_URL=$(echo "$response" | jq -r '.data.upload_url')
        local expires_in=$(echo "$response" | jq -r '.data.expires_in')

        print_success "生成上传凭证成功"
        print_info "File ID: ${FILE_ID}"
        print_info "Upload URL Host: $(echo "$UPLOAD_URL" | grep -oP 'https?://[^/]+')"
        print_info "Expires In: ${expires_in}s"

        # 调试：显示完整的 URL（截断）
        if [ ${#UPLOAD_URL} -gt 100 ]; then
            print_info "URL Preview: ${UPLOAD_URL:0:100}..."
        else
            print_info "URL: $UPLOAD_URL"
        fi

        return 0
    else
        print_error "生成上传凭证失败"
        return 1
    fi
}

# 测试3: 模拟文件上传到MinIO (创建临时测试文件)
test_upload_to_minio() {
    print_header "测试3: 模拟上传文件到MinIO"

    # 检查 UPLOAD_URL 是否已设置
    if [ -z "$UPLOAD_URL" ]; then
        print_error "UPLOAD_URL 未设置，请先运行测试2"
        return 1
    fi

    print_info "Upload URL: ${UPLOAD_URL:0:80}..."

    # 创建一个临时测试文件
    local temp_file=$(mktemp /tmp/test-file-XXXXXX.jpg)
    dd if=/dev/zero of="$temp_file" bs=1024 count=100 2>/dev/null

    print_info "创建临时测试文件: $temp_file (100KB)"

    # 使用curl上传到MinIO presigned URL (添加详细输出)
    print_info "上传文件到MinIO..."
    local temp_output=$(mktemp)
    local http_code=$(curl -s -w "%{http_code}" -o "$temp_output" -X PUT "$UPLOAD_URL" \
        -H "Content-Type: image/jpeg" \
        --data-binary "@$temp_file" \
        --max-time 30)

    # 显示响应内容（如果有错误）
    if [ "$http_code" != "200" ] && [ -s "$temp_output" ]; then
        print_info "MinIO 响应: $(cat $temp_output)"
    fi

    # 清理临时文件
    rm -f "$temp_file" "$temp_output"

    if [ "$http_code" = "200" ]; then
        print_success "文件上传到MinIO成功 (HTTP $http_code)"
        return 0
    else
        print_error "文件上传到MinIO失败 (HTTP $http_code)"
        print_info "可能原因: presigned URL 已过期、MinIO 不可访问、或URL格式错误"
        return 1
    fi
}

# 测试4: 完成上传
test_complete_upload() {
    print_header "测试4: 完成上传"

    print_info "通知服务端上传完成..."
    local response=$(http_post "${API_BASE}/files/${FILE_ID}/complete" "{}" "$USER_TOKEN")

    if check_response "$response"; then
        local status=$(echo "$response" | jq -r '.data.status')
        local file_name=$(echo "$response" | jq -r '.data.file_name')

        print_success "上传完成成功"
        print_info "File Name: ${file_name}"
        print_info "Status: ${status}"
        return 0
    else
        print_error "完成上传失败"
        return 1
    fi
}

# 测试5: 获取文件信息
test_get_file_info() {
    print_header "测试5: 获取文件信息"

    print_info "获取文件信息..."
    local response=$(http_get "${API_BASE}/files/${FILE_ID}" "$USER_TOKEN")

    if check_response "$response"; then
        local file_id=$(echo "$response" | jq -r '.data.file_id')
        local file_size=$(echo "$response" | jq -r '.data.file_size')
        local file_type=$(echo "$response" | jq -r '.data.file_type')

        print_success "获取文件信息成功"
        print_info "File ID: ${file_id}"
        print_info "File Size: ${file_size} bytes"
        print_info "File Type: ${file_type}"
        return 0
    else
        print_error "获取文件信息失败"
        return 1
    fi
}

# 测试6: 生成下载链接
test_generate_download_url() {
    print_header "测试6: 生成下载链接"

    print_info "生成下载链接..."
    local response=$(http_get "${API_BASE}/files/${FILE_ID}/download?expiresMinutes=30" "$USER_TOKEN")

    if check_response "$response"; then
        DOWNLOAD_URL=$(echo "$response" | jq -r '.data.download_url')
        local expires_in=$(echo "$response" | jq -r '.data.expires_in')

        print_success "生成下载链接成功"
        print_info "Expires In: ${expires_in}s"
        return 0
    else
        print_error "生成下载链接失败"
        return 1
    fi
}

# 测试7: 列出用户文件
test_list_user_files() {
    print_header "测试7: 列出用户文件"

    print_info "列出所有文件..."
    local response=$(http_get "${API_BASE}/files?page=1&pageSize=20" "$USER_TOKEN")

    if check_response "$response"; then
        local total=$(echo "$response" | jq -r '.data.total')
        local count=$(echo "$response" | jq -r '.data.files | length')

        print_success "列出文件成功"
        print_info "Total: ${total}"
        print_info "Current Page Count: ${count}"
        return 0
    else
        print_error "列出文件失败"
        return 1
    fi

    # 测试按类型过滤
    print_info "列出图片类型文件..."
    local response2=$(http_get "${API_BASE}/files?fileType=image&page=1&pageSize=20" "$USER_TOKEN")

    if check_response "$response2"; then
        local count2=$(echo "$response2" | jq -r '.data.files | length')
        print_success "按类型过滤成功 (图片数量: ${count2})"
        return 0
    else
        print_error "按类型过滤失败"
        return 1
    fi
}

# 测试8: 删除文件
test_delete_file() {
    print_header "测试8: 删除文件"

    print_info "删除文件..."
    local response=$(http_delete "${API_BASE}/files/${FILE_ID}" "$USER_TOKEN")

    if check_response "$response"; then
        local success=$(echo "$response" | jq -r '.data.success')

        if [ "$success" = "true" ]; then
            print_success "删除文件成功"
            return 0
        else
            print_error "删除文件失败 (success=false)"
            return 1
        fi
    else
        print_error "删除文件失败"
        return 1
    fi

    # 验证文件已删除
    print_info "验证文件已删除..."
    local verify_response=$(http_get "${API_BASE}/files/${FILE_ID}" "$USER_TOKEN")
    local code=$(echo "$verify_response" | jq -r '.code // -1')

    if [ "$code" != "0" ]; then
        print_success "验证文件已删除成功"
        return 0
    else
        print_error "验证失败：文件仍然存在"
        return 1
    fi
}

# ========================================
# 主函数
# ========================================

main() {
    echo "File Service API 测试脚本"
    echo "开始时间: $(date '+%Y-%m-%d %H:%M:%S')"

    # 检查依赖
    if ! command -v jq &> /dev/null; then
        print_error "jq 未安装，请先安装 jq"
        exit 1
    fi

    if ! command -v curl &> /dev/null; then
        print_error "curl 未安装，请先安装 curl"
        exit 1
    fi

    # 准备测试用户
    setup_test_user || exit 1

    # 运行测试
    local failed=0

    test_health_check || ((failed++))
    test_generate_upload_token || ((failed++))
    test_upload_to_minio || ((failed++))
    test_complete_upload || ((failed++))
    test_get_file_info || ((failed++))
    test_generate_download_url || ((failed++))
    test_list_user_files || ((failed++))
    test_delete_file || ((failed++))

    # 结果汇总
    print_header "测试完成"
    echo "结束时间: $(date '+%Y-%m-%d %H:%M:%S')"

    if [ $failed -eq 0 ]; then
        echo -e "${GREEN}所有测试通过! ✓${NC}"
        exit 0
    else
        echo -e "${RED}失败测试数: ${failed} ✗${NC}"
        exit 1
    fi
}

main "$@"
