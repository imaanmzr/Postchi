CREATE TYPE share_kind AS ENUM ('request', 'history');

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
