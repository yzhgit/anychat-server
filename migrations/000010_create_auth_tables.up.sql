-- 验证码记录表
CREATE TABLE IF NOT EXISTS verification_codes (
    id              BIGSERIAL    PRIMARY KEY,
    code_id         VARCHAR(64)  NOT NULL UNIQUE,
    target          VARCHAR(128) NOT NULL,
    target_type     VARCHAR(16)  NOT NULL,  -- sms/email
    code_hash       VARCHAR(128) NOT NULL,
    purpose         VARCHAR(32)  NOT NULL,  -- register/login/reset_password/bind_phone/change_phone/bind_email/change_email
    expires_at      TIMESTAMP    NOT NULL,
    verified_at     TIMESTAMP,
    status          VARCHAR(16)  NOT NULL DEFAULT 'pending',  -- pending/verified/expired/locked/cancelled
    send_ip         VARCHAR(64),
    send_device_id  VARCHAR(128),
    attempt_count   INT          NOT NULL DEFAULT 0,
    provider        VARCHAR(32),
    provider_message_id VARCHAR(128),
    created_at      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_verification_codes_target ON verification_codes (target, target_type, purpose);
CREATE INDEX idx_verification_codes_code_id ON verification_codes (code_id);
CREATE INDEX idx_verification_codes_expires_at ON verification_codes (expires_at);

-- 频率限制表
CREATE TABLE IF NOT EXISTS rate_limits (
    id              BIGSERIAL    PRIMARY KEY,
    target          VARCHAR(128) NOT NULL,
    target_type     VARCHAR(16)  NOT NULL,  -- sms/email
    action          VARCHAR(32)  NOT NULL,  -- send_code
    count           INT          NOT NULL DEFAULT 1,
    window_start    TIMESTAMP    NOT NULL,
    window_end      TIMESTAMP    NOT NULL,
    created_at      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(target, target_type, action, window_start)
);

CREATE INDEX idx_rate_limits_target ON rate_limits (target, target_type, action);

-- 验证码模板表
CREATE TABLE IF NOT EXISTS verification_templates (
    id              BIGSERIAL    PRIMARY KEY,
    purpose         VARCHAR(32)  NOT NULL UNIQUE,
    name            VARCHAR(64)  NOT NULL,
    sms_template_id VARCHAR(128),
    sms_content     VARCHAR(512),
    email_subject   VARCHAR(128),
    email_content   TEXT,
    is_active       BOOLEAN      NOT NULL DEFAULT true,
    created_at      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 插入默认模板
INSERT INTO verification_templates (purpose, name, sms_template_id, sms_content, email_subject, email_content) VALUES
('register', '注册', 'SMS_123456', '【AnyChat】您的验证码为 {code}，5分钟内有效，请勿泄露。', 'AnyChat 邮箱验证', '<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><div style="max-width: 600px; margin: 0 auto; padding: 20px;"><h2 style="color: #333;">AnyChat 邮箱验证</h2><p>您好，</p><p>您的邮箱验证码为：<strong style="font-size: 24px; color: #1890ff;">{code}</strong></p><p>验证码有效期为 5 分钟，请勿泄露给他人。</p><p style="color: #999; font-size: 12px;">如果这不是您的操作，请忽略此邮件。</p></div></body></html>'),
('login', '登录', 'SMS_123457', '【AnyChat】您的登录验证码为 {code}，5分钟内有效，请勿泄露。', 'AnyChat 登录验证码', '<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><div style="max-width: 600px; margin: 0 auto; padding: 20px;"><h2 style="color: #333;">AnyChat 登录验证</h2><p>您好，</p><p>您的登录验证码为：<strong style="font-size: 24px; color: #1890ff;">{code}</strong></p><p>验证码有效期为 5 分钟，请勿泄露给他人。</p></div></body></html>'),
('reset_password', '重置密码', 'SMS_123458', '【AnyChat】您正在重置密码，验证码为 {code}，5分钟内有效。', 'AnyChat 重置密码', '<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><div style="max-width: 600px; margin: 0 auto; padding: 20px;"><h2 style="color: #333;">AnyChat 重置密码</h2><p>您好，</p><p>您正在重置密码，验证码为：<strong style="font-size: 24px; color: #1890ff;">{code}</strong></p><p>验证码有效期为 5 分钟，请勿泄露给他人。</p></div></body></html>'),
('bind_phone', '绑定手机', 'SMS_123459', '【AnyChat】您正在绑定手机号，验证码为 {code}，5分钟内有效。', 'AnyChat 绑定手机', '<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><div style="max-width: 600px; margin: 0 auto; padding: 20px;"><h2 style="color: #333;">AnyChat 绑定手机</h2><p>您好，</p><p>您正在绑定手机号，验证码为：<strong style="font-size: 24px; color: #1890ff;">{code}</strong></p><p>验证码有效期为 5 分钟，请勿泄露给他人。</p></div></body></html>'),
('change_phone', '修改手机', 'SMS_123460', '【AnyChat】您正在修改手机号，验证码为 {code}，5分钟内有效。', 'AnyChat 修改手机', '<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><div style="max-width: 600px; margin: 0 auto; padding: 20px;"><h2 style="color: #333;">AnyChat 修改手机号</h2><p>您好，</p><p>您正在修改手机号，验证码为：<strong style="font-size: 24px; color: #1890ff;">{code}</strong></p><p>验证码有效期为 5 分钟，请勿泄露给他人。</p></div></body></html>'),
('bind_email', '绑定邮箱', 'SMS_123461', '【AnyChat】您正在绑定邮箱，验证码为 {code}，5分钟内有效。', 'AnyChat 绑定邮箱', '<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><div style="max-width: 600px; margin: 0 auto; padding: 20px;"><h2 style="color: #333;">AnyChat 绑定邮箱</h2><p>您好，</p><p>您正在绑定邮箱，验证码为：<strong style="font-size: 24px; color: #1890ff;">{code}</strong></p><p>验证码有效期为 5 分钟，请勿泄露给他人。</p></div></body></html>'),
('change_email', '修改邮箱', 'SMS_123462', '【AnyChat】您正在修改邮箱，验证码为 {code}，5分钟内有效。', 'AnyChat 修改邮箱', '<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><div style="max-width: 600px; margin: 0 auto; padding: 20px;"><h2 style="color: #333;">AnyChat 修改邮箱</h2><p>您好，</p><p>您正在修改邮箱，验证码为：<strong style="font-size: 24px; color: #1890ff;">{code}</strong></p><p>验证码有效期为 5 分钟，请勿泄露给他人。</p></div></body></html>');
