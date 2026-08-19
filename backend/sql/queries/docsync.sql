-- name: ListDocSources :many
SELECT id, workspace_id, collection_id, name, source_type, config, access_token_encrypted, last_synced_at, created_at
FROM doc_sources
WHERE workspace_id = @workspace_id
ORDER BY name;

-- name: CreateDocSource :one
INSERT INTO doc_sources (workspace_id, collection_id, name, source_type, config, access_token_encrypted, created_by)
VALUES (@workspace_id, @collection_id, @name, @source_type, @config, @access_token_encrypted, @created_by)
RETURNING id;

-- name: GetDocSourceForSync :one
SELECT workspace_id, source_type, config, access_token_encrypted
FROM doc_sources
WHERE id = @id;

-- name: UpdateDocSource :exec
UPDATE doc_sources
SET name = COALESCE(sqlc.narg('name'), name),
    config = COALESCE(sqlc.narg('config'), config),
    access_token_encrypted = COALESCE(sqlc.narg('access_token_encrypted'), access_token_encrypted)
WHERE id = @id AND workspace_id = @workspace_id;

-- name: GetDocSource :one
SELECT id, workspace_id, collection_id, name, source_type, config, access_token_encrypted, last_synced_at, created_at
FROM doc_sources
WHERE id = @id AND workspace_id = @workspace_id;

-- name: UpdateDocSourceLastSynced :exec
UPDATE doc_sources
SET last_synced_at = now()
WHERE id = @id;

-- name: DeleteWorkspaceDocsByDocSource :exec
DELETE FROM workspace_docs
WHERE workspace_id = @workspace_id AND doc_source_id = @doc_source_id;

-- name: DeleteDocSource :exec
DELETE FROM doc_sources
WHERE id = @id AND workspace_id = @workspace_id;

-- name: ListWorkspaceDocs :many
SELECT id, workspace_id, slug, title, content_md, source_path, is_local, linked_operation_ids, updated_at
FROM workspace_docs
WHERE workspace_id = @workspace_id
ORDER BY source_path, title;

-- name: ListWorkspaceDocSummaries :many
SELECT id, workspace_id, slug, title, source_path, is_local, updated_at
FROM workspace_docs
WHERE workspace_id = @workspace_id
ORDER BY source_path, title;

-- name: ListWorkspaceDocsByOperation :many
SELECT id, workspace_id, slug, title, content_md, source_path, is_local, linked_operation_ids, updated_at
FROM workspace_docs
WHERE workspace_id = @workspace_id
  AND sqlc.arg(operation_id)::text = ANY(linked_operation_ids)
ORDER BY source_path, title;

-- name: ClearWorkspaceDocSlugConflict :exec
DELETE FROM workspace_docs
WHERE workspace_id = @workspace_id
  AND slug = @slug
  AND source_path <> @source_path
  AND doc_source_id = @doc_source_id;

-- name: UpsertWorkspaceDoc :exec
INSERT INTO workspace_docs (workspace_id, doc_source_id, slug, title, content_md, source_path, is_local, linked_operation_ids, updated_at)
VALUES (@workspace_id, @doc_source_id, @slug, @title, @content_md, @source_path, @is_local, @linked_operation_ids, now())
ON CONFLICT (workspace_id, source_path) WHERE source_path <> '' DO UPDATE SET
    slug = EXCLUDED.slug,
    title = EXCLUDED.title,
    content_md = EXCLUDED.content_md,
    linked_operation_ids = EXCLUDED.linked_operation_ids,
    updated_at = now(),
    doc_source_id = EXCLUDED.doc_source_id,
    is_local = EXCLUDED.is_local;

-- name: CreateWorkspaceDoc :one
INSERT INTO workspace_docs (workspace_id, slug, title, content_md, source_path, is_local, linked_operation_ids, updated_at)
VALUES (@workspace_id, @slug, @title, @content_md, @source_path, true, @linked_operation_ids, now())
RETURNING id, workspace_id, slug, title, content_md, source_path, is_local, linked_operation_ids, updated_at;

-- name: GetWorkspaceDocBySlug :one
SELECT id, workspace_id, slug, title, content_md, source_path, is_local, linked_operation_ids, updated_at
FROM workspace_docs
WHERE workspace_id = @workspace_id AND slug = @slug;

-- name: UpdateWorkspaceDoc :exec
UPDATE workspace_docs
SET title = @title,
    content_md = @content_md,
    linked_operation_ids = @linked_operation_ids,
    updated_at = now()
WHERE workspace_id = @workspace_id AND slug = @slug;

-- name: GetWorkspaceDocByID :one
SELECT id, workspace_id, slug, title, content_md, source_path, is_local, linked_operation_ids, updated_at
FROM workspace_docs
WHERE id = @id AND workspace_id = @workspace_id;

-- name: CreateManualDocLink :one
INSERT INTO manual_doc_links (workspace_doc_id, request_id)
VALUES (@workspace_doc_id, @request_id)
ON CONFLICT (workspace_doc_id, request_id) DO UPDATE SET created_at = manual_doc_links.created_at
RETURNING id, workspace_doc_id, request_id, created_at;

-- name: GetManualDocLink :one
SELECT m.id, m.workspace_doc_id, m.request_id, m.created_at
FROM manual_doc_links m
JOIN workspace_docs d ON d.id = m.workspace_doc_id
WHERE m.id = @id AND d.workspace_id = @workspace_id;

-- name: DeleteManualDocLink :exec
DELETE FROM manual_doc_links m
USING workspace_docs d
WHERE m.id = @id AND m.workspace_doc_id = d.id AND d.workspace_id = @workspace_id;

-- name: ListManualDocLinksByDoc :many
SELECT m.id, m.request_id, r.name, r.method, r.url,
       COALESCE(r.source_operation_id, ''), c.name AS collection_name
FROM manual_doc_links m
JOIN workspace_docs d ON d.id = m.workspace_doc_id
JOIN requests r ON r.id = m.request_id
JOIN collections c ON c.id = r.collection_id
WHERE m.workspace_doc_id = @workspace_doc_id AND d.workspace_id = @workspace_id
ORDER BY c.name, r.sort_order, r.name;

-- name: ListManualDocLinksForRequest :many
SELECT m.id AS link_id, d.id, d.slug, d.title, d.content_md, d.source_path, d.is_local, d.linked_operation_ids, d.updated_at
FROM manual_doc_links m
JOIN workspace_docs d ON d.id = m.workspace_doc_id
WHERE m.request_id = @request_id
ORDER BY d.title;

-- name: ListManualDocLinksByWorkspace :many
SELECT m.id, m.workspace_doc_id, m.request_id, d.slug AS doc_slug, r.name AS request_name
FROM manual_doc_links m
JOIN workspace_docs d ON d.id = m.workspace_doc_id
JOIN requests r ON r.id = m.request_id
JOIN collections c ON c.id = r.collection_id
WHERE d.workspace_id = @workspace_id
ORDER BY d.slug, r.name;

-- name: VerifyRequestInWorkspace :one
SELECT r.id
FROM requests r
JOIN collections c ON c.id = r.collection_id
WHERE r.id = @request_id AND c.workspace_id = @workspace_id;
