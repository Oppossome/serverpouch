package docker

import (
	"context"
	"sync"

	"oppossome/serverpouch/internal/domain/server"

	"github.com/docker/docker/client"
	"github.com/rs/zerolog"
)

type dockerServerInstance struct {
	ctx           context.Context
	ctxCancel     context.CancelFunc
	ctxCancelDone chan struct{}

	client  client.APIClient
	events  *server.ServerInstanceEvents
	options *DockerServerInstanceOptions

	actionChan chan chan struct{}

	mu          sync.RWMutex
	containerID string
	status      server.ServerInstanceStatus
}

func (dsi *dockerServerInstance) Config() server.ServerInstanceConfig {
	return dsi.options
}

func (dsi *dockerServerInstance) Status() server.ServerInstanceStatus {
	dsi.mu.RLock()
	defer dsi.mu.RUnlock()

	return dsi.status
}

func (dsi *dockerServerInstance) setStatus(status server.ServerInstanceStatus) {
	dsi.mu.Lock()
	defer dsi.mu.Unlock()

	dsi.status = status
	dsi.events.Status.Dispatch(status)
}

func (dsi *dockerServerInstance) Events() *server.ServerInstanceEvents {
	return dsi.events
}

func (dsi *dockerServerInstance) Close() {
	dsi.ctxCancel()
	<-dsi.ctxCancelDone
}

func NewInstance(ctx context.Context, options *DockerServerInstanceOptions) *dockerServerInstance {
	ctx, ctxCancel := context.WithCancel(ctx)
	ctx = zerolog.Ctx(ctx).With().Stringer("id", options.ID()).Logger().WithContext(ctx)

	instance := &dockerServerInstance{
		ctx:           ctx,
		ctxCancel:     ctxCancel,
		ctxCancelDone: make(chan struct{}),

		client:  ClientFromContext(ctx),
		events:  server.NewServerInstanceEvents(),
		options: options,

		actionChan: make(chan chan struct{}),

		mu:          sync.RWMutex{},
		containerID: "",
		status:      server.ServerInstanceStatusInitializing,
	}

	go instance.lifecycle()
	go func() {
		containerID, err := instance.lifecycleInit(ctx)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("Failed to initialize instance")
			instance.setStatus(server.ServerInstanceStatusErrored)
			return
		}

		instance.mu.Lock()
		instance.containerID = containerID
		instance.mu.Unlock()

		instance.lifecycleInitAttach()
	}()

	return instance
}
