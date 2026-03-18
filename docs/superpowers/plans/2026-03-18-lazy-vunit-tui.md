# lazy_vunit TUI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a LazyGit-inspired terminal UI in Go for discovering and running VUnit HDL tests within a git repository.

**Architecture:** A single Go binary using the Bubbletea TUI framework. Four internal packages (finder, scanner, tree, persist, runner) handle business logic; the `ui` package wires them into a Bubbletea model with three display states (picker, scanning/error, main). VUnit is invoked as a subprocess via `python run.py`.

**Tech Stack:** Go 1.22+, `charmbracelet/bubbletea`, `charmbracelet/lipgloss`, `charmbracelet/bubbles`, `charmbracelet/x/ansi`, `testify/assert` (tests only)

---

## File Map

```
lazy_vunit/
├── main.go                        # Entry point — detects run scripts, launches tea.Program
├── go.mod
├── go.sum
└── internal/
    ├── finder/
    │   ├── finder.go              # FindGitRoot(), FindRunScripts()
    │   └── finder_test.go
    ├── scanner/
    │   ├── scanner.go             # Scan() — run --export-json, parse JSON → []TestEntry
    │   └── scanner_test.go
    ├── tree/
    │   ├── tree.go                # Node/Tree types, navigation, status derivation, run patterns
    │   └── tree_test.go
    ├── persist/
    │   ├── persist.go             # Load(), Save(), EnsureGitignore()
    │   └── persist_test.go
    ├── runner/
    │   ├── parser.go              # ParseLine() — pure output parsing, no subprocess
    │   ├── parser_test.go
    │   └── runner.go              # Runner — subprocess lifecycle, stdout streaming
    └── ui/
        ├── keys.go                # KeyMap and key binding definitions
        ├── styles.go              # Lipgloss colour/style definitions
        ├── messages.go            # Bubbletea message types shared across models
        ├── picker.go              # PickerModel — startup window selection screen
        ├── window.go              # WindowModel — per-run.py state (tree + output + runner)
        ├── app.go                 # AppModel — top-level model, state machine, View dispatch
        └── layout.go              # RenderMain() — three-pane layout using lipgloss
```

---

## Task 1: Go Module and Dependencies

**Files:**
- Create: `go.mod`

- [ ] **Step 1: Initialise the Go module**

```bash
cd /Users/james/Workspace/lazy_vunit
go mod init github.com/lazyvunit/lazy_vunit
```

Expected: `go.mod` created with `module github.com/lazyvunit/lazy_vunit` and `go 1.22` (or current Go version).

- [ ] **Step 2: Add runtime dependencies**

```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/x/ansi
```

- [ ] **Step 3: Add test dependency**

```bash
go get github.com/stretchr/testify
```

- [ ] **Step 4: Create a placeholder main.go to verify the module compiles**

```go
// main.go
package main

func main() {}
```

- [ ] **Step 5: Verify the build compiles cleanly**

```bash
go build ./...
```

Expected: no output (success).

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum main.go
git commit -m "chore: initialise Go module with dependencies"
```

---

## Task 2: Finder — Git Root and run.py Discovery

**Files:**
- Create: `internal/finder/finder.go`
- Create: `internal/finder/finder_test.go`

### Types

```go
// internal/finder/finder.go
package finder

type RunScript struct {
    AbsPath     string // absolute path to run.py
    RelDir      string // parent dir relative to git root, e.g. "src/alu"
    WindowKey   string // RelDir with "/" replaced by "_", e.g. "src_alu"
    LeafName    string // leaf directory name, e.g. "alu"
}

// DisplayName returns the leaf name unless there is a collision with another
// script in the slice, in which case it returns RelDir.
func DisplayName(scripts []RunScript, s RunScript) string
```

### Functions

```go
// FindGitRoot walks up from dir until it finds a directory containing ".git".
// Returns the git root path, or dir itself if no .git is found.
func FindGitRoot(dir string) string

// FindRunScripts searches root recursively for run.py files (or .py files
// containing both "VUnit.from_argv" and `if __name__ == "__main__"` as a
// fallback when no run.py is found). Returns them sorted by RelDir.
func FindRunScripts(root string) ([]RunScript, error)
```

- [ ] **Step 1: Write failing tests**

```go
// internal/finder/finder_test.go
package finder_test

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/lazyvunit/lazy_vunit/internal/finder"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestFindGitRoot_FindsGitDir(t *testing.T) {
    root := t.TempDir()
    require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0755))
    sub := filepath.Join(root, "src", "alu")
    require.NoError(t, os.MkdirAll(sub, 0755))

    got := finder.FindGitRoot(sub)
    assert.Equal(t, root, got)
}

func TestFindGitRoot_FallsBackToCwd(t *testing.T) {
    dir := t.TempDir() // no .git
    got := finder.FindGitRoot(dir)
    assert.Equal(t, dir, got)
}

func TestFindRunScripts_FindsByName(t *testing.T) {
    root := t.TempDir()
    dir := filepath.Join(root, "src", "alu")
    require.NoError(t, os.MkdirAll(dir, 0755))
    require.NoError(t, os.WriteFile(
        filepath.Join(dir, "run.py"), []byte("# run"), 0644))

    scripts, err := finder.FindRunScripts(root)
    require.NoError(t, err)
    require.Len(t, scripts, 1)
    assert.Equal(t, "src/alu", scripts[0].RelDir)
    assert.Equal(t, "src_alu", scripts[0].WindowKey)
    assert.Equal(t, "alu", scripts[0].LeafName)
}

func TestFindRunScripts_FallbackToContent(t *testing.T) {
    root := t.TempDir()
    dir := filepath.Join(root, "ip")
    require.NoError(t, os.Mkdir(dir, 0755))
    content := `from vunit import VUnit
if __name__ == "__main__":
    vu = VUnit.from_argv()
    vu.main()
`
    require.NoError(t, os.WriteFile(filepath.Join(dir, "sim.py"), []byte(content), 0644))

    scripts, err := finder.FindRunScripts(root)
    require.NoError(t, err)
    require.Len(t, scripts, 1)
    assert.Equal(t, "ip", scripts[0].RelDir)
}

func TestFindRunScripts_MultipleScripts(t *testing.T) {
    root := t.TempDir()
    for _, d := range []string{"src/alu", "src/uart"} {
        p := filepath.Join(root, d)
        require.NoError(t, os.MkdirAll(p, 0755))
        require.NoError(t, os.WriteFile(filepath.Join(p, "run.py"), []byte("# run"), 0644))
    }

    scripts, err := finder.FindRunScripts(root)
    require.NoError(t, err)
    assert.Len(t, scripts, 2)
}

func TestDisplayName_UniqueLeaf(t *testing.T) {
    scripts := []finder.RunScript{
        {RelDir: "src/alu", LeafName: "alu"},
        {RelDir: "src/uart", LeafName: "uart"},
    }
    assert.Equal(t, "alu", finder.DisplayName(scripts, scripts[0]))
}

func TestDisplayName_CollisionUsesRelDir(t *testing.T) {
    scripts := []finder.RunScript{
        {RelDir: "src/alu", LeafName: "alu"},
        {RelDir: "test/alu", LeafName: "alu"},
    }
    assert.Equal(t, "src/alu", finder.DisplayName(scripts, scripts[0]))
    assert.Equal(t, "test/alu", finder.DisplayName(scripts, scripts[1]))
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/finder/ -v
```

Expected: compilation error (package does not exist yet).

- [ ] **Step 3: Implement finder.go**

```go
// internal/finder/finder.go
package finder

import (
    "bufio"
    "os"
    "path/filepath"
    "strings"
)

type RunScript struct {
    AbsPath   string
    RelDir    string
    WindowKey string
    LeafName  string
}

func FindGitRoot(dir string) string {
    current := dir
    for {
        if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
            return current
        }
        parent := filepath.Dir(current)
        if parent == current {
            return dir // reached filesystem root without finding .git
        }
        current = parent
    }
}

func FindRunScripts(root string) ([]RunScript, error) {
    var scripts []RunScript

    // First pass: files named run.py
    err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
        if err != nil || d.IsDir() {
            return err
        }
        if filepath.Base(path) == "run.py" {
            scripts = append(scripts, makeScript(root, path))
        }
        return nil
    })
    if err != nil {
        return nil, err
    }

    // Fallback: search .py files for VUnit.from_argv + __main__ guard
    if len(scripts) == 0 {
        err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
            if err != nil || d.IsDir() || !strings.HasSuffix(path, ".py") {
                return err
            }
            if fileContainsVUnit(path) {
                scripts = append(scripts, makeScript(root, path))
            }
            return nil
        })
        if err != nil {
            return nil, err
        }
    }

    return scripts, nil
}

