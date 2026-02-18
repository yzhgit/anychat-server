-- 会话表
CREATE TABLE sessions (
    session_id      VARCHAR(100) PRIMARY KEY,
    session_type    VARCHAR(20)  NOT NULL,                -- single/group/system
    user_id         VARCHAR(100) NOT NULL,
    target_id       VARCHAR(100) NOT NULL,                -- 单聊为对方用户ID，群聊为群ID
    last_message_id VARCHAR(100),
    last_message_content TEXT,
    last_message_time    TIMESTAMPTZ,
    unread_count    INT          NOT NULL DEFAULT 0,
    is_pinned       BOOLEAN      NOT NULL DEFAULT FALSE,
    is_muted        BOOLEAN      NOT NULL DEFAULT FALSE,
    pin_time        TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX uk_session_user_target ON sessions (user_id, session_type, target_id);
CREATE INDEX idx_sessions_user_id      ON sessions (user_id);
CREATE INDEX idx_sessions_updated_at   ON sessions (updated_at);
