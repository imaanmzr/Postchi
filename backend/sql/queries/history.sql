-- name: ListHistoryByWorkspace :many
SELECT h.id, h.workspace_id, h.request_id, h.snapshot, h.response, h.test_results,
       h.executed_by, u.display_name, u.email, h.executed_at, h.duration_ms, h.status_code
FROM history h
JOIN users u ON u.id = h.executed_by
WHERE h.workspace_id = @workspace_id
ORDER BY h.executed_at DESC
LIMIT 100;

-- name: ListHistoryByWorkspaceAndRequest :many
SELECT h.id, h.workspace_id, h.request_id, h.snapshot, h.response, h.test_results,
       h.executed_by, u.display_name, u.email, h.executed_at, h.duration_ms, h.status_code
FROM history h
JOIN users u ON u.id = h.executed_by
WHERE h.workspace_id = @workspace_id
  AND h.request_id = @request_id
ORDER BY h.executed_at DESC
LIMIT 100;

-- name: GetHistoryForShareSnapshot :one
SELECT h.snapshot, h.response, h.test_results, h.duration_ms, h.status_code, h.executed_at,
       h.executed_by, u.display_name, u.email
FROM history h
JOIN users u ON u.id = h.executed_by
WHERE h.id = @id AND h.workspace_id = @workspace_id;
