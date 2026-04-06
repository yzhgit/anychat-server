# 会话管理设计

## 1. 概述

会话管理维护用户的登录会话状态，支持 Token 刷新和多设备登录。

## 2. 功能列表

- [x] 会话创建
- [x] 会话更新
- [x] 会话删除（登出）
- [x] Token 刷新

## 3. 数据模型

```go
type UserSession struct {
    ID                    string    // 会话ID
    UserID                string    // 用户ID
    DeviceID              string    // 设备ID
    AccessToken           string    // 访问令牌
    RefreshToken          string    // 刷新令牌
    AccessTokenExpiresAt  time.Time // AccessToken过期时间
    RefreshTokenExpiresAt time.Time // RefreshToken过期时间
    CreatedAt             time.Time
    UpdatedAt             time.Time
}
```

## 4. 业务流程

### 4.1 登录创建会话

```mermaid
sequenceDiagram
    participant Client
    participant Gateway
    participant AuthService
    participant JWTManager
    participant DB

    Client->>Gateway: POST /auth/login<br/>Body: {account, password, device_id, device_type}
    Gateway->>AuthService: gRPC Login
    AuthService->>AuthService: 验证用户密码
    AuthService->>JWTManager: 生成AccessToken
    JWTManager-->>AuthService: token
    AuthService->>JWTManager: 生成RefreshToken
    JWTManager-->>AuthService: refreshToken
    AuthService->>DB: 创建会话
    DB-->>AuthService: 成功
    AuthService-->>Gateway: 返回Token
    Gateway-->>Client: 200 OK
```

### 4.2 Token 刷新

```mermaid
sequenceDiagram
    participant Client
    participant Gateway
    participant AuthService
    participant JWTManager
    participant DB

    Client->>Gateway: POST /auth/refresh<br/>Body: {refresh_token}
    Gateway->>AuthService: gRPC RefreshToken(refreshToken)
    AuthService->>JWTManager: 验证RefreshToken
    JWTManager-->>AuthService: claims
    AuthService->>DB: 查询会话
    DB-->>AuthService: 会话信息
    AuthService->>AuthService: 检查RefreshToken过期
    AuthService->>JWTManager: 生成新Token
    AuthService->>DB: 更新会话
    DB-->>AuthService: 成功
    AuthService-->>Gateway: 返回新Token
    Gateway-->>Client: 200 OK
```

### 4.3 登出删除会话

```mermaid
sequenceDiagram
    participant Client
    participant Gateway
    participant AuthService
    participant DB
    participant Redis

    Client->>Gateway: POST /auth/logout<br/>Header: Authorization: Bearer {token}
    Gateway->>Gateway: 从JWT解析userId和deviceId
    Gateway->>AuthService: gRPC Logout(userId, deviceId)
    AuthService->>DB: 删除会话
    AuthService->>Redis: 清除Token缓存
    DB-->>AuthService: 成功
    AuthService-->>Gateway: 成功
    Gateway-->>Client: 200 OK
```

## 5. 过期检查

```go
func (s *UserSession) IsRefreshTokenExpired() bool {
    return time.Now().After(s.RefreshTokenExpiresAt)
}
```

RefreshToken 过期时需要重新登录。

## 6. 依赖服务

- **PostgreSQL**: 会话持久化
- **Redis**: 会话缓存（可选）
