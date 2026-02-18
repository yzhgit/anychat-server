-- 创建好友关系表
CREATE TABLE IF NOT EXISTS friendships (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    friend_id VARCHAR(36) NOT NULL,
    remark VARCHAR(50),
    status SMALLINT DEFAULT 1,  -- 0-已删除 1-正常
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_user_friend UNIQUE (user_id, friend_id)
);

CREATE INDEX idx_friendships_user_id ON friendships(user_id) WHERE status = 1;
CREATE INDEX idx_friendships_friend_id ON friendships(friend_id) WHERE status = 1;
CREATE INDEX idx_friendships_updated_at ON friendships(updated_at);

-- 创建好友申请表
CREATE TABLE IF NOT EXISTS friend_requests (
    id BIGSERIAL PRIMARY KEY,
    from_user_id VARCHAR(36) NOT NULL,
    to_user_id VARCHAR(36) NOT NULL,
    message VARCHAR(200),
    source VARCHAR(20),  -- search/qrcode/group/contacts
    status VARCHAR(20) DEFAULT 'pending',  -- pending/accepted/rejected/expired
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_friend_requests_to_user ON friend_requests(to_user_id, status);
CREATE INDEX idx_friend_requests_from_user ON friend_requests(from_user_id);
CREATE INDEX idx_friend_requests_created_at ON friend_requests(created_at);

-- 创建黑名单表
CREATE TABLE IF NOT EXISTS blacklists (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    blocked_user_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_user_blocked UNIQUE (user_id, blocked_user_id)
);

CREATE INDEX idx_blacklists_user_id ON blacklists(user_id);
CREATE INDEX idx_blacklists_blocked_user ON blacklists(blocked_user_id);
