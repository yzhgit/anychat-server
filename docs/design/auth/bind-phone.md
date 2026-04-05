# 绑定手机号设计

## 1. 概述

绑定手机号功能允许用户将手机号绑定到账号，用于登录、找回密码等场景。首次绑定手机号需要进行验证码验证。

## 2. 功能列表

- [x] 绑定手机号（需验证码）
- [x] 绑定前检查手机号是否已被占用

## 3. 业务流程

```mermaid
sequenceDiagram
    participant Client
    participant UserService
    participant AuthService
    participant VerifyService
    participant DB

    Client->>UserService: 绑定手机号(phoneNumber, verifyCode)
    UserService->>UserService: 检查手机号格式
    UserService->>UserService: 验证用户是否已绑定该手机号
    UserService->>AuthService: 验证验证码(phoneNumber, verifyCode, bind_phone)
    AuthService->>VerifyService: VerifyCode
    VerifyService-->>AuthService: 验证成功
    AuthService-->>UserService: 验证成功
    UserService->>DB: 更新用户手机号
    DB-->>UserService: 成功
    UserService-->>Client: 绑定成功
```

## 4. 验证规则

| 字段 | 规则 |
|------|------|
| 手机号 | 11位数字，以1开头 |
| 验证码 | 6位数字，有效期5分钟 |
| 绑定检查 | 手机号未被其他用户占用 |

## 5. API设计

### 5.1 请求

```protobuf
message BindPhoneRequest {
    string user_id = 1;
    string phone_number = 2;
    string verify_code = 3;
}
```

### 5.2 响应

```protobuf
message BindPhoneResponse {
    string phone_number = 1;
    bool is_primary = 2;  // 是否设为主要联系方式
}
```

### 5.3 错误码

| 错误码 | 说明 |
|--------|------|
| 10206 | 验证码错误 |
| 10207 | 验证码已过期 |
| 20106 | 手机号格式错误 |
| 20107 | 手机号已被占用 |
| 10104 | 用户不存在 |

## 6. 安全考虑

1. **验证码一次性使用**：验证成功后立即失效
2. **手机号唯一性**：同一手机号只能绑定一个账号
3. **脱敏显示**：API返回时对手机号进行脱敏处理（如 138****8000）
4. **绑定日志**：记录绑定操作日志供审计

## 7. 依赖服务

- **Auth Service**: 验证码验证
- **Verify Service**: 验证码校验与消费
- **PostgreSQL**: 用户信息持久化

---

返回: [认证服务总体设计](../auth/README.md)
