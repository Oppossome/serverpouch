package schema_test

import (
	"testing"

	"oppossome/serverpouch/internal/infrastructure/database/schema"

	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/assert"
)

func TestMigrate(t *testing.T) {
	t.Run("migrations can be applied and rolled back", func(t *testing.T) {
		connStr := schema.SetupTestContainer(t)

		upCount, err := schema.Migrate(t.Context(), connStr, migrate.Up)
		assert.NoError(t, err)

		downCount, err := schema.Migrate(t.Context(), connStr, migrate.Down)
		assert.NoError(t, err)

		// Validate that we've rolled the same amount of migrations that we applied
		assert.Equal(t, upCount, downCount)
	})
}
