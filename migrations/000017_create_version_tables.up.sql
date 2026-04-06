-- Create app_versions table
CREATE TABLE IF NOT EXISTS app_versions (
    id BIGSERIAL PRIMARY KEY,
    platform VARCHAR(20) NOT NULL,
    version VARCHAR(50) NOT NULL,
    build_number INTEGER DEFAULT 0,
    version_code INTEGER,
    min_version VARCHAR(50),
    min_build_number INTEGER,
    force_update BOOLEAN DEFAULT FALSE,
    release_type VARCHAR(20) DEFAULT 'stable',
    title VARCHAR(200),
    content TEXT,
    download_url VARCHAR(500),
    file_size BIGINT,
    file_hash VARCHAR(64),
    published_at TIMESTAMP,
    status VARCHAR(20) DEFAULT 'draft',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT uk_app_versions_platform_version_release UNIQUE (platform, version, release_type)
);

CREATE INDEX idx_app_versions_platform ON app_versions(platform);
CREATE INDEX idx_app_versions_published ON app_versions(published_at DESC);
CREATE INDEX idx_app_versions_status ON app_versions(status);

-- Create client_version_stats table
CREATE TABLE IF NOT EXISTS client_version_stats (
    id BIGSERIAL PRIMARY KEY,
    platform VARCHAR(20) NOT NULL,
    version VARCHAR(50) NOT NULL,
    count INTEGER DEFAULT 0,
    report_date DATE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT uk_client_version_stats_platform_version_date UNIQUE (platform, version, report_date)
);

CREATE INDEX idx_client_version_stats_date ON client_version_stats(report_date);