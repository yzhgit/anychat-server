# 快速开始指南

## 环境准备

### 1. 安装Go环境

确保安装了Go 1.24或更高版本：

```bash
go version
```

### 2. 安装Docker和Docker Compose

```bash
# 验证Docker
docker --version
docker-compose --version
```

### 3. 安装Mage构建工具

Mage是基于Go的构建工具，类似于Make但使用Go代码定义任务。

```bash
# 安装Mage
go install github.com/magefile/mage@latest

# 验证安装
mage -version
```

## 本地开发

### 1. 克隆项目

```bash
git clone https://github.com/yzhgit/anychat-server
cd server
```

### 2. 安装依赖

```bash
# 安装项目依赖
mage deps

# 安装开发工具（可选但推荐）
# 包括: golangci-lint, migrate, mockgen, protoc-gen-go等
mage install
```

### 3. 查看所有可用命令

```bash
# 列出所有Mage任务
mage -l
```

输出示例：
```
Targets:
  build:all          builds all services
  build:auth         builds auth-service
  build:gateway      builds gateway-service
  build:message      builds message-service
  build:user         builds user-service
  clean              removes build artifacts
  db:create          creates a new migration file
  db:down            runs database migrations down
  db:up              runs database migrations up
  deps               installs dependencies
  depsCheck          verifies dependencies
  dev:auth           runs auth-service locally
  dev:gateway        runs gateway-service locally
  dev:message        runs message-service locally
  dev:user           runs user-service locally
  docker:build       builds docker images
  docker:down        stops docker compose
  docker:logs        shows docker compose logs
  docker:ps          shows docker compose status
  docker:up          starts docker compose
  fmt                formats code
  install            installs required tools
  lint               runs linter
  mock               generates mock code
  proto              generates protobuf code
  test:all           runs all tests
  test:coverage      generates test coverage report
  test:unit          runs unit tests
```

### 4. 启动基础设施

```bash
# 启动PostgreSQL、Redis、NATS、MinIO等基础设施
mage docker:up

# 检查服务状态
mage docker:ps
```

### 5. 访问服务

启动后可以访问以下服务：

- **PostgreSQL**: localhost:5432
  - 用户名: anychat
  - 密码: anychat123
  - 数据库: anychat

- **Redis**: localhost:6379

- **NATS**: localhost:4222 (客户端), localhost:8222 (管理)

- **MinIO**:
  - API: localhost:9000
  - Console: http://localhost:9001
  - 用户名: minioadmin
  - 密码: minioadmin

- **Prometheus**: http://localhost:9090

- **Grafana**: http://localhost:3000
  - 用户名: admin
  - 密码: admin

- **Jaeger**: http://localhost:16686

### 6. 数据库迁移

```bash
# 运行迁移
mage db:up

# 创建新的迁移文件
mage db:create create_users_table

# 回滚迁移
mage db:down
```

### 7. 运行服务

```bash
# 运行auth服务
mage dev:auth

# 运行gateway服务
mage dev:gateway

# 运行message服务
mage dev:message

# 运行user服务
mage dev:user
```

## 开发流程

### 构建服务

```bash
# 构建所有服务
mage build:all

# 构建特定服务
mage build:auth
mage build:gateway
mage build:message
mage build:user
```

构建的二进制文件将输出到 `bin/` 目录。

### 添加新功能

1. 在对应的service层添加业务逻辑
2. 在handler层添加HTTP接口
3. 在proto文件中定义gRPC接口（如需要）
4. 生成protobuf代码: `mage proto`
5. 编写单元测试
6. 更新API文档

### 代码规范

```bash
# 格式化代码
mage fmt

# 代码检查
mage lint

# 运行测试
mage test:all

# 生成测试覆盖率报告
mage test:coverage
```

### 生成代码

```bash
# 生成protobuf代码
mage proto

# 生成Mock代码
mage mock
```

### 提交代码

```bash
# 提交格式
git commit -m "feat(auth): 添加用户注册功能"
```

提交类型：
- `feat`: 新功能
- `fix`: 修复bug
- `docs`: 文档更新
- `refactor`: 重构
- `test`: 测试
- `chore`: 构建/工具变动

## Mage常用命令速查

### 开发相关
```bash
mage deps                # 安装依赖
mage install             # 安装开发工具
mage dev:auth            # 运行auth服务
mage dev:gateway         # 运行gateway服务
mage fmt                 # 格式化代码
mage lint                # 代码检查
```

### 构建相关
```bash
mage build:all           # 构建所有服务
mage build:auth          # 构建auth服务
mage clean               # 清理构建产物
mage proto               # 生成protobuf代码
```

### 测试相关
```bash
mage test:all            # 运行所有测试
mage test:unit           # 运行单元测试
mage test:coverage       # 生成覆盖率报告
```

### Docker相关
```bash
mage docker:up           # 启动所有容器
mage docker:down         # 停止所有容器
mage docker:logs         # 查看日志
mage docker:ps           # 查看容器状态
mage docker:build        # 构建Docker镜像
```

### 数据库相关
```bash
mage db:up               # 运行迁移
mage db:down             # 回滚迁移
mage db:create <name>    # 创建迁移文件
```

## 常见问题

### Q: Mage命令找不到？

确保已安装Mage并且 `$GOPATH/bin` 在PATH中：

```bash
go install github.com/magefile/mage@latest
export PATH=$PATH:$(go env GOPATH)/bin
```

### Q: 端口已被占用怎么办？

修改 `deployments/docker-compose.yml` 中的端口映射。

### Q: 数据库连接失败？

检查PostgreSQL是否正常启动：

```bash
mage docker:logs
```

或者直接查看PostgreSQL日志：

```bash
docker-compose -f deployments/docker-compose.yml logs postgres
```

### Q: 如何清理Docker环境？

```bash
# 停止并删除所有容器
mage docker:down

# 删除所有卷（慎用，会删除数据！）
docker volume prune
```

### Q: 如何查看Mage任务的详细信息？

```bash
# 查看任务列表
mage -l

# 查看特定任务的帮助（如果有）
mage -h build:all
```

### Q: 运行测试时出现权限错误？

确保有足够的权限访问测试文件和数据库：

```bash
# Linux/Mac可能需要
chmod -R 755 tests/
```

## 进阶用法

### 自定义Mage任务

你可以在 `magefile.go` 中添加自己的任务：

```go
// Custom runs a custom task
func Custom() error {
    fmt.Println("Running custom task...")
    return sh.RunV("echo", "Hello from custom task!")
}
```

然后运行：

```bash
mage custom
```

### 任务依赖

Mage支持任务依赖，例如：

```go
func (Build) All() error {
    mg.Deps(ensureBinDir)  // 先执行ensureBinDir
    // ... 构建逻辑
}
```

### 并行执行

某些任务可以并行执行：

```bash
# 并行运行多个服务（需要多个终端）
# 终端1
mage dev:auth

# 终端2
mage dev:gateway

# 终端3
mage dev:message
```

## 下一步

- 查看[设计文档](/design/backend-design.md)了解系统架构
- 查看[API文档](/api/README.md)了解接口定义
- 阅读 `magefile.go` 了解更多构建任务
- 开始实现你的第一个功能！
