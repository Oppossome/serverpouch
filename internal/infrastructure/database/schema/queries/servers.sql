-- name: GetServer :one
SELECT * FROM servers
WHERE id = $1 LIMIT 1;

-- name: GetServers :many
SELECT * FROM servers
ORDER BY created_at DESC;

-- name: CreateServer :one
INSERT INTO servers (type, config) 
VALUES ($1, $2)
RETURNING *;

-- name: UpdateServer :one
UPDATE servers SET 
  config = $2, 
  updated_at = CURRENT_TIMESTAMP
WHERE id = $1 
RETURNING *;