# Qubex user manual

Qubex (`qbx`) helps you build jq queries by browsing JSON and choosing actions. It shows the generated jq and previews the result as you work. You do not need to install `jq`: the `qbx` binary runs queries itself.

## Start Qubex

Build from this repository with Go 1.26.8:

```sh
make build
./bin/qbx examples/services.json
```

The examples below use `./bin/qbx`. If you installed the binary on your `PATH`, use `qbx` instead.

You can also pipe **one complete JSON document** into Qubex:

```sh
cat examples/services.json | ./bin/qbx
```

When input comes from a pipe, Qubex reads keyboard input from your terminal. A file may contain an object, array, or scalar. Multiple consecutive JSON documents and NDJSON are not supported in this release.

## Read the screen and build a query

The **Source** tree shows the original JSON. The **Stage** tree shows values produced by the current query; press `v` to switch between them. The output panel shows the live result, and the generated jq remains visible. The footer shows the selected path and where a new action will apply.

Moving through the tree does not change the query. A new session starts with `.`, which returns the whole input. Press `?` to open the actions menu at the selected value, then choose an action. The menu only offers actions that make sense in the current context. For example, an array offers **Iterate elements** and **Map elements**. Press `/` in the menu to search for an action.

#### Form Navigation
Some actions open a form. Use `Tab` or Up/Down to move between fields, Left/Right to change a listed choice, `Enter` to apply, and `Esc` to cancel. Form changes are previewed after a short pause; they become part of the query only when you apply them. If a field is selected in the tree, its path and value are often filled in for you.

| Keys | Action |
| --- | --- |
| Arrows or `j`/`k`/`h`/`l` | Move, collapse, or expand in the tree |
| `Enter` | Expand or collapse the selected tree value |
| `g` or Home | Go to the tree root |
| `?` | Open the context-sensitive actions menu |
| `f`, `m`, `o`, `p` | Filter, transform, sort, or open path actions |
| `v` | Switch between Source and Stage |
| `Tab` | Focus the output panel or return to the data tree |
| `Ctrl+O` | Cycle output panel visibility and size |
| `/`, then `n`/`N` | Search the tree, then visit next/previous match |
| `u` or `Ctrl+U`, `Ctrl+R` | Undo, redo |
| `Ctrl+P` | Open the query pipeline editor |
| `s` or `Ctrl+S` | Save the current query |
| `Ctrl+H` | Browse saved queries |
| `c` | Copy generated jq; with output focused, copy the selected result |
| `q` or `Ctrl+C` | Quit or interrupt |

The clipboard request uses terminal support. If your terminal does not accept it, the generated query is still visible and can be exported from saved history.

## Starter example: extract a field

Create a small input file:

```sh
cat > /tmp/qbx-first.json <<'JSON'
{"service":{"name":"api","replicas":3}}
JSON
./bin/qbx /tmp/qbx-first.json
```

Expand `service`, move to `name`, press `?`, and choose **Extract value**. The result becomes `"api"`. The generated jq is `.service.name`. Notice that expanding `service` and moving to `name` did not change the query; choosing **Extract value** did.

## Starter example: filter, project, and sort services

Run `./bin/qbx examples/services.json`, then:

1. Select `services`, press `?`, and choose **Iterate elements**. The output now contains three separate results, one per service.
2. Expand a service, select `language`, and press `f`. The filter form should show the relative path `/language`, operator `eq`, value type `string`, and value `python`. Press `Enter` to keep matching services.
3. Press `?`, choose **Select multiple fields**, use `Space` to check `name` and `replicas`, and press `Enter`.
4. Press `o` to sort. Set **Key** to `/replicas`; move to **Direction** with `Tab` and choose `descending` with an arrow key. Press `Enter`.

The result is **one array**:

```json
[
  {"name":"worker","replicas":8},
  {"name":"api","replicas":3}
]
```

The generated jq is equivalent to:

```jq
[.services[] | select(.language == "python") | {name, replicas}]
| sort_by(.replicas) | reverse
```

Sorting a sequence of results adds a visible **Collect** step, so the sort applies to the entire sequence. Before collection, Qubex labels separate outputs **RESULT 1**, **RESULT 2**, and so on; those results are not an array unless the query creates one.

Press `c` while the data tree is focused to request a copy of the jq. Press `s`, enter a label such as `Python services by replicas`, and press `Enter` to save the editable query.

## Starter example: change a field while keeping its object

```sh
cat > /tmp/qbx-update.json <<'JSON'
{"name":" api ","replicas":3}
JSON
./bin/qbx /tmp/qbx-update.json
```

