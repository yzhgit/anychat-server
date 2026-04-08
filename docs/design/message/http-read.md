# HTTP 已读与未读设计

## 1. 背景与目标

系统存在两类“已读”语义：

- **会话级已读（Conversation 视角）**：清空会话未读角标；
- **消息级已读（Message 视角）**：维护 `last_read_seq` 与已读回执。

目标是同时支持：

1. 一键“本会话全部已读”；
2. 滚动列表时“按消息ID批量标记已读”。

## 2. 服务职责划分

- `ConversationService.ClearUnread`
  - 作用：会话列表角标归零。
  - 不负责消息粒度回执。
- `MessageService.MarkAsRead`
  - 作用：推进会话消息已读游标（`last_read_seq`），用于回执和会话内未读计算。
- `MessageService.GetUnreadCount`
  - 作用：返回“某个会话”的未读数。
- `ConversationService.GetTotalUnread`
  - 作用：返回“用户所有会话”的未读总数。

## 3. 路由设计（推荐）

### 3.1 会话级全部已读

- `POST /api/v1/conversations/:conversationId/read-all`

行为：

- Gateway 先调用 `MessageService.GetConversationSequence + MarkAsRead`，将 `last_read_seq` 推到当前最大值；
- 然后调用 `ConversationService.ClearUnread` 清零会话角标。

### 3.2 指定消息ID批量已读（新增能力）

- `POST /api/v1/conversations/:conversationId/messages/read`

请求体：

```json
{
  "message_ids": ["msg1", "msg2", "msg3"],
  "client_read_at": 1712550000,
  "idempotency_key": "read-batch-001"
}
```

响应体（data）：

```json
{
  "accepted_ids": ["msg1", "msg2"],
  "ignored_ids": ["msg3"],
  "advanced_last_read_seq": 1024
}
```

对应 gRPC：

```protobuf
rpc MarkMessagesRead(MarkMessagesReadRequest) returns (MarkMessagesReadResponse);
```

## 4. 读取状态查询路由

### 4.1 单会话未读数

- `GET /api/v1/conversations/:conversationId/messages/unread-count`
- gRPC: `MessageService.GetUnreadCount`

### 4.2 会话已读回执

- `GET /api/v1/conversations/:conversationId/messages/read-receipts`
- gRPC: `MessageService.GetReadReceipts`

### 4.3 用户总未读数

- `GET /api/v1/conversations/unread/total`
- gRPC: `ConversationService.GetTotalUnread`

## 5. 兼容与迁移

- 路径统一使用 `GET /api/v1/conversations/:conversationId/messages/*`，避免资源层级混乱。
