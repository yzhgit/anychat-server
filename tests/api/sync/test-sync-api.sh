#!/bin/bash
#
# Sync Service HTTP API Test Script
# Tests data sync related HTTP APIs
#
# Usage:
#   ./test-sync-api.sh
#   GATEWAY_URL=http://localhost:8080 ./test-sync-api.sh
#
# Notes:
#   Sync service is a stateless aggregation layer that calls 
#   friend/group/conversation/message services to get incremental data.
#   This script validates API response structure and error handling.
#   For empty accounts, full sync should return empty set not error.
#

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
API_BASE="${GATEWAY_URL}/api/v1"

# Test data
TIMESTAMP=$(date +%s)
TEST_EMAIL="sync_u1_${TIMESTAMP}@example.com"
TEST_PASSWORD="Test@123456"
TEST_DEVICE_ID="sync-test-device-${TIMESTAMP}"

# Global variables
USER_TOKEN=""
USER_ID=""
PASS=0
FAIL=0

# ────────────────────────────────────────
# Utility functions
# ────────────────────────────────────────

print_header() {
    echo -e "\n${YELLOW}========================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}========================================${NC}"
}

pass() {
    echo -e "  ${GREEN}✓ PASS${NC}: $1"
    PASS=$((PASS + 1))
}

fail() {
    echo -e "  ${RED}✗ FAIL${NC}: $1"
    echo -e "  ${RED}  Details: $2${NC}"
    FAIL=$((FAIL + 1))
}

check_http_status() {
    local desc="$1"
    local expected="$2"
    local actual="$3"
    local body="$4"

    if [ "$actual" = "$expected" ]; then
        pass "$desc (HTTP $actual)"
    else
        fail "$desc" "Expected HTTP $expected, actual HTTP $actual, response: $body"
    fi
}

check_json_field() {
    local desc="$1"
    local value="$2"
    local expected="$3"

    if [ "$value" = "$expected" ]; then
        pass "$desc"
    else
        fail "$desc" "Expected '$expected', actual '$value'"
    fi
}

# ────────────────────────────────────────
# Register and login user
# ────────────────────────────────────────

