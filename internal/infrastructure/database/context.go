package database

import (
	"context"
)

var databaseKey = &struct{ name string }{"databaseClient"}

func WithDatabase(ctx context.Context, db Database) context.Context {
	return context.WithValue(ctx, databaseKey, db)
}

func DatabaseFromContext(ctx context.Context) Database {
	db, ok := ctx.Value(databaseKey).(Database)
	if !ok {
		panic("Client not found in context!")
	}

	return db
}
