# ------------------------------------------------------------------------------
# Go Builder Image
# ------------------------------------------------------------------------------
FROM golang:1.17 AS build

WORKDIR /build

COPY ./go.mod .
COPY ./go.sum .

RUN go mod download

COPY . .

ENV CGO_ENABLED=0
ENV GOARCH=amd64
ENV GOOS=linux

RUN make build

# ------------------------------------------------------------------------------
# Target Image
# ------------------------------------------------------------------------------
FROM alpine AS release

WORKDIR /app/

COPY --from=build /build/dummy-blockchain /app/dummy-blockchain

RUN addgroup --gid 1234 app
RUN adduser --system --uid 1234 app

RUN chown -R app:app /app
USER 1234

ENTRYPOINT ["/app/dummy-blockchain"]
