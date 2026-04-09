#!/bin/bash
#
# LiveKit Calling Service API Test Script
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../common.sh"

BASE_URL="${GATEWAY_URL:-http://localhost:8080}/api/v1"

echo "=================================================="
echo "  LiveKit Calling Service API Test"
echo "=================================================="
echo ""

PASS=0
FAIL=0
TOKEN_A=""
TOKEN_B=""
USER_A_ID=""
USER_B_ID=""

# ── Helper functions ─────────────────────────────────────────

pass() { echo -e "${GREEN}✓ PASS${NC}: $1"; PASS=$((PASS + 1)); }
fail() { echo -e "${RED}✗ FAIL${NC}: $1"; FAIL=$((FAIL + 1)); }

# ── Setup test users ──────────────────────────────────────

echo "Registering test users..."
TOKEN_A=$(register_and_login_test_user "${BASE_URL}" "calling_00000001@test.com" "Test@1234" "User00000001" "calling-dev-00000001" "Web")
TOKEN_B=$(register_and_login_test_user "${BASE_URL}" "calling_00000002@test.com" "Test@1234" "User00000002" "calling-dev-00000002" "Web")

if [ -z "$TOKEN_A" ] || [ -z "$TOKEN_B" ]; then
    echo -e "${RED}Error: Cannot get test user token, skip Calling test${NC}"
    exit 0
fi

echo "User A token obtained"
echo "User B token obtained"

USER_A_ID=$(get_user_id_by_token "${BASE_URL}" "$TOKEN_A")
USER_B_ID=$(get_user_id_by_token "${BASE_URL}" "$TOKEN_B")
if [ -z "$USER_A_ID" ] || [ -z "$USER_B_ID" ]; then
    echo -e "${RED}Error: Cannot get test user ID, skip Calling test${NC}"
    exit 0
fi
echo "User A ID: ${USER_A_ID}"
echo "User B ID: ${USER_B_ID}"
echo ""

# ── Test 1: Unauthenticated call initiation (should return 401)────────────────

echo "Test 1: Unauthenticated call initiation"
RESP=$(curl -s -X POST "${BASE_URL}/calling/calls" \
    -H "Content-Type: application/json" \
    -d '{"calleeId":"user-xxx","callType":"audio"}')
CODE=$(json_code "$RESP")
if [ "$CODE" = "401" ]; then
    pass "Unauthenticated returns code 401"
else
    fail "Expected code 401, actual $CODE"
fi

# ── Test 2: Missing calleeId (should return 400)────────────────

echo "Test 2: Missing calleeId"
RESP=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BASE_URL}/calling/calls" \
    -H "Authorization: Bearer $TOKEN_A" \
    -H "Content-Type: application/json" \
    -d '{}')
if [ "$RESP" = "400" ]; then
    pass "Missing calleeId returns 400"
else
    fail "Expected 400, actual $RESP"
fi

# ── Test 3: Get call records (empty list)────────────────────

echo "Test 3: Get call records (initially empty)"
RESP=$(curl -s "${BASE_URL}/calling/calls" \
    -H "Authorization: Bearer $TOKEN_A")
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/calling/calls" \
    -H "Authorization: Bearer $TOKEN_A")
if [ "$HTTP_CODE" = "200" ]; then
    pass "Get call records successful (HTTP 200)"
else
    fail "Expected 200, actual $HTTP_CODE"
fi

# ── Test 4: Get non-existent call (should return error)────────────

echo "Test 4: Get non-existent call"
RESP=$(curl -s "${BASE_URL}/calling/calls/nonexistent-call-id" \
    -H "Authorization: Bearer $TOKEN_A")
CODE=$(json_code "$RESP")
if [ "$CODE" = "404" ] || [ "$CODE" = "500" ]; then
    pass "Non-existent call returns error code $CODE"
else
    fail "Expected code 404/500, actual $CODE"
fi

# ── Test 5: Unauthenticated create meeting (should return 401)────────────

