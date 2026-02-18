-- 创建推送日志表
CREATE TABLE IF NOT EXISTS push_logs (
    id          BIGSERIAL PRIMARY KEY,
    user_id     VARCHAR(36)  NOT NULL,
    push_type   VARCHAR(50)  NOT NULL,
    title       VARCHAR(200),
    content     VARCHAR(1000),
    target_count INT         NOT NULL DEFAULT 0,
    success_count INT        NOT NULL DEFAULT 0,
    failure_count INT        NOT NULL DEFAULT 0,
    jpush_msg_id VARCHAR(100),
    status      VARCHAR(20)  NOT NULL DEFAULT 'pending',  -- pending/sent/failed
    error_msg   TEXT,
    created_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_push_logs_user_id    ON push_logs (user_id);
CREATE INDEX idx_push_logs_created_at ON push_logs (created_at DESC);
CREATE INDEX idx_push_logs_status     ON push_logs (status);
