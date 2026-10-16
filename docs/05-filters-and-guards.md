---
layout: default
title: Filter and guard mixed data
nav_title: Filters and guards
nav_order: 5
section: Building queries
read_time: 5 min read
summary: Use comparison and string filters safely when records have different field types.
permalink: /05-filters-and-guards.html
---

# Filter and guard mixed data

Use **`examples/records.json`** for every recipe on this page. It has three records. `score` is numeric for `api` and `worker` but the `web` record uses `"n/a"`; `note` is a string for two records and `null` for the third.

```sh
./bin/qbx examples/records.json
```

First highlight `records` and choose `?` → **Iterate elements**. Each filter below starts from that point in a **fresh session**, unless a step says otherwise.

## Keep a numeric score above five

Switch to **Stage** with `v`, select the first record object, and press `f`. Set:

| Field | Value |
| --- | --- |
| Path | `/score` |
| Operator | `gt` |
| Value type | `number` |
| Value | `5` |
| Guard | `type` |

Apply with `Enter`. Only `worker` (score 8) remains. The explicit type guard checks that `score` is a number before comparing it. Without that guard, jq’s ordering across different JSON types can produce a surprising comparison result. A guard is visible in the generated jq; Qubex does not silently skip mixed records.

## Search inside text

Start again, iterate `records`, and filter with **Path** `/note`, **Operator** `contains`, **Value type** `string`, **Value** `API`, and **Guard** `type`. Only `api` remains. The guard matters because `web` has a null `note`, and string containment on null is a jq type error.

For a prefix or suffix, use `startswith` or `endswith`. String values in the form are entered as text (for example, `API`), **without** JSON quote marks. Matching is case-sensitive.

## Check presence or type explicitly

The `api` record has `"discount": null`, while the other two omit `discount`. Filter on `/discount` with **Operator** `exists` to keep `api`. An existing null field counts as present. `exists` does not use the Value field.

Alternatively, filter on `/score` with **Operator** `type`, **Value type** `string`, and **Value** `number`. This keeps the two records whose scores are numeric. Here `number` names the JSON type; it is not a numeric literal.

## Filter form reference

| Operator | Meaning |
| --- | --- |
| `eq` / `ne` | Equals / does not equal |
| `gt` / `ge` / `lt` / `le` | Greater than / at least / less than / at most |
| `contains` / `startswith` / `endswith` | Value contains text / starts with / ends with |
| `exists` / `type` | Path is present / path has a named JSON type |

Value types can be `string`, `number`, `boolean`, `null`, or `json`. For `json`, enter one valid JSON value. **Guard** offers `strict` (default), `exists`, or `type`. “Strict” means ordinary jq behavior, including null and type errors. A sampled suggestion is a hint, not proof that all records have the same fields.

Next: [Filter nested arrays](06-nested-collections.md).
