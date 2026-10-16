# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Open-source project files: LICENSE (MIT), contributing guide, code of conduct,
  security policy, issue/PR templates, and this changelog.
- `history run` execution limits with `--timeout`, `--max-results`, and
  `--max-bytes` flags; a default timeout now applies in headless mode.

### Changed
- Module path is now `github.com/Agrim-P777/qubex`.
- Sample fixture `data.json` moved to `examples/data.json`.

### Fixed
- Headless `history run` no longer executes without time/result/byte limits.
- gojq execution is panic-guarded so a runtime panic surfaces as a clean error.
- Terminal output of history-derived strings is sanitized against control-byte
  (escape-sequence) injection.

## [0.1.0] - 2026-10-08

### Added
- Initial interactive, structured JSON query builder with a jq-compatible
  compiler, embedded gojq execution, saved-query history, and a Bubble Tea TUI.
