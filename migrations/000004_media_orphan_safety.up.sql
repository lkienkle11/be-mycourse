ALTER TABLE media_files ADD COLUMN IF NOT EXISTS row_version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE media_files ADD COLUMN IF NOT EXISTS content_fingerprint VARCHAR(128) NOT NULL DEFAULT '';

CREATE TABLE media_pending_cloud_cleanup (
    id BIGSERIAL PRIMARY KEY,
    provider VARCHAR(16) NOT NULL,
    object_key VARCHAR(512) NOT NULL DEFAULT '',
    bunny_video_id VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    attempt_count INT NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    next_run_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_media_pending_cloud_cleanup_due ON media_pending_cloud_cleanup (next_run_at) WHERE status = 'pending';
CREATE INDEX idx_media_pending_cloud_cleanup_status ON media_pending_cloud_cleanup (status);
