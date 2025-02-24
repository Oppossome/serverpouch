package docker

import (
	"encoding/json"
	"fmt"

	"oppossome/serverpouch/internal/domain/server"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var _ server.ServerInstanceConfig = (*DockerServerInstanceOptions)(nil)

type DockerServerInstanceOptions struct {
	InstanceID       uuid.UUID         `json:"id"`
	Image            string            `json:"image"`
	ContainerVolumes map[string]string `json:"volumes"`
	ContainerPorts   map[int]string    `json:"ports"`
	ContainerEnv     []string          `json:"env"`
}

func (dsic *DockerServerInstanceOptions) toOptions() (*container.Config, *container.HostConfig) {
	config := container.Config{
		Image:        dsic.Image,
		ExposedPorts: nat.PortSet{},
		Volumes:      map[string]struct{}{},
	}

	hostConfig := container.HostConfig{
		PortBindings: nat.PortMap{},
		Binds:        []string{},
	}

	for hostPort, containerPort := range dsic.ContainerPorts {
		natPort := nat.Port(containerPort)
		config.ExposedPorts[natPort] = struct{}{}
		hostConfig.PortBindings[natPort] = []nat.PortBinding{
			{
				HostIP:   "",
				HostPort: fmt.Sprint(hostPort),
			},
		}
	}

	for hostVolume, containerVolume := range dsic.ContainerVolumes {
		hostConfig.Binds = append(hostConfig.Binds, fmt.Sprintf("%s:%s", hostVolume, containerVolume))
		config.Volumes[containerVolume] = struct{}{}
	}

	return &config, &hostConfig
}

func (dsio *DockerServerInstanceOptions) ID() uuid.UUID {
	return dsio.InstanceID
}

func (dsio *DockerServerInstanceOptions) Type() server.ServerInstanceType {
	return server.ServerInstanceTypeDocker
}

func (dsio *DockerServerInstanceOptions) Ports() []int {
	ports := []int{}
	for hostPort := range dsio.ContainerPorts {
		ports = append(ports, hostPort)
	}

	return ports
}

func (dsio *DockerServerInstanceOptions) ToJSON() (string, error) {
	json, err := json.Marshal(dsio)
	if err != nil {
		return "", errors.Wrap(err, "Error Encoding DockerServerInstanceOptions")
	}

	return string(json), nil
}
