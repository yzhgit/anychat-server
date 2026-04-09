-- Create messages table (monthly sharding strategy, this table is the template)
CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    message_id VARCHAR(64) NOT NULL UNIQUE,
    conversation_id VARCHAR(64) NOT NULL,
    conversation_type VARCHAR(20) NOT NULL,  -- single/group
    sender_id VARCHAR(36) NOT NULL,
    content_type VARCHAR(20) NOT NULL,  -- text/image/video/audio/file/location/card
    content JSONB NOT NULL,
    sequence BIGINT NOT NULL,  -- Incrementing sequence number within conversation
    reply_to VARCHAR(64),  -- Message ID being replied to
    at_users TEXT[],  -- List of user IDs being @ed
    status SMALLINT DEFAULT 0,  -- 0-normal, 1-recalled, 2-deleted
    burn_after_reading_seconds INT NOT NULL DEFAULT 0,  -- Burn-after-reading duration in seconds, 0 means disabled
    auto_delete_expire_time TIMESTAMPTZ,  -- Expiration time calculated from auto-delete policy
    burn_after_reading_expire_time TIMESTAMPTZ,  -- Expiration time calculated from burn-after-reading policy
    expire_time TIMESTAMPTZ,  -- Message expiration time, NULL means never expires
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_conversation_sequence UNIQUE (conversation_id, sequence)
);

-- Create indexes
CREATE INDEX idx_messages_conversation_sequence ON messages(conversation_id, sequence DESC);
CREATE INDEX idx_messages_sender_time ON messages(sender_id, created_at DESC);
CREATE INDEX idx_messages_message_id ON messages(message_id);
CREATE INDEX idx_messages_status ON messages(status) WHERE status = 0;
CREATE INDEX idx_messages_created_at ON messages(created_at DESC);
CREATE INDEX idx_messages_expire_time ON messages(expire_time) WHERE expire_time IS NOT NULL;

-- Create read receipts table
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

-- Create conversation sequence table (for generating incrementing sequence numbers)
CREATE TABLE IF NOT EXISTS conversation_sequences (
    id BIGSERIAL PRIMARY KEY,
    conversation_id VARCHAR(64) NOT NULL UNIQUE,
    current_seq BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_conversation_sequences_conversation ON conversation_sequences(conversation_id);

-- Create message references table (for recording reference relationships between messages)
CREATE TABLE IF NOT EXISTS message_references (
    id BIGSERIAL PRIMARY KEY,
    message_id VARCHAR(64) NOT NULL,
    reply_to_message_id VARCHAR(64) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_message_reply UNIQUE (message_id, reply_to_message_id)
);

CREATE INDEX idx_message_references_message ON message_references(message_id);
CREATE INDEX idx_message_references_reply_to ON message_references(reply_to_message_id);