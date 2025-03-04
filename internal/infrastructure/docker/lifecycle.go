package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"oppossome/serverpouch/internal/domain/server"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// MARK: lifecycle

// Begins the dockerServerInstance lifecycle loop.
func (dsi *dockerServerInstance) lifecycle() {
	defer func() {
		dsi.events.Status.Close()
		dsi.ctxCancelDone <- struct{}{}
	}()

	// actionChan is used to listen for a new action and is nil whilst we're processing an action.
	actionChan := dsi.actionChan
	var activeAction *dockerServerInstanceAction
	var activeActionDone chan struct{}

	for {
		select {
		case <-dsi.ctx.Done():
			// If we're in the middle of an action, see it through.
			if activeActionDone != nil {
				<-activeActionDone
				activeAction.done <- struct{}{}
			}

			return

		// Handle lifecycleAction completion
		case <-activeActionDone:
			activeAction.done <- struct{}{}
			actionChan = dsi.actionChan
			activeAction = nil
			activeActionDone = nil

		// Handle lifecycleActions
		case action := <-actionChan:
			actionChan = nil
			activeAction = action
			activeActionDone = make(chan struct{})

			go dsi.lifecycleAction(dsi.ctx, action.action, activeActionDone)

		// Passively update the lifecycle status
		case <-time.After(time.Second * 30):
			// Because actions can perform changes to the status, don't do anything that may interrupt their song and dance
			if activeAction != nil {
				continue
			}

			go dsi.lifecycleActionUpdateStatus()
		}
	}
}

// MARK: lifecycleAction

// lifecycleAction is used to perform the associated lifecycleActions
func (dsi *dockerServerInstance) lifecycleAction(ctx context.Context, action server.ServerInstanceAction, activeActionDone chan<- struct{}) {
	defer func() {
		dsi.lifecycleActionUpdateStatus()
		activeActionDone <- struct{}{}
	}()

	dsi.mu.RLock()
	status := dsi.status
	containerID := dsi.containerID
	dsi.mu.RUnlock()

	ctx = zerolog.Ctx(ctx).With().
		Str("action", string(action)).
		Str("status", string(status)).
		Logger().WithContext(ctx)

	switch {
	case containerID == "":
		if action != server.ServerInstanceActionStart {
			zerolog.Ctx(ctx).Error().Msgf("Invalid %s action: %s", server.ServerInstanceStatusInitializing, action)
			dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Invalid %s action: %s", server.ServerInstanceStatusInitializing, action))
			return
		}

		containerID, err := dsi.lifecycleInit(ctx)
		if err != nil {
			dsi.setStatus(server.ServerInstanceStatusErrored)
			dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Unable to get container: %s", err))
			zerolog.Ctx(ctx).Error().Msgf("Unable to get container: %s", err)
			return
		}

		dsi.mu.Lock()
		dsi.containerID = containerID
		dsi.mu.Unlock()

		go dsi.lifecycleInitAttach()

	case action == server.ServerInstanceActionStart:
		if status != server.ServerInstanceStatusIdle {
			zerolog.Ctx(ctx).Error().Msgf("Invalid %s action: %s", status, action)
			dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Invalid %s action: %s", status, action))
			return
		}

		dsi.setStatus(server.ServerInstanceStatusStarting)

		err := dsi.client.ContainerStart(ctx, containerID, container.StartOptions{})
		if err != nil {
			zerolog.Ctx(ctx).Error().Msgf("Unable to start container: %s", err)
			dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Unable to start container: %s", err))
		}

	case action == server.ServerInstanceActionStop:
		if status != server.ServerInstanceStatusRunning {
			zerolog.Ctx(ctx).Error().Msgf("Invalid %s action: %s", status, action)
			dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Invalid %s action: %s", status, action))
			return
		}

		dsi.setStatus(server.ServerInstanceStatusStopping)

		err := dsi.client.ContainerStop(ctx, dsi.containerID, container.StopOptions{})
		if err != nil {
			dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Unable to stop container: %s", err))
		}

	case action == server.ServerInstanceActionKill:
		if status != server.ServerInstanceStatusRunning {
			zerolog.Ctx(ctx).Error().Msgf("Invalid %s action: %s", status, action)
			dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Invalid %s action: %s", status, action))
			return
		}

		dsi.setStatus(server.ServerInstanceStatusStopping)

		err := dsi.client.ContainerKill(ctx, dsi.containerID, "SIGTERM")
		if err != nil {
			zerolog.Ctx(ctx).Error().Msgf("Unable to kill container: %s", err)
			dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Unable to kill container: %s", err))
		}
	}
}

// MARK: lifecycleActionUpdateStatus

// lifecycleActionUpdateStatus is responsible for determining the
// status of the container.
func (dsi *dockerServerInstance) lifecycleActionUpdateStatus() {
	dsi.mu.RLock()
	containerID := dsi.containerID
	dsi.mu.RUnlock()

	if containerID == "" {
		dsi.setStatus(server.ServerInstanceStatusInitializing)
		return
	}

	inspect, err := dsi.client.ContainerInspect(dsi.ctx, dsi.containerID)
	if err != nil {
		zerolog.Ctx(dsi.ctx).Error().Msgf("Unable to inspect container: %s", err)
		dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Unable to inspect container: %s", err))
		return
	}

	switch {
	case inspect.State.Status == "created":
		fallthrough
	case inspect.State.Status == "exited":
		dsi.setStatus(server.ServerInstanceStatusIdle)
	case inspect.State.Status == "running":
		dsi.setStatus(server.ServerInstanceStatusRunning)
	default:
		zerolog.Ctx(dsi.ctx).Error().Msgf("Unknown docker status: %s", inspect.State.Status)
		dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Unknown docker status: %s", inspect.State.Status))
		dsi.setStatus(server.ServerInstanceStatusErrored)
	}
}

