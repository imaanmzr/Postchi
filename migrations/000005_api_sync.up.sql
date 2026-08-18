ALTER TABLE environments ADD COLUMN IF NOT EXISTS stage TEXT NOT NULL DEFAULT 'custom';

ALTER TABLE requests
    ADD COLUMN IF NOT EXISTS source_spec_id UUID,
    ADD COLUMN IF NOT EXISTS source_operation_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS source_op_hash TEXT NOT NULL DEFAULT '';

CREATE TABLE api_specs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    collection_id UUID REFERENCES collections(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    source_type TEXT NOT NULL DEFAULT 'url',
    spec_url TEXT NOT NULL DEFAULT '',
    spec_hash TEXT NOT NULL DEFAULT '',
    base_url_var TEXT NOT NULL DEFAULT 'baseUrl',
    last_synced_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE api_spec_environment_urls (
    api_spec_id UUID NOT NULL REFERENCES api_specs(id) ON DELETE CASCADE,
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    base_url TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (api_spec_id, environment_id)
);

ALTER TABLE requests ADD CONSTRAINT fk_requests_source_spec
    FOREIGN KEY (source_spec_id) REFERENCES api_specs(id) ON DELETE SET NULL;

CREATE INDEX idx_api_specs_workspace ON api_specs(workspace_id);
