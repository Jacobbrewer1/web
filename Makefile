# Define variables
hash = $(shell git rev-parse --short HEAD)
DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

pr-approval:
	@echo "Running PR CI"
	go build ./...
	go vet ./...
	go test ./...
codegen:
	@echo "Generating code"
	go generate ./...
models:
	chmod +x ./pkg/models/generate.sh
	go generate ./pkg/models
apis:
	go generate ./pkg/apis/...
