package openapi

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/pkg/errors"

	"oppossome/serverpouch/internal/domain/server"
	"oppossome/serverpouch/internal/infrastructure/docker"
)

// MARK: ServerToOAPI

func ServerToOAPI(server server.ServerInstance) (*Server, error) {
	cfg, err := ConfigToOAPI(server.Config())
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert config to OAPI")
	}

	srv := &Server{
		Config: *cfg,
		Id:     server.Config().ID(),
		Status: ServerStatus(server.Status()),
	}

	return srv, nil
}

// MARK: ConfigToOAPI

func ConfigToOAPI(config server.ServerInstanceConfig) (*ServerConfig, error) {
	switch config.Type() {
	case server.ServerInstanceTypeDocker:
		config, ok := config.(*docker.DockerServerInstanceOptions)
		if !ok {
			return nil, errors.New("unable to convert docker config")
		}

		dSrvCfg := ServerConfigDocker{
			Environment: config.ContainerEnv,
			Image:       config.Image,
			Ports:       []string{},
			Type:        Docker,
			Volumes:     []string{},
		}

		for hostPort, containerPort := range config.ContainerPorts {
			portStr := fmt.Sprintf("%d:%s", hostPort, containerPort)
			dSrvCfg.Ports = append(dSrvCfg.Ports, portStr)
		}

		for hostVol, containerVol := range config.ContainerVolumes {
			volumeStr := fmt.Sprintf("%s:%s", hostVol, containerVol)
			dSrvCfg.Volumes = append(dSrvCfg.Volumes, volumeStr)
		}

		srvCfg := &ServerConfig{}
		err := srvCfg.FromServerConfigDocker(dSrvCfg)
		if err != nil {
			return nil, err
		}

		return srvCfg, nil

	default:
		return nil, fmt.Errorf("unknown config type %s", config.Type())
	}
}

// MARK: OAPIToConfig

func OAPIToConfig(config ServerConfig) (server.ServerInstanceConfig, error) {
	if dockerCfg, err := config.AsServerConfigDocker(); err == nil {
		return dockerOAPIToConfig(dockerCfg)
	}

	return nil, errors.New("unknown config")
}

// MARK: - dockerOAPIToConfig

var (
	// Pattern for Docker port mapping: "hostPort:containerPort/protocol"
	// Example: "8080:80/tcp"
	dockerPortPattern = regexp.MustCompile(`^(\d+):(\d+/(?:udp|tcp))$`)

	// Pattern for Docker volume mapping: "hostPath:containerPath"
	// Example: "/host/path:/container/path"
	dockerVolumePattern = regexp.MustCompile(`^([^:]+):([^:]+)$`)

	// Pattern for Docker environment variables: "KEY=value"
	// Example: "PORT=8080"
	dockerEnvPattern = regexp.MustCompile(`^\w+=.+$`)
)

func dockerOAPIToConfig(config ServerConfigDocker) (*docker.DockerServerInstanceOptions, error) {
	dockerOpts := &docker.DockerServerInstanceOptions{
		Image:            config.Image,
		ContainerVolumes: map[string]string{},
		ContainerPorts:   map[int]string{},
		ContainerEnv:     []string{},
	}

	for _, port := range config.Ports {
		portMatches := dockerPortPattern.FindStringSubmatch(port)
		if portMatches == nil {
			return nil, fmt.Errorf("invalid port config: %s", port)
		}

		hostPort, err := strconv.Atoi(portMatches[1])
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert host port to int")
		}

		dockerOpts.ContainerPorts[hostPort] = portMatches[2]
	}

	for _, volume := range config.Volumes {
		volumeMatches := dockerVolumePattern.FindStringSubmatch(volume)
		if volumeMatches == nil {
			return nil, fmt.Errorf("invalid volume config: %s", volume)
		}

		dockerOpts.ContainerVolumes[volumeMatches[1]] = volumeMatches[2]
	}

	for _, env := range config.Environment {
		envMatch := dockerEnvPattern.FindString(env)
		if envMatch == "" {
			return nil, fmt.Errorf("invalid environment config: %s", env)
		}

		dockerOpts.ContainerEnv = append(dockerOpts.ContainerEnv, envMatch)
	}

	return dockerOpts, nil
}
