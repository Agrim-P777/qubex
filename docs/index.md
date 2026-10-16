---
layout: default
title: Qubex documentation
nav_title: Overview
nav_order: 0
section: Start here
read_time: 3 min read
summary: A short route from your first JSON file to reusable jq queries.
permalink: /index.html
---

# Build jq queries by navigating JSON

Qubex (`qbx`) is a terminal query builder. Browse a JSON document, choose operations, and watch the generated jq and results update. The query is stored as editable operations; jq is the first output format. Execution uses embedded **gojq**, so an installed `jq` binary is not needed.

The tutorials use real files shipped in the repository. Run commands from the repository root after [building `qbx`](01-install-and-input.md). Each walkthrough sticks to **one sample JSON file**. Start a fresh session when a page says “start again”; that keeps its expected output easy to compare.

## A practical reading path

1. [Install and open JSON](01-install-and-input.md) — build the binary, load a file, or pipe one document.
2. [Build your first query](02-first-query.md) — filter Python services, project fields, and sort.
3. [Navigate and search](03-navigation-and-search.md) — understand Source, Stage, Output, focus, and search.
4. [Choose fields and collections](04-fields-and-collections.md) — extract, iterate, collect, project, and map.
5. [Filter and guard mixed data](05-filters-and-guards.md) — comparisons, text tests, existence, and types.
6. [Filter nested arrays](06-nested-collections.md) — choose a scope and use Any or All.
7. [Transform values](07-transforms.md) — extraction, in-object updates, array mapping, and arithmetic.
8. [Sort, deduplicate, and count](08-sorting-and-counting.md) — collection boundaries and cardinality.
9. [Edit the pipeline](09-pipeline-editing.md) — inspect, repair, reorder, undo, and redo.
10. [Save and reuse queries](10-history-and-reuse.md) — history in the TUI and headless replay.

Use [configuration and troubleshooting](11-configuration-and-troubleshooting.md), the [CLI reference](12-cli-reference.md), and the [keyboard and operation reference](13-keyboard-and-operations.md) when you need a specific setting or shortcut.

## The three sample files

| File | Used for |
| --- | --- |
| `examples/services.json` | First query, navigation, collection operations, sorting, pipeline editing, history |
| `examples/records.json` | Mixed field types, guards, string and numeric transformations |
| `examples/projects.json` | Nested Any/All and empty-collection behavior |

The hosted site provides [services.json](examples/services.json), [records.json](examples/records.json), and [projects.json](examples/projects.json) as downloads. In a clone, these same files live under the repository’s `examples/` directory.

> **Key idea:** Moving the cursor never edits a query. A new session begins with `.`, which returns the complete input. Choosing an operation changes the query.

Next: [Install and open JSON](01-install-and-input.md).
