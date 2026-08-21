-- name: ListTestCases :many
SELECT id, workspace_id, title, description, sort_order, created_by, created_at, updated_at
FROM test_cases
WHERE workspace_id = @workspace_id
ORDER BY sort_order ASC, title ASC;

-- name: GetTestCase :one
SELECT id, workspace_id, title, description, sort_order, created_by, created_at, updated_at
FROM test_cases
WHERE id = @id AND workspace_id = @workspace_id;

-- name: CreateTestCase :one
INSERT INTO test_cases (workspace_id, title, description, sort_order, created_by)
VALUES (@workspace_id, @title, @description, @sort_order, @created_by)
RETURNING id, workspace_id, title, description, sort_order, created_by, created_at, updated_at;

-- name: UpdateTestCase :one
UPDATE test_cases
SET title = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    sort_order = COALESCE(sqlc.narg('sort_order'), sort_order),
    updated_at = now()
WHERE id = @id AND workspace_id = @workspace_id
RETURNING id, workspace_id, title, description, sort_order, created_by, created_at, updated_at;

-- name: DeleteTestCase :exec
DELETE FROM test_cases
WHERE id = @id AND workspace_id = @workspace_id;

-- name: ListTestCaseRequestLinks :many
SELECT r.id, r.name, r.method, r.url, c.workspace_id, w.name AS workspace_name
FROM test_case_request_links tcl
JOIN requests r ON r.id = tcl.request_id
JOIN collections c ON c.id = r.collection_id
JOIN workspaces w ON w.id = c.workspace_id
WHERE tcl.test_case_id = @test_case_id;

-- name: AddTestCaseRequestLink :exec
INSERT INTO test_case_request_links (test_case_id, request_id)
VALUES (@test_case_id, @request_id)
ON CONFLICT DO NOTHING;

-- name: RemoveTestCaseRequestLink :exec
DELETE FROM test_case_request_links
WHERE test_case_id = @test_case_id AND request_id = @request_id;

-- name: VerifyTestCaseInWorkspace :one
SELECT EXISTS(
    SELECT 1 FROM test_cases
    WHERE id = @id AND workspace_id = @workspace_id
) AS exists;
