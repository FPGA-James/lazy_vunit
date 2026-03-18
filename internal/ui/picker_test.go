// internal/ui/picker_test.go
package ui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lazyvunit/lazy_vunit/internal/finder"
	"github.com/lazyvunit/lazy_vunit/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeScripts() []finder.RunScript {
	return []finder.RunScript{
		{AbsPath: "/p/src/alu/run.py", RelDir: "src/alu", WindowKey: "src_alu", LeafName: "alu"},
		{AbsPath: "/p/src/uart/run.py", RelDir: "src/uart", WindowKey: "src_uart", LeafName: "uart"},
	}
}

func TestPickerModel_InitialCursor(t *testing.T) {
	m := ui.NewPickerModel(makeScripts())
	assert.Equal(t, 0, m.Cursor())
}

func TestPickerModel_MoveDown(t *testing.T) {
	m := ui.NewPickerModel(makeScripts())
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	pm := newM.(ui.PickerModel)
	assert.Equal(t, 1, pm.Cursor())
}

func TestPickerModel_MoveDownClamps(t *testing.T) {
	m := ui.NewPickerModel(makeScripts())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m3, _ := m2.(ui.PickerModel).Update(tea.KeyMsg{Type: tea.KeyDown}) // past end
	assert.Equal(t, 1, m3.(ui.PickerModel).Cursor())
}

func TestPickerModel_EnterSelectsScript(t *testing.T) {
	m := ui.NewPickerModel(makeScripts())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	newM, cmd := m2.(ui.PickerModel).Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = newM
	require.NotNil(t, cmd)
	msg := cmd()
	selected, ok := msg.(ui.PickerSelectedMsg)
	require.True(t, ok)
	assert.Equal(t, "src/uart", selected.Script.RelDir)
}

func TestPickerModel_QuitEmitsQuit(t *testing.T) {
	m := ui.NewPickerModel(makeScripts())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	require.NotNil(t, cmd)
	msg := cmd()
	_, isQuit := msg.(tea.QuitMsg)
	assert.True(t, isQuit)
}

func TestPickerModel_ViewContainsWindowNames(t *testing.T) {
	m := ui.NewPickerModel(makeScripts())
	view := m.View()
	assert.Contains(t, view, "alu")
	assert.Contains(t, view, "uart")
}
