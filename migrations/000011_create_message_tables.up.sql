-- 创建消息表（按月分表策略，此表为模板）
CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    message_id VARCHAR(64) NOT NULL UNIQUE,
    conversation_id VARCHAR(64) NOT NULL,
    conversation_type VARCHAR(20) NOT NULL,  -- single/group
    sender_id VARCHAR(36) NOT NULL,
    content_type VARCHAR(20) NOT NULL,  -- text/image/video/audio/file/location/card
    content JSONB NOT NULL,
    sequence BIGINT NOT NULL,  -- 会话内递增序列号
    reply_to VARCHAR(64),  -- 回复的消息ID
    at_users TEXT[],  -- @的用户ID列表
    status SMALLINT DEFAULT 0,  -- 0-正常 1-撤回 2-删除
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_conversation_sequence UNIQUE (conversation_id, sequence)
);

-- 创建索引
CREATE INDEX idx_messages_conversation_sequence ON messages(conversation_id, sequence DESC);
CREATE INDEX idx_messages_sender_time ON messages(sender_id, created_at DESC);
CREATE INDEX idx_messages_message_id ON messages(message_id);
CREATE INDEX idx_messages_status ON messages(status) WHERE status = 0;
CREATE INDEX idx_messages_created_at ON messages(created_at DESC);

-- 创建已读回执表
CREATE TABLE IF NOT EXISTS message_read_receipts (
    id BIGSERIAL PRIMARY KEY,
    conversation_id VARCHAR(64) NOT NULL,
    conversation_type VARCHAR(20) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    last_read_seq BIGINT NOT NULL,
    last_read_message_id VARCHAR(64),
    read_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_conversation_user UNIQUE (conversation_id, user_id)
);

CREATE INDEX idx_read_receipts_conversation ON message_read_receipts(conversation_id);
CREATE INDEX idx_read_receipts_user ON message_read_receipts(user_id);

-- 创建会话序列号表（用于生成递增序列号）
CREATE TABLE IF NOT EXISTS conversation_sequences (
    id BIGSERIAL PRIMARY KEY,
    conversation_id VARCHAR(64) NOT NULL UNIQUE,
    current_seq BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_conversation_sequences_conversation ON conversation_sequences(conversation_id);

-- 创建消息引用表（用于记录消息之间的引用关系）
CREATE TABLE IF NOT EXISTS message_references (
    id BIGSERIAL PRIMARY KEY,
    message_id VARCHAR(64) NOT NULL,
    reply_to_message_id VARCHAR(64) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_message_reply UNIQUE (message_id, reply_to_message_id)
);

CREATE INDEX idx_message_references_message ON message_references(message_id);
CREATE INDEX idx_message_references_reply_to ON message_references(reply_to_message_id);
