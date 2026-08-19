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
