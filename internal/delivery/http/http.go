package http

import (
	"context"

	"oppossome/serverpouch/internal/delivery/http/openapi"
	"oppossome/serverpouch/internal/domain/usecases"

	"github.com/go-chi/chi/v5"
	"github.com/pkg/errors"
)

type httpImpl struct {
	appCtx   context.Context
	usecases usecases.Usecases
}

var _ openapi.StrictServerInterface = (*httpImpl)(nil)

func New(ctx context.Context) (*chi.Mux, error) {
	httpImpl := &httpImpl{
		appCtx:   ctx,
		usecases: usecases.UsecasesFromContext(ctx),
	}

	handler, err := openapi.New(httpImpl)
	if err != nil {
		return nil, errors.Wrap(err, "failed to instantiate openapi mux")
	}

	return handler, nil
}
