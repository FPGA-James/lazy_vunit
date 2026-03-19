// internal/ui/app_test.go
package ui_test

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lazyvunit/lazy_vunit/internal/finder"
	"github.com/lazyvunit/lazy_vunit/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func singleScript() finder.RunScript {
	return finder.RunScript{AbsPath: "/p/src/alu/run.py", RelDir: "src/alu", WindowKey: "src_alu", LeafName: "alu"}
}

func TestAppModel_QuitFromAnyState(t *testing.T) {
	m := ui.NewAppModel([]finder.RunScript{singleScript()}, "/p", "/p")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	require.NotNil(t, cmd)
	msg := cmd()
	_, isQuit := msg.(tea.QuitMsg)
	assert.True(t, isQuit)
}

func TestAppModel_SingleScriptSkipsPicker(t *testing.T) {
	m := ui.NewAppModel([]finder.RunScript{singleScript()}, "/p", "/p")
	assert.Equal(t, ui.StateScanning, m.AppState())
}

func TestAppModel_MultipleScriptsShowsPicker(t *testing.T) {
	scripts := []finder.RunScript{
		singleScript(),
		{AbsPath: "/p/src/uart/run.py", RelDir: "src/uart", WindowKey: "src_uart", LeafName: "uart"},
	}
	m := ui.NewAppModel(scripts, "/p", "/p")
	assert.Equal(t, ui.StatePicker, m.AppState())
}

func TestAppModel_ScanErrorSetsErrorState(t *testing.T) {
	m := ui.NewAppModel([]finder.RunScript{singleScript()}, "/p", "/p")
	m2, _ := m.Update(ui.ScanDoneMsg{Err: fmt.Errorf("no python")})
	assert.Equal(t, ui.StateError, m2.(ui.AppModel).AppState())
}

func TestAppModel_ScanSuccessSetsMainState(t *testing.T) {
	m := ui.NewAppModel([]finder.RunScript{singleScript()}, "/p", "/p")
	m2, _ := m.Update(ui.ScanDoneMsg{Entries: nil, Err: nil})
	assert.Equal(t, ui.StateMain, m2.(ui.AppModel).AppState())
}

func TestAppModel_SwitchWindowsWithBrackets(t *testing.T) {
	scripts := []finder.RunScript{
		singleScript(),
		{AbsPath: "/p/src/uart/run.py", RelDir: "src/uart", WindowKey: "src_uart", LeafName: "uart"},
	}
	m := ui.NewAppModel(scripts, "/p", "/p")
	// Select first window from picker
	m2, _ := m.Update(ui.PickerSelectedMsg{Script: scripts[0]})
	// Simulate scan done for window 0
	m3, _ := m2.(ui.AppModel).Update(ui.ScanDoneMsg{Entries: nil, Err: nil})
	// Press ] to go to next window
	m4, _ := m3.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	assert.Equal(t, 1, m4.(ui.AppModel).ActiveWindowIndex())
}

func TestAppModel_SettingsOpenWithS(t *testing.T) {
	m := ui.NewAppModel([]finder.RunScript{singleScript()}, "/p", "/p")
	m2, _ := m.Update(ui.ScanDoneMsg{})
	m3, _ := m2.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	assert.True(t, m3.(ui.AppModel).ShowSettings())
}

func TestAppModel_SettingsCloseWithS(t *testing.T) {
	m := ui.NewAppModel([]finder.RunScript{singleScript()}, "/p", "/p")
	m2, _ := m.Update(ui.ScanDoneMsg{})
	m3, _ := m2.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m4, _ := m3.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	assert.False(t, m4.(ui.AppModel).ShowSettings())
}

func TestAppModel_SettingsCursorNavigates(t *testing.T) {
	m := ui.NewAppModel([]finder.RunScript{singleScript()}, "/p", "/p")
	m2, _ := m.Update(ui.ScanDoneMsg{})
	m3, _ := m2.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m4, _ := m3.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m4.(ui.AppModel).SettingsCursor())
}

func TestAppModel_SettingsToggleViaSpack(t *testing.T) {
	m := ui.NewAppModel([]finder.RunScript{singleScript()}, "/p", "/p")
	m2, _ := m.Update(ui.ScanDoneMsg{})
	// open settings
	m3, _ := m2.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	// space toggles row 0 (Clean)
	m4, _ := m3.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	assert.True(t, m4.(ui.AppModel).ActiveSettings().Clean)
}

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

func TestAppModel_OutputPathSKeyDuringEditAppends(t *testing.T) {
	m := openSettingsAt(t, 6)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}) // enter edit
	m3, _ := m2.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	assert.True(t, m3.(ui.AppModel).EditingPath())  // still editing
	assert.True(t, m3.(ui.AppModel).ShowSettings()) // panel still open
	assert.Equal(t, "s", m3.(ui.AppModel).PathBuf()) // 's' appended
}
