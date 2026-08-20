CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT '',
    auth_provider TEXT NOT NULL DEFAULT 'local',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    variables JSONB NOT NULL DEFAULT '{"pre_request":[],"post_response":[]}'::jsonb,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TYPE workspace_role AS ENUM ('owner', 'editor', 'viewer');

CREATE TABLE workspace_members (
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role workspace_role NOT NULL DEFAULT 'viewer',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, user_id)
);

CREATE TABLE collections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES collections(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    variables JSONB NOT NULL DEFAULT '{"pre_request":[],"post_response":[]}'::jsonb,
    headers JSONB NOT NULL DEFAULT '[]',
    auth JSONB NOT NULL DEFAULT '{}',
    presets JSONB NOT NULL DEFAULT '[]',
    proxy JSONB NOT NULL DEFAULT '{}',
    client_certificates JSONB NOT NULL DEFAULT '[]',
    secrets JSONB NOT NULL DEFAULT '[]',
    pre_request_script TEXT NOT NULL DEFAULT '',
    test_script TEXT NOT NULL DEFAULT '',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_collections_workspace ON collections(workspace_id);
CREATE INDEX idx_collections_parent ON collections(parent_id);

CREATE TABLE environments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    stage TEXT NOT NULL DEFAULT 'custom',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE api_specs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    collection_id UUID REFERENCES collections(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    source_type TEXT NOT NULL DEFAULT 'url',
    spec_url TEXT NOT NULL DEFAULT '',
    spec_hash TEXT NOT NULL DEFAULT '',
    spec_content BYTEA,
    base_url_var TEXT NOT NULL DEFAULT 'baseUrl',
    last_synced_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_api_specs_workspace ON api_specs(workspace_id);

CREATE TABLE api_spec_environment_urls (
    api_spec_id UUID NOT NULL REFERENCES api_specs(id) ON DELETE CASCADE,
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    base_url TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (api_spec_id, environment_id)
);

CREATE TABLE requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    collection_id UUID NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    method TEXT NOT NULL DEFAULT 'GET',
    url TEXT NOT NULL DEFAULT '',
    headers JSONB NOT NULL DEFAULT '[]',
    params JSONB NOT NULL DEFAULT '[]',
    path_vars JSONB NOT NULL DEFAULT '[]',
    body JSONB NOT NULL DEFAULT '{}',
    auth JSONB NOT NULL DEFAULT '{}',
    settings JSONB NOT NULL DEFAULT '{}',
    pre_request_script TEXT NOT NULL DEFAULT '',
    test_script TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    description TEXT NOT NULL DEFAULT '',
    template_id UUID REFERENCES requests(id) ON DELETE SET NULL,
    is_template BOOLEAN NOT NULL DEFAULT false,
    overridden_fields TEXT[] NOT NULL DEFAULT '{}',
    source_spec_id UUID REFERENCES api_specs(id) ON DELETE SET NULL,
    source_operation_id TEXT NOT NULL DEFAULT '',
    source_op_hash TEXT NOT NULL DEFAULT '',
    api_doc JSONB NOT NULL DEFAULT '{}',
    docs_overridden BOOLEAN NOT NULL DEFAULT false,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_requests_collection ON requests(collection_id);
CREATE INDEX idx_requests_template ON requests(template_id);

CREATE TABLE environment_variables (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    value_encrypted TEXT NOT NULL DEFAULT '',
    is_secret BOOLEAN NOT NULL DEFAULT false,
    phase TEXT NOT NULL DEFAULT 'pre_request',
    enabled BOOLEAN NOT NULL DEFAULT true,
    type TEXT NOT NULL DEFAULT 'string',
    description TEXT NOT NULL DEFAULT '',
    expr TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (environment_id, key, phase)
);

CREATE TABLE history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    request_id UUID REFERENCES requests(id) ON DELETE SET NULL,
    snapshot JSONB NOT NULL DEFAULT '{}',
    response JSONB NOT NULL DEFAULT '{}',
    test_results JSONB NOT NULL DEFAULT '[]',
    executed_by UUID NOT NULL REFERENCES users(id),
    executed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    duration_ms BIGINT NOT NULL DEFAULT 0,
    status_code INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_history_workspace ON history(workspace_id, executed_at DESC);
CREATE INDEX idx_history_request ON history(request_id);

CREATE TABLE examples (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id UUID NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
    name TEXT NOT NULL DEFAULT 'Example',
    response JSONB NOT NULL DEFAULT '{}',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE activity_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    actor_id UUID NOT NULL REFERENCES users(id),
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id UUID,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_activity_workspace ON activity_log(workspace_id, created_at DESC);

CREATE TABLE workspace_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    role workspace_role NOT NULL DEFAULT 'viewer',
    token TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, email)
);

CREATE INDEX idx_workspace_invites_token ON workspace_invites(token);
CREATE INDEX idx_workspace_invites_workspace ON workspace_invites(workspace_id);

CREATE TYPE share_kind AS ENUM ('request', 'history', 'catalog');

CREATE TABLE shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    kind share_kind NOT NULL,
    source_id UUID NOT NULL,
    token TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL DEFAULT '',
    snapshot JSONB NOT NULL,
    visibility TEXT NOT NULL DEFAULT 'workspace',
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_shares_workspace ON shares(workspace_id, created_at DESC);
CREATE INDEX idx_shares_token ON shares(token);

CREATE TABLE doc_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    collection_id UUID REFERENCES collections(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    source_type TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    access_token_encrypted TEXT,
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
    source_path TEXT NOT NULL DEFAULT '',
    is_local BOOLEAN NOT NULL DEFAULT false,
    linked_operation_ids TEXT[] NOT NULL DEFAULT '{}',
    linked_request_names TEXT[] NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, slug)
);

CREATE UNIQUE INDEX idx_workspace_docs_workspace_source_path
    ON workspace_docs (workspace_id, source_path)
    WHERE source_path <> '';

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

CREATE TABLE manual_doc_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_doc_id UUID NOT NULL REFERENCES workspace_docs(id) ON DELETE CASCADE,
    request_id UUID NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_doc_id, request_id)
);

CREATE INDEX idx_manual_doc_links_request ON manual_doc_links(request_id);
CREATE INDEX idx_manual_doc_links_doc ON manual_doc_links(workspace_doc_id);

CREATE TABLE doc_link_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    workspace_doc_id UUID NOT NULL REFERENCES workspace_docs(id) ON DELETE CASCADE,
    request_id UUID NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    confidence TEXT NOT NULL,
    evidence JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at TIMESTAMPTZ,
    reviewed_by UUID REFERENCES users(id),
    UNIQUE (workspace_doc_id, request_id)
);

CREATE INDEX idx_doc_link_suggestions_workspace_status ON doc_link_suggestions(workspace_id, status);

CREATE INDEX idx_doc_sources_workspace ON doc_sources(workspace_id);
CREATE INDEX idx_workspace_docs_workspace ON workspace_docs(workspace_id);
CREATE INDEX idx_workspace_api_tokens_workspace ON workspace_api_tokens(workspace_id);

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
