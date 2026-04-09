-- Conversations table
CREATE TABLE conversations (
    conversation_id      VARCHAR(100) PRIMARY KEY,
    conversation_type    VARCHAR(20)  NOT NULL,                -- single/group/system
    user_id         VARCHAR(100) NOT NULL,
    target_id       VARCHAR(100) NOT NULL,                -- For private chat: peer user ID, for group chat: group ID
    last_message_id VARCHAR(100),
    last_message_content TEXT,
    last_message_time    TIMESTAMPTZ,
    unread_count    INT          NOT NULL DEFAULT 0,
    is_pinned       BOOLEAN      NOT NULL DEFAULT FALSE,
    is_muted        BOOLEAN      NOT NULL DEFAULT FALSE,
    pin_time        TIMESTAMPTZ,
    burn_after_reading INT       NOT NULL DEFAULT 0,       -- Burn-after-reading duration in seconds, 0 means disabled
    auto_delete_duration INT     NOT NULL DEFAULT 0,       -- Auto-delete duration in seconds, 0 means disabled
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX uk_conversation_user_target ON conversations (user_id, conversation_type, target_id);
CREATE INDEX idx_conversations_user_id      ON conversations (user_id);
CREATE INDEX idx_conversations_updated_at   ON conversations (updated_at);

-- Message send idempotency table
CREATE TABLE IF NOT EXISTS message_send_idempotency (
    id BIGSERIAL PRIMARY KEY,
    sender_id VARCHAR(36) NOT NULL,
    conversation_id VARCHAR(64) NOT NULL,
    local_id VARCHAR(128) NOT NULL,
    message_id VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_sender_conversation_local UNIQUE (sender_id, conversation_id, local_id)
);

CREATE INDEX idx_message_idempotency_message_id ON message_send_idempotency(message_id);