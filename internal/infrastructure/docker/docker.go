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

var _ server.ServerInstance = (*dockerServerInstance)(nil)

type dockerServerInstance struct {
	action  events.EventEmitter[server.ServerInstanceAction]
	events  *server.ServerInstanceEvents
	options *DockerServerInstanceOptions
	status  server.ServerInstanceStatus
}

func (d *dockerServerInstance) Action(action server.ServerInstanceAction) {
	fmt.Printf("Running Action %s\n", action)
	d.action.Dispatch(action)
}

func (d *dockerServerInstance) Status() server.ServerInstanceStatus {
	return d.status
}

func (d *dockerServerInstance) Events() *server.ServerInstanceEvents {
	return d.events
}

func (d *dockerServerInstance) Type() server.ServerInstanceType {
	return server.ServerInstanceTypeDocker
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

func New(ctx context.Context, options *DockerServerInstanceOptions) (*dockerServerInstance, error) {
	serverProcess := dockerServerInstance{
		action:  events.New[server.ServerInstanceAction](),
		events:  server.NewServerInstanceEvents(),
		options: options,
		status:  server.ServerInstanceStatusInitializing,
	}

	go serverProcess.init(ctx)

	return &serverProcess, nil
}

func (d *dockerServerInstance) setStatus(status server.ServerInstanceStatus) {
	d.status = status
	d.events.Status.Dispatch(status)
}

// MARK: init

func (d *dockerServerInstance) init(ctx context.Context) {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		d.setStatus(server.ServerInstanceStatusErrored)
		d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Unable to contact Docker: %s", err.Error()))
		return
	}

	// MARK: - Image
	images, err := client.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		d.setStatus(server.ServerInstanceStatusErrored)
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
			d.setStatus(server.ServerInstanceStatusErrored)
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

				d.setStatus(server.ServerInstanceStatusErrored)
				d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Failed to decode pull progress: %s", err.Error()))
				return
			}

			if pullEvent.Error != "" {
				d.setStatus(server.ServerInstanceStatusErrored)
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
			d.setStatus(server.ServerInstanceStatusErrored)
			d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Unable to create container: %s", err.Error()))
			return
		}

		containerID = container.ID
	}

	d.commandInit(ctx, client, containerID)
}

// MARK: command

func (d *dockerServerInstance) commandInit(ctx context.Context, client *client.Client, containerID string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		inspect, err := client.ContainerInspect(ctx, containerID)
		if err != nil {
			d.setStatus(server.ServerInstanceStatusErrored)
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
			d.setStatus(server.ServerInstanceStatusErrored)
			d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Unknown State: %s", inspect.State.Status))
			return
		}
	}
}

// MARK: commandIdle

func (d *dockerServerInstance) commandIdle(ctx context.Context, client *client.Client, containerId string) {
	d.setStatus(server.ServerInstanceStatusIdle)

	actionChan := d.action.On()
	defer d.action.Off(actionChan)

	for {
		select {
		case <-ctx.Done():
			return
		case action := <-actionChan:
			switch {
			case action == server.ServerInstanceActionStart:
				err := client.ContainerStart(ctx, containerId, container.StartOptions{})
				if err != nil {
					d.setStatus(server.ServerInstanceStatusErrored)
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

func (d *dockerServerInstance) commandRunning(ctx context.Context, client *client.Client, containerId string) {
	d.setStatus(server.ServerInstanceStatusRunning)

	actionChan := d.action.On()
	defer d.action.Off(actionChan)

	attach, err := client.ContainerAttach(ctx, containerId, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		d.setStatus(server.ServerInstanceStatusErrored)
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
			case action == server.ServerInstanceActionKill:
				err := client.ContainerStop(ctx, containerId, container.StopOptions{Signal: "SIGKILL"})
				if err != nil {
					d.setStatus(server.ServerInstanceStatusErrored)
					d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Unable to stop server: %s", err.Error()))
				}

				return
			case action == server.ServerInstanceActionStop:
				err := client.ContainerStop(ctx, containerId, container.StopOptions{})
				if err != nil {
					d.setStatus(server.ServerInstanceStatusErrored)
					d.events.TerminalOut.Dispatch(fmt.Sprintf("[Serverpouch] Unable to stop server: %s", err.Error()))
				}

				return
			}
		}
	}
}
