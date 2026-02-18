# 测试快速入门

本指南帮助您快速上手 AnyChat 的 API 测试。

## 目录

- [环境准备](#环境准备)
- [快速测试](#快速测试)
- [测试类型](#测试类型)
- [常用命令](#常用命令)

## 环境准备

### 1. 安装依赖工具

```bash
# 安装 jq（JSON 处理工具）
# macOS
brew install jq

# Ubuntu/Debian
sudo apt-get install jq

# 安装 grpcurl（gRPC 测试工具）
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

### 2. 启动服务

```bash
# 检查端口冲突（推荐先执行）
./scripts/check-ports.sh --check

# 启动基础设施（数据库、Redis、NATS等）
mage docker:up

# 等待服务启动完成，查看状态
mage docker:ps

# 运行数据库迁移
mage db:up

# 启动微服务（在不同的终端窗口）
mage dev:auth      # 启动 auth-service (端口 9001)
mage dev:user      # 启动 user-service (端口 9002)
mage dev:gateway   # 启动 gateway-service (端口 8080)
```

**端口说明:**
- Gateway: HTTP 8080
- Auth Service: HTTP 8001, gRPC 9001
- User Service: HTTP 8002, gRPC 9002
- 完整端口分配: 查看 `docs/development/port-allocation.md`

### 3. 验证服务运行

```bash
# 检查 Gateway 健康状态
curl http://localhost:8080/health

# 检查 Auth Service
grpcurl -plaintext localhost:9001 list

# 检查 User Service
grpcurl -plaintext localhost:9002 list
```

## 快速测试

### 一键运行所有测试

```bash
# 给脚本添加执行权限（首次运行需要）
chmod +x tests/api/test-all.sh

# 运行所有 API 测试（推荐）
./tests/api/test-all.sh

# 运行单个模块测试
./tests/api/auth/test-auth-api.sh
./tests/api/user/test-user-api.sh
./tests/api/friend/test-friend-api.sh
```

### 手动测试单个接口

**HTTP API 示例：**

```bash
# 1. 注册用户
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "phoneNumber": "13800138000",
    "password": "Test@123456",
    "verifyCode": "123456",
    "nickname": "测试用户",
    "deviceType": "iOS",
    "deviceId": "device-001"
  }' | jq

# 2. 登录（复制上面返回的 accessToken）
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "account": "13800138000",
    "password": "Test@123456",
    "deviceType": "iOS",
    "deviceId": "device-001"
  }' | jq

# 3. 获取个人资料（替换 YOUR_TOKEN）
curl -X GET http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer YOUR_TOKEN" | jq
```

**gRPC API 示例：**

```bash
# 1. 注册用户
grpcurl -plaintext -d '{
  "phone_number": "13800138000",
  "password": "Test@123456",
  "verify_code": "123456",
  "nickname": "测试用户",
  "device_type": "iOS",
  "device_id": "device-001"
}' localhost:9003 anychat.auth.AuthService/Register

# 2. 登录
grpcurl -plaintext -d '{
  "account": "13800138000",
  "password": "Test@123456",
  "device_type": "iOS",
  "device_id": "device-001"
}' localhost:9003 anychat.auth.AuthService/Login

# 3. 获取个人资料（替换 user-id）
grpcurl -plaintext -d '{
  "user_id": "YOUR_USER_ID"
}' localhost:9002 anychat.user.UserService/GetProfile
```

## 测试类型

### 1. Shell 脚本测试

**优点：**
- 快速、轻量
- 易于理解和修改
- 适合 CI/CD

**脚本列表：**
- `scripts/test-api.sh` - HTTP API 功能测试
- `scripts/test-grpc.sh` - gRPC API 功能测试
- `tests/e2e/test-e2e.sh` - 端到端场景测试

### 2. Go 集成测试

**优点：**
- 类型安全
- 更好的IDE支持
- 易于调试

**运行方式：**

```bash
# 运行集成测试
go test -v ./tests/integration/...

# 运行单个测试
go test -v ./tests/integration -run TestAuthServiceIntegration

# 带超时时间
go test -v -timeout 30s ./tests/integration/...
```

### 3. 单元测试

```bash
# 运行所有单元测试
mage test:unit

# 运行特定包的测试
go test -v -short ./internal/auth/service/...

# 生成覆盖率报告
mage test:coverage
```

## 常用命令

### 服务管理

```bash
# 启动所有基础设施
mage docker:up

# 停止所有基础设施
mage docker:down

# 查看容器状态
mage docker:ps

# 查看日志
mage docker:logs

# 启动特定服务
mage dev:auth
mage dev:user
mage dev:gateway
```

### 数据库管理

```bash
# 运行迁移
mage db:up

# 回滚迁移
mage db:down

# 创建新迁移
mage db:create add_user_table

# 连接到数据库
psql -h localhost -U anychat -d anychat
```

### 代码质量

```bash
# 格式化代码
mage fmt

