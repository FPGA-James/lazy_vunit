// internal/ui/app.go
package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lazyvunit/lazy_vunit/internal/finder"
	"github.com/lazyvunit/lazy_vunit/internal/persist"
	"github.com/lazyvunit/lazy_vunit/internal/runner"
)

type AppStateKind int

const (
	StatePicker   AppStateKind = iota
	StateScanning
	StateMain
	StateError
)

type AppModel struct {
	state      AppStateKind
	picker     PickerModel
	windows    []WindowModel
	activeIdx  int
	gitRoot    string
	termWidth  int
	termHeight int
	showHelp      bool
	showSettings  bool
	settingsCursor int
}

func NewAppModel(scripts []finder.RunScript, gitRoot, cwd string) AppModel {
	m := AppModel{gitRoot: gitRoot}
	if len(scripts) == 1 {
		win := NewWindowModel(scripts[0], scripts, gitRoot)
		m.windows = []WindowModel{win}
		m.state = StateScanning
	} else {
		m.picker = NewPickerModel(scripts)
		m.state = StatePicker
		for _, s := range scripts {
			m.windows = append(m.windows, NewWindowModel(s, scripts, gitRoot))
		}
	}
	return m
}

func (m AppModel) AppState() AppStateKind   { return m.state }
func (m AppModel) ActiveWindowIndex() int   { return m.activeIdx }
func (m *AppModel) activeWin() *WindowModel { return &m.windows[m.activeIdx] }

func (m AppModel) ShowSettings() bool  { return m.showSettings }
func (m AppModel) SettingsCursor() int { return m.settingsCursor }
func (m AppModel) ActiveSettings() persist.Settings {
	if len(m.windows) == 0 {
		return persist.Settings{}
	}
	return m.activeWin().Settings
}

func (m AppModel) Init() tea.Cmd {
	if m.state == StateScanning && len(m.windows) > 0 {
		return m.windows[0].ScanCmd()
	}
	return nil
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Clear transient status message each update cycle so it shows for exactly one frame.
	if len(m.windows) > 0 {
		m.activeWin().StatusMsg = ""
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth, m.termHeight = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case PickerSelectedMsg:
		for i, w := range m.windows {
			if w.Script.AbsPath == msg.Script.AbsPath {
				m.activeIdx = i
				break
			}
		}
		m.state = StateScanning
		return m, m.activeWin().ScanCmd()

	case ScanDoneMsg:
		m.activeWin().ApplyScanResult(msg.Entries, msg.Err)
		if msg.Err != nil {
			m.state = StateError
		} else {
			m.state = StateMain
		}
		return m, nil

	case runner.OutputLineMsg, runner.StatusUpdateMsg, runner.RunDoneMsg:
		if len(m.windows) == 0 {
			return m, nil
		}
		cmd := m.activeWin().HandleRunnerMsg(msg)
		return m, cmd
	}

	return m, nil
}

func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.state == StatePicker {
		newPicker, cmd := m.picker.Update(msg)
		m.picker = newPicker.(PickerModel)
		return m, cmd
	}

	keys := DefaultKeys

	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Help):
		m.showHelp = !m.showHelp

	case key.Matches(msg, keys.Settings):
		if m.state == StateMain {
			m.showSettings = !m.showSettings
			m.settingsCursor = 0
		}

	case key.Matches(msg, keys.Escape):
		m.showSettings = false
		m.showHelp = false

	case m.showSettings:
		switch {
		case key.Matches(msg, keys.Up):
			if m.settingsCursor > 0 {
				m.settingsCursor--
			}
		case key.Matches(msg, keys.Down):
			if m.settingsCursor < SettingCount()-1 {
				m.settingsCursor++
			}
		case key.Matches(msg, keys.Run): // space = Run key
			m.activeWin().ToggleSetting(m.settingsCursor)
		case key.Matches(msg, keys.Escape), key.Matches(msg, keys.Settings):
			m.showSettings = false
		}
		return m, nil

	case m.state != StateMain:
		return m, nil

	case key.Matches(msg, keys.Up):
		if m.activeWin().Tree != nil {
			m.activeWin().Tree.MoveUp()
		}

	case key.Matches(msg, keys.Down):
		if m.activeWin().Tree != nil {
			m.activeWin().Tree.MoveDown()
		}

	case key.Matches(msg, keys.Left):
		if t := m.activeWin().Tree; t != nil {
			node := t.CursorNode()
			if node != nil && node.Expanded {
				t.Toggle()
			}
		}

	case key.Matches(msg, keys.Right):
		if t := m.activeWin().Tree; t != nil {
			node := t.CursorNode()
			if node != nil && !node.Expanded {
				t.Toggle()
			}
		}

	case key.Matches(msg, keys.Run):
		return m, m.activeWin().StartRun(false)

	case key.Matches(msg, keys.RunGUI):
		return m, m.activeWin().StartRun(true)

	case key.Matches(msg, keys.PrevWin):
		if len(m.windows) > 1 {
			m.activeIdx = (m.activeIdx - 1 + len(m.windows)) % len(m.windows)
			if m.windows[m.activeIdx].State == WinStateScanning {
				return m, m.activeWin().ScanCmd()
			}
		}

	case key.Matches(msg, keys.NextWin):
		if len(m.windows) > 1 {
			m.activeIdx = (m.activeIdx + 1) % len(m.windows)
			if m.windows[m.activeIdx].State == WinStateScanning {
				return m, m.activeWin().ScanCmd()
			}
		}

	case key.Matches(msg, keys.Rescan):
		win := m.activeWin()
		if win.State == WinStateRunning {
			win.StatusMsg = "Cannot rescan while tests are running"
			return m, nil
		}
		win.State = WinStateScanning
		m.state = StateScanning
		return m, win.ScanCmd()

	case key.Matches(msg, keys.Cancel):
		if m.activeWin().State == WinStateRunning {
			m.activeWin().Cancel()
		}
	}

	return m, nil
}

func (m AppModel) View() string {
	switch m.state {
	case StatePicker:
		return m.picker.View()
	case StateScanning:
		return StyleHeader.Render("lazy_vunit") + "\n\n" +
			StyleSubtle.Render("  Scanning...") + "\n"
	case StateError:
		errMsg := ""
		if len(m.windows) > 0 {
			errMsg = m.activeWin().ErrMsg
		}
		return StyleHeader.Render("lazy_vunit") + "\n\n" +
			StyleFailed.Render("  Error: "+errMsg) + "\n\n" +
			StyleSubtle.Render("  Is VUnit installed? Try: pip install vunit-hdl\n") +
			StyleSubtle.Render("  Press ctrl+r to retry, q to quit.")
	case StateMain:
		if len(m.windows) == 0 {
			return ""
		}
		if m.showSettings {
			return RenderSettings(m)
		}
		if m.showHelp {
			return RenderHelp(m)
		}
		return RenderMain(m)
	}
	return ""
}

// AllWindowCounts returns aggregate (passed, failed, notRun) across all loaded windows.
func (m AppModel) AllWindowCounts() (int, int, int) {
	var tp, tf, tn int
	for i := range m.windows {
		p, f, n := m.windows[i].Counts()
		tp += p
		tf += f
		tn += n
	}
	return tp, tf, tn
}
