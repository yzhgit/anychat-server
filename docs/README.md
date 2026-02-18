# AnyChat 文档

欢迎来到 AnyChat 即时通讯系统的文档站点。

## 快速导航

### 开始使用

- [快速开始](development/getting-started.md) - 快速了解和运行项目
- [服务启动指南](development/service-startup-guide.md) - 详细的服务启动步骤
- [端口分配](development/port-allocation.md) - 系统端口规划和说明

### API 文档

- [API 快速开始](api/QUICKSTART.md) - API 使用入门
- [Gateway HTTP API](api/gateway-http-api.md) - 网关 HTTP 接口文档（交互式）

### 设计文档

- [系统架构设计](design/instant-messaging-backend-design.md) - 整体架构和设计思路

## 项目简介

AnyChat 是一个基于微服务架构的即时通讯（IM）后端系统，使用 Go 语言开发。系统由 12 个独立的微服务组成，通过 gRPC 和 HTTP 进行通信，使用 NATS 进行异步消息传递。

### 核心特性

- **微服务架构** - 12 个独立服务，职责清晰，易于扩展
- **高性能** - 基于 Go 语言和 gRPC，支持高并发
- **实时通信** - WebSocket 支持，消息实时推送
- **多设备同步** - 支持多端登录和消息同步
- **音视频通话** - 集成 LiveKit 实现音视频功能
- **可观测性** - Prometheus 监控、Jaeger 追踪、Grafana 可视化

### 技术栈

- **语言**: Go 1.24+
- **框架**: Gin (HTTP), gRPC (RPC)
- **数据库**: PostgreSQL 18.0
- **缓存**: Redis 7.0+
- **消息队列**: NATS with JetStream
- **对象存储**: MinIO
- **监控**: Prometheus + Grafana + Jaeger

## 文档组织

本文档站点包含以下内容：

1. **开发文档** - 帮助开发者快速上手和开发
2. **API 文档** - 自动生成的交互式 API 文档
3. **设计文档** - 架构设计和技术方案

## 贡献

欢迎提交 Issue 和 Pull Request 来帮助改进文档和项目。

## 许可证

MIT License - 详见 [LICENSE](LICENSE.md) 文件
