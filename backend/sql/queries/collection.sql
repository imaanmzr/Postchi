-- name: GetCollection :one
SELECT id, workspace_id, parent_id, name, description, sort_order, variables,
       headers, auth, presets, proxy, client_certificates, secrets,
       pre_request_script, test_script
FROM collections
WHERE id = @id;

-- name: ListCollectionsByWorkspace :many
SELECT id, workspace_id, parent_id, name, description, sort_order, variables,
       headers, auth, presets, proxy, client_certificates, secrets,
       pre_request_script, test_script
FROM collections
WHERE workspace_id = @workspace_id
ORDER BY sort_order, name;

-- name: ListCollectionsByParent :many
SELECT id, workspace_id, parent_id, name, description, sort_order, variables,
       headers, auth, presets, proxy, client_certificates, secrets,
       pre_request_script, test_script
FROM collections
WHERE parent_id = @parent_id;

-- name: CreateCollection :one
INSERT INTO collections (
    workspace_id, parent_id, name, description, sort_order, variables,
    headers, auth, presets, proxy, client_certificates, secrets,
    pre_request_script, test_script, created_by
) VALUES (
    @workspace_id, @parent_id, @name, @description, @sort_order, @variables,
    @headers, @auth, @presets, @proxy, @client_certificates, @secrets,
    @pre_request_script, @test_script, @created_by
) RETURNING id;

-- name: UpdateCollection :exec
UPDATE collections
SET name = @name,
    description = @description,
    parent_id = @parent_id,
    sort_order = @sort_order,
    variables = @variables,
    headers = @headers,
    auth = @auth,
    presets = @presets,
    proxy = @proxy,
    client_certificates = @client_certificates,
    secrets = @secrets,
    pre_request_script = @pre_request_script,
    test_script = @test_script,
    updated_at = now()
WHERE id = @id;

-- name: DeleteCollection :exec
DELETE FROM collections
WHERE id = @id;

-- name: ReorderCollection :exec
UPDATE collections
SET parent_id = @parent_id,
    sort_order = @sort_order,
    updated_at = now()
WHERE id = @id;

-- name: GetCollectionNameDescription :one
SELECT name, description
FROM collections
WHERE id = @id;

-- name: GetCollectionParentID :one
SELECT parent_id
FROM collections
WHERE id = @id;

-- name: GetCollectionContext :one
SELECT workspace_id, pre_request_script, test_script
FROM collections
WHERE id = @id;

-- name: GetCollectionAuth :one
SELECT auth
FROM collections
WHERE id = @id;

-- name: GetCollectionVariables :one
SELECT variables
FROM collections
WHERE id = @id;

-- name: GetCollectionWorkspaceID :one
SELECT workspace_id
FROM collections
WHERE id = @id;

-- name: CollectionExists :one
SELECT EXISTS(SELECT 1 FROM collections WHERE id = @id);

-- name: GetCollectionForExport :one
SELECT name, description, variables, headers, auth, pre_request_script, test_script
FROM collections
WHERE id = @id;

-- name: GetCollectionForCatalogShare :one
SELECT name, description
FROM collections
WHERE id = @id AND workspace_id = @workspace_id;

-- name: GetCollectionWorkspaceForMember :one
SELECT c.workspace_id
FROM collections c
JOIN workspace_members wm ON wm.workspace_id = c.workspace_id AND wm.user_id = @user_id
WHERE c.id = @collection_id;

-- name: ListChildCollectionIDs :many
SELECT id
FROM collections
WHERE parent_id = @parent_id;

-- name: ListCatalogCollections :many
SELECT id, name, description
FROM collections
WHERE workspace_id = @workspace_id
ORDER BY sort_order, name;

-- name: ListCatalogCollectionsByID :many
SELECT id, name, description
FROM collections
WHERE workspace_id = @workspace_id AND id = @collection_id
ORDER BY sort_order, name;
