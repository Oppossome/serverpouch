package openapi

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/pkg/errors"

	oapiMiddleware "github.com/oapi-codegen/nethttp-middleware"
)

func New(ssi StrictServerInterface) (*chi.Mux, error) {
	swagger, err := GetSwagger()
	if err != nil {
		return nil, errors.Wrap(err, "Error getting swagger")
	}

	router := chi.NewRouter()
	router.Use(oapiMiddleware.OapiRequestValidator(swagger))
	router.Use(middleware.Logger)

	strictHandler := NewStrictHandler(ssi, nil)
	HandlerFromMux(strictHandler, router)

	return router, nil
}
