-- name: GetServerConfig :one
SELECT * FROM server_configs
WHERE id = $1 LIMIT 1;

-- name: GetServerConfigs :many
SELECT * FROM server_configs
ORDER BY created_at DESC;

-- name: CreateServerConfig :one
INSERT INTO server_configs (type, config) 
VALUES ($1, $2)
RETURNING *;

-- name: UpdateServerConfig :one
UPDATE server_configs SET 
  config = $2, 
  updated_at = CURRENT_TIMESTAMP
WHERE id = $1 
RETURNING *;