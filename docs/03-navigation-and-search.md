---
layout: default
title: Navigate and search the interface
nav_title: Navigation and search
nav_order: 3
section: Working in the TUI
read_time: 5 min read
summary: Know which tree you are browsing, how focus works, and how to leave search.
permalink: /03-navigation-and-search.html
---

# Navigate and search the interface

Open **`examples/services.json`** again. This page uses that file only.

```sh
./bin/qbx examples/services.json
```

## Source, Stage, and Output

**Source** is the original immutable JSON. Press `v` to switch the left-hand tree between Source and **Stage**, which contains values at the current query stage. The **Output** panel is a separate result-inspection tree. After a committed query, Stage and Output usually hold the same values, but they have independent cursors. During a form’s temporary preview, Output can show draft values while Stage still represents the committed query.

Press `Tab` to focus Output and `Tab` again to return to the left tree. Press `Ctrl+O` to cycle the Output panel through expanded, hidden, and collapsed layouts. In a narrow terminal, the focused tree fills the available width.

The **PATH** footer shows the cursor’s absolute location in the visible tree; **TARGET** shows the relative path where the next operation will apply. After projection, sorting, or transformation, a Source field may no longer map safely to the current result. Switch to Stage with `v` and select the field there.

## Move without changing the query

Use Up/Down or `j`/`k` to move; Left/Right or `h`/`l` to collapse and expand. `Enter` toggles expansion. `g` or Home jumps to the root; `G` or End jumps to the end; PageUp/PageDown move a screen. The `?` palette also offers lazy **Expand all** and **Collapse all**. Expansion changes only the view, never the jq.

Try selecting `services`, then expanding `[0]` and its `language` field. The query line remains `.` until you choose an operation. Use `p` for path actions such as **Extract current path** or copying its jq path/JSON Pointer.

## Search keys and values

Press `/` in the tree. Enter `language` and press `Enter`. The search form closes and the cursor jumps to the first match. Press `n` for the next match or `N` for the previous one. The search form can choose fuzzy or exact matching, keys or values, document or subtree scope, and case sensitivity. It reports up to 1,000 matches.

Press `Esc` **before** submitting a search to cancel the form. After `Enter`, normal navigation has already resumed; `n`/`N` continue cycling the saved matches. There is currently no separate “clear matches” key; starting another search replaces the match list. Searching does not edit the query.

The `?` palette also has its own `/` search. `Esc` closes that menu. In a multi-field checklist, `/` filters field names; `Enter` leaves the checklist search, and `Esc` first leaves that search.

## Inspect long output

Use `w` to toggle wrapping in the result tree and `J` to switch between compact and pretty display. `[`/`]` scroll a long generated jq line horizontally. With Output focused, `c` copies its selected JSON value; otherwise, `c` copies the full generated jq. The [keyboard reference](13-keyboard-and-operations.md) lists every shortcut.

Next: [Choose fields and collections](04-fields-and-collections.md).
