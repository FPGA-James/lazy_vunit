package persist_test

import (
	"os"
	"path/filepath"
	"strings"
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
