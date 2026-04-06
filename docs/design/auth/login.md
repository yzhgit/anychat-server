# 用户登录设计

## 1. 概述

用户登录功能支持手机号、邮箱、账号密码登录，登录成功后返回认证令牌。

## 2. 功能列表

- [x] 账号密码登录（支持手机号/邮箱/账号）
- [x] 设备记录管理
- [x] 多设备登录支持
- [x] 登录状态返回

## 3. 业务流程

```mermaid
sequenceDiagram
    participant Client
    participant Gateway
    participant AuthService
    participant DB

    Client->>Gateway: POST /auth/login
    Gateway->>AuthService: gRPC Login
    AuthService->>AuthService: 验证设备类型
    AuthService->>DB: 查询用户(手机号/邮箱/账号)
    DB-->>AuthService: 用户信息
    AuthService->>AuthService: 验证密码(bcrypt)
    AuthService->>DB: 更新/创建设备记录
    AuthService->>AuthService: 生成JWT Token
    AuthService->>DB: 更新/创建会话
    AuthService-->>Gateway: 返回UserID + Tokens + 用户信息
    Gateway-->>Client: 登录成功
```

## 4. API设计

### 4.1 请求

```protobuf
message LoginRequest {
    string account = 1;       // 账号(手机号/邮箱/用户名)
    string password = 2;      // 密码
    string device_id = 3;     // 设备ID
    string device_type = 4;   // 设备类型 (ios/android/web/pc)
    string client_version = 5; // 客户端版本号
    string ip_address = 6;    // 客户端IP地址
}
```

### 4.2 响应

```protobuf
message LoginResponse {
    string user_id = 1;
    string access_token = 2;
    string refresh_token = 3;
    int32 expires_in = 4;      // 7200秒(2小时)
    UserInfo user = 5;
}

message UserInfo {
    string user_id = 1;
    string phone = 2;
    string email = 3;
}
```

### 4.3 错误码

| 错误码 | 说明 |
|--------|------|
| 1 | 参数错误 |
| 10104 | 用户不存在 |
| 10105 | 密码错误 |
| 10106 | 账号已被禁用 |

## 5. 设备处理

登录时自动记录设备信息：
- 首次登录：创建设备记录
- 重复登录：更新最后登录时间

## 6. 依赖服务

- **PostgreSQL**: 用户、设备、会话持久化
- **Redis**: Token缓存（可选）
