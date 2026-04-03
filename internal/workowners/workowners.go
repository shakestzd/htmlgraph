// Package workowners parses .htmlgraph/WORKOWNERS files that map gitignore-style
// glob patterns to track or feature IDs. This provides static, explicit ownership
// that overrides the heuristic DB-based file ownership resolution.
//
// Format (one rule per line):
//
//	# Comment
//	cmd/htmlgraph/**  trk-f2a1a880
//	internal/db/*.go  feat-abc123
//	*.md              trk-docs
//
// Patterns use filepath.Match semantics with ** for recursive matching.
// The last matching rule wins (like .gitignore).
package workowners

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Rule maps a glob pattern to an owner work item ID.
type Rule struct {
	Pattern string
	OwnerID string
}

// File represents a parsed WORKOWNERS file.
type File struct {
	Rules []Rule
}

// Parse reads and parses a WORKOWNERS file.
// Returns nil (no error) if the file doesn't exist.
func Parse(path string) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var rules []Rule
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		rules = append(rules, Rule{Pattern: parts[0], OwnerID: parts[1]})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &File{Rules: rules}, nil
}

// Resolve returns the owner ID for a file path. Returns empty string if no
// rule matches. The last matching rule wins (like .gitignore).
func (wf *File) Resolve(filePath string) string {
	if wf == nil || len(wf.Rules) == 0 {
		return ""
	}
	var match string
	for _, r := range wf.Rules {
		if matchPattern(r.Pattern, filePath) {
			match = r.OwnerID
		}
	}
	return match
}

// matchPattern checks if filePath matches a gitignore-style pattern.
// Supports ** for recursive directory matching.
func matchPattern(pattern, filePath string) bool {
	// Handle ** patterns by splitting into segments.
	if strings.Contains(pattern, "**") {
		// "dir/**" matches everything under dir/
		prefix := strings.TrimSuffix(pattern, "/**")
		if prefix != pattern {
			return strings.HasPrefix(filePath, prefix+"/") || filePath == prefix
		}
		// "**/suffix" matches suffix at any depth
		suffix := strings.TrimPrefix(pattern, "**/")
		if suffix != pattern {
			return strings.HasSuffix(filePath, suffix) ||
				strings.Contains(filePath, "/"+suffix)
		}
		// General **: try matching with each directory removed
		parts := strings.Split(pattern, "/**/")
		if len(parts) == 2 {
			if !strings.HasPrefix(filePath, parts[0]+"/") {
				return false
			}
			rest := filePath[len(parts[0])+1:]
			matched, _ := filepath.Match(parts[1], filepath.Base(rest))
			return matched || strings.HasSuffix(rest, parts[1])
		}
	}
	matched, _ := filepath.Match(pattern, filePath)
	if matched {
		return true
	}
	// Also try matching against just the filename.
	matched, _ = filepath.Match(pattern, filepath.Base(filePath))
	return matched
}
