# AnyChat - Instant Messaging Backend System

A microservice-based IM system built with Go.

## Features

- 🚀 Private & Group Chat
- 📞 Audio/Video Calls
- 📁 File Transfer
- ✅ Message Read Receipts
- 🔄 Multi-device Sync
- 📱 Offline Push Notifications

## Tech Stack

- **Language**: Go 1.24.9+
- **Database**: PostgreSQL 18+
- **Cache**: Redis 7.0+
- **Message Queue**: NATS
- **Object Storage**: MinIO
- **Audio/Video**: LiveKit
- **Monitoring**: Prometheus + Grafana
- **Tracing**: Jaeger
- **Build Tool**: Mage

## Quick Start

### Requirements

- Go 1.24.9+
- Docker 28.1.1+ & Docker Compose 2.35.1+
- protoc 3.12.4+ (for gRPC code generation, requires `--experimental_allow_proto3_optional`)
- Mage (build tool)

### Install Mage

```bash
go install github.com/magefile/mage@latest
```

### Local Development

```bash
# 1. Clone the repository
git clone https://github.com/yzhgit/anychat-server
cd server

# 2. Install dependencies
mage deps

# 3. Install development tools (optional)
mage install

# 4. Start infrastructure services
mage docker:up

# 5. Run database migrations
mage db:up

# 6. Start services
mage dev:auth
mage dev:gateway
```

## Project Structure

```
anychat_server/
├── api/                    # API definitions
│   └── proto/             # gRPC definitions
├── cmd/                    # Application entry points
├── internal/               # Private code
├── pkg/                    # Shared libraries
├── deployments/            # Deployment configurations
├── configs/                # Configuration files
├── migrations/             # Database migrations
├── docs/                   # Documentation
│   └── api/swagger/       # OpenAPI specifications (auto-generated)
├── tests/                  # Tests
└── magefile.go            # Mage build scripts
```

## Build

```bash
# List all available commands
mage -l

# Build all services
mage build:all

# Build specific service
mage build:auth
mage build:gateway

# Build Docker images
mage docker:build
```

## Test

```bash
# Run all tests
mage test:all

# Run unit tests
mage test:unit

# Generate coverage report
mage test:coverage

# Code linting
mage lint

# Code formatting
mage fmt
```

## Common Mage Commands

### Build
- `mage build:all` - Build all services
- `mage build:auth` - Build auth service
- `mage build:user` - Build user service
- `mage build:gateway` - Build gateway service
- `mage build:message` - Build message service

### Development
- `mage dev:auth` - Run auth service
- `mage dev:gateway` - Run gateway service
- `mage dev:message` - Run message service
- `mage proto` - Generate protobuf code

### Docker
- `mage docker:up` - Start all containers
- `mage docker:down` - Stop all containers
- `mage docker:build` - Build Docker images
- `mage docker:logs` - View logs
- `mage docker:ps` - View container status

### Database
- `mage db:up` - Run database migrations
- `mage db:down` - Rollback database migrations
- `mage db:create <name>` - Create new migration file

### Documentation
- `mage docs:generate` - Generate API documentation
- `mage docs:serve` - Start documentation server (http://localhost:3000)
- `mage docs:build` - Build documentation site
- `mage docs:validate` - Validate API documentation

### Other
- `mage deps` - Install dependencies
- `mage install` - Install development tools
- `mage clean` - Clean build artifacts
- `mage mock` - Generate mock code

## Documentation

### Online Documentation

- **Full Documentation Site**: [GitHub Pages](https://yzhgit.github.io/anychat-server/) (auto-deployed)
- **Local Preview**: Run `mage docs:serve` and visit http://localhost:3000

### Documentation Content

- [Getting Started](docs/development/getting-started.md) - Beginner guide
- [API Documentation](docs/api/gateway-http-api.md) - Interactive HTTP API documentation
- [System Design](docs/design/backend-design.md) - Architecture design document
- [Writing API Documentation](docs/development/writing-api-docs.md) - How to write API documentation

### Generate and Deploy Documentation

#### Local Generation

```bash
# Generate API documentation
mage docs:serve

# Preview documentation site locally
mage docs:build
```

#### Auto Deployment

- **Trigger**: Push to main branch or create Pull Request
- **Deployment Target**: GitHub Pages
- **Documentation URL**: https://yzhgit.github.io/anychat-server/

Documentation is automatically updated when:
1. Gateway service code changes
2. Documentation files change
3. CI configuration changes

#### Writing API Documentation

Add Swagger annotations for Gateway HTTP endpoints:

```go
// Login user login
// @Summary      User login
// @Description  User login with username and password
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "Login information"
// @Success      200      {object}  response.Response{data=AuthResponse}  "Login successful"
// @Failure      400      {object}  response.Response  "Invalid parameters"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
    // ...
}
```

See [Writing API Documentation Guide](docs/development/writing-api-docs.md) for details.

### Other

Pull Requests and Issues are welcome.

## License

MIT License - See [LICENSE](LICENSE) file
