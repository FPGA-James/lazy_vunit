package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lazyvunit/lazy_vunit/internal/finder"
	"github.com/lazyvunit/lazy_vunit/internal/persist"
	"github.com/lazyvunit/lazy_vunit/internal/ui"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error getting working directory:", err)
		os.Exit(1)
	}

	gitRoot := finder.FindGitRoot(cwd)
	scripts, err := finder.FindRunScripts(gitRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error scanning for run scripts:", err)
		os.Exit(1)
	}

	// Filter to scripts at or below cwd
	var filtered []finder.RunScript
	for _, s := range scripts {
		rel, err := filepath.Rel(cwd, s.AbsPath)
		if err == nil && !strings.HasPrefix(rel, "..") {
			filtered = append(filtered, s)
		}
	}
	scripts = filtered

	if len(scripts) == 0 {
		fmt.Fprintf(os.Stderr,
			"No VUnit run script found.\nSearched: %s\n\nExpected a run.py containing VUnit.from_argv.\n",
			cwd)
		os.Exit(1)
	}

	// Ensure .lazyvunit/ is in .gitignore
	if err := persist.EnsureGitignore(gitRoot); err != nil {
		fmt.Fprintln(os.Stderr, "warning: could not update .gitignore:", err)
	}

	model := ui.NewAppModel(scripts, gitRoot, cwd)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error running program:", err)
		os.Exit(1)
	}
}
