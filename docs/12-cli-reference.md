---
layout: default
title: CLI reference
nav_title: CLI reference
nav_order: 12
section: Reference
read_time: 5 min read
summary: Every qbx command, option, output rule, and exit code in one place.
permalink: /12-cli-reference.html
---

# CLI reference

This reference uses **`examples/services.json`** for its command examples. Commands run from the repository root and use `./bin/qbx`; an installed executable can be called `qbx`.

## Commands

| Command | Behavior |
| --- | --- |
| `qbx [file]` | Open the interactive builder; without a file, read one piped JSON document |
| `qbx history [--json]` | List saved definitions |
| `qbx history search TEXT [--json]` | Fuzzy-search names and generated jq |
| `qbx history show ID` | Print one saved record as JSON |
| `qbx history open ID [file]` | Restore an ID into the TUI on newly supplied JSON |
| `qbx history run ID-or-NAME [file]` | Execute an ID or exact name without the TUI |
| `qbx history copy ID` | Request terminal clipboard copy of its jq |
| `qbx history delete ID` | Delete one saved definition |
| `qbx history export ID --format FORMAT` | Export generated jq, editable IR, or a readable step list (`FORMAT` is `jq`, `json`, or `text`) |

The `--json` flag applies to `history` listing and `history search`. The default export format is `jq`. Only **`history run`** accepts a name; the other history commands require a numeric ID. Names are case-sensitive and must match exactly for `run`. If duplicates exist, use an ID.

```sh
./bin/qbx examples/services.json
cat examples/services.json | ./bin/qbx
./bin/qbx history run 'filter python' examples/services.json
./bin/qbx history run 4 < examples/services.json
```

The last two commands require a matching query to have been saved already; replace `4` with the ID printed by your own `history` list. `open` and `run` never reopen a remembered source filename. If no file is supplied, they require piped JSON.

## Options

| Option | Default | Purpose |
| --- | --- | --- |
| `--input-limit BYTES` | `524288000` (500 MiB) | Maximum input; accepts integers and `KiB`/`MiB`/`GiB` suffixes |
| `--preview-timeout TIME` | `2s` | TUI preview deadline; Go-style durations such as `10s` |
| `--preview-results N` | `10000` | Maximum results kept for the TUI preview |
| `--preview-bytes BYTES` | `8388608` (8 MiB) | Displayed JSON budget for the TUI preview |
| `--help` / `-h` | — | Print usage |
| `--version` | — | Print the build version |

Preview options do not truncate `history run`. The input limit applies to both interactive and headless reads. Options can appear before or after subcommands. Use `./bin/qbx --help` to see the build’s own usage text.

## Stdout, stderr, and exit status

Headless `history run` writes each complete result as indented JSON to **stdout**. More than one result is a sequence of JSON values, not an implicit array. It uses jq-style ANSI colors only when stdout is a terminal; pipes and files receive plain JSON. Diagnostics go to **stderr**. No external `jq` process is launched; the generated jq runs in embedded gojq.

| Code | Meaning |
| --- | --- |
| `0` | Success, including a successful query with no emitted values |
| `2` | Invocation or input failure |
| `3` | Query compilation or execution failure |
| `4` | History or output I/O failure |
| `130` | Interruption |

If execution fails after emitting earlier values, those values may already be on stdout. Check the exit code before treating the whole result as successful.

Next: [Keyboard and operations reference](13-keyboard-and-operations.md).