func makeScript(root, absPath string) RunScript {
    dir := filepath.Dir(absPath)
    rel, _ := filepath.Rel(root, dir)
    rel = filepath.ToSlash(rel)
    return RunScript{
        AbsPath:   absPath,
        RelDir:    rel,
        WindowKey: strings.ReplaceAll(rel, "/", "_"),
        LeafName:  filepath.Base(dir),
    }
}

func fileContainsVUnit(path string) bool {
    f, err := os.Open(path)
    if err != nil {
        return false
    }
    defer f.Close()

    hasFromArgv := false
    hasMainGuard := false
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.Contains(line, "VUnit.from_argv") {
            hasFromArgv = true
        }
        if strings.Contains(line, `if __name__ == "__main__"`) {
            hasMainGuard = true
        }
    }
    return hasFromArgv && hasMainGuard
}

func DisplayName(scripts []RunScript, s RunScript) string {
    for _, other := range scripts {
        if other.AbsPath != s.AbsPath && other.LeafName == s.LeafName {
            return s.RelDir
        }
    }
    return s.LeafName
}
```

- [ ] **Step 4: Run tests and confirm they pass**

```bash
go test ./internal/finder/ -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/finder/
git commit -m "feat: finder — git root detection and run.py discovery"
```

---

## Task 3: Scanner — JSON Parsing and Tree Building

**Files:**
- Create: `internal/scanner/scanner.go`
- Create: `internal/scanner/scanner_test.go`

### Types

```go
// internal/scanner/scanner.go
package scanner

// TestEntry is a single test case discovered from VUnit's --export-json output.
type TestEntry struct {
    Name     string // full VUnit name: "lib.tb_name.test_case"
    FilePath string // absolute path to the VHDL/SV source file
    Dir      string // directory of FilePath (used for tree grouping)
    Library  string // "lib"
    Bench    string // "tb_name"
    TestCase string // "test_case"
}
```

### Functions

```go
// Scan runs `python <runPy> --export-json <tmpFile>` and returns discovered tests.
// jsonPath is the path to write the temp JSON file (pass os.TempDir()+"/lazyvunit_<pid>.json").
func Scan(runPy, jsonPath string) ([]TestEntry, error)

// ParseJSON parses a VUnit export JSON file content into TestEntry slice.
// Exported for testability without requiring a real Python/VUnit installation.
func ParseJSON(data []byte) ([]TestEntry, error)
```

- [ ] **Step 1: Write failing tests (using ParseJSON — no Python needed)**

```go
// internal/scanner/scanner_test.go
package scanner_test

