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
