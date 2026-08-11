-- Add VOD processing columns to streams table
ALTER TABLE streams ADD COLUMN IF NOT EXISTS vod_hls_path TEXT;
ALTER TABLE streams ADD COLUMN IF NOT EXISTS vod_thumbnail_path TEXT;
ALTER TABLE streams ADD COLUMN IF NOT EXISTS recording_error TEXT;
