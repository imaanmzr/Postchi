-- name: UpsertWorkspaceInvite :one
INSERT INTO workspace_invites (workspace_id, email, role, token, expires_at, created_by)
VALUES (@workspace_id, @email, @role::workspace_role, @token, @expires_at, @created_by)
ON CONFLICT (workspace_id, email) DO UPDATE SET
    role = EXCLUDED.role,
    token = EXCLUDED.token,
    expires_at = EXCLUDED.expires_at,
    created_by = EXCLUDED.created_by,
    accepted_at = NULL
RETURNING id;

-- name: ListPendingWorkspaceInvites :many
SELECT id, workspace_id, email, role::text, token, expires_at, created_at
FROM workspace_invites
WHERE workspace_id = @workspace_id
  AND accepted_at IS NULL
  AND expires_at > now()
ORDER BY created_at DESC;

-- name: DeleteWorkspaceInvite :exec
DELETE FROM workspace_invites
WHERE id = @id AND workspace_id = @workspace_id;

-- name: GetWorkspaceInviteByToken :one
SELECT id, workspace_id, email, role::text, expires_at, accepted_at
FROM workspace_invites
WHERE token = @token;

-- name: MarkWorkspaceInviteAccepted :exec
UPDATE workspace_invites
SET accepted_at = now()
WHERE id = @id;
