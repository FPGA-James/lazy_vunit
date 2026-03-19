# Output Path Setting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an `--output-path` setting row to the settings panel that lets users type a git-root-relative path inline, persisted to the existing settings JSON file.

**Architecture:** Four-layer chain matching the existing settings pattern. Persistence first (add `OutputPath string` to `Settings`), then `WindowModel` wiring (`SetOutputPath`, `TotalSettingRows`, `StartRun` flag), then `AppModel` state (`editingPath`/`pathBuf` fields and key routing), then rendering (output-path row and header indicator). The output-path row is special-cased in the panel — it is not a boolean toggle and does not go through `settingItems`.

**Tech Stack:** Go, `encoding/json`, `charmbracelet/bubbletea`, `charmbracelet/lipgloss`. No new dependencies.

---

## File Map

```
internal/persist/persist.go        — add OutputPath string to Settings struct
internal/persist/persist_test.go   — add round-trip test for OutputPath
internal/ui/window.go              — add SetOutputPath, TotalSettingRows; update StartRun
internal/ui/window_test.go         — add SetOutputPath and TotalSettingRows tests
internal/ui/app.go                 — add editingPath/pathBuf fields and accessors;
                                     add strings import; editingPath key handler;
                                     extend cursor clamp; start edit on space at row 6
internal/ui/app_test.go            — add 6 new tests for edit mode behaviour
internal/ui/layout.go              — render output-path row; conditional hint; header indicator
```

---

## Task 1: Add OutputPath to persist.Settings

**Files:**
- Modify: `internal/persist/persist.go`
- Modify: `internal/persist/persist_test.go`

- [ ] **Step 1: Write the failing test**

Add at the bottom of `internal/persist/persist_test.go`:

```go
func TestSettings_OutputPath_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	in := persist.Settings{OutputPath: "sim/vunit_out"}
	require.NoError(t, persist.SaveSettings(dir, "src_alu", in))
	out, err := persist.LoadSettings(dir, "src_alu")
	require.NoError(t, err)
	assert.Equal(t, "sim/vunit_out", out.OutputPath)
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
export PATH="/opt/homebrew/bin:$PATH"
go test ./internal/persist/ -v -run TestSettings_OutputPath_RoundTrip
```

Expected: FAIL — `out.OutputPath` is `""` because the field does not exist yet.

- [ ] **Step 3: Add OutputPath field to Settings in persist.go**

In `internal/persist/persist.go`, add `OutputPath string` as the last field of `Settings`:

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

No other changes needed — `LoadSettings`/`SaveSettings` handle the new field automatically via JSON.

- [ ] **Step 4: Run all persist tests**

```bash
export PATH="/opt/homebrew/bin:$PATH"
go test ./internal/persist/ -v
```

