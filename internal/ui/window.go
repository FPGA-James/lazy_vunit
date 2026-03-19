package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lazyvunit/lazy_vunit/internal/finder"
	"github.com/lazyvunit/lazy_vunit/internal/persist"
	"github.com/lazyvunit/lazy_vunit/internal/runner"
	"github.com/lazyvunit/lazy_vunit/internal/scanner"
	"github.com/lazyvunit/lazy_vunit/internal/tree"
)

type WindowState int

const (
	WinStateScanning WindowState = iota
	WinStateReady
	WinStateRunning
	WinStateError
)

type WindowModel struct {
	Script      finder.RunScript
	AllScripts  []finder.RunScript // for display name resolution
	GitRoot     string
	LazyDir     string             // path to .lazyvunit/ directory
	State       WindowState
	Tree        *tree.Tree
	Output      []string           // lines of terminal output
	OutputTitle string             // test name or dir name shown in output pane header
	ErrMsg      string
	StatusMsg   string             // transient bottom-bar message
	RunnerCh    <-chan tea.Msg
	CancelFn    runner.CancelFunc
	PendingArgs []string           // remaining batch args
	Results     persist.Store
	Settings    persist.Settings  // per-window VUnit run flags
}

func NewWindowModel(script finder.RunScript, allScripts []finder.RunScript, gitRoot string) WindowModel {
	lazyDir := filepath.Join(gitRoot, ".lazyvunit")
	results, _ := persist.Load(lazyDir, script.WindowKey)
	settings, _ := persist.LoadSettings(lazyDir, script.WindowKey)
	return WindowModel{
		Script:     script,
		AllScripts: allScripts,
		GitRoot:    gitRoot,
		LazyDir:    lazyDir,
		State:      WinStateScanning,
		Results:    results,
		Settings:   settings,
	}
}

// SettingCount returns the total number of configurable settings.
// Used to clamp the settings panel cursor.
func SettingCount() int { return 6 }

// ToggleSetting flips the setting at the given index and persists the change.
// CompileOnly (2) and ElaborateOnly (3) are mutually exclusive.
func (w *WindowModel) ToggleSetting(idx int) {
	switch idx {
	case 0:
		w.Settings.Clean = !w.Settings.Clean
	case 1:
		w.Settings.Verbose = !w.Settings.Verbose
	case 2:
		w.Settings.CompileOnly = !w.Settings.CompileOnly
		if w.Settings.CompileOnly {
			w.Settings.ElaborateOnly = false
		}
	case 3:
		w.Settings.ElaborateOnly = !w.Settings.ElaborateOnly
		if w.Settings.ElaborateOnly {
			w.Settings.CompileOnly = false
		}
	case 4:
		w.Settings.FailFast = !w.Settings.FailFast
	case 5:
		w.Settings.XUnitXML = !w.Settings.XUnitXML
	}
	_ = persist.SaveSettings(w.LazyDir, w.Script.WindowKey, w.Settings)
}

// DisplayName returns the window's display name (leaf or full reldir on collision).
func (w *WindowModel) DisplayName() string {
	return finder.DisplayName(w.AllScripts, w.Script)
}

// ScanCmd returns a tea.Cmd that runs the VUnit scan and sends ScanDoneMsg.
func (w *WindowModel) ScanCmd() tea.Cmd {
	runPy := w.Script.AbsPath
	jsonPath := filepath.Join(os.TempDir(), fmt.Sprintf("lazyvunit_%d.json", os.Getpid()))
	return func() tea.Msg {
		entries, err := scanner.Scan(runPy, jsonPath)
		return ScanDoneMsg{Entries: entries, Err: err}
	}
}

// ApplyScanResult builds the tree from scan entries and merges persisted results.
func (w *WindowModel) ApplyScanResult(entries []scanner.TestEntry, err error) {
	if err != nil {
		w.State = WinStateError
		w.ErrMsg = err.Error()
		return
	}
	w.Tree = tree.BuildTree(entries, w.GitRoot)
	for name, r := range w.Results {
		s := tree.NotRun
		if r.Status == "pass" {
			s = tree.Passed
		} else if r.Status == "fail" {
			s = tree.Failed
		}
		w.Tree.SetStatus(name, s)
	}
	w.State = WinStateReady
}

