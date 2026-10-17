---
layout: default
title: Install and open JSON
nav_title: Install and input
nav_order: 1
section: Getting started
read_time: 5 min read
summary: Install Qubex, load one JSON document from a file or pipe, and learn what the CLI expects.
permalink: /01-install-and-input.html
---

# Install and open JSON

This page uses **`examples/services.json`** throughout. Qubex supports macOS and Linux on amd64 and arm64. The binary does not require an installed `jq`.

## Install a release

Once the public release and Homebrew tap are available, install with:

```sh
brew tap Agrim-P777/tap
brew trust --formula Agrim-P777/tap/qbx
brew install Agrim-P777/tap/qbx
qbx --version
```

Or download your platform's archive from the [GitHub Releases page](https://github.com/Agrim-P777/qubex/releases), extract it, and put `qbx` on your `PATH`. The release archives contain the license and third-party notices. Until the first release is published, build from source as below.

## Build from the repository

From the repository root:

```sh
make build
./bin/qbx --version
./bin/qbx examples/services.json
```

`make build` creates `bin/qbx`. The rest of this guide uses that path. If you put the binary on your `PATH`, replace `./bin/qbx` with `qbx`. The source repository’s README contains contributor checks and release-build commands; this site focuses on using the tool.

Qubex opens an interactive terminal screen. The **Source** tree initially shows the entire file, the generated query is `.`, and the output is the original JSON value. Arrow keys move the cursor; `Enter` expands or collapses a node. Press `q` to quit.

## Read from standard input

One complete JSON document can come from a pipe:

```sh
cat examples/services.json | ./bin/qbx
```

The pipe carries JSON only. Qubex reads key presses from the controlling terminal, so the interactive mode still needs a real terminal. If you run from a script without one, use a saved query with `history run` instead.

To process JSON produced by another command, pipe its **single JSON document** into `qbx` in the same way. Multiple adjacent JSON documents and newline-delimited JSON (NDJSON) are not accepted as input in this release. Arrays inside one document are fine: `services` is such an array.

## Input and output boundaries

The default input limit is 500 MiB. Raise it for a larger document:

```sh
./bin/qbx --input-limit 1GiB examples/services.json
```

The file is decoded once in memory. Expansion and navigation are lazy, but a large query operation such as sorting can still need substantial memory. The live preview has separate time and display limits; the complete headless run does not inherit them.

Object keys appear in deterministic sorted order in the tree. Your original JSON file is never modified by browsing, filtering, or saving a query.

## Check the available commands

```sh
./bin/qbx --help
```

`qbx [file]` launches the builder. `qbx history ...` lists, reopens, exports, and runs saved definitions. The [CLI reference](12-cli-reference.md) gives every subcommand, option, and exit code.

> If the screen says it needs a **controlling terminal**, provide a JSON file while running `qbx` directly in your terminal, or use `history run` for noninteractive execution.

Next: [Build your first query](02-first-query.md).
