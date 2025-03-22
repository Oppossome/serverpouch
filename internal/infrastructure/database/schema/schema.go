package schema

import (
	"context"
	"database/sql"
	"embed"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	migrate "github.com/rubenv/sql-migrate"
)

//go:embed migrations/*
var dbMigrations embed.FS

// Migrate executes database migrations in the specified direction using embedded SQL files.
// It returns the number of migrations applied and any error encountered.
// The migrations are loaded from the embedded "migrations" directory.
func Migrate(ctx context.Context, connStr string, direction migrate.MigrationDirection) (int, error) {
	sqlConn, err := sql.Open("postgres", connStr)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("failed to connect to database")
		return 0, err
	}
	defer sqlConn.Close()

	migrations := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: dbMigrations,
		Root:       "migrations",
	}

	zerolog.Ctx(ctx).Debug().Msg("migrating database...")
	applied, err := migrate.Exec(sqlConn, "postgres", migrations, direction)

	zerolog.Ctx(ctx).Info().Msgf("applied %d migrations", applied)
	return applied, err
}

// SetupTestContainer creates a temporary Postgres container for testing.
// It configures the container with default credentials and waits for it to be ready.
// The container is automatically cleaned up when the test completes.
// Returns a connection string for accessing the test database.
func SetupTestContainer(t *testing.T) string {
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
	assert.NoError(t, err)

	t.Cleanup(func() {
		err := postgresContainer.Terminate(context.Background())
		assert.NoError(t, err, "failed to terminate postgres container")
	})

	connStr, err := postgresContainer.ConnectionString(t.Context(), "sslmode=disable")
	assert.NoError(t, err)

	return connStr
}
