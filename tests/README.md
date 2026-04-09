# AnyChat API Tests

This directory contains HTTP API test scripts for the AnyChat project, which verify the functionality and correctness of external APIs through the Gateway Service HTTP interfaces.

## 📁 Directory Structure

```
tests/
├── api/                    # HTTP API tests
│   ├── README.md          # API test detailed documentation
│   ├── common.sh          # Shared function library
│   ├── test-all.sh        # Run all API tests
│   ├── auth/
│   │   └── test-auth-api.sh      # Auth Service API tests
│   ├── user/
│   │   └── test-user-api.sh      # User Service API tests
│   ├── friend/
│   │   └── test-friend-api.sh    # Friend Service API tests
│   ├── group/
│   │   └── test-group-api.sh     # Group Service API tests
│   ├── file/
│   │   └── test-file-api.sh      # File Service API tests
│   ├── conversation/
│   │   └── test-conversation-api.sh   # Conversation Service API tests
│   ├── sync/
│   │   └── test-sync-api.sh      # Sync Service API tests
│   ├── push/
│   │   └── test-push-api.sh      # Push Service API tests
│   ├── calling/
│   │   └── test-calling-api.sh      # Calling Service API tests
│   └── admin/
│       └── test-admin-api.sh     # Admin Service API tests
└── README.md              # This file
```

## 🚀 Quick Start

Prerequisite: Start all services

```bash
./scripts/start-services.sh
```

Run all tests:

```bash
./tests/api/test-all.sh
```

Run single service tests:

```bash
./tests/api/auth/test-auth-api.sh
./tests/api/user/test-user-api.sh
./tests/api/friend/test-friend-api.sh
./tests/api/group/test-group-api.sh
./tests/api/file/test-file-api.sh
./tests/api/conversation/test-conversation-api.sh
./tests/api/sync/test-sync-api.sh
./tests/api/push/test-push-api.sh
./tests/api/calling/test-calling-api.sh
ADMIN_URL=http://localhost:8011 ./tests/api/admin/test-admin-api.sh
```

For detailed documentation, see [api/README.md](./api/README.md).

## 📚 Related Documentation

- [API Test Detailed Documentation](./api/README.md)
- [Script Usage Guide](../scripts/README.md)
- [API Documentation](../docs/api/)
- [Development Guide](../docs/development/)

## 📄 License

MIT License