Expected: all PASS (11 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/persist/persist.go internal/persist/persist_test.go
git commit -m "feat: add OutputPath string to persist.Settings"
```

---

## Task 2: SetOutputPath, TotalSettingRows, and StartRun flag

**Files:**
- Modify: `internal/ui/window.go`
- Modify: `internal/ui/window_test.go`

- [ ] **Step 1: Write the failing tests**

Add at the bottom of `internal/ui/window_test.go`:

```go
func TestWindowModel_SetOutputPath(t *testing.T) {
	w := makeWindow()
	w.SetOutputPath("sim_out")
	assert.Equal(t, "sim_out", w.Settings.OutputPath)
}

func TestWindowModel_TotalSettingRows(t *testing.T) {
	assert.Equal(t, 7, ui.TotalSettingRows())
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
export PATH="/opt/homebrew/bin:$PATH"
go test ./internal/ui/ -v -run TestWindowModel_SetOutputPath -run TestWindowModel_TotalSettingRows
```

Expected: compilation errors — `SetOutputPath` and `TotalSettingRows` not defined.

- [ ] **Step 3: Add TotalSettingRows and SetOutputPath to window.go**

In `internal/ui/window.go`, add after `SettingCount()`:

```go
// TotalSettingRows returns the total number of rows in the settings panel
// (6 boolean toggles + 1 output-path text row).
func TotalSettingRows() int { return 7 }

// SetOutputPath sets the output path and persists the change.
func (w *WindowModel) SetOutputPath(path string) {
	w.Settings.OutputPath = path
	_ = persist.SaveSettings(w.LazyDir, w.Script.WindowKey, w.Settings)
}
```

- [ ] **Step 4: Update StartRun to apply --output-path**

In `internal/ui/window.go`, in the **compile/elaborate-only branch** (around line 147), add after the `if w.Settings.CompileOnly { ... } else { ... }` block and before `w.Output = ...`:

```go
		if w.Settings.OutputPath != "" {
			args = append(args, "--output-path", filepath.Join(w.GitRoot, w.Settings.OutputPath))
		}
```

In the **normal run branch** (around line 183), add after the `if w.Settings.XUnitXML { ... }` block and before `for _, name := range fullNamesFromNode(node) {`:

```go
	if w.Settings.OutputPath != "" {
		args = append(args, "--output-path", filepath.Join(w.GitRoot, w.Settings.OutputPath))
	}
```

`filepath` is already imported in `window.go`. No import changes needed.

- [ ] **Step 5: Run all ui tests**

```bash
export PATH="/opt/homebrew/bin:$PATH"
go test ./internal/ui/ -v
```

Expected: all PASS (including the 2 new window tests).

- [ ] **Step 6: Commit**

```bash
git add internal/ui/window.go internal/ui/window_test.go
git commit -m "feat: SetOutputPath, TotalSettingRows, --output-path applied in StartRun"
```

---

## Task 3: AppModel editingPath State and Key Handling

**Files:**
- Modify: `internal/ui/app.go`
- Modify: `internal/ui/app_test.go`

- [ ] **Step 1: Write the failing tests**

Add this helper and 6 tests at the bottom of `internal/ui/app_test.go`:

```go
// openSettingsAt opens the settings panel and navigates the cursor to the given row index.
func openSettingsAt(t *testing.T, row int) ui.AppModel {
	t.Helper()
	m := ui.NewAppModel([]finder.RunScript{singleScript()}, "/p", "/p")
	m2, _ := m.Update(ui.ScanDoneMsg{})
	m3, _ := m2.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	cur := m3.(ui.AppModel)
	for i := 0; i < row; i++ {
		m4, _ := cur.Update(tea.KeyMsg{Type: tea.KeyDown})
		cur = m4.(ui.AppModel)
	}
	return cur
}

func TestAppModel_SettingsCursorReachesRow6(t *testing.T) {
	m := openSettingsAt(t, 6)
	assert.Equal(t, 6, m.SettingsCursor())
}

func TestAppModel_OutputPathEditMode(t *testing.T) {
	m := openSettingsAt(t, 6)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	assert.True(t, m2.(ui.AppModel).EditingPath())
}

func TestAppModel_OutputPathTyping(t *testing.T) {
	m := openSettingsAt(t, 6)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}) // enter edit
	m3, _ := m2.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m4, _ := m3.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m5, _ := m4.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	assert.Equal(t, "sim", m5.(ui.AppModel).PathBuf())
}

func TestAppModel_OutputPathBackspace(t *testing.T) {
	m := openSettingsAt(t, 6)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}) // enter edit
	m3, _ := m2.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m4, _ := m3.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyBackspace}) // remove 's'
	assert.Equal(t, "", m4.(ui.AppModel).PathBuf())
	// backspace on empty buffer: no panic, buffer stays empty
	m5, _ := m4.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyBackspace})
	assert.Equal(t, "", m5.(ui.AppModel).PathBuf())
}

func TestAppModel_OutputPathConfirm(t *testing.T) {
	m := openSettingsAt(t, 6)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}) // enter edit
	// type "sim " (trailing space to test trim)
	m3, _ := m2.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m4, _ := m3.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m5, _ := m4.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	m6, _ := m5.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	// confirm with enter
	m7, _ := m6.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, m7.(ui.AppModel).EditingPath())
	assert.Equal(t, "", m7.(ui.AppModel).PathBuf())
	assert.Equal(t, "sim", m7.(ui.AppModel).ActiveSettings().OutputPath) // trailing space trimmed
}

