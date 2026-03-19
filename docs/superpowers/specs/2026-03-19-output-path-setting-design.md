# Output Path Setting Design

## Goal

Add an `--output-path` setting to the settings panel. Unlike the six boolean toggles, this setting holds a string value (a path relative to the git root). The panel row enters an inline text-edit mode when activated, allowing the user to type a path directly in the panel.

## Behaviour

- **Empty value (default):** `-o` / `--output-path` is omitted from VUnit args. VUnit uses its own default (`vunit_out/` next to the run script).
- **Non-empty value:** `--output-path <gitRoot>/<value>` is appended to VUnit args. The stored value is relative to the git root; `StartRun` resolves it to an absolute path.
- The setting is persisted in `.lazyvunit/<windowKey>_settings.json` alongside the existing boolean flags.

## Panel Layout

The output-path row appears as the 7th (last) row in the settings panel, after the six boolean rows:

```
  clean            wipe output dir before run
  verbose          print all test output
  compile only     compile only, skip simulation
  elaborate only   elaborate only, skip simulation
  fail fast        stop on first failure
  xunit xml        write report to .lazyvunit/<key>_report.xml
  output-path      sim_out                     relative to git root
```

The row has no `[✓]/[ ]` checkbox. Instead it shows the current value (or dim placeholder `vunit_out` when empty).

**Cursor on the row, not editing:**
The entire row is highlighted with `StyleCursor`, same as boolean rows.

**Editing mode (after pressing space):**
The value field shows the current buffer with a block cursor `█` appended:
```
  output-path      sim_out█                    relative to git root
```

**Hint line** changes based on mode:
- Normal:  `↑/↓ navigate  space toggle/edit  s close`
- Editing: `enter confirm  esc cancel`

## Interaction Model

**Opening edit mode:** Press `space` (the existing Run/toggle key) when the cursor is on the output-path row. `pathBuf` is pre-filled with the current `OutputPath` value.

**While editing** (`editingPath == true`):
- All other panel navigation is disabled.
- Character keys (`tea.KeyRunes`) append to `pathBuf`.
- `backspace` removes the last rune from `pathBuf`.
- `enter` saves: trims whitespace, calls `SetOutputPath(pathBuf)`, exits edit mode.
- `esc` cancels: discards `pathBuf`, exits edit mode. The saved value is unchanged.

**Outer keys while editing:** `q` (quit) and `s`/`esc` are consumed by the editing handler before they reach the outer switch, so only `enter` and `esc` exit the mode.

## Header Indicator

When `OutputPath` is non-empty, `renderHeader` appends a dim indicator:

```
StyleSubtle.Render("out:" + s.OutputPath)
```

This appears after the existing flag indicators (clean, verbose, etc.).

## File Map

```
internal/persist/persist.go   — add OutputPath string field to Settings
internal/ui/window.go         — add SetOutputPath(path string), TotalSettingRows() int, update StartRun
internal/ui/app.go            — add editingPath bool + pathBuf string to AppModel;
                                add EditingPath()/PathBuf() accessors;
                                handle editingPath keys before outer switch;
                                extend cursor clamp to TotalSettingRows()-1;
                                start edit on space when cursor == SettingCount()
internal/ui/layout.go         — render output-path row after settingItems loop;
                                conditional hint line; header indicator
```

## Detailed Changes

### persist/persist.go

Add `OutputPath string` to `Settings`:

```go
type Settings struct {
    Clean         bool   `json:"clean"`
    Verbose       bool   `json:"verbose"`
    CompileOnly   bool   `json:"compile_only"`
    ElaborateOnly bool   `json:"elaborate_only"`
    FailFast      bool   `json:"fail_fast"`
    XUnitXML      bool   `json:"xunit_xml"`
    OutputPath    string `json:"output_path"`
}
```

No other changes to persist.go — `LoadSettings`/`SaveSettings` handle the new field automatically via JSON.

### window.go

**New function `TotalSettingRows`** (used by AppModel for cursor clamping):

```go
// TotalSettingRows returns the total number of rows in the settings panel
// (6 boolean toggles + 1 output-path text row).
func TotalSettingRows() int { return 7 }
```

