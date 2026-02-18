# HTTP API 测试

## 概述

本目录包含 AnyChat 所有服务的 HTTP API 测试脚本。测试通过 Gateway Service 的 HTTP 接口进行，验证对外 API 的功能完整性和正确性。

## 目录结构

```
api/
├── README.md           # 本文件
├── common.sh           # 共享函数库（HTTP请求、打印函数等）
├── test-all.sh         # 运行所有API测试的入口脚本
├── auth/
│   └── test-auth-api.sh      # Auth Service API测试（7个用例）
├── user/
│   └── test-user-api.sh      # User Service API测试（9个用例）
├── friend/
│   └── test-friend-api.sh    # Friend Service API测试（12个用例）
├── group/
│   └── test-group-api.sh     # Group Service API测试（15个用例）
├── file/
│   └── test-file-api.sh      # File Service API测试
├── session/
│   └── test-session-api.sh   # Session Service API测试
├── sync/
│   └── test-sync-api.sh      # Sync Service API测试（7个用例）
├── push/
│   └── test-push-api.sh      # Push Service API测试
├── rtc/
│   └── test-rtc-api.sh       # RTC Service API测试（9个用例）
└── admin/
    └── test-admin-api.sh     # Admin Service API测试
```

## 测试范围

### Auth Service (7个测试用例)
- 健康检查
- 用户注册
- 用户登录
- 修改密码
- 新密码登录验证
- 刷新Token
- 登出

### User Service (9个测试用例)
- 获取个人资料
- 更新个人资料
- 验证资料已更新
- 搜索用户
- 获取用户设置
- 更新用户设置
- 验证设置已更新
- 刷新二维码
- 更新推送Token

### Friend Service (12个测试用例)
- 发送好友申请
- 获取收到的好友申请
- 获取发送的好友申请
- 接受好友申请
- 获取好友列表
- 更新好友备注
- 增量同步好友列表
- 添加到黑名单
- 获取黑名单
- 从黑名单移除
- 删除好友
- 验证好友已删除

### Group Service (15个测试用例)
- 健康检查
- 创建群组
- 获取群组信息
- 获取群成员列表
- 更新群信息
- 邀请成员（需要验证）
- 获取入群申请列表
- 处理入群申请（接受）
- 验证成员已加入
- 更新成员角色
- 更新群昵称
- 移除群成员
- 退出群组
- 获取我的群组列表
- 解散群组

### File Service
- 获取上传Token
- 完成上传
- 获取文件信息
- 获取下载URL
- 文件列表
- 删除文件

### Session Service
- 获取会话列表
- 获取单个会话
- 标记已读
- 置顶/取消置顶
- 免打扰设置
- 删除会话
- 获取未读总数

### Sync Service (7个测试用例)
- 全量同步（空账号）
- 增量同步（带 lastSyncTime）
- 未认证同步（返回401）
- 消息补齐（空会话列表）
- 未认证消息补齐（返回401）
- 消息补齐（不存在的会话）
- 消息补齐（query param 指定 limit）

### Push Service
- 未认证注册设备Token（返回401）
- 注册设备Token
- 未认证发送推送（返回401）
- 发送推送通知

### RTC Service (9个测试用例)
- 未认证发起通话（返回401）
- 缺少 calleeId 发起通话（返回400）
- 获取通话记录（初始为空）
- 获取不存在的通话（返回错误）
- 未认证创建会议室（返回401）
- 创建会议室缺少 title（返回400）
- 列举会议室
- 接听不存在的通话（返回错误）
- 获取不存在的会议室（返回错误）

### Admin Service
- 管理员登录
- 获取用户列表
- 获取系统配置
- 审计日志查询

## 运行测试

### 前置条件

1. **安装依赖工具**：
   ```bash
   # Ubuntu/Debian
   apt-get install jq curl

   # macOS
   brew install jq
   ```

2. **启动所有服务**：
   ```bash
   ./scripts/start-services.sh
   ```

3. **确认服务状态**：
   ```bash
   ./scripts/check-ports.sh
   ```

### 运行所有测试

```bash
# 在项目根目录执行
./tests/api/test-all.sh
```

### 运行单个服务测试

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

# Session Service
./tests/api/session/test-session-api.sh

# Sync Service
./tests/api/sync/test-sync-api.sh

# Push Service
./tests/api/push/test-push-api.sh

# RTC Service
./tests/api/rtc/test-rtc-api.sh

# Admin Service
ADMIN_URL=http://localhost:8011 ./tests/api/admin/test-admin-api.sh
```

### 自定义Gateway地址

```bash
# 默认: http://localhost:8080
export GATEWAY_URL="http://192.168.1.100:8080"
./tests/api/test-all.sh
```

## 测试输出示例

```
╔═══════════════════════════════════════════╗
║   AnyChat HTTP API 测试套件               ║
╚═══════════════════════════════════════════╝