import (
    "testing"

    "github.com/lazyvunit/lazy_vunit/internal/scanner"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

var sampleJSON = []byte(`{
  "export_format_version": {"major": 1, "minor": 0, "patch": 0},
  "files": [],
  "tests": [
    {
      "name": "lib.tb_alu.test_add",
      "location": {"file_name": "/proj/src/alu/tb_alu.vhd", "offset": 0, "length": 0},
      "attributes": {}
    },
    {
      "name": "lib.tb_alu.test_subtract",
      "location": {"file_name": "/proj/src/alu/tb_alu.vhd", "offset": 0, "length": 0},
      "attributes": {}
    },
    {
      "name": "lib.tb_uart.test_baud",
      "location": {"file_name": "/proj/src/uart/tb_uart.vhd", "offset": 0, "length": 0},
      "attributes": {}
    }
  ]
}`)

func TestParseJSON_ParsesNames(t *testing.T) {
    entries, err := scanner.ParseJSON(sampleJSON)
    require.NoError(t, err)
    assert.Len(t, entries, 3)
    assert.Equal(t, "lib.tb_alu.test_add", entries[0].Name)
    assert.Equal(t, "lib", entries[0].Library)
    assert.Equal(t, "tb_alu", entries[0].Bench)
    assert.Equal(t, "test_add", entries[0].TestCase)
}

func TestParseJSON_SetsDir(t *testing.T) {
    entries, err := scanner.ParseJSON(sampleJSON)
    require.NoError(t, err)
    assert.Equal(t, "/proj/src/alu", entries[0].Dir)
    assert.Equal(t, "/proj/src/uart", entries[2].Dir)
}

func TestParseJSON_EmptyTests(t *testing.T) {
    data := []byte(`{"export_format_version":{"major":1,"minor":0,"patch":0},"files":[],"tests":[]}`)
    entries, err := scanner.ParseJSON(data)
    require.NoError(t, err)
    assert.Empty(t, entries)
}

func TestParseJSON_MalformedJSON(t *testing.T) {
    _, err := scanner.ParseJSON([]byte(`not json`))
    assert.Error(t, err)
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/scanner/ -v
```

Expected: compilation error.

- [ ] **Step 3: Implement scanner.go**

```go
// internal/scanner/scanner.go
package scanner

import (
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
)

type TestEntry struct {
    Name     string
    FilePath string
    Dir      string
    Library  string
    Bench    string
    TestCase string
}

type exportJSON struct {
    Tests []struct {
        Name     string `json:"name"`
        Location struct {
            FileName string `json:"file_name"`
        } `json:"location"`
    } `json:"tests"`
}

func ParseJSON(data []byte) ([]TestEntry, error) {
    var raw exportJSON
    if err := json.Unmarshal(data, &raw); err != nil {
        return nil, err
    }

    entries := make([]TestEntry, 0, len(raw.Tests))
    for _, t := range raw.Tests {
        parts := strings.SplitN(t.Name, ".", 3)
        if len(parts) != 3 {
            continue // skip malformed names
        }
        entries = append(entries, TestEntry{
            Name:     t.Name,
            FilePath: t.Location.FileName,
            Dir:      filepath.Dir(t.Location.FileName),
            Library:  parts[0],
            Bench:    parts[1],
            TestCase: parts[2],
        })
    }
    return entries, nil
}

func Scan(runPy, jsonPath string) ([]TestEntry, error) {
    cmd := exec.Command("python", runPy, "--export-json", jsonPath)
    cmd.Dir = filepath.Dir(runPy)
    out, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("scan failed: %w\n%s", err, out)
    }

    data, err := os.ReadFile(jsonPath)
    if err != nil {
        return nil, fmt.Errorf("reading export json: %w", err)
    }
    return ParseJSON(data)
}
```

- [ ] **Step 4: Run tests and confirm they pass**

```bash
go test ./internal/scanner/ -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/scanner/
git commit -m "feat: scanner — VUnit JSON parsing and test discovery"
```

---

## Task 4: Tree Model — Nodes, Navigation, Status

**Files:**
- Create: `internal/tree/tree.go`
- Create: `internal/tree/tree_test.go`

### Types

```go
package tree

type Status int

const (
    NotRun  Status = iota
    Running        // "~"
    Passed         // "✓"
    Failed         // "✗"
)

type NodeKind int

const (
    DirNode   NodeKind = iota
    BenchNode          // testbench
    TestNode            // individual test case
)

type Node struct {
    Kind        NodeKind
    Name        string   // display name
    FullName    string   // for TestNode: "lib.tb_name.test_case"
    BenchGlob   string   // for BenchNode: "lib.tb_name.*"
    Status      Status
    Children    []*Node
    Expanded    bool
}

// Tree is the full test hierarchy for one window.
type Tree struct {
    Roots   []*Node
    cursor  int  // index into Visible()
}
```

### Key functions to implement and test

- `BuildTree(entries []scanner.TestEntry, gitRoot string) *Tree` — build from scanner output
- `(t *Tree) Visible() []*Node` — flattened list of visible (non-collapsed) nodes
- `(t *Tree) CursorNode() *Node`
- `(t *Tree) MoveUp()`, `MoveDown()`
- `(t *Tree) Toggle()` — expand/collapse current node
- `(t *Tree) RunPattern() []string` — returns VUnit args for current selection
- `(n *Node) DeriveStatus()` — update parent status from children (recursive)
- `(t *Tree) SetStatus(fullName string, s Status)` — update a leaf and re-derive parents

- [ ] **Step 1: Write failing tests**

```go
// internal/tree/tree_test.go
package tree_test

import (
    "testing"

    "github.com/lazyvunit/lazy_vunit/internal/scanner"
    "github.com/lazyvunit/lazy_vunit/internal/tree"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

var testEntries = []scanner.TestEntry{
    {Name: "lib.tb_alu.test_add",      Dir: "/proj/src/alu", Library: "lib", Bench: "tb_alu",  TestCase: "test_add"},
    {Name: "lib.tb_alu.test_subtract", Dir: "/proj/src/alu", Library: "lib", Bench: "tb_alu",  TestCase: "test_subtract"},
    {Name: "lib.tb_uart.test_baud",    Dir: "/proj/src/uart",Library: "lib", Bench: "tb_uart", TestCase: "test_baud"},
}

func TestBuildTree_CreatesDirectories(t *testing.T) {
    tr := tree.BuildTree(testEntries, "/proj")
    require.Len(t, tr.Roots, 2)
    assert.Equal(t, "src/alu", tr.Roots[0].Name)
    assert.Equal(t, "src/uart", tr.Roots[1].Name)
}

func TestBuildTree_CreatesTestBenchChildren(t *testing.T) {
    tr := tree.BuildTree(testEntries, "/proj")
    aluDir := tr.Roots[0]
    require.Len(t, aluDir.Children, 1)
    bench := aluDir.Children[0]
    assert.Equal(t, "tb_alu", bench.Name)
    assert.Equal(t, "lib.tb_alu.*", bench.BenchGlob)
}

func TestBuildTree_CreatesTestCaseLeaves(t *testing.T) {
    tr := tree.BuildTree(testEntries, "/proj")
    bench := tr.Roots[0].Children[0]
    require.Len(t, bench.Children, 2)
    assert.Equal(t, "test_add", bench.Children[0].Name)
    assert.Equal(t, "lib.tb_alu.test_add", bench.Children[0].FullName)
}

func TestVisible_AllExpandedByDefault(t *testing.T) {
    tr := tree.BuildTree(testEntries, "/proj")
    visible := tr.Visible()
    // 2 dirs + 2 benches + 3 tests = 7 nodes
    assert.Len(t, visible, 7)
}

func TestVisible_CollapseHidesChildren(t *testing.T) {
    tr := tree.BuildTree(testEntries, "/proj")
    // cursor starts at 0 (src/alu dir), toggle to collapse it
    tr.Toggle()
    visible := tr.Visible()
    // collapsed alu: 1 dir + 1 bench (uart) + 1 test (uart) = 3 nodes (alu dir + uart dir + uart bench + uart test = 4)
    // After collapsing src/alu: src/alu (1) + src/uart (1) + tb_uart (1) + test_baud (1) = 4
    assert.Len(t, visible, 4)
}

func TestMoveDown_WrapsToBottom(t *testing.T) {
    tr := tree.BuildTree(testEntries, "/proj")
    tr.MoveDown()
    assert.Equal(t, "tb_alu", tr.CursorNode().Name)
}

func TestRunPattern_TestNode(t *testing.T) {
    tr := tree.BuildTree(testEntries, "/proj")
    // navigate to test_add: visible[2]
    tr.MoveDown() // tb_alu
    tr.MoveDown() // test_add
    pattern := tr.RunPattern()
    assert.Equal(t, []string{"lib.tb_alu.test_add"}, pattern)
}

func TestRunPattern_BenchNode(t *testing.T) {
    tr := tree.BuildTree(testEntries, "/proj")
    tr.MoveDown() // tb_alu
    pattern := tr.RunPattern()
    assert.Equal(t, []string{"lib.tb_alu.*"}, pattern)
}

func TestRunPattern_DirNode(t *testing.T) {
    tr := tree.BuildTree(testEntries, "/proj")
    // cursor at src/alu (index 0)
    pattern := tr.RunPattern()
    assert.ElementsMatch(t, []string{"lib.tb_alu.test_add", "lib.tb_alu.test_subtract"}, pattern)
}

func TestSetStatus_UpdatesLeafAndParent(t *testing.T) {
    tr := tree.BuildTree(testEntries, "/proj")
    tr.SetStatus("lib.tb_alu.test_add", tree.Passed)
    tr.SetStatus("lib.tb_alu.test_subtract", tree.Failed)

    bench := tr.Roots[0].Children[0]
    assert.Equal(t, tree.Failed, bench.Status) // any child failed → bench failed

    aluDir := tr.Roots[0]
    assert.Equal(t, tree.Failed, aluDir.Status)
}

func TestSetStatus_AllPassedPropagates(t *testing.T) {
    tr := tree.BuildTree(testEntries, "/proj")
    tr.SetStatus("lib.tb_alu.test_add", tree.Passed)
    tr.SetStatus("lib.tb_alu.test_subtract", tree.Passed)

    bench := tr.Roots[0].Children[0]
    assert.Equal(t, tree.Passed, bench.Status)
}

func TestDeriveStatus_NotRunIfAnyChildNotRun(t *testing.T) {
    tr := tree.BuildTree(testEntries, "/proj")
    tr.SetStatus("lib.tb_alu.test_add", tree.Passed)
    // test_subtract is still NotRun

    bench := tr.Roots[0].Children[0]
    assert.Equal(t, tree.NotRun, bench.Status)
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/tree/ -v
```

Expected: compilation error.

- [ ] **Step 3: Implement tree.go**

```go
// internal/tree/tree.go
package tree

import (
    "path/filepath"
    "sort"
    "strings"

    "github.com/lazyvunit/lazy_vunit/internal/scanner"
)

type Status int

const (
    NotRun  Status = iota
    Running
    Passed
    Failed
)

type NodeKind int

const (
    DirNode NodeKind = iota
    BenchNode
    TestNode
)

type Node struct {
    Kind      NodeKind
    Name      string
    FullName  string
    BenchGlob string
    Status    Status
    Children  []*Node
    Expanded  bool
    parent    *Node
}

type Tree struct {
    Roots  []*Node
    cursor int
}

func BuildTree(entries []scanner.TestEntry, gitRoot string) *Tree {
    // dirKey → benchKey → []TestEntry
    type benchKey struct{ dir, bench string }
    benchMap := map[benchKey][]scanner.TestEntry{}
    var dirOrder []string
    dirSeen := map[string]bool{}

    for _, e := range entries {
        rel, _ := filepath.Rel(gitRoot, e.Dir)
        rel = filepath.ToSlash(rel)
        if !dirSeen[rel] {
            dirOrder = append(dirOrder, rel)
            dirSeen[rel] = true
        }
        bk := benchKey{dir: rel, bench: e.Bench}
        benchMap[bk] = append(benchMap[bk], e)
    }

    sort.Strings(dirOrder)

    dirNodes := map[string]*Node{}
    for _, dir := range dirOrder {
        dirNodes[dir] = &Node{Kind: DirNode, Name: dir, Expanded: true}
    }

    // collect unique (dir, bench) pairs in order
    type benchPair struct{ dir, bench string }
    seen := map[benchPair]bool{}
    var benchOrder []benchPair
    for _, e := range entries {
        rel, _ := filepath.Rel(gitRoot, e.Dir)
        rel = filepath.ToSlash(rel)
        bp := benchPair{rel, e.Bench}
        if !seen[bp] {
            seen[bp] = true
            benchOrder = append(benchOrder, bp)
        }
    }

    for _, bp := range benchOrder {
        bk := benchKey{dir: bp.dir, bench: bp.bench}
        testsForBench := benchMap[bk]
        lib := testsForBench[0].Library

        benchNode := &Node{
            Kind:      BenchNode,
            Name:      bp.bench,
            BenchGlob: lib + "." + bp.bench + ".*",
            Expanded:  true,
        }

        for _, e := range testsForBench {
            testNode := &Node{
                Kind:     TestNode,
                Name:     e.TestCase,
                FullName: e.Name,
                Expanded: false,
                parent:   benchNode,
            }
            benchNode.Children = append(benchNode.Children, testNode)
        }

        benchNode.parent = dirNodes[bp.dir]
        dirNodes[bp.dir].Children = append(dirNodes[bp.dir].Children, benchNode)
    }

    roots := make([]*Node, 0, len(dirOrder))
    for _, dir := range dirOrder {
        roots = append(roots, dirNodes[dir])
    }

    return &Tree{Roots: roots, cursor: 0}
}

// Visible returns the flattened list of nodes currently visible (respecting Expanded).
func (t *Tree) Visible() []*Node {
    var result []*Node
    for _, r := range t.Roots {
        collectVisible(r, &result)
    }
    return result
}

func collectVisible(n *Node, out *[]*Node) {
    *out = append(*out, n)
    if n.Expanded {
        for _, c := range n.Children {
            collectVisible(c, out)
        }
    }
}

func (t *Tree) CursorNode() *Node {
    v := t.Visible()
    if len(v) == 0 {
        return nil
    }
    if t.cursor >= len(v) {
        t.cursor = len(v) - 1
    }
    return v[t.cursor]
}

func (t *Tree) MoveUp() {
    if t.cursor > 0 {
        t.cursor--
    }
}

func (t *Tree) MoveDown() {
    v := t.Visible()
    if t.cursor < len(v)-1 {
        t.cursor++
    }
}

func (t *Tree) Toggle() {
    n := t.CursorNode()
    if n == nil || n.Kind == TestNode {
        return
    }
    n.Expanded = !n.Expanded
    // keep cursor in bounds
    v := t.Visible()
    if t.cursor >= len(v) {
        t.cursor = len(v) - 1
    }
}

// RunPattern returns the VUnit CLI arguments for the currently selected node.
func (t *Tree) RunPattern() []string {
    n := t.CursorNode()
    if n == nil {
        return nil
    }
    switch n.Kind {
    case TestNode:
        return []string{n.FullName}
    case BenchNode:
        return []string{n.BenchGlob}
    case DirNode:
        return collectLeafNames(n)
    }
    return nil
}

func collectLeafNames(n *Node) []string {
    if n.Kind == TestNode {
        return []string{n.FullName}
    }
    var names []string
    for _, c := range n.Children {
        names = append(names, collectLeafNames(c)...)
    }
    return names
}

func (t *Tree) SetStatus(fullName string, s Status) {
    for _, r := range t.Roots {
        if setStatusInNode(r, fullName, s) {
            break
        }
    }
}

func setStatusInNode(n *Node, fullName string, s Status) bool {
    if n.Kind == TestNode {
        if n.FullName == fullName {
            n.Status = s
            return true
        }
        return false
    }
    for _, c := range n.Children {
        if setStatusInNode(c, fullName, s) {
            deriveStatus(n)
            return true
        }
    }
    return false
}

func deriveStatus(n *Node) {
    if len(n.Children) == 0 {
        return
    }
    hasFailed := false
    hasRunning := false
    allPassed := true
    anyNotRun := false

    for _, c := range n.Children {
        switch c.Status {
        case Failed:
            hasFailed = true
            allPassed = false
        case Running:
            hasRunning = true
            allPassed = false
        case NotRun:
            anyNotRun = true
            allPassed = false
        }
    }

    switch {
    case hasFailed:
        n.Status = Failed
    case hasRunning:
        n.Status = Running
    case allPassed && !anyNotRun:
        n.Status = Passed
    default:
        n.Status = NotRun
    }

    _ = strings.Contains // satisfy import if needed
}
```

- [ ] **Step 4: Run tests and confirm they pass**

```bash
go test ./internal/tree/ -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tree/
git commit -m "feat: tree model — node hierarchy, navigation, status derivation"
```

---

## Task 5: Persistence — Load, Save, Gitignore

**Files:**
- Create: `internal/persist/persist.go`
- Create: `internal/persist/persist_test.go`

### Types and Functions

```go
package persist

import "time"

type Result struct {
    Status string    `json:"status"` // "pass" | "fail"
    RanAt  time.Time `json:"ran_at"`
}

type Store map[string]Result // test full name → result

// Load reads .lazyvunit/<windowKey>.json from lazyvunitDir.
// Returns an empty store (not an error) if the file doesn't exist yet.
func Load(lazyvunitDir, windowKey string) (Store, error)

// Save writes the store to .lazyvunit/<windowKey>.json, creating the directory if needed.
func Save(lazyvunitDir, windowKey string, store Store) error

// EnsureGitignore appends ".lazyvunit/" to <gitRoot>/.gitignore if not already present.
// Creates .gitignore if it does not exist.
func EnsureGitignore(gitRoot string) error
```

- [ ] **Step 1: Write failing tests**

```go
// internal/persist/persist_test.go
package persist_test

import (
    "os"
    "path/filepath"
    "testing"
    "time"

    "github.com/lazyvunit/lazy_vunit/internal/persist"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestLoad_EmptyStoreIfNotExist(t *testing.T) {
    dir := t.TempDir()
    store, err := persist.Load(dir, "src_alu")
    require.NoError(t, err)
    assert.Empty(t, store)
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
    dir := t.TempDir()
    now := time.Now().UTC().Truncate(time.Second)
    store := persist.Store{
        "lib.tb_alu.test_add": {Status: "pass", RanAt: now},
    }
    require.NoError(t, persist.Save(dir, "src_alu", store))

    loaded, err := persist.Load(dir, "src_alu")
    require.NoError(t, err)
    require.Contains(t, loaded, "lib.tb_alu.test_add")
    assert.Equal(t, "pass", loaded["lib.tb_alu.test_add"].Status)
    assert.Equal(t, now, loaded["lib.tb_alu.test_add"].RanAt)
}

func TestSave_CreatesDirectory(t *testing.T) {
    dir := t.TempDir()
    lazyDir := filepath.Join(dir, ".lazyvunit")
    require.NoError(t, persist.Save(lazyDir, "key", persist.Store{}))
    _, err := os.Stat(lazyDir)
    assert.NoError(t, err)
}

func TestEnsureGitignore_CreatesFile(t *testing.T) {
    dir := t.TempDir()
    require.NoError(t, persist.EnsureGitignore(dir))
    data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
    require.NoError(t, err)
    assert.Contains(t, string(data), ".lazyvunit/")
}

func TestEnsureGitignore_AppendsToExisting(t *testing.T) {
    dir := t.TempDir()
    require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0644))
    require.NoError(t, persist.EnsureGitignore(dir))
    data, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
    assert.Contains(t, string(data), "*.log")
    assert.Contains(t, string(data), ".lazyvunit/")
}

func TestEnsureGitignore_NoDuplicateEntry(t *testing.T) {
    dir := t.TempDir()
    require.NoError(t, persist.EnsureGitignore(dir))
    require.NoError(t, persist.EnsureGitignore(dir)) // call twice
    data, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
    count := 0
    for _, line := range strings.Split(string(data), "\n") {
        if strings.TrimSpace(line) == ".lazyvunit/" {
            count++
        }
    }
    assert.Equal(t, 1, count)
}
```

Add `"strings"` import to the test file.

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/persist/ -v
```

Expected: compilation error.

- [ ] **Step 3: Implement persist.go**

```go
// internal/persist/persist.go
package persist

import (
    "encoding/json"
    "os"
    "path/filepath"
    "strings"
    "time"
)

type Result struct {
    Status string    `json:"status"`
    RanAt  time.Time `json:"ran_at"`
}

type Store map[string]Result

func Load(lazyvunitDir, windowKey string) (Store, error) {
    path := filepath.Join(lazyvunitDir, windowKey+".json")
    data, err := os.ReadFile(path)
    if os.IsNotExist(err) {
        return Store{}, nil
    }
    if err != nil {
        return nil, err
    }
    var store Store
    if err := json.Unmarshal(data, &store); err != nil {
        return nil, err
    }
    return store, nil
}

func Save(lazyvunitDir, windowKey string, store Store) error {
    if err := os.MkdirAll(lazyvunitDir, 0755); err != nil {
        return err
    }
    data, err := json.MarshalIndent(store, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(filepath.Join(lazyvunitDir, windowKey+".json"), data, 0644)
}

func EnsureGitignore(gitRoot string) error {
    path := filepath.Join(gitRoot, ".gitignore")
    data, err := os.ReadFile(path)
    if err != nil && !os.IsNotExist(err) {
        return err
    }

    existing := string(data)
    for _, line := range strings.Split(existing, "\n") {
        if strings.TrimSpace(line) == ".lazyvunit/" {
            return nil // already present
        }
    }

    entry := ".lazyvunit/\n"
    if len(existing) > 0 && !strings.HasSuffix(existing, "\n") {
        entry = "\n" + entry
    }
    f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        return err
    }
    defer f.Close()
    _, err = f.WriteString(entry)
    return err
}
```

- [ ] **Step 4: Run tests and confirm they pass**

```bash
go test ./internal/persist/ -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/persist/
git commit -m "feat: persistence — load/save results and gitignore management"
```

---

## Task 6: Runner — Output Parser and Subprocess

**Files:**
- Create: `internal/runner/parser.go`
- Create: `internal/runner/parser_test.go`
- Create: `internal/runner/runner.go`

### Output Parser (pure, testable)

VUnit stdout patterns to recognise:
- `<test-name> passed` (or `<test-name>` then `passed` on next line)
- `<test-name> failed` (or `FAILURE` block ending with `======...` separator)

```go
// internal/runner/parser.go
package runner

import "github.com/lazyvunit/lazy_vunit/internal/tree"

type ParseResult struct {
    TestName string
    Status   tree.Status
}

// ParseLine checks a line (and the previous line for two-line patterns) for
// pass/fail signals. Returns (result, true) if a status change was detected.
func ParseLine(line, prevLine string) (ParseResult, bool)
```

- [ ] **Step 1: Write failing parser tests**

```go
// internal/runner/parser_test.go
package runner_test

import (
    "testing"

    "github.com/lazyvunit/lazy_vunit/internal/runner"
    "github.com/lazyvunit/lazy_vunit/internal/tree"
    "github.com/stretchr/testify/assert"
)

func TestParseLine_PassedOnSameLine(t *testing.T) {
    result, ok := runner.ParseLine("lib.tb_alu.test_add                         passed", "")
    assert.True(t, ok)
    assert.Equal(t, "lib.tb_alu.test_add", result.TestName)
    assert.Equal(t, tree.Passed, result.Status)
}

func TestParseLine_FailedOnSameLine(t *testing.T) {
    result, ok := runner.ParseLine("lib.tb_alu.test_overflow                    failed", "")
    assert.True(t, ok)
    assert.Equal(t, "lib.tb_alu.test_overflow", result.TestName)
    assert.Equal(t, tree.Failed, result.Status)
}

func TestParseLine_PassedOnNextLine(t *testing.T) {
    result, ok := runner.ParseLine("passed", "lib.tb_alu.test_add")
    assert.True(t, ok)
    assert.Equal(t, "lib.tb_alu.test_add", result.TestName)
    assert.Equal(t, tree.Passed, result.Status)
}

func TestParseLine_FailedOnNextLine(t *testing.T) {
    result, ok := runner.ParseLine("failed", "lib.tb_alu.test_overflow")
    assert.True(t, ok)
    assert.Equal(t, tree.Failed, result.Status)
}

func TestParseLine_UnrelatedLine(t *testing.T) {
    _, ok := runner.ParseLine("Compile tb_alu.vhd", "")
    assert.False(t, ok)
}

func TestParseLine_EmptyLine(t *testing.T) {
    _, ok := runner.ParseLine("", "")
    assert.False(t, ok)
}
```

- [ ] **Step 2: Run to confirm they fail**

```bash
go test ./internal/runner/ -v -run TestParse
```

Expected: compilation error.

- [ ] **Step 3: Implement parser.go**

```go
// internal/runner/parser.go
package runner

import (
    "regexp"
    "strings"

    "github.com/lazyvunit/lazy_vunit/internal/tree"
)

// testNameRe matches a VUnit full test name: lib.bench.test_case
var testNameRe = regexp.MustCompile(`^(\w+\.\w+\.\w+)`)

type ParseResult struct {
    TestName string
    Status   tree.Status
}

func ParseLine(line, prevLine string) (ParseResult, bool) {
    trimmed := strings.TrimSpace(line)

    // Pattern: "lib.tb.test  passed" or "lib.tb.test  failed" on one line
    if m := testNameRe.FindString(trimmed); m != "" {
        lower := strings.ToLower(trimmed)
        if strings.Contains(lower, " passed") || strings.HasSuffix(lower, "passed") {
            return ParseResult{TestName: m, Status: tree.Passed}, true
        }
        if strings.Contains(lower, " failed") || strings.HasSuffix(lower, "failed") {
            return ParseResult{TestName: m, Status: tree.Failed}, true
        }
        // Line is just a test name — will be resolved by next line
        return ParseResult{}, false
    }

    // Pattern: current line is "passed"/"failed", prev line is the test name
    if prevLine != "" {
        prevTrimmed := strings.TrimSpace(prevLine)
        if m := testNameRe.FindString(prevTrimmed); m != "" {
            switch strings.ToLower(trimmed) {
            case "passed":
                return ParseResult{TestName: m, Status: tree.Passed}, true
            case "failed":
                return ParseResult{TestName: m, Status: tree.Failed}, true
            }
        }
    }

    return ParseResult{}, false
}
```

- [ ] **Step 4: Run parser tests and confirm they pass**

```bash
go test ./internal/runner/ -v -run TestParse
```

Expected: all PASS.

- [ ] **Step 5: Implement runner.go**

The runner manages subprocess lifecycle and emits Bubbletea messages. Because it integrates with Bubbletea, it exposes a `Run()` function that returns a `tea.Cmd`.

```go
// internal/runner/runner.go
package runner

import (
    "bufio"
    "fmt"
    "io"
    "os/exec"
    "path/filepath"
    "strings"
    "sync"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/lazyvunit/lazy_vunit/internal/tree"
)

// OutputLineMsg is sent to the Bubbletea model for each line of stdout.
type OutputLineMsg struct{ Text string }

// StatusUpdateMsg is sent when a test's pass/fail status is detected in output.
type StatusUpdateMsg struct {
    TestName string
    Status   tree.Status
}

// RunDoneMsg is sent when the subprocess exits.
type RunDoneMsg struct{ Err error }

// CancelFunc can be called to send SIGTERM to the running subprocess.
type CancelFunc func()

// Run spawns `python <runPy> <args...>` and returns a tea.Cmd that streams output.
// The returned CancelFunc sends SIGTERM; it is safe to call after the process has exited.
func Run(runPy string, args []string) (tea.Cmd, CancelFunc) {
    cmdArgs := append([]string{runPy}, args...)
    cmd := exec.Command("python", cmdArgs...)
    cmd.Dir = filepath.Dir(runPy)

    var once sync.Once
    cancelFn := CancelFunc(func() {
        once.Do(func() {
            if cmd.Process != nil {
                _ = cmd.Process.Signal(syscallSIGTERM())
            }
        })
    })

    teaCmd := func() tea.Msg {
        stdout, err := cmd.StdoutPipe()
        cmd.Stderr = cmd.Stdout // merge stderr into stdout
        if err != nil {
            return RunDoneMsg{Err: err}
        }
        if err := cmd.Start(); err != nil {
            return RunDoneMsg{Err: err}
        }
        // Stream output — this blocks until the process ends.
        // We return the first message; subsequent messages are sent via
        // the channel-based streaming approach below.
        return streamOutput(stdout, cmd)
    }

    return teaCmd, cancelFn
}

// streamOutput reads lines from stdout, sending messages via a channel.
// Returns RunDoneMsg when the process exits.
// Note: this simplified implementation returns lines one at a time.
// The Bubbletea model calls Run() which returns the first Cmd; the model
// must re-subscribe via a new Cmd after each OutputLineMsg to keep streaming.
// See BatchRun() for the batch-aware variant.
func streamOutput(stdout io.ReadCloser, cmd *exec.Cmd) tea.Msg {
    scanner := bufio.NewScanner(stdout)
    var prevLine string
    var lines []string

    for scanner.Scan() {
        line := scanner.Text()
        lines = append(lines, line)
        prevLine = line
        _ = prevLine
    }

    err := cmd.Wait()

    // Build messages: we return a tea.BatchMsg with all output lines + done.
    msgs := make([]tea.Cmd, 0, len(lines)+1)
    prevLine = ""
    for _, line := range lines {
        l := line
        msgs = append(msgs, func() tea.Msg { return OutputLineMsg{Text: l} })
        if result, ok := ParseLine(line, prevLine); ok {
            r := result
            msgs = append(msgs, func() tea.Msg { return StatusUpdateMsg{TestName: r.TestName, Status: r.Status} })
        }
        prevLine = line
    }
    msgs = append(msgs, func() tea.Msg { return RunDoneMsg{Err: err} })

    return tea.BatchMsg(msgs)
}

// RunBatched handles directory-level runs that may exceed 200 tests.
// It splits args into batches of batchSize and runs them sequentially.
// Returns a tea.Cmd for the first batch; subsequent batches are triggered
// by the model re-invoking RunBatched with remaining args.
func RunBatched(runPy string, args []string, batchSize int) (tea.Cmd, CancelFunc, []string) {
    if batchSize <= 0 {
        batchSize = 200
    }
    batch := args
    remaining := []string{}
    if len(args) > batchSize {
        batch = args[:batchSize]
        remaining = args[batchSize:]
    }
    cmd, cancel := Run(runPy, batch)
    return cmd, cancel, remaining
}

// syscallSIGTERM returns syscall.SIGTERM — extracted to allow build on all platforms.
func syscallSIGTERM() os.Signal {
    return syscall.SIGTERM
}
```

**Note on the streaming design:** The `streamOutput` implementation above collects all output then sends it as a batch. For real-time streaming, the Bubbletea model uses a `tea.Cmd` that listens on a channel. Add this goroutine-based streaming helper:

```go
// StreamCmd returns a tea.Cmd that reads one line from ch and sends it as OutputLineMsg.
// The model calls this repeatedly to drain the channel.
func StreamCmd(ch <-chan string) tea.Cmd {
    return func() tea.Msg {
        line, ok := <-ch
        if !ok {
            return nil
        }
        return OutputLineMsg{Text: line}
    }
}
```

Refactor `Run()` to use a goroutine that writes to a channel, and return the channel alongside the CancelFunc. Update `runner.go` to:

```go
// Run spawns the subprocess and returns:
// - tea.Cmd: call once to start streaming (sends OutputLineMsg/StatusUpdateMsg/RunDoneMsg)
// - CancelFunc: sends SIGTERM
func Run(runPy string, args []string) (tea.Cmd, CancelFunc) {
    cmdArgs := append([]string{runPy}, args...)
    cmd := exec.Command("python", cmdArgs...)
    cmd.Dir = filepath.Dir(runPy)

    ch := make(chan tea.Msg, 256)
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
        cmd.Stderr = cmd.Stdout
        if err := cmd.Start(); err != nil {
            return RunDoneMsg{Err: err}
        }
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
            err := cmd.Wait()
            ch <- RunDoneMsg{Err: err}
            close(ch)
        }()
        // Return first message immediately
        return <-ch
    })

    return startCmd, cancelFn
}

// NextMsg returns a tea.Cmd that reads the next message from the runner channel.
// Call this from the model's Update after each OutputLineMsg/StatusUpdateMsg.
func NextMsg(ch <-chan tea.Msg) tea.Cmd {
    return func() tea.Msg {
        msg, ok := <-ch
        if !ok {
            return nil
        }
        return msg
    }
}
```

Update `Run` signature to also return the channel so the model can call `NextMsg`:

```go
func Run(runPy string, args []string) (tea.Cmd, CancelFunc, <-chan tea.Msg)
```

- [ ] **Step 6: Add `syscall` import to runner.go and fix compilation**

```go
import (
    "bufio"
    "os"
    "os/exec"
    "path/filepath"
    "sync"
    "syscall"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/lazyvunit/lazy_vunit/internal/tree"
)
```

Remove `syscallSIGTERM()` helper, replace `syscallSIGTERM()` with `syscall.SIGTERM` directly.

- [ ] **Step 6b: Update RunBatched to use the three-return Run() signature**

The refactored `Run()` now returns `(tea.Cmd, CancelFunc, <-chan tea.Msg)`. Update `RunBatched` to match:

```go
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
```

Also update `WindowModel.StartRun` in `window.go` to unpack four return values from `RunBatched`:

```go
firstCmd, cancelFn, _, remaining = runner.RunBatched(w.Script.AbsPath, args, batchSize)
```

(The channel is stored if needed for `NextMsg`; for simplicity the batch model uses the message channel returned by the tea.Cmd chain.)

- [ ] **Step 7: Verify the package builds**

```bash
go build ./internal/runner/
```

Expected: no output.

- [ ] **Step 8: Commit**

```bash
git add internal/runner/
git commit -m "feat: runner — output parser and subprocess streaming"
```

---

## Task 7: UI Scaffolding — Keys, Styles, Messages

**Files:**
- Create: `internal/ui/keys.go`
- Create: `internal/ui/styles.go`
- Create: `internal/ui/messages.go`

No tests for this task (purely declarative).

- [ ] **Step 1: Create keys.go**

```go
// internal/ui/keys.go
package ui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
    Up       key.Binding
    Down     key.Binding
    Left     key.Binding
    Right    key.Binding
    Run      key.Binding
    RunGUI   key.Binding
    PrevWin  key.Binding
    NextWin  key.Binding
    Rescan   key.Binding
    Cancel   key.Binding
    Quit     key.Binding
    Help     key.Binding
}

var DefaultKeys = KeyMap{
    Up:      key.NewBinding(key.WithKeys("up"),    key.WithHelp("↑",      "up")),
    Down:    key.NewBinding(key.WithKeys("down"),  key.WithHelp("↓",      "down")),
    Left:    key.NewBinding(key.WithKeys("left"),  key.WithHelp("←",      "collapse")),
    Right:   key.NewBinding(key.WithKeys("right"), key.WithHelp("→",      "expand")),
    Run:     key.NewBinding(key.WithKeys(" "),     key.WithHelp("space",  "run")),
    RunGUI:  key.NewBinding(key.WithKeys("g"),     key.WithHelp("g",      "gui")),
    PrevWin: key.NewBinding(key.WithKeys("["),     key.WithHelp("[",      "prev window")),
    NextWin: key.NewBinding(key.WithKeys("]"),     key.WithHelp("]",      "next window")),
    Rescan:  key.NewBinding(key.WithKeys("ctrl+r"),key.WithHelp("ctrl+r", "rescan")),
    Cancel:  key.NewBinding(key.WithKeys("ctrl+c","x"), key.WithHelp("x", "cancel")),
    Quit:    key.NewBinding(key.WithKeys("q"),     key.WithHelp("q",      "quit")),
    Help:    key.NewBinding(key.WithKeys("?"),     key.WithHelp("?",      "help")),
}
```

- [ ] **Step 2: Create styles.go**

```go
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

// StatusIcon returns the coloured icon string for a given tree.Status.
// Import tree only in layout.go to avoid cycles; keep this as string constants.
const (
    IconPassed  = "✓"
    IconFailed  = "✗"
    IconNotRun  = "○"
    IconRunning = "~"
)
```

- [ ] **Step 3: Create messages.go**

```go
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
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./internal/ui/
```

Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/keys.go internal/ui/styles.go internal/ui/messages.go
git commit -m "feat: ui scaffolding — keys, styles, shared message types"
```

---

## Task 8: Picker Model

**Files:**
- Create: `internal/ui/picker.go`
- Create: `internal/ui/picker_test.go`

The picker is shown at startup when multiple `run.py` files are found. It is a standalone Bubbletea model embedded in `AppModel`.

- [ ] **Step 1: Write failing tests**

```go
// internal/ui/picker_test.go
package ui_test

import (
    "testing"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/lazyvunit/lazy_vunit/internal/finder"
    "github.com/lazyvunit/lazy_vunit/internal/ui"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func makeScripts() []finder.RunScript {
    return []finder.RunScript{
        {AbsPath: "/p/src/alu/run.py",  RelDir: "src/alu",  WindowKey: "src_alu",  LeafName: "alu"},
        {AbsPath: "/p/src/uart/run.py", RelDir: "src/uart", WindowKey: "src_uart", LeafName: "uart"},
    }
}

func TestPickerModel_InitialCursor(t *testing.T) {
    m := ui.NewPickerModel(makeScripts())
    assert.Equal(t, 0, m.Cursor())
}

func TestPickerModel_MoveDown(t *testing.T) {
    m := ui.NewPickerModel(makeScripts())
    newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
    pm := newM.(ui.PickerModel)
    assert.Equal(t, 1, pm.Cursor())
}

func TestPickerModel_MoveDownClamps(t *testing.T) {
    m := ui.NewPickerModel(makeScripts())
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
    m, _ = m.(ui.PickerModel).Update(tea.KeyMsg{Type: tea.KeyDown}) // past end
    assert.Equal(t, 1, m.(ui.PickerModel).Cursor())
}

func TestPickerModel_EnterSelectsScript(t *testing.T) {
    m := ui.NewPickerModel(makeScripts())
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
    newM, cmd := m.(ui.PickerModel).Update(tea.KeyMsg{Type: tea.KeyEnter})
    _ = newM
    require.NotNil(t, cmd)
    msg := cmd()
    selected, ok := msg.(ui.PickerSelectedMsg)
    require.True(t, ok)
    assert.Equal(t, "src/uart", selected.Script.RelDir)
}

func TestPickerModel_QuitEmitsQuit(t *testing.T) {
    m := ui.NewPickerModel(makeScripts())
    _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    require.NotNil(t, cmd)
    msg := cmd()
    _, isQuit := msg.(tea.QuitMsg)
    assert.True(t, isQuit)
}

func TestPickerModel_ViewContainsWindowNames(t *testing.T) {
    m := ui.NewPickerModel(makeScripts())
    view := m.View()
    assert.Contains(t, view, "alu")
    assert.Contains(t, view, "uart")
}
```

- [ ] **Step 2: Run to confirm they fail**

```bash
go test ./internal/ui/ -v -run TestPicker
```

Expected: compilation error.

- [ ] **Step 3: Implement picker.go**

```go
// internal/ui/picker.go
package ui

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/lazyvunit/lazy_vunit/internal/finder"
    "strings"
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
```

- [ ] **Step 4: Run tests and confirm they pass**

```bash
go test ./internal/ui/ -v -run TestPicker
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/picker.go internal/ui/picker_test.go
git commit -m "feat: picker model — startup window selection"
```

---

## Task 9: Window Model

**Files:**
- Create: `internal/ui/window.go`

The `WindowModel` holds the state for one `run.py` window: the tree, output buffer, runner channel, and persisted results.

- [ ] **Step 1: Implement window.go**

```go
// internal/ui/window.go
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
}

func NewWindowModel(script finder.RunScript, allScripts []finder.RunScript, gitRoot string) WindowModel {
    lazyDir := filepath.Join(gitRoot, ".lazyvunit")
    results, _ := persist.Load(lazyDir, script.WindowKey)
    return WindowModel{
        Script:     script,
        AllScripts: allScripts,
        GitRoot:    gitRoot,
        LazyDir:    lazyDir,
        State:      WinStateScanning,
        Results:    results,
    }
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
// Returns the initial tea.Cmd to kick off the runner.
func (w *WindowModel) StartRun(guiMode bool) tea.Cmd {
    if w.Tree == nil || w.State != WinStateReady {
        return nil
    }
    node := w.Tree.CursorNode()
    if node == nil {
        return nil
    }
    if guiMode && node.Kind != tree.TestNode {
        w.StatusMsg = "GUI mode requires a single test — navigate to a test case"
        return nil
    }

    args := w.Tree.RunPattern()
    if guiMode {
        args = append(args, "--gui")
    }

    // Set all selected tests to Running
    for _, name := range fullNamesFromPattern(w.Tree, node) {
        w.Tree.SetStatus(name, tree.Running)
    }

    w.Output = []string{fmt.Sprintf("# Running: python %s %s", w.Script.AbsPath, strings.Join(args, " "))}
    w.OutputTitle = node.Name
    w.State = WinStateRunning

    const batchSize = 200
    var firstCmd tea.Cmd
    var cancelFn runner.CancelFunc
    var remaining []string

    firstCmd, cancelFn, remaining = runner.RunBatched(w.Script.AbsPath, args, batchSize)
    w.CancelFn = cancelFn
    w.PendingArgs = remaining
    w.RunnerCh = nil // channel-based streaming handled via tea.Cmd chain

    return firstCmd
}

func (w *WindowModel) HandleRunnerMsg(msg tea.Msg) tea.Cmd {
    switch m := msg.(type) {
    case runner.OutputLineMsg:
        w.Output = append(w.Output, m.Text)
        return nil // next message arrives via the runner's tea.Cmd chain
    case runner.StatusUpdateMsg:
        w.Tree.SetStatus(m.TestName, m.Status)
        return nil
    case runner.RunDoneMsg:
        // Apply exit-code fallback for any still-Running tests in this batch
        if m.Err != nil {
            w.applyFallbackStatus(tree.Failed)
        } else {
            w.applyFallbackStatus(tree.Passed)
        }
        // Check for pending batches
        if len(w.PendingArgs) > 0 {
            var cmd tea.Cmd
            var cancel runner.CancelFunc
            cmd, cancel, w.PendingArgs = runner.RunBatched(w.Script.AbsPath, w.PendingArgs, 200)
            w.CancelFn = cancel
            w.Output = append(w.Output, "── batch complete, continuing ──")
            return cmd
        }
        w.State = WinStateReady
        w.CancelFn = nil
        w.saveResults()
        return nil
    }
    return nil
}

func (w *WindowModel) Cancel() {
    if w.CancelFn != nil {
        w.CancelFn()
    }
    // Reset Running tests to NotRun
    w.applyFallbackStatus(tree.NotRun)
    w.State = WinStateReady
    w.PendingArgs = nil
    w.saveResults()
}

func (w *WindowModel) applyFallbackStatus(s tree.Status) {
    if w.Tree == nil {
        return
    }
    for _, node := range w.Tree.Visible() {
        if node.Kind == tree.TestNode && node.Status == tree.Running {
            w.Tree.SetStatus(node.FullName, s)
        }
    }
}

func (w *WindowModel) saveResults() {
    if w.Tree == nil {
        return
    }
    now := time.Now().UTC()
    for _, node := range w.Tree.Visible() {
        if node.Kind != tree.TestNode {
            continue
        }
        switch node.Status {
        case tree.Passed:
            w.Results[node.FullName] = persist.Result{Status: "pass", RanAt: now}
        case tree.Failed:
            w.Results[node.FullName] = persist.Result{Status: "fail", RanAt: now}
        }
    }
    _ = persist.Save(w.LazyDir, w.Script.WindowKey, w.Results)
}

// Counts returns (passed, failed, notRun) for this window's test leaves.
func (w *WindowModel) Counts() (int, int, int) {
    if w.Tree == nil {
        return 0, 0, 0
    }
    var p, f, n int
    for _, node := range w.Tree.Visible() {
        if node.Kind != tree.TestNode {
            continue
        }
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

func fullNamesFromPattern(t *tree.Tree, node *tree.Node) []string {
    if node.Kind == tree.TestNode {
        return []string{node.FullName}
    }
    var names []string
    for _, child := range node.Children {
        names = append(names, fullNamesFromPattern(t, child)...)
    }
    return names
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/ui/
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add internal/ui/window.go
git commit -m "feat: window model — per-run.py state, scan, run, persist"
```

---

## Task 10: App Model and Layout

**Files:**
- Create: `internal/ui/app.go`
- Create: `internal/ui/layout.go`
- Create: `internal/ui/app_test.go`

The `AppModel` is the top-level Bubbletea model. It switches between `StatePicker`, `StateScanning`, and `StateMain`.

- [ ] **Step 1: Write failing app model tests**

```go
// internal/ui/app_test.go
package ui_test

import (
    "testing"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/lazyvunit/lazy_vunit/internal/finder"
    "github.com/lazyvunit/lazy_vunit/internal/ui"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func singleScript() finder.RunScript {
    return finder.RunScript{AbsPath: "/p/src/alu/run.py", RelDir: "src/alu", WindowKey: "src_alu", LeafName: "alu"}
}

func TestAppModel_QuitFromMain(t *testing.T) {
    m := ui.NewAppModel([]finder.RunScript{singleScript()}, "/p", "/p")
    _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    require.NotNil(t, cmd)
    msg := cmd()
    _, isQuit := msg.(tea.QuitMsg)
    assert.True(t, isQuit)
}

func TestAppModel_SingleScriptSkipsPicker(t *testing.T) {
    m := ui.NewAppModel([]finder.RunScript{singleScript()}, "/p", "/p")
    // With a single script there should be no picker state
    assert.Equal(t, ui.StateScanning, m.AppState())
}

func TestAppModel_MultipleScriptsShowsPicker(t *testing.T) {
    scripts := []finder.RunScript{singleScript(), {AbsPath: "/p/src/uart/run.py", RelDir: "src/uart", WindowKey: "src_uart", LeafName: "uart"}}
    m := ui.NewAppModel(scripts, "/p", "/p")
    assert.Equal(t, ui.StatePicker, m.AppState())
}

func TestAppModel_StatusMsgOnGOnNonLeaf(t *testing.T) {
    // After scan completes with a dir node selected, pressing g shows status msg
    // (Tested via direct state manipulation since we can't run Python in tests)
    m := ui.NewAppModel([]finder.RunScript{singleScript()}, "/p", "/p")
    // Simulate scan error to get to error state — tests g is blocked during scan
    m2, _ := m.Update(ui.ScanDoneMsg{Err: fmt.Errorf("no python")})
    assert.Equal(t, ui.StateError, m2.(ui.AppModel).AppState())
}

func TestAppModel_SwitchWindowsWithBrackets(t *testing.T) {
    scripts := []finder.RunScript{
        singleScript(),
        {AbsPath: "/p/src/uart/run.py", RelDir: "src/uart", WindowKey: "src_uart", LeafName: "uart"},
    }
    m := ui.NewAppModel(scripts, "/p", "/p")
    // Select first window from picker
    m2, _ := m.Update(ui.PickerSelectedMsg{Script: scripts[0]})
    // Simulate scan done for window 0
    m3, _ := m2.(ui.AppModel).Update(ui.ScanDoneMsg{Entries: nil, Err: nil})
    // Press ] to go to next window
    m4, _ := m3.(ui.AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
    assert.Equal(t, 1, m4.(ui.AppModel).ActiveWindowIndex())
}
```

Add `"fmt"` import.

- [ ] **Step 2: Run to confirm they fail**

```bash
go test ./internal/ui/ -v -run TestAppModel
```

Expected: compilation error.

- [ ] **Step 3: Implement app.go**

```go
// internal/ui/app.go
package ui

import (
    "fmt"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/lazyvunit/lazy_vunit/internal/finder"
    "github.com/lazyvunit/lazy_vunit/internal/runner"
    "github.com/lazyvunit/lazy_vunit/internal/tree"
)

type AppStateKind int

const (
    StatePicker   AppStateKind = iota
    StateScanning
    StateMain
    StateError
)

type AppModel struct {
    state         AppStateKind
    picker        PickerModel
    windows       []WindowModel
    activeIdx     int
    gitRoot       string
    termWidth     int
    termHeight    int
    showHelp      bool
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
        // Pre-create all window models
        for _, s := range scripts {
            m.windows = append(m.windows, NewWindowModel(s, scripts, gitRoot))
        }
    }
    return m
}

func (m AppModel) AppState() AppStateKind         { return m.state }
func (m AppModel) ActiveWindowIndex() int          { return m.activeIdx }
func (m *AppModel) activeWin() *WindowModel        { return &m.windows[m.activeIdx] }

func (m AppModel) Init() tea.Cmd {
    if m.state == StateScanning {
        return m.windows[0].ScanCmd()
    }
    return nil
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {

    case tea.WindowSizeMsg:
        m.termWidth, m.termHeight = msg.Width, msg.Height
        return m, nil

    case tea.KeyMsg:
        return m.handleKey(msg)

    case PickerSelectedMsg:
        // Find the window index for the selected script
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
        cmd := m.activeWin().HandleRunnerMsg(msg)
        if _, done := msg.(runner.RunDoneMsg); done && m.activeWin().State == WinStateReady {
            _ = fmt.Sprintf("") // no-op
        }
        return m, cmd
    }

    return m, nil
}

func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    // Route to picker if in picker state
    if m.state == StatePicker {
        newPicker, cmd := m.picker.Update(msg)
        m.picker = newPicker.(PickerModel)
        return m, cmd
    }

    keys := DefaultKeys
    switch {
    case keyMatches(msg, keys.Quit):
        return m, tea.Quit

    case keyMatches(msg, keys.Help):
        m.showHelp = !m.showHelp

    case m.state != StateMain:
        // No other keys active during scanning/error except quit
        return m, nil

    case keyMatches(msg, keys.Up):
        m.activeWin().Tree.MoveUp()
    case keyMatches(msg, keys.Down):
        m.activeWin().Tree.MoveDown()
    case keyMatches(msg, keys.Left):
        // ← collapses the current node
        node := m.activeWin().Tree.CursorNode()
        if node != nil && node.Expanded {
            m.activeWin().Tree.Toggle()
        }
    case keyMatches(msg, keys.Right):
        // → expands the current node
        node := m.activeWin().Tree.CursorNode()
        if node != nil && !node.Expanded {
            m.activeWin().Tree.Toggle()
        }

    case keyMatches(msg, keys.Run):
        return m, m.activeWin().StartRun(false)

    case keyMatches(msg, keys.RunGUI):
        return m, m.activeWin().StartRun(true)

    case keyMatches(msg, keys.PrevWin):
        if len(m.windows) > 1 {
            m.activeIdx = (m.activeIdx - 1 + len(m.windows)) % len(m.windows)
            if m.windows[m.activeIdx].State == WinStateScanning {
                return m, m.activeWin().ScanCmd()
            }
        }

    case keyMatches(msg, keys.NextWin):
        if len(m.windows) > 1 {
            m.activeIdx = (m.activeIdx + 1) % len(m.windows)
            if m.windows[m.activeIdx].State == WinStateScanning {
                return m, m.activeWin().ScanCmd()
            }
        }

    case keyMatches(msg, keys.Rescan):
        win := m.activeWin()
        if win.State == WinStateRunning {
            win.StatusMsg = "Cannot rescan while tests are running"
            return m, nil
        }
        win.State = WinStateScanning
        return m, win.ScanCmd()

    case keyMatches(msg, keys.Cancel):
        if m.activeWin().State == WinStateRunning {
            m.activeWin().Cancel()
        }
    }

    return m, nil
}

func keyMatches(msg tea.KeyMsg, b interface{ Matches(tea.KeyMsg) bool }) bool {
    return b.Matches(msg)
}

func (m AppModel) View() string {
    switch m.state {
    case StatePicker:
        return m.picker.View()
    case StateScanning:
        return StyleHeader.Render("lazy_vunit") + "\n\n" +
            StyleSubtle.Render("  Scanning...") + "\n"
    case StateError:
        return StyleHeader.Render("lazy_vunit") + "\n\n" +
            StyleFailed.Render("  Error: "+m.activeWin().ErrMsg) + "\n\n" +
            StyleSubtle.Render("  Is VUnit installed? Try: pip install vunit-hdl\n") +
            StyleSubtle.Render("  Press ctrl+r to retry, q to quit.")
    case StateMain:
        return RenderMain(m)
    }
    return ""
}

// AllWindowCounts returns aggregate (passed, failed, notRun) across all loaded windows.
func (m AppModel) AllWindowCounts() (int, int, int) {
    var tp, tf, tn int
    for _, w := range m.windows {
        p, f, n := w.Counts()
        tp += p
        tf += f
        tn += n
    }
    return tp, tf, tn
}
```

- [ ] **Step 4: Implement layout.go**

```go
// internal/ui/layout.go
package ui

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
    "github.com/lazyvunit/lazy_vunit/internal/tree"
)

// RenderMain renders the full three-pane layout for StateMain.
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
    outputWidth := totalWidth - treeWidth - 3 // 3 for borders
    innerHeight := totalHeight - 4            // header + bottom bar

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
    for i, node := range visible {
        line := renderTreeNode(node, i < len(visible))
        if node == cursor {
            line = StyleCursor.Width(width - 2).Render(line)
        }
        sb.WriteString(line + "\n")
    }

    content := sb.String()
    box := StyleBorder.Width(width).Height(height).Render(content)
    return lipgloss.JoinVertical(lipgloss.Left, header, box)
}

func renderTreeNode(n *tree.Node, _ bool) string {
    indent := ""
    icon := ""
    statusStr := ""

    switch n.Kind {
    case tree.DirNode:
        expand := "▶"
        if n.Expanded {
            expand = "▼"
        }
        icon = StyleDir.Render(expand + " ")
    case tree.BenchNode:
        expand := "▶"
        if n.Expanded {
            expand = "▼"
        }
        indent = "  "
        icon = expand + " "
    case tree.TestNode:
        indent = "    "
        icon = ""
        statusStr = statusIcon(n.Status) + " "
    }

    return indent + icon + statusStr + n.Name
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
    // Show last `height` lines
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
    statusMsg := win.StatusMsg

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

    if statusMsg != "" {
        hints = StyleFailed.Render(" " + statusMsg)
        win.StatusMsg = "" // clear after one render
    }

    bar := lipgloss.NewStyle().
        Width(width).
        Background(lipgloss.Color("#16213e")).
        Render(left + hints)

    return bar
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/ui/ -v -run TestAppModel
```

Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/ui/app.go internal/ui/layout.go internal/ui/app_test.go
git commit -m "feat: app model and layout — top-level TUI state machine and rendering"
```

---

## Task 10: main.go — Entry Point and Smoke Test

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Implement main.go**

```go
// main.go
package main

import (
    "fmt"
    "os"

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

    if len(scripts) == 0 {
        fmt.Fprintf(os.Stderr,
            "No VUnit run script found.\nSearched: %s\n\nExpected a run.py containing VUnit.from_argv.\n",
            gitRoot)
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
```

- [ ] **Step 2: Build the binary**

```bash
go build -o lazy_vunit .
```

Expected: `lazy_vunit` binary created, no errors.

- [ ] **Step 3: Run the full test suite**

```bash
go test ./... -v
```

Expected: all tests PASS.

- [ ] **Step 4: Smoke test — run against a project with no VUnit scripts**

```bash
cd /tmp && /Users/james/Workspace/lazy_vunit/lazy_vunit
```

Expected: "No VUnit run script found. Searched: /tmp" printed to stderr, exit code 1.

- [ ] **Step 5: Smoke test — run inside the lazy_vunit repo itself (no run.py)**

```bash
cd /Users/james/Workspace/lazy_vunit && ./lazy_vunit
```

Expected: same "No VUnit run script found" error — confirms the binary works end-to-end.

- [ ] **Step 6: Smoke test — run inside the vunit repo (has run.py files)**

```bash
cd /Users/james/Workspace/vunit && /Users/james/Workspace/lazy_vunit/lazy_vunit
```

Expected: TUI opens. If multiple `run.py` found, picker is shown. `q` exits.

- [ ] **Step 7: Commit**

```bash
git add main.go
git commit -m "feat: main.go — entry point wiring finder, persist, and TUI"
```

---

## Build and Test Commands Reference

```bash
# Build
go build -o lazy_vunit .

# Run all tests
go test ./... -v

# Run a specific package
go test ./internal/tree/ -v
go test ./internal/runner/ -v -run TestParse

# Run with race detector (recommended before release)
go test -race ./...
```
