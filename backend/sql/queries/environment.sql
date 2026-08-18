-- name: ListEnvironments :many
SELECT id, workspace_id, name, stage
FROM environments
WHERE workspace_id = @workspace_id
ORDER BY name;

-- name: CreateEnvironment :one
INSERT INTO environments (workspace_id, name, stage, created_by)
VALUES (@workspace_id, @name, @stage, @created_by)
RETURNING id;

-- name: UpdateEnvironmentName :exec
UPDATE environments
SET name = @name, updated_at = now()
WHERE id = @id;

-- name: UpdateEnvironmentStage :exec
UPDATE environments
SET stage = @stage, updated_at = now()
WHERE id = @id;

-- name: DeleteEnvironment :exec
DELETE FROM environments
WHERE id = @id;

-- name: DeleteEnvironmentVariables :exec
DELETE FROM environment_variables
WHERE environment_id = @environment_id;

-- name: UpsertEnvironmentVariable :exec
INSERT INTO environment_variables (environment_id, key, value_encrypted, is_secret, phase, enabled, type, description, expr)
VALUES (@environment_id, @key, @value_encrypted, @is_secret, @phase, @enabled, @type, @description, @expr)
ON CONFLICT (environment_id, key, phase) DO UPDATE SET
    value_encrypted = EXCLUDED.value_encrypted,
    is_secret = EXCLUDED.is_secret,
    enabled = EXCLUDED.enabled,
    type = EXCLUDED.type,
    description = EXCLUDED.description,
    expr = EXCLUDED.expr,
    updated_at = now();

-- name: GetEnvironment :one
SELECT id, workspace_id, name, stage
FROM environments
WHERE id = @id;

-- name: ListEnvironmentVariables :many
SELECT id, key, value_encrypted, is_secret, phase, enabled, type, description, expr
FROM environment_variables
WHERE environment_id = @environment_id
ORDER BY phase, key;

-- name: ListEnvironmentVariablesForDecrypt :many
SELECT key, value_encrypted, is_secret, phase, enabled, expr
FROM environment_variables
WHERE environment_id = @environment_id;
