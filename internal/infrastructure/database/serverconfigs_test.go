package database_test

import (
	"testing"

	"oppossome/serverpouch/internal/infrastructure/database"
	"oppossome/serverpouch/internal/infrastructure/database/schema"
	"oppossome/serverpouch/internal/infrastructure/docker"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetServerConfig(t *testing.T) {
	t.Parallel()

	t.Run("Ok", func(t *testing.T) {
		queries, dbRepo, err := database.NewTestDatabase(t)
		assert.NoError(t, err)

		cfg := &docker.DockerServerInstanceOptions{
			InstanceID: uuid.Nil,
			Image:      "hello-world",
		}

		cfgJSON, err := cfg.ToJSON()
		assert.NoError(t, err)

		_, err = queries.CreateServerConfig(t.Context(), schema.CreateServerConfigParams{
			ID:     cfg.ID(),
			Type:   string(cfg.Type()),
			Config: []byte(cfgJSON),
		})
		assert.NoError(t, err)

		dbCfg, err := dbRepo.GetServerConfig(t.Context(), cfg.ID())
		assert.NoError(t, err)

		assert.Equal(t, cfg, dbCfg)
	})
}

func TestListServerConfigs(t *testing.T) {
	t.Parallel()

	t.Run("Ok", func(t *testing.T) {
		queries, dbRepo, err := database.NewTestDatabase(t)
		assert.NoError(t, err)

		cfg := &docker.DockerServerInstanceOptions{
			InstanceID: uuid.Nil,
			Image:      "hello-world",
		}

		cfgJSON, err := cfg.ToJSON()
		assert.NoError(t, err)

		_, err = queries.CreateServerConfig(t.Context(), schema.CreateServerConfigParams{
			ID:     cfg.ID(),
			Type:   string(cfg.Type()),
			Config: []byte(cfgJSON),
		})
		assert.NoError(t, err)

		dbCfgs, err := dbRepo.ListServerConfigs(t.Context())
		assert.NoError(t, err)

		assert.Equal(t, 1, len(dbCfgs))
		assert.Equal(t, cfg, dbCfgs[0])
	})
}

func TestUpdateServerConfig(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		queries, dbRepo, err := database.NewTestDatabase(t)
		assert.NoError(t, err)

		cfg := &docker.DockerServerInstanceOptions{
			InstanceID: uuid.Nil,
			Image:      "hello-world",
		}

		cfgJSON, err := cfg.ToJSON()
		assert.NoError(t, err)

		_, err = queries.CreateServerConfig(t.Context(), schema.CreateServerConfigParams{
			ID:     cfg.ID(),
			Type:   string(cfg.Type()),
			Config: []byte(cfgJSON),
		})
		assert.NoError(t, err)

		updatedCfg := &docker.DockerServerInstanceOptions{
			InstanceID: uuid.Nil,
			Image:      "test-image",
		}

		dbConfig, err := dbRepo.UpdateServerConfig(t.Context(), updatedCfg)
		assert.NoError(t, err)

		assert.Equal(t, updatedCfg, dbConfig)
	})
}

func TestCreateServerConfig(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		_, dbRepo, err := database.NewTestDatabase(t)
		assert.NoError(t, err)

		cfg := &docker.DockerServerInstanceOptions{
			InstanceID: uuid.Nil,
			Image:      "hello-world",
		}

		dbCfg, err := dbRepo.CreateServerConfig(t.Context(), cfg)
		assert.NoError(t, err)

		assert.Equal(t, cfg, dbCfg)
	})
}
