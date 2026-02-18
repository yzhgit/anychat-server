# 编写 API 文档注释

本指南说明如何为 Gateway HTTP 接口编写 Swagger 注释，以便自动生成 API 文档。

## 概述

AnyChat 使用 [swaggo/swag](https://github.com/swaggo/swag) 工具从 Go 代码注释中自动生成 OpenAPI 规范文档。所有注释都遵循 Swagger 2.0 规范。

## 全局配置

在 `cmd/gateway-service/main.go` 文件的 package 注释中定义全局 API 信息：

```go
// @title           AnyChat Gateway API
// @version         1.0
// @description     AnyChat 即时通讯系统的网关 API 服务

// @contact.name   AnyChat API Support
// @contact.url    https://github.com/yzhgit/anchat-server
// @contact.email  support@anychat.example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
```

## 接口注释

### 基本结构

每个 HTTP handler 函数都需要添加 Swagger 注释：

```go
// FunctionName 简短描述
// @Summary      接口摘要（一句话）
// @Description  详细描述接口功能
// @Tags         标签分类
// @Accept       json
// @Produce      json
// @Param        参数名  类型  数据类型  必填  "说明"
// @Success      200  {object}  响应类型  "成功描述"
// @Failure      400  {object}  响应类型  "失败描述"
// @Router       /path [method]
func (h *Handler) FunctionName(c *gin.Context) {
    // ...
}
```

### 详细说明

#### @Summary 和 @Description

- `@Summary`: 接口的简短摘要（建议一行，不超过 120 字符）
- `@Description`: 详细描述，可以多行

```go
// @Summary      用户登录
// @Description  用户通过账号密码登录，支持手机号和邮箱登录
```

#### @Tags

用于分组，相同 tag 的接口会在文档中归为一组：

```go
// @Tags  认证
// @Tags  用户
```

#### @Accept 和 @Produce

指定请求和响应的内容类型：

```go
// @Accept   json
// @Produce  json
```

#### @Param

定义请求参数，格式：`@Param 参数名 位置 数据类型 必填 "说明" [其他属性]`

**位置类型**:
- `query` - URL 查询参数
- `path` - 路径参数
- `header` - HTTP Header
- `body` - 请求体
- `formData` - 表单数据

**示例**:

```go
// 路径参数
// @Param  userId  path  string  true  "用户ID"

// 查询参数
// @Param  page     query  int     false  "页码"  default(1)
// @Param  keyword  query  string  true   "搜索关键字"

// 请求体
// @Param  request  body  LoginRequest  true  "登录信息"
```

#### @Success 和 @Failure

定义响应，格式：`@Success 状态码 {类型} 数据类型 "说明"`

```go
// @Success  200  {object}  response.Response{data=AuthResponse}  "登录成功"
// @Failure  400  {object}  response.Response  "参数错误"
// @Failure  401  {object}  response.Response  "认证失败"
// @Failure  500  {object}  response.Response  "服务器错误"
```

**泛型响应**:
使用 `{data=类型}` 语法指定 data 字段的具体类型：

```go
// @Success  200  {object}  response.Response{data=UserProfile}
```

#### @Security

需要认证的接口添加安全标识：

```go
// @Security  BearerAuth
```

#### @Router

定义路由路径和 HTTP 方法：

```go
// @Router  /auth/login [post]
// @Router  /users/{userId} [get]
// @Router  /users/me [put]
```

## 数据模型

### 定义请求/响应结构体

为了让文档更清晰，建议定义独立的请求和响应结构体：

```go
// LoginRequest 用户登录请求
type LoginRequest struct {
    Account    string `json:"account" binding:"required" example:"13800138000"`
    Password   string `json:"password" binding:"required" example:"password123"`
    DeviceType string `json:"deviceType" binding:"required" example:"ios" enums:"ios,android,web"`
}

// AuthResponse 认证响应
type AuthResponse struct {
    UserID       string `json:"userId" example:"user-123"`
    AccessToken  string `json:"accessToken" example:"eyJhbGciOiJIUzI1NiI..."`
    RefreshToken string `json:"refreshToken" example:"eyJhbGciOiJIUzI1NiI..."`
    ExpiresIn    int64  `json:"expiresIn" example:"7200"`
}
```

### 结构体标签说明

- `json:"字段名"` - JSON 序列化名称（必须）
- `binding:"required"` - Gin 验证标签
- `example:"示例值"` - 在文档中显示的示例
- `enums:"值1,值2"` - 枚举值
- `default:"默认值"` - 默认值
- `format:"格式"` - 数据格式（如 `date-time`）
- `minimum:"最小值"` - 数值最小值
- `maximum:"最大值"` - 数值最大值

## 完整示例

### 公开接口（无需认证）

```go
// Login 用户登录
// @Summary      用户登录
// @Description  用户通过账号密码登录，账号支持手机号或邮箱
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "登录信息"
// @Success      200      {object}  response.Response{data=AuthResponse}  "登录成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "账号或密码错误"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
    var req LoginRequest
    // ...
}
```

### 需要认证的接口

```go
// UpdateProfile 更新个人资料
// @Summary      更新个人资料
// @Description  更新当前登录用户的个人资料信息
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      UpdateProfileRequest  true  "资料信息"
// @Success      200      {object}  response.Response{data=UserProfile}  "更新成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /users/me [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
    var req UpdateProfileRequest
    // ...
}
```

### 带路径参数的接口

```go
// GetUserInfo 获取用户信息
// @Summary      获取用户信息
// @Description  获取指定用户的公开信息
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userId  path      string  true  "用户ID"
// @Success      200     {object}  response.Response{data=UserInfo}  "获取成功"
// @Failure      401     {object}  response.Response  "未授权"
// @Failure      404     {object}  response.Response  "用户不存在"
// @Failure      500     {object}  response.Response  "服务器错误"
// @Router       /users/{userId} [get]
func (h *UserHandler) GetUserInfo(c *gin.Context) {
    userId := c.Param("userId")
    // ...
}
```

### 带查询参数的接口

```go
// SearchUsers 搜索用户
// @Summary      搜索用户
// @Description  通过关键字搜索用户，支持昵称、手机号、用户ID
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        keyword   query     string  true   "搜索关键字"
// @Param        page      query     int     false  "页码"         default(1)
// @Param        pageSize  query     int     false  "每页数量"      default(20)
// @Success      200       {object}  response.Response{data=UserSearchResult}  "搜索成功"
// @Failure      400       {object}  response.Response  "参数错误"
// @Failure      401       {object}  response.Response  "未授权"
// @Failure      500       {object}  response.Response  "服务器错误"
// @Router       /users/search [get]
func (h *UserHandler) SearchUsers(c *gin.Context) {
    keyword := c.Query("keyword")
    // ...
}
```

## 生成文档

### 命令行生成

```bash
# 生成文档
mage docs:generate

# 验证文档
mage docs:validate

# 本地预览
mage docs:serve
```

### 自动化

文档会在以下情况自动生成：
- CI/CD 流程中
- 提交代码前的 Git Hook（可选）

## 最佳实践

### 1. 保持注释简洁准确

- Summary 一行概括功能
- Description 补充必要细节
- 避免冗余信息

### 2. 提供有意义的示例

```go
type LoginRequest struct {
    Account  string `json:"account" example:"13800138000"`  // 好
    Password string `json:"password" example:"string"`      // 不好，太模糊
}
```

### 3. 明确标注认证要求

需要认证的接口一定要加 `@Security BearerAuth`

### 4. 使用适当的 HTTP 状态码

- 200: 成功
- 201: 创建成功
- 400: 参数错误
- 401: 未认证
- 403: 无权限
- 404: 资源不存在
- 409: 冲突（如资源已存在）
- 500: 服务器错误

### 5. 统一错误响应格式

所有失败响应都使用 `response.Response` 类型

### 6. 数据模型复用

相同的数据结构定义一次，多处引用：

```go
type UserInfo struct {
    UserID   string `json:"userId" example:"user-123"`
    Nickname string `json:"nickname" example:"张三"`
    // ...
}

// 在多个响应中使用
type LoginResponse struct {
    Token string   `json:"token"`
    User  UserInfo `json:"user"`  // 复用
}
```

## 常见问题

### 文档没有更新？

重新生成文档：`mage docs:generate`

### 结构体没有显示？

确保：
1. 结构体是 exported（首字母大写）
2. 在 handler 注释中引用了该结构体
3. 使用了 `--parseDependency` 和 `--parseInternal` 参数

### 如何隐藏内部接口？

不要为内部接口添加 Swagger 注释。

### 如何添加更多示例？

可以在 Description 中使用 Markdown 格式添加示例代码。

## 参考资料

- [Swag 官方文档](https://github.com/swaggo/swag)
- [OpenAPI 规范](https://swagger.io/specification/)
- [Gin Swagger 示例](https://github.com/swaggo/gin-swagger)
