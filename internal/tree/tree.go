package tree

import (
	"path/filepath"
	"sort"

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
	v := t.Visible()
	if t.cursor >= len(v) {
		t.cursor = len(v) - 1
	}
}

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

// AllLeaves returns all TestNode leaves in the tree, regardless of expand/collapse state.
func (t *Tree) AllLeaves() []*Node {
	var result []*Node
	for _, r := range t.Roots {
		collectAllLeaves(r, &result)
	}
	return result
}

func collectAllLeaves(n *Node, out *[]*Node) {
	if n.Kind == TestNode {
		*out = append(*out, n)
		return
	}
	for _, c := range n.Children {
		collectAllLeaves(c, out)
	}
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
}
