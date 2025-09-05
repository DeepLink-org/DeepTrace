VERSION := 1.1.0
COMMIT := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_TAG := beta

build:
	set -e
	mkdir -p output

	CGO_ENABLED=0 GOOS=linux go build -ldflags "\
		-X 'deeptrace/pkg/version.AgentVersion=$(VERSION)' \
		-X 'deeptrace/pkg/version.Commit=$(COMMIT)' \
		-X 'deeptrace/pkg/version.BuildTime=$(BUILD_TIME)' \
		-X 'deeptrace/pkg/version.BuildTag=$(BUILD_TAG)'" \
		-o output/deeptraced cmd/agent/main.go

	CGO_ENABLED=0 GOOS=linux go build -ldflags "\
		-X 'deeptrace/pkg/version.ClientVersion=$(VERSION)' \
		-X 'deeptrace/pkg/version.Commit=$(COMMIT)' \
		-X 'deeptrace/pkg/version.BuildTime=$(BUILD_TIME)' \
		-X 'deeptrace/pkg/version.BuildTag=$(BUILD_TAG)'" \
		-o output/deeptracex cmd/client/main.go

generate:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative v1/deeptrace.proto
