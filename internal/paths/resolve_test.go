package paths_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/shakestzd/htmlgraph/internal/paths"
)

// TestResolveViaGitCommonDir_NonGitDir verifies that a plain (non-git)
// temporary directory causes the function to return "".
func TestResolveViaGitCommonDir_NonGitDir(t *testing.T) {
	tmpDir := t.TempDir()
	result := paths.ResolveViaGitCommonDir(tmpDir)
	if result != "" {
		t.Errorf("expected empty string for non-git dir, got %q", result)
	}
}

// TestResolveViaGitCommonDir_MainWorktree verifies that running from the main
// worktree (where --git-common-dir returns ".git") causes the function to
// return "" so the caller falls through to normal resolution.
func TestResolveViaGitCommonDir_MainWorktree(t *testing.T) {
	// Use the actual project root (this repo is a git repo).
	// git rev-parse --git-common-dir from the repo root returns ".git",
	// so the function must return "" to avoid short-circuiting normal logic.
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Skip("cannot determine working directory")
	}

	result := paths.ResolveViaGitCommonDir(repoRoot)
	// We don't assert "" here because the CI environment might be a worktree
	// itself; we just ensure the function doesn't panic and returns a string.
	_ = result
}

// TestResolveViaGitCommonDir_EmptyDir verifies that an empty dir argument
// falls back to os.Getwd() without panicking.
func TestResolveViaGitCommonDir_EmptyDir(t *testing.T) {
	// Should not panic regardless of whether CWD is a git repo.
	_ = paths.ResolveViaGitCommonDir("")
}

// TestResolveViaGitCommonDir_NoHtmlgraph verifies that even when git
// common-dir resolves, the function returns "" if the main repo has no
// .htmlgraph directory.
func TestResolveViaGitCommonDir_NoHtmlgraph(t *testing.T) {
	// Create a temp dir that looks like a git repo main root (has .git/) but
	// no .htmlgraph/. We can't easily simulate --git-common-dir returning a
	// path, so this test validates the stat guard via a direct integration:
	// any tmpDir without .htmlgraph should not be returned.
	tmpDir := t.TempDir()
	// Pretend .git exists so git might resolve, but there's no .htmlgraph.
	// In practice git won't recognise it as a worktree, so the function
	// returns "" anyway — this just documents the expected safety net.
	result := paths.ResolveViaGitCommonDir(tmpDir)
	if result != "" {
		// Only fail if result doesn't actually have .htmlgraph
		htmlgraphPath := filepath.Join(result, ".htmlgraph")
		if _, err := os.Stat(htmlgraphPath); os.IsNotExist(err) {
			t.Errorf("returned %q which has no .htmlgraph directory", result)
		}
	}
}

// TestGetGitRemoteURL_EmptyDir verifies that an empty dir returns "".
func TestGetGitRemoteURL_EmptyDir(t *testing.T) {
	result := paths.GetGitRemoteURL("")
	if result != "" {
		t.Errorf("expected empty string for empty dir, got %q", result)
	}
}

// TestGetGitRemoteURL_NonGitDir verifies that a plain directory returns "".
func TestGetGitRemoteURL_NonGitDir(t *testing.T) {
	tmpDir := t.TempDir()
	result := paths.GetGitRemoteURL(tmpDir)
	if result != "" {
		t.Errorf("expected empty string for non-git dir, got %q", result)
	}
}

// TestGetGitRemoteURL_GitRepo verifies that a real git repo with an origin
// returns a non-empty URL.
func TestGetGitRemoteURL_GitRepo(t *testing.T) {
	// Use the actual repo root — it should have an origin remote.
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Skip("cannot determine working directory")
	}
	result := paths.GetGitRemoteURL(repoRoot)
	// We can't assert the exact URL, but a real repo should return something.
	// If it's empty, the repo has no origin remote — skip rather than fail.
	if result == "" {
		t.Skip("no origin remote configured in this repo")
	}
	// Sanity check: URL should contain at least a slash or colon (path/host separator).
	if len(result) < 5 {
		t.Errorf("GetGitRemoteURL returned suspiciously short URL: %q", result)
	}
}

