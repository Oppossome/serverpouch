dev:
	go run ./cmd/serverpouch

fmt:
	gofumpt -l -w .

generate:
	go generate ./tools/tools.go
	$(MAKE) fmt

test:
	go test ./... -timeout 30s