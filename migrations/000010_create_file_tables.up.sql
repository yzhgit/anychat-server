-- 文件元数据表
CREATE TABLE IF NOT EXISTS files (
    id BIGSERIAL PRIMARY KEY,
    file_id VARCHAR(64) NOT NULL UNIQUE,
    user_id VARCHAR(36) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(20) NOT NULL,  -- image/video/audio/file
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    storage_path VARCHAR(500) NOT NULL,  -- MinIO object key
    thumbnail_path VARCHAR(500),
    bucket_name VARCHAR(50) NOT NULL,
    status SMALLINT DEFAULT 1,  -- 0-deleted 1-active 2-processing
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    metadata JSONB
);

-- 索引
CREATE INDEX idx_files_file_id ON files(file_id) WHERE status = 1;
CREATE INDEX idx_files_user_id ON files(user_id) WHERE status = 1;
CREATE INDEX idx_files_created_at ON files(created_at);
CREATE INDEX idx_files_expires_at ON files(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_files_bucket ON files(bucket_name);
CREATE INDEX idx_files_status ON files(status);

-- 文件上传追踪表（用于分片上传，Phase 2）
CREATE TABLE IF NOT EXISTS file_uploads (
    id BIGSERIAL PRIMARY KEY,
    upload_id VARCHAR(64) NOT NULL UNIQUE,
    user_id VARCHAR(36) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    chunk_size BIGINT NOT NULL,
    uploaded_chunks JSONB,
    status VARCHAR(20) DEFAULT 'pending',  -- pending/uploading/completed/failed
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

-- 索引
CREATE INDEX idx_file_uploads_upload_id ON file_uploads(upload_id);
CREATE INDEX idx_file_uploads_user_id ON file_uploads(user_id);
CREATE INDEX idx_file_uploads_status ON file_uploads(status);
CREATE INDEX idx_file_uploads_expires_at ON file_uploads(expires_at);

-- 添加注释
COMMENT ON TABLE files IS '文件元数据表';
COMMENT ON COLUMN files.file_id IS '文件唯一标识';
COMMENT ON COLUMN files.user_id IS '上传用户ID';
COMMENT ON COLUMN files.file_type IS '文件类型：image/video/audio/file';
COMMENT ON COLUMN files.status IS '状态：0-已删除 1-激活 2-处理中';
COMMENT ON COLUMN files.storage_path IS 'MinIO存储路径';
COMMENT ON COLUMN files.thumbnail_path IS '缩略图路径';
COMMENT ON COLUMN files.metadata IS '扩展元数据（宽度、高度、时长等）';

COMMENT ON TABLE file_uploads IS '文件分片上传追踪表';
COMMENT ON COLUMN file_uploads.upload_id IS '上传会话ID';
COMMENT ON COLUMN file_uploads.uploaded_chunks IS '已上传分片信息';
