package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"oppossome/serverpouch/internal/domain/server"
	"oppossome/serverpouch/internal/infrastructure/database/schema"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type Database interface {
	GetServerConfig(context.Context, uuid.UUID) (server.ServerInstanceConfig, error)
	ListServerConfigs(context.Context) ([]server.ServerInstanceConfig, error)
	UpdateServerConfig(context.Context, server.ServerInstanceConfig) (server.ServerInstanceConfig, error)
	CreateServerConfig(context.Context, server.ServerInstanceConfig) (server.ServerInstanceConfig, error)
}

type databaseImpl struct {
	conn    *pgx.Conn
	queries *schema.Queries
}

var _ Database = (*databaseImpl)(nil)

func New(context context.Context, connStr string) (*databaseImpl, error) {
	conn, err := pgx.Connect(context, connStr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}

	database := &databaseImpl{
		conn:    conn,
		queries: schema.New(conn),
	}

	return database, nil
}

func NewTestDatabase(t *testing.T) (*schema.Queries, *databaseImpl, error) {
	postgresContainer, err := postgres.Run(
		t.Context(),
		"postgres:16-alpine",
		postgres.WithUsername("username"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	t.Cleanup(func() {
		if err := postgresContainer.Terminate(context.Background()); err != nil {
			t.Errorf("failed to terminate postgres container: %v", err)
		}
	})

	connStr, err := postgresContainer.ConnectionString(t.Context())
	if err != nil {
		return nil, nil, err
	}

	sqlConn, err := sql.Open("postgres", connStr+" sslmode=disable")
	if err != nil {
		return nil, nil, err
	}
	defer sqlConn.Close()

	err = schema.Migrate(sqlConn)
	if err != nil {
		return nil, nil, err
	}

	dbImpl, err := New(t.Context(), connStr)
	if err != nil {
		return nil, nil, err
	}

	t.Cleanup(func() {
		if err := dbImpl.conn.Close(t.Context()); err != nil {
			t.Errorf("failed to close database connection: %v", err)
		}
	})

	return dbImpl.queries, dbImpl, nil
}