func TestAppModel_OutputPathCancel(t *testing.T) {
	m := openSettingsAt(t, 6)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}) // enter edit
	m3, _ := m2.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	// cancel with esc
	m4, _ := m3.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, m4.(ui.AppModel).EditingPath())
	assert.Equal(t, "", m4.(ui.AppModel).ActiveSettings().OutputPath) // value unchanged
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
export PATH="/opt/homebrew/bin:$PATH"
go test ./internal/ui/ -v -run TestAppModel_OutputPath -run TestAppModel_SettingsCursorReachesRow6
```

Expected: compilation errors — `EditingPath`, `PathBuf` not defined.

- [ ] **Step 3: Add fields and accessors to app.go**

In `internal/ui/app.go`, add `"strings"` to the imports block.

Add `editingPath` and `pathBuf` to `AppModel`:

```go
type AppModel struct {
	state          AppStateKind
	picker         PickerModel
	windows        []WindowModel
	activeIdx      int
	gitRoot        string
	termWidth      int
	termHeight     int
	showHelp       bool
	showSettings   bool
	settingsCursor int
	editingPath    bool
	pathBuf        string
}
```

Add two new accessors after the existing `ActiveSettings()` method:

```go
func (m AppModel) EditingPath() bool { return m.editingPath }
func (m AppModel) PathBuf() string   { return m.pathBuf }
```

- [ ] **Step 4: Add the editingPath key handler in handleKey**

In `handleKey`, insert this block after `keys := DefaultKeys` and before the outer `switch`:

```go
	if m.editingPath {
		switch msg.Type {
		case tea.KeyEnter:
			m.activeWin().SetOutputPath(strings.TrimSpace(m.pathBuf))
			m.editingPath = false
			m.pathBuf = ""
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

All key types hit one of these cases or fall through to the `return m, nil`, so no key reaches the outer switch while editing.

- [ ] **Step 5: Update cursor clamp and Run case in the showSettings block**

In the `case m.showSettings:` block, change the `Down` clamp:

```go
		case key.Matches(msg, keys.Down):
			if m.settingsCursor < TotalSettingRows()-1 {
				m.settingsCursor++
			}
```

Change the `Run` (space) case to detect the output-path row:

```go
		case key.Matches(msg, keys.Run): // space
			if m.settingsCursor == SettingCount() { // output-path row (index 6)
				m.editingPath = true
				m.pathBuf = m.activeWin().Settings.OutputPath
			} else {
				m.activeWin().ToggleSetting(m.settingsCursor)
			}
```

- [ ] **Step 6: Run all ui tests**

```bash
export PATH="/opt/homebrew/bin:$PATH"
go test ./internal/ui/ -v
```

Expected: all PASS (including the 6 new app tests + cursor test).

- [ ] **Step 7: Commit**

```bash
git add internal/ui/app.go internal/ui/app_test.go
git commit -m "feat: editingPath state — inline text edit for output-path setting"
```

---

## Task 4: Rendering — Output-Path Row and Header Indicator

**Files:**
- Modify: `internal/ui/layout.go`

No new tests for this task — rendering is verified by the build + full test suite run.

- [ ] **Step 1: Add the output-path row to RenderSettings in layout.go**

In `RenderSettings`, after the `settingItems` loop (after `sb.WriteString(row + "\n")`), add the output-path row before the hint line:

```go
	// output-path row — index SettingCount() (6), rendered separately (no checkbox)
	{
		valueStr := s.OutputPath
		placeholder := valueStr == ""
		if placeholder {
			valueStr = "vunit_out"
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
		if cursor == SettingCount() {
			row = StyleCursor.Render(row)
		}
		sb.WriteString(row + "\n")
	}
```

- [ ] **Step 2: Make the hint line conditional on EditingPath**

Replace the existing single hint line:

```go
	sb.WriteString("\n" + StyleSubtle.Render("  ↑/↓ navigate  space toggle/edit  s close"))
```

with:

```go
	if m.EditingPath() {
		sb.WriteString("\n" + StyleSubtle.Render("  enter confirm  esc cancel"))
	} else {
		sb.WriteString("\n" + StyleSubtle.Render("  ↑/↓ navigate  space toggle/edit  s close"))
	}
```

- [ ] **Step 3: Add the header indicator in renderHeader**

In `renderHeader`, after the `if s.XUnitXML` block, add:

```go
	if s.OutputPath != "" {
		flags = append(flags, StyleSubtle.Render("out:"+s.OutputPath))
	}
```

- [ ] **Step 4: Build and run all tests**

```bash
export PATH="/opt/homebrew/bin:$PATH"
go build -o ~/.local/bin/lazy_vunit . && go test ./... 2>&1
```

Expected: clean build, all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/layout.go
git commit -m "feat: output-path row rendering and header indicator"
```
