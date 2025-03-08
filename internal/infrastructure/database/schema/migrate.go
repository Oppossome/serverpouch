package schema

import (
	"context"
	"database/sql"
	"embed"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog"

	migrate "github.com/rubenv/sql-migrate"
)

//go:embed migrations/*
var dbMigrations embed.FS

func Migrate(ctx context.Context, connStr string) error {
	sqlConn, err := sql.Open("postgres", connStr)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("failed to connect to database")
		return err
	}
	defer sqlConn.Close()

	migrations := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: dbMigrations,
		Root:       "migrations",
	}

	zerolog.Ctx(ctx).Debug().Msg("migrating database...")
	applied, err := migrate.Exec(sqlConn, "postgres", migrations, migrate.Up)

	zerolog.Ctx(ctx).Info().Msgf("applied %d migrations", applied)
	return err
}
