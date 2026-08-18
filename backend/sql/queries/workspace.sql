-- name: ListWorkspacesByUser :many
SELECT w.id, w.name, w.description, w.variables, wm.role::text, w.created_at
FROM workspaces w
JOIN workspace_members wm ON wm.workspace_id = w.id
WHERE wm.user_id = @user_id
ORDER BY w.created_at DESC;

-- name: CreateWorkspace :one
INSERT INTO workspaces (name, description, created_by)
VALUES (@name, @description, @created_by)
RETURNING id;

-- name: UserHasWorkspaceNamed :one
SELECT EXISTS(
    SELECT 1
    FROM workspaces w
    INNER JOIN workspace_members wm ON wm.workspace_id = w.id
    WHERE wm.user_id = @user_id
      AND lower(trim(w.name)) = lower(trim(@name))
) AS exists;

-- name: UserHasWorkspaceNamedExcluding :one
SELECT EXISTS(
    SELECT 1
    FROM workspaces w
    INNER JOIN workspace_members wm ON wm.workspace_id = w.id
    WHERE wm.user_id = @user_id
      AND lower(trim(w.name)) = lower(trim(@name))
      AND w.id <> @exclude_workspace_id
) AS exists;

-- name: AddWorkspaceOwner :exec
INSERT INTO workspace_members (workspace_id, user_id, role)
VALUES (@workspace_id, @user_id, 'owner'::workspace_role);

-- name: CreateDefaultCollection :exec
INSERT INTO collections (workspace_id, name, created_by)
VALUES (@workspace_id, 'My Collection', @created_by);

-- name: GetWorkspaceByID :one
SELECT id, name, description, variables
FROM workspaces
WHERE id = @id;

-- name: UpdateWorkspaceName :exec
UPDATE workspaces
SET name = @name, updated_at = now()
WHERE id = @id;

-- name: UpdateWorkspaceDescription :exec
UPDATE workspaces
SET description = @description, updated_at = now()
WHERE id = @id;

-- name: UpdateWorkspaceVariables :exec
UPDATE workspaces
SET variables = @variables, updated_at = now()
WHERE id = @id;

-- name: ListWorkspaceMembers :many
SELECT u.id, u.email, u.display_name, wm.role::text
FROM workspace_members wm
JOIN users u ON u.id = wm.user_id
WHERE wm.workspace_id = @workspace_id;

-- name: GetUserIDByEmail :one
SELECT id
FROM users
WHERE email = @email;

-- name: UpsertWorkspaceMember :exec
INSERT INTO workspace_members (workspace_id, user_id, role)
VALUES (@workspace_id, @user_id, @role::workspace_role)
ON CONFLICT (workspace_id, user_id) DO UPDATE SET role = EXCLUDED.role;

-- name: DeleteWorkspace :exec
DELETE FROM workspaces
WHERE id = @id;

-- name: CountWorkspaceOwners :one
SELECT COUNT(*)
FROM workspace_members
WHERE workspace_id = @workspace_id AND role = 'owner';

-- name: UpdateWorkspaceMemberRole :exec
UPDATE workspace_members
SET role = @role::workspace_role
WHERE workspace_id = @workspace_id AND user_id = @user_id;

-- name: DeleteWorkspaceMember :exec
DELETE FROM workspace_members
WHERE workspace_id = @workspace_id AND user_id = @user_id;

-- name: ListWorkspaceActivity :many
SELECT al.id, al.action, al.entity_type, al.entity_id, al.metadata, al.created_at, u.email
FROM activity_log al
JOIN users u ON u.id = al.actor_id
WHERE al.workspace_id = @workspace_id
ORDER BY al.created_at DESC
LIMIT 100;

-- name: CreateActivityLog :exec
INSERT INTO activity_log (workspace_id, actor_id, action, entity_type, entity_id, metadata)
VALUES (@workspace_id, @actor_id, @action, @entity_type, @entity_id, @metadata);

-- name: GetWorkspaceName :one
SELECT name
FROM workspaces
WHERE id = @id;

-- name: WorkspaceExists :one
SELECT EXISTS(SELECT 1 FROM workspaces WHERE id = @id);
