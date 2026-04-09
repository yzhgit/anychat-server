#!/bin/bash
#
# File Service HTTP API Test Script
# Tests file management related HTTP APIs
#

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
API_BASE="${GATEWAY_URL}/api/v1"

# Test data
TIMESTAMP=$(date +%s)
TEST_EMAIL="filetest_${TIMESTAMP}@example.com"
TEST_PASSWORD="Test@123456"
TEST_DEVICE_ID="test-device-${TIMESTAMP}"

# Global variables
USER_TOKEN=""
USER_ID=""
FILE_ID=""
UPLOAD_URL=""
DOWNLOAD_URL=""

# Print functions
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

# HTTP request functions
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

# Check JSON response code field
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
# Setup: Create test user
# ========================================

setup_test_user() {
    print_header "Preparing test users"

    print_info "Registering user: ${TEST_EMAIL}"
    local data=$(cat <<EOF
{
    "email": "${TEST_EMAIL}",
    "password": "${TEST_PASSWORD}",
    "verifyCode": "123456",
    "nickname": "FileTestUser_${TIMESTAMP}",
    "deviceType": "iOS",
    "deviceId": "${TEST_DEVICE_ID}",
    "clientVersion": "1.0.0"
}
EOF
)
    local response=$(http_post "${API_BASE}/auth/register" "$data")
    if check_response "$response"; then
        USER_ID=$(echo "$response" | jq -r '.data.userId')
        USER_TOKEN=$(echo "$response" | jq -r '.data.accessToken')
        print_success "User registered successfully (ID: ${USER_ID})"
    else
        print_error "User registration failed"
        return 1
    fi
}

# ========================================
# Test cases
# ========================================

# Test 1: Health check
test_health_check() {
    print_header "Test 1: Health Check"

    local response=$(http_get "${GATEWAY_URL}/health")
    if echo "$response" | jq -e '.status == "ok"' > /dev/null; then
        print_success "Health check passed"
        return 0
    else
        print_error "Health check failed"
        return 1
    fi
}