Select `name`, press `?`, and choose **Update field in object**. In the form, change **Transform** to `trim` and press `Enter`. The result is `{"name":"api","replicas":3}`: the object remains intact.

If you instead choose **Extract and transform** on `name` in a fresh session, the result is just `"api"`. Use **Update field in object** when later pipeline steps still need sibling fields.

## Filtering nested arrays

To keep a parent object based on elements in a nested array, use an explicit **Any** or **All** choice. Try this input:

```sh
cat > /tmp/qbx-projects.json <<'JSON'
{"projects":[{"name":"api","tags":["prod","urgent"]},{"name":"demo","tags":["dev"]}]}
JSON
./bin/qbx /tmp/qbx-projects.json
```

Select `projects` and choose **Iterate elements**. Press `v` to inspect the project objects in Stage, then press `f` on one of them. Set **Path** to `/tags/0`, **Operator** to `eq`, **Value type** to `string`, and **Value** to `prod`. Choose `any` for **Array .tags** and apply. The result sequence contains only the `api` project. The `0` identifies an element while setting up the path; **Any** tests the tag elements and succeeds when at least one matches. **All** succeeds only when every tag matches. An empty array fails **Any** and passes **All**.

If you press `f` while browsing an array in Source before iterating it, Qubex may ask whether to filter the current value or the array's elements. Choose the collection whose elements you intend to keep. After a projection or sort changes the shape, switch to Stage (`v`) to target fields in the current result.

Filters follow jq behavior for missing values and incompatible types. The filter form's **Guard** field can add an explicit existence or type check. Suggested fields are inferred from a sample of the input, so inspect the preview when records have different shapes.

## Paths, search, and editing

Paths entered in forms use JSON Pointer syntax: `/metadata/name` selects a nested field; an empty path means the current value. Within a field name, write `~0` for `~` and `~1` for `/`. Qubex resolves numeric segments as array indexes when the known shape is an array.

Press `/` while browsing a tree to search. The search form offers fuzzy or exact matching, key/value targets, a document or subtree scope, and case options. Use `n` and `N` to move among matches. Search changes the cursor, not the query.

Press `Ctrl+P` to edit pipeline steps. Use Up/Down to select a step, `Enter` to edit, `x` to delete, `K`/`J` to move it, or `i` to inspect the result after that step. Press `Esc` to leave the pipeline or intermediate-stage inspection. If an edit invalidates a later step, Qubex keeps the step for repair and marks the displayed last successful output as belonging to an earlier query revision.

Other actions in `?` include mapping array elements, selecting multiple fields, unique values, counting current results, measuring the length of each value, and collecting a result sequence into an array. **Count current results** and **Length of value** answer different questions.

## Reuse saved queries

Saved queries contain the editable operation model and generated jq. They do not contain the input document. IDs shown by `history` are used in these commands:

```sh
./bin/qbx history
./bin/qbx history search services
./bin/qbx history show 1
./bin/qbx history open 1 examples/services.json
./bin/qbx history run 1 examples/services.json
./bin/qbx history run "Python services by replicas" examples/services.json
./bin/qbx history export 1 --format jq
./bin/qbx history export 1 --format json
./bin/qbx history copy 1
./bin/qbx history delete 1
```

`history open` restores the operations into the TUI on the input you supply now. `history run` executes without the TUI and prints each complete JSON result with indentation and jq-style terminal colors. Piped output contains plain JSON without color codes. `run` accepts either an ID or an exact saved name; use an ID if multiple queries share a name. Single and double shell quotes both work for names with spaces. Both commands accept piped JSON if you omit the file:

```sh
cat examples/services.json | ./bin/qbx history run 1
```

For a machine-readable history list, add `--json` to `history` or `history search`. Export also supports `--format text`. History is stored under `$XDG_CONFIG_HOME/qubex`, or `~/.config/qubex` when that variable is unset. Query literals, such as filter values, are part of a saved definition.

## Previews and input limits

The default preview has a two-second timeout, a 10,000-result display limit, and an 8 MiB display budget. A **truncated** or **timed out** preview is marked as such; it does not mean the saved query will truncate headless output. Use **Preview limits** in `?` or start Qubex with larger limits:

```sh
./bin/qbx --preview-timeout 10s --preview-results 20000 --preview-bytes 16MiB examples/services.json
```

The default input limit is 500 MiB. Change it with `--input-limit 1GiB` if needed. Sorting and counting use the full relevant input, even when only some results can be displayed in the preview. Use `./bin/qbx --help` for all command-line options.
