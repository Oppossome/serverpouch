package server

import (
	"context"

	"oppossome/serverpouch/internal/common/events"

	"github.com/google/uuid"
)

type ServerInstanceStatus string

const (
	ServerInstanceStatusInitializing ServerInstanceStatus = "initializing"
	ServerInstanceStatusIdle         ServerInstanceStatus = "idle"
	ServerInstanceStatusStarting     ServerInstanceStatus = "starting"
	ServerInstanceStatusRunning      ServerInstanceStatus = "running"
	ServerInstanceStatusStopping     ServerInstanceStatus = "stopping"
	ServerInstanceStatusErrored      ServerInstanceStatus = "errored"
)

type ServerInstanceType string

const (
	ServerInstanceTypeDocker ServerInstanceType = "docker"
)

type ServerInstanceEvents struct {
	Status      events.EventEmitter[ServerInstanceStatus]
	TerminalOut events.EventEmitter[string]
	TerminalIn  events.EventEmitter[string]
}

func NewServerInstanceEvents() *ServerInstanceEvents {
	return &ServerInstanceEvents{
		Status:      events.New[ServerInstanceStatus](),
		TerminalOut: events.New[string](),
		TerminalIn:  events.New[string](),
	}
}

type ServerInstance interface {
	Start() error
	Stop() error
	Kill() error

	Config() ServerInstanceConfig
	Status() ServerInstanceStatus
	Events() *ServerInstanceEvents
	Close()
}

type ServerInstanceConfig interface {
	ID() uuid.UUID
	Type() ServerInstanceType
	Ports() []int
	ToJSON() (string, error)
	NewInstance(context.Context) ServerInstance
}
