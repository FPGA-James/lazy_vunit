// internal/ui/window_test.go
package ui_test

import (
	"testing"

	"github.com/lazyvunit/lazy_vunit/internal/finder"
	"github.com/lazyvunit/lazy_vunit/internal/persist"
	"github.com/lazyvunit/lazy_vunit/internal/ui"
	"github.com/stretchr/testify/assert"
)

func makeWindow() ui.WindowModel {
	script := finder.RunScript{
		AbsPath: "/p/src/alu/run.py", RelDir: "src/alu",
		WindowKey: "src_alu", LeafName: "alu",
	}
	return ui.NewWindowModel(script, []finder.RunScript{script}, "/p")
}

func TestWindowModel_ToggleSetting_Verbose(t *testing.T) {
	w := makeWindow()
	assert.False(t, w.Settings.Verbose)
	w.ToggleSetting(1) // index 1 = Verbose
	assert.True(t, w.Settings.Verbose)
	w.ToggleSetting(1)
	assert.False(t, w.Settings.Verbose)
}

func TestWindowModel_ToggleSetting_CompileOnlyClearsElaborate(t *testing.T) {
	w := makeWindow()
	w.Settings.ElaborateOnly = true
	w.ToggleSetting(2) // index 2 = CompileOnly
	assert.True(t, w.Settings.CompileOnly)
	assert.False(t, w.Settings.ElaborateOnly)
}

func TestWindowModel_ToggleSetting_ElaborateOnlyClearsCompile(t *testing.T) {
	w := makeWindow()
	w.Settings.CompileOnly = true
	w.ToggleSetting(3) // index 3 = ElaborateOnly
	assert.True(t, w.Settings.ElaborateOnly)
	assert.False(t, w.Settings.CompileOnly)
}

func TestWindowModel_SettingCount(t *testing.T) {
	// Ensure SettingCount returns the right number so the UI panel is sized correctly.
	assert.Equal(t, 6, ui.SettingCount())
}

func TestWindowModel_SetOutputPath(t *testing.T) {
	w := makeWindow()
	w.SetOutputPath("sim_out")
	assert.Equal(t, "sim_out", w.Settings.OutputPath)
}

func TestWindowModel_TotalSettingRows(t *testing.T) {
	assert.Equal(t, 7, ui.TotalSettingRows())
}

// Compile-time check: persist.Settings must be accessible via WindowModel.
var _ persist.Settings = persist.Settings{}
