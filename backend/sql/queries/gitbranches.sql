-- name: GetGitBranchCache :one
SELECT branches, fetched_at
FROM git_branch_cache
WHERE workspace_id = @workspace_id AND repo_key = @repo_key;

-- name: UpsertGitBranchCache :exec
INSERT INTO git_branch_cache (workspace_id, repo_key, branches, fetched_at)
VALUES (@workspace_id, @repo_key, @branches, now())
ON CONFLICT (workspace_id, repo_key)
DO UPDATE SET branches = EXCLUDED.branches, fetched_at = now();
