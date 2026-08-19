-- name: UpsertDocLinkSuggestion :one
INSERT INTO doc_link_suggestions (workspace_id, workspace_doc_id, request_id, reason, confidence, evidence, status)
VALUES (@workspace_id, @workspace_doc_id, @request_id, @reason, @confidence, @evidence, 'pending')
ON CONFLICT (workspace_doc_id, request_id) DO UPDATE SET
    reason = EXCLUDED.reason,
    confidence = EXCLUDED.confidence,
    evidence = EXCLUDED.evidence,
    status = CASE
        WHEN doc_link_suggestions.status = 'rejected' THEN doc_link_suggestions.status
        ELSE 'pending'
    END,
    reviewed_at = CASE
        WHEN doc_link_suggestions.status = 'rejected' THEN doc_link_suggestions.reviewed_at
        ELSE NULL
    END,
    reviewed_by = CASE
        WHEN doc_link_suggestions.status = 'rejected' THEN doc_link_suggestions.reviewed_by
        ELSE NULL
    END
RETURNING id, workspace_id, workspace_doc_id, request_id, reason, confidence, evidence, status, created_at;

-- name: ListDocLinkSuggestions :many
SELECT s.id, s.workspace_doc_id, s.request_id, s.reason, s.confidence, s.evidence, s.status, s.created_at,
       d.title AS doc_title, d.slug AS doc_slug,
       r.name AS request_name, r.method, r.url,
       COALESCE(r.source_operation_id, ''), c.name AS collection_name
FROM doc_link_suggestions s
JOIN workspace_docs d ON d.id = s.workspace_doc_id
JOIN requests r ON r.id = s.request_id
JOIN collections c ON c.id = r.collection_id
WHERE s.workspace_id = @workspace_id
  AND s.status = @status
ORDER BY
  CASE s.confidence WHEN 'high' THEN 0 WHEN 'medium' THEN 1 ELSE 2 END,
  d.title, r.name;

-- name: ListPendingDocLinkSuggestionsByDoc :many
SELECT s.id, s.request_id, s.reason, s.confidence, s.evidence,
       r.name AS request_name, r.method, r.url,
       COALESCE(r.source_operation_id, ''), c.name AS collection_name
FROM doc_link_suggestions s
JOIN requests r ON r.id = s.request_id
JOIN collections c ON c.id = r.collection_id
WHERE s.workspace_doc_id = @workspace_doc_id
  AND s.workspace_id = @workspace_id
  AND s.status = 'pending'
ORDER BY r.name;

-- name: GetDocLinkSuggestion :one
SELECT s.id, s.workspace_id, s.workspace_doc_id, s.request_id, s.reason, s.confidence, s.status
FROM doc_link_suggestions s
WHERE s.id = @id AND s.workspace_id = @workspace_id;

-- name: UpdateDocLinkSuggestionStatus :exec
UPDATE doc_link_suggestions
SET status = @status,
    reviewed_at = now(),
    reviewed_by = @reviewed_by
WHERE id = @id AND workspace_id = @workspace_id;

-- name: CountPendingDocLinkSuggestions :one
SELECT COUNT(*)::int
FROM doc_link_suggestions
WHERE workspace_id = @workspace_id AND status = 'pending';

-- name: ListDocLinkSuggestionsForAnalyze :many
SELECT id, workspace_doc_id, request_id, status
FROM doc_link_suggestions
WHERE workspace_id = @workspace_id;

-- name: DeleteStalePendingDocLinkSuggestions :exec
DELETE FROM doc_link_suggestions
WHERE workspace_id = @workspace_id
  AND status = 'pending'
  AND id <> ALL(@keep_ids::uuid[]);
