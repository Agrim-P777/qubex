# Contributing to Qubex

Thanks for your interest in improving Qubex! This document covers how to build,
test, and submit changes.

## Prerequisites

- Go 1.26.8 (the `Makefile` also recognizes a workspace-local toolchain under
  `.tools/go`).
- Python 3 (only for the PTY smoke tests and the third-party-notices script).
- Optional: `jq` 1.8.1 on your `PATH` for the differential tests.

## Build and test

```sh
make build                      # build ./bin/qbx
make test                       # go test ./...
make check                      # go vet + go test -race ./... + jq differential tests
QBX_REQUIRE_JQ=1 make check     # make a missing jq reference a hard failure
make pty                        # real-terminal (PTY) smoke tests
make bench                      # 5/50/500 MiB throughput profiling
```

Please run `make check` (ideally with `QBX_REQUIRE_JQ=1`) and `gofmt -l .`
before opening a pull request. CI runs the same checks on Linux and macOS.

## Pull requests

1. Fork and create a topic branch.
2. Keep changes focused; add or update tests alongside behavior changes.
3. Ensure `gofmt`/`go vet` are clean and the full suite passes.
4. Write a clear PR description explaining the motivation and the change.

## Reporting bugs and requesting features

Use the GitHub issue templates. For security-sensitive reports, see
[SECURITY.md](SECURITY.md) instead of filing a public issue.

## Questions

Open a GitHub issue or reach out at agrimbuilds@gmail.com.
