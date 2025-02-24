package docker

import (
	"context"

	"github.com/docker/docker/client"
)

var dockerClientKey = &struct{ name string }{"dockerClient"}

func WithClient(ctx context.Context, cl client.APIClient) context.Context {
	return context.WithValue(ctx, dockerClientKey, cl)
}

func ClientFromContext(ctx context.Context) client.APIClient {
	cl, ok := ctx.Value(dockerClientKey).(client.APIClient)
	if !ok {
		panic("Client not found in context!")
	}

	return cl
}
