package tree_test

import (
	"testing"

	"github.com/lazyvunit/lazy_vunit/internal/scanner"
	"github.com/lazyvunit/lazy_vunit/internal/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testEntries = []scanner.TestEntry{
	{Name: "lib.tb_alu.test_add", Dir: "/proj/src/alu", Library: "lib", Bench: "tb_alu", TestCase: "test_add"},
	{Name: "lib.tb_alu.test_subtract", Dir: "/proj/src/alu", Library: "lib", Bench: "tb_alu", TestCase: "test_subtract"},
	{Name: "lib.tb_uart.test_baud", Dir: "/proj/src/uart", Library: "lib", Bench: "tb_uart", TestCase: "test_baud"},
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
	// After collapsing src/alu: src/alu(1) + src/uart(1) + tb_uart(1) + test_baud(1) = 4
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
