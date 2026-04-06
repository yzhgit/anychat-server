# WebSocket网关设计

## 1. 概述

Gateway服务负责WebSocket连接管理、消息实时推送、在线状态维护。

## 2. 功能列表

- [x] WebSocket连接建立
- [x] 连接认证
- [x] 心跳保活
- [x] 消息推送
- [x] 在线状态管理

## 3. 业务流程

### 3.1 连接建立

```mermaid
sequenceDiagram
    participant Client
    participant Gateway
    participant AuthService
    participant Redis
    participant NATS

    Client->>Gateway: WebSocket握手 /ws?token=xxx
    Gateway->>Gateway: 101 Switching Protocols
    Client->>Gateway: Auth消息 {token, device_id, platform}
    Gateway->>AuthService: gRPC ValidateToken(token)
    AuthService-->>Gateway: userId, deviceId, deviceType
    Gateway->>Redis: 记录用户在线状态
    Gateway->>NATS: 订阅用户通知主题
    Gateway->>Client: 认证成功 {user_id, server_time}
```

### 3.2 心跳保活

```mermaid
sequenceDiagram
    participant Client
    participant Gateway
    participant Redis

    Client->>Gateway: ping {timestamp}
    Gateway->>Gateway: 更新心跳时间
    Gateway->>Redis: 更新在线状态TTL
    Gateway->>Client: pong {server_time}
```

### 3.3 消息推送

```mermaid
sequenceDiagram
    participant MessageService
    participant NATS
    participant Gateway
    participant Redis
    participant Client

    MessageService->>NATS: 发布消息事件
    NATS->>Gateway: 订阅消息
    Gateway->>Gateway: 查找用户连接
    Gateway->>Redis: 查询在线状态
    Gateway->>Client: WebSocket推送
```

### 3.4 连接断开

```mermaid
sequenceDiagram
    participant Client
    participant Gateway
    participant Redis
    participant NATS

    Client->>Gateway: 关闭连接
    Gateway->>Redis: 删除在线状态
    Gateway->>NATS: 取消订阅
    Gateway->>Gateway: 清理连接资源
```

## 4. 连接管理

```go
type Client struct {
    Conn      *websocket.Conn // WebSocket连接
    UserID    string          // 用户ID
    DeviceID  string          // 设备ID
    Send      chan []byte     // 发送通道
    Heartbeat time.Time       // 最后心跳时间
}
```

## 5. 心跳机制

- 客户端每30秒发送心跳
- 服务端60秒内未收到心跳则断开连接

## 6. 通知订阅

Gateway为每个在线用户订阅：
- `notification.message.new.{user_id}`
- `notification.session.*.{user_id}`
- `notification.user.*.{user_id}`
- `notification.friend.*.{user_id}`
- `notification.group.*.{user_id}`
