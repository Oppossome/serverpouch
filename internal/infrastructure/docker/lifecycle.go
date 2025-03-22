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

func (dsi *dockerServerInstance) lifecycle() {
	defer func() { dsi.ctxCancelDone <- struct{}{} }()

	actionChan := dsi.actionChan
	var actionDone chan struct{}
	defer func() {
		if actionDone != nil {
			<-actionDone
		}
	}()

	for {
		select {
		case <-dsi.ctx.Done():
			return

		// Block new actions while we're working
		case actionDone = <-actionChan:
			actionChan = nil

		// Whenever the action is done, listen for more work
		case <-actionDone:
			actionChan = dsi.actionChan
			actionDone = nil

		// If we're not busy, update our status
		case <-time.After(time.Second * 30):
			if actionDone == nil {
				dsi.lifecycleActionUpdateStatus()
			}
		}
	}
}

// MARK: lifecycleAction

func (dsi *dockerServerInstance) lifecycleAction(ctx context.Context) (func(), error) {
	doneChan := make(chan struct{})

	select {
	case <-ctx.Done():
		return nil, errors.New("context closed")
	case dsi.actionChan <- doneChan:
		return func() {
			dsi.lifecycleActionUpdateStatus()
			doneChan <- struct{}{}
		}, nil
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

	inspect, err := dsi.client.ContainerInspect(dsi.ctx, containerID)
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
	actionDone, err := dsi.lifecycleAction(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to acquire init action")
	}
	defer actionDone()

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

			zerolog.Ctx(ctx).Info().Msgf("Found container \"%s\"", container.ID)
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
