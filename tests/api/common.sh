#!/bin/bash
#
# Shared function library - HTTP API testing utilities
# Common functions for all API test scripts
#

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Check response is a failure (code != 0)
check_response_fail() {
    local response=$1
    local code
    code=$(json_code "$response")

    if [ "$code" != "0" ]; then
        return 0
    else
        print_error "Expected request to fail, but returned success"
        return 1
    fi
}

# Check if failure response code matches expected
check_fail_code() {
    local response=$1
    local expected_code=$2
    local code
    code=$(json_code "$response")

    if [ "$code" = "$expected_code" ]; then
        return 0
    else
        local message
        if command -v jq &> /dev/null; then
            message=$(echo "$response" | jq -r '.message // "Unknown error"')
        else
            message="Unknown error"
        fi
        print_error "Failure code does not match: got=${code}, expected=${expected_code}, message=${message}"
        return 1
    fi
}

# Parse standard response code (prefer jq, fallback to grep)
json_code() {
    local response=$1
    if command -v jq &> /dev/null; then
        echo "$response" | jq -r '.code // -1'
    else
        echo "$response" | grep -o '"code":[0-9]*' | head -n1 | cut -d: -f2
    fi
}

# Extract userId from response
extract_user_id() {
    local response=$1
    if command -v jq &> /dev/null; then
        echo "$response" | jq -r '.data.userId // .data.user_id // empty'
    else
        echo "$response" | grep -o '"userId":"[^"]*"' | head -n1 | cut -d'"' -f4
    fi
}

# Extract accessToken from response
extract_access_token() {
    local response=$1
    if command -v jq &> /dev/null; then
        echo "$response" | jq -r '.data.accessToken // .data.access_token // empty'
    else
        echo "$response" | grep -o '"accessToken":"[^"]*"' | head -n1 | cut -d'"' -f4
    fi
}

# Register test user, returns raw response
register_test_user() {
    local api_base=$1
    local email=$2
    local password=$3
    local nickname=$4
    local device_id=$5
    local device_type=${6:-iOS}
    local client_version=${7:-1.0.0}
    local verify_code=${8:-123456}

    local data
    data=$(cat <<EOF
{
    "email": "${email}",
    "password": "${password}",
    "verifyCode": "${verify_code}",
    "nickname": "${nickname}",
    "deviceType": "${device_type}",
    "deviceId": "${device_id}",
    "clientVersion": "${client_version}"
}
EOF
)
    http_post "${api_base}/auth/register" "$data"
}

# Login test user, returns raw response
login_test_user() {
    local api_base=$1
    local account=$2
    local password=$3
    local device_id=$4
    local device_type=${5:-Web}
    local client_version=${6:-1.0.0}

    local data
    data=$(cat <<EOF
{
    "account": "${account}",
    "password": "${password}",
    "deviceId": "${device_id}",
    "deviceType": "${device_type}",
    "clientVersion": "${client_version}"
}
EOF
)
    http_post "${api_base}/auth/login" "$data"
}

# Register and login test user (registration failure doesn't affect login), outputs accessToken
register_and_login_test_user() {
    local api_base=$1
    local email=$2
    local password=$3
    local nickname=$4
    local device_id=$5
    local device_type=${6:-Web}
    local client_version=${7:-1.0.0}
    local verify_code=${8:-123456}

    register_test_user "$api_base" "$email" "$password" "$nickname" "$device_id" "$device_type" "$client_version" "$verify_code" >/dev/null
    local login_resp
    login_resp=$(login_test_user "$api_base" "$email" "$password" "$device_id" "$device_type" "$client_version")
    extract_access_token "$login_resp"
}

# Get current user ID by token
get_user_id_by_token() {
    local api_base=$1
    local token=$2
    local resp
    resp=$(http_get "${api_base}/users/me" "$token")
    extract_user_id "$resp"
}

# Check dependency tools
check_dependencies() {
    if ! command -v jq &> /dev/null; then
        print_error "jq tool needs to be installed: apt-get install jq or brew install jq"
        exit 1
    fi

    if ! command -v curl &> /dev/null; then
        print_error "curl tool needs to be installed"
        exit 1
    fi
}