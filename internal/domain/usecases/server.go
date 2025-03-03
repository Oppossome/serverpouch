package usecases

import (
	"context"
	"fmt"

	"oppossome/serverpouch/internal/domain/server"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

func (usc *usecasesImpl) GetServer(ctx context.Context, id uuid.UUID) (server.ServerInstance, error) {
	usc.srvMu.RLock()
	defer usc.srvMu.RUnlock()

	instance, ok := usc.srvInstances[id]
	if !ok {
		zerolog.Ctx(ctx).Error().Str("id", id.String()).Msg("instance not found")
		return nil, fmt.Errorf("instance of ID \"%s\" not found", id.String())
	}

	return instance, nil
}

func (usc *usecasesImpl) CreateServer(ctx context.Context, cfg server.ServerInstanceConfig) (server.ServerInstance, error) {
	dbCfg, err := usc.db.CreateServerConfig(ctx, cfg)
	if err != nil {
		zerolog.Ctx(ctx).Error().Interface("config", cfg).Err(err).Msg("failed to create server config")
		return nil, errors.Wrap(err, "failed to create server config")
	}

	usc.srvMu.Lock()
	defer usc.srvMu.Unlock()

	srvInstance := dbCfg.NewInstance(ctx)
	usc.srvInstances[dbCfg.ID()] = srvInstance

	return srvInstance, nil
}
