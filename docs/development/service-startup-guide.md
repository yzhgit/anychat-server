# AnyChat 服务启动指南

## 问题解决方案

端口冲突问题已解决。启动服务的正确方法：

### 方案 1: 使用独立终端（推荐用于开发调试）

打开 **3 个终端窗口**，按顺序执行：

**终端 1 - 启动 user-service:**
```bash
cd /home/mosee/Develop/anchat_server
mage dev:user
```

**终端 2 - 启动 auth-service:**
```bash
cd /home/mosee/Develop/anchat_server
mage dev:auth
```

**终端 3 - 启动 gateway-service:**
```bash
cd /home/mosee/Develop/anchat_server
mage dev:gateway
```

### 方案 2: 使用后台脚本（推荐用于测试）

```bash
# 启动所有服务
./scripts/start-services.sh

# 查看日志
tail -f logs/gateway-service.log
tail -f logs/auth-service.log
tail -f logs/user-service.log

# 停止所有服务
./scripts/stop-services.sh
```

### 方案 3: 使用 tmux/screen（推荐用于生产环境）

```bash
# 安装 tmux
sudo apt-get install tmux  # Ubuntu/Debian
brew install tmux          # macOS

# 启动 tmux 会话
tmux new -s anychat

# 创建窗口并启动服务
# Ctrl+B, C  创建新窗口
# Ctrl+B, N  切换到下一个窗口
# Ctrl+B, P  切换到上一个窗口
# Ctrl+B, D  detach 会话

# 窗口 0: user-service
mage dev:user

# 窗口 1: auth-service (新建窗口)
mage dev:auth

# 窗口 2: gateway-service (新建窗口)
mage dev:gateway

# Detach: Ctrl+B, D
# Re-attach: tmux attach -t anychat
```

## 端口验证

启动前检查端口：
```bash
./scripts/check-ports.sh --check
```

清理进程：
```bash
./scripts/check-ports.sh --clean
```

## 服务验证

所有服务启动后，验证：

```bash
# 健康检查
curl http://localhost:8080/health

# 查看端口
lsof -i :8080  # gateway
lsof -i :9001  # auth gRPC
lsof -i :9002  # user gRPC

# 运行 API 测试
./tests/e2e/test-e2e.sh
```

## 常见问题

### Q1: 端口被占用
**症状**: `bind: address already in use`

**解决**:
```bash
./scripts/stop-services.sh
./scripts/check-ports.sh --check
```

### Q2: 服务无法连接
**症状**: `Failed to connect to backend services`

**解决**:
1. 确保按顺序启动：user → auth → gateway
2. 等待后端服务完全启动（看到 "gRPC server listening" 日志）
3. 检查端口配置是否正确

### Q3: 数据库连接失败
**症状**: `Failed to connect database`

**解决**:
```bash
# 确保 Docker 基础设施运行
mage docker:up
mage docker:ps

# 运行数据库迁移
mage db:up
```

## 端口分配

| 服务 | HTTP | gRPC | 说明 |
|------|------|------|------|
| gateway-service | 8080 | - | API 网关 |
| auth-service | 8001 | 9001 | 认证服务 |
| user-service | 8002 | 9002 | 用户服务 |

详细端口分配: `docs/development/port-allocation.md`
