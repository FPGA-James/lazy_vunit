package scanner

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TestEntry is a single test case discovered from VUnit's --export-json output.
type TestEntry struct {
	Name     string // full VUnit name: "lib.tb_name.test_case"
	FilePath string // absolute path to the VHDL/SV source file
	Dir      string // directory of FilePath (used for tree grouping)
	Library  string // "lib"
	Bench    string // "tb_name"
	TestCase string // "test_case"
}

// Scan runs `python <runPy> --export-json <tmpFile>` and returns discovered tests.
func Scan(runPy, jsonPath string) ([]TestEntry, error) {
	cmd := exec.Command("python", runPy, "--export-json", jsonPath)
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, err
	}

	return ParseJSON(data)
}

// ParseJSON parses a VUnit export JSON file content into TestEntry slice.
// Exported for testability without requiring a real Python/VUnit installation.
func ParseJSON(data []byte) ([]TestEntry, error) {
	var raw struct {
		Tests []struct {
			Name     string `json:"name"`
			Location struct {
				FileName string `json:"file_name"`
			} `json:"location"`
		} `json:"tests"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var entries []TestEntry
	for _, test := range raw.Tests {
		// Parse the name: "lib.bench.testcase"
		parts := strings.Split(test.Name, ".")
		if len(parts) != 3 {
			// Skip malformed names silently
			continue
		}

		dir := filepath.Dir(test.Location.FileName)

		entry := TestEntry{
			Name:     test.Name,
			FilePath: test.Location.FileName,
			Dir:      dir,
			Library:  parts[0],
			Bench:    parts[1],
			TestCase: parts[2],
		}
		entries = append(entries, entry)
	}

	return entries, nil
}