setup_user() {
    print_header "Initializing test user"

    # Register
    local reg_resp
    reg_resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\",\"verifyCode\":\"123456\",\"nickname\":\"SyncTestUser\",\"deviceId\":\"${TEST_DEVICE_ID}\",\"deviceType\":\"Web\",\"clientVersion\":\"1.0.0\"}")
    local reg_status
    reg_status=$(echo "$reg_resp" | tail -1)
    local reg_body
    reg_body=$(echo "$reg_resp" | head -n -1)

    if [ "$reg_status" != "200" ]; then
        echo -e "${RED}User registration failed (HTTP $reg_status): $reg_body${NC}"
        exit 1
    fi

    # Login
    local login_resp
    login_resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"account\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\",\"deviceId\":\"${TEST_DEVICE_ID}\",\"deviceType\":\"Web\",\"clientVersion\":\"1.0.0\"}")
    local login_status
    login_status=$(echo "$login_resp" | tail -1)
    local login_body
    login_body=$(echo "$login_resp" | head -n -1)

    if [ "$login_status" != "200" ]; then
        echo -e "${RED}Login failed (HTTP $login_status): $login_body${NC}"
        exit 1
    fi

    USER_TOKEN=$(echo "$login_body" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['accessToken'])" 2>/dev/null || \
                 echo "$login_body" | grep -o '"accessToken":"[^"]*"' | head -1 | cut -d'"' -f4)
    USER_ID=$(echo "$login_body" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['userId'])" 2>/dev/null || \
              echo "$login_body" | grep -o '"userId":"[^"]*"' | head -1 | cut -d'"' -f4)

    if [ -z "$USER_TOKEN" ]; then
        echo -e "${RED}Cannot get accessToken${NC}"
        exit 1
    fi
    echo "  User registered and logged in successfully (ID: ${USER_ID})"
}

# ────────────────────────────────────────
# Test cases
# ────────────────────────────────────────

test_full_sync_empty_account() {
    print_header "Test 1: Full Sync (Empty Account)"

    local resp
    resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/sync" \
        -H "Authorization: Bearer ${USER_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{"lastSyncTime":0,"conversationSeqs":[]}')
    local status
    status=$(echo "$resp" | tail -1)
    local body
    body=$(echo "$resp" | head -n -1)

    check_http_status "Full sync returns 200" "200" "$status" "$body"

    local code
    code=$(echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code','-1'))" 2>/dev/null || \
           echo "$body" | grep -o '"code":[0-9]*' | head -1 | cut -d: -f2)
    check_json_field "Response code is 0" "$code" "0"

    local sync_time
    sync_time=$(echo "$body" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('sync_time',''))" 2>/dev/null || \
                echo "$body" | grep -o '"sync_time":[0-9]*' | head -1 | cut -d: -f2)
    if [ -n "$sync_time" ] && [ "$sync_time" != "0" ]; then
        pass "Response contains sync_time"
    else
        fail "Response contains sync_time" "sync_time is empty or 0, body: $body"
    fi
}

test_incremental_sync() {
    print_header "Test 2: Incremental Sync (with lastSyncTime)"

    local last_sync=$((TIMESTAMP - 3600))
    local resp
    resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/sync" \
        -H "Authorization: Bearer ${USER_TOKEN}" \
        -H "Content-Type: application/json" \
        -d "{\"lastSyncTime\":${last_sync},\"conversationSeqs\":[]}")
    local status
    status=$(echo "$resp" | tail -1)
    local body
    body=$(echo "$resp" | head -n -1)

    check_http_status "Incremental sync returns 200" "200" "$status" "$body"

    local code
    code=$(echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code','-1'))" 2>/dev/null || \
           echo "$body" | grep -o '"code":[0-9]*' | head -1 | cut -d: -f2)
    check_json_field "Response code is 0" "$code" "0"
}

test_sync_without_auth() {
    print_header "Test 3: Unauthenticated Sync"

    local body
    body=$(curl -s -X POST "${API_BASE}/sync" \
        -H "Content-Type: application/json" \
        -d '{"lastSyncTime":0}')
    local code
    code=$(echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code','-1'))" 2>/dev/null || \
           echo "$body" | grep -o '"code":[0-9]*' | head -1 | cut -d: -f2)

    check_json_field "Unauthenticated returns code 401" "$code" "401"
}

test_sync_messages_empty() {
    print_header "Test 4: Message Sync (Empty Conversation List)"

    local resp
    resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/sync/messages" \
        -H "Authorization: Bearer ${USER_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{"conversationSeqs":[],"limitPerConversation":20}')
    local status
    status=$(echo "$resp" | tail -1)
    local body
    body=$(echo "$resp" | head -n -1)

    check_http_status "Message sync (empty list) returns 200" "200" "$status" "$body"

    local code
    code=$(echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code','-1'))" 2>/dev/null || \
           echo "$body" | grep -o '"code":[0-9]*' | head -1 | cut -d: -f2)
    check_json_field "Response code is 0" "$code" "0"
}

test_sync_messages_without_auth() {
    print_header "Test 5: Unauthenticated Message Sync"

    local body
    body=$(curl -s -X POST "${API_BASE}/sync/messages" \
        -H "Content-Type: application/json" \
        -d '{"conversationSeqs":[]}')
    local code
    code=$(echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code','-1'))" 2>/dev/null || \
           echo "$body" | grep -o '"code":[0-9]*' | head -1 | cut -d: -f2)

    check_json_field "Unauthenticated returns code 401" "$code" "401"
}

test_sync_with_nonexistent_conversation() {
    print_header "Test 6: Message Sync (Non-existent Conversation)"

    local resp
    resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/sync/messages" \
        -H "Authorization: Bearer ${USER_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{"conversationSeqs":[{"conversationId":"nonexistent-conv-id","conversationType":"single","lastSeq":0}],"limitPerConversation":20}')
    local status
    status=$(echo "$resp" | tail -1)
    local body
    body=$(echo "$resp" | head -n -1)

    # Non-existent conversation: sync service skips and returns empty list, should not error
    check_http_status "Returns 200 when conversation doesn't exist" "200" "$status" "$body"
}

test_sync_with_limit_query_param() {
    print_header "Test 7: Message Sync (via query param limit)"

    local resp
    resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/sync/messages?limit=10" \
        -H "Authorization: Bearer ${USER_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{"conversationSeqs":[]}')
    local status
    status=$(echo "$resp" | tail -1)
    local body
    body=$(echo "$resp" | head -n -1)

    check_http_status "Message sync with limit param returns 200" "200" "$status" "$body"
}

# ────────────────────────────────────────
# Summary
# ────────────────────────────────────────

print_summary() {
    echo ""
    echo -e "${YELLOW}════════════════════════════════════════${NC}"
    echo -e "Test Results: ${GREEN}${PASS} passed${NC} / ${RED}${FAIL} failed${NC}"
    echo -e "${YELLOW}════════════════════════════════════════${NC}"

    if [ $FAIL -eq 0 ]; then
        echo -e "${GREEN}All sync service tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}${FAIL} tests failed${NC}"
        exit 1
    fi
}

# ────────────────────────────────────────
# Main flow
# ────────────────────────────────────────

echo -e "${GREEN}╔══════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   Sync Service HTTP API Test             ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}"
echo "Gateway: ${GATEWAY_URL}"
echo "Time: $(date '+%Y-%m-%d %H:%M:%S')"

setup_user

test_full_sync_empty_account
test_incremental_sync
test_sync_without_auth
test_sync_messages_empty
test_sync_messages_without_auth
test_sync_with_nonexistent_conversation
test_sync_with_limit_query_param

print_summary