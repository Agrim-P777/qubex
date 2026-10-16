---
layout: default
title: Keyboard and operations reference
nav_title: Keys and actions
nav_order: 13
section: Reference
read_time: 5 min read
summary: All interactive shortcuts and the operation palette in one quick reference.
permalink: /13-keyboard-and-operations.html
---

# Keyboard and operations reference

This is a reference page. For a step-by-step exercise, use [your first query](02-first-query.md) with `examples/services.json`.

## Main view

| Key | Action |
| --- | --- |
| Up/Down or `k`/`j` | Previous/next tree row |
| Left/Right or `h`/`l` | Collapse/expand, or enter the first child |
| `Enter` | Expand/collapse; does not change the query |
| `g`/Home, `G`/End, PageUp/PageDown | Root, end, and page navigation |
| `?` | Open searchable context-sensitive actions |
| `f`, `m`, `o`, `p` | Filter, extract-and-transform, sort, path actions |
| `v` | Switch left tree between Source and Stage |
| `Tab`, `Ctrl+O` | Focus Output; cycle Output panel layout |
| `/`, `n`, `N` | Search; next/previous result |
| `w`, `J` | Wrap output; switch output compact/pretty |
| `[`, `]` | Scroll the generated jq line horizontally |
| `s`/`Ctrl+S`, `Ctrl+H` | Save; browse history |
| `c` | Copy jq, or selected JSON when Output is focused |
| `u`/`Ctrl+U`, `Ctrl+R` | Undo; redo |
| `Ctrl+P` | Edit the query pipeline |
| `q`, `Ctrl+C` | Quit; interrupt |

In menus, Up/Down chooses an item, `/` searches actions, `Enter` selects, `?` shows the selected item’s detail, and `Esc` closes the menu. In forms, `Tab`/Up/Down changes fields, Left/Right changes fixed choices, `Enter` applies, and `Esc` cancels. In the multi-field checklist, `Space` checks a field; `/` searches fields and `Ctrl+N` adds the exact typed name.

## Palette actions

| Action | What it changes |
| --- | --- |
| **Extract value** / **Iterate elements** | Select one path / emit each array element |
| **Collect results** / **Count current results** | Make a result sequence one array / count emitted results |
| **Length of value** | Measure each selected array, object, string, or number per jq semantics |
| **Select multiple fields** / **Map elements** | Construct an object / operate on each element inside one array |
| **Filter** | Keep values satisfying a comparison, string test, existence/type check, or nested Any/All |
| **Extract and transform** | Return only the selected transformed value |
| **Update field in object** | Transform one field and keep its containing object |
| **Sort current results** | Sort an array; insert visible Collect first for a result stream |
| **Unique current results** | Deduplicate an array by whole value or a key |
| **Copy path** / **Copy value** | Request clipboard copy for the selected tree location |
| **Expand all** / **Collapse all** | Change tree navigation only |
| **Query pipeline** / **Saved query history** | Edit stages / preview and restore saved definitions |
| **Preview limits** / **Rerun query** | Change live-preview caps / execute the current query again |
| **Switch Source / Stage** / **Help** | Change the left tree / open built-in help |

Operations appear only when relevant to the selected value and current stage. The [filters page](05-filters-and-guards.md) lists every predicate operator; the [transforms page](07-transforms.md) lists every text and numeric transform. These actions build semantic operations first; the jq compiler produces the visible query.

## Pipeline editor

Press `Ctrl+P`. Use Up/Down to select a step; `Enter` edits, `x` or Delete removes, `K`/`J` reorders, `i` inspects the stage through that step, `c` copies its jq expression, and `Esc` closes. Applying a form is one committed revision. An invalid downstream step remains available for repair.

## Path and search reminders

Form paths are JSON Pointers such as `/owner/display name`; an empty path targets the current value. Escape a literal `~` as `~0` and a literal `/` as `~1`. Numeric segments are indexes only when the current shape identifies an array.

Tree search (`/`) offers fuzzy/exact mode, key/value target, document/subtree scope, and case setting. `Esc` cancels an open search form. After `Enter` runs it, the form closes and `n`/`N` remain available; a new search replaces the matches.

Return to the [documentation overview](index.md).
