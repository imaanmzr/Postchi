-- name: ListWorkspaceApiTokens :many
SELECT id, workspace_id, name, token_prefix, scopes, created_at, revoked_at
FROM workspace_api_tokens
WHERE workspace_id = @workspace_id
ORDER BY created_at DESC;

-- name: CreateWorkspaceApiToken :one
INSERT INTO workspace_api_tokens (workspace_id, name, token_hash, token_prefix, scopes, created_by)
VALUES (@workspace_id, @name, @token_hash, @token_prefix, @scopes, @created_by)
RETURNING id;

-- name: RevokeWorkspaceApiToken :exec
UPDATE workspace_api_tokens
SET revoked_at = now()
WHERE id = @id;

-- name: GetWorkspaceApiTokenByHash :one
SELECT workspace_id, scopes, revoked_at
FROM workspace_api_tokens
WHERE token_hash = @token_hash;
