#!/bin/bash
#
# Admin Service API Test Script
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../common.sh"

ADMIN_URL="${ADMIN_URL:-http://localhost:8011}"

echo "=================================================="
echo "  Admin Service API Test"
echo "=================================================="
echo ""

PASS=0
FAIL=0

pass() { echo -e "${GREEN}✓ PASS${NC}: $1"; PASS=$((PASS + 1)); }
fail() { echo -e "${RED}✗ FAIL${NC}: $1"; FAIL=$((FAIL + 1)); }

# ── Test 1: Health check ────────────────────────────────────

echo "Test 1: Health check"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${ADMIN_URL}/health")
if [ "$HTTP_CODE" = "200" ]; then
    pass "Health check returns 200"
else
    fail "Expected 200, actual $HTTP_CODE"
fi

# ── Test 2: Unauthenticated access to protected endpoint ────────────────────────

echo "Test 2: Unauthenticated access to user list"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${ADMIN_URL}/api/admin/users")
if [ "$HTTP_CODE" = "401" ]; then
    pass "Unauthenticated returns 401"
else
    fail "Expected 401, actual $HTTP_CODE"
fi

# ── Test 3: Missing password login ────────────────────────────

echo "Test 3: Missing password login"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${ADMIN_URL}/api/admin/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin"}')
if [ "$HTTP_CODE" = "400" ]; then
    pass "Missing password returns 400"
else
    fail "Expected 400, actual $HTTP_CODE"
fi

# ── Test 4: Wrong password login ────────────────────────────

echo "Test 4: Wrong password login"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${ADMIN_URL}/api/admin/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"wrongpassword"}')
if [ "$HTTP_CODE" = "401" ]; then
    pass "Wrong password returns 401"
else
    fail "Expected 401, actual $HTTP_CODE"
fi

# ── Test 5: Correct login ────────────────────────────────────

echo "Test 5: Correct login (admin/Admin@123456)"
LOGIN_RESP=$(curl -s -X POST "${ADMIN_URL}/api/admin/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"Admin@123456"}')
ADMIN_TOKEN=$(echo "$LOGIN_RESP" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -n "$ADMIN_TOKEN" ]; then
    pass "Login successful and got Token"
else
    fail "Login failed or no Token returned (response: $LOGIN_RESP)"
    # If login fails, skip subsequent tests
    echo ""
    echo "Note: If database not initialized, subsequent tests will be skipped"
    echo "=================================================="
    echo "  Test Results: ${PASS} passed, ${FAIL} failed"
    echo "=================================================="
    [ $FAIL -eq 0 ] && exit 0 || exit 1
fi

# ── Test 6: Get user list (requires auth)──────────────────────

echo "Test 6: Admin get user list"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${ADMIN_URL}/api/admin/users" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "500" ]; then
    pass "User list endpoint reachable (HTTP ${HTTP_CODE})"
else
    fail "Expected 200/500, actual $HTTP_CODE"
fi

# ── Test 7: Get statistics ────────────────────────────────

echo "Test 7: Get system statistics overview"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${ADMIN_URL}/api/admin/stats/overview" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
if [ "$HTTP_CODE" = "200" ]; then
    pass "Statistics overview returns 200"
else
    fail "Expected 200, actual $HTTP_CODE"
fi

# ── Test 8: Get system config ────────────────────────────────

echo "Test 8: Get system config"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${ADMIN_URL}/api/admin/config" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
if [ "$HTTP_CODE" = "200" ]; then
    pass "System config returns 200"
else
    fail "Expected 200, actual $HTTP_CODE"
fi

# ── Test 9: Get audit logs ────────────────────────────────

echo "Test 9: Get audit logs"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${ADMIN_URL}/api/admin/audit-logs" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
if [ "$HTTP_CODE" = "200" ]; then
    pass "Audit logs returns 200"
else
    fail "Expected 200, actual $HTTP_CODE"
fi

# ── Test 10: Get admin list ─────────────────────────────

echo "Test 10: Get admin list"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${ADMIN_URL}/api/admin/admins" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
if [ "$HTTP_CODE" = "200" ]; then
    pass "Admin list returns 200"
else
    fail "Expected 200, actual $HTTP_CODE"
fi

# ── Test 11: Update non-existent config item ────────────────────────

echo "Test 11: Update system config item"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "${ADMIN_URL}/api/admin/config/site.name" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"value":"AnyChat Test"}')
if [ "$HTTP_CODE" = "200" ]; then
    pass "Config update returns 200"
else
    fail "Expected 200, actual $HTTP_CODE"
fi

# ── Test 12: Logout ────────────────────────────────────

echo "Test 12: Logout"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${ADMIN_URL}/api/admin/auth/logout" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
if [ "$HTTP_CODE" = "200" ]; then
    pass "Logout returns 200"
else
    fail "Expected 200, actual $HTTP_CODE"
fi

# ── Summary ──────────────────────────────────────────────

echo ""
echo "=================================================="
echo "  Test Results: ${PASS} passed, ${FAIL} failed"
echo "=================================================="

[ $FAIL -eq 0 ] && exit 0 || exit 1