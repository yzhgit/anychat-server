# Scripts 使用指南

本目录包含 AnyChat 项目的环境管理和服务管理脚本。

## 📁 脚本列表

### 环境管理

- **`setup-env.sh`** - 环境检查和设置脚本
  - 检查开发工具（Go, Docker, jq, curl 等）
  - 检查基础设施服务（PostgreSQL, Redis, NATS, MinIO）
  - 检查微服务状态
  - 检查数据库连接和 Go 模块
  - 提供快速修复建议

- **`health-check.sh`** - 服务健康检查
  - 检查所有微服务的 HTTP 健康端点
  - 检查 gRPC 端口是否监听
  - 检查基础设施服务端口
  - 检查 Docker 容器状态

- **`check-ports.sh`** - 端口检查工具
  - 检查所有服务端口占用情况
  - 识别端口冲突

### 服务管理

- **`start-services.sh`** - 启动所有微服务
- **`stop-services.sh`** - 停止所有微服务

## 🚀 快速开始

### 1. 环境检查

首次使用或遇到问题时，运行环境检查：

```bash
./scripts/setup-env.sh
```

这会检查：
- ✓ 开发工具是否安装
- ✓ Go 模块是否正常
- ✓ 基础设施是否运行
- ✓ 数据库连接是否正常

### 2. 启动基础设施

```bash
# 启动 PostgreSQL, Redis, NATS, MinIO
mage docker:up

# 检查容器状态
mage docker:ps

# 运行数据库迁移
mage db:up
```

### 3. 启动微服务

使用一键启动脚本，或在不同终端分别启动：

```bash
# 一键启动所有服务
./scripts/start-services.sh

# 或按需单独启动
mage dev:auth
mage dev:user
mage dev:friend
mage dev:group
mage dev:message
mage dev:session
mage dev:file
mage dev:gateway
mage dev:push
mage dev:calling
mage dev:sync
mage dev:admin
```

### 4. 健康检查

```bash
./scripts/health-check.sh
```

预期输出：
```
========================================
基础设施服务
========================================
✓ PostgreSQL - 端口 5432 正在监听
✓ Redis - 端口 6379 正在监听
✓ NATS - 端口 4222 正在监听
✓ MinIO API - 端口 9000 正在监听
✓ MinIO Console - 端口 9091 正在监听

========================================
HTTP 服务
========================================
✓ Auth Service - 健康 (HTTP 200)
✓ User Service - 健康 (HTTP 200)
✓ Friend Service - 健康 (HTTP 200)
✓ Gateway Service - 健康 (HTTP 200)

========================================
健康检查总结
========================================
健康服务: 13 / 13 (100%)
✓ 所有服务健康!
```

## 🔧 常见问题

### 问题 1: 服务未运行

**错误信息**：
```
✗ Gateway Service (端口 8080) 未运行
部分服务未运行，请先启动所有服务
```

**解决方法**：
```bash
# 检查哪些服务未运行
./scripts/health-check.sh

# 启动缺失的服务
mage dev:gateway  # 或其他服务
```

### 问题 2: 基础设施未运行

**错误信息**：
```
✗ PostgreSQL - 端口 5432 未监听
```

**解决方法**：
```bash
# 启动基础设施
mage docker:up

# 等待服务就绪（约10秒）
sleep 10

# 运行数据库迁移
mage db:up
```

### 问题 3: 端口冲突

**错误信息**：
```
bind: address already in use
```

**解决方法**：
```bash
# 检查端口占用
./scripts/check-ports.sh

# 或手动检查
lsof -i :8080

# 停止占用端口的进程
kill -9 <PID>
```

### 问题 4: 数据库连接失败

**错误信息**：
```
Error: database connection failed
```

**解决方法**：
```bash
# 检查 PostgreSQL 状态
docker ps | grep postgres

# 检查数据库连接
PGPASSWORD=anychat123 psql -h localhost -U anychat -d anychat -c "SELECT 1"

# 如果失败，重启 PostgreSQL
docker restart postgres
```

### 问题 5: jq 命令未找到

**错误信息**：
```
需要安装 jq 工具
```

**解决方法**：
```bash
# Ubuntu/Debian
sudo apt-get install jq

# macOS
brew install jq

# CentOS/RHEL
sudo yum install jq
```

## 🎯 最佳实践

1. **开发前检查环境**
   ```bash
   ./scripts/setup-env.sh
   ```

2. **定期运行健康检查**
   ```bash
   ./scripts/health-check.sh
   ```

3. **遇到问题时查看日志**
   ```bash
   # 查看基础设施日志
   mage docker:logs

   # 查看特定容器日志
   docker logs postgres
   docker logs redis
   ```

4. **清理和重启**
   ```bash
   # 停止所有服务
   mage docker:down

   # 清理构建产物
   mage clean

   # 重新启动
   mage docker:up
   mage db:up
   ```

## 📚 更多资源

- [项目 README](../README.md)
- [API 测试](../tests/README.md)
- [API 文档](../docs/api/)
- [开发指南](../docs/development/)
- [设计文档](../docs/design/)

## 📄 许可证

MIT License
