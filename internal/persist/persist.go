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
