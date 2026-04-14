package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestHistoryResolvesFilePath verifies resolveHistoryPath maps each work-item
// prefix to the correct subdirectory under .htmlgraph/.
func TestHistoryResolvesFilePath(t *testing.T) {
	t.Parallel()

	// Build a temporary .htmlgraph tree with one file per type.
	root := t.TempDir()
	hgDir := filepath.Join(root, ".htmlgraph")

	dirs := map[string]string{
		"feat-abc12345": "features",
		"bug-abc12345":  "bugs",
		"spk-abc12345":  "spikes",
		"plan-abc12345": "plans",
		"trk-abc12345":  "tracks",
	}

	// Create directories and stub files.
	for id, sub := range dirs {
		dir := filepath.Join(hgDir, sub)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		ext := ".html"
		if sub == "plans" {
			ext = ".yaml"
		}
		f := filepath.Join(dir, id+ext)
		if err := os.WriteFile(f, []byte("stub"), 0644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}

	tests := []struct {
		id      string
		wantDir string
		wantExt string
	}{
		{"feat-abc12345", "features", ".html"},
		{"bug-abc12345", "bugs", ".html"},
		{"spk-abc12345", "spikes", ".html"},
		{"plan-abc12345", "plans", ".yaml"},
		{"trk-abc12345", "tracks", ".html"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			t.Parallel()
			got, err := resolveHistoryPath(hgDir, tt.id)
			if err != nil {
				t.Fatalf("resolveHistoryPath(%q) error: %v", tt.id, err)
			}
			want := filepath.Join(hgDir, tt.wantDir, tt.id+tt.wantExt)
			if got != want {
				t.Errorf("resolveHistoryPath(%q) = %q, want %q", tt.id, got, want)
			}
		})
	}
}

// TestHistoryMissingFile verifies a clear error when neither the primary nor
// archive path exists.
func TestHistoryMissingFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	hgDir := filepath.Join(root, ".htmlgraph")
	if err := os.MkdirAll(hgDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	_, err := resolveHistoryPath(hgDir, "feat-deadbeef")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "feat-deadbeef") {
		t.Errorf("error should mention the id; got: %v", err)
	}
}

// seedRepo creates a git repo with two commits to an HTML file and returns the
// repo root path.
func seedRepo(t *testing.T) (repoRoot string, filePath string) {
	t.Helper()

	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Tester",
			"GIT_AUTHOR_EMAIL=tester@example.com",
			"GIT_COMMITTER_NAME=Tester",
			"GIT_COMMITTER_EMAIL=tester@example.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}

	run("git", "init", "-b", "main")
	run("git", "config", "user.email", "tester@example.com")
	run("git", "config", "user.name", "Tester")

	hgDir := filepath.Join(dir, ".htmlgraph", "features")
	if err := os.MkdirAll(hgDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	filePath = filepath.Join(hgDir, "feat-test0001.html")
	if err := os.WriteFile(filePath, []byte("<html>v1</html>"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "first commit")

	if err := os.WriteFile(filePath, []byte("<html>v2</html>"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "second commit")

	return dir, filePath
}

// TestHistoryRunsGitLog verifies that runHistoryLog returns log lines for both
// commits on a seeded repo.
func TestHistoryRunsGitLog(t *testing.T) {
	t.Parallel()

	repoRoot, _ := seedRepo(t)
	hgDir := filepath.Join(repoRoot, ".htmlgraph")

	path, err := resolveHistoryPath(hgDir, "feat-test0001")
	if err != nil {
		t.Fatalf("resolveHistoryPath: %v", err)
	}

	entries, err := runHistoryLog(repoRoot, path)
	if err != nil {
		t.Fatalf("runHistoryLog: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 log entries, got %d", len(entries))
	}

	subjects := make([]string, len(entries))
	for i, e := range entries {
		subjects[i] = e.Subject
	}
	joined := strings.Join(subjects, " | ")
	if !strings.Contains(joined, "first commit") {
		t.Errorf("expected 'first commit' in subjects; got: %s", joined)
	}
	if !strings.Contains(joined, "second commit") {
		t.Errorf("expected 'second commit' in subjects; got: %s", joined)
	}
}

// TestHistoryJSONOutput verifies that --json flag produces a parseable array
// of HistoryEntry objects.
func TestHistoryJSONOutput(t *testing.T) {
	t.Parallel()

	repoRoot, _ := seedRepo(t)
	hgDir := filepath.Join(repoRoot, ".htmlgraph")

	path, err := resolveHistoryPath(hgDir, "feat-test0001")
	if err != nil {
		t.Fatalf("resolveHistoryPath: %v", err)
	}

	entries, err := runHistoryLog(repoRoot, path)
	if err != nil {
		t.Fatalf("runHistoryLog: %v", err)
	}

	data, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var parsed []historyEntry
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if len(parsed) < 2 {
		t.Fatalf("expected at least 2 entries in JSON array, got %d", len(parsed))
	}
	for _, e := range parsed {
		if e.SHA == "" {
			t.Error("entry missing SHA")
		}
		if e.Subject == "" {
			t.Error("entry missing Subject")
		}
	}
}
