-- name: GetUserByID :one
SELECT id, email, name, provider, created_at
FROM users
WHERE id = ?
LIMIT 1;

-- name: UpsertUser :one
INSERT INTO users (id, email, name, provider)
VALUES (?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    email = excluded.email,
    name  = excluded.name
RETURNING id, email, name, provider, created_at;
