package runner

import (
	"bufio"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lazyvunit/lazy_vunit/internal/tree"
)

// OutputLineMsg is sent for each line of stdout.
type OutputLineMsg struct{ Text string }

// StatusUpdateMsg is sent when a pass/fail is detected in output.
type StatusUpdateMsg struct {
	TestName string
	Status   tree.Status
}

// RunDoneMsg is sent when the subprocess exits.
type RunDoneMsg struct{ Err error }

// CancelFunc sends SIGTERM to the running subprocess. Safe to call after exit.
type CancelFunc func()

// Run spawns `python <runPy> <args...>` and returns:
//   - tea.Cmd: call once to kick off streaming (sends first OutputLineMsg, then RunDoneMsg)
//   - CancelFunc: sends SIGTERM
//   - <-chan tea.Msg: channel of subsequent messages after the first
func Run(runPy string, args []string) (tea.Cmd, CancelFunc, <-chan tea.Msg) {
	ch := make(chan tea.Msg, 256)

	cmdArgs := append([]string{runPy}, args...)
	cmd := exec.Command("python", cmdArgs...)
	cmd.Dir = filepath.Dir(runPy)

	var once sync.Once
	cancelFn := CancelFunc(func() {
		once.Do(func() {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(syscall.SIGTERM)
			}
		})
	})

	startCmd := tea.Cmd(func() tea.Msg {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return RunDoneMsg{Err: err}
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return RunDoneMsg{Err: err}
		}
		if err := cmd.Start(); err != nil {
			return RunDoneMsg{Err: err}
		}
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				ch <- OutputLineMsg{Text: scanner.Text()}
			}
		}()
		go func() {
			scanner := bufio.NewScanner(stdout)
			var prevLine string
			for scanner.Scan() {
				line := scanner.Text()
				ch <- OutputLineMsg{Text: line}
				if result, ok := ParseLine(line, prevLine); ok {
					ch <- StatusUpdateMsg{TestName: result.TestName, Status: result.Status}
				}
				prevLine = line
			}
			wg.Wait() // ensure all stderr lines are sent before RunDoneMsg
			err := cmd.Wait()
			ch <- RunDoneMsg{Err: err}
			close(ch)
		}()
		// Return the first message from the channel.
		return <-ch
	})

	return startCmd, cancelFn, ch
}

// NextMsg returns a tea.Cmd that reads the next message from the runner channel.
// Call this from the model's Update after each OutputLineMsg or StatusUpdateMsg
// to keep draining the stream.
func NextMsg(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

// RunBatched handles large test sets by splitting args into batches of batchSize.
// Returns the tea.Cmd for the first batch plus remaining args for subsequent batches.
// The caller must call RunBatched again with remaining args when RunDoneMsg arrives.
func RunBatched(runPy string, args []string, batchSize int) (tea.Cmd, CancelFunc, <-chan tea.Msg, []string) {
	if batchSize <= 0 {
		batchSize = 200
	}
	batch := args
	remaining := []string{}
	if len(args) > batchSize {
		batch = args[:batchSize]
		remaining = args[batchSize:]
	}
	cmd, cancel, ch := Run(runPy, batch)
	return cmd, cancel, ch, remaining
}
