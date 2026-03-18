package runner

import (
	"regexp"
	"strings"

	"github.com/lazyvunit/lazy_vunit/internal/tree"
)

// testNameRe matches a VUnit full test name: word.word.word
var testNameRe = regexp.MustCompile(`^(\w+\.\w+\.\w+)`)

type ParseResult struct {
	TestName string
	Status   tree.Status
}

// ParseLine checks line (and prevLine for two-line patterns) for pass/fail signals.
// Returns (result, true) if a status change was detected.
func ParseLine(line, prevLine string) (ParseResult, bool) {
	trimmed := strings.TrimSpace(line)

	// Pattern: "lib.tb.test  passed" or "lib.tb.test  failed" on one line
	if m := testNameRe.FindString(trimmed); m != "" {
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, " passed") || strings.HasSuffix(lower, "passed") {
			return ParseResult{TestName: m, Status: tree.Passed}, true
		}
		if strings.Contains(lower, " failed") || strings.HasSuffix(lower, "failed") {
			return ParseResult{TestName: m, Status: tree.Failed}, true
		}
		return ParseResult{}, false
	}

	// Pattern: current line is "passed"/"failed", prev line is the test name
	if prevLine != "" {
		prevTrimmed := strings.TrimSpace(prevLine)
		if m := testNameRe.FindString(prevTrimmed); m != "" {
			switch strings.ToLower(trimmed) {
			case "passed":
				return ParseResult{TestName: m, Status: tree.Passed}, true
			case "failed":
				return ParseResult{TestName: m, Status: tree.Failed}, true
			}
		}
	}

	return ParseResult{}, false
}
