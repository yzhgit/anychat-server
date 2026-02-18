# AnyChat API 测试

本目录包含 AnyChat 项目的 HTTP API 测试脚本，通过 Gateway Service 的 HTTP 接口验证对外 API 的功能完整性和正确性。

## 📁 目录结构

```
tests/
├── api/                    # HTTP API 测试
│   ├── README.md          # API测试详细说明
│   ├── common.sh          # 共享函数库
│   ├── test-all.sh        # 运行所有API测试
│   ├── auth/
│   │   └── test-auth-api.sh      # Auth Service API测试
│   ├── user/
│   │   └── test-user-api.sh      # User Service API测试
│   ├── friend/
│   │   └── test-friend-api.sh    # Friend Service API测试
│   ├── group/
│   │   └── test-group-api.sh     # Group Service API测试
│   ├── file/
│   │   └── test-file-api.sh      # File Service API测试
│   ├── session/
│   │   └── test-session-api.sh   # Session Service API测试
│   ├── sync/
│   │   └── test-sync-api.sh      # Sync Service API测试
│   ├── push/
│   │   └── test-push-api.sh      # Push Service API测试
│   ├── rtc/
│   │   └── test-rtc-api.sh       # RTC Service API测试
│   └── admin/
│       └── test-admin-api.sh     # Admin Service API测试
└── README.md              # 本文件
```

## 🚀 快速开始

前置条件：启动所有服务

```bash
./scripts/start-services.sh
```

运行所有测试：

```bash
./tests/api/test-all.sh
```

运行单个服务测试：

```bash
./tests/api/auth/test-auth-api.sh
./tests/api/user/test-user-api.sh
./tests/api/friend/test-friend-api.sh
./tests/api/group/test-group-api.sh
./tests/api/file/test-file-api.sh
./tests/api/session/test-session-api.sh
./tests/api/sync/test-sync-api.sh
./tests/api/push/test-push-api.sh
./tests/api/rtc/test-rtc-api.sh
ADMIN_URL=http://localhost:8011 ./tests/api/admin/test-admin-api.sh
```

详细说明参见 [api/README.md](./api/README.md)。

## 📚 相关文档

- [API测试详细说明](./api/README.md)
- [脚本使用指南](../scripts/README.md)
- [API 文档](../docs/api/)
- [开发指南](../docs/development/)

## 📄 许可证

MIT License
