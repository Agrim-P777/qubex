---
layout: default
title: Edit the pipeline
nav_title: Pipeline editing
nav_order: 9
section: Working in the TUI
read_time: 5 min read
summary: Inspect intermediate results, edit or move operations, and recover with undo.
permalink: /09-pipeline-editing.html
---

# Edit the pipeline

Use **`examples/services.json`**. Rebuild the [first query](02-first-query.md): iterate `services`, filter `language == "python"`, project `name` and `replicas`, then sort by `replicas` descending.

```sh
./bin/qbx examples/services.json
```

Open the pipeline with `Ctrl+P`. Each row is an operation in the editable query model. **Collect** is visible because the sort operates on the emitted service results.

## Inspect without changing the query

Use Up/Down to select the filter step and press `i`. Stage now shows the result **after that step**. The saved pipeline still includes all later operations. Press `Esc` to leave inspection and return to the full query. This is useful when a later projection has hidden a field you need to reason about.

If you choose an action while inspecting an intermediate stage, Qubex edits at that stage and keeps later steps, then revalidates them. Use the generated jq and output to assess the resulting pipeline.

## Edit, move, delete, or copy a step

Open `Ctrl+P` again. On a selected row:

| Key | Effect |
| --- | --- |
| `Enter` | Edit the step in a typed form |
| `x` or Delete | Delete the step |
| `K` / `J` | Move the step up / down |
| `i` | Inspect the result through this step |
| `c` | Copy the step’s jq expression |
| `Esc` | Close the pipeline editor |

For example, edit the filter’s Value from `python` to `go` and press `Enter`. The later projection and sort remain in place. Press `Esc` while still in the form to discard a draft instead.

## Undo, redo, and invalid downstream steps

Press `u` or `Ctrl+U` in the main view to undo one committed action; press `Ctrl+R` to redo. Sorting a stream and inserting Collect are one undoable action. Navigation and search do not enter undo history because they do not change the query.

Form edits generate temporary previews after about 150 ms of quiet typing. Applying a form commits one revision; cancelling it commits none. If editing, deleting, or moving a step makes a later step invalid, Qubex retains the steps for repair, pauses execution, and labels the visible last successful result as belonging to an earlier revision. Use the pipeline editor to repair the affected step, or undo.

The query is saved to history **only** when you explicitly press `s` or `Ctrl+S`. Exiting the TUI does not silently save your latest edits.

Next: [Save and reuse queries](10-history-and-reuse.md).
