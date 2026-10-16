# Qubex

![Qubex](docs/assets/qubex_banner.png)

[![CI](https://github.com/Agrim-P777/qubex/actions/workflows/ci.yml/badge.svg)](https://github.com/Agrim-P777/qubex/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8.svg)](go.mod)

**Build jq queries by navigating JSON.** `qbx` keeps a structured query model,
translates it to visible jq, and runs it with an embedded Go interpreter.

<!-- TODO: replace with a TUI demo recording (GIF or asciinema cast). -->
> _Demo recording coming soon._

For short tutorials and references, start with the [documentation pages](docs/index.md).
The [single-file user manual](USER_MANUAL.md) remains available.

## Install

Download a prebuilt binary for your platform from the
[releases page](https://github.com/Agrim-P777/qubex/releases), or install with Go:

```sh
go install github.com/Agrim-P777/qubex/cmd/qbx@latest
```

Or build from source:

```sh
make build
./bin/qbx examples/data.json
cat examples/data.json | ./bin/qbx
```

Builds use Go 1.26.8, Bubble Tea/Bubbles/Lip Gloss v2 and gojq. The binary has
no jq, clipboard-command or CGO requirement. macOS and Linux are supported;
clipboard requests depend on the terminal supporting OSC 52.

## Build and verify

Install Go 1.26.8, then run `make build`. The Makefile also recognizes the
workspace-local toolchain under `.tools/go`. Dependencies and build caches
stay in `.cache/`; exact versions are recorded in `go.mod` and `go.sum`.

```sh
make test                       # semantic, execution, persistence and UI tests
QBX_REQUIRE_JQ=1 make check      # vet, race detector and jq differential tests
make pty                        # real terminal smoke tests (Python 3)
make bench                      # 5/50/500 MiB profiling
make release VERSION=0.1.0       # four CGO-free binaries, checksums and notices
```

The differential tests use `jq` from PATH; CI installs verified jq 1.8.1.
Setting `QBX_REQUIRE_JQ=1` makes a missing reference executable a test failure.
Only the test suite needs jq. PTY tests need a controlling-terminal-capable
environment, including permission to open `/dev/tty`.

## First query

With `examples/services.json`:

1. Move onto `services`, open `?`, and choose **Iterate elements**.
2. Expand an element, move onto `language`, and press `f`. Its path, string
   type and `python` value are prefilled. Press Enter to apply.
3. Open `?`, choose **Select multiple fields**, and check `name` and `replicas`
   with Space. Press Enter.
4. Press `o`. Enter `/replicas` as the key, Tab to Direction, and use the arrow
   keys to select descending. Press Enter. Collect and Sort form one undo step.
5. Press `c` to request clipboard copy, or `s` to save the editable definition.

Generated jq:

```jq
[.services[] | select((.language == "python")) | {"name": .name, "replicas": .replicas}]
| sort_by(.replicas) | reverse
```

The output is one array containing worker/8 followed by api/3. A query that
emits several values is instead displayed as **RESULT 1**, **RESULT 2**, etc.;
the interface does not pretend a result stream is a JSON array.

## Keyboard

| Keys | Action |
| --- | --- |
| Arrows / `j k h l` | Move, collapse, expand |
| Enter | Expand/collapse; never changes the query while browsing |
| Home / `g`, End, PageUp/PageDown | Root, end, paging |
| `?` | Searchable operation palette and help |
| `f`, `m`, `o`, `p` | Filter, extract/transform, sort, path actions |
| `v` | Switch Source / Stage |
| Tab, Ctrl+O | Focus output / cycle hidden, collapsed and expanded output |
| `/`, `n`, `N` | Search, next/previous match |
| `w`, `J` | Output wrapping, compact/pretty display |
| `s` / Ctrl+S | Save current definition |
| `c` | Copy jq; copy selected value when output has focus |
| `u` / Ctrl+U, Ctrl+R | Undo, redo |
| Ctrl+P | Pipeline: Enter edit, `x` delete, `K/J` move, `i` inspect, `c` copy |
| Ctrl+H | Saved history (also available in the palette) |
| `[` / `]` | Scroll long generated queries horizontally |
| Escape | Cancel a form/menu; leave intermediate-stage inspection |
| `q`, Ctrl+C | Quit; interrupt |

Forms use Tab/Up/Down for focus, Left/Right for enumerated options, Enter for
Apply and Escape for Cancel. Text-entry keys stay inside the active input.
Queries preview after a 150 ms typing pause. Incomplete forms never commit.
Field checklists support `/` fuzzy search. Enter leaves search; Space toggles
the highlighted field. Ctrl+N while searching adds the exact entered field name
when it is absent from the sampled suggestions.

Paths in forms are **JSON Pointers**, e.g. `/metadata/name`. `/` means an empty
field name; an empty path means the current value. Escape `~` as `~0` and `/`
inside a field name as `~1`. Numeric segments become indexes only when the
current shape identifies an array. Tab cycles form fields; suggestions remain
visible as hints rather than changing a path implicitly.

## Context and operations

Source is immutable. Stage shows the current query's values. The footer shows
the cursor path and its relative operation target. If a source location cannot
be mapped safely after projection, sorting or transformation, switch to Stage.
Collection-wide actions remain available even if the source cursor points at a
removed field.

Filtering inside multiple arrays asks for the target collection. Retaining a
parent because a nested array matches requires an explicit **any** or **all**
choice for each intervening array. Empty arrays satisfy All and fail Any, as
in jq. Field/type suggestions are sampled, not guarantees about all records.

Filters default to jq behavior, including null/missing-field semantics and
type errors. The Guard field adds an explicit existence or type predicate.
An existing null field counts as present. Numeric comparison operators retain
jq's comparison semantics, including ordering across types unless guarded.

Transforms include ASCII case conversion, whitespace trim, literal replacement,
split/join, arithmetic and rounding. **Extract and transform** returns the
selected transformed value; **Update field in object** retains its container.
Map operates on each array element. Count results and length of a value are
separate operations.

Pipeline edits keep invalid downstream steps for repair. Execution pauses and
the last successful output is labeled as earlier output. Inspecting a step
evaluates its prefix without changing the saved query. Subsequent changes at
that stage retain and revalidate the remaining steps.

## Reuse from the CLI

```sh
qbx history
qbx history search services --json
qbx history show 1
qbx history open 1 examples/services.json
qbx history run 1 < examples/services.json
qbx history run "Python services by replicas" examples/services.json
qbx history export 1 --format jq
qbx history export 1 --format json
qbx history export 1 --format text
qbx history copy 1
qbx history delete 1
```

Exported JSON contains the versioned semantic definition, not source data.
`open` and `run` always use input explicitly supplied now; they do not reopen a
stored path. `run` accepts an exact saved name as well as an ID; if names are
duplicated, use an ID. Shell single and double quotes both pass names with
spaces. Headless runs apply a default 60s timeout (override with `--timeout`)
and otherwise print the full output with indentation; `--max-results` and
`--max-bytes` can cap results or output size. Terminal output uses jq-style
colors; piped output has no ANSI escapes. A later runtime failure can leave
earlier emitted values on stdout; its nonzero exit status indicates failure.

Exit codes: `0` success, `2` arguments/input, `3` query execution, `4`
history/output I/O, `130` interruption. An empty successful result exits `0`.

## Configuration and limits

History/configuration live in `$XDG_CONFIG_HOME/qubex`, or
`~/.config/qubex`. History saves are explicit, locked and atomic. They contain
IR, generated jq, labels, timestamps and a filename basename or `stdin`;
input documents and originating shell commands are never saved. Query literals
are part of saved definitions, so a filter value is retained when you save it.

Optional `config.json`:

```json
{
  "keys": {"ctrl+y": "ctrl+r"},
  "input_limit": 524288000,
  "preview_timeout": "2s",
  "preview_results": 10000,
  "preview_bytes": 8388608
}
```

Keys map a terminal key to an existing main-view shortcut. Forms keep their own
bindings and Ctrl+C always interrupts. Flag overrides include `--input-limit
500MiB`, `--preview-timeout 10s`, `--preview-results 20000` and `--preview-bytes
16MiB`. Limits can also be changed in the operation palette.

The input is decoded in memory once. Tree rows are virtualized, including lazy
Expand All. Sorting/counting always sees the full relevant data, never a
display sample. Preview truncation limits retained/displayed results; it does
not cap memory used by an operation. A single result exceeding the display
budget is reported as truncated; increase the budget or project fewer fields.

## Compatibility and boundaries

Generated operations use standard jq syntax and have differential tests against
jq 1.8.1. gojq preserves large integers more precisely than jq for some
arithmetic. Decimal arithmetic is floating point. Those numeric edge cases are
not an exact-parity guarantee. Modules, OS environment access and external
command execution are not enabled in the embedded runtime.

This release accepts **one complete JSON document**, including JSON supplied by
stdin/process substitution. NDJSON/live input, manual jq editing, grouping,
general aggregations, YAML and other compiler targets are later work.
