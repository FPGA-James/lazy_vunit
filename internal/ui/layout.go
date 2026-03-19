// internal/ui/layout.go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/lazyvunit/lazy_vunit/internal/persist"
	"github.com/lazyvunit/lazy_vunit/internal/tree"
)

// RenderMain renders the three-pane layout for StateMain.
func RenderMain(m AppModel) string {
	w := m.activeWin()
	totalWidth := m.termWidth
	if totalWidth < 40 {
		totalWidth = 120
	}
	totalHeight := m.termHeight
	if totalHeight < 10 {
		totalHeight = 40
	}

	treeWidth := totalWidth * 38 / 100
	outputWidth := totalWidth - treeWidth - 3
	innerHeight := totalHeight - 4 // header + bottom bar

	header := renderHeader(m, totalWidth)
	treePane := renderTree(w, treeWidth, innerHeight)
	outputPane := renderOutput(w, outputWidth, innerHeight)
	body := lipgloss.JoinHorizontal(lipgloss.Top, treePane, outputPane)
	bottomBar := renderBottomBar(m, totalWidth)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, bottomBar)
}

func renderHeader(m AppModel, width int) string {
	win := m.activeWin()
	s := win.Settings

	var flags []string
	if s.Clean         { flags = append(flags, StyleFailed.Render("clean")) }
	if s.Verbose       { flags = append(flags, StyleRunning.Render("verbose")) }
	if s.CompileOnly   { flags = append(flags, StyleRunning.Render("compile")) }
	if s.ElaborateOnly { flags = append(flags, StyleRunning.Render("elaborate")) }
	if s.FailFast      { flags = append(flags, StyleFailed.Render("fail-fast")) }
	if s.XUnitXML      { flags = append(flags, StylePassed.Render("xunit")) }

	title := fmt.Sprintf(" lazy_vunit — %s  [%s]", m.gitRoot, win.DisplayName())
	if len(flags) > 0 {
		title += "  " + strings.Join(flags, "  ")
	}
	return StyleHeader.Width(width).Render(title)
}

func renderTree(w *WindowModel, width, height int) string {
	header := StyleHeader.Width(width).Render(" TESTS  ctrl+r scan")
	if w.Tree == nil {
		return lipgloss.JoinVertical(lipgloss.Left, header, StyleBorder.Width(width).Height(height).Render(""))
	}

	visible := w.Tree.Visible()
	cursor := w.Tree.CursorNode()
	contentHeight := height - 2 // subtract top and bottom border rows

	var sb strings.Builder
	for i, node := range visible {
		if i >= contentHeight {
			break
		}
		line := renderTreeNode(node)
		if node == cursor {
			line = StyleCursor.Width(width - 2).Render(line)
		} else {
			line = lipgloss.NewStyle().MaxWidth(width - 2).Render(line)
		}
		sb.WriteString(line + "\n")
	}

	box := StyleBorder.Width(width).Height(height).Render(sb.String())
	return lipgloss.JoinVertical(lipgloss.Left, header, box)
}

func renderTreeNode(n *tree.Node) string {
	switch n.Kind {
	case tree.DirNode:
		expand := "▶"
		if n.Expanded {
			expand = "▼"
		}
		return StyleDir.Render(expand+" ") + n.Name

	case tree.BenchNode:
		expand := "▶"
		if n.Expanded {
			expand = "▼"
		}
		return "  " + expand + " " + n.Name

	case tree.TestNode:
		return "    " + statusIcon(n.Status) + " " + n.Name
	}
	return n.Name
}

func statusIcon(s tree.Status) string {
	switch s {
	case tree.Passed:
		return StylePassed.Render(IconPassed)
	case tree.Failed:
		return StyleFailed.Render(IconFailed)
	case tree.Running:
		return StyleRunning.Render(IconRunning)
	default:
		return StyleNotRun.Render(IconNotRun)
	}
}

func renderOutput(w *WindowModel, width, height int) string {
	title := " OUTPUT"
	if w.OutputTitle != "" {
		title += "  " + StyleSubtle.Render(w.OutputTitle)
	}
	header := StyleHeader.Width(width).Render(title)

	contentHeight := height - 2 // subtract top and bottom border rows
	lines := w.Output
	start := 0
	if len(lines) > contentHeight {
		start = len(lines) - contentHeight
	}
	var sb strings.Builder
	for i, line := range lines[start:] {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(lipgloss.NewStyle().MaxWidth(width - 2).Render(line))
	}
	box := StyleBorder.Width(width).Height(height).Render(sb.String())
	return lipgloss.JoinVertical(lipgloss.Left, header, box)
}

// settingItem describes one row in the settings panel.
type settingItem struct {
	label string
	desc  string
	value func(persist.Settings) bool
}

var settingItems = []settingItem{
	{"clean",          "wipe output dir before run",           func(s persist.Settings) bool { return s.Clean }},
	{"verbose",        "print all test output",                func(s persist.Settings) bool { return s.Verbose }},
	{"compile only",   "compile only, skip simulation",        func(s persist.Settings) bool { return s.CompileOnly }},
	{"elaborate only", "elaborate only, skip simulation",      func(s persist.Settings) bool { return s.ElaborateOnly }},
	{"fail fast",      "stop on first failure",                func(s persist.Settings) bool { return s.FailFast }},
	{"xunit xml",      "write report to .lazyvunit/<key>.xml", func(s persist.Settings) bool { return s.XUnitXML }},
}

