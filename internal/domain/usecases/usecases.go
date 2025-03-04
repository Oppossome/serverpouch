package usecases

import (
	"context"
	"sync"

	"oppossome/serverpouch/internal/domain/server"
	"oppossome/serverpouch/internal/infrastructure/database"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type Usecases interface {
	GetServer(context.Context, uuid.UUID) (server.ServerInstance, error)
	CreateServer(context.Context, server.ServerInstanceConfig) (server.ServerInstance, error)
	Close()
}

type usecasesImpl struct {
	db database.Database

	srvMu        sync.RWMutex
	srvInstances map[uuid.UUID]server.ServerInstance
}

var _ Usecases = (*usecasesImpl)(nil)

func New(ctx context.Context) (*usecasesImpl, error) {
	usecases := &usecasesImpl{
		db: database.DatabaseFromContext(ctx),

		srvMu:        sync.RWMutex{},
		srvInstances: make(map[uuid.UUID]server.ServerInstance),
	}

	err := usecases.init(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize usecases")
	}

	zerolog.Ctx(ctx).Debug().Msg("usecases initialized")
	return usecases, nil
}

func (usc *usecasesImpl) init(ctx context.Context) error {
	usc.srvMu.Lock()
	defer usc.srvMu.Unlock()

	srvConfigs, err := usc.db.ListServerConfigs(ctx)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve configs")
	}

	for _, config := range srvConfigs {
		usc.srvInstances[config.ID()] = config.NewInstance(ctx)
	}

	zerolog.Ctx(ctx).Debug().Msgf("%d server instances loaded", len(usc.srvInstances))
	return nil
}

func (usc *usecasesImpl) Close() {
	var wg sync.WaitGroup
	wg.Add(len(usc.srvInstances))
	for _, config := range usc.srvInstances {
		go func() {
			defer wg.Done()
			config.Close()
		}()
	}

	wg.Wait()
}
