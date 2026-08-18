CREATE TABLE doc_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    collection_id UUID REFERENCES collections(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    source_type TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    last_synced_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE workspace_docs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    doc_source_id UUID REFERENCES doc_sources(id) ON DELETE SET NULL,
    slug TEXT NOT NULL,
    title TEXT NOT NULL,
    content_md TEXT NOT NULL DEFAULT '',
    linked_operation_ids TEXT[] NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, slug)
);

CREATE TABLE workspace_api_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    token_prefix TEXT NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT '{spec:push}',
    created_by UUID NOT NULL REFERENCES users(id),
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_doc_sources_workspace ON doc_sources(workspace_id);
CREATE INDEX idx_workspace_docs_workspace ON workspace_docs(workspace_id);
CREATE INDEX idx_workspace_api_tokens_workspace ON workspace_api_tokens(workspace_id);
