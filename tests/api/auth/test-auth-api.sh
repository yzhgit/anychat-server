#!/bin/bash
#
# Auth Service HTTP API Test Script
# Tests authentication-related HTTP interfaces
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
TEST_PHONE="138${TIMESTAMP:(-8)}"
TEST_EMAIL="test${TIMESTAMP}@example.com"
TEST_PASSWORD="Test@123456"
TEST_DEVICE_ID="test-device-${TIMESTAMP}"
FIXED_CODE="${VERIFY_DEBUG_FIXED_CODE:-123456}"

# Global variables
ACCESS_TOKEN=""
REFRESH_TOKEN=""
USER_ID=""

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

# Check code field in JSON response
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
# Test cases
# ========================================

# 0. Health check
test_health_check() {
    print_header "0. Health Check"

    local response=$(http_get "${GATEWAY_URL}/health")
    print_info "Response: $response"

    local status=$(echo "$response" | jq -r '.status // ""')
    if [ "$status" = "ok" ]; then
        print_success "Health check passed"
        return 0
    else
        print_error "Health check failed"
        return 1
    fi
}

# 1. Send SMS verification code
test_send_sms_code() {
    print_header "1. Send SMS Verification Code"

    local phone=$(make_phone 1)
    local response=$(send_code "$phone" "sms" "register" "${TEST_DEVICE_ID}-sms")
    print_info "Response: $response"

    if check_response "$response"; then
        local code_id=$(echo "$response" | jq -r '.data.codeId // empty')
        print_success "SMS verification code sent successfully"
        print_info "CodeId: ${code_id}"
        return 0
    fi

    return 1
}

# 2. Send email verification code
test_send_email_code() {
    print_header "2. Send Email Verification Code"

    local email="send_${TIMESTAMP}@example.com"
    local response=$(send_code "$email" "email" "register" "${TEST_DEVICE_ID}-email")
    print_info "Response: $response"

    if check_response "$response"; then
        local code_id=$(echo "$response" | jq -r '.data.codeId // empty')
        print_success "Email verification code sent successfully"
        print_info "CodeId: ${code_id}"
        return 0
    fi

    return 1
}

# 3. Invalid target format
test_invalid_target_format() {
    print_header "3. Invalid Target Format"

    local response=$(send_code "invalid-phone" "sms" "register" "${TEST_DEVICE_ID}-invalid")
    print_info "Response: $response"

    local code=$(echo "$response" | jq -r '.code // -1')
    if [ "$code" != "0" ]; then
        print_success "Invalid target format correctly rejected"
        return 0
    fi

    print_error "Invalid target format should not succeed"
    return 1
}

# 4. Rate limit
test_rate_limit() {
    print_header "4. Rate Limit"

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
        print_success "Rate limit active (triggered ${failed_count} times)"
        return 0
    fi

    print_error "Rate limit not triggered"
    return 1
}

# 5. Register with wrong verification code
test_register_with_wrong_code() {
    print_header "5. Register with Wrong Verification Code"

    local email="wrong_code_${TIMESTAMP}@example.com"
    local device_id="${TEST_DEVICE_ID}-wrong"
    local send_response=$(send_code "$email" "email" "register" "$device_id")
    print_info "Send code response: $send_response"
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
    print_info "Response: $response"

    local code=$(echo "$response" | jq -r '.code // -1')
    if [ "$code" != "0" ]; then
        print_success "Wrong verification code correctly rejected"
        return 0
    fi

    print_error "Wrong verification code should not register successfully"
    return 1
}

# 6. Register with fixed verification code
test_register_with_fixed_code() {
    print_header "6. Register with Fixed Verification Code"

    local email="success_${TIMESTAMP}@example.com"
    local device_id="${TEST_DEVICE_ID}-success"
    local send_response=$(send_code "$email" "email" "register" "$device_id")
    print_info "Send code response: $send_response"
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
    print_info "Response: $response"

    if check_response "$response"; then
        print_success "Fixed verification code registration successful"
        return 0
    fi

    return 1
}

# 7. Send reset password verification code
test_send_reset_password_code() {
    print_header "7. Send Reset Password Verification Code"

    local phone=$(make_phone 2)
    local response=$(send_code "$phone" "sms" "reset_password" "${TEST_DEVICE_ID}-reset")
    print_info "Response: $response"

    if check_response "$response"; then
        print_success "Reset password verification code sent successfully"
        return 0
    fi

    return 1
}

# 8. User registration
test_register() {
    print_header "8. User Registration"

    local send_response=$(send_register_code)
    print_info "Send code response: $send_response"
    if ! check_response "$send_response"; then
        return 1
    fi

    local data=$(cat <<EOF
{
    "phoneNumber": "${TEST_PHONE}",
    "password": "${TEST_PASSWORD}",
    "verifyCode": "${FIXED_CODE}",
    "nickname": "TestUser${TIMESTAMP}",
    "deviceType": "iOS",
    "deviceId": "${TEST_DEVICE_ID}",
    "clientVersion": "1.0.0"
}
EOF
)

    print_info "Registration info: phone=${TEST_PHONE}"

    local response=$(http_post "${API_BASE}/auth/register" "$data")
    print_info "Response: $response"

    if check_response "$response"; then
        USER_ID=$(echo "$response" | jq -r '.data.userId // .data.user_id // empty')
        ACCESS_TOKEN=$(echo "$response" | jq -r '.data.accessToken // .data.access_token // empty')
        REFRESH_TOKEN=$(echo "$response" | jq -r '.data.refreshToken // .data.refresh_token // empty')

        if [ -z "$USER_ID" ] || [ "$USER_ID" = "null" ]; then
            print_error "Cannot get user ID"
            return 1
        fi

        print_success "Registration successful"
        print_info "User ID: ${USER_ID}"
        print_info "AccessToken: ${ACCESS_TOKEN:0:20}..."
        return 0
    else
        return 1
    fi
}

