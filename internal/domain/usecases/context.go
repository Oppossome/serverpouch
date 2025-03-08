package usecases

import "context"

var usecasesKey = &struct{ name string }{"usecases"}

func WithUsecases(ctx context.Context, usecases Usecases) context.Context {
	return context.WithValue(ctx, usecasesKey, usecases)
}

func UsecasesFromContext(ctx context.Context) Usecases {
	db, ok := ctx.Value(usecasesKey).(Usecases)
	if !ok {
		panic("Usecases not found in context!")
	}

	return db
}
