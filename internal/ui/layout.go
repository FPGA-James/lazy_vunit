// internal/ui/layout.go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	title := fmt.Sprintf(" lazy_vunit — %s  [%s]", m.gitRoot, win.DisplayName())
	return StyleHeader.Width(width).Render(title)
}

func renderTree(w *WindowModel, width, height int) string {
	header := StyleHeader.Width(width).Render(" TESTS  ctrl+r scan")
	if w.Tree == nil {
		return lipgloss.JoinVertical(lipgloss.Left, header, "")
	}

	visible := w.Tree.Visible()
	cursor := w.Tree.CursorNode()

	var sb strings.Builder
	for _, node := range visible {
		line := renderTreeNode(node)
		if node == cursor {
			line = StyleCursor.Width(width - 2).Render(line)
		}
		sb.WriteString(line + "\n")
	}

	content := sb.String()
	box := StyleBorder.Width(width).Height(height).Render(content)
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

	lines := w.Output
	start := 0
	if len(lines) > height {
		start = len(lines) - height
	}
	content := strings.Join(lines[start:], "\n")
	box := StyleBorder.Width(width).Height(height).Render(content)
	return lipgloss.JoinVertical(lipgloss.Left, header, box)
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
		win.StatusMsg = "" // clear after render
	}

	bar := lipgloss.NewStyle().
		Width(width).
		Background(lipgloss.Color("#16213e")).
		Render(left + hints)

	return bar
}
