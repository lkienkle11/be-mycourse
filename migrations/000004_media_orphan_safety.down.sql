DROP TABLE IF EXISTS media_pending_cloud_cleanup;

ALTER TABLE media_files DROP COLUMN IF EXISTS content_fingerprint;
ALTER TABLE media_files DROP COLUMN IF EXISTS row_version;
