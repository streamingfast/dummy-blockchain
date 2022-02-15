PKG = github.com/streamingfast/dummy-blockchain
BUILD_COMMIT = $(shell git rev-parse HEAD)
BUILD_TIME = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" | tr -d '\n')
BUILD_PATH ?= dummy-blockchain
DIST_PATH ?= dist/$(BUILD_PATH)
LDFLAGS = -s -w -X main.BuildCommit=$(BUILD_COMMIT) -X main.BuildTime=$(BUILD_TIME)

# Build the binary
.PHONY: build
build:
	@go build -ldflags "$(LDFLAGS)" -o $(BUILD_PATH)
	@echo "You can now execute './dummy-blockchain start' command"

# Build binaries for all platforms
.PHONY: release
release:
	GOOS=linux  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_PATH)_linux_amd64
	GOOS=linux  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST_PATH)_linux_arm64
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_PATH)_darwin_amd64
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST_PATH)_darwin_arm64

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

# Start dummy chain in a docker container
.PHONY: docker-start
docker-start:
	docker run -p 8080:8080 -it streamingfast/dummy-blockchain start
