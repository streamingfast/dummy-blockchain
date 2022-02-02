.PHONY: build proto generate

# Build the binary
build:
	go build
	@echo "You can now execute './dummy-chain start' command"

# Generate protobuf package code
proto:
	protoc \
		--go_out=paths=source_relative:. \
		./proto/codec.proto
