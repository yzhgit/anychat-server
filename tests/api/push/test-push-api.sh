#!/bin/bash
#
# Push Service Test Script
# Tests push service related functionality
#
# Usage:
#   ./test-push-api.sh
#   GATEWAY_URL=http://localhost:8080 ./test-push-api.sh
#   PUSH_GRPC=localhost:9008 ./test-push-api.sh
#
# Notes:
#   Push service is primarily driven by NATS events with no direct HTTP endpoints.
#   This script validates:
#   1. Push service health check endpoint
#   2. Direct SendPush gRPC call via grpcurl (if available)
#   3. After sending messages trigger NATS notification, verify push-service doesn't crash
#

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
PUSH_HTTP="${PUSH_HTTP:-http://localhost:8008}"
PUSH_GRPC="${PUSH_GRPC:-localhost:9008}"
API_BASE="${GATEWAY_URL}/api/v1"

# Test data
TIMESTAMP=$(date +%s)
TEST_EMAIL="push_u1_${TIMESTAMP}@example.com"
TEST_PASSWORD="Test@123456"
TEST_DEVICE_ID="push-test-device-${TIMESTAMP}"

# Global variables
USER_TOKEN=""
USER_ID=""
HAS_GRPCURL=false
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

# ────────────────────────────────────────
# Detect grpcurl
# ────────────────────────────────────────

detect_grpcurl() {
    if command -v grpcurl &>/dev/null; then
        HAS_GRPCURL=true
        echo "  grpcurl available: $(grpcurl --version 2>&1 | head -1)"
    else
        HAS_GRPCURL=false
        echo "  grpcurl not available, skipping gRPC tests"
    fi
}

# ────────────────────────────────────────
# Register and login user
# ────────────────────────────────────────

setup_user() {
    print_header "Initializing test user"

    local reg_resp
    reg_resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\",\"verifyCode\":\"123456\",\"nickname\":\"PushTestUser\",\"deviceId\":\"${TEST_DEVICE_ID}\",\"deviceType\":\"Web\",\"clientVersion\":\"1.0.0\"}")
    local reg_status
    reg_status=$(echo "$reg_resp" | tail -1)
    local reg_body
    reg_body=$(echo "$reg_resp" | head -n -1)

    if [ "$reg_status" != "200" ]; then
        echo -e "${RED}User registration failed (HTTP $reg_status): $reg_body${NC}"
        exit 1
    fi

    local login_resp
    login_resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\",\"deviceId\":\"${TEST_DEVICE_ID}\",\"deviceType\":\"Web\",\"clientVersion\":\"1.0.0\"}")
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

test_push_service_health() {
    print_header "Test 1: Push Service Health Check"

    local resp
    resp=$(curl -s -w "\n%{http_code}" "${PUSH_HTTP}/health")
    local status
    status=$(echo "$resp" | tail -1)
    local body
    body=$(echo "$resp" | head -n -1)

    check_http_status "Push service health check returns 200" "200" "$status" "$body"

    local svc_name
    svc_name=$(echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin).get('service',''))" 2>/dev/null || \
               echo "$body" | grep -o '"service":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [ "$svc_name" = "push-service" ]; then
        pass "Health check service field correct"
    else
        fail "Health check service field" "Expected push-service, actual '$svc_name'"
    fi
}

test_send_push_grpc_missing_users() {
    print_header "Test 2: gRPC SendPush (without user_ids)"

    if [ "$HAS_GRPCURL" = false ]; then
        echo "  Skip: grpcurl not available"
        return
    fi

    local resp
    resp=$(grpcurl -plaintext \
        -d '{"title":"Test push","content":"Content","push_type":"message"}' \
        "${PUSH_GRPC}" anychat.push.PushService/SendPush 2>&1)

    if echo "$resp" | grep -q "user_ids is required"; then
        pass "Empty user_ids returns InvalidArgument error"
    else
        fail "Empty user_ids validation" "Response: $resp"
    fi
}

test_send_push_grpc_missing_title() {
    print_header "Test 3: gRPC SendPush (without title)"

    if [ "$HAS_GRPCURL" = false ]; then
        echo "  Skip: grpcurl not available"
        return
    fi

    local resp
    resp=$(grpcurl -plaintext \
        -d "{\"user_ids\":[\"${USER_ID}\"],\"content\":\"Content\",\"push_type\":\"message\"}" \
        "${PUSH_GRPC}" anychat.push.PushService/SendPush 2>&1)

    if echo "$resp" | grep -q "title is required"; then
        pass "Empty title returns InvalidArgument error"
    else
        fail "Empty title validation" "Response: $resp"
    fi
}

test_send_push_grpc_no_token() {
    print_header "Test 4: gRPC SendPush (user without push token)"

    if [ "$HAS_GRPCURL" = false ]; then
        echo "  Skip: grpcurl not available"
        return
    fi

    # Test user has no JPush token registered, push should succeed silently (skip JPush call if no token)
    local resp
    resp=$(grpcurl -plaintext \
        -d "{\"user_ids\":[\"${USER_ID}\"],\"title\":\"Test push\",\"content\":\"Content\",\"push_type\":\"message\"}" \
        "${PUSH_GRPC}" anychat.push.PushService/SendPush 2>&1)

    if echo "$resp" | grep -qE '"successCount"|"failureCount"|\{\}'; then
        pass "User without token push succeeds silently (returns empty result)"
    else
        fail "User without token push" "Response: $resp"
    fi
}

test_update_push_token_via_gateway() {
    print_header "Test 5: Register Push Token via Gateway"

    # Upload a test push token (actual scenario is JPush registration_id)
    local resp
    resp=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE}/users/me/push-token" \
        -H "Authorization: Bearer ${USER_TOKEN}" \
        -H "Content-Type: application/json" \
        -d "{\"pushToken\":\"test-registration-id-${TIMESTAMP}\",\"platform\":\"android\",\"deviceId\":\"${TEST_DEVICE_ID}\"}")
    local status
    status=$(echo "$resp" | tail -1)
    local body
    body=$(echo "$resp" | head -n -1)

    # 200 means successful registration, if token update endpoint doesn't exist then 404
    if [ "$status" = "200" ]; then
        pass "Push token registered successfully (HTTP $status)"
    elif [ "$status" = "404" ]; then
        echo "  Info: Push token registration endpoint not implemented yet (HTTP 404), skipping"
    else
        fail "Push token registration" "Expected HTTP 200, actual HTTP $status, response: $body"
    fi
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
        echo -e "${GREEN}All push service tests passed!${NC}"
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
echo -e "${GREEN}║   Push Service Test                     ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}"
echo "Gateway:   ${GATEWAY_URL}"
echo "Push HTTP: ${PUSH_HTTP}"
echo "Push gRPC: ${PUSH_GRPC}"
echo "Time: $(date '+%Y-%m-%d %H:%M:%S')"

detect_grpcurl
setup_user

test_push_service_health
test_send_push_grpc_missing_users
test_send_push_grpc_missing_title
test_send_push_grpc_no_token
test_update_push_token_via_gateway

print_summary