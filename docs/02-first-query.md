---
layout: default
title: Your first query
nav_title: First query
nav_order: 2
section: Getting started
read_time: 5 min read
summary: Filter Python services, select fields, and sort them without typing jq.
permalink: /02-first-query.html
---

# Your first query

Use only **`examples/services.json`** on this page:

```sh
./bin/qbx examples/services.json
```

The file has three services: `api` and `worker` use `python`, while `web` uses `go`. We will keep the two Python services, select their names and replica counts, and sort by replicas.

## 1. Iterate the array

In the **Source** tree, highlight `services`. Press `?` to open the action palette, move to **Iterate elements**, and press `Enter`. You can press `/` inside the palette and type part of the action name to find it.

The generated query now begins with `.services[]`. The output shows three separate results. Merely pressing `Enter` on the array would expand it for browsing; it would **not** add iteration to the query.

## 2. Keep Python services

Expand `[0]` under `services` and highlight `language: "python"`. Press `f`. Check the form:

| Field | Value |
| --- | --- |
| Path | `/language` |
| Operator | `eq` |
| Value type | `string` |
| Value | `python` |

Use `Tab` to move between fields and Left/Right to change a choice. Press `Enter` to apply. Only `api` and `worker` remain. The value is lowercase because JSON string comparison is case-sensitive. To exclude Python instead, use `ne`.

The generated jq is equivalent to `.services[] | select(.language == "python")`. Qubex may add parentheses when displaying its deterministic compiler output.

## 3. Select two fields

Press `?` and choose **Select multiple fields**. Highlight `name` and `replicas` in the checklist, toggling each with `Space`. Press `Enter`. Each output object now has those two fields.

The checklist is based on sampled fields. If a field is missing from suggestions, press `/` in the checklist, type its exact name, and use `Ctrl+N` to add it.

## 4. Sort the result sequence

Press `o`. Set **Key** to `/replicas`, move to **Direction**, choose `descending`, and press `Enter`. Qubex visibly inserts **Collect** before **Sort** because sorting needs one array, rather than independent objects.

Your final output is one array:

```json
[
  {"name": "worker", "replicas": 8},
  {"name": "api", "replicas": 3}
]
```

Its jq is equivalent to:

```jq
[.services[] | select(.language == "python") | {name, replicas}]
| sort_by(.replicas) | reverse
```

Press `c` while the data tree is focused to copy the generated jq, or `s` to save the editable query. [History and reuse](10-history-and-reuse.md) shows how to run it on another file.

Next: [Navigate and search the interface](03-navigation-and-search.md).
