-- name: CreateShare :one
INSERT INTO shares (workspace_id, kind, source_id, token, title, snapshot, visibility, expires_at, created_by)
VALUES (@workspace_id, @kind::share_kind, @source_id, @token, @title, @snapshot, @visibility, @expires_at, @created_by)
RETURNING id;

-- name: ListActiveShares :many
SELECT id, workspace_id, kind::text, source_id, token, title, visibility, expires_at, created_by, created_at
FROM shares
WHERE workspace_id = @workspace_id
  AND revoked_at IS NULL
  AND (expires_at IS NULL OR expires_at > now())
ORDER BY created_at DESC;

-- name: GetShareWorkspaceAndCreator :one
SELECT workspace_id, created_by
FROM shares
WHERE id = @id;

-- name: RevokeShare :exec
UPDATE shares
SET revoked_at = now()
WHERE id = @id;

-- name: GetShareByToken :one
SELECT id, workspace_id, kind::text, source_id, token, title, snapshot, visibility, expires_at, revoked_at, created_by, created_at
FROM shares
WHERE token = @token;
