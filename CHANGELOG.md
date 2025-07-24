# CHANGELOG

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
