# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AnyChat is a microservices-based instant messaging (IM) backend system written in Go. The system consists of 12 independent services that communicate via gRPC and HTTP, with NATS for async messaging.

## Build System: Mage

This project uses **Mage** (not Make) as its build tool. Mage uses Go code to define build tasks.

### Essential Mage Commands

```bash
# Setup
mage deps                    # Install Go dependencies
mage install                 # Install dev tools (golangci-lint, migrate, mockgen, protoc-gen-go, etc.)

# Development
mage dev:auth                # Run auth-service locally
mage dev:gateway             # Run gateway-service locally
mage dev:message             # Run message-service locally
mage dev:user                # Run user-service locally

# Building
mage build:all               # Build all 12 services to bin/
mage build:auth              # Build specific service
mage proto                   # Generate protobuf code from api/proto/**/*.proto

# Testing
mage test:all                # Run all tests with race detection and coverage
mage test:unit               # Run unit tests only (with -short flag)
mage test:coverage           # Generate HTML coverage report
mage lint                    # Run golangci-lint
mage fmt                     # Format code with gofmt

# Infrastructure
mage docker:up               # Start PostgreSQL, Redis, NATS, MinIO, Prometheus, Grafana, Jaeger
mage docker:down             # Stop all containers
mage docker:logs             # Follow logs
mage docker:ps               # Show container status

# Database
mage db:up                   # Run migrations up
mage db:down                 # Rollback migrations
mage db:create <name>        # Create new migration file

# Documentation
mage docs:generate           # Generate API documentation (Swagger)
mage docs:serve              # Start documentation server at http://localhost:3000
mage docs:build              # Build static documentation site
mage docs:validate           # Validate API documentation

# Other
mage clean                   # Remove bin/, coverage files
mage mock                    # Generate mock code
mage -l                      # List all available tasks
```

## Microservices Architecture

### The 12 Services

1. **auth-service**: Authentication and authorization (JWT tokens, ZITADEL integration)
2. **user-service**: User profile management
3. **friend-service**: Friend relationships and requests
4. **group-service**: Group chat management
5. **message-service**: Message storage and delivery
6. **session-service**: Session/conversation management
7. **file-service**: File uploads/downloads via MinIO
8. **push-service**: Offline push notifications
9. **gateway-service**: API gateway (WebSocket connections, HTTP/gRPC routing)
10. **rtc-service**: Audio/video calling via LiveKit
11. **sync-service**: Multi-device synchronization
12. **admin-service**: Admin panel backend

### Service Structure Pattern

Each service in `internal/<service-name>/` follows this structure:

```
internal/<service>/
├── model/       # Database models (GORM)
├── dto/         # Data Transfer Objects (API request/response)
├── repository/  # Data access layer (database operations)
├── service/     # Business logic layer
├── handler/     # HTTP handlers (Gin framework)
├── grpc/        # gRPC server implementation
└── client/      # gRPC client for calling other services
```

### Service Communication

- **Internal**: Services communicate via gRPC (port 9xxx)
- **External**: Clients connect via HTTP/WebSocket to gateway-service (port 8xxx)
- **Async**: Services publish/subscribe via NATS for event-driven operations
- **Config**: Each service reads from `configs/config.yaml` with environment variable overrides

## Technology Stack

| Component | Technology | Port/Access |
|-----------|-----------|-------------|
| Database | PostgreSQL 18.0 | localhost:5432 (user: anychat, pass: anychat123) |
| Cache | Redis 7.0+ | localhost:6379 |
| Message Queue | NATS with JetStream | localhost:4222 (client), :8222 (monitoring) |
| Object Storage | MinIO | localhost:9000 (API), :9091 (Console, admin/admin) |
| Auth Provider | ZITADEL | External service |
| Audio/Video | LiveKit | ws://localhost:7880 |
| API Gateway | Krakend | (planned) |
| Monitoring | Prometheus | localhost:9090 |
| Dashboards | Grafana | localhost:3000 (admin/admin) |
| Tracing | Jaeger | localhost:16686 |

## Development Workflow

### Starting New Work

1. Start infrastructure: `mage docker:up`
2. Wait for health checks to pass: `mage docker:ps`
3. Run migrations: `mage db:up`
4. Start relevant service(s): `mage dev:auth`, `mage dev:gateway`, etc.

### Adding a New Feature

1. **Database changes**: Create migration with `mage db:create <name>`, edit files in `migrations/`, then `mage db:up`
2. **Proto changes**: Edit `.proto` files in `api/proto/`, then run `mage proto` to regenerate
3. **Code changes**: Follow the layered architecture (handler → service → repository → model)
4. **API documentation**: Add Swagger comments to HTTP handlers (see API Documentation section below)
5. **Testing**: Write tests, run `mage test:unit` or `mage test:all`
6. **Code quality**: `mage fmt && mage lint` before committing
7. **Documentation**: Update `mage docs:generate` if you added/modified Gateway HTTP APIs

