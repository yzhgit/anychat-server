#!/bin/bash
#
# Friend Service HTTP API Test Script
# Tests friend management related HTTP APIs
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../common.sh"

# Configuration
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
API_BASE="${GATEWAY_URL}/api/v1"

# Test data
TIMESTAMP=$(date +%s)
TEST_PHONE_1="138${TIMESTAMP:(-8)}"
TEST_PHONE_2="139${TIMESTAMP:(-8)}"
TEST_EMAIL_1="user1_${TIMESTAMP}@example.com"
TEST_EMAIL_2="user2_${TIMESTAMP}@example.com"
TEST_PASSWORD="Test@123456"
TEST_DEVICE_ID="test-device-${TIMESTAMP}"

# Global variables
USER1_TOKEN=""
USER2_TOKEN=""
USER1_ID=""
USER2_ID=""
FRIEND_REQUEST_ID=""
USER2_CONVERSATION_ID=""
POST_UNBLOCK_REQUEST_ID=""

# ========================================
# Setup: Create test users
# ========================================

setup_test_users() {
    print_header "Preparing test users"

    # Register user 1
    print_info "Registering user 1: ${TEST_EMAIL_1}"
    local response1
    response1=$(register_test_user "${API_BASE}" "${TEST_EMAIL_1}" "${TEST_PASSWORD}" "TestUser1_${TIMESTAMP}" "${TEST_DEVICE_ID}_1" "iOS")
    if check_response "$response1"; then
        USER1_ID=$(extract_user_id "$response1")
        USER1_TOKEN=$(extract_access_token "$response1")
        print_success "User 1 registered successfully (ID: ${USER1_ID})"
    else
        print_error "User 1 registration failed"
        return 1
    fi

    sleep 1

    # Register user 2
    print_info "Registering user 2: ${TEST_EMAIL_2}"
    local response2
    response2=$(register_test_user "${API_BASE}" "${TEST_EMAIL_2}" "${TEST_PASSWORD}" "TestUser2_${TIMESTAMP}" "${TEST_DEVICE_ID}_2" "iOS")
    if check_response "$response2"; then
        USER2_ID=$(extract_user_id "$response2")
        USER2_TOKEN=$(extract_access_token "$response2")
        print_success "User 2 registered successfully (ID: ${USER2_ID})"
    else
        print_error "User 2 registration failed"
        return 1
    fi
}

# ========================================
# Test cases
# ========================================