# 运行 Linter
mage lint

# 生成 protobuf 代码
mage proto

# 生成 mock
mage mock
```

### 构建

```bash
# 构建所有服务
mage build:all

# 构建特定服务
mage build:auth
mage build:user
mage build:gateway
```

## 测试场景示例

### 场景1: 用户完整注册流程

```bash
# 1. 注册
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "phoneNumber": "13812345678",
    "password": "Test@123456",
    "verifyCode": "123456",
    "nickname": "张三",
    "deviceType": "iOS",
    "deviceId": "iphone-12-pro"
  }' > register_response.json

# 2. 提取 Token
export TOKEN=$(cat register_response.json | jq -r '.data.accessToken')

# 3. 获取个人资料
curl -X GET http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer $TOKEN" | jq

# 4. 更新个人资料
curl -X PUT http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "signature": "开心每一天！",
    "gender": 1,
    "region": "中国-上海"
  }' | jq
```

### 场景2: 多设备登录

```bash
# 设置变量
PHONE="13812345678"
PASSWORD="Test@123456"

# iOS 设备登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d "{
    \"account\": \"$PHONE\",
    \"password\": \"$PASSWORD\",
    \"deviceType\": \"iOS\",
    \"deviceId\": \"iphone-001\"
  }" > ios_login.json

# Android 设备登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d "{
    \"account\": \"$PHONE\",
    \"password\": \"$PASSWORD\",
    \"deviceType\": \"Android\",
    \"deviceId\": \"android-001\"
  }" > android_login.json

# 验证两个 Token 都有效
IOS_TOKEN=$(cat ios_login.json | jq -r '.data.accessToken')
ANDROID_TOKEN=$(cat android_login.json | jq -r '.data.accessToken')

curl -H "Authorization: Bearer $IOS_TOKEN" \
  http://localhost:8080/api/v1/users/me | jq '.data.userId'

curl -H "Authorization: Bearer $ANDROID_TOKEN" \
  http://localhost:8080/api/v1/users/me | jq '.data.userId'
```

## 故障排查

### 问题1: 连接被拒绝

```bash
# 检查端口冲突
./scripts/check-ports.sh --check

# 检查服务是否启动
lsof -i :8080  # Gateway
lsof -i :9001  # Auth Service
lsof -i :9002  # User Service

# 检查服务日志
mage docker:logs

# 重启服务
mage docker:down
mage docker:up
```

### 问题2: 端口已被占用

```bash
# 使用端口管理工具清理
./scripts/check-ports.sh --clean  # 停止所有微服务

# 或手动停止特定端口的进程
./scripts/check-ports.sh --kill 9001

# 查看端口使用情况
./scripts/check-ports.sh  # 交互模式，选择 "3) 显示端口使用情况"

# 完整清理（微服务 + Docker）
./scripts/check-ports.sh --full-clean
```

### 问题2: 数据库错误

```bash
# 检查数据库是否启动
docker ps | grep postgres

# 检查迁移状态
mage db:status

# 重新运行迁移
mage db:down
mage db:up
```

### 问题3: Token 无效

```bash
# 检查 JWT 密钥配置
cat configs/config.yaml | grep jwt

# 重新登录获取新 Token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{...}' | jq '.data.accessToken'
```

## 下一步

### API 规范文档

AnyChat 提供完整的 API 规范文档，支持自动生成 SDK：

- 📖 **[Gateway HTTP API](api/swagger-ui.html ':ignore')** - 基于 OpenAPI 3.0 规范的 REST API 文档
  - 交互式 Swagger UI，可直接测试 API
  - 下载 `openapi.json` 用于 SDK 生成

- 🔌 **[Gateway WebSocket API](api/asyncapi-ui.html ':ignore')** - 基于 AsyncAPI 3.0 规范的 WebSocket API 文档
  - 实时消息和通知推送
  - 下载 `asyncapi.yaml` 用于 SDK 生成

### SDK 生成

使用 OpenAPI Generator 和 AsyncAPI Generator 自动生成客户端 SDK：

```bash
# 生成 TypeScript SDK (HTTP API)
npx @openapitools/openapi-generator-cli generate \
  -i docs/api/swagger/openapi.json \
  -g typescript-axios \
  -o ./sdk/typescript-http

# 生成 Java SDK (HTTP API)
npx @openapitools/openapi-generator-cli generate \
  -i docs/api/swagger/openapi.json \
  -g java \
  -o ./sdk/java-http

# 生成 WebSocket SDK
npx @asyncapi/generator docs/api/asyncapi.yaml @asyncapi/html-template -o ./sdk/websocket-docs
```

### 更多资源

- 🧪 运行更多测试场景
- 📝 编写自定义测试用例
- 📚 查看 [开发指南](/development/getting-started.md)

## 获取帮助

- 查看项目 README
- 查看 `CLAUDE.md` 项目说明
- 提交 GitHub Issue
- 联系开发团队
