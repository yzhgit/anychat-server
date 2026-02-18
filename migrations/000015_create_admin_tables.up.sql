-- 管理员用户表
CREATE TABLE IF NOT EXISTS admin_users (
    id            VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    username      VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email         VARCHAR(100),
    role          VARCHAR(20) NOT NULL DEFAULT 'admin', -- superadmin, admin, readonly
    status        SMALLINT NOT NULL DEFAULT 1,          -- 1=active, 0=disabled
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 审计日志表
CREATE TABLE IF NOT EXISTS audit_logs (
    id            BIGSERIAL PRIMARY KEY,
    admin_id      VARCHAR(36),
    action        VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id   VARCHAR(100),
    details       JSONB,
    ip_address    VARCHAR(50),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_admin_id   ON audit_logs(admin_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action     ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);

-- 系统配置表
CREATE TABLE IF NOT EXISTS system_configs (
    key         VARCHAR(100) PRIMARY KEY,
    value       TEXT NOT NULL DEFAULT '',
    description VARCHAR(255),
    updated_by  VARCHAR(36),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 初始管理员账户 (username: admin, password: Admin@123456)
INSERT INTO admin_users (username, password_hash, role)
VALUES ('admin', '$2a$10$AN7p9wUwBMIHWhFsw6h8ouO41ZtoK7XG9ddCZSqQH6pcTZ12v1tPu', 'superadmin')
ON CONFLICT (username) DO NOTHING;

-- 初始系统配置
INSERT INTO system_configs (key, value, description) VALUES
    ('site.name',           'AnyChat', '网站名称'),
    ('site.description',    'AnyChat即时通讯系统', '网站描述'),
    ('user.max_friends',    '1000', '最大好友数'),
    ('group.max_members',   '500', '群组最大成员数'),
    ('message.max_recall',  '120', '消息可撤回时间(秒)')
ON CONFLICT (key) DO NOTHING;
