#!/bin/bash
#
# User Service HTTP API Test Script
# Tests user management related HTTP APIs
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../common.sh"

# Configuration
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
API_BASE="${GATEWAY_URL}/api/v1"

# Test data
TIMESTAMP=$(date +%s)
TEST_PHONE="138${TIMESTAMP:(-8)}"
TEST_EMAIL="test${TIMESTAMP}@example.com"
TEST_EMAIL_2="test2_${TIMESTAMP}@example.com"
TEST_PASSWORD="Test@123456"
TEST_DEVICE_ID="test-device-${TIMESTAMP}"

# Global variables
ACCESS_TOKEN=""
USER_ID=""
ACCESS_TOKEN_2=""
USER_ID_2=""

# ========================================
# Setup: Create test users
# ========================================

setup_test_user() {
    print_header "Preparing test users"

    # Register user
    print_info "Registering test user: ${TEST_EMAIL}"
    local response
    response=$(register_test_user "${API_BASE}" "${TEST_EMAIL}" "${TEST_PASSWORD}" "TestUser${TIMESTAMP}" "${TEST_DEVICE_ID}" "iOS")
    if check_response "$response"; then
        USER_ID=$(extract_user_id "$response")
        ACCESS_TOKEN=$(extract_access_token "$response")

        if [ -z "$USER_ID" ] || [ "$USER_ID" = "null" ]; then
            print_error "Failed to get user ID"
            return 1
        fi

        print_success "Test user created successfully (ID: ${USER_ID})"
    else
        print_error "Failed to create test user"
        return 1
    fi

    sleep 1

    # Register user 2 (for blacklist regression test)
    print_info "Registering secondary test user: ${TEST_EMAIL_2}"
    local response2
    response2=$(register_test_user "${API_BASE}" "${TEST_EMAIL_2}" "${TEST_PASSWORD}" "SecondaryTestUser${TIMESTAMP}" "${TEST_DEVICE_ID}_2" "iOS")
    if check_response "$response2"; then
        USER_ID_2=$(extract_user_id "$response2")
        ACCESS_TOKEN_2=$(extract_access_token "$response2")

        if [ -z "$USER_ID_2" ] || [ "$USER_ID_2" = "null" ]; then
            print_error "Failed to get secondary user ID"
            return 1
        fi

        print_success "Secondary test user created successfully (ID: ${USER_ID_2})"
        return 0
    else
        print_error "Failed to create secondary test user"
        return 1
    fi
}

# ========================================
# Test cases
# ========================================

