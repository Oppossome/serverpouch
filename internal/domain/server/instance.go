package server

import (
	"oppossome/serverpouch/internal/common/events"
)

type ServerInstanceStatus string

const (
	ServerInstanceStatusInitializing ServerInstanceStatus = "Initializing"
	ServerInstanceStatusIdle         ServerInstanceStatus = "Idle"
	ServerInstanceStatusStarting     ServerInstanceStatus = "Starting"
	ServerInstanceStatusRunning      ServerInstanceStatus = "Running"
	ServerInstanceStatusStopping     ServerInstanceStatus = "Stopping"
	ServerInstanceStatusErrored      ServerInstanceStatus = "Errored"
)

type ServerInstanceAction string

const (
	ServerInstanceActionStart ServerInstanceAction = "Start"
	ServerInstanceActionStop  ServerInstanceAction = "Stop"
	ServerInstanceActionKill  ServerInstanceAction = "Kill"
)

type ServerInstanceType string

const (
	ServerInstanceTypeDocker ServerInstanceType = "Docker"
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
	Type() ServerInstanceType
	Action(action ServerInstanceAction)
	Status() ServerInstanceStatus
	Events() *ServerInstanceEvents
}
