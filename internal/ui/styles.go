// internal/ui/styles.go
package ui

import "github.com/charmbracelet/lipgloss"

var (
	ColorPassed  = lipgloss.Color("#50fa7b")
	ColorFailed  = lipgloss.Color("#ff5555")
	ColorRunning = lipgloss.Color("#8be9fd")
	ColorNotRun  = lipgloss.Color("#f1fa8c")
	ColorDir     = lipgloss.Color("#8888ff")
	ColorSubtle  = lipgloss.Color("#6272a4")
	ColorHeader  = lipgloss.Color("#7070a0")

	StylePassed  = lipgloss.NewStyle().Foreground(ColorPassed)
	StyleFailed  = lipgloss.NewStyle().Foreground(ColorFailed)
	StyleRunning = lipgloss.NewStyle().Foreground(ColorRunning)
	StyleNotRun  = lipgloss.NewStyle().Foreground(ColorNotRun)
	StyleDir     = lipgloss.NewStyle().Foreground(ColorDir)
	StyleSubtle  = lipgloss.NewStyle().Foreground(ColorSubtle)
	StyleHeader  = lipgloss.NewStyle().Foreground(ColorHeader)
	StyleCursor  = lipgloss.NewStyle().Background(lipgloss.Color("#1e2a4a")).Bold(true)

	StyleBorder = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#333333"))
)

const (
	IconPassed  = "✓"
	IconFailed  = "✗"
	IconNotRun  = "○"
	IconRunning = "~"
)
