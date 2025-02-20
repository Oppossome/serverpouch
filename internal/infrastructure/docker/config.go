package docker

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

type DockerServerInstanceOptions struct {
	ID      uuid.UUID
	Image   string
	Volumes map[string]string
	Ports   map[int]string
	Env     []string
}

func (o *DockerServerInstanceOptions) toOptions() (*container.Config, *container.HostConfig) {
	config := container.Config{
		Image:        o.Image,
		ExposedPorts: nat.PortSet{},
		Volumes:      map[string]struct{}{},
	}

	hostConfig := container.HostConfig{
		PortBindings: nat.PortMap{},
		Binds:        []string{},
	}

	for hostPort, containerPort := range o.Ports {
		natPort := nat.Port(containerPort)
		config.ExposedPorts[natPort] = struct{}{}
		hostConfig.PortBindings[natPort] = []nat.PortBinding{
			{
				HostIP:   "",
				HostPort: fmt.Sprint(hostPort),
			},
		}
	}

	for hostVolume, containerVolume := range o.Volumes {
		hostConfig.Binds = append(hostConfig.Binds, fmt.Sprintf("%s:%s", hostVolume, containerVolume))
		config.Volumes[containerVolume] = struct{}{}
	}

	return &config, &hostConfig
}
