package server

import (
	"oppossome/serverpouch/internal/common/events"
)

type HandlerStatus string

const (
	HandlerStatusInitializing HandlerStatus = "Initializing"
	HandlerStatusIdle         HandlerStatus = "Idle"
	HandlerStatusStarting     HandlerStatus = "Starting"
	HandlerStatusRunning      HandlerStatus = "Running"
	HandlerStatusStopping     HandlerStatus = "Stopping"
	HandlerStatusErrored      HandlerStatus = "Errored"
)

type HandlerAction string

const (
	HandlerActionStart HandlerAction = "Start"
	HandlerActionStop  HandlerAction = "Stop"
	HandlerActionKill  HandlerAction = "Kill"
)

type HandlerEvents struct {
	Status      events.EventEmitter[HandlerStatus]
	TerminalOut events.EventEmitter[string]
	TerminalIn  events.EventEmitter[string]
}

func NewHandlerEvents() *HandlerEvents {
	return &HandlerEvents{
		Status:      events.New[HandlerStatus](),
		TerminalOut: events.New[string](),
		TerminalIn:  events.New[string](),
	}
}

type ServerHandler interface {
	Action(action HandlerAction)
	Status() HandlerStatus
	Events() *HandlerEvents
}
