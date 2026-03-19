# v0.1.0 — Initial Release

First public release of lazy_vunit — a keyboard-driven terminal UI for running [VUnit](https://vunit.github.io/) HDL simulations, inspired by [lazygit](https://github.com/jesseduffield/lazygit).

## Features

- **Test tree** — collapsible hierarchy of VUnit test suites, benchmarks, and individual tests
- **Run tests** — `space` to run the selected test, benchmark, or entire directory; `g` to open the simulator GUI for a single test
- **Live output** — simulator output streams into the output pane as it runs
- **Full-width output** — press `o` to expand the output pane to full width for easy copy/paste
- **Settings panel** — per-script toggles for `--clean`, `--verbose`, `--compile`, `--elaborate`, `--fail-fast`, `--xunit-xml`, and `--output-path`; all persisted to `.lazyvunit/`
- **Multi-window** — open multiple `run.py` scripts and switch between them with `[` / `]`
- **Smart discovery** — finds `run.py` files at or below your current directory; skips `.venv`, `venv`, `__pycache__`, `.tox`, and other environment directories automatically
- **Auto-select** — if only one `run.py` is found, skips the picker and launches directly
- **Persistent results** — pass/fail history remembered between sessions

## Platform

Linux and macOS (amd64, arm64). Windows is not supported.

## Requirements

- Python 3 with [VUnit](https://vunit.github.io/installing.html) installed (`pip install vunit-hdl`)
- A supported HDL simulator (GHDL, ModelSim, Questa, Riviera-PRO, etc.)

## Installation

Download the archive for your platform from the assets below, extract, and place the binary on your `$PATH`:

```bash
tar xz -f lazy_vunit_darwin_arm64.tar.gz
mv lazy_vunit ~/.local/bin/
```

## Notes

This tool was built for a specific personal workflow and released as-is. It has not been tested across a wide range of simulator configurations. Built entirely by [Claude](https://claude.ai) (Anthropic) under strict engineering guidelines.
