# Scripts Guide

This directory contains environment management and service management scripts for the AnyChat project.

## 📁 Script List

### Environment Management

- **`setup-env.sh`** - Environment check and setup script
  - Checks development tools (Go, Docker, jq, curl, etc.)
  - Checks infrastructure services (PostgreSQL, Redis, NATS, MinIO)
  - Checks microservice status
  - Checks database connections and Go modules
  - Provides quick fix suggestions

- **`health-check.sh`** - Service health check
  - Checks HTTP health endpoints for all microservices
  - Checks if gRPC ports are listening
  - Checks infrastructure service ports
  - Checks Docker container status

- **`check-ports.sh`** - Port check utility
  - Checks port usage for all services
  - Identifies port conflicts

### Service Management

- **`start-services.sh`** - Start all microservices
- **`stop-services.sh`** - Stop all microservices

## 🚀 Quick Start

### 1. Environment Check

Run environment check when first using or encountering issues:

```bash
./scripts/setup-env.sh
```

This checks:
- ✓ Development tools installed
- ✓ Go modules working
- ✓ Infrastructure running
- ✓ Database connections healthy

### 2. Start Infrastructure

```bash
# Start PostgreSQL, Redis, NATS, MinIO
mage docker:up

# Check container status
mage docker:ps

# Run database migrations
mage db:up
```

### 3. Start Microservices

Use the startup script, or start individually in separate terminals:

```bash
# Start all services at once
./scripts/start-services.sh

# Or start individually as needed
mage dev:auth
mage dev:user
mage dev:friend
mage dev:group
mage dev:message
mage dev:conversation
mage dev:file
mage dev:gateway
mage dev:push
mage dev:calling
mage dev:sync
mage dev:admin
```

### 4. Health Check

```bash
./scripts/health-check.sh
```

Expected output:
```
========================================
Infrastructure Services
========================================
✓ PostgreSQL - port 5432 listening
✓ Redis - port 6379 listening
✓ NATS - port 4222 listening
✓ MinIO API - port 9000 listening
✓ MinIO Console - port 9091 listening

========================================
HTTP Services
========================================
✓ Auth Service - healthy (HTTP 200)
✓ User Service - healthy (HTTP 200)
✓ Friend Service - healthy (HTTP 200)
✓ Gateway Service - healthy (HTTP 200)

========================================
Health Check Summary
========================================
Healthy services: 13 / 13 (100%)
✓ All services healthy!
```

## 🔧 Common Issues

### Issue 1: Service Not Running

**Error**:
```
✗ Gateway Service (port 8080) not running
Some services are not running, please start all services first
```

**Solution**:
```bash
# Check which services are not running
./scripts/health-check.sh

# Start missing services
mage dev:gateway  # or other services
```

### Issue 2: Infrastructure Not Running

**Error**:
```
✗ PostgreSQL - port 5432 not listening
```

**Solution**:
```bash
# Start infrastructure
mage docker:up

# Wait for services to be ready (~10 seconds)
sleep 10

# Run database migrations
mage db:up
```

### Issue 3: Port Conflict

**Error**:
```
bind: address already in use
```

**Solution**:
```bash
# Check port usage
./scripts/check-ports.sh

# Or manually check
lsof -i :8080

# Kill the process using the port
kill -9 <PID>
```

### Issue 4: Database Connection Failed

**Error**:
```
Error: database connection failed
```

**Solution**:
```bash
# Check PostgreSQL status
docker ps | grep postgres

# Check database connection
PGPASSWORD=anychat123 psql -h localhost -U anychat -d anychat -c "SELECT 1"

# If failed, restart PostgreSQL
docker restart postgres
```

### Issue 5: jq Command Not Found

**Error**:
```
jq tool needs to be installed
```

**Solution**:
```bash
# Ubuntu/Debian
sudo apt-get install jq

# macOS
brew install jq

# CentOS/RHEL
sudo yum install jq
```

## 🎯 Best Practices

1. **Check environment before development**
   ```bash
   ./scripts/setup-env.sh
   ```

2. **Run health check regularly**
   ```bash
   ./scripts/health-check.sh
   ```

3. **Check logs when encountering issues**
   ```bash
   # View infrastructure logs
   mage docker:logs

   # View specific container logs
   docker logs postgres
   docker logs redis
   ```

4. **Cleanup and restart**
   ```bash
   # Stop all services
   mage docker:down

   # Clean build artifacts
   mage clean

   # Restart
   mage docker:up
   mage db:up
   ```

## 📚 More Resources

- [Project README](../README.md)
- [API Tests](../tests/README.md)
- [API Documentation](../docs/api/)
- [Development Guide](../docs/development/)
- [Design Documents](../docs/design/)

## 📄 License

MIT License