package docker

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	"oppossome/serverpouch/internal/domain/server"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/pkg/errors"
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

			go dsi.lifecycleAction(action.action, activeActionDone)
		}
	}
}

// MARK: lifecycleAction

// lifecycleAction is used to perform the associated lifecycleActions
func (dsi *dockerServerInstance) lifecycleAction(action server.ServerInstanceAction, activeActionDone chan<- struct{}) {
	defer func() {
		dsi.updateContainerStatus()
		activeActionDone <- struct{}{}
	}()

	dsi.mu.RLock()
	status := dsi.status
	containerID := dsi.containerID
	dsi.mu.RUnlock()

	switch {
	case containerID == "":
		if action != server.ServerInstanceActionStart {
			dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Invalid %s action: %s", server.ServerInstanceStatusInitializing, action))
			return
		}

		containerID, err := dsi.getContainer()
		if err != nil {
			dsi.setStatus(server.ServerInstanceStatusErrored)
			dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Unable to get container: %s", err))
			return
		}

		dsi.mu.Lock()
		dsi.containerID = containerID
		dsi.mu.Unlock()

		go dsi.lifecycleInitAttach()

	case action == server.ServerInstanceActionStart:
		if status != server.ServerInstanceStatusIdle {
			dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Invalid %s action: %s", status, action))
			return
		}

		err := dsi.client.ContainerStart(dsi.ctx, containerID, container.StartOptions{})
		if err != nil {
			dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Unable to start container: %s", err))
		}

	case action == server.ServerInstanceActionStop:
		if status != server.ServerInstanceStatusRunning {
			dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Invalid %s action: %s", status, action))
			return
		}

		err := dsi.client.ContainerStop(dsi.ctx, dsi.containerID, container.StopOptions{})
		if err != nil {
			dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Unable to stop container: %s", err))
		}

	case action == server.ServerInstanceActionKill:
		if status != server.ServerInstanceStatusRunning {
			dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Invalid %s action: %s", status, action))
			return
		}

		err := dsi.client.ContainerKill(dsi.ctx, dsi.containerID, "SIGTERM")
		if err != nil {
			dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Unable to kill container: %s", err))
		}
	}
}

// MARK: getContainer

type dockerEvent struct {
	Status         string `json:"status"`
	Error          string `json:"error"`
	Progress       string `json:"progress"`
	ProgressDetail struct {
		Current int `json:"current"`
		Total   int `json:"total"`
	} `json:"progressDetail"`
}

func (dsi *dockerServerInstance) getContainer() (string, error) {
	containers, err := dsi.client.ContainerList(dsi.ctx, container.ListOptions{All: true})
	if err != nil {
		return "", errors.Wrap(err, "Unable to list containers")
	}

	// Check if we have the container already.
	for _, container := range containers {
		if slices.Contains(container.Names, "/"+dsi.options.ID.String()) {
			if container.Image != dsi.options.Image {
				return "", errors.Wrapf(err, "Found non-matching container image: %s", container.Image)
			}

			return container.ID, nil
		}
	}

	// Check if we have the image already.
	images, err := dsi.client.ImageList(dsi.ctx, image.ListOptions{All: true})
	if err != nil {
		return "", errors.Wrap(err, "Unable to list images")
	}

	foundImage := false
	for _, image := range images {
		name, ok := image.Labels["org.opencontainers.image.ref.name"]
		if ok && name == dsi.options.Image {
			foundImage = true
			break
		}
	}

	// Since we couldn't find the image, we'll pull it.
	if !foundImage {
		dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Pulling image \"%s\"", dsi.options.Image))
		reader, err := dsi.client.ImagePull(dsi.ctx, dsi.options.Image, image.PullOptions{})
		if err != nil {
			return "", errors.Wrapf(err, "Failed to pull image \"%s\"", dsi.options.Image)
		}

		decoder := json.NewDecoder(reader)
		for {
			var pullEvent dockerEvent
			if err := decoder.Decode(&pullEvent); err != nil {
				if err == io.EOF {
					break
				}

				return "", errors.Wrap(err, "Failed to decode pull progress")
			}

			if pullEvent.Error != "" {
				return "", errors.Errorf("Pull errored: %s", pullEvent.Error)
			}

			if pullEvent.Status != "" {
				dsi.events.TerminalOut.Dispatch(fmt.Sprintf("[Docker] %s", pullEvent.Status))
			}
		}
	}

	// Create the container.
	opts, hostOpts := dsi.options.toOptions()
	container, err := dsi.client.ContainerCreate(dsi.ctx, opts, hostOpts, nil, nil, dsi.options.ID.String())
	if err != nil {
		return "", errors.Wrap(err, "Unable to create container")
	}

	return container.ID, nil
}

// MARK: updateContainerStatus

func (dsi *dockerServerInstance) updateContainerStatus() {
	dsi.mu.RLock()
	containerID := dsi.containerID
	dsi.mu.RUnlock()

	if containerID == "" {
		dsi.setStatus(server.ServerInstanceStatusInitializing)
		return
	}

	inspect, err := dsi.client.ContainerInspect(dsi.ctx, dsi.containerID)
	if err != nil {
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
	}
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

			_, err := attach.Conn.Write([]byte(termIn))
			if err != nil {
				dsi.events.TerminalOut.Dispatch(fmt.Sprintf("Error writing to container: %s", err))
				return
			}
		}
	}
}
