# Dummy Chain

This dummy "blockchain" serves as a demonstration on how to instrument node for the
Firehose integration. Instrumenter, "DeepMind" is a part responsible for data
extraction and could be turned on with an environment variable when running the chain
process.

## Building

Clone the repository:

```bash
git clone https://github.com/streamingfast/dummy-blockchain.git
cd dummy-blockchain
```

Install dependencies:

```bash
go mod download
```

Generate protobuf files:

```bash
make proto
```

Then build the binary:

```bash
make build
```

## Usage

Run `./dummy-blockchain --help` to see list of all available flags:

```
CLI for the Dummy Chain

Usage:
  dummy-blockchain [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  init        Initialize local blockchain state
  reset       Reset local blockchain state
  start       Start blockchian service

Flags:
      --block-rate int        Block production rate (per second) (default 1)
      --genesis-height uint   Blockchain genesis height (default 1)
  -h, --help                  help for dummy-blockchain
      --log-level string      Logging level (default "info")
      --server-addr string    Server address (default "0.0.0.0:8080")
      --store-dir string      Directory for storing blockchain state (default "./data")
  -v, --version               version for dummy-blockchain

Use "dummy-blockchain [command] --help" for more information about a command.
```

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

To enable DeepMind instrumentation:

```
DM_ENABLED=1 ./dummy-blockchain
```

Output will look like:

```
INFO[2022-01-13T11:55:52-06:00] initializing node
INFO[2022-01-13T11:55:52-06:00] initializing store
DEBU[2022-01-13T11:55:52-06:00] creating store root directory                 dir=./data
INFO[2022-01-13T11:55:52-06:00] loading last block                            tip=6
INFO[2022-01-13T11:55:52-06:00] initializing engine
INFO[2022-01-13T11:55:52-06:00] starting block producer                       rate=1s
INFO[2022-01-13T11:55:53-06:00] processing block                              hash=7902699be42c8a8e46fbbb4501726517e86b22c56a189f7625a6da49081b2451 height=7
DMLOG BLOCK_BEGIN 7
DMLOG BLOCK CAcSQDc5MDI2OTliZTQyYzhhOGU0NmZiYmI0NTAxNzI2NTE3ZTg2YjIyYzU2YTE4OWY3NjI1YTZkYTQ5MDgxYjI0NTEaQGU3ZjZjMDExNzc2ZThkYjdjZDMzMGI1NDE3NGZkNzZmN2QwMjE2YjYxMjM4N2E1ZmZjZmI4MWU2ZjA5MTk2ODMqjAEKCHRyYW5zZmVyEkBiMTEwZDg4OWUzNGU2MTdlMmIyYmZmNTdhYWMzNTU3Njc2YzJmNjgxZjM2NWJhZDVhODk2MTVkN2E4MDZmMGY0GgoweERFQURCRUFGIgoweEJBQUFBQUFEKgAyBAoCJxA4AUIcCg50b2tlbl90cmFuc2ZlchIKCgNmb28SA2JhciqSAQoIdHJhbnNmZXISQDBlYzE5MmMwZjkwZDEzMzJmMmFiY2E0Mzk4NTk2ZDM5Nzg0MzRlY2JhZTZhYmVhOGZmZDk4OTQxMmI1OTI0NTgaCjB4REVBREJFQUYiCjB4QkFBQUFBQUQqBgoEO5rKADIECgInEDgBQhwKDnRva2VuX3RyYW5zZmVyEgoKA2ZvbxIDYmFyKpIBCgh0cmFuc2ZlchJAMWFlNWEwYzkwMDE3Mzk4NzllZjgxMmE3Y2IzZjMyOTQyMzNmNTBlNWQxZGJkZTc0NzFiNDUxNjMzMDdjNmNkORoKMHhERUFEQkVBRiIKMHhCQUFBQUFBRCoGCgR3NZQAMgQKAicQOAFCHAoOdG9rZW5fdHJhbnNmZXISCgoDZm9vEgNiYXIqkgEKCHRyYW5zZmVyEkBiNjJmODNhYzc5MmJhYWNkMTdmNDI4NTg1NDM3Yzg0NTY2NjlkMGM1MGNmYjVmZGMxMWM5YTY3NTgxZDgxMzExGgoweERFQURCRUFGIgoweEJBQUFBQUFEKgYKBLLQXgAyBAoCJxA4AUIcCg50b2tlbl90cmFuc2ZlchIKCgNmb28SA2JhciqSAQoIdHJhbnNmZXISQGI5YjUwYzU5ZjQyNTFlOWQyZDRkYzQ5Mjc1ZWM0NzYwYTNjOTcwYTllNWQ5MjU0OGQwNDg5MzIzNDkzYmFkODUaCjB4REVBREJFQUYiCjB4QkFBQUFBQUQqBgoE7msoADIECgInEDgBQhwKDnRva2VuX3RyYW5zZmVyEgoKA2ZvbxIDYmFyKpMBCgh0cmFuc2ZlchJANWUzZjViZDMyMDYxNTQ3ZjdkMTAzNWQ0NDg2NGU5Mjg2YTE1OTRiOWJkMDUyOWMzMTU5ODhkOWNkMDdiYzU5MxoKMHhERUFEQkVBRiIKMHhCQUFBQUFBRCoHCgUBKgXyADIECgInEDgBQhwKDnRva2VuX3RyYW5zZmVyEgoKA2ZvbxIDYmFyKpMBCgh0cmFuc2ZlchJAZmYwM2ViZDU2OWJiZTgzMzg3ZTU2M2NkMTdkZDcxODBiZWI3MmNiOGMyYmZmODY3MDAyYzdhZGQyMjUxNGExMxoKMHhERUFEQkVBRiIKMHhCQUFBQUFBRCoHCgUBZaC8ADIECgInEDgBQhwKDnRva2VuX3RyYW5zZmVyEgoKA2ZvbxIDYmFy
DMLOG BLOCK_END 7
```

Customize DM log output with environment variable:

- `DM_OUTPUT=stdout` - Log to STDOUT (default)
- `DM_OUTPUT=stderr` - Log to STDERR
- `DM_OUTPUT=/path/to/file.log` - Log to regular file

## Running in Docker

Build the docker image:

```bash
make docker-build
```

Start the dummy blockchain process:

```bash
docker run -it streamingfast/dummy-blockchain start
```

## Contributors

- [Figment](https://github.com/figment-networks): Initial Implementation

## License

TBD
