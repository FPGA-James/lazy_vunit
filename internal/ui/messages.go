// internal/ui/messages.go
package ui

import "github.com/lazyvunit/lazy_vunit/internal/scanner"

// ScanDoneMsg is sent when a VUnit scan completes.
type ScanDoneMsg struct {
	Entries []scanner.TestEntry
	Err     error
}

// ScanStartMsg triggers a rescan in the Bubbletea update loop.
type ScanStartMsg struct{}
