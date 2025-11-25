# CHANGELOG

## 1.7.3

- Added `--with-flash-blocks` flag to send flash blocks (4 per block)

## 1.7.2

- Added `--with-skipped-blocks` flag to control behavior of skipping blocks that are multiples of 13.
- Added `--with-reorgs` flag to control behavior of producing 2 block reorg sequences every 17 blocks.

## 1.7.1

- Fixed `--log-level` not being picked up correctly.

## 1.7.0

- Added `--block-size` (in bytes) controlling approximatively how big blocks get.

- Deprecating and ignoring `--genesis-height` and `--genesis-time` flag, they are now hard-coded so Firehose validation for chain is always correct.

## 1.6.1

- Re-added support for CLI to live also at root of project so that old `go install github.com/streamingfast/dummy-blockchain@latest` still works.

## 1.6.0

- Move command line to `./cmd/dummy-blockchain` and bumped Golang version.

## 1.0.0 - 1.5.1

- No release notes available.

## 1.0.0

- Align with Firehose 1.0.0 version.
