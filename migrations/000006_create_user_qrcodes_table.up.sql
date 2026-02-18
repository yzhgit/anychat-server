-- 创建用户二维码表
CREATE TABLE IF NOT EXISTS user_qrcodes (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    qrcode_token VARCHAR(100) UNIQUE NOT NULL,
    qrcode_url VARCHAR(500) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_qrcodes_user_id ON user_qrcodes(user_id);
CREATE INDEX idx_user_qrcodes_token ON user_qrcodes(qrcode_token);
CREATE INDEX idx_user_qrcodes_expires_at ON user_qrcodes(expires_at);
