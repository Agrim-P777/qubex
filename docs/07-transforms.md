---
layout: default
title: Transform values
nav_title: Transform values
nav_order: 7
section: Advanced queries
read_time: 5 min read
summary: Choose between extracting a transformed value, updating an object, and mapping an array.
permalink: /07-transforms.html
---

# Transform values

Use **`examples/records.json`** for every recipe here. Each name has surrounding spaces; two scores are numbers and one is the string `"n/a"`.

```sh
./bin/qbx examples/records.json
```

## Update a name but retain its record

Highlight `records` and choose **Iterate elements**. Expand the first record in Source and highlight its `name`. Choose `?` → **Update field in object**, set **Transform** to `trim`, and apply. Qubex updates `name` in each current record while retaining `id`, `score`, `owner`, and the other fields. The first result’s name changes from `" api "` to `"api"`.

Start again and use `m` or **Extract and transform** on `name`, also choosing `trim`. This emits just the transformed strings `"api"`, `"worker"`, and `"web"`. The distinction is crucial: extraction changes the result shape, while an in-object update preserves sibling fields for later steps.

## Map the array in place

Start again. On the `records` array, choose `?` → **Map elements**. Set **Array** to `/records`, **Mode** to `update`, **Element path** to `/name`, and **Transform** to `trim`. Apply. The result stays **one array** of three complete objects, with all names trimmed. `/name` is relative to one element, not the document root.

Map also offers `extract`, `project`, and `transform` modes. For `project`, enter **Fields JSON** as a JSON array such as `["id","name"]`. For `transform`, use **Element path** when you want to transform one value inside each element.

## Text and numeric transforms

| Transform | Input and form values |
| --- | --- |
| `upper` / `lower` | ASCII-only case change on a string |
| `trim` | Remove leading and trailing whitespace |
| `replace` | **Argument** is a nonempty literal search string; **Replacement** is the new text |
| `split` | Split a string using **Argument** as separator |
| `join` | Join an array of strings using **Argument** as separator |
| `add` / `subtract` / `multiply` / `divide` | **Argument** must be a number |
| `floor` / `ceil` / `round` | Round a number; no argument |

For a numeric example, start again and iterate `records`. Filter with **Path** `/score`, **Operator** `type`, **Value type** `string`, and **Value** `number`. Then select `score` on a remaining Stage record, choose **Update field in object**, set **Transform** to `add` and **Argument** to `2`. The two numeric scores become 5 and 10. The type filter explicitly excludes the `"n/a"` record; without it, arithmetic on that string would fail.

Transforms use jq semantics. Literal replacement is not a regular-expression search. Division, rounding, and very large numbers can differ at numeric edges between gojq and a system `jq` executable.

Next: [Sort, deduplicate, and count](08-sorting-and-counting.md).
