# lazy_vunit

A terminal UI for running [VUnit](https://vunit.github.io/) HDL simulations — inspired by [lazygit](https://github.com/jesseduffield/lazygit).

Navigate your test hierarchy, run individual tests or whole suites, and watch output stream in — all without leaving the terminal.

> **Disclaimer:** This project was autocreated by [Claude](https://claude.ai) (Anthropic) using strict engineering guidelines: test-driven development, spec-reviewed design documents, and per-task code quality review. All implementation decisions were made by Claude; the human guided requirements only.

---

## Screenshot

```
 lazy_vunit — /home/user/project  [alu]  verbose  xunit

 TESTS  ctrl+r scan          OUTPUT  compile
┌────────────────────────┐┌────────────────────────────────────────────────┐
│▼ alu                   ││# Running: python src/alu/run.py --verbose       │
│  ▼ tb_alu              ││vunit_out/...                                    │
│    ✓ test_add          ││test 'lib.tb_alu.test_add' passed                │
│    ✗ test_sub          ││test 'lib.tb_alu.test_sub' failed                │
│    ○ test_mul          ││                                                 │
└────────────────────────┘└────────────────────────────────────────────────┘
 [alu] ✓ 1  ✗ 1  ○ 1  │  all: ✓ 1  ✗ 1  ○ 1  │  space run  g gui  [ ]  s settings  q quit  ? help
```

---

## Requirements

| Dependency | Purpose |
|---|---|
| [Go 1.21+](https://golang.org/dl/) | Build the binary |
| [Python 3](https://www.python.org/) | Required by VUnit |
| [VUnit](https://vunit.github.io/installing.html) | HDL simulation framework (`pip install vunit-hdl`) |
| A simulator | e.g. [GHDL](https://github.com/ghdl/ghdl), ModelSim, Questa, Riviera-PRO |

lazy_vunit does not bundle VUnit or any simulator — it invokes your existing `run.py` scripts.

---

## Installation

```bash
git clone https://github.com/lazyvunit/lazy_vunit.git
cd lazy_vunit
go build -o ~/.local/bin/lazy_vunit .
```

Make sure `~/.local/bin` (or wherever you place the binary) is on your `$PATH`.

---

## How to Use

### Starting

Run `lazy_vunit` from anywhere inside a git repository that contains VUnit `run.py` scripts.

```bash
cd /path/to/your/hdl/project
lazy_vunit
```

**Script discovery:**

- Only `run.py` files at or below your **current working directory** are shown.
- If only one script is found, the tool launches directly into it — no picker needed.
- If multiple scripts are found, a picker lets you choose which one to open.
- Common Python environment directories are automatically excluded from discovery (`.venv`, `venv`, `env`, `.tox`, `__pycache__`, etc.).

### Navigation

| Key | Action |
|---|---|
| `↑` / `↓` | Move cursor up/down through the test tree |
| `→` | Expand a directory or benchmark node |
| `←` | Collapse a directory or benchmark node |
| `[` / `]` | Switch between open script windows |

### Running Tests

| Key | Action |
|---|---|
| `space` | Run the selected test, benchmark, or directory |
| `g` | Run the selected **test** in GUI mode (single test only) |
| `x` / `ctrl+c` | Cancel an in-progress run |
| `ctrl+r` | Re-scan the test hierarchy |

Selecting a **directory** or **benchmark** node runs all tests under it. Results are colour-coded: `✓` passed, `✗` failed, `○` not yet run.

### Settings Panel

Press `s` to open the settings panel. Navigate rows with `↑`/`↓` and toggle with `space`.

| Setting | VUnit flag | Description |
|---|---|---|
| clean | `--clean` | Wipe the output directory before each run |
| verbose | `--verbose` | Print all test output to the output pane |
| compile only | `--compile` | Compile without running simulations |
| elaborate only | `--elaborate` | Elaborate without running simulations |
| fail fast | `--fail-fast` | Stop on the first test failure |
| xunit xml | `--xunit-xml` | Write a JUnit-compatible report to `.lazyvunit/<key>_report.xml` |
| output-path | `--output-path` | Set a custom simulation output directory (relative to git root) |

The `output-path` row is a text field. Navigate to it and press `space` to enter edit mode:

- Type the path (relative to your git root, e.g. `sim/vunit_out`)
- `enter` — confirm and save
- `esc` — cancel without saving

Settings are persisted per script to `.lazyvunit/<window_key>_settings.json` inside your git root.

### Other Keys

| Key | Action |
|---|---|
| `s` | Open / close the settings panel |
| `esc` | Close any open panel |
| `?` | Open / close the keybindings help panel |
| `q` | Quit |

---

## Persistence

lazy_vunit stores state in a `.lazyvunit/` directory at the root of your git repository:

```
.lazyvunit/
  <window_key>_results.json    # last known pass/fail status per test
  <window_key>_settings.json   # per-script settings (flags, output path)
  <window_key>_report.xml      # xunit report (when xunit xml is enabled)
```

This directory is automatically added to `.gitignore` on first run.

---

## How run.py Scripts Are Discovered

1. lazy_vunit walks from your current directory downward looking for files named `run.py`.
2. If none are found by name, it falls back to scanning all `.py` files for `VUnit.from_argv` and a `if __name__ == "__main__"` guard.
3. The following directories are always skipped:

   `.venv` · `venv` · `env` · `.env` · `virtualenv` · `.tox` · `.nox` · `__pycache__` · `.mypy_cache` · `.ruff_cache` · `node_modules` · `.git`

---

## Inspiration

lazy_vunit is directly inspired by [lazygit](https://github.com/jesseduffield/lazygit) — the idea that a well-designed terminal UI can make a complex tool feel effortless. The same philosophy applied to HDL simulation: stay in the terminal, see everything at once, run with a keypress.

---

## Built With

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — terminal UI framework (Elm architecture)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — terminal styling and layout
- [VUnit](https://vunit.github.io/) — HDL simulation and test framework
