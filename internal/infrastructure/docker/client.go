package docker

import (
	"context"

	"github.com/docker/docker/client"
)

var dockerClientKey = &struct{ name string }{"dockerClient"}

func WithClient(ctx context.Context, cl client.APIClient) (context.Context, error) {
	if cl == nil {
		client, err := client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			return nil, err
		}

		cl = client
	}

	return context.WithValue(ctx, dockerClientKey, cl), nil
}

func ClientFromContext(ctx context.Context) client.APIClient {
	cl, ok := ctx.Value(dockerClientKey).(client.APIClient)
	if !ok {
		panic("Client not found in context!")
	}

	return cl
}