**New method `SetOutputPath`** (mirrors ToggleSetting's persist pattern):

```go
// SetOutputPath sets the output path and persists the change.
func (w *WindowModel) SetOutputPath(path string) {
    w.Settings.OutputPath = path
    _ = persist.SaveSettings(w.LazyDir, w.Script.WindowKey, w.Settings)
}
```

**`StartRun` update** — add `--output-path` in both branches when `OutputPath` is non-empty:

In the compile/elaborate-only branch, after the `--verbose` arg:
```go
if w.Settings.OutputPath != "" {
    args = append(args, "--output-path", filepath.Join(w.GitRoot, w.Settings.OutputPath))
}
```

In the normal run branch, after `--xunit-xml`:
```go
if w.Settings.OutputPath != "" {
    args = append(args, "--output-path", filepath.Join(w.GitRoot, w.Settings.OutputPath))
}
```

### app.go

**New fields on `AppModel`:**

```go
type AppModel struct {
    // ... existing fields ...
    editingPath bool
    pathBuf     string
}
```

**New accessors:**

```go
func (m AppModel) EditingPath() bool { return m.editingPath }
func (m AppModel) PathBuf() string   { return m.pathBuf }
```

**`handleKey` — editingPath handler inserted after the StatePicker check, before the outer switch:**

```go
if m.editingPath {
    switch msg.Type {
    case tea.KeyEnter:
        m.activeWin().SetOutputPath(strings.TrimSpace(m.pathBuf))
        m.editingPath = false
    case tea.KeyEsc:
        m.editingPath = false
        m.pathBuf = ""
    case tea.KeyBackspace, tea.KeyDelete:
        if len(m.pathBuf) > 0 {
            runes := []rune(m.pathBuf)
            m.pathBuf = string(runes[:len(runes)-1])
        }
    case tea.KeyRunes:
        m.pathBuf += string(msg.Runes)
    }
    return m, nil
}
```

**`handleKey` — cursor clamp update in `case m.showSettings:`:**

```go
case key.Matches(msg, keys.Down):
    if m.settingsCursor < TotalSettingRows()-1 {
        m.settingsCursor++
    }
```

**`handleKey` — space on output-path row starts edit mode:**

```go
case key.Matches(msg, keys.Run): // space
    if m.settingsCursor == SettingCount() { // output-path row
        m.editingPath = true
        m.pathBuf = m.activeWin().Settings.OutputPath
    } else {
        m.activeWin().ToggleSetting(m.settingsCursor)
    }
```

### layout.go

**`RenderSettings`** — render the output-path row after the `settingItems` loop:

```go
// output-path row (index == SettingCount())
valueStr := s.OutputPath
placeholder := false
if valueStr == "" {
    valueStr = "vunit_out"
    placeholder = true
}
if m.EditingPath() {
    valueStr = m.PathBuf() + "█"
}
var displayVal string
if placeholder && !m.EditingPath() {
    displayVal = StyleSubtle.Render(valueStr)
} else {
    displayVal = valueStr
}
row := fmt.Sprintf("  %-16s %-22s %s", "output-path", displayVal, StyleSubtle.Render("relative to git root"))
if SettingCount() == cursor {
    row = StyleCursor.Render(row)
}
sb.WriteString(row + "\n")
```

**Hint line** — conditional on `EditingPath()`:

```go
if m.EditingPath() {
    sb.WriteString("\n" + StyleSubtle.Render("  enter confirm  esc cancel"))
} else {
    sb.WriteString("\n" + StyleSubtle.Render("  ↑/↓ navigate  space toggle/edit  s close"))
}
```

**`renderHeader`** — add output-path indicator when non-empty (after existing flag indicators):

```go
if s.OutputPath != "" {
    flags = append(flags, StyleSubtle.Render("out:"+s.OutputPath))
}
```

## Testing

**persist:** One new test — round-trip with non-empty `OutputPath` value.

**window:**
- `TestWindowModel_SetOutputPath` — sets path, verify `Settings.OutputPath` updated.
- `TestWindowModel_TotalSettingRows` — returns 7.

**app:**
- `TestAppModel_OutputPathEditMode` — open settings, navigate to row 6, press space, verify `EditingPath() == true`.
- `TestAppModel_OutputPathTyping` — while editing, send rune keys, verify `PathBuf()` accumulates.
- `TestAppModel_OutputPathConfirm` — while editing, send enter, verify `EditingPath() == false` and `ActiveSettings().OutputPath` updated.
- `TestAppModel_OutputPathCancel` — while editing, send esc, verify `EditingPath() == false` and value unchanged.
- `TestAppModel_SettingsCursorReachesRow6` — pressing down 6 times from row 0 reaches row 6.
