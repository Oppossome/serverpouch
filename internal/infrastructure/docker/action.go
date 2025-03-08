package docker

import (
	"fmt"

	"oppossome/serverpouch/internal/domain/server"

	"github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// MARK: Start

func (dsi *dockerServerInstance) Start() error {
	actionDone, err := dsi.lifecycleAction(dsi.ctx)
	if err != nil {
		return errors.Wrap(err, "failed to acquire start action")
	}
	defer actionDone()

	status := dsi.Status()
	if status != server.ServerInstanceStatusIdle {
		msg := fmt.Sprintf("Start is an invalid action for status %s", status)
		dsi.events.TerminalOut.Dispatch(msg)
		return errors.New(msg)
	}

	dsi.mu.RLock()
	containerID := dsi.containerID
	dsi.mu.RUnlock()

	dsi.setStatus(server.ServerInstanceStatusStarting)

	err = dsi.client.ContainerStart(dsi.ctx, containerID, container.StartOptions{})
	if err != nil {
		zerolog.Ctx(dsi.ctx).Error().Msgf("Unable to start container: %s", err)
		dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Unable to start container: %s", err))
	}

	return nil
}

// MARK: Stop

func (dsi *dockerServerInstance) Stop() error {
	actionDone, err := dsi.lifecycleAction(dsi.ctx)
	if err != nil {
		return errors.Wrap(err, "failed to acquire stop action")
	}
	defer actionDone()

	status := dsi.Status()
	if status != server.ServerInstanceStatusRunning {
		msg := fmt.Sprintf("Stop is an invalid action for status %s", status)
		dsi.events.TerminalOut.Dispatch(msg)
		return errors.New(msg)
	}

	dsi.mu.RLock()
	containerID := dsi.containerID
	dsi.mu.RUnlock()

	dsi.setStatus(server.ServerInstanceStatusStopping)

	err = dsi.client.ContainerStop(dsi.ctx, containerID, container.StopOptions{})
	if err != nil {
		zerolog.Ctx(dsi.ctx).Error().Msgf("Unable to stop container: %s", err)
		dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Unable to stop container: %s", err))
	}

	return nil
}

// MARK: Kill

func (dsi *dockerServerInstance) Kill() error {
	actionDone, err := dsi.lifecycleAction(dsi.ctx)
	if err != nil {
		return errors.Wrap(err, "failed to acquire kill action")
	}
	defer actionDone()

	status := dsi.Status()
	if status != server.ServerInstanceStatusRunning {
		msg := fmt.Sprintf("Kill is an invalid action for status %s", status)
		dsi.events.TerminalOut.Dispatch(msg)
		return errors.New(msg)
	}

	dsi.mu.RLock()
	containerID := dsi.containerID
	dsi.mu.RUnlock()

	dsi.setStatus(server.ServerInstanceStatusStopping)

	err = dsi.client.ContainerKill(dsi.ctx, containerID, "SIGKILL")
	if err != nil {
		zerolog.Ctx(dsi.ctx).Error().Msgf("Unable to kill container: %s", err)
		dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Unable to kill container: %s", err))
	}

	return nil
}
