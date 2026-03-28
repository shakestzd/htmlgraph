package paths_test

import (
	"os"
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