# 9. User login
test_login() {
    print_header "9. User Login"

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

    print_info "Login info: account=${TEST_PHONE}"

    local response=$(http_post "${API_BASE}/auth/login" "$data")
    print_info "Response: $response"

    if check_response "$response"; then
        USER_ID=$(echo "$response" | jq -r '.data.userId // .data.user_id // empty')
        ACCESS_TOKEN=$(echo "$response" | jq -r '.data.accessToken // .data.access_token // empty')
        REFRESH_TOKEN=$(echo "$response" | jq -r '.data.refreshToken // .data.refresh_token // empty')

        if [ -z "$USER_ID" ] || [ "$USER_ID" = "null" ]; then
            print_error "Cannot get user ID"
            return 1
        fi

        print_success "Login successful"
        print_info "User ID: ${USER_ID}"
        return 0
    else
        return 1
    fi
}

# 10. Change password
test_change_password() {
    print_header "10. Change Password"

    local new_password="NewPass@123456"
    local data=$(cat <<EOF
{
    "deviceId": "${TEST_DEVICE_ID}",
    "oldPassword": "${TEST_PASSWORD}",
    "newPassword": "${new_password}"
}
EOF
)

    print_info "Changing password"

    local response=$(http_post "${API_BASE}/auth/password/change" "$data" "$ACCESS_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        print_success "Password changed successfully"
        TEST_PASSWORD="${new_password}"
        return 0
    else
        return 1
    fi
}

# 11. Login with new password
test_login_with_new_password() {
    print_header "11. Login with New Password"

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

    print_info "Login with new password"

    local response=$(http_post "${API_BASE}/auth/login" "$data")
    print_info "Response: $response"

    if check_response "$response"; then
        ACCESS_TOKEN=$(echo "$response" | jq -r '.data.accessToken // .data.access_token // empty')
        REFRESH_TOKEN=$(echo "$response" | jq -r '.data.refreshToken // .data.refresh_token // empty')

        print_success "New password login successful"
        return 0
    else
        return 1
    fi
}

# 12. Refresh token
test_refresh_token() {
    print_header "12. Refresh Token"

    local data=$(cat <<EOF
{
    "refreshToken": "${REFRESH_TOKEN}"
}
EOF
)

    print_info "Using RefreshToken to refresh"

    local response=$(http_post "${API_BASE}/auth/refresh" "$data")
    print_info "Response: $response"

    if check_response "$response"; then
        local new_access=$(echo "$response" | jq -r '.data.accessToken // .data.access_token // empty')
        local new_refresh=$(echo "$response" | jq -r '.data.refreshToken // .data.refresh_token // empty')

        if [ -z "$new_access" ] || [ "$new_access" = "null" ]; then
            print_error "Cannot get new AccessToken"
            return 1
        fi

        ACCESS_TOKEN="$new_access"
        REFRESH_TOKEN="$new_refresh"

        print_success "Token refreshed successfully"
        print_info "New AccessToken: ${ACCESS_TOKEN:0:20}..."
        return 0
    else
        return 1
    fi
}

# 13. Logout
test_logout() {
    print_header "13. Logout"

    local data=$(cat <<EOF
{
    "deviceId": "${TEST_DEVICE_ID}"
}
EOF
)

    local response=$(http_post "${API_BASE}/auth/logout" "$data" "$ACCESS_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        print_success "Logout successful"
        return 0
    else
        return 1
    fi
}

# ========================================
# Main function
# ========================================

main() {
    echo -e "${GREEN}"
    echo "╔═══════════════════════════════════════════╗"
    echo "║   Auth Service API Test Script           ║"
    echo "╚═══════════════════════════════════════════╝"
    echo -e "${NC}"

    echo "Test environment: ${GATEWAY_URL}"
    echo "Start time: $(date '+%Y-%m-%d %H:%M:%S')"
    echo ""

    # Check dependencies
    if ! command -v jq &> /dev/null; then
        print_error "jq tool needs to be installed: apt-get install jq or brew install jq"
        exit 1
    fi

    # Execute tests
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

    # Output test results
    echo ""
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}Test Results${NC}"
    echo -e "${YELLOW}========================================${NC}"
    echo "End time: $(date '+%Y-%m-%d %H:%M:%S')"

    if [ $failed -eq 0 ]; then
        echo -e "${GREEN}All tests passed! ✓${NC}"
        exit 0
    else
        echo -e "${RED}Failed tests: ${failed} ✗${NC}"
        exit 1
    fi
}

# Run main function
main "$@"