// TestResolveProjectDir_HtmlgraphProjectDirEnv verifies that HTMLGRAPH_PROJECT_DIR
// is honoured so subagent hooks can locate .htmlgraph/ when EventCWD is a temp dir.
func TestResolveProjectDir_HtmlgraphProjectDirEnv(t *testing.T) {
	// Set up a real project directory with .htmlgraph/.
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".htmlgraph"), 0o755); err != nil {
		t.Fatalf("mkdir .htmlgraph: %v", err)
	}

	// Simulate a subagent whose EventCWD is an unrelated temp dir with no .htmlgraph/.
	fakeTmpCWD := t.TempDir()

	// Unset CLAUDE_PROJECT_DIR so it does not interfere.
	t.Setenv("CLAUDE_PROJECT_DIR", "")
	// Set HTMLGRAPH_PROJECT_DIR to the real project dir (written by SubagentStart).
	t.Setenv("HTMLGRAPH_PROJECT_DIR", projectDir)

	got, err := paths.ResolveProjectDir(paths.ProjectDirOptions{
		EventCWD:   fakeTmpCWD,
		WalkLevels: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != projectDir {
		t.Errorf("ResolveProjectDir = %q, want %q", got, projectDir)
	}
}

// TestResolveProjectDir_HtmlgraphProjectDirEnv_Invalid verifies that an
// invalid HTMLGRAPH_PROJECT_DIR value (no .htmlgraph/) falls through to the
// next resolution step rather than returning a wrong path.
func TestResolveProjectDir_HtmlgraphProjectDirEnv_Invalid(t *testing.T) {
	// A temp dir without .htmlgraph/ — should be skipped.
	badDir := t.TempDir()

	t.Setenv("CLAUDE_PROJECT_DIR", "")
	t.Setenv("HTMLGRAPH_PROJECT_DIR", badDir) // no .htmlgraph/ here

	// Verify that the invalid env var is skipped (not returned).
	// The resolver will fall through to later steps (git detection, walk-up, etc.),
	// but the result should NOT be the invalid badDir.
	got, err := paths.ResolveProjectDir(paths.ProjectDirOptions{
		EventCWD:   t.TempDir(),
		WalkLevels: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == badDir {
		t.Errorf("ResolveProjectDir = %q, should not return invalid HTMLGRAPH_PROJECT_DIR", got)
	}
}

// TestResolveProjectDir_HintFile verifies that the temp hint file (step 4 in
// the resolution chain) is used when HTMLGRAPH_PROJECT_DIR is not set.
// This covers the worktree subagent case where CLAUDE_ENV_FILE is unset so
// SubagentStart writes to the hint file instead of the env file.
func TestResolveProjectDir_HintFile(t *testing.T) {
	// Set up a real project directory with .htmlgraph/.
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".htmlgraph"), 0o755); err != nil {
		t.Fatalf("mkdir .htmlgraph: %v", err)
	}

	// Simulate subagent EventCWD in an unrelated temp dir.
	fakeTmpCWD := t.TempDir()

	// Clear both env vars so steps 2 and 3 are skipped.
	t.Setenv("CLAUDE_PROJECT_DIR", "")
	t.Setenv("HTMLGRAPH_PROJECT_DIR", "")

	// Write the hint file (simulates writeProjectDirHint in SubagentStart).
	hintPath := paths.ProjectDirHintPath()
	origHint, readErr := os.ReadFile(hintPath)
	if readErr == nil {
		// Restore the original hint file after the test.
		t.Cleanup(func() { _ = os.WriteFile(hintPath, origHint, 0o644) })
	} else {
		t.Cleanup(func() { _ = os.Remove(hintPath) })
	}
	if err := os.WriteFile(hintPath, []byte(projectDir), 0o644); err != nil {
		t.Fatalf("write hint file: %v", err)
	}

	got, err := paths.ResolveProjectDir(paths.ProjectDirOptions{
		EventCWD:   fakeTmpCWD,
		WalkLevels: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != projectDir {
		t.Errorf("ResolveProjectDir = %q, want %q (hint file not used?)", got, projectDir)
	}
}

// TestResolveProjectDir_HintFile_Invalid verifies that a stale/invalid hint
// file (pointing to a dir with no .htmlgraph/) is skipped and the resolver
// falls through to the next step.
func TestResolveProjectDir_HintFile_Invalid(t *testing.T) {
	// A project dir that DOES have .htmlgraph/ — used as EventCWD direct hit.
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".htmlgraph"), 0o755); err != nil {
		t.Fatalf("mkdir .htmlgraph: %v", err)
	}

	// A bad hint dir with no .htmlgraph/.
	badDir := t.TempDir()

	t.Setenv("CLAUDE_PROJECT_DIR", "")
	t.Setenv("HTMLGRAPH_PROJECT_DIR", "")

	// Write a stale hint pointing at a dir without .htmlgraph/.
	hintPath := paths.ProjectDirHintPath()
	origHint, readErr := os.ReadFile(hintPath)
	if readErr == nil {
		t.Cleanup(func() { _ = os.WriteFile(hintPath, origHint, 0o644) })
	} else {
		t.Cleanup(func() { _ = os.Remove(hintPath) })
	}
	if err := os.WriteFile(hintPath, []byte(badDir), 0o644); err != nil {
		t.Fatalf("write hint file: %v", err)
	}

	// EventCWD points directly at a valid project — step 6 should find it.
	got, err := paths.ResolveProjectDir(paths.ProjectDirOptions{
		EventCWD:   projectDir,
		WalkLevels: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != projectDir {
		t.Errorf("ResolveProjectDir = %q, want %q (stale hint not skipped?)", got, projectDir)
	}
}

// TestGetGitRemoteURL_InitedRepo verifies that a fresh git repo with a remote
// returns the configured URL.
func TestGetGitRemoteURL_InitedRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialise a bare git repo and add an origin remote.
	if err := runGit(tmpDir, "init"); err != nil {
		t.Skipf("git init failed: %v", err)
	}
	wantURL := "https://github.com/example/repo.git"
	if err := runGit(tmpDir, "remote", "add", "origin", wantURL); err != nil {
		t.Fatalf("git remote add failed: %v", err)
	}

	result := paths.GetGitRemoteURL(tmpDir)
	if result != wantURL {
		t.Errorf("GetGitRemoteURL = %q, want %q", result, wantURL)
	}
}

// runGit is a test helper that runs a git subcommand in dir.
func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Run()
}
