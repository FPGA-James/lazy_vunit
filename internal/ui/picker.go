// internal/ui/picker.go
package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lazyvunit/lazy_vunit/internal/finder"
)

// PickerSelectedMsg is emitted when the user selects a run script.
type PickerSelectedMsg struct {
	Script finder.RunScript
}

type PickerModel struct {
	scripts []finder.RunScript
	cursor  int
}

func NewPickerModel(scripts []finder.RunScript) PickerModel {
	return PickerModel{scripts: scripts}
}

func (m PickerModel) Cursor() int { return m.cursor }

func (m PickerModel) Init() tea.Cmd { return nil }

func (m PickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			if m.cursor < len(m.scripts)-1 {
				m.cursor++
			}
		case tea.KeyEnter:
			selected := m.scripts[m.cursor]
			return m, func() tea.Msg { return PickerSelectedMsg{Script: selected} }
		case tea.KeyRunes:
			if string(msg.Runes) == "q" {
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m PickerModel) View() string {
	var sb strings.Builder
	sb.WriteString(StyleHeader.Render("Select a VUnit project window\n\n"))
	for i, s := range m.scripts {
		display := finder.DisplayName(m.scripts, s)
		line := "  " + display + "  (" + s.RelDir + "/run.py)"
		if i == m.cursor {
			line = StyleCursor.Render("> " + display + "  (" + s.RelDir + "/run.py)")
		}
		sb.WriteString(line + "\n")
	}
	sb.WriteString(StyleSubtle.Render("\n↑↓ navigate   enter select   q quit"))
	return sb.String()
}
