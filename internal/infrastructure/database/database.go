package database

import (
	"context"
	"testing"

	"oppossome/serverpouch/internal/domain/server"
	"oppossome/serverpouch/internal/infrastructure/database/schema"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/assert"
)

type Database interface {
	GetServer(context.Context, uuid.UUID) (server.ServerInstanceConfig, error)
	ListServers(context.Context) ([]server.ServerInstanceConfig, error)
	UpdateServer(context.Context, uuid.UUID, server.ServerInstanceConfig) (server.ServerInstanceConfig, error)
	CreateServer(context.Context, server.ServerInstanceConfig) (server.ServerInstanceConfig, error)
}

type databaseImpl struct {
	conn    *pgx.Conn
	queries *schema.Queries
}

var _ Database = (*databaseImpl)(nil)

func New(ctx context.Context, connStr string) (*databaseImpl, error) {
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("failed to connect to database")
		return nil, errors.Wrap(err, "failed to connect to database")
	}

	database := &databaseImpl{
		conn:    conn,
		queries: schema.New(conn),
	}

	zerolog.Ctx(ctx).Debug().Msg("connected to database")
	return database, nil
}

func NewTestDatabase(t *testing.T) (*schema.Queries, *databaseImpl) {
	connStr := schema.SetupTestContainer(t)

	_, err := schema.Migrate(t.Context(), connStr, migrate.Up)
	assert.NoError(t, err)

	dbImpl, err := New(t.Context(), connStr)
	assert.NoError(t, err)

	t.Cleanup(func() {
		err := dbImpl.conn.Close(t.Context())
		assert.NoError(t, err, "failed to close database connection")
	})

	return dbImpl.queries, dbImpl
}
