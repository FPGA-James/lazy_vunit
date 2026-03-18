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
