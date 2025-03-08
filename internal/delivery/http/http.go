package http

import (
	"context"
	"net/http"

	"oppossome/serverpouch/internal/delivery/http/openapi"
	"oppossome/serverpouch/internal/domain/usecases"

	"github.com/pkg/errors"
)

type httpImpl struct {
	usecases usecases.Usecases
}

var _ openapi.StrictServerInterface = (*httpImpl)(nil)

func New(ctx context.Context, httpURL string) (*http.Server, error) {
	httpImpl := &httpImpl{
		usecases: usecases.UsecasesFromContext(ctx),
	}

	handler, err := openapi.New(httpImpl)
	if err != nil {
		return &http.Server{}, errors.Wrap(err, "failed to instantiate openapi mux")
	}

	server := &http.Server{
		Handler: handler,
		Addr:    httpURL,
	}

	return server, nil
}
