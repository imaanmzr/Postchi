-- name: GetRequest :one
SELECT id, collection_id, name, method, url, headers, params, path_vars, body, auth, settings,
       pre_request_script, test_script, sort_order, description, template_id, is_template,
       COALESCE(overridden_fields, '{}'), source_spec_id,
       COALESCE(source_operation_id, ''), COALESCE(source_op_hash, ''),
       COALESCE(api_doc, '{}'), docs_overridden
FROM requests
WHERE id = @id;

-- name: ListRequestsByCollection :many
SELECT id, collection_id, name, method, url, headers, params, path_vars, body, auth, settings,
       pre_request_script, test_script, sort_order, description, template_id, is_template,
       COALESCE(overridden_fields, '{}'), source_spec_id,
       COALESCE(source_operation_id, ''), COALESCE(source_op_hash, ''),
       COALESCE(api_doc, '{}'), docs_overridden
FROM requests
WHERE collection_id = @collection_id
ORDER BY sort_order;

-- name: ListRequestsByWorkspace :many
SELECT r.id, r.collection_id, r.name, r.method, r.url, r.headers, r.params, r.path_vars, r.body, r.auth, r.settings,
       r.pre_request_script, r.test_script, r.sort_order, r.description, r.template_id, r.is_template,
       COALESCE(r.overridden_fields, '{}'), r.source_spec_id,
       COALESCE(r.source_operation_id, ''), COALESCE(r.source_op_hash, ''),
       COALESCE(r.api_doc, '{}'), r.docs_overridden
FROM requests r
JOIN collections c ON c.id = r.collection_id
WHERE c.workspace_id = @workspace_id
ORDER BY c.sort_order, c.name, r.sort_order;

-- name: ListRequestIDsByTemplate :many
SELECT id
FROM requests
WHERE template_id = @template_id;

-- name: ListTemplateChildOverriddenFields :many
SELECT id, overridden_fields
FROM requests
WHERE template_id = @template_id;

-- name: ListRequestIDsByCollection :many
SELECT id
FROM requests
WHERE collection_id = @collection_id
ORDER BY sort_order;

-- name: ListRequestIDAndNameByCollection :many
SELECT id, name
FROM requests
WHERE collection_id = @collection_id
ORDER BY sort_order;

-- name: ListRequestsForDuplicate :many
SELECT id, collection_id, name, method, url, headers, params, path_vars, body, auth, settings,
       pre_request_script, test_script, sort_order, description
FROM requests
WHERE collection_id = @collection_id;

-- name: ListCollectionDocEndpoints :many
SELECT id, name, method, url, description, COALESCE(api_doc, '{}'), source_spec_id
FROM requests
WHERE collection_id = @collection_id
ORDER BY sort_order;

-- name: ListRequestsForExport :many
SELECT name, method, url, headers, body, pre_request_script, test_script, sort_order, description
FROM requests
WHERE collection_id = @collection_id
ORDER BY sort_order;

-- name: ListRequestsForCatalogShare :many
SELECT r.id, r.name, r.method, r.url, r.description, COALESCE(r.api_doc, '{}')
FROM requests r
WHERE r.collection_id = @collection_id
ORDER BY r.sort_order;

-- name: ListCatalogEndpointsByWorkspace :many
SELECT r.id, r.collection_id, c.name, r.name, r.method, r.url, r.description,
       COALESCE(r.api_doc, '{}'), r.source_spec_id,
       COALESCE(r.source_operation_id, '')
FROM requests r
JOIN collections c ON c.id = r.collection_id
WHERE c.workspace_id = @workspace_id
ORDER BY c.sort_order, c.name, r.sort_order;

-- name: ListCatalogEndpointsByWorkspaceAndCollection :many
SELECT r.id, r.collection_id, c.name, r.name, r.method, r.url, r.description,
       COALESCE(r.api_doc, '{}'), r.source_spec_id,
       COALESCE(r.source_operation_id, '')
FROM requests r
JOIN collections c ON c.id = r.collection_id
WHERE c.workspace_id = @workspace_id AND r.collection_id = @collection_id
ORDER BY c.sort_order, c.name, r.sort_order;

-- name: GetRequestDocsBundle :one
SELECT r.description, COALESCE(r.api_doc, '{}'), COALESCE(r.source_operation_id, ''), r.collection_id,
       r.method, r.url
FROM requests r
WHERE r.id = @id;

-- name: ListRequestIDsByOperationInWorkspace :many
SELECT r.id
FROM requests r
JOIN collections c ON c.id = r.collection_id
WHERE c.workspace_id = @workspace_id AND r.source_operation_id = @operation_id;

-- name: CreateRequest :one
INSERT INTO requests (
    collection_id, name, method, url, headers, params, path_vars, body, auth, settings,
    pre_request_script, test_script, sort_order, description, template_id, is_template, overridden_fields,
    source_spec_id, source_operation_id, source_op_hash, api_doc, docs_overridden, created_by
) VALUES (
    @collection_id, @name, @method, @url, @headers, @params, @path_vars, @body, @auth, @settings,
    @pre_request_script, @test_script, @sort_order, @description, @template_id, @is_template, @overridden_fields,
    @source_spec_id, @source_operation_id, @source_op_hash, @api_doc, @docs_overridden, @created_by
) RETURNING id;

-- name: DuplicateRequest :exec
INSERT INTO requests (
    collection_id, name, method, url, headers, params, path_vars, body, auth, settings,
    pre_request_script, test_script, sort_order, description, created_by
) VALUES (
    @collection_id, @name, @method, @url, @headers, @params, @path_vars, @body, @auth, @settings,
    @pre_request_script, @test_script, @sort_order, @description, @created_by
);