// MARK: lifecycleInit

type dockerEvent struct {
	Status         string `json:"status"`
	Error          string `json:"error"`
	Progress       string `json:"progress"`
	ProgressDetail struct {
		Current int `json:"current"`
		Total   int `json:"total"`
	} `json:"progressDetail"`
}

func (dsi *dockerServerInstance) lifecycleInit(ctx context.Context) (string, error) {
	containers, err := dsi.client.ContainerList(dsi.ctx, container.ListOptions{All: true})
	if err != nil {
		zerolog.Ctx(ctx).Error().Msg("Unable to list containers")
		return "", errors.Wrap(err, "Unable to list containers")
	}

	// Check if we have the container already.
	for _, container := range containers {
		if slices.Contains(container.Names, "/"+dsi.options.InstanceID.String()) {
			if container.Image != dsi.options.Image {
				zerolog.Ctx(ctx).Error().Msgf("Found non-matching container image: %s", container.Image)
				return "", errors.Wrapf(err, "Found non-matching container image: %s", container.Image)
			}

			zerolog.Ctx(ctx).Info().Msgf("Found container \"%s\"", dsi.options.InstanceID)
			return container.ID, nil
		}
	}

	// Check if we have the image already.
	images, err := dsi.client.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		return "", errors.Wrap(err, "Unable to list images")
	}

	foundImage := false
	for _, image := range images {
		name, ok := image.Labels["org.opencontainers.image.ref.name"]
		if ok && name == dsi.options.Image {
			zerolog.Ctx(ctx).Info().Msgf("Found image \"%s\"", dsi.options.Image)
			foundImage = true
			break
		}
	}

	// Since we couldn't find the image, we'll pull it.
	if !foundImage {
		zerolog.Ctx(ctx).Info().Msgf("Pulling image \"%s\"", dsi.options.Image)
		dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Pulling image \"%s\"", dsi.options.Image))
		reader, err := dsi.client.ImagePull(ctx, dsi.options.Image, image.PullOptions{})
		if err != nil {
			zerolog.Ctx(ctx).Error().Msgf("Failed to pull image \"%s\"", dsi.options.Image)
			return "", errors.Wrapf(err, "Failed to pull image \"%s\"", dsi.options.Image)
		}

		defer reader.Close()
		decoder := json.NewDecoder(reader)
		for {
			var pullEvent dockerEvent
			if err := decoder.Decode(&pullEvent); err != nil {
				if err == io.EOF {
					break
				}

				zerolog.Ctx(ctx).Error().Msg("Failed to decode pull progress")
				return "", errors.Wrap(err, "Failed to decode pull progress")
			}

			if pullEvent.Error != "" {
				zerolog.Ctx(ctx).Error().Msgf("Pull errored: %s", pullEvent.Error)
				return "", errors.Errorf("Pull errored: %s", pullEvent.Error)
			}

			if pullEvent.Status != "" {
				zerolog.Ctx(ctx).Info().Msgf("[Docker] %s", pullEvent.Status)
				dsi.events.TerminalOut.Dispatch(fmt.Sprintf("[Docker] %s", pullEvent.Status))
			}
		}

		zerolog.Ctx(ctx).Info().Msgf("Pulled image \"%s\"", dsi.options.Image)
	}

	// Create the container.
	opts, hostOpts := dsi.options.toOptions()
	container, err := dsi.client.ContainerCreate(ctx, opts, hostOpts, nil, nil, dsi.options.InstanceID.String())
	if err != nil {
		zerolog.Ctx(ctx).Error().Msg("Unable to create container")
		return "", errors.Wrap(err, "Unable to create container")
	}

	zerolog.Ctx(ctx).Info().Msgf("Created container \"%s\"", dsi.options.InstanceID)
	return container.ID, nil
}

// MARK: lifecycleInitAttach

// lifecycleInitAttach is responsible for attaching to the container and
// replicating messages to and from the client.
func (dsi *dockerServerInstance) lifecycleInitAttach() {
	attach, err := dsi.client.ContainerAttach(dsi.ctx, dsi.containerID, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		zerolog.Ctx(dsi.ctx).Error().Msgf("Unable to attach to container: %s", err)
		dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Unable to attach to container: %s", err))
		return
	}
	defer attach.Close()

	go func() {
		scanner := bufio.NewScanner(attach.Reader)
		for scanner.Scan() {
			dsi.events.TerminalOut.Dispatch(scanner.Text())
		}
	}()

	termInChan := dsi.events.TerminalIn.On()
	defer dsi.events.TerminalIn.Off(termInChan)

	for {
		select {
		case <-dsi.ctx.Done():
			return
		case termIn := <-termInChan:
			if !strings.HasSuffix(termIn, "\n") {
				termIn += "\n"
			}

			zerolog.Ctx(dsi.ctx).Debug().Msgf("Executing command: %s", termIn)
			_, err := attach.Conn.Write([]byte(termIn))
			if err != nil {
				zerolog.Ctx(dsi.ctx).Error().Msgf("Error writing to container: %s", err)
				dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Error writing to container: %s", err))
				return
			}
		}
	}
}
