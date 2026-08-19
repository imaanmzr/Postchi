-- name: ListBrunoSources :many
SELECT id, workspace_id, collection_id, name, config, access_token_encrypted, last_synced_at, created_at
FROM bruno_sources
WHERE workspace_id = @workspace_id
ORDER BY name;

-- name: CreateBrunoSource :one
INSERT INTO bruno_sources (workspace_id, name, config, access_token_encrypted, created_by)
VALUES (@workspace_id, @name, @config, @access_token_encrypted, @created_by)
RETURNING id;

-- name: GetBrunoSource :one
SELECT id, workspace_id, collection_id, name, config, access_token_encrypted, last_synced_at, created_at
FROM bruno_sources
WHERE id = @id AND workspace_id = @workspace_id;

-- name: GetBrunoSourceForSync :one
SELECT id, workspace_id, collection_id, name, config, access_token_encrypted, last_synced_at
FROM bruno_sources
WHERE id = @id;

-- name: UpdateBrunoSource :exec
UPDATE bruno_sources
SET name = COALESCE(sqlc.narg('name'), name),
    config = COALESCE(sqlc.narg('config'), config),
    access_token_encrypted = COALESCE(sqlc.narg('access_token_encrypted'), access_token_encrypted)
WHERE id = @id AND workspace_id = @workspace_id;

-- name: UpdateBrunoSourceCollectionID :exec
UPDATE bruno_sources
SET collection_id = @collection_id
WHERE id = @id;

-- name: UpdateBrunoSourceLastSynced :exec
UPDATE bruno_sources
SET last_synced_at = now()
WHERE id = @id;

-- name: DeleteBrunoSource :exec
DELETE FROM bruno_sources
WHERE id = @id AND workspace_id = @workspace_id;

-- name: InsertBrunoSyncedCollection :one
INSERT INTO collections (
    workspace_id, parent_id, name, description, sort_order, variables, headers, auth,
    pre_request_script, test_script, source_path, bruno_source_id, created_by
)
VALUES (
    @workspace_id, @parent_id, @name, @description, @sort_order, @variables, @headers, @auth,
    @pre_request_script, @test_script, @source_path, @bruno_source_id, @created_by
)
RETURNING id;

-- name: UpdateBrunoSyncedCollection :exec
UPDATE collections
SET name = @name,
    description = @description,
    sort_order = @sort_order,
    variables = @variables,
    headers = @headers,
    auth = @auth,
    pre_request_script = @pre_request_script,
    test_script = @test_script,
    parent_id = @parent_id,
    updated_at = now()
WHERE id = @id AND bruno_source_id = @bruno_source_id;

-- name: ListBrunoSyncedCollections :many
SELECT id, parent_id, source_path, name
FROM collections
WHERE bruno_source_id = @bruno_source_id;

-- name: ListBrunoSyncedRequests :many
SELECT id, collection_id, source_path, COALESCE(source_op_hash, ''), name
FROM requests
WHERE bruno_source_id = @bruno_source_id AND source_path <> '';

-- name: InsertBrunoSyncedRequest :exec
INSERT INTO requests (
    collection_id, name, method, url, headers, params, path_vars, body, auth, settings,
    pre_request_script, test_script, sort_order, description,
    source_path, bruno_source_id, source_operation_id, source_op_hash, created_by
)
VALUES (
    @collection_id, @name, @method, @url, @headers, @params, @path_vars, @body, @auth, @settings,
    @pre_request_script, @test_script, @sort_order, @description,
    @source_path, @bruno_source_id, @source_operation_id, @source_op_hash, @created_by
);

-- name: UpdateBrunoSyncedRequest :exec
UPDATE requests
SET name = @name,
    method = @method,
    url = @url,
    headers = @headers,
    params = @params,
    path_vars = @path_vars,
    body = CASE WHEN 'body' = ANY(overridden_fields) THEN body ELSE @body END,
    auth = @auth,
    pre_request_script = @pre_request_script,
    test_script = @test_script,
    sort_order = @sort_order,
    description = @description,
    collection_id = @collection_id,
    source_op_hash = @source_op_hash,
    source_operation_id = @source_operation_id,
    updated_at = now()
WHERE id = @id AND bruno_source_id = @bruno_source_id;

-- name: DeleteBrunoSyncedCollectionsBySource :exec
DELETE FROM collections
WHERE bruno_source_id = @bruno_source_id;

-- name: DeleteBrunoSyncedRequestsNotInPaths :execrows
DELETE FROM requests
WHERE bruno_source_id = @bruno_source_id
  AND source_path <> ''
  AND NOT (source_path = ANY(@keep_paths::text[]));

-- name: DeleteBrunoSyncedCollectionsNotInPaths :execrows
DELETE FROM collections
WHERE bruno_source_id = @bruno_source_id
  AND source_path <> ''
  AND NOT (source_path = ANY(@keep_paths::text[]));