// RenderSettings renders the settings toggles as a floating box over the main layout.
func RenderSettings(m AppModel) string {
	bg := RenderMain(m)
	win := m.activeWin()
	s := win.Settings
	cursor := m.SettingsCursor()

	var sb strings.Builder
	sb.WriteString(StyleHeader.Render(fmt.Sprintf(" settings — %s", win.DisplayName())) + "\n\n")
	for i, item := range settingItems {
		check := "[ ]"
		if item.value(s) {
			check = StylePassed.Render("[✓]")
		}
		row := fmt.Sprintf("  %s %-16s %s", check, item.label, StyleSubtle.Render(item.desc))
		if i == cursor {
			row = StyleCursor.Render(row)
		}
		sb.WriteString(row + "\n")
	}
	sb.WriteString("\n" + StyleSubtle.Render("  ↑/↓ navigate  space toggle  s close"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7070a0")).
		Padding(0, 1).
		Render(sb.String())

	boxW := lipgloss.Width(box)
	boxH := lipgloss.Height(box)
	x := (m.termWidth - boxW) / 2
	y := (m.termHeight - boxH) / 2
	if x < 0 { x = 0 }
	if y < 0 { y = 0 }

	return placeOverlay(x, y, box, bg)
}

// RenderHelp renders the help keybindings as a floating box over the main layout.
func RenderHelp(m AppModel) string {
	bg := RenderMain(m)

	keys := DefaultKeys
	rows := []struct{ key, desc string }{
		{keys.Up.Help().Key, keys.Up.Help().Desc},
		{keys.Down.Help().Key, keys.Down.Help().Desc},
		{keys.Left.Help().Key, keys.Left.Help().Desc},
		{keys.Right.Help().Key, keys.Right.Help().Desc},
		{keys.Run.Help().Key, keys.Run.Help().Desc},
		{keys.RunGUI.Help().Key, keys.RunGUI.Help().Desc},
		{keys.PrevWin.Help().Key, keys.PrevWin.Help().Desc},
		{keys.NextWin.Help().Key, keys.NextWin.Help().Desc},
		{keys.Rescan.Help().Key, keys.Rescan.Help().Desc},
		{keys.Cancel.Help().Key, keys.Cancel.Help().Desc},
		{keys.Quit.Help().Key, keys.Quit.Help().Desc},
		{keys.Help.Help().Key, keys.Help.Help().Desc},
		{keys.Settings.Help().Key, keys.Settings.Help().Desc},
		{keys.Escape.Help().Key,   keys.Escape.Help().Desc},
	}

	var sb strings.Builder
	sb.WriteString(StyleHeader.Render(" keybindings") + "\n\n")
	for _, r := range rows {
		sb.WriteString(fmt.Sprintf("  %-10s  %s\n",
			StyleRunning.Render(r.key),
			StyleSubtle.Render(r.desc),
		))
	}
	sb.WriteString("\n" + StyleSubtle.Render("  ? to close"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7070a0")).
		Padding(0, 1).
		Render(sb.String())

	boxW := lipgloss.Width(box)
	boxH := lipgloss.Height(box)
	x := (m.termWidth - boxW) / 2
	y := (m.termHeight - boxH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	return placeOverlay(x, y, box, bg)
}

// placeOverlay draws fg on top of bg at column x, row y.
func placeOverlay(x, y int, fg, bg string) string {
	fgLines := strings.Split(fg, "\n")
	bgLines := strings.Split(bg, "\n")
	for i, fgLine := range fgLines {
		row := y + i
		if row < 0 || row >= len(bgLines) {
			continue
		}
		bgLine := bgLines[row]
		bgW := lipgloss.Width(bgLine)
		fgW := lipgloss.Width(fgLine)
		// left portion: columns 0..x-1
		left := ansi.Truncate(bgLine, x, "")
		// pad if bg is shorter than x
		if bgW < x {
			left += strings.Repeat(" ", x-bgW)
		}
		// right portion: columns x+fgW..end
		right := ""
		if x+fgW < bgW {
			right = ansi.TruncateLeft(bgLine, x+fgW, "")
		}
		bgLines[row] = left + fgLine + right
	}
	return strings.Join(bgLines, "\n")
}

func renderBottomBar(m AppModel, width int) string {
	win := m.activeWin()
	p, f, n := win.Counts()
	ap, af, an := m.AllWindowCounts()

	winName := win.DisplayName()

	left := fmt.Sprintf(" [%s] %s %s %s  │  all: %s %s %s",
		winName,
		StylePassed.Render(fmt.Sprintf("✓ %d", p)),
		StyleFailed.Render(fmt.Sprintf("✗ %d", f)),
		StyleNotRun.Render(fmt.Sprintf("○ %d", n)),
		StylePassed.Render(fmt.Sprintf("✓ %d", ap)),
		StyleFailed.Render(fmt.Sprintf("✗ %d", af)),
		StyleNotRun.Render(fmt.Sprintf("○ %d", an)),
	)

	hints := StyleSubtle.Render(" space run  g gui  [ ]  ctrl+r  q quit  ? help")

	if win.StatusMsg != "" {
		hints = StyleFailed.Render(" " + win.StatusMsg)
	}

	bar := lipgloss.NewStyle().
		Width(width).
		Background(lipgloss.Color("#16213e")).
		Render(left + hints)

	return bar
}
