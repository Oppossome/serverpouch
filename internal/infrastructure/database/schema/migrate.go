package schema

import (
	"database/sql"
	"embed"

	migrate "github.com/rubenv/sql-migrate"
)

//go:embed migrations/*
var dbMigrations embed.FS

func Migrate(db *sql.DB) error {
	migrations := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: dbMigrations,
		Root:       "migrations",
	}

	_, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
	return err
}