测试环境: http://localhost:8080
开始时间: 2026-02-16 19:30:00

[1/3] 运行 Auth Service API 测试...
========================================
0. 健康检查
========================================
  响应: {"status":"ok"}
✓ 健康检查通过

========================================
1. 用户注册
========================================
  注册信息: 手机号=13877123456
  响应: {"code":0,"message":"success","data":{...}}
✓ 注册成功
  用户ID: abc-123-def
  AccessToken: eyJhbGciOiJIUzI1NiIs...
...

✓ Auth Service 测试通过

[2/3] 运行 User Service API 测试...
...

[3/3] 运行 Friend Service API 测试...
...

═══════════════════════════════════════════
结束时间: 2026-02-16 19:35:00

╔═══════════════════════════════════════════╗
║   所有测试通过! ✓                          ║
╚═══════════════════════════════════════════╝
```

## 测试原理

### 测试协议
- **协议**: HTTP/REST（通过Gateway Service）
- **认证**: JWT Bearer Token
- **数据格式**: JSON

### 测试流程
1. **健康检查**: 确认Gateway服务正常
2. **创建测试用户**: 注册新用户获取Token
3. **执行测试用例**: 按顺序测试各个API
4. **验证响应**: 检查返回码、数据格式、业务逻辑
5. **输出结果**: 彩色输出成功/失败

### 共享函数库 (common.sh)

提供统一的工具函数：
```bash
# HTTP请求
http_post <url> <data> [token]
http_get <url> [token]
http_put <url> <data> <token>
http_delete <url> <token>

# 响应检查
check_response <response>

# 打印输出
print_header <text>
print_success <text>
print_error <text>
print_info <text>
```

## 添加新服务测试

### Step 1: 创建目录
```bash
mkdir -p tests/api/<service-name>
```

### Step 2: 创建测试脚本
```bash
cp tests/api/auth/test-auth-api.sh tests/api/<service-name>/test-<service-name>-api.sh
```

### Step 3: 修改测试脚本
- 更新服务名称和API端点
- 添加服务特定的测试用例
- 确保使用common.sh中的共享函数

### Step 4: 更新test-all.sh
在`tests/api/test-all.sh`中添加新服务的测试调用

### Step 5: 测试验证
```bash
./tests/api/<service-name>/test-<service-name>-api.sh
./tests/api/test-all.sh
```

## 测试最佳实践

### 1. 独立性
- 每个测试脚本应该独立运行
- 使用时间戳生成唯一的测试数据
- 不依赖其他测试的状态

### 2. 幂等性
- 测试可以重复运行
- 使用随机数据避免冲突
- 不影响生产数据

### 3. 覆盖全面
- 测试正常场景和边界情况
- 验证成功和失败响应
- 检查错误消息格式

### 4. 易于调试
- 使用彩色输出区分成功/失败
- 打印详细的请求和响应
- 提供清晰的错误提示

### 5. JSON字段兼容性
- 同时支持camelCase和snake_case
- 使用`jq`的`//`运算符提供fallback
- 验证字段是否为空或null

示例：
```bash
USER_ID=$(echo "$response" | jq -r '.data.userId // .data.user_id // empty')
if [ -z "$USER_ID" ] || [ "$USER_ID" = "null" ]; then
    print_error "无法获取用户ID"
    return 1
fi
```

## 故障排查

### 测试失败常见原因

1. **服务未启动**
   ```bash
   # 检查服务状态
   ./scripts/check-ports.sh

   # 启动服务
   ./scripts/start-services.sh
   ```

2. **端口冲突**
   ```bash
   # 检查端口占用
   lsof -i :8080
   lsof -i :9001
   ```

3. **数据库未初始化**
   ```bash
   # 运行数据库迁移
   mage db:up
   ```

4. **依赖工具缺失**
   ```bash
   # 检查jq
   which jq

   # 安装jq
   sudo apt-get install jq  # Ubuntu
   brew install jq          # macOS
   ```

5. **Gateway URL错误**
   ```bash
   # 检查Gateway
   curl http://localhost:8080/health

   # 自定义URL
   export GATEWAY_URL="http://your-gateway:8080"
   ```

### 查看详细日志

测试脚本会输出详细的请求和响应，可以：
1. 检查API响应的code和message字段
2. 验证返回的数据格式
3. 确认Token是否有效

### 单独测试某个用例

可以注释掉测试脚本中的其他用例，只运行特定的测试函数。

## CI/CD 集成

### GitHub Actions 示例

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

## 参考

- [设计文档](../../docs/design/backend-design.md)
- [API快速开始](../../docs/api/QUICKSTART.md)
- [开发指南](../../docs/development/getting-started.md)
- [测试策略](../../docs/development/testing-strategy.md)
