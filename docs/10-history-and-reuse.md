---
layout: default
title: Save and reuse queries
nav_title: History and reuse
nav_order: 10
section: Reuse
read_time: 5 min read
summary: Save an editable query and run it later by ID or exact name.
permalink: /10-history-and-reuse.html
---

# Save and reuse queries

Use **`examples/services.json`** for this walkthrough:

```sh
./bin/qbx examples/services.json
```

Highlight `services` and choose **Iterate elements**. Select `language` under the first service and press `f`. Keep `/language` equal to the string `python`, then apply. Two separate service objects remain.

## Save from the TUI

Press `s` or `Ctrl+S`. Enter the label `filter python` and press `Enter`. The notice gives a numeric history ID. Query definitions are saved only on this explicit action. History contains the operation model, generated jq, label, timestamps, and minimal source metadata; it does **not** contain the JSON document. Literal values such as `python` are part of the saved query.

Press `Ctrl+H` to open the history browser. Choose an entry and press `Enter` to preview it; press `Enter` again on **Restore this query** to make it the editable current pipeline. `Esc` leaves the preview without restoring.

## Run the saved query on JSON

From the repository root:

```sh
./bin/qbx history
./bin/qbx history run 'filter python' examples/services.json
```

`history run` uses embedded gojq. It prints each matching object as indented JSON. In a terminal, keys, strings, and other JSON tokens use jq-style colors; redirected or piped output is plain JSON without ANSI escape codes. Two emitted objects remain **two JSON results**, not one fabricated array.

Both shell quote styles work around a name with spaces: `'filter python'` and `"filter python"` pass the same name. Name lookup is **exact**. If two saved queries share a name, Qubex reports the matching IDs and asks you to use one. Numeric IDs take precedence for numeric arguments.

You can also pipe a fresh document into a saved query:

```sh
cat examples/services.json | ./bin/qbx history run 'filter python'
```

`history run` has no live-preview timeout or display truncation. It writes values as they are produced, so an error after earlier results can leave those earlier results on stdout; check the exit status in scripts.

## Inspect, reopen, and export

Replace `ID` below with the number shown by `history`:

```sh
./bin/qbx history search python --json
./bin/qbx history show ID
./bin/qbx history open ID examples/services.json
./bin/qbx history export ID --format jq
./bin/qbx history export ID --format json
./bin/qbx history export ID --format text
./bin/qbx history copy ID
./bin/qbx history delete ID
```

`open` launches the TUI with the saved editable operations. It accepts an **ID**, and it needs a new file or piped JSON; it never silently reopens the old source path. `export --format json` emits the versioned operation model, not source data. `copy` requests the terminal clipboard; `export --format jq` is the text fallback.

Next: [Configuration and troubleshooting](11-configuration-and-troubleshooting.md).
