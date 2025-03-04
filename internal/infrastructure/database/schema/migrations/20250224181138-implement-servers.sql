
-- +migrate Up

CREATE TABLE server_configs (
  id UUID PRIMARY KEY,
  type TEXT NOT NULL,
  config JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +migrate Down

DROP TABLE server_configs;