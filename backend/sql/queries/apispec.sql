-- name: CreateApiSpec :one
INSERT INTO api_specs (workspace_id, collection_id, name, spec_url, spec_hash, base_url_var, created_by)
VALUES (@workspace_id, @collection_id, @name, @spec_url, @spec_hash, @base_url_var, @created_by)
RETURNING id;

-- name: DeleteApiSpec :exec
DELETE FROM api_specs
WHERE id = @id;

-- name: UpdateApiSpecLastSynced :exec
UPDATE api_specs
SET last_synced_at = now(), updated_at = now()
WHERE id = @id;

-- name: ListApiSpecs :many
SELECT id, workspace_id, collection_id, name, source_type, spec_url, spec_hash, base_url_var, last_synced_at, created_at, updated_at
FROM api_specs
WHERE workspace_id = @workspace_id
ORDER BY name;

-- name: UpdateApiSpecName :exec
UPDATE api_specs
SET name = @name, updated_at = now()
WHERE id = @id;

-- name: UpdateApiSpecURL :exec
UPDATE api_specs
SET spec_url = @spec_url, updated_at = now()
WHERE id = @id;

-- name: UpdateApiSpecBaseURLVar :exec
UPDATE api_specs
SET base_url_var = @base_url_var, updated_at = now()
WHERE id = @id;

-- name: UpsertApiSpecEnvironmentURL :exec
INSERT INTO api_spec_environment_urls (api_spec_id, environment_id, base_url)
VALUES (@api_spec_id, @environment_id, @base_url)
ON CONFLICT (api_spec_id, environment_id) DO UPDATE SET base_url = EXCLUDED.base_url;

-- name: GetApiSpecRow :one
SELECT id, workspace_id, collection_id, name, COALESCE(source_type, 'url'), spec_url, spec_hash, base_url_var, spec_content
FROM api_specs
WHERE id = @id;

-- name: GetApiSpec :one
SELECT id, workspace_id, collection_id, name, source_type, spec_url, spec_hash, base_url_var, last_synced_at, created_at, updated_at
FROM api_specs
WHERE id = @id;

-- name: ListSyncedRequestsBySpec :many
SELECT id, COALESCE(source_operation_id, ''), COALESCE(source_op_hash, ''), name, method, url
FROM requests
WHERE source_spec_id = @source_spec_id;

-- name: UpdateApiSpecCollectionID :exec
UPDATE api_specs
SET collection_id = @collection_id, updated_at = now()
WHERE id = @id;

-- name: UpdateSyncedRequest :exec
UPDATE requests
SET method = @method,
    url = @url,
    headers = @headers,
    params = @params,
    path_vars = @path_vars,
    source_op_hash = @source_op_hash,
    api_doc = @api_doc,
    description = CASE WHEN docs_overridden THEN description ELSE @description END,
    body = CASE WHEN 'body' = ANY(overridden_fields) THEN body ELSE @body END,
    updated_at = now()
WHERE source_spec_id = @source_spec_id AND source_operation_id = @source_operation_id;

-- name: UpdateApiSpecAfterSync :exec
UPDATE api_specs
SET spec_hash = @spec_hash, last_synced_at = now(), updated_at = now()
WHERE id = @id;

-- name: CreateUploadedApiSpec :one
INSERT INTO api_specs (workspace_id, collection_id, name, source_type, spec_url, spec_hash, spec_content, base_url_var, created_by)
VALUES (@workspace_id, @collection_id, @name, 'upload', '', @spec_hash, @spec_content, 'baseUrl', @created_by)
RETURNING id;

-- name: UpdateApiSpecLastSyncedOnly :exec
UPDATE api_specs
SET last_synced_at = now()
WHERE id = @id;

-- name: UpdateApiSpecReupload :exec
UPDATE api_specs
SET spec_content = @spec_content, spec_hash = @spec_hash, source_type = 'upload', updated_at = now()
WHERE id = @id;

-- name: GetApiSpecIDByWorkspaceAndName :one
SELECT id
FROM api_specs
WHERE workspace_id = @workspace_id AND name = @name
LIMIT 1;

-- name: CreatePushedApiSpec :one
INSERT INTO api_specs (workspace_id, collection_id, name, source_type, spec_hash, spec_content, created_by)
VALUES (@workspace_id, @collection_id, @name, 'push', @spec_hash, @spec_content, @created_by)
RETURNING id;

-- name: UpdatePushedApiSpec :exec
UPDATE api_specs
SET spec_content = @spec_content, spec_hash = @spec_hash, source_type = 'push', updated_at = now()
WHERE id = @id;

-- name: CreateSpecCollection :one
INSERT INTO collections (workspace_id, name, created_by)
VALUES (@workspace_id, @name, @created_by)
RETURNING id;
