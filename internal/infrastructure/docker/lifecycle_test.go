package docker

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"oppossome/serverpouch/internal/common/test/mocks/github.com/docker/docker/client"
	"oppossome/serverpouch/internal/domain/server"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/google/uuid"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
)

// MARK: Helpers

func testDockerServerInstance(t *testing.T, options *DockerServerInstanceOptions) (*client.MockAPIClient, *dockerServerInstance) {
	testCtx, testCtxCancel := context.WithCancel(t.Context())
	mockAPIClient := &client.MockAPIClient{}

	return mockAPIClient, &dockerServerInstance{
		ctx:           testCtx,
		ctxCancel:     testCtxCancel,
		ctxCancelDone: make(chan struct{}),

		client:  mockAPIClient,
		events:  server.NewServerInstanceEvents(),
		options: options,

		actionChan: make(chan chan struct{}),

		mu:          sync.RWMutex{},
		containerID: "",
		status:      server.ServerInstanceStatusInitializing,
	}
}

func assertTerminalOut(t *testing.T, dsi *dockerServerInstance, done chan<- struct{}, expected []string) {
	termOut := dsi.events.TerminalOut.On()

	go func() {
		defer dsi.events.TerminalOut.Off(termOut)
		defer func() { done <- struct{}{} }()

		for _, expected := range expected {
			select {
			case <-time.After(5 * time.Second):
				t.Error("testAssertTermOut Timed Out!")
				return
			case msg, ok := <-termOut:
				assert.True(t, ok, "Messages ended prematurely")
				assert.Equal(t, expected, msg)
			}
		}
	}()
}

// MARK: Tests

// MARK: - lifecycle

func TestLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("Ok - Action lifecycle behaves appropriately", func(t *testing.T) {
		mockClient, dsi := testDockerServerInstance(t, &DockerServerInstanceOptions{
			InstanceID: uuid.New(),
			Image:      "Test",
		})

		dsi.containerID = uuid.Nil.String()
		dsi.status = server.ServerInstanceStatusIdle

		// First we will start the container
		mockClient.EXPECT().ContainerStart(
			dsi.ctx,
			dsi.containerID,
			container.StartOptions{},
		).Return(nil)

		// Second behave as if the container has started
		mockClient.EXPECT().ContainerInspect(
			dsi.ctx,
			dsi.containerID,
		).Return(
			types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{Status: "running"},
				},
				Mounts:          []types.MountPoint{},
				Config:          &container.Config{},
				NetworkSettings: &types.NetworkSettings{},
			},
			nil,
		).Once()

		// Third we will shutdown the container
		mockClient.EXPECT().ContainerStop(
			dsi.ctx,
			dsi.containerID,
			container.StopOptions{},
		).Return(nil)

		// Fourth behave as if we shutdown the container
		mockClient.EXPECT().ContainerInspect(
			dsi.ctx,
			dsi.containerID,
		).Return(
			types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{Status: "exited"},
				},
				Mounts:          []types.MountPoint{},
				Config:          &container.Config{},
				NetworkSettings: &types.NetworkSettings{},
			},
			nil,
		).Once()

		statusChan := dsi.events.Status.On()
		defer dsi.Events().Status.Off(statusChan)

		go dsi.lifecycle()
		go func() {
			dsi.Start()
			dsi.Stop()
		}()

		assert.Equal(t, <-statusChan, server.ServerInstanceStatusStarting)
		assert.Equal(t, <-statusChan, server.ServerInstanceStatusRunning)
		assert.Equal(t, <-statusChan, server.ServerInstanceStatusStopping)
		assert.Equal(t, <-statusChan, server.ServerInstanceStatusIdle)
	})

	t.Run("Edgecase - Shutting down mid-action", func(t *testing.T) {
		mockClient, dsi := testDockerServerInstance(t, &DockerServerInstanceOptions{
			InstanceID: uuid.New(),
			Image:      "Test",
		})

		dsi.containerID = uuid.Nil.String()
		dsi.status = server.ServerInstanceStatusIdle

		// The action we will shutdown during
		mockClient.EXPECT().ContainerStart(
			dsi.ctx,
			dsi.containerID,
			container.StartOptions{},
		).Return(nil).WaitUntil(
			time.After(1 * time.Second),
		)

		mockClient.EXPECT().ContainerInspect(
			dsi.ctx,
			dsi.containerID,
		).Return(
			types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{Status: "running"},
				},
				Mounts:          []types.MountPoint{},
				Config:          &container.Config{},
				NetworkSettings: &types.NetworkSettings{},
			},
			nil,
		).Once()

		go dsi.lifecycle()

		done := make(chan struct{})
		go func() {
			dsi.Start()
			done <- struct{}{}
		}()

		go func() {
			time.Sleep(time.Millisecond * 500)
			dsi.Close()
		}()

		<-done
	})
}

// MARK: - lifecycleActionUpdateStatus

func TestLifecycleActionUpdateStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		containerID     string
		containerStatus string
		expected        server.ServerInstanceStatus
	}{
		{
			name:            "Ok - Without Container",
			containerID:     "",
			containerStatus: "",
			expected:        server.ServerInstanceStatusInitializing,
		},
		{
			name:            "Ok - Created",
			containerID:     uuid.Nil.String(),
			containerStatus: "created",
			expected:        server.ServerInstanceStatusIdle,
		},
		{
			name:            "Ok - Exited",
			containerID:     uuid.Nil.String(),
			containerStatus: "exited",
			expected:        server.ServerInstanceStatusIdle,
		},
		{
			name:            "Ok - Running",
			containerID:     uuid.Nil.String(),
			containerStatus: "running",
			expected:        server.ServerInstanceStatusRunning,
		},
	}

	for _, tt := range tests {
		t.Run(t.Name(), func(t *testing.T) {
			mockClient, dsi := testDockerServerInstance(t, &DockerServerInstanceOptions{
				InstanceID: uuid.New(),
				Image:      "Test",
			})

			dsi.containerID = tt.containerID

			if tt.containerStatus != "" {
				mockClient.EXPECT().ContainerInspect(
					dsi.ctx,
					dsi.containerID,
				).Return(
					types.ContainerJSON{
						ContainerJSONBase: &types.ContainerJSONBase{
							State: &types.ContainerState{Status: tt.containerStatus},
						},
						Mounts:          []types.MountPoint{},
						Config:          &container.Config{},
						NetworkSettings: &types.NetworkSettings{},
					},
					nil,
				).Once()
			}

			dsi.lifecycleActionUpdateStatus()
			assert.Equal(t, tt.expected, dsi.Status())
		})
	}
}

// MARK: - lifecycleInit

func TestLifecycleInit(t *testing.T) {
	t.Parallel()

	t.Run("Ok - Finds the container", func(t *testing.T) {
		mockClient, dsi := testDockerServerInstance(t, &DockerServerInstanceOptions{
			InstanceID: uuid.New(),
			Image:      "Test",
		})

		// First, it lists all containers
		mockClient.EXPECT().ContainerList(
			dsi.ctx,
			container.ListOptions{All: true},
		).Return(
			[]types.Container{{
				ID:    uuid.Nil.String(),
				Image: dsi.options.Image,
				// Append / to the beginning like docker is doing.
				Names: []string{"/" + dsi.options.InstanceID.String()},
			}},
			nil,
		)

		go dsi.lifecycle()
		containerID, err := dsi.lifecycleInit(dsi.ctx)
		assert.Equal(t, uuid.Nil.String(), containerID)
		assert.NoError(t, err)
	})

	t.Run("Ok - Creates container after finding image", func(*testing.T) {
		mockClient, dsi := testDockerServerInstance(t, &DockerServerInstanceOptions{
			InstanceID: uuid.New(),
			Image:      "Test",
		})

		// First, it lists all containers
		mockClient.EXPECT().ContainerList(
			dsi.ctx,
			container.ListOptions{All: true},
		).Return(
			[]types.Container{},
			nil,
		)

		// Second, because it can't find a container, it lists all images
		mockClient.EXPECT().ImageList(
			dsi.ctx,
			image.ListOptions{All: true},
		).Return(
			[]image.Summary{{
				Labels: map[string]string{
					"org.opencontainers.image.ref.name": dsi.options.Image,
				},
			}},
			nil,
		)

		// Third, now that we've found the image it should create a container
		opts, hostOpts := dsi.options.toOptions()
		mockClient.EXPECT().ContainerCreate(
			dsi.ctx,
			opts,
			hostOpts,
			(*network.NetworkingConfig)(nil),
			(*v1.Platform)(nil),
			dsi.options.InstanceID.String(),
		).Return(
			container.CreateResponse{ID: uuid.Nil.String()},
			nil,
		)

		go dsi.lifecycle()
		containerID, err := dsi.lifecycleInit(dsi.ctx)
		assert.Equal(t, uuid.Nil.String(), containerID)
		assert.NoError(t, err)
	})

	t.Run("Ok - Creates container after pulling image", func(t *testing.T) {
		mockClient, dsi := testDockerServerInstance(t, &DockerServerInstanceOptions{
			InstanceID: uuid.New(),
			Image:      "Test",
		})

		// First, it lists all containers
		mockClient.EXPECT().ContainerList(
			dsi.ctx,
			container.ListOptions{All: true},
		).Return(
			[]types.Container{},
			nil,
		)

		// Second, because it can't find a container, it lists all images
		mockClient.EXPECT().ImageList(
			dsi.ctx,
			image.ListOptions{All: true},
		).Return(
			[]image.Summary{},
			nil,
		)

		// Third, because it can't find a container it pulls the image
		mockClient.EXPECT().ImagePull(
			dsi.ctx,
			dsi.options.Image,
			image.PullOptions{},
		).Return(
			io.NopCloser(strings.NewReader(`
				{"status":"pulling fs layer"}
				{"status":"downloading","progressDetail":{"current":1,"total":100}}
				{"status":"download complete"}
			`)),
			nil,
		)

		// Fourth, now that we've pulled the image it should create a container
		opts, hostOpts := dsi.options.toOptions()
		mockClient.EXPECT().ContainerCreate(
			dsi.ctx,
			opts,
			hostOpts,
			(*network.NetworkingConfig)(nil),
			(*v1.Platform)(nil),
			dsi.options.InstanceID.String(),
		).Return(
			container.CreateResponse{ID: uuid.Nil.String()},
			nil,
		)

		done := make(chan struct{})
		assertTerminalOut(t, dsi, done, []string{
			"Pulling image \"Test\"",
			"[Docker] pulling fs layer",
			"[Docker] downloading",
			"[Docker] download complete",
		})

		go dsi.lifecycle()
		containerID, err := dsi.lifecycleInit(dsi.ctx)
		assert.Equal(t, uuid.Nil.String(), containerID)
		assert.NoError(t, err)

		<-done
	})
}
