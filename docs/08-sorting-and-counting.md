---
layout: default
title: Sort, deduplicate, and count
nav_title: Sort, unique, count
nav_order: 8
section: Advanced queries
read_time: 5 min read
summary: Understand when a stream becomes an array and which count or length you need.
permalink: /08-sorting-and-counting.html
---

# Sort, deduplicate, and count

This page uses **`examples/services.json`**. Start a fresh session for each recipe.

```sh
./bin/qbx examples/services.json
```

## Sort a result sequence

Highlight `services` and choose **Iterate elements**. You now have three independent service results. Press `o`, set **Key** to `/replicas`, set **Direction** to `descending`, and apply. The output is one array ordered `worker` (8), `api` (3), then `web` (2). The form defaults to ascending order when you do not change Direction.

The pipeline shows **Collect** immediately before **Sort**. It is a visible, undoable change. Qubex must surround the values produced by iteration before sorting; adding a trailing `[]` would split results again. If your current stage already holds one array value, no extra collection is needed.

For more complex pipelines with multiple iterations, the sort form may offer a **Boundary** choice. Select the collection whose elements should be ordered. Inspect the generated jq and output to confirm that you chose the intended level.

## Unique values or unique by a key

Start again, iterate `services`, and choose `?` → **Unique current results**. Leave **Key** empty to deduplicate whole service objects. Set **Key** to `/language` to keep one object per distinct language. A stream is collected first, just as with sorting. Unique operations use jq’s sorting semantics; do not treat them as a stable “first occurrence” filter.

On an existing array value, **Unique current results** works on that array directly. The same menu action therefore depends on the current value’s shape and whether the query has emitted a sequence.

## Count results versus measure a value

Start again and iterate `services`. Choose `?` → **Count current results**. The result is `3`, the number of emitted services. If you first filter for `python`, the count becomes `2`. A count always uses the full relevant input, not just the display preview.

Start yet again. Highlight the **Source** `services` array and choose **Length of value**. Qubex first selects the array, then measures it; the result is also `3`, but for a different reason. On a string, length measures that string; on an object, it measures its key count. After iteration, length applies to **each** emitted value, while count totals the result sequence.

One JSON array value and three separate results may look similar in a tree. The Output panel labels a sequence `RESULT 1`, `RESULT 2`, and so on. Only a query that actually creates an array displays array brackets.

Next: [Edit the pipeline](09-pipeline-editing.md).