### Creating a New Service

Services follow a naming convention: `<name>-service`

**IMPORTANT**: When implementing a new service, you MUST complete ALL the following steps. Missing scripts, tests, or configuration updates is a common mistake.

#### Complete Checklist for New Service Implementation

##### 1. Database Layer
- [ ] Create migration files in `migrations/`
  - `000XXX_create_<name>_tables.up.sql`
  - `000XXX_create_<name>_tables.down.sql`
- [ ] Run `mage db:up` to apply migrations
- [ ] Verify tables created: `psql -U anychat -d anychat -c "\dt"`

##### 2. API Definition
- [ ] Create proto file in `api/proto/<name>/<name>.proto`
- [ ] Define gRPC service interface and message types
- [ ] Run `mage proto` to generate Go code
- [ ] Verify generated files: `api/proto/<name>/<name>.pb.go` and `<name>_grpc.pb.go`

##### 3. Service Implementation
- [ ] Create `internal/<name>/model/` - GORM models
- [ ] Create `internal/<name>/repository/` - Data access layer with transaction support
- [ ] Create `internal/<name>/dto/` - Request/response DTOs with JSON tags
- [ ] Create `internal/<name>/service/` - Business logic layer
- [ ] Create `internal/<name>/grpc/` - gRPC server implementation
- [ ] Create `cmd/<name>-service/main.go` - Service entry point

##### 4. Gateway Integration (if HTTP API needed)
- [ ] Extend `internal/gateway/client/manager.go`:
  - Add gRPC client connection
  - Add client getter method
- [ ] Create `internal/gateway/handler/<name>_handler.go`:
  - Implement HTTP handlers
  - **Add complete Swagger comments** (@Summary, @Tags, @Security, @Router, etc.)
- [ ] Update `internal/gateway/handler/routes.go`:
  - Register routes in authorized group
  - Add route group with all endpoints
- [ ] Update `cmd/gateway-service/main.go`:
  - Add service address to ClientManager initialization

##### 5. Error Codes
- [ ] Update `pkg/errors/errors.go`:
  - Add error code constants (e.g., 30xxx for friend service)
  - Add error messages to `errorMessages` map
  - Follow the pattern: auth=10xxx, user=20xxx, friend=30xxx, etc.

##### 6. Configuration
- [ ] Update `configs/config.yaml`:
  - Add service configuration under `services.<name>`
  - Define `grpc_addr` with environment variable support
  - Example: `grpc_addr: ${FRIEND_GRPC_ADDR:localhost:9003}`

##### 7. Build System
- [ ] Update `magefile.go`:
  - Verify service is in `Build.All()` services list
  - Add `Dev.<Name>()` method for local development
  - Follow the pattern of existing `Dev.Auth()`, `Dev.User()` methods

##### 8. **API Test Scripts** (DO NOT SKIP)
- [ ] Create `tests/api/<name>/test-<name>-api.sh`:
  - Complete API test suite
  - Test all major workflows
  - Include setup, test cases, and cleanup
  - Handle edge cases (null IDs, empty responses, etc.)
  - Use shared functions from `tests/api/common.sh`
- [ ] Update `tests/api/test-all.sh`:
  - Add new service test to the test suite
  - Update service health checks
- [ ] Update `tests/api/README.md`:
  - Document new test script
  - Add usage examples
  - Update test coverage matrix
- [ ] Update service management scripts:
  - `scripts/start-services.sh`: Add service startup logic
  - `scripts/stop-services.sh`: Add service cleanup
  - `scripts/check-ports.sh`: Add service ports (HTTP 8xxx and gRPC 9xxx)

##### 9. **Unit Tests** (DO NOT SKIP)
- [ ] Create unit tests:
  - `internal/<name>/repository/<name>_repository_test.go`
  - `internal/<name>/service/<name>_service_test.go`
  - Use gomock for mocking dependencies
- [ ] Verify tests run correctly:
  - `go test -short ./internal/<name>/...`
  - `mage test:unit` (runs all unit tests)

##### 10. Documentation
- [ ] Run `mage docs:generate` to update Swagger docs
- [ ] Verify Swagger UI shows all endpoints: `mage docs:serve`
- [ ] Update design docs if architectural changes were made

##### 11. Docker & Deployment (if needed)
- [ ] Add Docker configuration in `deployments/docker/<name>-service/`
- [ ] Update `docker-compose.yml` if service needs to run in container