echo "Test 5: Unauthenticated create meeting"
RESP=$(curl -s -X POST "${BASE_URL}/calling/meetings" \
    -H "Content-Type: application/json" \
    -d '{"title":"Test Meeting"}')
CODE=$(json_code "$RESP")
if [ "$CODE" = "401" ]; then
    pass "Unauthenticated returns code 401"
else
    fail "Expected code 401, actual $CODE"
fi

# ── Test 6: Missing title (should return 400)────────────────────

echo "Test 6: Create meeting missing title"
RESP=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BASE_URL}/calling/meetings" \
    -H "Authorization: Bearer $TOKEN_A" \
    -H "Content-Type: application/json" \
    -d '{}')
if [ "$RESP" = "400" ]; then
    pass "Missing title returns 400"
else
    fail "Expected 400, actual $RESP"
fi

# ── Test 7: List meetings (initially empty)────────────────────

echo "Test 7: List meetings (initially empty)"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    "${BASE_URL}/calling/meetings" \
    -H "Authorization: Bearer $TOKEN_A")
if [ "$HTTP_CODE" = "200" ]; then
    pass "List meetings successful (HTTP 200)"
else
    fail "Expected 200, actual $HTTP_CODE"
fi

# ── Test 8: Answer non-existent call (should return error)────────────

echo "Test 8: Answer non-existent call"
RESP=$(curl -s -X POST "${BASE_URL}/calling/calls/fake-call-id/join" \
    -H "Authorization: Bearer $TOKEN_B")
CODE=$(json_code "$RESP")
if [ "$CODE" = "404" ] || [ "$CODE" = "500" ]; then
    pass "Answer non-existent call returns error code $CODE"
else
    fail "Expected code 404/500, actual $CODE"
fi

# ── Test 9: Get non-existent meeting (should return error)─────────

echo "Test 9: Get non-existent meeting"
RESP=$(curl -s "${BASE_URL}/calling/meetings/nonexistent-room" \
    -H "Authorization: Bearer $TOKEN_A")
CODE=$(json_code "$RESP")
if [ "$CODE" = "404" ] || [ "$CODE" = "500" ]; then
    pass "Non-existent meeting returns error code $CODE"
else
    fail "Expected code 404/500, actual $CODE"
fi

# ── Test 10: Blacklist restriction - blocked party cannot initiate call ──────────

echo "Test 10: Blacklist restriction (cannot initiate call)"
RESP=$(curl -s -X POST "${BASE_URL}/friends/blacklist" \
    -H "Authorization: Bearer $TOKEN_A" \
    -H "Content-Type: application/json" \
    -d "{\"userId\":\"${USER_B_ID}\"}")
CODE=$(json_code "$RESP")
if [ "$CODE" = "0" ]; then
    pass "User A blocked User B successfully"
else
    fail "User A failed to block User B, code=$CODE"
fi

RESP=$(curl -s -X POST "${BASE_URL}/calling/calls" \
    -H "Authorization: Bearer $TOKEN_B" \
    -H "Content-Type: application/json" \
    -d "{\"calleeId\":\"${USER_A_ID}\",\"callType\":\"audio\"}")
CODE=$(json_code "$RESP")
if [ "$CODE" = "403" ]; then
    pass "Blocked party call initiation intercepted (code 403)"
else
    fail "Expected interception to return code 403, actual $CODE"
fi

# Cleanup blacklist to avoid affecting subsequent manual debugging
RESP=$(curl -s -X DELETE "${BASE_URL}/friends/blacklist/${USER_B_ID}" \
    -H "Authorization: Bearer $TOKEN_A")
CODE=$(json_code "$RESP")
if [ "$CODE" = "0" ]; then
    pass "Cleanup blacklist successful"
else
    fail "Cleanup blacklist failed, code=$CODE"
fi

# ── Summary ──────────────────────────────────────────────

echo ""
echo "=================================================="
echo "  Test Results: ${PASS} passed, ${FAIL} failed"
echo "=================================================="

if [ $FAIL -eq 0 ]; then
    exit 0
else
    exit 1
fi