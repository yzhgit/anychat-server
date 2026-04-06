# 推送Token管理设计

## 1. 概述

推送Token管理用于绑定用户设备推送标识，支持多设备离线推送。

## 2. 功能列表

- [x] 绑定推送Token
- [x] 更新推送Token
- [x] 多设备推送Token支持

## 3. 数据模型

```go
type UserPushToken struct {
    ID          string    // 主键
    UserID      string    // 用户ID
    Token       string    // 推送Token
    Platform    string    // 平台: ios/fcm/huawei/xg
    DeviceID    string    // 设备ID
    DeviceType  string    // 设备类型
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

## 4. 推送平台

| 平台 | 说明 |
|------|------|
| ios | Apple APNs |
| fcm | Firebase Cloud Messaging |
| huawei | 华为推送 |
| xg | 腾讯信鸽 |

## 5. 业务流程

```mermaid
sequenceDiagram
    participant Client
    participant Gateway
    participant UserService
    participant DB
    participant PushService

    Client->>Gateway: POST /user/push-token<br/>Header: Authorization: Bearer {token}<br/>Body: {push_token, platform, device_id, device_type}
    Gateway->>Gateway: 从JWT解析userId
    Gateway->>UserService: gRPC UpdatePushToken(userId, token, platform, deviceId)
    UserService->>DB: 查询是否存在
    alt 存在
        DB-->>UserService: 记录
        UserService->>DB: 更新Token
    else 不存在
        DB-->>UserService: NotFound
        UserService->>DB: 创建新记录
    end
    UserService->>PushService: 更新设备推送状态
    UserService-->>Gateway: 成功
    Gateway-->>Client: 200 OK
```

## 6. API设计

```protobuf
message UpdatePushTokenRequest {
    string user_id = 1;
    string token = 2;
    string platform = 3;
    string device_id = 4;
    string device_type = 5;
}
```

## 7. 依赖服务

- **PostgreSQL**: Token持久化
- **PushService**: 离线推送
