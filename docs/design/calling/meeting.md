# 视频会议设计

## 1. 概述

视频会议功能支持创建会议室、多人视频会议、屏幕共享等。

## 2. 功能列表

- [x] 创建会议室
- [x] 加入会议
- [x] 结束会议
- [x] 会议列表查询

## 3. 业务流程

### 3.1 创建会议

```mermaid
sequenceDiagram
    participant Client
    participant Gateway
    participant CallingService
    participant LiveKit
    participant NATS

    Client->>Gateway: POST /meeting/create<br/>Header: Authorization: Bearer {token}<br/>Body: {title, password, max_participants}
    Gateway->>Gateway: 从JWT解析userId
    Gateway->>CallingService: gRPC CreateMeeting(userId, title, password, maxParticipants)
    CallingService->>CallingService: 生成RoomID
    CallingService->>LiveKit: 创建Room
    LiveKit-->>CallingService: Room创建成功
    CallingService->>CallingService: 生成Token
    CallingService->>NATS: 发布会议创建事件
    CallingService-->>Gateway: 返回roomId + roomName + token
    Gateway-->>Client: 200 OK
```

### 3.2 加入会议

```mermaid
sequenceDiagram
    participant Client
    participant Gateway
    participant CallingService
    participant LiveKit

    Client->>Gateway: POST /meeting/join<br/>Header: Authorization: Bearer {token}<br/>Body: {room_id, password}
    Gateway->>Gateway: 从JWT解析userId
    Gateway->>CallingService: gRPC JoinMeeting(userId, roomId, password)
    CallingService->>LiveKit: 生成JoinToken
    LiveKit-->>CallingService: Token
    CallingService-->>Gateway: 返回Token + roomName
    Gateway-->>Client: 200 OK
```

### 3.3 结束会议

```mermaid
sequenceDiagram
    participant Client
    participant Gateway
    participant CallingService
    participant LiveKit
    participant NATS

    Client->>Gateway: POST /meeting/end<br/>Header: Authorization: Bearer {token}<br/>Body: {room_id}
    Gateway->>CallingService: gRPC EndMeeting(roomId)
    CallingService->>LiveKit: 关闭Room
    CallingService->>NATS: 发布会议结束事件
    CallingService-->>Gateway: 成功
    Gateway-->>Client: 200 OK
```

## 4. API设计

### 3.1 创建会议

```protobuf
message CreateMeetingRequest {
    string creator_id = 1;
    string title = 2;
    string password = 3; // 可选
    int32 max_participants = 4;
}

message CreateMeetingResponse {
    string room_id = 1;
    string room_name = 2;
    string token = 3;
}
```

### 3.2 加入会议

```protobuf
message JoinMeetingRequest {
    string user_id = 1;
    string room_id = 2;
    string password = 3;
}

message JoinMeetingResponse {
    string token = 1;
    string room_name = 2;
}
```

## 4. 会议设置

| 设置项 | 说明 |
|--------|------|
| 成员入会静音 | 新成员入会自动静音 |
| 允许成员自我解除静音 | |
| 允许成员开启视频 | |
| 主持人共享屏幕 | 只有主持人可共享 |
| 会议密码 | 可选 |

## 5. 依赖服务

- **LiveKit**: 视频会议引擎
