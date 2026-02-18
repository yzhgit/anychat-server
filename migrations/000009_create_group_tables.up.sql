-- 创建群组表
CREATE TABLE IF NOT EXISTS groups (
    id BIGSERIAL PRIMARY KEY,
    group_id VARCHAR(36) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    avatar VARCHAR(255),
    announcement TEXT,
    owner_id VARCHAR(36) NOT NULL,
    member_count INT DEFAULT 0,
    max_members INT DEFAULT 500,
    join_verify BOOLEAN DEFAULT TRUE,
    is_muted BOOLEAN DEFAULT FALSE,
    status SMALLINT DEFAULT 1,  -- 0-已解散 1-正常
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_groups_owner_id ON groups(owner_id);
CREATE INDEX idx_groups_status ON groups(status);
CREATE INDEX idx_groups_created_at ON groups(created_at);

-- 创建群成员表
CREATE TABLE IF NOT EXISTS group_members (
    id BIGSERIAL PRIMARY KEY,
    group_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    group_nickname VARCHAR(50),
    role VARCHAR(20) DEFAULT 'member',  -- owner/admin/member
    is_muted BOOLEAN DEFAULT FALSE,
    joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_group_user UNIQUE (group_id, user_id)
);

CREATE INDEX idx_group_members_user_id ON group_members(user_id);
CREATE INDEX idx_group_members_role ON group_members(role);
CREATE INDEX idx_group_members_joined_at ON group_members(joined_at);

-- 创建群组设置表
CREATE TABLE IF NOT EXISTS group_settings (
    group_id VARCHAR(36) PRIMARY KEY,
    allow_member_invite BOOLEAN DEFAULT TRUE,
    allow_view_history BOOLEAN DEFAULT TRUE,
    allow_add_friend BOOLEAN DEFAULT TRUE,
    show_member_nickname BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建入群申请表
CREATE TABLE IF NOT EXISTS group_join_requests (
    id BIGSERIAL PRIMARY KEY,
    group_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    inviter_id VARCHAR(36),  -- 邀请人ID（NULL表示主动申请）
    message VARCHAR(200),
    status VARCHAR(20) DEFAULT 'pending',  -- pending/accepted/rejected
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_group_join_requests_group_id ON group_join_requests(group_id);
CREATE INDEX idx_group_join_requests_user_id ON group_join_requests(user_id);
CREATE INDEX idx_group_join_requests_status ON group_join_requests(status);