##### 12. Verification
- [ ] `mage build:<name>` - Build succeeds
- [ ] `mage build:gateway` - Gateway compiles with new integration
- [ ] `mage fmt && mage lint` - Code quality passes
- [ ] `mage test:unit` - Unit tests pass
- [ ] Start infrastructure: `mage docker:up && mage db:up`
- [ ] `mage dev:<name>` - Service starts successfully
- [ ] `./tests/api/<name>/test-<name>-api.sh` - API tests pass
- [ ] `./tests/api/test-all.sh` - All tests pass
- [ ] Check logs for errors
- [ ] Verify health endpoints

#### Common Mistakes to Avoid

1. ❌ **Forgetting scripts directory updates**
   - Services won't be managed by `start-services.sh` and `stop-services.sh`
   - No automated testing scripts

2. ❌ **Forgetting tests directory**
   - No integration tests for the new service
   - Hard to verify functionality

3. ❌ **Missing Swagger comments**
   - Gateway HTTP APIs won't appear in documentation
   - Missing `@Security BearerAuth` for protected endpoints

4. ❌ **Not updating error codes**
   - Using wrong error code ranges
   - Missing error messages in `errorMessages` map

5. ❌ **Incomplete Gateway integration**
   - Forgetting to update `ClientManager`
   - Not registering routes properly
   - Missing authentication middleware

6. ❌ **Not testing JSON response formats**
   - Field naming issues (camelCase vs snake_case)
   - Missing validation for null/empty values
   - Not handling protobuf JSON serialization properly

7. ❌ **Wrong port allocation**
   - Using conflicting ports
   - Not following the 8xxx/9xxx convention
   - See `docs/development/port-allocation.md` for reserved ports

#### Port Allocation Guide

When creating a new service, assign ports following this pattern:
- HTTP API: `80XX` (e.g., 8003 for friend-service)
- gRPC: `90XX` (e.g., 9003 for friend-service)
- Metrics: `2112` (standard for all services)

Reserved ports can be found in `docs/development/port-allocation.md`

#### Reference Implementation

For a complete reference implementation, see the **friend-service**:
- Database: `migrations/000008_create_friend_tables.up.sql`
- Proto: `api/proto/friend/friend.proto`
- Service: `internal/friend/` (model, repository, service, grpc, dto)
- Entry: `cmd/friend-service/main.go`
- Gateway: `internal/gateway/handler/friend_handler.go`
- API Tests: `tests/api/friend/test-friend-api.sh`
- Errors: `pkg/errors/errors.go` (codes 30101-30118)

## Configuration

Configuration lives in `configs/config.yaml` with environment variable support using `${VAR:default}` syntax.

Example: `${DB_HOST:localhost}` uses `DB_HOST` env var or defaults to `localhost`.

When running in Docker, services use container names (postgres, redis, nats) as hostnames instead of localhost.

## Database Migrations

- Tool: golang-migrate
- Location: `migrations/` directory
- Connection string pattern: `postgresql://anychat:anychat123@localhost:5432/anychat?sslmode=disable`
- Files are sequential: `000001_<name>.up.sql` and `000001_<name>.down.sql`

## Testing Strategy

- Unit tests use `-short` flag to skip integration tests
- Integration tests require running infrastructure (docker:up)
- Mocks are generated with mockgen
- Coverage reports go to `coverage.out` and `coverage.html`

## Code Quality

The project uses golangci-lint with these enabled linters (see `.golangci.yml`):
- gofmt, golint, govet, errcheck, staticcheck
- unused, gosimple, ineffassign, deadcode
- goconst, gocyclo (complexity < 15), misspell

Generated protobuf files (`*.pb.go`) and test files are excluded from linting.

## Port Conventions

- **8xxx**: HTTP/REST APIs (8001 for auth, 8002 for user, 8080 for gateway, etc.)
- **9xxx**: gRPC services (9001 for auth, 9002 for user, etc.)
- **90xx**: Infrastructure management ports (9000 for MinIO API, 9091 for MinIO Console, 9090 for Prometheus)
- **2112**: Prometheus metrics endpoints (each service exposes metrics here)

**详细端口分配**: 参见 `docs/development/port-allocation.md`

## Important Notes

- The codebase is in early stages - most services have skeleton main.go files with TODOs
- Services are designed to run both locally (via `mage dev:*`) and in Docker
- All services support graceful shutdown on SIGINT/SIGTERM
- The project uses Chinese comments in some places (IM即时通讯 = instant messaging)
- Commit message format: `<type>(<scope>): <description>` (e.g., `feat(auth): add user registration`)

## API Documentation

### Documentation System

The project uses **swaggo/swag** to generate OpenAPI specifications from Go code comments, and **Docsify** with **docsify-openapi** plugin to render an interactive documentation site.

### Writing API Documentation

When adding or modifying Gateway HTTP handlers, **ALWAYS** add Swagger comments:

#### Required Elements

1. **Function description** (first line)
2. **@Summary** - One-line summary
3. **@Description** - Detailed description
4. **@Tags** - Group name (e.g., "认证", "用户")
5. **@Accept** and **@Produce** - Content types (usually `json`)
6. **@Param** - Request parameters (path, query, body, header)
7. **@Success** - Success responses with types
8. **@Failure** - Error responses with types
9. **@Security** - Add `BearerAuth` for authenticated endpoints
10. **@Router** - Route path and HTTP method

#### Example: Public Endpoint

```go
// Login 用户登录
// @Summary      用户登录
// @Description  用户通过账号密码登录，账号支持手机号或邮箱
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "登录信息"
// @Success      200      {object}  response.Response{data=AuthResponse}  "登录成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "账号或密码错误"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
    var req LoginRequest
    // ...
}
```

#### Example: Authenticated Endpoint

```go
// UpdateProfile 更新个人资料
// @Summary      更新个人资料
// @Description  更新当前登录用户的个人资料信息
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      UpdateProfileRequest  true  "资料信息"
// @Success      200      {object}  response.Response{data=UserProfile}  "更新成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /users/me [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
    var req UpdateProfileRequest
    // ...
}
```

#### Example: Path Parameter

```go
// @Param  userId  path  string  true  "用户ID"
// @Router /users/{userId} [get]
```

#### Example: Query Parameters

```go
// @Param  keyword   query  string  true   "搜索关键字"
// @Param  page      query  int     false  "页码"  default(1)
// @Param  pageSize  query  int     false  "每页数量"  default(20)
// @Router /users/search [get]
```

### Data Models

Define request/response structs with proper tags:

```go
// LoginRequest 用户登录请求
type LoginRequest struct {
    Account    string `json:"account" binding:"required" example:"13800138000"`
    Password   string `json:"password" binding:"required" example:"password123"`
    DeviceType string `json:"deviceType" binding:"required" example:"ios" enums:"ios,android,web"`
}

// AuthResponse 认证响应
type AuthResponse struct {
    UserID       string `json:"userId" example:"user-123"`
    AccessToken  string `json:"accessToken" example:"eyJhbGci..."`
    RefreshToken string `json:"refreshToken" example:"eyJhbGci..."`
    ExpiresIn    int64  `json:"expiresIn" example:"7200"`
}
```

**Struct tags**:
- `json:"fieldName"` - JSON field name (required)
- `binding:"required"` - Gin validation
- `example:"value"` - Example value in docs
- `enums:"val1,val2"` - Enum values
- `default:"value"` - Default value

### Generating Documentation

```bash
# Generate Swagger JSON from code comments
mage docs:generate

# Preview documentation locally at http://localhost:3000
mage docs:serve

# Validate generated documentation
mage docs:validate
```

### Automatic Deployment

- Documentation is **automatically** generated and deployed to GitHub Pages on push to `main` branch
- CI validates documentation on Pull Requests
- Access deployed docs at: `https://yzhgit.github.io/anychat-server/`

### Documentation Structure

```
docs/
├── index.html              # Docsify configuration
├── .nojekyll              # Disable Jekyll processing
├── README.md              # Documentation homepage
├── _sidebar.md            # Sidebar navigation
├── api/
│   ├── QUICKSTART.md      # API quick start guide
│   ├── gateway-http-api.md # Gateway API (renders OpenAPI spec)
│   └── swagger/           # Auto-generated Swagger files
│       ├── swagger.json   # OpenAPI specification
│       ├── swagger.yaml   # (alternative format)
│       └── docs.go        # Generated Go code
├── development/
│   ├── getting-started.md
│   ├── writing-api-docs.md # How to write API docs
│   └── ...
└── design/
    └── ...
```

### Best Practices

1. **Always add Swagger comments** when creating Gateway HTTP handlers
2. **Use meaningful examples** in struct tags
3. **Define request/response structs** instead of inline anonymous structs
4. **Add `@Security BearerAuth`** for authenticated endpoints
5. **Use appropriate HTTP status codes** (200, 400, 401, 403, 404, 500)
6. **Group related endpoints** with `@Tags`
7. **Run `mage docs:generate`** before committing API changes
8. **Check documentation locally** with `mage docs:serve`

### Common Mistakes

- ❌ Forgetting `@Security BearerAuth` on protected endpoints
- ❌ Using inline anonymous structs instead of named types
- ❌ Missing `example` tags on struct fields
- ❌ Incorrect router path format (must match actual route)
- ❌ Not regenerating docs after API changes

### Reference

- Full guide: `docs/development/writing-api-docs.md`
- Swag documentation: https://github.com/swaggo/swag
- OpenAPI spec: https://swagger.io/specification/

