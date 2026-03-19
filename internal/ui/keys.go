// internal/ui/keys.go
package ui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	Run        key.Binding
	RunGUI     key.Binding
	PrevWin    key.Binding
	NextWin    key.Binding
	Rescan     key.Binding
	Cancel     key.Binding
	Quit       key.Binding
	Help       key.Binding
	Settings   key.Binding
	Escape     key.Binding
	FullOutput key.Binding
}

var DefaultKeys = KeyMap{
	Up:         key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "up")),
	Down:       key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "down")),
	Left:       key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "collapse")),
	Right:      key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "expand")),
	Run:        key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "run")),
	RunGUI:     key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "gui")),
	PrevWin:    key.NewBinding(key.WithKeys("["), key.WithHelp("[", "prev window")),
	NextWin:    key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next window")),
	Rescan:     key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "rescan")),
	Cancel:     key.NewBinding(key.WithKeys("ctrl+c", "x"), key.WithHelp("x", "cancel")),
	Quit:       key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Settings:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "settings")),
	Escape:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
	FullOutput: key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "expand output")),
}
