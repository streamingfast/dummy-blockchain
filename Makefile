PKG = github.com/streamingfast/dummy-blockchain
LDFLAGS = -s -w
BUILD_COMMIT = $(shell git rev-parse HEAD)
BUILD_TIME = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" | tr -d '\n')

# Build the binary
.PHONY: build
build: LDFLAGS += -X main.BuildCommit=$(BUILD_COMMIT)
build: LDFLAGS += -X main.BuildTime=$(BUILD_TIME)
build:
	@go build -ldflags "$(LDFLAGS)"
	@echo "You can now execute './dummy-blockchain start' command"

# Generate protobuf package code
.PHONY: proto
proto:
	protoc \
		--go_out=paths=source_relative:. \
		./proto/codec.proto

# Build docker image
.PHONY: docker-build
docker-build:
	docker build -t streamingfast/dummy-blockchain .
