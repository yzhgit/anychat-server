-- 创建用户设置表
CREATE TABLE IF NOT EXISTS user_settings (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(36) UNIQUE NOT NULL,
    notification_enabled BOOLEAN NOT NULL DEFAULT true,
    sound_enabled BOOLEAN NOT NULL DEFAULT true,
    vibration_enabled BOOLEAN NOT NULL DEFAULT true,
    message_preview_enabled BOOLEAN NOT NULL DEFAULT true,
    friend_verify_required BOOLEAN NOT NULL DEFAULT true,
    search_by_phone BOOLEAN NOT NULL DEFAULT true,
    search_by_id BOOLEAN NOT NULL DEFAULT true,
    language VARCHAR(10) NOT NULL DEFAULT 'zh_CN',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_settings_user_id ON user_settings(user_id);
