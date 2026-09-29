# CHANGELOG

## Unreleased

### Changed

* Bump Go to 1.27 (go.mod and Dockerfile), CI now derives its Go version from `go.mod` instead of a hardcoded matrix value.
* Bump GitHub Actions (`checkout`, `setup-go`, `cache`, `setup-buildx-action`, `login-action`, `metadata-action`, `build-push-action`) to their latest majors to drop the Node.js 20 runtime deprecation warning.

## 1.7.8

### Fixed

* Fix flash blocks sometimes missing a partial (1, 2 or 3) because the block and flash tickers could fire at the same instant and race through `select`; block production now drives both from a single ticker with a deterministic slot counter.
* Track `firehose-core` `latest` in the Docker image, picking up hub/forkable fixes for partial blocks around forks.

## 1.7.7

* Updating to latest `firehose-core` version.

## 1.7.6

* Send 'lastFinal' signal at every block by doing 'idx+1000' -- except multiples of 11
* Fix partial blocks sometimes having the same ID as a 'wrong full block' when running with-reorgs
* Fix 'burst' handling with reorgs: now behaves as expected (300 blocks + reorgs will bring you to block 300, no reorgs sent during burst)

## 1.7.5

- Add a `data` field in transaction to more easily create representative transactions.

- Fixed stop block not properly working in all cases.

- Improved representativity of created blocks against a real chain.

- Greatly improved speed at which big block (> 1MiB) are created.

- Flag `--block-size` now accepts human bytes representation (e.g. `4KiB`, `32mb`, `10 MiB`) and underscores to separate integers.

- Put back `firehose-core@latest` as Docker image to pick now that flash blocks is merged.

- Add a bit of variability in generated transaction content (for more real-world alignment).

## 1.7.4

- Fixed `--with-flash-blocks` ordering (ex blocks: `2.full, 3.p1, 3.p2, 3.p3, 3.p4, 3.full`)

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
