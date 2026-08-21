-- name: ListWorkspaceDiagrams :many
SELECT id, workspace_id, slug, title, updated_at
FROM workspace_diagrams
WHERE workspace_id = @workspace_id
ORDER BY title ASC;

-- name: GetWorkspaceDiagramBySlug :one
SELECT id, workspace_id, slug, title, content, created_by, created_at, updated_at
FROM workspace_diagrams
WHERE workspace_id = @workspace_id AND slug = @slug;

-- name: CreateWorkspaceDiagram :one
INSERT INTO workspace_diagrams (workspace_id, slug, title, content, created_by)
VALUES (@workspace_id, @slug, @title, @content, @created_by)
RETURNING id, workspace_id, slug, title, content, created_by, created_at, updated_at;

-- name: UpdateWorkspaceDiagram :one
UPDATE workspace_diagrams
SET title = COALESCE(sqlc.narg('title'), title),
    content = COALESCE(sqlc.narg('content'), content),
    updated_at = now()
WHERE workspace_id = @workspace_id AND slug = @slug
RETURNING id, workspace_id, slug, title, content, created_by, created_at, updated_at;

-- name: DeleteWorkspaceDiagram :exec
DELETE FROM workspace_diagrams
WHERE workspace_id = @workspace_id AND slug = @slug;

-- name: WorkspaceDiagramSlugExists :one
SELECT EXISTS(
    SELECT 1 FROM workspace_diagrams
    WHERE workspace_id = @workspace_id AND slug = @slug
) AS exists;

-- name: ListDiagramRequestLinks :many
SELECT r.id, r.name, r.method, r.url, c.workspace_id, w.name AS workspace_name
FROM diagram_request_links drl
JOIN requests r ON r.id = drl.request_id
JOIN collections c ON c.id = r.collection_id
JOIN workspaces w ON w.id = c.workspace_id
WHERE drl.diagram_id = @diagram_id;

-- name: AddDiagramRequestLink :exec
INSERT INTO diagram_request_links (diagram_id, request_id)
VALUES (@diagram_id, @request_id)
ON CONFLICT DO NOTHING;

-- name: RemoveDiagramRequestLink :exec
DELETE FROM diagram_request_links
WHERE diagram_id = @diagram_id AND request_id = @request_id;