# 1. Send friend request
test_send_friend_request() {
    print_header "1. Send Friend Request"

    local data=$(cat <<EOF
{
    "userId": "${USER2_ID}",
    "message": "Hello, I'd like to add you as a friend",
    "source": "search"
}
EOF
)

    print_info "User 1 sends friend request to User 2"

    local response=$(http_post "${API_BASE}/friends/requests" "$data" "$USER1_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        # Note: protobuf request_id is requestId (camelCase) in JSON
        FRIEND_REQUEST_ID=$(echo "$response" | jq -r '.data.requestId // .data.request_id // empty')
        local auto_accepted=$(echo "$response" | jq -r '.data.autoAccepted // .data.auto_accepted // false')

        if [ -z "$FRIEND_REQUEST_ID" ] || [ "$FRIEND_REQUEST_ID" = "null" ]; then
            print_error "Failed to get request ID, response data: $(echo "$response" | jq -r '.data')"
            return 1
        fi

        print_success "Send friend request successful"
        print_info "Request ID: ${FRIEND_REQUEST_ID}"
        print_info "Auto accepted: ${auto_accepted}"
        return 0
    else
        return 1
    fi
}

# 2. Get received friend requests
test_get_received_requests() {
    print_header "2. Get Received Friend Requests"

    print_info "User 2 gets received friend requests"

    local response=$(http_get "${API_BASE}/friends/requests?type=received" "$USER2_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        local total=$(echo "$response" | jq -r '.data.total // 0')
        print_success "Get friend requests successful"
        print_info "Received ${total} friend requests"

        # Fallback: if FRIEND_REQUEST_ID not obtained before, extract from list
        if [ -z "$FRIEND_REQUEST_ID" ] || [ "$FRIEND_REQUEST_ID" = "null" ]; then
            FRIEND_REQUEST_ID=$(echo "$response" | jq -r '.data.requests[0].id // empty')
            if [ -n "$FRIEND_REQUEST_ID" ] && [ "$FRIEND_REQUEST_ID" != "null" ]; then
                print_info "Obtained request ID from list: ${FRIEND_REQUEST_ID}"
            fi
        fi

        return 0
    else
        return 1
    fi
}

# 3. Get sent friend requests
test_get_sent_requests() {
    print_header "3. Get Sent Friend Requests"

    print_info "User 1 gets sent friend requests"

    local response=$(http_get "${API_BASE}/friends/requests?type=sent" "$USER1_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        local total=$(echo "$response" | jq -r '.data.total // 0')
        print_success "Get sent friend requests successful"
        print_info "Sent ${total} friend requests"
        return 0
    else
        return 1
    fi
}

# 4. Accept friend request
test_accept_friend_request() {
    print_header "4. Accept Friend Request"

    # Check if FRIEND_REQUEST_ID is valid
    if [ -z "$FRIEND_REQUEST_ID" ] || [ "$FRIEND_REQUEST_ID" = "null" ]; then
        print_error "Invalid request ID, skip this test"
        print_info "Tip: Ensure previous tests executed successfully"
        return 1
    fi

    local data=$(cat <<EOF
{
    "action": "accept"
}
EOF
)

    print_info "User 2 accepts friend request (ID: ${FRIEND_REQUEST_ID})"

    local response=$(http_put "${API_BASE}/friends/requests/${FRIEND_REQUEST_ID}" "$data" "$USER2_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        print_success "Accept friend request successful"
        return 0
    else
        return 1
    fi
}

# 5. Get friend list
test_get_friend_list() {
    print_header "5. Get Friend List"

    # User 1 gets friend list
    print_info "User 1 gets friend list"
    local response1=$(http_get "${API_BASE}/friends" "$USER1_TOKEN")
    print_info "Response: $response1"

    if check_response "$response1"; then
        local total1=$(echo "$response1" | jq -r '.data.total // 0')
        print_success "User 1 get friend list successful (${total1} friends)"
    else
        return 1
    fi

    # User 2 gets friend list
    print_info "User 2 gets friend list"
    local response2=$(http_get "${API_BASE}/friends" "$USER2_TOKEN")
    print_info "Response: $response2"

    if check_response "$response2"; then
        local total2=$(echo "$response2" | jq -r '.data.total // 0')
        print_success "User 2 get friend list successful (${total2} friends)"
        return 0
    else
        return 1
    fi
}

# 5.1 Prepare User2 to User1 single conversation ID
prepare_user2_single_conversation() {
    print_header "5.1 Prepare Single Conversation ID"

    local max_retry=5
    local i=1
    while [ $i -le $max_retry ]; do
        local response=$(http_get "${API_BASE}/conversations?limit=100" "$USER2_TOKEN")
        print_info "Query ${i} - conversation list"

        if check_response "$response"; then
            USER2_CONVERSATION_ID=$(echo "$response" | jq -r --arg uid "$USER1_ID" '
                .data.conversations[]? |
                select((.conversationType // .conversation_type) == "single" and (.targetId // .target_id) == $uid) |
                (.conversationId // .conversation_id)
            ' | head -n 1)

            if [ -n "$USER2_CONVERSATION_ID" ] && [ "$USER2_CONVERSATION_ID" != "null" ]; then
                print_success "Get conversation ID successful: ${USER2_CONVERSATION_ID}"
                return 0
            fi
        fi

        sleep 1
        i=$((i + 1))
    done

    print_error "Cannot find User2 to User1 single conversation ID, subsequent message blocking tests will fail"
    return 1
}

# 6. Update friend remark
test_update_friend_remark() {
    print_header "6. Update Friend Remark"

    local data=$(cat <<EOF
{
    "remark": "My good friend"
}
EOF
)

    print_info "User 1 updates User 2's remark"

    local response=$(http_put "${API_BASE}/friends/${USER2_ID}/remark" "$data" "$USER1_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        print_success "Update friend remark successful"
        return 0
    else
        return 1
    fi
}

# 7. Incremental sync friend list
test_incremental_sync() {
    print_header "7. Incremental Sync Friend List"

    # Use past timestamp (5 minutes ago) to test incremental sync
    # This captures the friend relationship created just now
    local last_time=$(($(date +%s) - 300))
    print_info "Using timestamp for incremental sync: ${last_time}"

    local response=$(http_get "${API_BASE}/friends?lastUpdateTime=${last_time}" "$USER1_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        local total=$(echo "$response" | jq -r '.data.total // 0')
        print_success "Incremental sync successful"
        print_info "Updated ${total} friends"
        return 0
    else
        return 1
    fi
}

# 8. Add to blacklist
test_add_to_blacklist() {
    print_header "8. Add to Blacklist"

    local data=$(cat <<EOF
{
    "userId": "${USER2_ID}"
}
EOF
)

    print_info "User 1 adds User 2 to blacklist"

    local response=$(http_post "${API_BASE}/friends/blacklist" "$data" "$USER1_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        print_success "Add to blacklist successful"
        return 0
    else
        return 1
    fi
}

# 9. Get blacklist
test_get_blacklist() {
    print_header "9. Get Blacklist"

    print_info "User 1 gets blacklist"

    local response=$(http_get "${API_BASE}/friends/blacklist" "$USER1_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        local total=$(echo "$response" | jq -r '.data.total // 0')
        print_success "Get blacklist successful"
        print_info "Blacklist contains ${total} users"
        return 0
    else
        return 1
    fi
}

# 10. Verify automatic friend removal after blocking
test_blacklist_auto_remove_friend() {
    print_header "10. Verify Automatic Friend Removal After Blocking"

    local response1=$(http_get "${API_BASE}/friends" "$USER1_TOKEN")
    print_info "User 1 friend list response: $response1"
    if ! check_response "$response1"; then
        return 1
    fi

    local total1=$(echo "$response1" | jq -r '.data.total // 0')
    if [ "$total1" -ne 0 ]; then
        print_error "User 1 friend list should be empty, actual total=${total1}"
        return 1
    fi

    local response2=$(http_get "${API_BASE}/friends" "$USER2_TOKEN")
    print_info "User 2 friend list response: $response2"
    if ! check_response "$response2"; then
        return 1
    fi

    local total2=$(echo "$response2" | jq -r '.data.total // 0')
    if [ "$total2" -ne 0 ]; then
        print_error "User 2 friend list should be empty, actual total=${total2}"
        return 1
    fi

    print_success "Friend relationship automatically removed after blocking"
    return 0
}

# 11. Verify blacklist restriction: cannot send messages
test_blacklist_blocks_message() {
    print_header "11. Verify Blacklist Restriction: Cannot Send Messages"

    if [ -z "$USER2_CONVERSATION_ID" ] || [ "$USER2_CONVERSATION_ID" = "null" ]; then
        print_error "Missing conversation ID, cannot execute message blocking test"
        return 1
    fi

    local data=$(cat <<EOF
{
    "conversation_id": "${USER2_CONVERSATION_ID}",
    "content_type": "text",
    "content": "{\"text\":\"blacklist block test\"}",
    "local_id": "local-blacklist-${TIMESTAMP}"
}
EOF
)

    local response=$(http_post "${API_BASE}/messages" "$data" "$USER2_TOKEN")
    print_info "Response: $response"

    if check_response_fail "$response" && check_fail_code "$response" "403"; then
        print_success "Blacklist message blocking active (403)"
        return 0
    fi
    return 1
}

# 12. Verify blacklist restriction: cannot initiate audio/video calls
test_blacklist_blocks_call() {
    print_header "12. Verify Blacklist Restriction: Cannot Initiate Audio/Video Calls"

    local data=$(cat <<EOF
{
    "calleeId": "${USER1_ID}",
    "callType": "audio"
}
EOF
)

    local response=$(http_post "${API_BASE}/calling/calls" "$data" "$USER2_TOKEN")
    print_info "Response: $response"

    if check_response_fail "$response" && check_fail_code "$response" "403"; then
        print_success "Blacklist call blocking active (403)"
        return 0
    fi
    return 1
}

# 13. Verify blacklist restriction: cannot view user profile
test_blacklist_blocks_user_info() {
    print_header "13. Verify Blacklist Restriction: Cannot View User Profile"

    local response=$(http_get "${API_BASE}/users/${USER1_ID}" "$USER2_TOKEN")
    print_info "Response: $response"

    if check_response_fail "$response" && check_fail_code "$response" "403"; then
        print_success "Blacklist profile access restriction active (403)"
        return 0
    fi
    return 1
}

# 14. Remove from blacklist
test_remove_from_blacklist() {
    print_header "14. Remove from Blacklist"

    print_info "User 1 removes User 2 from blacklist"

    local response=$(http_delete "${API_BASE}/friends/blacklist/${USER2_ID}" "$USER1_TOKEN")
    print_info "Response: $response"

    if check_response "$response"; then
        print_success "Remove from blacklist successful"
        return 0
    else
        return 1
    fi
}

# 15. Verify not friends after removing from blacklist (won't auto restore)
test_verify_not_friend_after_unblock() {
    print_header "15. Verify Not Friends After Removing from Blacklist"

    local response1=$(http_get "${API_BASE}/friends" "$USER1_TOKEN")
    print_info "User 1 friend list response: $response1"
    if ! check_response "$response1"; then
        return 1
    fi

    local total1=$(echo "$response1" | jq -r '.data.total // 0')
    if [ "$total1" -ne 0 ]; then
        print_error "User 1 friend list should be empty, actual total=${total1}"
        return 1
    fi

    local response2=$(http_get "${API_BASE}/friends" "$USER2_TOKEN")
    print_info "User 2 friend list response: $response2"
    if ! check_response "$response2"; then
        return 1
    fi

    local total2=$(echo "$response2" | jq -r '.data.total // 0')
    if [ "$total2" -ne 0 ]; then
        print_error "User 2 friend list should be empty, actual total=${total2}"
        return 1
    fi

    print_success "Friend relationship not auto restored after removing from blacklist (as expected)"
    return 0
}

# 16. Verify can send friend request after removing from blacklist
test_send_friend_request_after_unblock() {
    print_header "16. Verify Can Send Friend Request After Removing from Blacklist"

    local data=$(cat <<EOF
{
    "userId": "${USER1_ID}",
    "message": "Re-applying friend after unblock",
    "source": "search"
}
EOF
)

    local response=$(http_post "${API_BASE}/friends/requests" "$data" "$USER2_TOKEN")
    print_info "Response: $response"

    if ! check_response "$response"; then
        return 1
    fi

    POST_UNBLOCK_REQUEST_ID=$(echo "$response" | jq -r '.data.requestId // .data.request_id // empty')
    if [ -z "$POST_UNBLOCK_REQUEST_ID" ] || [ "$POST_UNBLOCK_REQUEST_ID" = "null" ]; then
        print_error "Failed to get re-application requestId"
        return 1
    fi

    print_success "Can send friend request after removing from blacklist"
    print_info "New request ID: ${POST_UNBLOCK_REQUEST_ID}"
    return 0
}

# ========================================
# Main function
# ========================================

main() {
    echo -e "${GREEN}"
    echo "╔═══════════════════════════════════════════╗"
    echo "║   Friend Service API Test Script          ║"
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
    setup_test_users || exit 1

    # Execute tests
    local failed=0

    test_send_friend_request || ((failed++))
    sleep 1
    test_get_received_requests || ((failed++))
    test_get_sent_requests || ((failed++))
    test_accept_friend_request || ((failed++))
    sleep 1
    test_get_friend_list || ((failed++))
    prepare_user2_single_conversation || ((failed++))
    test_update_friend_remark || ((failed++))
    sleep 1
    test_incremental_sync || ((failed++))
    test_add_to_blacklist || ((failed++))
    test_get_blacklist || ((failed++))
    test_blacklist_auto_remove_friend || ((failed++))
    test_blacklist_blocks_message || ((failed++))
    test_blacklist_blocks_call || ((failed++))
    test_blacklist_blocks_user_info || ((failed++))
    test_remove_from_blacklist || ((failed++))
    test_verify_not_friend_after_unblock || ((failed++))
    test_send_friend_request_after_unblock || ((failed++))

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