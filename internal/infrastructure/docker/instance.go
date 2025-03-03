package docker

import (
	"context"
	"sync"

	"oppossome/serverpouch/internal/domain/server"

	"github.com/docker/docker/client"
	"github.com/rs/zerolog"
)

var _ server.ServerInstance = (*dockerServerInstance)(nil)

type dockerServerInstance struct {
	ctx           context.Context
	ctxCancel     context.CancelFunc
	ctxCancelDone chan struct{}

	client  client.APIClient
	options *DockerServerInstanceOptions

	actionChan chan *dockerServerInstanceAction
	events     *server.ServerInstanceEvents

	mu          sync.RWMutex
	containerID string
	status      server.ServerInstanceStatus
}

type dockerServerInstanceAction struct {
	action server.ServerInstanceAction
	done   chan struct{}
}

func (d *dockerServerInstance) Config() server.ServerInstanceConfig {
	return d.options
}

func (d *dockerServerInstance) Action(action server.ServerInstanceAction) {
	instAction := &dockerServerInstanceAction{
		action: action,
		done:   make(chan struct{}),
	}

	select {
	case <-d.ctx.Done():
	case d.actionChan <- instAction:
		<-instAction.done
	}
}

func (d *dockerServerInstance) Status() server.ServerInstanceStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.status
}

func (d *dockerServerInstance) setStatus(status server.ServerInstanceStatus) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.status == status {
		return
	}

	d.status = status
	d.events.Status.Dispatch(status)
}

func (d *dockerServerInstance) Events() *server.ServerInstanceEvents {
	return d.events
}

func (d *dockerServerInstance) Close() {
	d.ctxCancel()
	<-d.ctxCancelDone
}

// Creates a new instance of the dockerServerInstance and kicks off its lifecycle.
func NewDockerServerInstance(ctx context.Context, options *DockerServerInstanceOptions) *dockerServerInstance {
	ctx, ctxCancel := context.WithCancel(ctx)

	// Add the instance ID to the context logger.
	ctx = zerolog.Ctx(ctx).With().
		Stringer("instance_id", options.InstanceID).Logger().
		WithContext(ctx)

	serverInstance := &dockerServerInstance{
		ctx:           ctx,
		ctxCancel:     ctxCancel,
		ctxCancelDone: make(chan struct{}),

		client:  ClientFromContext(ctx),
		options: options,

		actionChan: make(chan *dockerServerInstanceAction),
		events:     server.NewServerInstanceEvents(),

		mu:          sync.RWMutex{},
		containerID: "",
		status:      server.ServerInstanceStatusInitializing,
	}

	go serverInstance.lifecycle()
	go serverInstance.Action(server.ServerInstanceActionStart)

	return serverInstance
}
