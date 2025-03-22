package usecases

import (
	"context"
	"fmt"

	"oppossome/serverpouch/internal/domain/server"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

func (usc *usecasesImpl) ListServers(ctx context.Context) []server.ServerInstance {
	usc.srvMu.RLock()
	defer usc.srvMu.RUnlock()

	insts := make([]server.ServerInstance, 0, len(usc.srvInstances))
	for _, instance := range usc.srvInstances {
		insts = append(insts, instance)
	}

	return insts
}

func (usc *usecasesImpl) GetServer(ctx context.Context, id uuid.UUID) (server.ServerInstance, error) {
	usc.srvMu.RLock()
	defer usc.srvMu.RUnlock()

	inst, ok := usc.srvInstances[id]
	if !ok {
		zerolog.Ctx(ctx).Error().Str("id", id.String()).Msg("instance not found")
		return nil, fmt.Errorf("instance of ID \"%s\" not found", id.String())
	}

	return inst, nil
}

func (usc *usecasesImpl) CreateServer(ctx context.Context, cfg server.ServerInstanceConfig) (server.ServerInstance, error) {
	dbCfg, err := usc.db.CreateServer(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write config to db")
	}

	inst := dbCfg.NewInstance(ctx)

	usc.srvMu.Lock()
	defer usc.srvMu.Unlock()
	usc.srvInstances[dbCfg.ID()] = inst

	return inst, nil
}
