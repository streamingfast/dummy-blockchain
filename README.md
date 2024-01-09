# Dummy Chain

This dummy "blockchain" serves as a demonstration on how to instrument node for the
Firehose integration. Instrumentors, "firehose" is a part responsible for data
extraction and could be turned on with an environment variable when running the chain
process.

## Requirements

- Go

## Getting Started

To install the binary:

```bash
go install github.com/streamingfast/dummy-blockchain@laest
```

> [!NOTE]
> Ensure `go env GOPATH` directoy is part of your PATH (export PATH="`` `go env GOPATH` ``:$PATH")

To start the chain, run:

```shell
./dummy-blockchain start
```

You'll start seeing output like:

```
INFO[2022-01-13T11:55:07-06:00] initializing node
INFO[2022-01-13T11:55:07-06:00] initializing store
INFO[2022-01-13T11:55:07-06:00] initializing engine
INFO[2022-01-13T11:55:07-06:00] starting block producer                       rate=1s
INFO[2022-01-13T11:55:08-06:00] processing block                              hash=6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b height=1
INFO[2022-01-13T11:55:09-06:00] processing block                              hash=d4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35 height=2
INFO[2022-01-13T11:55:10-06:00] processing block                              hash=4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce height=3
INFO[2022-01-13T11:55:11-06:00] processing block                              hash=4b227777d4dd1fc61c6f884f48641d02b4d121d3fd328cb08b5531fcacdabf8a height=4
INFO[2022-01-13T11:55:12-06:00] processing block                              hash=ef2d127de37b942baad06145e54b0c619a1f22327b2ebbcfbec78f5564afe39d height=5
INFO[2022-01-13T11:55:13-06:00] processing block                              hash=e7f6c011776e8db7cd330b54174fd76f7d0216b612387a5ffcfb81e6f0919683 height=6
```

To enable firehose instrumentation:

```
./dummy-blockchain start --tracer=firehose
```

Firehose logger statement will be printed as blocks are executed/produced. This mode is meant to be run
using by a Firehose `reader-node`, see https://github.com/streamingfast/firehose-acme.

## Tracer

This project showcase a "fake" blockchain's node codebase. For developers looking into integrating a native Firehose integration, we suggest to integrate in blockchain's client code directly by some form of tracing plugin that is able to receive all the important callback's while transactions are execution integrating as deeply as wanted.

You will see here in [tracer/tracer.go](./tracer/tracer.go) and [tracer/firehose_tracer.go](./tracer/firehose_tracer.go) a sketch of such "plugin" in Golang, `geth` can be inspected to see a full fledged block synchronization tracing plugin in a production codebase.

The output format must strictly respect https://github.com/streamingfast/firehose-core standard, the [tracer/firehose_tracer.go](./tracer/firehose_tracer.go) implementation shows how we suggest implementing such tracer, you are free to implement the way you like.

## Building

Clone the repository:

```bash
git clone https://github.com/streamingfast/dummy-blockchain.git
cd dummy-blockchain

go run . start
```

## HTTP API

Dummy chain comes equipped with a simple HTTP API to check on status and blocks.

API server starts on `0.0.0.0:8080` by default.

List of available endpoints:

- `/`               - Readme page
- `/status`         - Get chain status
- `/block`          - Get block for latest height
- `/blocks/:height` - Get block for a specific height

## Contributors

- [Figment](https://github.com/figment-networks): Initial Implementation

## License

Apache 2.0
