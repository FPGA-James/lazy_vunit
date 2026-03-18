package finder

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
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

	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].RelDir < scripts[j].RelDir
	})
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
	// If the given script has an AbsPath, check for collisions using it
	if s.AbsPath != "" {
		for _, other := range scripts {
			if other.AbsPath != s.AbsPath && other.LeafName == s.LeafName {
				return s.RelDir
			}
		}
		return s.LeafName
	}

	// If the given script has no AbsPath (test scenario), check for LeafName collisions directly
	count := 0
	for _, other := range scripts {
		if other.LeafName == s.LeafName {
			count++
		}
	}
	if count > 1 {
		return s.RelDir
	}
	return s.LeafName
}
