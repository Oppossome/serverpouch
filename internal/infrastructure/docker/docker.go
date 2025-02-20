package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"time"

	"oppossome/serverpouch/internal/common/events"
	"oppossome/serverpouch/internal/domain/server"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

var _ server.ServerHandler = (*dockerServerHandler)(nil)

type dockerServerHandler struct {
	action  events.EventEmitter[server.HandlerAction]
	events  *server.HandlerEvents
	options *DockerOptions
	status  server.HandlerStatus
}

func (d *dockerServerHandler) Action(action server.HandlerAction) {
	fmt.Printf("Running Action %s\n", action)
	d.action.Dispatch(action)
}

func (d *dockerServerHandler) Status() server.HandlerStatus {
	return d.status
}

func (d *dockerServerHandler) Events() *server.HandlerEvents {
	return d.events
}

type dockerEvent struct {
	Status         string `json:"status"`
	Error          string `json:"error"`
	Progress       string `json:"progress"`
	ProgressDetail struct {
		Current int `json:"current"`
		Total   int `json:"total"`
	} `json:"progressDetail"`
}

func New(ctx context.Context, options *DockerOptions) (*dockerServerHandler, error) {
	serverHandler := dockerServerHandler{
		action:  events.New[server.HandlerAction](),
		events:  server.NewHandlerEvents(),
		options: options,
		status:  server.HandlerStatusInitializing,
	}

	go serverHandler.init(ctx)

	return &serverHandler, nil
}

func (d *dockerServerHandler) setStatus(status server.HandlerStatus) {
	d.status = status
	d.events.Status.Dispatch(status)
}

// MARK: init

func (d *dockerServerHandler) init(ctx context.Context) {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		d.setStatus(server.HandlerStatusErrored)
		d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Unable to contact Docker: %s", err.Error()))
		return
	}

	// MARK: - Image
	images, err := client.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		d.setStatus(server.HandlerStatusErrored)
		d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Unable to list images: %s", err.Error()))
		return
	}

	foundImage := false
	for _, img := range images {
		name, ok := img.Labels["org.opencontainers.image.ref.name"]
		if ok && name == d.options.Image {
			foundImage = true
			break
		}
	}

	if !foundImage {
		d.events.TerminalOut.Dispatch(fmt.Sprintf("Pulling Image %s", d.options.Image))
		reader, err := client.ImagePull(ctx, d.options.Image, image.PullOptions{})
		if err != nil {
			d.setStatus(server.HandlerStatusErrored)
			d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Image pull failed: %s", err.Error()))
			return
		}

		decoder := json.NewDecoder(reader)
		for {
			var pullEvent dockerEvent
			if err := decoder.Decode(&pullEvent); err != nil {
				if err == io.EOF {
					break
				}

				d.setStatus(server.HandlerStatusErrored)
				d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Failed to decode pull progress: %s", err.Error()))
				return
			}

			if pullEvent.Error != "" {
				d.setStatus(server.HandlerStatusErrored)
				d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Pull failed: %s", pullEvent.Error))
				return
			}

			// TODO: Progress
			if pullEvent.Status != "" {
				d.events.TerminalOut.Dispatch(fmt.Sprintf("[Docker] %s", pullEvent.Status))
			}
		}
	}

	// MARK: - Container
	containers, err := client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		d.events.TerminalOut.Dispatch(fmt.Sprintf("Unable to list containers: %s", err))
		return
	}

	containerID := ""
	for _, container := range containers {
		if slices.Contains(container.Names, "/"+d.options.ID.String()) {
			if container.Image != d.options.Image {
				d.events.TerminalOut.Dispatch(fmt.Sprintf("Non-matching Image: %s", container.Image))

				return
			}

			containerID = container.ID
			break
		}
	}

	if containerID == "" {
		d.events.TerminalOut.Dispatch("Container not found, Creating")

		opts, hostOpts := d.options.toOptions()
		container, err := client.ContainerCreate(ctx, opts, hostOpts, nil, nil, d.options.ID.String())
		if err != nil {
			d.setStatus(server.HandlerStatusErrored)
			d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Unable to create container: %s", err.Error()))
			return
		}

		containerID = container.ID
	}

	d.commandInit(ctx, client, containerID)
}

// MARK: command

func (d *dockerServerHandler) commandInit(ctx context.Context, client *client.Client, containerID string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		inspect, err := client.ContainerInspect(ctx, containerID)
		if err != nil {
			d.setStatus(server.HandlerStatusErrored)
			d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Failed to inspect container: %s", err.Error()))
			return
		}

		switch {
		case inspect.State.Status == "created":
			fallthrough
		case inspect.State.Status == "exited":
			d.commandIdle(ctx, client, containerID)
		case inspect.State.Status == "running":
			d.commandRunning(ctx, client, containerID)
		case inspect.State.Status == "restarting":
			time.Sleep(time.Millisecond * 500)
		default:
			d.setStatus(server.HandlerStatusErrored)
			d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Unknown State: %s", inspect.State.Status))
			return
		}
	}
}

// MARK: commandIdle

func (d *dockerServerHandler) commandIdle(ctx context.Context, client *client.Client, containerId string) {
	d.setStatus(server.HandlerStatusIdle)

	actionChan := d.action.On()
	defer d.action.Off(actionChan)

	for {
		select {
		case <-ctx.Done():
			return
		case action := <-actionChan:
			switch {
			case action == server.HandlerActionStart:
				err := client.ContainerStart(ctx, containerId, container.StartOptions{})
				if err != nil {
					d.setStatus(server.HandlerStatusErrored)
					d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Unable to start server: %s", err.Error()))
				}

				return
			default:
				d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Invalid action: %s", action))
			}
		}
	}
}

// MARK: commandRunning

func (d *dockerServerHandler) commandRunning(ctx context.Context, client *client.Client, containerId string) {
	d.setStatus(server.HandlerStatusRunning)

	actionChan := d.action.On()
	defer d.action.Off(actionChan)

	attach, err := client.ContainerAttach(ctx, containerId, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		d.setStatus(server.HandlerStatusErrored)
		d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Unable to inspect container: %s", err.Error()))
		return
	}
	defer attach.Close()

	go func() {
		scanner := bufio.NewScanner(attach.Reader)
		for scanner.Scan() {
			d.events.TerminalOut.Dispatch(scanner.Text())
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case action := <-actionChan:
			switch {
			case action == server.HandlerActionKill:
				err := client.ContainerStop(ctx, containerId, container.StopOptions{Signal: "SIGKILL"})
				if err != nil {
					d.setStatus(server.HandlerStatusErrored)
					d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Unable to stop server: %s", err.Error()))
				}

				return
			case action == server.HandlerActionStop:
				err := client.ContainerStop(ctx, containerId, container.StopOptions{})
				if err != nil {
					d.setStatus(server.HandlerStatusErrored)
					d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Unable to stop server: %s", err.Error()))
				}

				return
			}
		}
	}
}
