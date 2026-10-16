---
layout: default
title: Configuration and troubleshooting
nav_title: Config and limits
nav_order: 11
section: Operate Qubex
read_time: 5 min read
summary: Tune input and preview limits, configure keys, and understand common output states.
permalink: /11-configuration-and-troubleshooting.html
---

# Configuration and troubleshooting

The commands on this page use **`examples/records.json`**. Configuration and behavior apply to any single JSON document.

## Preview versus complete execution

The live preview defaults to a **2-second timeout**, at most **10,000 displayed results**, and **8 MiB** of displayed JSON. These limits bound preview work and retained display output. Sorting, unique, and counting still evaluate the full relevant data; a display sample is never used as their input.

To rerun a slow query, choose `?` → **Preview limits**, increase **Timeout**, **Results**, or **Bytes** (enter a numeric byte count in that form), apply, then choose **Rerun query** if needed. You can also start with flags:

```sh
./bin/qbx --preview-timeout 10s --preview-results 20000 --preview-bytes 16MiB examples/records.json
```

Input has a separate default **500 MiB** limit:

```sh
./bin/qbx --input-limit 1GiB examples/records.json
```

`history run` executes completely and does not inherit preview caps. An operation such as sorting may still require memory proportional to its data; increasing the display budget is not a memory limit.

## Optional configuration

Qubex reads `config.json` from `$XDG_CONFIG_HOME/qubex` when `XDG_CONFIG_HOME` is an absolute path; otherwise it uses `~/.config/qubex`. For example:

```json
{
  "keys": {"ctrl+y": "ctrl+r"},
  "input_limit": 524288000,
  "preview_timeout": "2s",
  "preview_results": 10000,
  "preview_bytes": 8388608
}
```

Key mappings replace an existing **main-view** shortcut. Forms keep their own controls, and `Ctrl+C` always interrupts. Command-line flags override these values for one invocation. Saved history is stored in the same `qubex` directory using a versioned JSON file with locking and atomic replacement.

If the history file is corrupt or uses an unsupported schema version, Qubex reports its path and refuses to overwrite it. Back up that file, repair it or use a compatible Qubex version, then retry. Input documents are never stored in history, but saved query literals remain part of each definition.

## Read the status before changing the query

| Status or message | What to do |
| --- | --- |
| **Complete, empty** | The query succeeded but emitted no results. Check the filter value and case. |
| **Truncated** | Raise the result/byte preview limit, project fewer fields, or use `history run` for full output. |
| **Timed out** | Raise the preview timeout and rerun, or use `history run`. |
| **Runtime error** | Check the operation, value type, and missing fields; add an explicit guard where appropriate. |
| **Invalid pipeline step** | Use `Ctrl+P` to repair it or undo. Last successful output is marked as older. |
| **Source path unavailable** | Press `v` and choose the current field in Stage, or inspect an earlier step. |

Loading and search run asynchronously with cancellation. Object-key order is deterministic in the tree, and field suggestions come from bounded samples rather than a whole-document proof.

## Terminal and clipboard notes

Interactive mode needs a controlling terminal. The TUI tries macOS `pbcopy` first where available and otherwise asks the terminal to copy using OSC 52; `history copy` uses terminal clipboard support. If nothing reaches the clipboard, use visible jq or `history export ID --format jq`. `history run` colors only terminal output; `NO_COLOR` or `TERM=dumb` disables it.

Qubex accepts one JSON document per invocation. This release does not provide NDJSON/live input, manual jq editing, YAML, grouping/general aggregation, or modules and environment access in the embedded runtime.

Next: [CLI reference](12-cli-reference.md).
