# DiskScope

`diskscope` is a read-only terminal disk-usage explorer for large filesystems. It
scans a directory tree, aggregates allocated sizes (or apparent sizes), groups
files into storage categories, and lets you drill into the directories and files
responsible for the usage. It is built with Go and Bubble Tea and runs entirely
in the terminal.

The interface adapts to the terminal: a details pane appears on wide windows,
columns collapse on narrow ones, and a resize hint is shown when there is not
enough room to render safely.

## Features

- Concurrent, cancellable filesystem scan with live progress, throughput, and
  unreadable-path reporting.
- Accurate allocated sizes from `st_blocks` (including sparse files) or apparent
  sizes with `--apparent-size`.
- Hard-link de-duplication by device/inode, symlinks never followed, and
  filesystem-boundary protection.
- Overview with capacity gauge, category distribution, largest files and
  directories, and actionable insights.
- Browser with sortable directory/file listings, size bars, category labels,
  drill-down, parent navigation, and per-directory filtering.
- Categories view grouping storage into video, images, audio, archives,
  documents, source code, databases, data & models, binaries, caches/temp, logs,
  fonts, VM/disk images, containers, system, and other.
- Bounded memory model: all files above `--min-file-size`, the largest
  `--top-per-dir` small files per directory, and global/category top-N lists;
  remaining small files are counted and rolled up.
- Read-only by design: no delete, move, or write operations.
- Adaptive, high-contrast color palette with `--no-color` and full ASCII
  fallbacks.

## Requirements

- Go 1.26 or newer.
- Linux or macOS. Windows compiles with apparent-size scanning and native volume
  capacity support.
- A terminal with at least 40x10 cells.

## Build

```sh
go build -trimpath -o bin/diskscope .
```

Or run directly:

```sh
go run . ~/Downloads
```

## Install

Install the binary into `~/.local/bin` (override with `PREFIX=/usr/local`):

```sh
make install
```

Make sure `~/.local/bin` is on your `PATH`, then run `diskscope`. For a short
command that scans the current directory:

```sh
alias ds='diskscope .'
```

Add that line to your shell's startup file if you want it permanently.

## Usage

```text
diskscope [path] [flags]

  --apparent-size            use apparent size (st_size) instead of allocated blocks
  --one-file-system          do not cross filesystem boundaries (default true)
  --exclude GLOB             glob or absolute path to exclude (repeatable)
  --top-files N              largest files retained globally and per category (default 500)
  --top-per-dir N            small files retained per directory (default 20)
  --min-file-size SIZE       retain every file at least this large (default 1MiB)
  --workers N                scan worker goroutines (default: NumCPU)
  --no-color                 disable colors
  --ascii                    use ASCII-only glyphs
  --version                  print version and exit
```

The path defaults to `$HOME`. Sizes accept suffixes such as `512K`, `1MiB`,
`2GB`, and `1T`.

Examples:

```sh
diskscope /                       # whole root filesystem
diskscope /var --apparent-size
diskscope ~ --exclude '*.iso' --exclude '~/VirtualBox VMs'
diskscope /data --top-files 2000 --min-file-size 64KiB
```

## Keymap

| Key | Action |
| --- | --- |
| `j`/`k`, arrows | Move selection or scroll |
| `pgup`/`pgdn` | Page up / down |
| `g`/`G` | First / last item |
| `enter`/`l` | Open directory, category, or category file list |
| `backspace`/`h` | Parent directory or previous view |
| `tab` / `shift+tab` | Cycle views |
| `1`/`2`/`3` | Jump to Overview / Browser / Categories |
| `s` | Cycle sort: size, name, files, modified |
| `S` | Reverse sort direction |
| `/` | Filter entries in the current browser directory |
| `esc` | Clear filter or leave current view |
| `c` | Copy selected path (clipboard helper or OSC 52) |
| `o` | Open selected location with the OS file manager |
| `r` | Rescan with the same options |
| `?` | Help |
| `q` | Quit; during a scan, cancels it |
| `ctrl+c` | Quit immediately |

`q` during a scan cancels the scan and shows partial results. Press `q` again to
quit.

## Categories

Categories are path- and extension-based so results are reproducible:

- Caches/temp: `.cache`, `node_modules`, package manager caches, `tmp`, trash.
- Git packs and loose object files are grouped as Archives.
- Containers and VM/disk images are detected before generic system paths.
- Logs are detected from `log`/`logs`/`journal` path segments or `.log`.
- OS trees (`/usr`, `/etc`, `/var/lib`, `/boot`, `/System`, ...) are grouped as
  System.
- Everything else falls back to an extension map, then Other.

Color is never the only signal: every category also has a text label.

## Architecture

```text
main.go                     CLI flags and program startup
internal/scan               concurrent scanner, statfs/block helpers, retention heaps
internal/model              aggregated directory tree
internal/categories         deterministic storage classification
internal/ui                 Bubble Tea model, responsive rendering, palette
internal/clipboard          clipboard helpers with OSC 52 fallback
```

Scanning proceeds breadth-first in parallel. Each worker reads one directory,
records direct file totals, and queues child directories. Directory totals are
merged bottom-up after traversal. This avoids deep recursion and keeps
aggregation deterministic.

Memory is bounded by retained entries rather than total files. A scan of a
multi-million-file tree retains the directory tree, every file at least
`--min-file-size`, a bounded number of small files per directory, and
bookkeeping for multiply-linked inodes. The rest are represented by a single
"smaller files" rollup row per directory and still appear in category totals.

With `--one-file-system` (default), the scanner skips directories whose device
ID differs from the scan root, so mounted network shares and external volumes are
not traversed accidentally. Set `--one-file-system=false` to include them. The
scanner also skips `/proc`, `/sys`, `/dev`, and `/run` by default to avoid
virtual and device trees.

## Validation

```sh
gofmt -l .
go vet ./...
go test -race ./...
go build ./...
```

Tests cover classification, aggregation, hard-link de-duplication, exclusions,
permission errors, cancellation, bounded retention, size formatting, responsive
layout, keyboard navigation, and rendering across terminal sizes.

## Notes and limitations

- Windows allocated-size reporting falls back to apparent size; volume capacity
  uses `GetDiskFreeSpaceEx`.
- Directory inode/metadata overhead is not counted; sizes represent file
  content (allocated blocks or apparent bytes).
- Hard links are counted once per scan, not once per path.
- The scanner is read-only and does not follow symlinks.
