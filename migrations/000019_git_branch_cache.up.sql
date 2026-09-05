CREATE TABLE git_branch_cache (
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    repo_key TEXT NOT NULL,
    branches JSONB NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, repo_key)
);

CREATE INDEX idx_git_branch_cache_fetched ON git_branch_cache(fetched_at);
