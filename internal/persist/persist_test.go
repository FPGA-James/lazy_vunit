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

func TestLoadSettings_DefaultsIfNotExist(t *testing.T) {
	dir := t.TempDir()
	s, err := persist.LoadSettings(dir, "src_alu")
	require.NoError(t, err)
	assert.Equal(t, persist.Settings{}, s) // all false
}

func TestLoadSettings_CreatesFileOnMiss(t *testing.T) {
	dir := t.TempDir()
	_, err := persist.LoadSettings(dir, "src_alu")
	require.NoError(t, err)
	_, statErr := os.Stat(filepath.Join(dir, "src_alu_settings.json"))
	assert.NoError(t, statErr, "settings file should be created on first load")
}

func TestSaveAndLoadSettings_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	in := persist.Settings{Verbose: true, FailFast: true}
	require.NoError(t, persist.SaveSettings(dir, "src_alu", in))
	out, err := persist.LoadSettings(dir, "src_alu")
	require.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestSaveSettings_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", ".lazyvunit")
	err := persist.SaveSettings(dir, "key", persist.Settings{})
	require.NoError(t, err)
	_, statErr := os.Stat(dir)
	assert.NoError(t, statErr)
}
