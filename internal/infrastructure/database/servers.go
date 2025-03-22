package database

import (
	"context"
	"encoding/json"
	"fmt"

	"oppossome/serverpouch/internal/domain/server"
	"oppossome/serverpouch/internal/infrastructure/database/schema"
	"oppossome/serverpouch/internal/infrastructure/docker"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

func convertToServer(schema *schema.Server) (server.ServerInstanceConfig, error) {
	switch {
	case schema.Type == string(server.ServerInstanceTypeDocker):
		var dockerOptions docker.DockerServerInstanceOptions
		if err := json.Unmarshal(schema.Config, &dockerOptions); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal docker server config")
		}

		dockerOptions.InstanceID = schema.ID

		return &dockerOptions, nil
	default:
		return nil, fmt.Errorf("unknown server instance type \"%s\"", schema.Type)
	}
}

func (d *databaseImpl) GetServer(ctx context.Context, id uuid.UUID) (server.ServerInstanceConfig, error) {
	dbConfig, err := d.queries.GetServer(ctx, id)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("failed to retrieve server config")
		return nil, errors.Wrap(err, "failed to retrieve server config")
	}

	return convertToServer(&dbConfig)
}

func (d *databaseImpl) ListServers(ctx context.Context) ([]server.ServerInstanceConfig, error) {
	dbConfigs, err := d.queries.GetServers(ctx)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("failed to retrieve server configs")
		return nil, errors.Wrap(err, "failed to retrieve server configs")
	}

	configs := make([]server.ServerInstanceConfig, len(dbConfigs))
	for idx, dbConfig := range dbConfigs {
		config, err := convertToServer(&dbConfig)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("failed to convert server config")
			return nil, errors.Wrap(err, "failed to convert server config")
		}

		configs[idx] = config
	}

	return configs, nil
}

func (d *databaseImpl) UpdateServer(ctx context.Context, id uuid.UUID, config server.ServerInstanceConfig) (server.ServerInstanceConfig, error) {
	configJSON, err := config.ToJSON()
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("failed to convert config to json")
		return nil, errors.Wrap(err, "failed to convert config to json")
	}

	dbConfig, err := d.queries.UpdateServer(ctx, schema.UpdateServerParams{
		ID:     id,
		Config: []byte(configJSON),
	})
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("failed to update server config")
		return nil, errors.Wrap(err, "failed to update server config")
	}

	return convertToServer(&dbConfig)
}

func (d *databaseImpl) CreateServer(ctx context.Context, config server.ServerInstanceConfig) (server.ServerInstanceConfig, error) {
	configJSON, err := config.ToJSON()
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("failed to convert config to json")
		return nil, errors.Wrap(err, "failed to convert config to json")
	}

	dbConfig, err := d.queries.CreateServer(ctx, schema.CreateServerParams{
		Type:   string(config.Type()),
		Config: []byte(configJSON),
	})
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("failed to create server config")
		return nil, errors.Wrap(err, "failed to create server config")
	}

	return convertToServer(&dbConfig)
}
