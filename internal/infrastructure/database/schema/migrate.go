package schema

import (
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/lib/pq"

	migrate "github.com/rubenv/sql-migrate"
)

//go:embed migrations/*
var dbMigrations embed.FS

func Migrate(connStr string) error {
	sqlConn, err := sql.Open("postgres", connStr+" sslmode=disable")
	if err != nil {
		fmt.Println("failed to connect to database...")
		return err
	}
	defer sqlConn.Close()

	migrations := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: dbMigrations,
		Root:       "migrations",
	}

	fmt.Printf("migrating database...\n")
	applied, err := migrate.Exec(sqlConn, "postgres", migrations, migrate.Up)

	fmt.Printf("applied %d migrations\n", applied)
	return err
}
