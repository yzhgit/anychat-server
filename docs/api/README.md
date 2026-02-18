# API 测试文档

本目录包含 AnyChat 项目的 API 文档和测试脚本。

## 目录结构

```
docs/api/
├── gateway-api.md       # Gateway HTTP API 文档
└── README.md           # 本文件

tests/api/
├── test-all.sh         # 运行所有 API 测试（推荐入口）
├── common.sh           # 共享函数库
├── README.md           # 详细测试文档
├── auth/
│   └── test-auth-api.sh    # Auth Service API 测试
├── user/
│   └── test-user-api.sh    # User Service API 测试
└── friend/
    └── test-friend-api.sh  # Friend Service API 测试
```

> **注意**: 旧的测试脚本路径已更改：
> - `scripts/test-api.sh` → `tests/api/auth/test-auth-api.sh` + `tests/api/user/test-user-api.sh`
> - `scripts/test-friend-api.sh` → `tests/api/friend/test-friend-api.sh`
> - `scripts/debug-friend-api.sh` → 已删除
> - `tests/integration/*_service_test.go` → 已删除（统一使用 Shell API 测试）
> - `tests/e2e/test-e2e.sh` → 已删除（功能合并到模块化 API 测试）

## API 文档

### Gateway HTTP API

查看 [gateway-api.md](./gateway-api.md) 获取完整的 HTTP API 文档。

**主要接口：**
- 认证：注册、登录、登出、刷新Token、修改密码
- 用户：个人资料、用户信息、搜索、设置、二维码、推送Token

## 测试脚本使用

### 前置条件

1. **安装必要工具**

```bash
# 安装 jq（JSON 处理工具）
# macOS
brew install jq

# Ubuntu/Debian
sudo apt-get install jq

# CentOS/RHEL
sudo yum install jq
```

2. **安装 grpcurl（用于 gRPC 测试）**

```bash
# 使用 Go 安装
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# 或下载二进制文件
# https://github.com/fullstorydev/grpcurl/releases
```

3. **启动服务**

```bash
# 启动基础设施
mage docker:up

# 运行数据库迁移
mage db:up

# 启动服务（在不同终端窗口）
mage dev:auth      # 启动 auth-service
mage dev:user      # 启动 user-service
mage dev:gateway   # 启动 gateway-service
```

### HTTP API 测试

**快速开始：**

```bash
# 运行所有 API 测试（推荐）
./tests/api/test-all.sh

# 运行单个模块测试
./tests/api/auth/test-auth-api.sh
./tests/api/user/test-user-api.sh
./tests/api/friend/test-friend-api.sh
```

**自定义 Gateway 地址：**

```bash
# 测试远程服务器
GATEWAY_URL=http://your-server:8080 ./tests/api/test-all.sh
```

**测试内容：**

**Auth Service API** (`tests/api/auth/test-auth-api.sh`):
- ✓ 用户注册
- ✓ 用户登录
- ✓ Token 刷新
- ✓ 修改密码
- ✓ 用户登出

**User Service API** (`tests/api/user/test-user-api.sh`):
- ✓ 获取个人资料
- ✓ 更新个人资料
- ✓ 搜索用户
- ✓ 获取/更新用户设置
- ✓ 刷新二维码
- ✓ 更新推送Token

**Friend Service API** (`tests/api/friend/test-friend-api.sh`):
- ✓ 发送好友申请
- ✓ 获取好友申请列表
- ✓ 接受/拒绝好友申请
- ✓ 获取好友列表
- ✓ 更新好友备注
- ✓ 黑名单管理
- ✓ 删除好友

## 手动测试示例

### 使用 cURL 测试 HTTP API

```bash
# 1. 用户注册
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "phoneNumber": "13800138000",
    "password": "Test@123456",
    "verifyCode": "123456",
    "nickname": "测试用户",
    "deviceType": "iOS",
    "deviceId": "device-001"
  }'

# 2. 用户登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "account": "13800138000",
    "password": "Test@123456",
    "deviceType": "iOS",
    "deviceId": "device-001"
  }'

# 3. 获取个人资料（需要替换 YOUR_ACCESS_TOKEN）
curl -X GET http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### 使用 grpcurl 测试 gRPC API

```bash
# 1. 列出所有服务
grpcurl -plaintext localhost:9003 list

# 2. 查看服务方法
grpcurl -plaintext localhost:9003 list anychat.auth.AuthService

# 3. 查看方法详情
grpcurl -plaintext localhost:9003 describe anychat.auth.AuthService.Login

# 4. 调用登录接口
grpcurl -plaintext -d '{
  "account": "13800138000",
  "password": "Test@123456",
  "device_type": "iOS",
  "device_id": "device-001"
}' localhost:9003 anychat.auth.AuthService/Login

# 5. 调用用户资料接口
grpcurl -plaintext -d '{
  "user_id": "user-id-from-login"
}' localhost:9002 anychat.user.UserService/GetProfile
```

## 常见问题

### 1. 测试脚本权限错误

```bash
# 解决方法：添加执行权限
chmod +x tests/api/test-all.sh
chmod +x tests/api/auth/test-auth-api.sh
chmod +x tests/api/user/test-user-api.sh
chmod +x tests/api/friend/test-friend-api.sh
```

### 2. jq 命令未找到

```bash
# 安装 jq
# macOS
brew install jq

# Ubuntu/Debian
sudo apt-get install jq
```

### 3. 连接被拒绝

```bash
# 检查服务是否启动
./scripts/health-check.sh

# 检查端口是否被占用
lsof -i :8080  # Gateway HTTP
lsof -i :9001  # Auth gRPC
lsof -i :9002  # User gRPC
lsof -i :9003  # Friend gRPC
```

### 4. 数据库错误

```bash
# 确保数据库已启动
mage docker:up

# 运行迁移
mage db:up

# 检查数据库连接
psql -h localhost -U anychat -d anychat
```

## 持续集成 (CI)

可以将测试脚本集成到 CI/CD 流程中：

```yaml
# .github/workflows/test.yml 示例
name: API Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Install dependencies
        run: |
          sudo apt-get install -y jq
          go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

      - name: Start services
        run: |
          mage docker:up
          mage db:up
          mage dev:auth &
          mage dev:user &
          mage dev:gateway &
          sleep 10  # 等待服务启动

      - name: Run API tests
        run: ./tests/api/test-all.sh
```

## 性能测试

对于负载测试，可以使用以下工具：

- **HTTP API**: [Apache Bench](https://httpd.apache.org/docs/2.4/programs/ab.html), [wrk](https://github.com/wg/wrk)
- **gRPC API**: [ghz](https://ghz.sh/)

示例：

```bash
# 安装 ghz
go install github.com/bojand/ghz/cmd/ghz@latest

# 压力测试登录接口
ghz --insecure \
  --proto api/proto/auth/auth.proto \
  --call anychat.auth.AuthService/Login \
  -d '{
    "account": "13800138000",
    "password": "Test@123456",
    "device_type": "iOS",
    "device_id": "device-001"
  }' \
  -c 10 \
  -n 1000 \
  localhost:9003
```

## 贡献指南

如需添加新的测试用例：

1. 在相应的测试脚本中添加新的测试函数
2. 在 `main()` 函数中调用新的测试函数
3. 更新本文档
4. 提交 Pull Request

## 反馈和问题

如果发现 API 问题或测试脚本错误，请提交 Issue 或联系开发团队。
