# AnyChat - 即时通讯后端系统

基于Go语言开发的微服务架构IM系统。

## 功能特性

- 🚀 私聊、群聊
- 📞 音视频通话
- 📁 文件传输
- ✅ 消息已读回执
- 🔄 多端同步
- 📱 离线推送

## 技术栈

- **开发语言**: Go 1.24+
- **数据库**: PostgreSQL 18+
- **缓存**: Redis 7.0+
- **消息队列**: NATS
- **对象存储**: MinIO
- **音视频**: LiveKit
- **监控**: Prometheus + Grafana
- **链路追踪**: Jaeger
- **构建工具**: Mage

## 快速开始

### 环境要求

- Go 1.24+
- Docker & Docker Compose
- Mage (构建工具)

### 安装 Mage

```bash
go install github.com/magefile/mage@latest
```

### 本地开发

```bash
# 1. 克隆代码
git clone https://github.com/yzhgit/anchat-server
cd server

# 2. 安装依赖
mage deps

# 3. 安装开发工具（可选）
mage install

# 4. 启动基础设施
mage docker:up

# 5. 运行数据库迁移
mage db:up

# 6. 启动服务
mage dev:auth
mage dev:gateway
```

## 项目结构

```
anychat_server/
├── api/                    # API定义
│   └── proto/             # gRPC定义
├── cmd/                    # 应用入口
├── internal/               # 私有代码
├── pkg/                    # 公共库
├── deployments/            # 部署配置
├── configs/                # 配置文件
├── migrations/             # 数据库迁移
├── docs/                   # 文档
│   └── api/swagger/       # OpenAPI规范（自动生成）
├── tests/                  # 测试
└── magefile.go            # Mage构建脚本
```

## 构建

```bash
# 查看所有可用命令
mage -l

# 构建所有服务
mage build:all

# 构建特定服务
mage build:auth
mage build:gateway

# 构建Docker镜像
mage docker:build
```

## 测试

```bash
# 运行所有测试
mage test:all

# 运行单元测试
mage test:unit

# 生成覆盖率报告
mage test:coverage

# 代码检查
mage lint

# 代码格式化
mage fmt
```

## Mage 常用命令

### 构建相关
- `mage build:all` - 构建所有服务
- `mage build:auth` - 构建认证服务
- `mage build:user` - 构建用户服务
- `mage build:gateway` - 构建网关服务
- `mage build:message` - 构建消息服务

### 开发相关
- `mage dev:auth` - 运行认证服务
- `mage dev:gateway` - 运行网关服务
- `mage dev:message` - 运行消息服务
- `mage proto` - 生成protobuf代码

### Docker相关
- `mage docker:up` - 启动所有容器
- `mage docker:down` - 停止所有容器
- `mage docker:build` - 构建Docker镜像
- `mage docker:logs` - 查看日志
- `mage docker:ps` - 查看容器状态

### 数据库相关
- `mage db:up` - 运行数据库迁移
- `mage db:down` - 回滚数据库迁移
- `mage db:create <name>` - 创建新的迁移文件

### 文档相关
- `mage docs:generate` - 生成 API 文档
- `mage docs:serve` - 启动文档服务器（http://localhost:3000）
- `mage docs:build` - 构建文档站点
- `mage docs:validate` - 验证 API 文档

### 其他
- `mage deps` - 安装依赖
- `mage install` - 安装开发工具
- `mage clean` - 清理构建产物
- `mage mock` - 生成Mock代码

## 文档

### 在线文档

- **完整文档站点**: [GitHub Pages](https://yzhgit.github.io/anchat-server/) (自动部署)
- **本地预览**: 运行 `mage docs:serve` 后访问 http://localhost:3000

### 文档内容

- [快速开始](docs/development/getting-started.md) - 新手入门指南
- [API 文档](docs/api/gateway-http-api.md) - 交互式 HTTP API 文档
- [系统设计](docs/design/instant-messaging-backend-design.md) - 架构设计文档
- [API 文档编写](docs/development/writing-api-docs.md) - 如何编写 API 文档

### 生成和部署文档

#### 本地生成

```bash
# 生成 API 文档
mage docs:generate

# 本地预览文档站点
mage docs:serve

# 构建静态文档（用于部署）
mage docs:build
```

#### 自动部署

- **触发条件**: 推送到 main 分支或创建 Pull Request
- **部署目标**: GitHub Pages
- **文档地址**: https://yzhgit.github.io/anchat-server/

文档会在以下情况自动更新：
1. Gateway 服务代码变更
2. 文档文件变更
3. CI 配置变更

#### 编写 API 文档

为 Gateway HTTP 接口添加 Swagger 注释：

```go
// Login 用户登录
// @Summary      用户登录
// @Description  用户通过账号密码登录
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "登录信息"
// @Success      200      {object}  response.Response{data=AuthResponse}  "登录成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
    // ...
}
```

详细说明请参考 [API 文档编写指南](docs/development/writing-api-docs.md)。

### 其他

欢迎提交 Pull Request 和 Issue。

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件
