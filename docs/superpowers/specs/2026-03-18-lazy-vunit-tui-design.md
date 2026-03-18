# lazy_vunit TUI — Design Spec

**Date:** 2026-03-18
**Status:** Approved

## Overview

`lazy_vunit` is a terminal UI for running VUnit HDL tests, inspired by LazyGit. It auto-discovers VUnit test benches within a git repository, presents them in a navigable tree, and lets the user run tests interactively with real-time output streaming.

---

## Technology Stack

- **Language:** Go
- **TUI framework:** [Bubbletea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss) + [Bubbles](https://github.com/charmbracelet/bubbles)
- **ANSI handling:** [`charmbracelet/x/ansi`](https://github.com/charmbracelet/x) for stripping/rendering ANSI codes in the output viewport
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
- **Right pane** — terminal output (full height), streaming stdout of the current/last run, rendered with ANSI colour support (no `--no-color` flag passed to VUnit)
- **Bottom bar** — left: current window pass/fail totals; centre: all-windows aggregate totals; right: key hint strip

---

## Test Discovery

### Finding `run.py`

1. Walk up from `cwd` to the git root (fallback: use `cwd` if no `.git` found)
2. Recursively search the entire directory tree rooted at the git root (or `cwd`) for Python files named `run.py`; if none found, broaden to any `.py` file containing both `VUnit.from_argv` and a `if __name__ == "__main__"` guard — this avoids matching test helpers or documentation files
3. If **one** found — show a loading spinner while scanning, then load the tree
4. If **multiple** found — show a startup picker; user selects the first window to focus

### Startup picker UI

Shown only when multiple `run.py` files are discovered:
- Displays a navigable list of window names with their full relative path from git root (e.g. `alu  (src/alu/run.py)`)
- `↑` / `↓` to navigate, `Enter` to select and load, `q` to exit cleanly
- No confirmation step — `Enter` immediately begins the scan for that window

### Loading state

While the initial scan runs (either auto-loaded or after picker selection):
- Tree pane shows a spinner and "Scanning…" message
- All keybindings except `q` are disabled until scan completes

### Building the tree

1. Run `python run.py --export-json <os.TempDir()>/lazyvunit_<pid>.json`
   - No `--exit-0` flag — this is a discovery-only invocation that does not run tests
   - The temp file is left in place until the process exits (overwritten on each `ctrl+r` rescan since the pid is constant for the session)
2. Parse the JSON — each test entry contains `name` (`lib.tb_foo.test_bar`) and `location.file_name`
3. Group tests by the **filesystem directory** of their source file
4. Tree hierarchy: **directory → testbench → test case**
5. Merge with cached results from `.lazyvunit/<window-key>.json`

### Tree scroll behaviour

The viewport follows the cursor with scroll-at-edges: the tree scrolls when the cursor is within 5 lines of the top or bottom of the visible area.

---

## Windows

Each discovered `run.py` becomes a **named window**, named after its parent directory (e.g. `alu/run.py` → window name `alu`).

- `[` / `]` — cycle between windows
- Each window has its own independent tree, terminal output pane, and cached results
- The window key used for persistence is the **relative path from the git root** with slashes replaced by underscores (e.g. `src/alu` → `src_alu`), to avoid collisions between identically-named directories in different parts of the project
- The **display name** shown in the header and bottom bar is the leaf directory name (e.g. `alu`). If two windows share the same leaf name (e.g. `src/alu` and `test/alu`), both display the full relative path instead (e.g. `src/alu` and `test/alu`)

---

## Keybindings

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate tree |
| `←` / `→` | Collapse / expand node |
| `space` | Run selected node and all tests beneath it |
| `g` | Run selected **single test** in GUI mode (`--gui`), leaf nodes only |
| `[` / `]` | Switch to previous / next window |
| `ctrl+r` | Rescan project for tests (blocked while a run is in progress) |
| `ctrl+c` / `x` | Cancel running test |
| `q` | Quit |
| `?` | Help overlay (all keybindings) |

### `g` on non-leaf nodes

Pressing `g` on a directory or testbench node shows a brief status message in the bottom bar: `"GUI mode requires a single test — navigate to a test case"`. No action is taken.

### `ctrl+r` during an active run

Rescan is silently blocked while a run is in progress. The bottom bar shows `"Cannot rescan while tests are running"`. Once the run completes or is cancelled, `ctrl+r` works normally.

### Run patterns by node type

VUnit CLI patterns take the form `lib.tb_name.test_case`. There is no native VUnit glob for "all tests in a directory", so the directory run strategy is to **enumerate all matching test names** from the in-memory tree and pass them as separate positional arguments:

```
python run.py lib.tb_foo.test_a lib.tb_foo.test_b lib.tb_bar.test_c ...
```

On very large directories (>200 tests), fall back to running the tests in batches of 200 and streaming the output sequentially. The output pane title shows the directory node name (not individual test names) during a batch run. Output is appended across batches with a separator line between each.

**Batch exit-code fallback:** the exit-code fallback rule (see Output Parsing) applies only to the test names passed to the current batch invocation — not to all `~` tests globally. The runner tracks which test names belong to the active batch and applies the fallback only to those. Tests in future not-yet-started batches remain `○` and are never affected by an earlier batch's exit code.

| Selected node | Strategy |
|---------------|----------|
| Directory | Enumerate all descendant test names, pass as positional args (batched at 200 if needed) |
| Testbench | `lib.tb_name.*` glob — may pick up tests added after last scan; this is intentional |
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
- Triggered on startup and on `ctrl+r` (when no run is active)

### Tree Model
- Holds directory/testbench/test hierarchy in memory
- Tracks: cursor position, collapse/expand state per node, pass/fail/not-run/running status per test
- Derives parent node status from children

### Runner
- Spawns `python run.py <pattern(s)>` (no `--no-color` — ANSI codes are rendered in the viewport)
- Streams stdout line-by-line via a goroutine into a channel → Bubbletea `Cmd`
- Parses output lines for pass/fail signals to update tree status in real time (see Output Parsing below)
- Cancellable: sends `SIGTERM` to subprocess on `ctrl+c` / `x`
- On cancel: tests that completed before cancellation retain their result; only tests still marked `~` (running) are reset to `○`. Tests in future not-yet-started batches were never marked `~` and remain `○` unchanged.

### Persistence
- Reads `.lazyvunit/<window-key>.json` on startup after discovery
- Writes after each completed (or cancelled) run
- On first run, appends `.lazyvunit/` to `.gitignore`; creates `.gitignore` if it doesn't exist; checks for existing entry before appending to avoid duplicates

---

## Output Parsing

The Runner parses VUnit's stdout to update inline test status. VUnit emits consistent pass/fail markers regardless of simulator:

| Pattern in stdout | Action |
|-------------------|--------|
| `lib.tb_name.test_case` followed by `passed` on the same or next line | Mark that test `✓` |
| `lib.tb_name.test_case` followed by `failed` on the same or next line | Mark that test `✗` |
| A line containing only `=` characters (VUnit separator) after a `FAILURE` block | Mark the test currently in output as `✗` |

If a test name cannot be parsed from the output (e.g. non-standard simulator output), the test's status remains `~` until the process exits. On process exit with code 0 all remaining `~` tests are set to `✓`; on non-zero exit all remaining `~` tests are set to `✗`.

This fallback ensures status is always updated even if line-by-line parsing fails.

---

## Persistence Format

```
.lazyvunit/
  src_alu.json
  src_uart.json
```

Window key is the relative path from git root with `/` replaced by `_` (e.g. `src/alu` → `src_alu`).

Each file stores a flat map of test name → last result and timestamp:

```json
{
  "lib.tb_alu.test_add": { "status": "pass", "ran_at": "2026-03-18T14:23:01Z" },
  "lib.tb_alu.test_overflow": { "status": "fail", "ran_at": "2026-03-18T14:23:05Z" }
}
```

---

## Bottom Bar Aggregation

- **Current window totals** — count of all tests in the active window by status (pass/fail/not-run), regardless of whether they have a persisted result
- **All-windows aggregate** — computed only from windows that have been loaded (discovery run) in the current session. At startup, only the initially selected window is loaded; other windows show `○` for all their tests until they are visited. The aggregate updates as windows are loaded.

---

## Error States

| Situation | Behaviour |
|-----------|-----------|
| No `run.py` found | Error screen: "No VUnit run script found" with the path searched |
| `run.py` exits non-zero before JSON | Show stderr in terminal pane + hint to install vunit-hdl |
| Scan fails | Tree shows error state; `ctrl+r` to retry |
| No git root | Fall back to `cwd` as project root |
| `q` pressed at startup picker | Exit cleanly |
| `g` pressed on non-leaf node | Status bar message: "GUI mode requires a single test" |
| `ctrl+r` during active run | Status bar message: "Cannot rescan while tests are running" |
| Directory run exceeds 200 tests | Split into batches of 200, run sequentially |

---

## Out of Scope (v1)

- File watching / auto-rescan on HDL source changes
- Parallel test execution across multiple windows
- Simulator selection UI (users pass simulator args via run.py directly)
- Test filtering / fuzzy search (can be added later)
- Windows OS support (uses `SIGTERM` and `/`-based paths)
