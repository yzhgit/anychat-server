# Message Service (消息服务)

## 1. 服务概述

**职责**: 消息存储、消息路由、消息状态管理

**核心功能**:
- 消息发送（单聊、群聊）
- 消息类型支持（文本、图片、视频、语音、文件等）
- 消息状态管理（发送中、已送达、已读）
- 消息操作（撤回、删除、转发、引用）
- 消息搜索
- 离线消息处理

## 2. 文档导航

| 功能 | 文档 | 说明 |
|------|------|------|
| 消息发送 | [send.md](send.md) | 单聊/群聊消息发送 |
| 消息撤回 | [recall.md](recall.md) | 消息撤回功能 |
| 已读回执 | [read-receipt.md](read-receipt.md) | 已读回执功能 |
| HTTP发送消息 | [http-send.md](http-send.md) | HTTP 发送消息能力 |
| HTTP消息查询 | [http-query.md](http-query.md) | 历史消息、消息详情、序列号 |
| HTTP已读与未读 | [http-read.md](http-read.md) | 会话已读、逐条已读（设计）、未读数与回执 |
| HTTP消息搜索 | [http-search.md](http-search.md) | 关键词搜索消息 |
| 消息队列架构 | [message-service-architecture.md](message-service-architecture.md) | 服务架构设计 |
| 消息队列对比 | [message-queue-comparison.md](message-queue-comparison.md) | 技术选型对比 |

## 3. 数据模型

- **Message**: 消息主表（按月分表）
- **MessageStatus**: 消息状态表
- **MessageRead**: 消息已读记录（群聊）
- **MessageReference**: 消息引用关系
- **MessageEdit**: 消息编辑记录

## 4. 推送通知

- `notification.message.new.{to_user_id}` - 新消息通知
- `notification.message.read_receipt.{from_user_id}` - 消息已读回执通知
- `notification.message.recalled.{conversation_id}` - 消息撤回通知
- `notification.message.typing.{to_user_id}` - 正在输入提示
- `notification.message.mentioned.{user_id}` - @提及通知

## 5. 依赖服务

- **User Service**: 发送者信息
- **Friend Service**: 好友关系校验
- **Group Service**: 群成员校验
- **File Service**: 媒体文件
- **Gateway Service**: 实时推送
- **NATS**: 消息分发
- **Redis**: 消息缓存、序列号生成
- **PostgreSQL**: 消息持久化

---

返回: [后端总体设计](../backend-design.md)
