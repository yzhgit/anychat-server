# HTTP API Tests

## Overview

This directory contains HTTP API test scripts for all AnyChat services. Tests are conducted through the Gateway Service HTTP interfaces to verify the functionality and correctness of external APIs.

## Directory Structure

```
api/
├── README.md           # This file
├── common.sh           # Shared function library (HTTP requests, print functions, etc.)
├── test-all.sh         # Entry script to run all API tests
├── auth/
│   └── test-auth-api.sh      # Auth Service API tests (with verification code flow)
├── user/
│   └── test-user-api.sh      # User Service API tests (9 test cases)
├── friend/
│   └── test-friend-api.sh    # Friend Service API tests (12 test cases)
├── group/
│   └── test-group-api.sh     # Group Service API tests (15 test cases)
├── file/
│   └── test-file-api.sh      # File Service API tests
├── conversation/
│   └── test-conversation-api.sh   # Conversation Service API tests
├── sync/
│   └── test-sync-api.sh      # Sync Service API tests (7 test cases)
├── push/
│   └── test-push-api.sh      # Push Service API tests
├── calling/
│   └── test-calling-api.sh      # Calling Service API tests (9 test cases)
└── admin/
    └── test-admin-api.sh     # Admin Service API tests
```

## Test Coverage

### Auth Service (14 test cases)
- Health check
- Send SMS verification code
- Send email verification code
- Invalid target format validation
- Verification code sending rate limit
- Wrong verification code registration failure
- Fixed verification code registration success
- Reset password verification code sending
- User registration
- User login
- Change password
- New password login verification
- Refresh token
- Logout

> Verification code sending/consumption tests have been merged into `tests/api/auth/test-auth-api.sh`, no longer maintaining a separate verify test directory.
> To verify real email sending, configure `auth-service` with `verify.email.*` / `EMAIL_*` SMTP parameters first.

### User Service (9 test cases)
- Get personal profile
- Update personal profile
- Verify profile updated
- Search users
- Get user settings
- Update user settings
- Verify settings updated
- Refresh QR code
- Update push token

### Friend Service (12 test cases)
- Send friend request
- Get received friend requests
- Get sent friend requests
- Accept friend request
- Get friend list
- Update friend remark
- Incremental sync friend list
- Add to blacklist
- Get blacklist
- Remove from blacklist
- Delete friend
- Verify friend deleted

### Group Service (15 test cases)
- Health check
- Create group
- Get group info
- Get group member list
- Update group info
- Invite member (needs verification)
- Get join request list
- Process join request (accept)
- Verify member joined
- Update member role
- Update group nickname
- Remove group member
- Leave group
- Get my group list
- Dissolve group

### File Service
- Get upload token
- Complete upload
- Get file info
- Get download URL
- File list
- Delete file

### Conversation Service
- Get conversation list
- Get single conversation
- Mark as read
- Pin/unpin
- Mute settings
- Delete conversation
- Get unread count

### Sync Service (7 test cases)
- Full sync (empty account)
- Incremental sync (with lastSyncTime)
- Unauthenticated sync (returns 401)
- Message fill (empty conversation list)
- Unauthenticated message fill (returns 401)
- Message fill (non-existent conversation)
- Message fill (query param specifies limit)

### Push Service
- Unauthenticated device token registration (returns 401)
- Register device token
- Unauthenticated push sending (returns 401)
- Send push notification

### Calling Service (9 test cases)
- Unauthenticated call initiation (returns 401)
- Initiate call without calleeId (returns 400)
- Get call records (initially empty)
- Get non-existent call (returns error)
- Unauthenticated create meeting room (returns 400)
- Create meeting room without title (returns 400)
- List meeting rooms
- Answer non-existent call (returns error)
- Get non-existent meeting room (returns error)

### Admin Service
- Admin login
- Get user list
- Get system config
- Audit log query

## Running Tests

### Prerequisites

1. **Install dependencies**:
   ```bash
   # Ubuntu/Debian
   apt-get install jq curl

   # macOS
   brew install jq
   ```

2. **Start all services**:
   ```bash
   ./scripts/start-services.sh
   ```

3. **Verify service status**:
   ```bash
   ./scripts/check-ports.sh
   ```

### Run All Tests

```bash
# Run from project root
./tests/api/test-all.sh
```

### Run Single Service Tests

```bash
# Auth Service
./tests/api/auth/test-auth-api.sh

# User Service
./tests/api/user/test-user-api.sh

# Friend Service
./tests/api/friend/test-friend-api.sh

# Group Service
./tests/api/group/test-group-api.sh

# File Service
./tests/api/file/test-file-api.sh

# Conversation Service
./tests/api/conversation/test-conversation-api.sh

# Sync Service
./tests/api/sync/test-sync-api.sh

# Push Service
./tests/api/push/test-push-api.sh

# Calling Service
./tests/api/calling/test-calling-api.sh

# Admin Service
ADMIN_URL=http://localhost:8011 ./tests/api/admin/test-admin-api.sh
```

### Custom Gateway Address

