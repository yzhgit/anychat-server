# 端口分配规划

AnyChat 项目完整端口分配表，避免端口冲突。

## 端口分配规则

- **5xxx**: 数据库
- **6xxx**: 缓存
- **4xxx**: 消息队列
- **8xxx**: 微服务 HTTP/REST API
- **9xxx**: 微服务 gRPC
  - **90xx**: 基础设施管理端口
  - **91xx**: 监控相关端口

## 基础设施服务

| 服务 | 端口 | 协议 | 说明 |
|------|------|------|------|
| **PostgreSQL** | 5432 | TCP | 主数据库 |
| **Redis** | 6379 | TCP | 缓存和会话存储 |
| **NATS** | 4222 | TCP | 消息队列客户端端口 |
| **NATS Monitoring** | 8222 | HTTP | NATS 监控接口 |
| **MinIO API** | 9000 | HTTP | 对象存储 API |
| **MinIO Console** | 9091 | HTTP | MinIO 管理控制台 |

## 微服务端口

### 核心服务

| 服务名 | HTTP 端口 | gRPC 端口 | 说明 |
|--------|-----------|-----------|------|
| **gateway-service** | 8080 | - | API 网关，对外统一入口 |
| **auth-service** | 8001 | 9001 | 认证授权服务 |
| **user-service** | 8002 | 9002 | 用户资料服务 |

### 社交服务

| 服务名 | HTTP 端口 | gRPC 端口 | 说明 |
|--------|-----------|-----------|------|
| **friend-service** | 8003 | 9003 | 好友关系服务 |
| **group-service** | 8004 | 9004 | 群组管理服务 |

### 消息服务

| 服务名 | HTTP 端口 | gRPC 端口 | 说明 |
|--------|-----------|-----------|------|
| **message-service** | 8005 | 9005 | 消息存储和投递 |
| **session-service** | 8006 | 9006 | 会话管理服务 |

### 扩展服务

| 服务名 | HTTP 端口 | gRPC 端口 | 说明 |
|--------|-----------|-----------|------|
| **file-service** | 8007 | 9007 | 文件上传下载服务 |
| **push-service** | 8008 | 9008 | 推送通知服务 |
| **rtc-service** | 8009 | 9009 | 音视频通话服务 |
| **sync-service** | 8010 | 9010 | 多端同步服务 |
| **admin-service** | 8011 | 9011 | 管理后台服务 |

## 监控和管理

| 服务 | 端口 | 协议 | 说明 |
|------|------|------|------|
| **Prometheus** | 9090 | HTTP | 指标采集和存储 |
| **Grafana** | 3000 | HTTP | 可视化监控面板 (admin/admin) |
| **Jaeger UI** | 16686 | HTTP | 分布式追踪 UI |
| **Jaeger Collector** | 14268 | HTTP | 追踪数据收集器 |
| **Metrics (所有服务)** | 2112 | HTTP | Prometheus 指标端点 |

## 第三方服务

| 服务 | 端口 | 协议 | 说明 |
|------|------|------|------|
| **LiveKit** | 7880 | WebSocket | LiveKit 音视频服务器 |


## 端口使用指南

### 添加新服务

1. **HTTP 端口**: 从 8012 开始递增
2. **gRPC 端口**: 从 9012 开始递增
3. **更新本文档**: 在相应表格中添加新服务信息

### 本地开发

所有服务默认监听 `localhost`，确保：
- Docker 基础设施端口与主机端口一致
- 微服务可以通过 `localhost:<port>` 访问基础设施
- 服务间通过 `localhost:<grpc-port>` 互相调用

### Docker 部署

在 Docker 环境中：
- 基础设施使用容器名称（如 `postgres`, `redis`）
- 微服务之间通过服务名称解析
- 外部访问通过主机端口映射

## 配置文件参考

### configs/config.yaml

```yaml
server:
  http_port: 8001  # 各服务不同
  grpc_port: 9001  # 各服务不同
  mode: debug      # debug/release

services:
  auth:
    grpc_addr: localhost:9001
  user:
    grpc_addr: localhost:9002
  # 其他服务...

database:
  postgres:
    host: localhost
    port: 5432
    # ...

redis:
  host: localhost
  port: 6379
  # ...

nats:
  url: nats://localhost:4222
  # ...

minio:
  endpoint: localhost:9000
  console_endpoint: localhost:9091
  # ...
```

## 端口检查命令

```bash
# 检查端口占用
lsof -i :8001  # HTTP
lsof -i :9001  # gRPC

# 查看所有微服务端口
lsof -i :8001-8011

# 查看所有基础设施端口
docker-compose -f deployments/docker-compose.yml ps

# 检查端口冲突
netstat -tuln | grep -E ':(5432|6379|4222|8222|9000|9091|8080|8001|9001|8002|9002)'
```

## 故障排查

### 端口已被占用

```bash
# 1. 查找占用进程
lsof -i :<port>

# 2. 停止进程
kill -9 <PID>

# 3. 或使用 mage 命令重启
mage docker:down  # 停止基础设施
pkill -f auth-service  # 停止微服务
```

### 服务无法启动

1. 检查端口是否被占用
2. 检查配置文件中的端口设置
3. 检查 Docker 容器状态 `mage docker:ps`
4. 查看服务日志

## 更新记录

| 日期 | 变更 | 说明 |
|------|------|------|
| 2024-02-15 | 初始版本 | 创建端口分配规划 |
| 2024-02-15 | MinIO Console 端口 | 从 9001 改为 9091，避免与 auth-service 冲突 |
| 2024-02-15 | auth-service gRPC | 从 9003 改为 9001，遵循端口规划 |
