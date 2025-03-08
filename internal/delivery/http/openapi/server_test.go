package openapi_test

import (
	"testing"

	"oppossome/serverpouch/internal/delivery/http/openapi"
	"oppossome/serverpouch/internal/domain/server"
	"oppossome/serverpouch/internal/infrastructure/docker"

	"github.com/stretchr/testify/assert"
)

func TestOAPIToConfig(t *testing.T) {
	dockerTests := []struct {
		name      string
		config    openapi.ServerConfigDocker
		want      server.ServerInstanceConfig
		wantError string
	}{
		{
			name: "Ok",
			config: openapi.ServerConfigDocker{
				Environment: []string{"PORT=8080"},
				Image:       "test",
				Ports:       []string{"80:8080/tcp", "81:8081/udp"},
				Type:        openapi.Docker,
				Volumes:     []string{"/host:/container"},
			},
			want: &docker.DockerServerInstanceOptions{
				Image:            "test",
				ContainerEnv:     []string{"PORT=8080"},
				ContainerPorts:   map[int]string{80: "8080/tcp", 81: "8081/udp"},
				ContainerVolumes: map[string]string{"/host": "/container"},
			},
		},
		{
			name: "Invalid Environment",
			config: openapi.ServerConfigDocker{
				Environment: []string{"invalid"},
				Image:       "test",
				Ports:       []string{},
				Type:        openapi.Docker,
				Volumes:     []string{},
			},
			wantError: "invalid environment config: invalid",
		},
		{
			name: "Invalid Port",
			config: openapi.ServerConfigDocker{
				Environment: []string{},
				Image:       "test",
				Ports:       []string{"invalid"},
				Type:        openapi.Docker,
				Volumes:     []string{},
			},
			wantError: "invalid port config: invalid",
		},
		{
			name: "Invalid Volume",
			config: openapi.ServerConfigDocker{
				Environment: []string{},
				Image:       "test",
				Ports:       []string{},
				Type:        openapi.Docker,
				Volumes:     []string{"invalid"},
			},
			wantError: "invalid volume config: invalid",
		},
	}

	for _, dt := range dockerTests {
		t.Run(dt.name, func(t *testing.T) {
			srvCfg := openapi.ServerConfig{}
			err := srvCfg.FromServerConfigDocker(dt.config)
			assert.NoError(t, err)

			cfg, err := openapi.OAPIToConfig(srvCfg)
			if dt.want != nil {
				assert.NoError(t, err)
				assert.Equal(t, dt.want, cfg)
			}

			if dt.wantError != "" {
				assert.Equal(t, dt.wantError, err.Error())
			}
		})
	}
}
