CREATE TABLE media_files (
    id UUID PRIMARY KEY,
    object_key VARCHAR(512) NOT NULL UNIQUE,
    kind VARCHAR(16) NOT NULL,
    provider VARCHAR(16) NOT NULL,
    filename VARCHAR(512) NOT NULL,
    mime_type VARCHAR(255) NOT NULL DEFAULT '',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    url TEXT NOT NULL,
    origin_url TEXT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'READY',
    b2_bucket_name VARCHAR(255) NOT NULL DEFAULT '',
    bunny_video_id VARCHAR(255),
    bunny_library_id VARCHAR(255) NOT NULL DEFAULT '',
    duration BIGINT NOT NULL DEFAULT 0,
    video_provider VARCHAR(64) NOT NULL DEFAULT '',
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_media_files_kind ON media_files(kind);
CREATE INDEX idx_media_files_provider ON media_files(provider);
CREATE INDEX idx_media_files_bunny_video_id ON media_files(bunny_video_id);
CREATE INDEX idx_media_files_created_at ON media_files(created_at);
