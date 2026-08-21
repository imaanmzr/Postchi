-- name: CreateUser :one
INSERT INTO users (email, password_hash, display_name)
VALUES (@email, @password_hash, @display_name)
RETURNING id;

-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES (@user_id, @token_hash, @expires_at);

-- name: GetUserByEmail :one
SELECT id, password_hash, display_name
FROM users
WHERE email = @email;

-- name: GetRefreshTokenWithUser :one
SELECT rt.user_id, u.email, rt.expires_at
FROM refresh_tokens rt
JOIN users u ON u.id = rt.user_id
WHERE rt.token_hash = @token_hash;

-- name: DeleteRefreshTokenByHash :exec
DELETE FROM refresh_tokens
WHERE token_hash = @token_hash;

-- name: GetUserByID :one
SELECT email, display_name
FROM users
WHERE id = @id;

-- name: GetUserAuthByID :one
SELECT password_hash, auth_provider
FROM users
WHERE id = @id;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = @password_hash, updated_at = now()
WHERE id = @id;

-- name: DeleteRefreshTokensByUserID :exec
DELETE FROM refresh_tokens
WHERE user_id = @user_id;

-- name: GetWorkspaceMemberRole :one
SELECT role::text
FROM workspace_members
WHERE workspace_id = @workspace_id AND user_id = @user_id;

-- name: UserExistsByEmail :one
SELECT EXISTS(SELECT 1 FROM users WHERE email = @email);
