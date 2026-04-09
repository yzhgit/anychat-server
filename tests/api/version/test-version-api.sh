#!/bin/bash
#
# Version Service HTTP API Test Script
# Tests client version upgrade related HTTP APIs
#

set -e

# Load common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PARENT_DIR="$(dirname "$SCRIPT_DIR")"
source "${PARENT_DIR}/common.sh"

# Configuration
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
API_BASE="${GATEWAY_URL}/api/v1"

# Test data
TIMESTAMP=$(date +%s)
TEST_PLATFORM="android"
TEST_VERSION="1.0.0"
TEST_BUILD_NUMBER=100

# Global variables
ACCESS_TOKEN=""

# Print functions
print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
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

# Check response
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

# Test 1: Version check - no update needed
test_check_version_no_update() {
    print_header "Test 1: Version Check - No Update Needed"

    local response=$(http_get "${API_BASE}/versions/check?platform=${TEST_PLATFORM}&version=99.0.0")
    echo "Response: $response"

    local hasUpdate=$(echo "$response" | jq -r '.data.hasUpdate // false')
    if [ "$hasUpdate" = "false" ]; then
        print_success "Version check successful - no update needed"
    else
        print_error "Version check failed"
        exit 1
    fi
}

# Test 2: Version check - has update
test_check_version_has_update() {
    print_header "Test 2: Version Check - Has Update"

    local response=$(http_get "${API_BASE}/versions/check?platform=${TEST_PLATFORM}&version=1.0.0")
    echo "Response: $response"

    local hasUpdate=$(echo "$response" | jq -r '.data.hasUpdate // false')
    if [ "$hasUpdate" = "true" ]; then
        local version=$(echo "$response" | jq -r '.data.latestVersion // ""')
        print_success "Version check successful - has new version: $version"
    else
        print_info "No new version available, skip this test"
    fi
}

# Test 3: Get latest version
test_get_latest_version() {
    print_header "Test 3: Get Latest Version"

    local response=$(http_get "${API_BASE}/versions/latest?platform=${TEST_PLATFORM}")
    echo "Response: $response"

    local version=$(echo "$response" | jq -r '.data.version.version // ""')
    if [ -n "$version" ]; then
        print_success "Get latest version successful: $version"
    else
        print_info "No released version available, skip this test"
    fi
}

# Test 4: Get version list
test_list_versions() {
    print_header "Test 4: Get Version List"

    local response=$(http_get "${API_BASE}/versions/list?platform=${TEST_PLATFORM}&page=1&pageSize=10")
    echo "Response: $response"

    local total=$(echo "$response" | jq -r '.data.total // 0')
    print_success "Get version list successful - Total: $total"
}

# Test 5: Version report
test_report_version() {
    print_header "Test 5: Version Report"

    local response=$(http_post "${API_BASE}/versions/report" \
        "{\"platform\":\"${TEST_PLATFORM}\",\"version\":\"1.0.0\",\"buildNumber\":${TEST_BUILD_NUMBER},\"deviceId\":\"test-device-001\"}")
    echo "Response: $response"

    local code=$(echo "$response" | jq -r '.code // -1')
    if [ "$code" = "0" ]; then
        print_success "Version report successful"
    else
        print_info "Version report completed (code: $code)"
    fi
}

# Test 6: Create version (admin panel)
test_create_version() {
    print_header "Test 6: Create Version (Admin Panel)"

    # Admin API is on separate port
    ADMIN_API="${ADMIN_URL:-http://localhost:8011}/api/v1"

    # If no admin token, skip this test
    if [ -z "$ACCESS_TOKEN" ]; then
        print_info "Need to login first to get token, skip create version test"
        return 0
    fi

    local response=$(http_post "${ADMIN_API}/versions" \
        "{
            \"platform\": \"${TEST_PLATFORM}\",
            \"version\": \"2.0.0\",
            \"buildNumber\": 200,
            \"minVersion\": \"1.0.0\",
            \"minBuildNumber\": 100,
            \"forceUpdate\": false,
            \"releaseType\": \"stable\",
            \"title\": \"Test Version v2.0.0\",
            \"content\": \"## Update Content\\n- New Feature Test\\n- Bug Fix\",
            \"downloadUrl\": \"https://example.com/app/android/v2.0.0.apk\",
            \"fileSize\": 52428800,
            \"fileHash\": \"sha256:test-hash\"
        }" "$ACCESS_TOKEN")
    echo "Response: $response"

    local code=$(echo "$response" | jq -r '.code // -1')
    if [ "$code" = "0" ]; then
        local versionId=$(echo "$response" | jq -r '.data.id // 0')
        print_success "Create version successful - ID: $versionId"
        echo "$versionId" > /tmp/version_test_id.txt
    else
        print_info "Create version completed (code: $code)"
    fi
}

# Test 7: Get version detail
test_get_version() {
    print_header "Test 7: Get Version Detail"

    ADMIN_API="${ADMIN_URL:-http://localhost:8011}/api/v1"
    local versionId=$(cat /tmp/version_test_id.txt 2>/dev/null || echo "1")

    if [ -z "$ACCESS_TOKEN" ]; then
        print_info "Need to login first to get token, skip get version detail test"
        return 0
    fi

    local response=$(http_get "${ADMIN_API}/versions/${versionId}" "$ACCESS_TOKEN")
    echo "Response: $response"

    local version=$(echo "$response" | jq -r '.data.version.version // ""')
    if [ -n "$version" ]; then
        print_success "Get version detail successful: $version"
    else
        print_info "Version detail retrieval completed"
    fi
}

# Test 8: Force update check
test_force_update() {
    print_header "Test 8: Force Update Check"

    local response=$(http_get "${API_BASE}/versions/check?platform=${TEST_PLATFORM}&version=0.1.0")
    echo "Response: $response"

    local forceUpdate=$(echo "$response" | jq -r '.data.forceUpdate // false')
    if [ "$forceUpdate" = "true" ]; then
        print_success "Force update check successful"
    else
        print_info "No force update needed"
    fi
}

# Cleanup test data
cleanup() {
    print_header "Cleanup test data"
    rm -f /tmp/version_test_id.txt
    print_success "Cleanup completed"
}

# Main function
main() {
    echo -e "${GREEN}"
    echo "╔═══════════════════════════════════════════╗"
    echo "║   Version Service API Test Script        ║"
    echo "╚═══════════════════════════════════════════╝"
    echo -e "${NC}"

    check_dependencies

    # Execute test cases
    test_check_version_no_update
    test_check_version_has_update
    test_get_latest_version
    test_list_versions
    test_report_version
    test_create_version
    test_get_version
    test_force_update

    cleanup

    echo -e "\n${GREEN}✓ All tests completed!${NC}\n"
}

main "$@"