// StartRun begins running the tests for the current tree selection.
// Returns the initial tea.Cmd to kick off the runner, or nil if nothing to run.
func (w *WindowModel) StartRun(guiMode bool) tea.Cmd {
	if w.Tree == nil || w.State != WinStateReady {
		return nil
	}

	// Compile-only / elaborate-only: run without test patterns, no GUI
	if w.Settings.CompileOnly || w.Settings.ElaborateOnly {
		var args []string
		if w.Settings.Clean {
			args = append(args, "--clean")
		}
		if w.Settings.Verbose {
			args = append(args, "--verbose")
		}
		if w.Settings.CompileOnly {
			args = append(args, "--compile")
			w.OutputTitle = "compile"
		} else {
			args = append(args, "--elaborate")
			w.OutputTitle = "elaborate"
		}
		w.Output = []string{fmt.Sprintf("# Running: python %s %s", w.Script.AbsPath, strings.Join(args, " "))}
		w.State = WinStateRunning
		cmd, cancelFn, ch := runner.Run(w.Script.AbsPath, args)
		w.CancelFn = cancelFn
		w.RunnerCh = ch
		w.PendingArgs = nil
		return cmd
	}

	// Normal test run
	node := w.Tree.CursorNode()
	if node == nil {
		return nil
	}
	if guiMode && node.Kind != tree.TestNode {
		w.StatusMsg = "GUI mode requires a single test — navigate to a test case"
		return nil
	}

	var args []string
	if w.Settings.Clean {
		args = append(args, "--clean")
	}
	args = append(args, w.Tree.RunPattern()...)
	if guiMode {
		args = append(args, "--gui")
	}
	if w.Settings.Verbose {
		args = append(args, "--verbose")
	}
	if w.Settings.FailFast {
		args = append(args, "--fail-fast")
	}
	if w.Settings.XUnitXML {
		reportPath := filepath.Join(w.LazyDir, w.Script.WindowKey+"_report.xml")
		args = append(args, "--xunit-xml", reportPath)
	}

	for _, name := range fullNamesFromNode(node) {
		w.Tree.SetStatus(name, tree.Running)
	}

	w.Output = []string{fmt.Sprintf("# Running: python %s %s", w.Script.AbsPath, strings.Join(args, " "))}
	w.OutputTitle = node.Name
	w.State = WinStateRunning

	const batchSize = 200
	firstCmd, cancelFn, ch, remaining := runner.RunBatched(w.Script.AbsPath, args, batchSize)
	w.CancelFn = cancelFn
	w.RunnerCh = ch
	w.PendingArgs = remaining
	return firstCmd
}

// HandleRunnerMsg processes a message from the runner and returns the next tea.Cmd (if any).
func (w *WindowModel) HandleRunnerMsg(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case runner.OutputLineMsg:
		w.Output = append(w.Output, m.Text)
		return runner.NextMsg(w.RunnerCh)
	case runner.StatusUpdateMsg:
		w.Tree.SetStatus(m.TestName, m.Status)
		return runner.NextMsg(w.RunnerCh)
	case runner.RunDoneMsg:
		// Apply exit-code fallback only to tests in the current batch (still Running)
		fallbackStatus := tree.Passed
		if m.Err != nil {
			fallbackStatus = tree.Failed
		}
		w.applyRunningFallback(fallbackStatus)

		// Start next batch if pending
		if len(w.PendingArgs) > 0 {
			firstCmd, cancelFn, ch, remaining := runner.RunBatched(w.Script.AbsPath, w.PendingArgs, 200)
			w.CancelFn = cancelFn
			w.RunnerCh = ch
			w.PendingArgs = remaining
			w.Output = append(w.Output, "── batch complete, continuing ──")
			return firstCmd
		}

		w.State = WinStateReady
		w.CancelFn = nil
		w.saveResults()
		return nil
	}
	return nil
}

// Cancel sends SIGTERM to the running subprocess and resets in-flight tests to NotRun.
func (w *WindowModel) Cancel() {
	if w.CancelFn != nil {
		w.CancelFn()
	}
	w.applyRunningFallback(tree.NotRun)
	w.State = WinStateReady
	w.PendingArgs = nil
	w.CancelFn = nil
	w.saveResults()
}

// applyRunningFallback sets all leaf tests currently in Running state to the given status.
// This only affects tests that were part of the active batch (they were set to Running).
// Tests in future batches (still NotRun) are not affected.
func (w *WindowModel) applyRunningFallback(s tree.Status) {
	if w.Tree == nil {
		return
	}
	for _, node := range w.Tree.AllLeaves() {
		if node.Status == tree.Running {
			w.Tree.SetStatus(node.FullName, s)
		}
	}
}

func (w *WindowModel) saveResults() {
	if w.Tree == nil {
		return
	}
	now := time.Now().UTC()
	for _, node := range w.Tree.AllLeaves() {
		switch node.Status {
		case tree.Passed:
			w.Results[node.FullName] = persist.Result{Status: "pass", RanAt: now}
		case tree.Failed:
			w.Results[node.FullName] = persist.Result{Status: "fail", RanAt: now}
		}
	}
	_ = persist.Save(w.LazyDir, w.Script.WindowKey, w.Results)
}

// Counts returns (passed, failed, notRun) for this window's test leaf nodes.
func (w *WindowModel) Counts() (int, int, int) {
	if w.Tree == nil {
		return 0, 0, 0
	}
	var p, f, n int
	for _, node := range w.Tree.AllLeaves() {
		switch node.Status {
		case tree.Passed:
			p++
		case tree.Failed:
			f++
		default:
			n++
		}
	}
	return p, f, n
}

// fullNamesFromNode returns all TestNode FullNames reachable from n.
func fullNamesFromNode(n *tree.Node) []string {
	if n.Kind == tree.TestNode {
		return []string{n.FullName}
	}
	var names []string
	for _, child := range n.Children {
		names = append(names, fullNamesFromNode(child)...)
	}
	return names
}
