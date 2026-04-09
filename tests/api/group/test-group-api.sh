#!/bin/bash
#
# Group Service HTTP API Test Script
# Tests group management related HTTP APIs
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
TEST_PHONE_1="138${TIMESTAMP:(-8)}"
TEST_PHONE_2="139${TIMESTAMP:(-8)}"
TEST_PHONE_3="137${TIMESTAMP:(-8)}"
TEST_EMAIL_1="user1_${TIMESTAMP}@example.com"
TEST_EMAIL_2="user2_${TIMESTAMP}@example.com"
TEST_EMAIL_3="user3_${TIMESTAMP}@example.com"
TEST_PASSWORD="Test@123456"
TEST_DEVICE_ID="test-device-${TIMESTAMP}"
GROUP_NAME="TestGroup_${TIMESTAMP}"

# Global variables
USER1_TOKEN=""
USER2_TOKEN=""
USER3_TOKEN=""
USER1_ID=""
USER2_ID=""
USER3_ID=""
GROUP_ID=""
GROUP_CURRENT_NAME=""
JOIN_REQUEST_ID=""
TEST_PASSED=0
TEST_FAILED=0
TEST_PIN_MESSAGE_ID="test-pin-msg-${TIMESTAMP}"

# Print functions
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

# Find an existing message ID from group conversation (for pin test)
resolve_group_message_id_for_pin() {
    local retries=5
    local i

    for i in $(seq 1 $retries); do
        local sync_data="{\"conversationSeqs\":[{\"conversationId\":\"${GROUP_ID}\",\"conversationType\":\"group\",\"lastSeq\":0}],\"limitPerConversation\":20}"
        local sync_resp=$(http_post "${API_BASE}/sync/messages" "$sync_data" "$USER1_TOKEN")
        local sync_code=$(echo "$sync_resp" | jq -r '.code // empty')

        if [ "$sync_code" = "0" ]; then
            local message_id=$(echo "$sync_resp" | jq -r --arg gid "$GROUP_ID" '
                .data.conversations[]?
                | select((.conversationId // .conversation_id) == $gid and (.conversationType // .conversation_type) == "group")
                | .messages[]?
                | (.messageId // .message_id)
                | select(. != null and . != "")
            ' | head -n 1)

            if [ -n "$message_id" ] && [ "$message_id" != "null" ]; then
                echo "$message_id"
                return 0
            fi
        fi

        sleep 1
    done

    return 1
}

# Check JSON response status
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

# Setup: Create test users
setup_test_users() {
    print_header "Setup: Create Test Users"

    # Register user 1 (auto login after registration returns token)
    print_info "Registering user 1: ${TEST_EMAIL_1}"
    local data1="{\"email\":\"${TEST_EMAIL_1}\",\"password\":\"${TEST_PASSWORD}\",\"verifyCode\":\"123456\",\"nickname\":\"TestUser1_${TIMESTAMP}\",\"deviceType\":\"Web\",\"deviceId\":\"${TEST_DEVICE_ID}_1\",\"clientVersion\":\"1.0.0\"}"
    local response1=$(http_post "${API_BASE}/auth/register" "$data1")

    USER1_ID=$(echo "$response1" | jq -r '.data.userId // empty')
    USER1_TOKEN=$(echo "$response1" | jq -r '.data.accessToken // empty')

    if [ -z "$USER1_TOKEN" ] || [ "$USER1_TOKEN" = "null" ]; then
        print_error "User 1 registration failed"
        print_info "Response: $response1"
        exit 1
    fi
    print_success "User 1 registered successfully (ID: ${USER1_ID})"

    # Register user 2
    print_info "Registering user 2: ${TEST_EMAIL_2}"
    local data2="{\"email\":\"${TEST_EMAIL_2}\",\"password\":\"${TEST_PASSWORD}\",\"verifyCode\":\"123456\",\"nickname\":\"TestUser2_${TIMESTAMP}\",\"deviceType\":\"Web\",\"deviceId\":\"${TEST_DEVICE_ID}_2\",\"clientVersion\":\"1.0.0\"}"
    local response2=$(http_post "${API_BASE}/auth/register" "$data2")

    USER2_ID=$(echo "$response2" | jq -r '.data.userId // empty')
    USER2_TOKEN=$(echo "$response2" | jq -r '.data.accessToken // empty')

    if [ -z "$USER2_TOKEN" ] || [ "$USER2_TOKEN" = "null" ]; then
        print_error "User 2 registration failed"
        print_info "Response: $response2"
        exit 1
    fi
    print_success "User 2 registered successfully (ID: ${USER2_ID})"

    # Register user 3
    print_info "Registering user 3: ${TEST_EMAIL_3}"
    local data3="{\"email\":\"${TEST_EMAIL_3}\",\"password\":\"${TEST_PASSWORD}\",\"verifyCode\":\"123456\",\"nickname\":\"TestUser3_${TIMESTAMP}\",\"deviceType\":\"Web\",\"deviceId\":\"${TEST_DEVICE_ID}_3\",\"clientVersion\":\"1.0.0\"}"
    local response3=$(http_post "${API_BASE}/auth/register" "$data3")

    USER3_ID=$(echo "$response3" | jq -r '.data.userId // empty')
    USER3_TOKEN=$(echo "$response3" | jq -r '.data.accessToken // empty')

    if [ -z "$USER3_TOKEN" ] || [ "$USER3_TOKEN" = "null" ]; then
        print_error "User 3 registration failed"
        print_info "Response: $response3"
        exit 1
    fi
    print_success "User 3 registered successfully (ID: ${USER3_ID})"

    print_success "3 test users created successfully"
    print_info "User1 ID: $USER1_ID"
    print_info "User2 ID: $USER2_ID"
    print_info "User3 ID: $USER3_ID"
}

# Test 1: Health check
test_health_check() {
    print_header "Test 1: Health Check"

    local response=$(http_get "${GATEWAY_URL}/health")
    local status=$(echo "$response" | jq -r '.status // empty')

    if [ "$status" = "ok" ]; then
        print_success "Health check passed"
    else
        print_error "Health check failed"
    fi
}

# Test 2: Create group
test_create_group() {
    print_header "Test 2: Create Group"

    local data="{\"name\":\"${GROUP_NAME}\",\"memberIds\":[\"${USER2_ID}\"],\"joinVerify\":true}"
    local response=$(http_post "${API_BASE}/groups" "$data" "$USER1_TOKEN")

    GROUP_ID=$(echo "$response" | jq -r '.data.groupId // empty')

    if [ -n "$GROUP_ID" ] && [ "$GROUP_ID" != "null" ]; then
        GROUP_CURRENT_NAME=$(echo "$response" | jq -r '.data.name // empty')
        if [ -z "$GROUP_CURRENT_NAME" ] || [ "$GROUP_CURRENT_NAME" = "null" ]; then
            GROUP_CURRENT_NAME="$GROUP_NAME"
        fi
        check_response "$response" "0" "Create group"
        print_info "Group ID: $GROUP_ID"
    else
        print_error "Create group failed - cannot get group ID"
        print_info "Response: $response"
    fi
}

# Test 3: Get group info
test_get_group_info() {
    print_header "Test 3: Get Group Info"

    if [ -z "$GROUP_ID" ]; then
        print_error "Skip test - group ID is empty"
        return 1
    fi

    local response=$(http_get "${API_BASE}/groups/${GROUP_ID}" "$USER1_TOKEN")
    local name=$(echo "$response" | jq -r '.data.name // empty')

    if [ "$name" = "$GROUP_CURRENT_NAME" ]; then
        check_response "$response" "0" "Get group info"
        print_info "Group name: $name"
        print_info "Owner ID: $(echo "$response" | jq -r '.data.ownerId')"
        print_info "Member count: $(echo "$response" | jq -r '.data.memberCount')"
        print_info "My role: $(echo "$response" | jq -r '.data.myRole')"
    else
        print_error "Get group info failed"
    fi
}

# Test 4: Get group members
test_get_group_members() {
    print_header "Test 4: Get Group Members"

    if [ -z "$GROUP_ID" ]; then
        print_error "Skip test - group ID is empty"
        return 1
    fi

    local response=$(http_get "${API_BASE}/groups/${GROUP_ID}/members" "$USER1_TOKEN")
    local total=$(echo "$response" | jq -r '.data.total // 0')

    if [ "$total" -ge "2" ]; then
        check_response "$response" "0" "Get group members"
        print_info "Total members: $total"
    else
        print_error "Get group members failed - incorrect member count"
    fi
}

# Test 5: Update group info
test_update_group() {
    print_header "Test 5: Update Group Info"

    if [ -z "$GROUP_ID" ]; then
        print_error "Skip test - group ID is empty"
        return 1
    fi

    local new_name="${GROUP_NAME}_updated"
    local data="{\"name\":\"${new_name}\",\"announcement\":\"This is test announcement\"}"
    local response=$(http_put "${API_BASE}/groups/${GROUP_ID}" "$data" "$USER1_TOKEN")

    if check_response "$response" "0" "Update group info"; then
        GROUP_CURRENT_NAME="$new_name"
    fi
}

# Test 6: Invite members (requires verification)
test_invite_members() {
    print_header "Test 6: Invite Members (Requires Verification)"

    if [ -z "$GROUP_ID" ]; then
        print_error "Skip test - group ID is empty"
        return 1
    fi

    local data="{\"userIds\":[\"${USER3_ID}\"]}"
    local response=$(http_post "${API_BASE}/groups/${GROUP_ID}/members" "$data" "$USER1_TOKEN")

    check_response "$response" "0" "Invite members"
    print_info "Invited user 3 to join group (requires verification)"
}

# Test 7: Get join request list
test_get_join_requests() {
    print_header "Test 7: Get Join Request List"

    if [ -z "$GROUP_ID" ]; then
        print_error "Skip test - group ID is empty"
        return 1
    fi

    # Wait a moment to ensure request is created
    sleep 1

    local response=$(http_get "${API_BASE}/groups/${GROUP_ID}/requests?status=pending" "$USER1_TOKEN")
    local total=$(echo "$response" | jq -r '.data.total // 0')

    if [ "$total" -gt "0" ]; then
        check_response "$response" "0" "Get join request list"
        JOIN_REQUEST_ID=$(echo "$response" | jq -r '.data.requests[0].id // empty')
        print_info "Pending requests: $total"
        print_info "Request ID: $JOIN_REQUEST_ID"
    else
        print_error "Get join request list failed - no pending requests"
    fi
}

# Test 8: Process join request (accept)
test_accept_join_request() {
    print_header "Test 8: Process Join Request (Accept)"

    if [ -z "$GROUP_ID" ] || [ -z "$JOIN_REQUEST_ID" ]; then
        print_error "Skip test - group ID or request ID is empty"
        return 1
    fi

    local data="{\"accept\":true}"
    local response=$(http_put "${API_BASE}/groups/${GROUP_ID}/requests/${JOIN_REQUEST_ID}" "$data" "$USER1_TOKEN")

    check_response "$response" "0" "Accept join request"
}

# Test 9: Verify member joined
test_verify_member_joined() {
    print_header "Test 9: Verify Member Joined"

    if [ -z "$GROUP_ID" ]; then
        print_error "Skip test - group ID is empty"
        return 1
    fi

    # Wait a moment to ensure member is added
    sleep 1

    local response=$(http_get "${API_BASE}/groups/${GROUP_ID}/members" "$USER1_TOKEN")
    local total=$(echo "$response" | jq -r '.data.total // 0')

    if [ "$total" -ge "3" ]; then
        check_response "$response" "0" "Verify member joined"
        print_info "Current member count: $total"
    else
        print_error "Verification failed - incorrect member count (expected >=3, actual: $total)"
    fi
}

# Test 10: Update member role
test_update_member_role() {
    print_header "Test 10: Update Member Role to Admin"

    if [ -z "$GROUP_ID" ] || [ -z "$USER2_ID" ]; then
        print_error "Skip test - group ID or user ID is empty"
        return 1
    fi

    local data="{\"role\":\"admin\"}"
    local response=$(http_put "${API_BASE}/groups/${GROUP_ID}/members/${USER2_ID}/role" "$data" "$USER1_TOKEN")

    check_response "$response" "0" "Set member as admin"
}

# Test 11: Update member nickname
test_update_member_nickname() {
    print_header "Test 11: Update Member Nickname"

    if [ -z "$GROUP_ID" ]; then
        print_error "Skip test - group ID is empty"
        return 1
    fi

    local data="{\"nickname\":\"My group nickname\"}"
    local response=$(http_put "${API_BASE}/groups/${GROUP_ID}/nickname" "$data" "$USER1_TOKEN")

    check_response "$response" "0" "Update group nickname"
}

# Test 12: Set/Clear group remark and verify display name
test_update_group_remark() {
    print_header "Test 12: Set/Clear Group Remark and Verify Display Name"

    if [ -z "$GROUP_ID" ]; then
        print_error "Skip test - group ID is empty"
        return 1
    fi

    local remark="Product Discussion Group_${TIMESTAMP}"
    local set_data="{\"remark\":\"${remark}\"}"
    local set_response=$(http_put "${API_BASE}/group/${GROUP_ID}/remark" "$set_data" "$USER1_TOKEN")
    check_response "$set_response" "0" "Set group remark"

    # Verify: current user sees remark in group details
    local detail_with_remark=$(http_get "${API_BASE}/group/${GROUP_ID}" "$USER1_TOKEN")
    local my_display_name=$(echo "$detail_with_remark" | jq -r '.data.displayName // empty')
    if [ "$my_display_name" = "$remark" ]; then
        print_success "Group detail display name uses remark (current user)"
    else
        print_error "Group detail display name does not use remark (current user)"
        print_info "Response: $detail_with_remark"
    fi

    # Verify: current user sees remark in group list (using singular path per design doc)
    local list_with_remark=$(http_get "${API_BASE}/group/list" "$USER1_TOKEN")
    local my_list_display_name=$(echo "$list_with_remark" | jq -r --arg gid "$GROUP_ID" '
        (.data.groups // [])[]?
        | select((.groupId // .group_id) == $gid)
        | (.displayName // .display_name // empty)
    ' | head -n 1)
    if [ "$my_list_display_name" = "$remark" ]; then
        print_success "Group list display name uses remark (current user)"
    else
        print_error "Group list display name does not use remark (current user)"
        print_info "Response: $list_with_remark"
    fi

    # Verify: other members unaffected, see real group name
    local detail_other_user=$(http_get "${API_BASE}/group/${GROUP_ID}" "$USER2_TOKEN")
    local other_display_name=$(echo "$detail_other_user" | jq -r '.data.displayName // empty')
    if [ "$other_display_name" = "$GROUP_CURRENT_NAME" ]; then
        print_success "Remark only visible to self (other members see real group name)"
    else
        print_error "Remark visibility issue (other member display name incorrect)"
        print_info "Response: $detail_other_user"
    fi

    # Clear remark
    local clear_data='{"remark":""}'
    local clear_response=$(http_put "${API_BASE}/group/${GROUP_ID}/remark" "$clear_data" "$USER1_TOKEN")
    check_response "$clear_response" "0" "Clear group remark"

    # Verify: after clearing, real group name restored
    local detail_after_clear=$(http_get "${API_BASE}/group/${GROUP_ID}" "$USER1_TOKEN")
    local display_after_clear=$(echo "$detail_after_clear" | jq -r '.data.displayName // empty')
    if [ "$display_after_clear" = "$GROUP_CURRENT_NAME" ]; then
        print_success "After clearing remark, display name restored to real group name"
    else
        print_error "After clearing remark, display name not restored to real group name"
        print_info "Response: $detail_after_clear"
    fi
}

# Test 13: Remove group member
test_remove_member() {
    print_header "Test 13: Remove Group Member"

    if [ -z "$GROUP_ID" ] || [ -z "$USER3_ID" ]; then
        print_error "Skip test - group ID or user ID is empty"
        return 1
    fi

    local response=$(http_delete "${API_BASE}/groups/${GROUP_ID}/members/${USER3_ID}" "$USER1_TOKEN")

    check_response "$response" "0" "Remove group member"
}

# Test 14: Quit group
test_quit_group() {
    print_header "Test 14: User 2 Quit Group"

    if [ -z "$GROUP_ID" ]; then
        print_error "Skip test - group ID is empty"
        return 1
    fi

    local response=$(http_post "${API_BASE}/groups/${GROUP_ID}/quit" "{}" "$USER2_TOKEN")

    check_response "$response" "0" "Quit group"
}

# Test 15: Get my group list
test_get_my_groups() {
    print_header "Test 15: Get My Group List"

    local response=$(http_get "${API_BASE}/groups" "$USER1_TOKEN")
    local total=$(echo "$response" | jq -r '.data.total // 0')

    if [ "$total" -ge "1" ]; then
        check_response "$response" "0" "Get my group list"
        print_info "My joined groups count: $total"
    else
        print_error "Get group list failed"
    fi
}

# Test 16: Enable/Disable all mute
test_set_group_mute() {
    print_header "Test 16: Enable/Disable All Mute"

    if [ -z "$GROUP_ID" ]; then
        print_error "Skip test - group ID is empty"
        return 1
    fi

    local enable_data="{\"enabled\":true}"
    local enable_resp=$(http_put "${API_BASE}/groups/${GROUP_ID}/mute" "$enable_data" "$USER1_TOKEN")
    check_response "$enable_resp" "0" "Enable all mute"

    local disable_data="{\"enabled\":false}"
    local disable_resp=$(http_put "${API_BASE}/groups/${GROUP_ID}/mute" "$disable_data" "$USER1_TOKEN")
    check_response "$disable_resp" "0" "Disable all mute"
}

# Test 17: Pin/Unpin message
test_pin_unpin_message() {
    print_header "Test 17: Pin/Unpin Message"

    if [ -z "$GROUP_ID" ]; then
        print_error "Skip test - group ID is empty"
        return 1
    fi

    TEST_PIN_MESSAGE_ID=$(resolve_group_message_id_for_pin)
    if [ -z "$TEST_PIN_MESSAGE_ID" ] || [ "$TEST_PIN_MESSAGE_ID" = "null" ]; then
        print_error "Pin message failed - no pinnable group message ID found"
        return 1
    fi
    print_info "Message ID for pin: ${TEST_PIN_MESSAGE_ID}"

    local pin_data="{\"messageId\":\"${TEST_PIN_MESSAGE_ID}\"}"
    local pin_resp=$(http_post "${API_BASE}/groups/${GROUP_ID}/pin" "$pin_data" "$USER1_TOKEN")
    check_response "$pin_resp" "0" "Pin message"

    local list_resp=$(http_get "${API_BASE}/groups/${GROUP_ID}/pins" "$USER1_TOKEN")
    check_response "$list_resp" "0" "Get pinned message list"

    local total=$(echo "$list_resp" | jq -r '.data.total // 0')
    if [ "$total" -ge "1" ]; then
        print_success "Pinned list total field valid"
    else
        print_error "Pinned list total field abnormal: $total"
    fi

    local top_message_id=$(echo "$list_resp" | jq -r '.data.topMessage.messageId // empty')
    if [ "$top_message_id" = "$TEST_PIN_MESSAGE_ID" ]; then
        print_success "Pinned list topMessage returns latest pinned message"
    else
        print_error "Pinned list topMessage validation failed, expected=${TEST_PIN_MESSAGE_ID} actual=${top_message_id}"
    fi

    local unpin_resp=$(http_delete "${API_BASE}/groups/${GROUP_ID}/pin/${TEST_PIN_MESSAGE_ID}" "$USER1_TOKEN")
    check_response "$unpin_resp" "0" "Unpin message"
}

# Test 18: Dissolve group
test_dissolve_group() {
    print_header "Test 18: Dissolve Group"

    if [ -z "$GROUP_ID" ]; then
        print_error "Skip test - group ID is empty"
        return 1
    fi

    local response=$(http_delete "${API_BASE}/groups/${GROUP_ID}" "$USER1_TOKEN")

    check_response "$response" "0" "Dissolve group"
}

# Print test result summary
print_summary() {
    print_header "Test Results Summary"

    local total=$((TEST_PASSED + TEST_FAILED))
    echo -e "Total tests: $total"
    echo -e "${GREEN}Passed: $TEST_PASSED${NC}"
    echo -e "${RED}Failed: $TEST_FAILED${NC}"

    if [ $TEST_FAILED -eq 0 ]; then
        echo -e "\n${GREEN}All tests passed! ✓${NC}"
        return 0
    else
        echo -e "\n${RED}Some tests failed ✗${NC}"
        return 1
    fi
}

# Main function
main() {
    echo "======================================"
    echo "Group Service HTTP API Test"
    echo "======================================"
    echo "Start time: $(date '+%Y-%m-%d %H:%M:%S')"
    echo "Gateway URL: $GATEWAY_URL"
    echo ""

    # Check required tools
    if ! command -v jq &> /dev/null; then
        print_error "jq required: sudo apt-get install jq"
        exit 1
    fi

    if ! command -v curl &> /dev/null; then
        print_error "curl required"
        exit 1
    fi

    # Setup
    setup_test_users || exit 1

    # Execute tests
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
    test_update_group_remark
    test_remove_member
    test_quit_group
    test_get_my_groups
    test_set_group_mute
    test_pin_unpin_message
    test_dissolve_group

    # Print results
    echo ""
    echo "End time: $(date '+%Y-%m-%d %H:%M:%S')"
    print_summary
}

# Execute main function
main "$@"