-- name: InsertSyncedRequest :exec
INSERT INTO requests (
    collection_id, name, method, url,
    headers, params, path_vars, body, auth, settings,
    sort_order, description,
    source_spec_id, source_operation_id, source_op_hash,
    api_doc, created_by
) VALUES (
    @collection_id, @name, @method, @url,
    @headers, @params, @path_vars, @body, @auth, @settings,
    @sort_order, @description,
    @source_spec_id, @source_operation_id, @source_op_hash,
    @api_doc, @created_by
);

-- name: ImportCurlRequest :one
INSERT INTO requests (collection_id, name, method, url, headers, params, path_vars, body, auth, settings, created_by)
VALUES (@collection_id, @name, @method, @url, @headers, '[]', '[]', @body, '{}', '{}', @created_by)
RETURNING id;

-- name: InsertImportedRequest :exec
INSERT INTO requests (collection_id, name, method, url, headers, params, path_vars, body, auth, settings, pre_request_script, test_script, sort_order, description, source_operation_id, created_by)
VALUES (@collection_id, @name, @method, @url, @headers, @params, @path_vars, @body, @auth, @settings, @pre_request_script, @test_script, @sort_order, @description, @source_operation_id, @created_by);

-- name: BackfillRequestOperationIDs :execrows
UPDATE requests r
SET source_operation_id = @source_operation_id,
    updated_at = now()
FROM collections c
WHERE r.collection_id = c.id
  AND c.workspace_id = @workspace_id
  AND r.id = @request_id
  AND r.source_operation_id = ''
  AND r.source_spec_id IS NULL;

-- name: ListRequestsForOperationBackfill :many
SELECT r.id, r.method, r.url
FROM requests r
JOIN collections c ON c.id = r.collection_id
WHERE c.workspace_id = @workspace_id
  AND r.source_operation_id = ''
  AND r.source_spec_id IS NULL;

-- name: ImportSharedRequest :one
INSERT INTO requests (collection_id, name, method, url, headers, params, path_vars, body, auth, settings,
                      pre_request_script, test_script, sort_order, description, created_by)
VALUES (@collection_id, @name, @method, @url, @headers, @params, @path_vars, @body, @auth, @settings,
        @pre_request_script, @test_script, @sort_order, @description, @created_by)
RETURNING id;

-- name: UpdateRequest :exec
UPDATE requests
SET name = @name,
    method = @method,
    url = @url,
    headers = @headers,
    params = @params,
    path_vars = @path_vars,
    body = @body,
    auth = @auth,
    settings = @settings,
    pre_request_script = @pre_request_script,
    test_script = @test_script,
    sort_order = @sort_order,
    description = @description,
    overridden_fields = @overridden_fields,
    api_doc = COALESCE(@api_doc, api_doc),
    docs_overridden = @docs_overridden,
    updated_at = now()
WHERE id = @id;

-- name: SnapshotTemplateChild :exec
UPDATE requests
SET method = @method,
    url = @url,
    headers = @headers,
    params = @params,
    path_vars = @path_vars,
    body = @body,
    auth = @auth,
    settings = @settings,
    pre_request_script = @pre_request_script,
    test_script = @test_script,
    template_id = NULL,
    overridden_fields = @overridden_fields,
    updated_at = now()
WHERE id = @id;

-- name: UpdateRequestOverriddenFields :exec
UPDATE requests
SET overridden_fields = @overridden_fields,
    updated_at = now()
WHERE id = @id;

-- name: PromoteRequestToTemplate :exec
UPDATE requests
SET is_template = true,
    updated_at = now()
WHERE id = @id;

-- name: MoveRequest :exec
UPDATE requests
SET collection_id = @collection_id,
    sort_order = @sort_order,
    updated_at = now()
WHERE id = @id;

-- name: UpdateRequestSortOrder :exec
UPDATE requests
SET sort_order = @sort_order,
    updated_at = now()
WHERE id = @id;

-- name: DeleteRequest :exec
DELETE FROM requests
WHERE id = @id;

-- name: GetRequestForShareSnapshot :one
SELECT r.id, r.collection_id, r.name, r.method, r.url, r.headers, r.params, r.path_vars, r.body, r.auth, r.settings,
       r.pre_request_script, r.test_script, r.sort_order, r.description
FROM requests r
JOIN collections c ON c.id = r.collection_id
WHERE r.id = @id AND c.workspace_id = @workspace_id;

-- name: CreateHistoryEntry :one
INSERT INTO history (
    workspace_id, request_id, snapshot, response, test_results,
    executed_by, duration_ms, status_code
) VALUES (
    @workspace_id, @request_id, @snapshot, @response, @test_results,
    @executed_by, @duration_ms, @status_code
) RETURNING id;

-- name: CreateExample :one
INSERT INTO examples (request_id, name, response, created_by)
VALUES (@request_id, @name, @response, @created_by)
RETURNING id;

-- name: CreateSharedResponseExample :exec
INSERT INTO examples (request_id, name, response, created_by)
VALUES (@request_id, @name, @response, @created_by);

-- name: GetWorkspaceVariables :one
SELECT variables
FROM workspaces
WHERE id = @id;

-- name: ListEnvironmentVariablesForExecutor :many
SELECT key, value_encrypted, is_secret, phase, enabled
FROM environment_variables
WHERE environment_id = @environment_id;

-- name: GetSpecBaseURLForEnvironment :one
SELECT s.base_url_var, u.base_url
FROM api_specs s
JOIN api_spec_environment_urls u ON u.api_spec_id = s.id AND u.environment_id = @environment_id
WHERE s.id = @spec_id;