# Test 2: Generate upload token
test_generate_upload_token() {
    print_header "Test 2: Generate Upload Token"

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

    print_info "Requesting upload token generation..."
    local response=$(http_post "${API_BASE}/files/upload-token" "$data" "$USER_TOKEN")

    if check_response "$response"; then
        FILE_ID=$(echo "$response" | jq -r '.data.file_id')
        UPLOAD_URL=$(echo "$response" | jq -r '.data.upload_url')
        local expires_in=$(echo "$response" | jq -r '.data.expires_in')

        print_success "Generate upload token successful"
        print_info "File ID: ${FILE_ID}"
        print_info "Upload URL Host: $(echo "$UPLOAD_URL" | grep -oP 'https?://[^/]+')"
        print_info "Expires In: ${expires_in}s"

        # Debug: show full URL (truncated)
        if [ ${#UPLOAD_URL} -gt 100 ]; then
            print_info "URL Preview: ${UPLOAD_URL:0:100}..."
        else
            print_info "URL: $UPLOAD_URL"
        fi

        return 0
    else
        print_error "Generate upload token failed"
        return 1
    fi
}

# Test 3: Simulate file upload to MinIO (create temp test file)
test_upload_to_minio() {
    print_header "Test 3: Simulate Upload to MinIO"

    # Check if UPLOAD_URL is set
    if [ -z "$UPLOAD_URL" ]; then
        print_error "UPLOAD_URL not set, please run test 2 first"
        return 1
    fi

    print_info "Upload URL: ${UPLOAD_URL:0:80}..."

    # Create a temp test file
    local temp_file=$(mktemp /tmp/test-file-XXXXXX.jpg)
    dd if=/dev/zero of="$temp_file" bs=1024 count=100 2>/dev/null

    print_info "Created temp test file: $temp_file (100KB)"

    # Upload to MinIO presigned URL (with verbose output)
    print_info "Uploading file to MinIO..."
    local temp_output=$(mktemp)
    local http_code=$(curl -s -w "%{http_code}" -o "$temp_output" -X PUT "$UPLOAD_URL" \
        -H "Content-Type: image/jpeg" \
        --data-binary "@$temp_file" \
        --max-time 30)

    # Show response content (if error)
    if [ "$http_code" != "200" ] && [ -s "$temp_output" ]; then
        print_info "MinIO response: $(cat $temp_output)"
    fi

    # Cleanup temp files
    rm -f "$temp_file" "$temp_output"

    if [ "$http_code" = "200" ]; then
        print_success "File uploaded to MinIO successfully (HTTP $http_code)"
        return 0
    else
        print_error "File upload to MinIO failed (HTTP $http_code)"
        print_info "Possible causes: presigned URL expired, MinIO unreachable, or URL format error"
        return 1
    fi
}

# Test 4: Complete upload
test_complete_upload() {
    print_header "Test 4: Complete Upload"

    print_info "Notifying server of upload completion..."
    local response=$(http_post "${API_BASE}/files/${FILE_ID}/complete" "{}" "$USER_TOKEN")

    if check_response "$response"; then
        local status=$(echo "$response" | jq -r '.data.status')
        local file_name=$(echo "$response" | jq -r '.data.file_name')

        print_success "Upload completed successfully"
        print_info "File Name: ${file_name}"
        print_info "Status: ${status}"
        return 0
    else
        print_error "Complete upload failed"
        return 1
    fi
}

# Test 5: Get file info
test_get_file_info() {
    print_header "Test 5: Get File Info"

    print_info "Getting file info..."
    local response=$(http_get "${API_BASE}/files/${FILE_ID}" "$USER_TOKEN")

    if check_response "$response"; then
        local file_id=$(echo "$response" | jq -r '.data.file_id')
        local file_size=$(echo "$response" | jq -r '.data.file_size')
        local file_type=$(echo "$response" | jq -r '.data.file_type')

        print_success "Get file info successful"
        print_info "File ID: ${file_id}"
        print_info "File Size: ${file_size} bytes"
        print_info "File Type: ${file_type}"
        return 0
    else
        print_error "Get file info failed"
        return 1
    fi
}

# Test 6: Generate download URL
test_generate_download_url() {
    print_header "Test 6: Generate Download URL"

    print_info "Generating download URL..."
    local response=$(http_get "${API_BASE}/files/${FILE_ID}/download?expiresMinutes=30" "$USER_TOKEN")

    if check_response "$response"; then
        DOWNLOAD_URL=$(echo "$response" | jq -r '.data.download_url')
        local expires_in=$(echo "$response" | jq -r '.data.expires_in')

        print_success "Generate download URL successful"
        print_info "Expires In: ${expires_in}s"
        return 0
    else
        print_error "Generate download URL failed"
        return 1
    fi
}

# Test 7: List user files
test_list_user_files() {
    print_header "Test 7: List User Files"

    print_info "Listing all files..."
    local response=$(http_get "${API_BASE}/files?page=1&pageSize=20" "$USER_TOKEN")

    if check_response "$response"; then
        local total=$(echo "$response" | jq -r '.data.total')
        local count=$(echo "$response" | jq -r '.data.files | length')

        print_success "List files successful"
        print_info "Total: ${total}"
        print_info "Current Page Count: ${count}"
        return 0
    else
        print_error "List files failed"
        return 1
    fi

    # Test filter by type
    print_info "Listing image type files..."
    local response2=$(http_get "${API_BASE}/files?fileType=image&page=1&pageSize=20" "$USER_TOKEN")

    if check_response "$response2"; then
        local count2=$(echo "$response2" | jq -r '.data.files | length')
        print_success "Filter by type successful (image count: ${count2})"
        return 0
    else
        print_error "Filter by type failed"
        return 1
    fi
}

# Test 8: Delete file
test_delete_file() {
    print_header "Test 8: Delete File"

    print_info "Deleting file..."
    local response=$(http_delete "${API_BASE}/files/${FILE_ID}" "$USER_TOKEN")

    if check_response "$response"; then
        local success=$(echo "$response" | jq -r '.data.success')

        if [ "$success" = "true" ]; then
            print_success "Delete file successful"
            return 0
        else
            print_error "Delete file failed (success=false)"
            return 1
        fi
    else
        print_error "Delete file failed"
        return 1
    fi

    # Verify file deleted
    print_info "Verifying file deleted..."
    local verify_response=$(http_get "${API_BASE}/files/${FILE_ID}" "$USER_TOKEN")
    local code=$(echo "$verify_response" | jq -r '.code // -1')

    if [ "$code" != "0" ]; then
        print_success "Verify file deleted successfully"
        return 0
    else
        print_error "Verification failed: file still exists"
        return 1
    fi
}

# ========================================
# Main function
# ========================================

main() {
    echo "File Service API Test Script"
    echo "Start time: $(date '+%Y-%m-%d %H:%M:%S')"

    # Check dependencies
    if ! command -v jq &> /dev/null; then
        print_error "jq not installed, please install jq first"
        exit 1
    fi

    if ! command -v curl &> /dev/null; then
        print_error "curl not installed, please install curl first"
        exit 1
    fi

    # Setup test user
    setup_test_user || exit 1

    # Run tests
    local failed=0

    test_health_check || ((failed++))
    test_generate_upload_token || ((failed++))
    test_upload_to_minio || ((failed++))
    test_complete_upload || ((failed++))
    test_get_file_info || ((failed++))
    test_generate_download_url || ((failed++))
    test_list_user_files || ((failed++))
    test_delete_file || ((failed++))

    # Summary
    print_header "Tests Complete"
    echo "End time: $(date '+%Y-%m-%d %H:%M:%S')"

    if [ $failed -eq 0 ]; then
        echo -e "${GREEN}All tests passed! ✓${NC}"
        exit 0
    else
        echo -e "${RED}Failed tests: ${failed} ✗${NC}"
        exit 1
    fi
}

main "$@"