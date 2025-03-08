package http

import (
	"context"

	"oppossome/serverpouch/internal/delivery/http/openapi"
)

// Create a new server
// (POST /server)
func (hi *httpImpl) CreateServer(ctx context.Context, request openapi.CreateServerRequestObject) (openapi.CreateServerResponseObject, error) {
	panic("Unimplemented")
}

// Get a server by ID
// (GET /server/{id})
func (hi *httpImpl) GetServer(ctx context.Context, request openapi.GetServerRequestObject) (openapi.GetServerResponseObject, error) {
	panic("Unimplemented")
}

// List all servers
// (GET /servers)
func (hi *httpImpl) ListServers(ctx context.Context, request openapi.ListServersRequestObject) (openapi.ListServersResponseObject, error) {
	panic("Unimplemented")
}
