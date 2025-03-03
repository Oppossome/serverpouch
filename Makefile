dev:
	go run ./cmd/serverpouch

test:
	go test ./...

generate:
	go generate ./tools/tools.go

	