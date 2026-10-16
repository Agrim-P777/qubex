---
layout: default
title: Choose fields and collections
nav_title: Fields and collections
nav_order: 4
section: Building queries
read_time: 5 min read
summary: Extract a value, iterate or collect results, project objects, and map an array.
permalink: /04-fields-and-collections.html
---

# Choose fields and collections

Every recipe here starts from a fresh session with **`examples/services.json`**:

```sh
./bin/qbx examples/services.json
```

## Extract one value

Expand `services`, then `[0]`, and highlight `name: "api"`. Press `?` and choose **Extract value** (or use `p` → **Extract current path**). The query selects that one path and returns `"api"`. This is a scalar result.

Paths displayed in the tree use jq notation; paths entered in forms use **JSON Pointers** such as `/services/0/name`. An empty form path means “the current value.” The special pointer escapes `~0` and `~1` represent a literal tilde and slash in a key.

## Iterate versus collect

Highlight the `services` array and choose **Iterate elements**. The query emits each object independently, shown as **RESULT 1**, **RESULT 2**, and **RESULT 3**. A result sequence is not a JSON array.

Now press `?` and choose **Collect results**. The three emitted objects become one array result. This boundary matters: a later sort operates over the collected array. You can also select the whole `services` field with **Extract value** to get its original array directly, but that does not create separate per-service stages for filtering.

## Project fields from each service

Start again, iterate `services`, then choose **Select multiple fields** from `?`. Check `name` and `replicas` with `Space` and apply. The operation constructs a new object for **each current result**. The two fields are editable in the pipeline later. If you need to keep the original object and change a field, use **Update field in object** instead.

## Map an array without emitting a stream

Start again and highlight the `services` array. Open `?` → **Map elements**. Keep **Array** as `/services`, choose **Mode** `project`, and set **Fields JSON** to `["name","replicas"]`. Apply. The output is **one array** of three smaller objects, in the original array order.

The map form also supports `extract` (one path from each element), `transform` (replace each element or selected value), and `update` (change an element field while retaining its object). **Element path** is relative to one array element; for example, `/name` points to a service’s name. [Transform values](07-transforms.md) gives a full update example.

> To decide between the two: **Iterate** is useful when later stages filter or emit individual records; **Map** is useful when the current value should stay one array.

Next: [Filter and guard mixed data](05-filters-and-guards.md).
