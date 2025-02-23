package events

import (
	"sync"
)

type EventEmitter[O any] interface {
	On() <-chan O
	Off(<-chan O)
	Dispatch(O)
	Close()
}

type eventEmitterImpl[O any] struct {
	sync.RWMutex

	listeners []chan O
}

var _ EventEmitter[any] = (*eventEmitterImpl[any])(nil)

func (e *eventEmitterImpl[O]) On() <-chan O {
	e.Lock()
	defer e.Unlock()

	listener := make(chan O)
	e.listeners = append(e.listeners, listener)

	return listener
}

func (e *eventEmitterImpl[O]) Off(listener <-chan O) {
	e.Lock()
	defer e.Unlock()

	for i, channel := range e.listeners {
		if channel != listener {
			continue
		}

		e.listeners = append(e.listeners[:i], e.listeners[i+1:]...)
		close(channel)
		return
	}
}

func (e *eventEmitterImpl[O]) Dispatch(value O) {
	e.RLock()
	defer e.RUnlock()

	var wg sync.WaitGroup
	wg.Add(len(e.listeners))
	for _, channel := range e.listeners {
		go func(channel chan O) {
			defer wg.Done()
			channel <- value
		}(channel)
	}

	wg.Wait()
}

func (e *eventEmitterImpl[O]) Close() {
	e.Lock()
	defer e.Unlock()

	for _, channel := range e.listeners {
		close(channel)
	}

	e.listeners = []chan O{}
}

func New[O any]() *eventEmitterImpl[O] {
	return &eventEmitterImpl[O]{
		listeners: []chan O{},
	}
}
