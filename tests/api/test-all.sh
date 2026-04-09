#!/bin/bash
#
# Entry script to run all HTTP API tests
#

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Color output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}╔═══════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   AnyChat HTTP API Test Suite             ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════╝${NC}"
echo ""
echo "Test environment: ${GATEWAY_URL:-http://localhost:8080}"
echo "Start time: $(date '+%Y-%m-%d %H:%M:%S')"
echo ""

FAILED=0

# Run Auth Service tests
echo -e "${YELLOW}[1/10] Running Auth Service API tests...${NC}"
if "${SCRIPT_DIR}/auth/test-auth-api.sh"; then
    echo -e "${GREEN}✓ Auth Service tests passed${NC}"
else
    echo -e "${RED}✗ Auth Service tests failed${NC}"
    ((FAILED++))
fi
echo ""

# Run User Service tests
echo -e "${YELLOW}[2/10] Running User Service API tests...${NC}"
if "${SCRIPT_DIR}/user/test-user-api.sh"; then
    echo -e "${GREEN}✓ User Service tests passed${NC}"
else
    echo -e "${RED}✗ User Service tests failed${NC}"
    ((FAILED++))
fi
echo ""

# Run Friend Service tests
echo -e "${YELLOW}[3/9] Running Friend Service API tests...${NC}"
if "${SCRIPT_DIR}/friend/test-friend-api.sh"; then
    echo -e "${GREEN}✓ Friend Service tests passed${NC}"
else
    echo -e "${RED}✗ Friend Service tests failed${NC}"
    ((FAILED++))
fi
echo ""

# Run Group Service tests
echo -e "${YELLOW}[4/9] Running Group Service API tests...${NC}"
if "${SCRIPT_DIR}/group/test-group-api.sh"; then
    echo -e "${GREEN}✓ Group Service tests passed${NC}"
else
    echo -e "${RED}✗ Group Service tests failed${NC}"
    ((FAILED++))
fi
echo ""

# Run File Service tests
echo -e "${YELLOW}[5/10] Running File Service API tests...${NC}"
if "${SCRIPT_DIR}/file/test-file-api.sh"; then
    echo -e "${GREEN}✓ File Service tests passed${NC}"
else
    echo -e "${RED}✗ File Service tests failed${NC}"
    ((FAILED++))
fi
echo ""

# Run Conversation Service tests
echo -e "${YELLOW}[6/10] Running Conversation Service API tests...${NC}"
if "${SCRIPT_DIR}/conversation/test-conversation-api.sh"; then
    echo -e "${GREEN}✓ Conversation Service tests passed${NC}"
else
    echo -e "${RED}✗ Conversation Service tests failed${NC}"
    ((FAILED++))
fi
echo ""

# Run Sync Service tests
echo -e "${YELLOW}[7/10] Running Sync Service API tests...${NC}"
if "${SCRIPT_DIR}/sync/test-sync-api.sh"; then
    echo -e "${GREEN}✓ Sync Service tests passed${NC}"
else
    echo -e "${RED}✗ Sync Service tests failed${NC}"
    ((FAILED++))
fi
echo ""

# # Run Push Service tests
# echo -e "${YELLOW}[8/10] Running Push Service API tests...${NC}"
# if "${SCRIPT_DIR}/push/test-push-api.sh"; then
#     echo -e "${GREEN}✓ Push Service tests passed${NC}"
# else
#     echo -e "${RED}✗ Push Service tests failed${NC}"
#     ((FAILED++))
# fi
# echo ""

# Run Calling Service tests
echo -e "${YELLOW}[8/10] Running Calling Service API tests...${NC}"
if "${SCRIPT_DIR}/calling/test-calling-api.sh"; then
    echo -e "${GREEN}✓ Calling Service tests passed${NC}"
else
    echo -e "${RED}✗ Calling Service tests failed${NC}"
    ((FAILED++))
fi
echo ""

# Run Admin Service tests
echo -e "${YELLOW}[9/10] Running Admin Service API tests...${NC}"
if ADMIN_URL="${ADMIN_URL:-http://localhost:8011}" "${SCRIPT_DIR}/admin/test-admin-api.sh"; then
    echo -e "${GREEN}✓ Admin Service tests passed${NC}"
else
    echo -e "${RED}✗ Admin Service tests failed${NC}"
    ((FAILED++))
fi
echo ""

# Run Version Service tests
echo -e "${YELLOW}[10/10] Running Version Service API tests...${NC}"
if "${SCRIPT_DIR}/version/test-version-api.sh"; then
    echo -e "${GREEN}✓ Version Service tests passed${NC}"
else
    echo -e "${RED}✗ Version Service tests failed${NC}"
    ((FAILED++))
fi
echo ""

# Output summary
echo -e "${YELLOW}═══════════════════════════════════════════${NC}"
echo "End time: $(date '+%Y-%m-%d %H:%M:%S')"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}╔═══════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║   All tests passed! ✓                    ║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════╝${NC}"
    exit 0
else
    echo -e "${RED}╔═══════════════════════════════════════════╗${NC}"
    echo -e "${RED}║   Failed tests: ${FAILED}                          ║${NC}"
    echo -e "${RED}╚═══════════════════════════════════════════╝${NC}"
    exit 1
fi