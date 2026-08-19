DROP INDEX IF EXISTS idx_requests_bruno_source_path;
DROP INDEX IF EXISTS idx_collections_bruno_source_path;
DROP INDEX IF EXISTS idx_requests_bruno_source;
DROP INDEX IF EXISTS idx_collections_bruno_source;
DROP INDEX IF EXISTS idx_bruno_sources_workspace;

ALTER TABLE requests
    DROP COLUMN IF EXISTS bruno_source_id,
    DROP COLUMN IF EXISTS source_path;

ALTER TABLE collections
    DROP COLUMN IF EXISTS bruno_source_id,
    DROP COLUMN IF EXISTS source_path;

DROP TABLE IF EXISTS bruno_sources;
