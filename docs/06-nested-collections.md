---
layout: default
title: Filter nested arrays
nav_title: Nested Any and All
nav_order: 6
section: Advanced queries
read_time: 5 min read
summary: Retain a parent object when any or all nested elements match a condition.
permalink: /06-nested-collections.html
---

# Filter nested arrays

This page uses only **`examples/projects.json`**. It has three projects: `api` has production and urgent tags; `demo` has a development tag; `empty` has no tags or tasks.

```sh
./bin/qbx examples/projects.json
```

There are two different questions you can ask. **Filter array elements** emits individual matching elements. **Any/All** keeps a *parent project* based on its nested array. Qubex makes you choose which meaning you intend.

## Keep projects with any production tag

Highlight `projects` and choose `?` → **Iterate elements**. Press `v` to browse the current project objects in Stage. Select a project object and press `f`. Set **Path** to `/tags/0`, **Operator** to `eq`, **Value type** to `string`, and **Value** to `prod`.

The form adds **Array .tags** because that pointer crosses an array. Choose `any` and apply. The output is one project result, `api`. The `0` in the pointer helps identify an element path while building the predicate; with `any`, the test runs against **every tag** and succeeds if at least one matches. It does not mean “check only tag zero.”

## Keep projects whose tasks are all done

Start a fresh session, iterate `projects`, and filter from a Stage project object. Use **Path** `/tasks/0/status`, **Operator** `eq`, **Value type** `string`, and **Value** `done`. Choose `all` for **Array .tasks**.

The result sequence contains `demo` and `empty`. `api` has a `todo` task, so it fails. `empty` passes because **All of an empty collection is true**. By contrast, **Any of an empty collection is false**. These are jq’s empty-collection semantics.

## Let the scope chooser help

You can also press `f` while pointing inside a nested array in **Source**, before explicitly iterating `projects`. If the scope is ambiguous, Qubex opens **Choose filter scope**. **Filter current query values** tests the current parent values. **Filter elements of .projects** inserts iteration first, then filters project objects. A nested Any/All choice still appears when the predicate crosses `tags` or `tasks`.

After filtering or sorting, do not assume Source array indexes still identify the same current results. Switch to **Stage** (`v`) when you need a field from the current values. If a source field was removed by an earlier projection, Stage or an earlier pipeline step is the correct place to edit it.

> With more than one nested array, the form asks for an explicit Any or All choice at **each** crossed collection boundary.

Next: [Transform values](07-transforms.md).
