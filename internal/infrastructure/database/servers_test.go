package database_test

import (
	"testing"

	"oppossome/serverpouch/internal/infrastructure/database"
	"oppossome/serverpouch/internal/infrastructure/database/schema"
	"oppossome/serverpouch/internal/infrastructure/docker"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetServer(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		queries, dbRepo := database.NewTestDatabase(t)

		cfg := &docker.DockerServerInstanceOptions{
			Image: "hello-world",
		}

		cfgJSON, err := cfg.ToJSON()
		assert.NoError(t, err)

		srvCfg, err := queries.CreateServer(t.Context(), schema.CreateServerParams{
			Type:   string(cfg.Type()),
			Config: []byte(cfgJSON),
		})
		assert.NoError(t, err)
		cfg.InstanceID = srvCfg.ID // Update cfg to have correct ID

		dbCfg, err := dbRepo.GetServer(t.Context(), srvCfg.ID)
		assert.NoError(t, err)
		assert.Equal(t, cfg, dbCfg)
	})
}

func TestListServers(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		queries, dbRepo := database.NewTestDatabase(t)

		cfg := &docker.DockerServerInstanceOptions{
			InstanceID: uuid.Nil,
			Image:      "hello-world",
		}

		cfgJSON, err := cfg.ToJSON()
		assert.NoError(t, err)

		srvCfg, err := queries.CreateServer(t.Context(), schema.CreateServerParams{
			Type:   string(cfg.Type()),
			Config: []byte(cfgJSON),
		})
		assert.NoError(t, err)
		cfg.InstanceID = srvCfg.ID

		dbCfgs, err := dbRepo.ListServers(t.Context())
		assert.NoError(t, err)

		assert.Equal(t, 1, len(dbCfgs))
		assert.Equal(t, cfg, dbCfgs[0])
	})
}

func TestUpdateServer(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		queries, dbRepo := database.NewTestDatabase(t)

		cfg := &docker.DockerServerInstanceOptions{
			Image: "hello-world",
		}

		cfgJSON, err := cfg.ToJSON()
		assert.NoError(t, err)

		srvCfg, err := queries.CreateServer(t.Context(), schema.CreateServerParams{
			Type:   string(cfg.Type()),
			Config: []byte(cfgJSON),
		})
		assert.NoError(t, err)

		updatedCfg := &docker.DockerServerInstanceOptions{
			InstanceID: srvCfg.ID,
			Image:      "test-image",
		}

		dbConfig, err := dbRepo.UpdateServer(t.Context(), srvCfg.ID, updatedCfg)
		assert.NoError(t, err)
		assert.Equal(t, updatedCfg, dbConfig)
	})
}

func TestCreateServer(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		_, dbRepo := database.NewTestDatabase(t)

		cfg := &docker.DockerServerInstanceOptions{
			Image: "hello-world",
		}

		srvCfg, err := dbRepo.CreateServer(t.Context(), cfg)
		assert.NoError(t, err)

		cfg.InstanceID = srvCfg.ID() // Update cfg to have correct ID
		assert.Equal(t, cfg, srvCfg)
	})
}
