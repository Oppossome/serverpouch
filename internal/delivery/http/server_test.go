package http_test

import (
	"net/http"
	"testing"

	"oppossome/serverpouch/internal/delivery/http/openapi"
	"oppossome/serverpouch/internal/domain/server"
	"oppossome/serverpouch/internal/infrastructure/docker"

	mockServer "oppossome/serverpouch/internal/common/test/mocks/domain/server"

	"github.com/Eun/go-hit"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateServer(t *testing.T) {
	t.Run("201 - Ok", func(t *testing.T) {
		sCtx, mockUsecases, testServer := NewTestServer(t)
		testClient := testServer.Client()

		// Create a test server configuration with minimal settings
		cfg := docker.DockerServerInstanceOptions{
			Image:            "test",
			ContainerVolumes: map[string]string{},
			ContainerPorts:   map[int]string{},
			ContainerEnv:     []string{},
		}
		oaCfg, err := openapi.ConfigToOAPI(&cfg)
		assert.NoError(t, err)

		// Setup mock expectations
		inst := mockServer.NewMockServerInstance(t)
		inst.EXPECT().Config().Return(&docker.DockerServerInstanceOptions{InstanceID: uuid.New(), Image: cfg.Image})
		inst.EXPECT().Status().Return(server.ServerInstanceStatusIdle)

		mockUsecases.EXPECT().CreateServer(sCtx, &cfg).Return(inst, nil)

		// Convert mock server instance to OpenAPI format for response validation
		oInst, err := openapi.ServerToOAPI(inst)
		assert.NoError(t, err)

		hit.MustDo(
			hit.Post("%s/api/servers", testServer.URL),
			hit.HTTPClient(testClient),
			hit.Send().Headers("Content-Type").Add("application/json"),
			hit.Send().Body().JSON(openapi.NewServer{Config: *oaCfg}),
			hit.Expect().Status().Equal(http.StatusCreated),
			hitBodyJSONEquals(t, openapi.ServerResponse{Server: *oInst}),
		)
	})
}

func TestGetServer(t *testing.T) {
	t.Run("200 - OK", func(t *testing.T) {
		_, mockUsecases, testServer := NewTestServer(t)
		testClient := testServer.Client()

		// Setup mock expectations
		inst := mockServer.NewMockServerInstance(t)
		inst.EXPECT().Config().Return(&docker.DockerServerInstanceOptions{InstanceID: uuid.New(), Image: "test"})
		inst.EXPECT().Status().Return(server.ServerInstanceStatusIdle)

		mockUsecases.EXPECT().GetServer(mock.Anything, inst.Config().ID()).Return(inst, nil)

		// Convert mock server instance to OpenAPI format for response validation
		oInst, err := openapi.ServerToOAPI(inst)
		assert.NoError(t, err)

		hit.MustDo(
			hit.Get("%s/api/servers/%s", testServer.URL, inst.Config().ID()),
			hit.HTTPClient(testClient),
			hit.Expect().Status().Equal(http.StatusOK),
			hitBodyJSONEquals(t, openapi.ServerResponse{Server: *oInst}),
		)
	})

	t.Run("404 - Not Found", func(t *testing.T) {
		_, mockUsecases, testServer := NewTestServer(t)
		testClient := testServer.Client()

		// Setup mock expectations
		mockUsecases.EXPECT().GetServer(mock.Anything, uuid.Nil).Return(nil, errors.New("Not found!"))

		hit.MustDo(
			hit.Get("%s/api/servers/%s", testServer.URL, uuid.Nil),
			hit.HTTPClient(testClient),
			hit.Expect().Status().Equal(http.StatusNotFound),
		)
	})
}

func TestListServers(t *testing.T) {
	t.Run("200 - OK", func(t *testing.T) {
		_, mockUsecases, testServer := NewTestServer(t)
		testClient := testServer.Client()

		// Setup mock expectations
		inst := mockServer.NewMockServerInstance(t)
		inst.EXPECT().Config().Return(&docker.DockerServerInstanceOptions{InstanceID: uuid.New(), Image: "test"})
		inst.EXPECT().Status().Return(server.ServerInstanceStatusIdle)

		mockUsecases.EXPECT().ListServers(mock.Anything).Return([]server.ServerInstance{inst, inst})

		// Convert mock server instance to OpenAPI format for response validation
		oInst, err := openapi.ServerToOAPI(inst)
		assert.NoError(t, err)

		hit.MustDo(
			hit.Get("%s/api/servers", testServer.URL),
			hit.HTTPClient(testClient),
			hit.Expect().Status().Equal(http.StatusOK),
			hitBodyJSONEquals(t, openapi.ServersResponse{Servers: []openapi.Server{*oInst, *oInst}}),
		)
	})
}
