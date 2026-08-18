ALTER TABLE workspace_docs
    ADD COLUMN source_path TEXT NOT NULL DEFAULT '',
    ADD COLUMN is_local BOOLEAN NOT NULL DEFAULT false;

-- Best-effort backfill for docs synced before source_path existed.
UPDATE workspace_docs
SET source_path = REPLACE(slug, '-', '/')
WHERE source_path = '' AND is_local = false;

CREATE UNIQUE INDEX idx_workspace_docs_workspace_source_path
    ON workspace_docs (workspace_id, source_path)
    WHERE source_path <> '';
