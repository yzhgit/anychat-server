# Gateway API 文档

AnyChat Gateway Service HTTP API 接口文档

## 基础信息

- **Base URL**: `http://localhost:8080`
- **API Version**: v1
- **认证方式**: JWT Bearer Token

## 认证说明

需要认证的接口需要在请求头中携带 JWT Token：

```
Authorization: Bearer <access_token>
```

## 接口列表

### 1. 认证相关接口

#### 1.1 用户注册

**接口地址**: `POST /api/v1/auth/register`

**请求参数**:

```json
{
  "phoneNumber": "13800138000",  // 手机号（与email二选一）
  "email": "user@example.com",   // 邮箱（与phoneNumber二选一）
  "password": "password123",     // 密码，8-32位
  "verifyCode": "123456",        // 验证码
  "nickname": "昵称",            // 可选
  "deviceType": "iOS",           // 设备类型：iOS/Android/Web
  "deviceId": "device-uuid"      // 设备唯一标识
}
```

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "userId": "user-123",
    "accessToken": "eyJhbGciOiJIUzI1NiIs...",
    "refreshToken": "eyJhbGciOiJIUzI1NiIs...",
    "expiresIn": 7200
  }
}
```

#### 1.2 用户登录

**接口地址**: `POST /api/v1/auth/login`

**请求参数**:

```json
{
  "account": "13800138000",      // 手机号或邮箱
  "password": "password123",     // 密码
  "deviceType": "iOS",           // 设备类型
  "deviceId": "device-uuid"      // 设备唯一标识
}
```

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "userId": "user-123",
    "accessToken": "eyJhbGciOiJIUzI1NiIs...",
    "refreshToken": "eyJhbGciOiJIUzI1NiIs...",
    "expiresIn": 7200,
    "user": {
      "userId": "user-123",
      "nickname": "昵称",
      "avatar": "https://example.com/avatar.jpg",
      "phoneNumber": "13800138000"
    }
  }
}
```

#### 1.3 刷新Token

**接口地址**: `POST /api/v1/auth/refresh`

**请求头**:

```
Authorization: Bearer <refresh_token>
```

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIs...",
    "refreshToken": "eyJhbGciOiJIUzI1NiIs...",
    "expiresIn": 7200
  }
}
```

#### 1.4 用户登出

**接口地址**: `POST /api/v1/auth/logout`

**需要认证**: 是

**请求参数**:

```json
{
  "deviceId": "device-uuid"      // 设备唯一标识
}
```

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

#### 1.5 修改密码

**接口地址**: `POST /api/v1/auth/password/change`

**需要认证**: 是

**请求参数**:

```json
{
  "oldPassword": "old_password",
  "newPassword": "new_password"  // 8-32位
}
```

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

### 2. 用户相关接口

#### 2.1 获取个人资料

**接口地址**: `GET /api/v1/users/me`

**需要认证**: 是

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "userId": "user-123",
    "nickname": "昵称",
    "avatar": "https://example.com/avatar.jpg",
    "signature": "个性签名",
    "gender": 1,               // 0:未知 1:男 2:女
    "birthday": "1990-01-01",
    "region": "中国-北京",
    "phone": "13800138000",
    "email": "user@example.com",
    "qrcodeUrl": "https://example.com/qrcode/xxx",
    "createdAt": "2024-01-01T00:00:00Z"
  }
}
```

#### 2.2 更新个人资料

**接口地址**: `PUT /api/v1/users/me`

**需要认证**: 是

**请求参数**:

```json
{
  "nickname": "新昵称",           // 可选
  "avatar": "https://...",      // 可选
  "signature": "新签名",         // 可选
  "gender": 1,                  // 可选
  "birthday": "1990-01-01",     // 可选
  "region": "中国-北京"          // 可选
}
```

**响应示例**: 同获取个人资料

#### 2.3 获取用户信息

**接口地址**: `GET /api/v1/users/:userId`

**需要认证**: 是

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "userId": "user-456",
    "nickname": "昵称",
    "avatar": "https://example.com/avatar.jpg",
    "signature": "个性签名",
    "gender": 1,
    "region": "中国-上海",
    "isFriend": false,
    "isBlocked": false
  }
}
```

#### 2.4 搜索用户

**接口地址**: `GET /api/v1/users/search`

**需要认证**: 是

**请求参数**:

```
?keyword=昵称或ID
&page=1
&pageSize=20
```

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 100,
    "users": [
      {
        "userId": "user-789",
        "nickname": "昵称",
        "avatar": "https://example.com/avatar.jpg",
        "signature": "个性签名"
      }
    ]
  }
}
```

#### 2.5 获取用户设置

**接口地址**: `GET /api/v1/users/me/settings`

**需要认证**: 是

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "userId": "user-123",
    "notificationEnabled": true,
    "soundEnabled": true,
    "vibrationEnabled": true,
    "messagePreviewEnabled": true,
    "friendVerifyRequired": true,
    "searchByPhone": true,
    "searchById": true,
    "language": "zh-CN"
  }
}
```

#### 2.6 更新用户设置

**接口地址**: `PUT /api/v1/users/me/settings`

**需要认证**: 是

**请求参数**: 同获取用户设置的字段（除userId外均可选）

**响应示例**: 同获取用户设置

#### 2.7 刷新二维码

**接口地址**: `POST /api/v1/users/me/qrcode/refresh`

**需要认证**: 是

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "qrcodeUrl": "https://example.com/qrcode/xxx",
    "expiresAt": "2024-01-02T00:00:00Z"
  }
}
```

#### 2.8 通过二维码获取用户

**接口地址**: `GET /api/v1/users/qrcode`

**需要认证**: 是

**请求参数**:

```
?code=qrcode-string
```

**响应示例**: 同获取用户信息

#### 2.9 更新推送Token

**接口地址**: `POST /api/v1/users/me/push-token`

**需要认证**: 是

**请求参数**:

```json
{
  "deviceId": "device-uuid",
  "pushToken": "firebase-token-xxx",
  "platform": "iOS"           // iOS/Android
}
```

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

### 3. 系统接口

#### 3.1 健康检查

**接口地址**: `GET /health`

**需要认证**: 否

**响应示例**:

```json
{
  "status": "ok",
  "service": "gateway-service"
}
```

## 错误码说明

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未授权/Token无效 |
| 403 | 禁止访问 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |
| 1001 | 用户不存在 |
| 1002 | 密码错误 |
| 1003 | 用户已存在 |
| 1004 | 验证码错误 |
| 1005 | Token已过期 |

## 响应格式

所有接口统一返回格式：

```json
{
  "code": 0,           // 错误码，0表示成功
  "message": "success", // 消息
  "data": {}           // 业务数据
}
```

## 示例代码

### cURL

```bash
# 登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "account": "13800138000",
    "password": "password123",
    "deviceType": "iOS",
    "deviceId": "device-001"
  }'

# 获取个人资料（需要Token）
curl -X GET http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer <access_token>"
```

### JavaScript (Fetch)

```javascript
// 登录
const response = await fetch('http://localhost:8080/api/v1/auth/login', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    account: '13800138000',
    password: 'password123',
    deviceType: 'Web',
    deviceId: 'browser-uuid'
  })
});

const data = await response.json();
const accessToken = data.data.accessToken;

// 获取个人资料
const profileResponse = await fetch('http://localhost:8080/api/v1/users/me', {
  headers: {
    'Authorization': `Bearer ${accessToken}`
  }
});
```
