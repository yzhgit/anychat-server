-- 创建用户资料表
CREATE TABLE IF NOT EXISTS user_profiles (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(36) UNIQUE NOT NULL,
    nickname VARCHAR(50) NOT NULL,
    avatar VARCHAR(500),
    signature VARCHAR(200),
    gender INT NOT NULL DEFAULT 0, -- 0-未知 1-男 2-女
    birthday DATE,
    region VARCHAR(100),
    qrcode_url VARCHAR(500),
    qrcode_updated_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_profiles_user_id ON user_profiles(user_id);
CREATE INDEX idx_user_profiles_nickname ON user_profiles(nickname);
