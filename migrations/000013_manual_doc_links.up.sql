CREATE TABLE manual_doc_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_doc_id UUID NOT NULL REFERENCES workspace_docs(id) ON DELETE CASCADE,
    request_id UUID NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_doc_id, request_id)
);

CREATE INDEX idx_manual_doc_links_request ON manual_doc_links(request_id);
CREATE INDEX idx_manual_doc_links_doc ON manual_doc_links(workspace_doc_id);
