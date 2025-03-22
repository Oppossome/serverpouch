// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: servers.sql

package schema

import (
	"context"

	"github.com/google/uuid"
)

const createServer = `-- name: CreateServer :one
INSERT INTO servers (type, config) 
VALUES ($1, $2)
RETURNING id, type, config, created_at, updated_at
`

type CreateServerParams struct {
	Type   string
	Config []byte
}

func (q *Queries) CreateServer(ctx context.Context, arg CreateServerParams) (Server, error) {
	row := q.db.QueryRow(ctx, createServer, arg.Type, arg.Config)
	var i Server
	err := row.Scan(
		&i.ID,
		&i.Type,
		&i.Config,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getServer = `-- name: GetServer :one
SELECT id, type, config, created_at, updated_at FROM servers
WHERE id = $1 LIMIT 1
`

func (q *Queries) GetServer(ctx context.Context, id uuid.UUID) (Server, error) {
	row := q.db.QueryRow(ctx, getServer, id)
	var i Server
	err := row.Scan(
		&i.ID,
		&i.Type,
		&i.Config,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getServers = `-- name: GetServers :many
SELECT id, type, config, created_at, updated_at FROM servers
ORDER BY created_at DESC
`

func (q *Queries) GetServers(ctx context.Context) ([]Server, error) {
	rows, err := q.db.Query(ctx, getServers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Server
	for rows.Next() {
		var i Server
		if err := rows.Scan(
			&i.ID,
			&i.Type,
			&i.Config,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateServer = `-- name: UpdateServer :one
UPDATE servers SET 
  config = $2, 
  updated_at = CURRENT_TIMESTAMP
WHERE id = $1 
RETURNING id, type, config, created_at, updated_at
`

type UpdateServerParams struct {
	ID     uuid.UUID
	Config []byte
}

func (q *Queries) UpdateServer(ctx context.Context, arg UpdateServerParams) (Server, error) {
	row := q.db.QueryRow(ctx, updateServer, arg.ID, arg.Config)
	var i Server
	err := row.Scan(
		&i.ID,
		&i.Type,
		&i.Config,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}
