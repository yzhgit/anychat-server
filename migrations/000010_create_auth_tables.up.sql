-- Verification codes table
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

-- Rate limits table
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

-- Verification templates table
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

-- Insert default templates
INSERT INTO verification_templates (purpose, name, sms_template_id, sms_content, email_subject, email_content) VALUES
('register', 'Register', 'SMS_123456', '【AnyChat】Your verification code is {code}, valid for 5 minutes. Do not share.', 'AnyChat Email Verification', '<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><div style="max-width: 600px; margin: 0 auto; padding: 20px;"><h2 style="color: #333;">AnyChat Email Verification</h2><p>Hello,</p><p>Your email verification code is: <strong style="font-size: 24px; color: #1890ff;">{code}</strong></p><p>The code is valid for 5 minutes, please do not share with others.</p><p style="color: #999; font-size: 12px;">If this was not your operation, please ignore this email.</p></div></body></html>'),
('login', 'Login', 'SMS_123457', '【AnyChat】Your login verification code is {code}, valid for 5 minutes. Do not share.', 'AnyChat Login Verification', '<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><div style="max-width: 600px; margin: 0 auto; padding: 20px;"><h2 style="color: #333;">AnyChat Login Verification</h2><p>Hello,</p><p>Your login verification code is: <strong style="font-size: 24px; color: #1890ff;">{code}</strong></p><p>The code is valid for 5 minutes, please do not share with others.</p></div></body></html>'),
('reset_password', 'Reset Password', 'SMS_123458', '【AnyChat】You are resetting your password, verification code is {code}, valid for 5 minutes.', 'AnyChat Reset Password', '<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><div style="max-width: 600px; margin: 0 auto; padding: 20px;"><h2 style="color: #333;">AnyChat Reset Password</h2><p>Hello,</p><p>You are resetting your password, verification code is: <strong style="font-size: 24px; color: #1890ff;">{code}</strong></p><p>The code is valid for 5 minutes, please do not share with others.</p></div></body></html>'),
('bind_phone', 'Bind Phone', 'SMS_123459', '【AnyChat】You are binding your phone number, verification code is {code}, valid for 5 minutes.', 'AnyChat Bind Phone', '<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><div style="max-width: 600px; margin: 0 auto; padding: 20px;"><h2 style="color: #333;">AnyChat Bind Phone</h2><p>Hello,</p><p>You are binding your phone number, verification code is: <strong style="font-size: 24px; color: #1890ff;">{code}</strong></p><p>The code is valid for 5 minutes, please do not share with others.</p></div></body></html>'),
('change_phone', 'Change Phone', 'SMS_123460', '【AnyChat】You are changing your phone number, verification code is {code}, valid for 5 minutes.', 'AnyChat Change Phone', '<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><div style="max-width: 600px; margin: 0 auto; padding: 20px;"><h2 style="color: #333;">AnyChat Change Phone Number</h2><p>Hello,</p><p>You are changing your phone number, verification code is: <strong style="font-size: 24px; color: #1890ff;">{code}</strong></p><p>The code is valid for 5 minutes, please do not share with others.</p></div></body></html>'),
('bind_email', 'Bind Email', 'SMS_123461', '【AnyChat】You are binding your email, verification code is {code}, valid for 5 minutes.', 'AnyChat Bind Email', '<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><div style="max-width: 600px; margin: 0 auto; padding: 20px;"><h2 style="color: #333;">AnyChat Bind Email</h2><p>Hello,</p><p>You are binding your email, verification code is: <strong style="font-size: 24px; color: #1890ff;">{code}</strong></p><p>The code is valid for 5 minutes, please do not share with others.</p></div></body></html>'),
('change_email', 'Change Email', 'SMS_123462', '【AnyChat】You are changing your email, verification code is {code}, valid for 5 minutes.', 'AnyChat Change Email', '<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><div style="max-width: 600px; margin: 0 auto; padding: 20px;"><h2 style="color: #333;">AnyChat Change Email</h2><p>Hello,</p><p>You are changing your email, verification code is: <strong style="font-size: 24px; color: #1890ff;">{code}</strong></p><p>The code is valid for 5 minutes, please do not share with others.</p></div></body></html>');