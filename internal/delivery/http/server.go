package http

import (
	"context"

	"oppossome/serverpouch/internal/delivery/http/openapi"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// Create a new server
// (POST /api/servers)
func (hi *httpImpl) CreateServer(ctx context.Context, request openapi.CreateServerRequestObject) (openapi.CreateServerResponseObject, error) {
	instCfg, err := openapi.OAPIToConfig(request.Body.Config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode openapi server")
	}

	// Utilize the application context so it isn't cancelled when the request ends.
	inst, err := hi.usecases.CreateServer(hi.appCtx, instCfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create server")
	}

	oInst, err := openapi.ServerToOAPI(inst)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode openapi server")
	}

	return openapi.CreateServer201JSONResponse{Server: *oInst}, nil
}

// Get a server by ID
// (GET /api/servers/{id})
func (hi *httpImpl) GetServer(ctx context.Context, request openapi.GetServerRequestObject) (openapi.GetServerResponseObject, error) {
	inst, err := hi.usecases.GetServer(ctx, request.Id)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msgf("Failed to get server of id %s", request.Id)
		return openapi.GetServer404Response{}, nil
	}

	oInst, err := openapi.ServerToOAPI(inst)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode openapi server")
	}

	return openapi.GetServer200JSONResponse{Server: *oInst}, nil
}

// List all servers
// (GET /api/servers)
func (hi *httpImpl) ListServers(ctx context.Context, request openapi.ListServersRequestObject) (openapi.ListServersResponseObject, error) {
	insts := hi.usecases.ListServers(ctx)

	oInsts := make([]openapi.Server, len(insts))
	for idx, inst := range insts {
		oInst, err := openapi.ServerToOAPI(inst)
		if err != nil {
			return nil, errors.Wrap(err, "failed to encode openapi server")
		}

		oInsts[idx] = *oInst
	}

	return openapi.ListServers200JSONResponse{Servers: oInsts}, nil
}
