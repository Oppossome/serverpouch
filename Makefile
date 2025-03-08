dev:
	go run ./cmd/serverpouch

test:
	go test ./... -timeout 30s

generate:
	go generate ./tools/tools.go

fmt:
	gofumpt -l -w .
