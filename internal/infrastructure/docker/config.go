package docker

import (
	"fmt"
	"io/fs"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

type DockerOptions struct {
	ID      uuid.UUID
	Image   string
	Volumes map[fs.DirEntry]string
	Ports   map[int]string
	Env     []string
}

func (o *DockerOptions) toOptions() (*container.Config, *container.HostConfig) {
	config := container.Config{Image: o.Image, ExposedPorts: nat.PortSet{}}
	hostConfig := container.HostConfig{PortBindings: nat.PortMap{}}

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

	return &config, &hostConfig
}