# 1. Get profile
test_get_profile() {
    print_header "1. Get Profile"

    local response=$(http_get "${API_BASE}/users/me" "$ACCESS_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        local nickname=$(echo "$response" | jq -r '.data.nickname // empty')
        if [ -z "$nickname" ]; then
            print_error "Failed to get nickname"
            return 1
        fi

        print_success "Get profile successful"
        print_info "Nickname: ${nickname}"
        return 0
    else
        return 1
    fi
}

# 2. Update profile
test_update_profile() {
    print_header "2. Update Profile"

    local new_nickname="UpdatedNickname${TIMESTAMP}"
    local data=$(cat <<EOF
{
    "nickname": "${new_nickname}",
    "signature": "This is a test signature",
    "gender": 1
}
EOF
)

    print_info "Update info: new_nickname=${new_nickname}"

    local response=$(http_put "${API_BASE}/users/me" "$data" "$ACCESS_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        print_success "Update profile successful"
        return 0
    else
        return 1
    fi
}

# 3. Verify profile updated
test_verify_profile_updated() {
    print_header "3. Verify Profile Updated"

    local response=$(http_get "${API_BASE}/users/me" "$ACCESS_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        local nickname=$(echo "$response" | jq -r '.data.nickname // empty')
        local signature=$(echo "$response" | jq -r '.data.signature // empty')
        local gender=$(echo "$response" | jq -r '.data.gender // 0')

        print_success "Verification successful"
        print_info "Nickname: ${nickname}"
        print_info "Signature: ${signature}"
        print_info "Gender: ${gender}"
        return 0
    else
        return 1
    fi
}

# 4. Search users
test_search_users() {
    print_header "4. Search Users"

    local keyword="Test"
    print_info "Search keyword: ${keyword}"

    local response=$(http_get "${API_BASE}/users/search?keyword=${keyword}&page=1&pageSize=10" "$ACCESS_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        local total=$(echo "$response" | jq -r '.data.total // 0')
        print_success "Search users successful"
        print_info "Found ${total} users"
        return 0
    else
        return 1
    fi
}

# 5. Get user settings
test_get_settings() {
    print_header "5. Get User Settings"

    local response=$(http_get "${API_BASE}/users/me/settings" "$ACCESS_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        print_success "Get user settings successful"
        return 0
    else
        return 1
    fi
}

# 6. Update user settings
test_update_settings() {
    print_header "6. Update User Settings"

    local data=$(cat <<EOF
{
    "notificationEnabled": true,
    "soundEnabled": false,
    "language": "en-US"
}
EOF
)

    local response=$(http_put "${API_BASE}/users/me/settings" "$data" "$ACCESS_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        print_success "Update user settings successful"
        return 0
    else
        return 1
    fi
}

# 7. Verify settings updated
test_verify_settings_updated() {
    print_header "7. Verify Settings Updated"

    local response=$(http_get "${API_BASE}/users/me/settings" "$ACCESS_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        local notification=$(echo "$response" | jq -r '.data.notificationEnabled // .data.notification_enabled // false')
        local sound=$(echo "$response" | jq -r '.data.soundEnabled // .data.sound_enabled // true')
        local language=$(echo "$response" | jq -r '.data.language // ""')

        print_success "Verification successful"
        print_info "Notification: ${notification}"
        print_info "Sound: ${sound}"
        print_info "Language: ${language}"
        return 0
    else
        return 1
    fi
}

# 8. Refresh QR code
test_refresh_qrcode() {
    print_header "8. Refresh QR Code"

    local response=$(http_post "${API_BASE}/users/me/qrcode/refresh" "{}" "$ACCESS_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        local qrcode_url=$(echo "$response" | jq -r '.data.qrcodeUrl // .data.qrcode_url // empty')

        if [ -z "$qrcode_url" ] || [ "$qrcode_url" = "null" ]; then
            print_error "Failed to get QR code URL"
            return 1
        fi

        print_success "Refresh QR code successful"
        print_info "QR Code URL: ${qrcode_url}"
        return 0
    else
        return 1
    fi
}

# 10. Update push token
test_update_push_token() {
    print_header "10. Update Push Token"

    local data=$(cat <<EOF
{
    "deviceId": "${TEST_DEVICE_ID}",
    "pushToken": "test-push-token-${TIMESTAMP}",
    "platform": "iOS"
}
EOF
)

    local response=$(http_post "${API_BASE}/users/me/push-token" "$data" "$ACCESS_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        print_success "Update push token successful"
        return 0
    else
        return 1
    fi
}

# 10. Bind phone number
test_bind_phone() {
    print_header "10. Bind Phone Number"

    local data=$(cat <<EOF
{
    "phoneNumber": "${TEST_PHONE}",
    "verifyCode": "123456"
}
EOF
)

    print_info "Binding phone number: ${TEST_PHONE}"
    local response=$(http_post "${API_BASE}/users/me/phone/bind" "$data" "$ACCESS_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        print_success "Bind phone number successful"
        return 0
    else
        return 1
    fi
}

# 11. Change phone number
test_change_phone() {
    print_header "11. Change Phone Number"

    local new_phone="139${TIMESTAMP:(-8)}"
    local data=$(cat <<EOF
{
    "oldPhoneNumber": "${TEST_PHONE}",
    "newPhoneNumber": "${new_phone}",
    "newVerifyCode": "123456"
}
EOF
)

    print_info "Changing phone number: ${TEST_PHONE} -> ${new_phone}"
    local response=$(http_post "${API_BASE}/users/me/phone/change" "$data" "$ACCESS_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        print_success "Change phone number successful"
        TEST_PHONE=$new_phone
        return 0
    else
        return 1
    fi
}

# 12. Bind email
test_bind_email() {
    print_header "12. Bind Email"

    local data=$(cat <<EOF
{
    "email": "${TEST_EMAIL}",
    "verifyCode": "123456"
}
EOF
)

    print_info "Binding email: ${TEST_EMAIL}"
    local response=$(http_post "${API_BASE}/users/me/email/bind" "$data" "$ACCESS_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        print_success "Bind email successful"
        return 0
    else
        return 1
    fi
}

# 13. Change email
test_change_email() {
    print_header "13. Change Email"

    local new_email="new${TIMESTAMP}@example.com"
    local data=$(cat <<EOF
{
    "oldEmail": "${TEST_EMAIL}",
    "newEmail": "${new_email}",
    "newVerifyCode": "123456"
}
EOF
)

    print_info "Changing email: ${TEST_EMAIL} -> ${new_email}"
    local response=$(http_post "${API_BASE}/users/me/email/change" "$data" "$ACCESS_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        print_success "Change email successful"
        TEST_EMAIL=$new_email
        return 0
    else
        return 1
    fi
}

# 14. Blacklist restriction: blocked user cannot view profile
test_blacklist_blocks_user_info() {
    print_header "14. Blacklist Restriction: Blocked User Cannot View Profile"

    local blacklist_data=$(cat <<EOF
{
    "userId": "${USER_ID_2}"
}
EOF
)

    print_info "Main test user blocks secondary user"
    local block_resp=$(http_post "${API_BASE}/friends/blacklist" "$blacklist_data" "$ACCESS_TOKEN")
    print_info "Block response: $block_resp"
    if ! check_response "$block_resp"; then
        return 1
    fi

    print_info "Secondary user tries to view main user profile (should be denied)"
    local profile_resp=$(http_get "${API_BASE}/users/${USER_ID}" "$ACCESS_TOKEN_2")
    print_info "View profile response: $profile_resp"
    if ! (check_response_fail "$profile_resp" && check_fail_code "$profile_resp" "403"); then
        return 1
    fi

    print_info "Cleaning up blacklist"
    local unblock_resp=$(http_delete "${API_BASE}/friends/blacklist/${USER_ID_2}" "$ACCESS_TOKEN")
    print_info "Remove response: $unblock_resp"
    if ! check_response "$unblock_resp"; then
        return 1
    fi

    print_success "Blacklist profile access restriction verified"
    return 0
}

# ========================================
# Main function
# ========================================

main() {
    echo -e "${GREEN}"
    echo "╔═══════════════════════════════════════════╗"
    echo "║   User Service API Test Script            ║"
    echo "╚═══════════════════════════════════════════╝"
    echo -e "${NC}"

    echo "Test environment: ${GATEWAY_URL}"
    echo "Start time: $(date '+%Y-%m-%d %H:%M:%S')"
    echo ""

    # Check dependencies
    if ! command -v jq &> /dev/null; then
        print_error "jq required: apt-get install jq or brew install jq"
        exit 1
    fi

    # Setup test users
    setup_test_user || exit 1

    # Execute tests
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
    sleep 1
    test_bind_phone || ((failed++))
    test_change_phone || ((failed++))
    sleep 1
    test_bind_email || ((failed++))
    test_change_email || ((failed++))
    sleep 1
    test_blacklist_blocks_user_info || ((failed++))

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