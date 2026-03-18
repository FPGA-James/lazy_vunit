# lazy_vunit TUI — Design Spec

**Date:** 2026-03-18
**Status:** Approved

## Overview

`lazy_vunit` is a terminal UI for running VUnit HDL tests, inspired by LazyGit. It auto-discovers VUnit test benches within a git repository, presents them in a navigable tree, and lets the user run tests interactively with real-time output streaming.

---

## Technology Stack

- **Language:** Go
- **TUI framework:** [Bubbletea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss) + [Bubbles](https://github.com/charmbracelet/bubbles)
- **VUnit integration:** subprocess (`python run.py ...`) — no direct Python API dependency

---

## Layout

```
┌─────────────────────────────────────────────────────────────────┐
│ lazy_vunit — /path/to/project  [git: main]  [window: alu]       │
├──────────────────────┬──────────────────────────────────────────┤
│ TESTS   ctrl+r scan  │ OUTPUT  lib.tb_alu.test_overflow          │
│                      │                                          │
│ ▼ src/alu/           │ # Running: python run.py ...             │
│   ▼ tb_alu           │ Compile tb_alu.vhd          passed       │
│     ✓ test_add       │ FAILURE test_overflow after 0 ps         │
│     ✓ test_subtract  │   Expected: overflow = '1'               │
│     ✗ test_overflow  │   Got:      overflow = '0'               │
│   ▶ tb_alu_edge      │                                          │
│ ▶ src/uart/          │                                          │
│ ▶ src/fifo/          │                                          │
│                      │                                          │
├──────────────────────┴──────────────────────────────────────────┤
│ [alu] ✓ 3  ✗ 1  ○ 3  │  all: ✓ 14  ✗ 2  ○ 8   space g [  ]  ? │
└─────────────────────────────────────────────────────────────────┘
```

**Three regions:**
- **Left pane** — directory/testbench/test tree with inline ✓/✗/○ status icons
- **Right pane** — terminal output (full height), streaming stdout of the current/last run
- **Bottom bar** — left: current window pass/fail totals; centre: all-windows aggregate totals; right: key hint strip

---

## Test Discovery

### Finding `run.py`

1. Walk up from `cwd` to the git root (fallback: use `cwd` if no `.git` found)
2. Recursively search for Python files containing `VUnit.from_argv`
3. If **one** found — load it immediately
4. If **multiple** found — show a startup picker (navigable list); user selects the first window to focus

### Building the tree

1. Run `python run.py --export-json /tmp/lazyvunit_<pid>.json --exit-0`
2. Parse the JSON — each test entry contains `name` (`lib.tb_foo.test_bar`) and `location.file_name`
3. Group tests by the **filesystem directory** of their source file
4. Tree hierarchy: **directory → testbench → test case**
5. Merge with cached results from `.lazyvunit/results_<folder>.json`

---

## Windows

Each discovered `run.py` becomes a **named window**, named after its parent directory (e.g. a `run.py` at `alu/run.py` → window name `alu`).

- `[` / `]` — cycle between windows
- Each window has its own independent tree, terminal output pane, and cached results
- The startup picker is shown when multiple `run.py` files are found; `q` at the picker exits cleanly

---

## Keybindings

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate tree |
| `←` / `→` | Collapse / expand node |
| `space` | Run selected node and all tests beneath it |
| `g` | Run selected **single test** in GUI mode (`--gui`), leaf nodes only |
| `[` / `]` | Switch to previous / next window |
| `ctrl+r` | Rescan project for tests |
| `ctrl+c` / `x` | Cancel running test |
| `q` | Quit |
| `?` | Help overlay (all keybindings) |

### Run patterns by node type

| Selected node | VUnit pattern passed to `run.py` |
|---------------|----------------------------------|
| Directory | All tests whose source file is in that directory |
| Testbench | `lib.tb_name.*` |
| Single test | `lib.tb_name.test_case` |

---

## Status Icons

| Icon | Meaning |
|------|---------|
| `✓` (green) | Last run passed |
| `✗` (red) | Last run failed |
| `○` (yellow) | Not yet run |
| `~` (blue) | Currently running |

Icons are shown inline in the tree at the test-case level. Testbench and directory nodes show a derived status: ✗ if any child failed, ✓ if all children passed, ○ otherwise.

---

## Architecture

Four components wired together via Bubbletea's `Model`/`Cmd` message-passing pattern:

### Scanner
- Locates `run.py` files and shells out to `python run.py --export-json ...`
- Parses JSON output, builds the tree model
- Triggered on startup and on `ctrl+r`

### Tree Model
- Holds directory/testbench/test hierarchy in memory
- Tracks: cursor position, collapse/expand state per node, pass/fail/not-run/running status per test
- Derives parent node status from children

### Runner
- Spawns `python run.py <pattern> [--gui] [--no-color]` as a subprocess
- Streams stdout line-by-line via a goroutine into a channel → Bubbletea `Cmd`
- Parses output lines for pass/fail markers to update tree status in real time
- Cancellable: sends `SIGTERM` to subprocess on `ctrl+c` / `x`
- On cancel: restores mid-run tests to `○` (not failed)

### Persistence
- Reads `.lazyvunit/results_<window>.json` on startup after discovery
- Writes after each completed (or cancelled) run
- On first run, appends `.lazyvunit/` to `.gitignore` automatically

---

## Persistence Format

```
.lazyvunit/
  results_alu.json
  results_uart.json
```

Each file stores a flat map of test name → last result and timestamp:

```json
{
  "lib.tb_alu.test_add": { "status": "pass", "ran_at": "2026-03-18T14:23:01Z" },
  "lib.tb_alu.test_overflow": { "status": "fail", "ran_at": "2026-03-18T14:23:05Z" }
}
```

---

## Error States

| Situation | Behaviour |
|-----------|-----------|
| No `run.py` found | Error screen: "No VUnit run script found" with the path searched |
| `run.py` exits non-zero before JSON | Show stderr in terminal pane + hint to install vunit-hdl |
| Scan fails | Tree shows error state; `ctrl+r` to retry |
| No git root | Fall back to `cwd` as project root |
| `q` pressed at startup picker | Exit cleanly |

---

## Out of Scope (v1)

- File watching / auto-rescan on HDL source changes
- Parallel test execution across multiple windows
- Simulator selection UI (users pass simulator args via run.py directly)
- Test filtering / fuzzy search (can be added later)
