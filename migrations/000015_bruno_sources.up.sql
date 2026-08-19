CREATE TABLE bruno_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    collection_id UUID REFERENCES collections(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    access_token_encrypted TEXT,
    last_synced_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE collections
    ADD COLUMN IF NOT EXISTS source_path TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS bruno_source_id UUID REFERENCES bruno_sources(id) ON DELETE SET NULL;

ALTER TABLE requests
    ADD COLUMN IF NOT EXISTS source_path TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS bruno_source_id UUID REFERENCES bruno_sources(id) ON DELETE SET NULL;

CREATE INDEX idx_bruno_sources_workspace ON bruno_sources(workspace_id);
CREATE INDEX idx_collections_bruno_source ON collections(bruno_source_id) WHERE bruno_source_id IS NOT NULL;
CREATE INDEX idx_requests_bruno_source ON requests(bruno_source_id) WHERE bruno_source_id IS NOT NULL;

CREATE UNIQUE INDEX idx_collections_bruno_source_path
    ON collections (bruno_source_id, source_path)
    WHERE bruno_source_id IS NOT NULL AND source_path <> '';

CREATE UNIQUE INDEX idx_requests_bruno_source_path
    ON requests (bruno_source_id, source_path)
    WHERE bruno_source_id IS NOT NULL AND source_path <> '';
