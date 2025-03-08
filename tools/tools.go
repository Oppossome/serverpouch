//go:build tools
// +build tools

package main

import (
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
	_ "github.com/rubenv/sql-migrate/sql-migrate"
	_ "github.com/sqlc-dev/sqlc/cmd/sqlc"
	_ "github.com/vektra/mockery/v2"
)

//go:generate go run github.com/vektra/mockery/v2

//go:generate go run github.com/sqlc-dev/sqlc/cmd/sqlc generate

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=oapi-codegen.yml ../internal/delivery/http/schema/openapi.yml
