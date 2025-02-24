package server

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type ServerInstanceCollection interface {
	Add(instance ServerInstance)
	Get(ID uuid.UUID) (ServerInstance, error)
}

type serverInstanceCollectionImpl struct {
	mu sync.RWMutex
	instances map[uuid.UUID]ServerInstance
}

var _ ServerInstanceCollection = (*serverInstanceCollectionImpl)(nil)

func (sic *serverInstanceCollectionImpl) Add(instance ServerInstance) {
	sic.mu.Lock()
	defer sic.mu.Unlock()

	sic.instances[instance.Config().ID()] = instance
}

func (sic *serverInstanceCollectionImpl) Get(ID uuid.UUID) (ServerInstance, error) {
	sic.mu.RLock()
	defer sic.mu.RUnlock()

	instance, ok := sic.instances[ID]
	if !ok {
		return nil, fmt.Errorf("couldn't find instance of id \"%s\"", ID)
	}

	return instance, nil
}

