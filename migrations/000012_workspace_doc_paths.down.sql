DROP INDEX IF EXISTS idx_workspace_docs_workspace_source_path;

ALTER TABLE workspace_docs
    DROP COLUMN IF EXISTS is_local,
    DROP COLUMN IF EXISTS source_path;
