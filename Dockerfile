ARG FIRECORE_VERSION=v1.10.1

FROM golang:1.24-bookworm AS build
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . ./

# Build the binary with version information
ARG VERSION="dev"
ARG BINARY_NAME=dummy-blockchain
RUN go build -v -ldflags "-X 'main.Version=${VERSION}'" -o ${BINARY_NAME} ./cmd/${BINARY_NAME}

FROM ghcr.io/streamingfast/firehose-core:${FIRECORE_VERSION}

ARG BINARY_NAME=dummy-blockchain

# Copy the binary to the firehose-core image
COPY --from=build /app/${BINARY_NAME} /app/${BINARY_NAME}

# We use firecore entrypoint since it's the main application that people should run
ENTRYPOINT ["/app/firecore"]