```bash
# Default: http://localhost:8080
export GATEWAY_URL="http://192.168.1.100:8080"
./tests/api/test-all.sh
```

## Test Output Example

```
╔═══════════════════════════════════════════╗
║   AnyChat HTTP API Test Suite              ║
╚═══════════════════════════════════════════╝

Test environment: http://localhost:8080
Start time: 2026-02-16 19:30:00

[1/3] Running Auth Service API tests...
===========================================
0. Health Check
===========================================
  Response: {"status":"ok"}
✓ Health check passed

===========================================
1. User Registration
===========================================
  Registration info: phone=13877123456
  Response: {"code":0,"message":"success","data":{...}}
✓ Registration successful
  User ID: abc-123-def
  AccessToken: eyJhbGciOiJIUzI1NiIs...
...

✓ Auth Service tests passed

[2/3] Running User Service API tests...
...

[3/3] Running Friend Service API tests...
...

══════════════════════════════════════════
End time: 2026-02-16 19:35:00

╔═══════════════════════════════════════════╗
║   All tests passed! ✓                     ║
╚═══════════════════════════════════════════╝
```

## Test Principles

### Test Protocol
- **Protocol**: HTTP/REST (via Gateway Service)
- **Authentication**: JWT Bearer Token
- **Data Format**: JSON

### Test Flow
1. **Health check**: Verify Gateway service is healthy
2. **Create test user**: Register new user to get token
3. **Execute test cases**: Test each API in order
4. **Verify response**: Check return code, data format, business logic
5. **Output results**: Color-coded output for success/failure

### Shared Function Library (common.sh)

Provides unified utility functions:
```bash
# HTTP requests
http_post <url> <data> [token]
http_get <url> [token]
http_put <url> <data> <token>
http_delete <url> [token]

# Response checking
check_response <response>

# Print output
print_header <text>
print_success <text>
print_error <text>
print_info <text>
```

## Adding New Service Tests

### Step 1: Create directory
```bash
mkdir -p tests/api/<service-name>
```

### Step 2: Create test script
```bash
cp tests/api/auth/test-auth-api.sh tests/api/<service-name>/test-<service-name>-api.sh
```

### Step 3: Modify test script
- Update service name and API endpoints
- Add service-specific test cases
- Ensure using shared functions from common.sh

### Step 4: Update test-all.sh
Add the new service test call in `tests/api/test-all.sh`

### Step 5: Test verification
```bash
./tests/api/<service-name>/test-<service-name>-api.sh
./tests/api/test-all.sh
```

## Test Best Practices

### 1. Independence
- Each test script should run independently
- Use timestamps to generate unique test data
- Do not depend on other test states

### 2. Idempotency
- Tests can be run repeatedly
- Use random data to avoid conflicts
- Do not affect production data

### 3. Comprehensive Coverage
- Test normal scenarios and edge cases
- Verify success and failure responses
- Check error message format

### 4. Easy Debugging
- Use color output to distinguish success/failure
- Print detailed request and response
- Provide clear error messages

### 5. JSON Field Compatibility
- Support both camelCase and snake_case
- Use `jq`'s `//` operator for fallback
- Check if fields are empty or null

Example:
```bash
USER_ID=$(echo "$response" | jq -r '.data.userId // .data.user_id // empty')
if [ -z "$USER_ID" ] || [ "$USER_ID" = "null" ]; then
    print_error "Unable to get user ID"
    return 1
fi
```

## Troubleshooting

### Common Test Failure Reasons

1. **Service not started**
   ```bash
   # Check service status
   ./scripts/check-ports.sh

   # Start services
   ./scripts/start-services.sh
   ```

2. **Port conflict**
   ```bash
   # Check port usage
   lsof -i :8080
   lsof -i :9001
   ```

3. **Database not initialized**
   ```bash
   # Run database migrations
   mage db:up
   ```

4. **Missing dependency tools**
   ```bash
   # Check jq
   which jq

   # Install jq
   sudo apt-get install jq  # Ubuntu
   brew install jq          # macOS
   ```

5. **Wrong Gateway URL**
   ```bash
   # Check Gateway
   curl http://localhost:8080/health

   # Custom URL
   export GATEWAY_URL="http://your-gateway:8080"
   ```

### View Detailed Logs

Test scripts output detailed requests and responses:
1. Check API response code and message fields
2. Verify returned data format
3. Confirm token is valid

### Test Specific Cases Individually

You can comment out other test cases in the test script to run only specific test functions.

## CI/CD Integration

### GitHub Actions Example

```yaml
name: API Tests

on: [push, pull_request]

jobs:
  api-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y jq curl

      - name: Start services
        run: |
          mage docker:up
          mage db:up
          ./scripts/start-services.sh

      - name: Run API tests
        run: ./tests/api/test-all.sh
```

## References

- [Design Documents](../../docs/design/backend-design.md)
- [API Quick Start](../../docs/api/QUICKSTART.md)
- [Getting Started Guide](../../docs/development/getting-started.md)
- [Testing Strategy](../../docs/development/testing-strategy.md)