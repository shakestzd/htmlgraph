package worktree_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shakestzd/htmlgraph/internal/worktree"
)

// TestRepairGitdirStalePath is the primary TDD test: create a real worktree,
// overwrite its .git file with a bogus absolute path, call RepairGitdir, and
// assert the file is corrected.
func TestRepairGitdirStalePath(t *testing.T) {
	dir := setupGitRepo(t)

	// Create a real worktree so git registers it under .git/worktrees/.
	worktreePath, err := worktree.EnsureForTrack("trk-repair111", dir, io.Discard)
	if err != nil {
		t.Fatalf("EnsureForTrack: %v", err)
	}

	// Overwrite the .git file with a stale cross-machine path.
	gitFile := filepath.Join(worktreePath, ".git")
	bogusGitdir := "/nonexistent/cross/machine/path/.git/worktrees/trk-repair111"
	if err := os.WriteFile(gitFile, []byte("gitdir: "+bogusGitdir+"\n"), 0644); err != nil {
		t.Fatalf("overwrite .git: %v", err)
	}

	// Call repair.
	mainGitDir := filepath.Join(dir, ".git")
	if err := worktree.RepairGitdir(worktreePath, mainGitDir); err != nil {
		t.Fatalf("RepairGitdir: %v", err)
	}

	// Assert the .git file now contains a valid gitdir.
	content, err := os.ReadFile(gitFile)
	if err != nil {
		t.Fatalf("read .git after repair: %v", err)
	}

	line := strings.TrimSpace(string(content))
	if !strings.HasPrefix(line, "gitdir: ") {
		t.Fatalf("repaired .git has unexpected format: %q", line)
	}

	repairedGitdir := strings.TrimPrefix(line, "gitdir: ")

	// The repaired path must exist on disk.
	if _, err := os.Stat(repairedGitdir); err != nil {
		t.Errorf("repaired gitdir %q does not exist: %v", repairedGitdir, err)
	}

	// The repaired path must be under the main repo's .git/worktrees/.
	expectedPrefix := filepath.Join(mainGitDir, "worktrees")
	if !strings.HasPrefix(repairedGitdir, expectedPrefix) {
		t.Errorf("repaired gitdir %q not under %q", repairedGitdir, expectedPrefix)
	}
}

// TestRepairGitdirNoOpWhenValid verifies that repair is a no-op when the
// gitdir already points to an existing path.
func TestRepairGitdirNoOpWhenValid(t *testing.T) {
	dir := setupGitRepo(t)

	worktreePath, err := worktree.EnsureForTrack("trk-repair222", dir, io.Discard)
	if err != nil {
		t.Fatalf("EnsureForTrack: %v", err)
	}

	// Read the original .git content before repair.
	gitFile := filepath.Join(worktreePath, ".git")
	originalContent, err := os.ReadFile(gitFile)
	if err != nil {
		t.Fatalf("read original .git: %v", err)
	}

	mainGitDir := filepath.Join(dir, ".git")
	if err := worktree.RepairGitdir(worktreePath, mainGitDir); err != nil {
		t.Fatalf("RepairGitdir (no-op): %v", err)
	}

	afterContent, err := os.ReadFile(gitFile)
	if err != nil {
		t.Fatalf("read .git after repair: %v", err)
	}

	if string(originalContent) != string(afterContent) {
		t.Errorf("file changed on no-op repair:\nbefore: %q\nafter:  %q",
			originalContent, afterContent)
	}
}

// TestRepairGitdirMissingFile verifies that repair returns nil (not an error)
// when the .git file does not exist.
func TestRepairGitdirMissingFile(t *testing.T) {
	dir := t.TempDir()
	mainGitDir := filepath.Join(dir, ".git")

	if err := worktree.RepairGitdir("/nonexistent/worktree", mainGitDir); err != nil {
		t.Errorf("expected nil error for missing .git file, got: %v", err)
	}
}

// TestRepairGitdirUnrecognizedFormat verifies that repair is a no-op when
// the .git file content does not start with "gitdir: ".
func TestRepairGitdirUnrecognizedFormat(t *testing.T) {
	dir := t.TempDir()
	worktreePath := dir

	gitFile := filepath.Join(worktreePath, ".git")
	if err := os.WriteFile(gitFile, []byte("not a gitdir line\n"), 0644); err != nil {
		t.Fatalf("write fake .git: %v", err)
	}

	mainGitDir := filepath.Join(dir, "fake.git")
	if err := worktree.RepairGitdir(worktreePath, mainGitDir); err != nil {
		t.Errorf("expected nil error for unrecognized format, got: %v", err)
	}

	// File must be unchanged.
	content, _ := os.ReadFile(gitFile)
	if string(content) != "not a gitdir line\n" {
		t.Errorf("file was modified: %q", content)
	}
}

// TestRepairGitdirFromRepoRoot verifies the convenience wrapper that derives
// mainGitDir from the repo root.
func TestRepairGitdirFromRepoRoot(t *testing.T) {
	dir := setupGitRepo(t)

	worktreePath, err := worktree.EnsureForTrack("trk-repair333", dir, io.Discard)
	if err != nil {
		t.Fatalf("EnsureForTrack: %v", err)
	}

	// Overwrite with a stale path.
	gitFile := filepath.Join(worktreePath, ".git")
	if err := os.WriteFile(gitFile, []byte("gitdir: /old/machine/path/.git/worktrees/trk-repair333\n"), 0644); err != nil {
		t.Fatalf("overwrite .git: %v", err)
	}

	if err := worktree.RepairGitdirFromRepoRoot(worktreePath, dir); err != nil {
		t.Fatalf("RepairGitdirFromRepoRoot: %v", err)
	}

	content, err := os.ReadFile(gitFile)
	if err != nil {
		t.Fatalf("read .git after repair: %v", err)
	}

	line := strings.TrimSpace(string(content))
	repairedGitdir := strings.TrimPrefix(line, "gitdir: ")
	if _, err := os.Stat(repairedGitdir); err != nil {
		t.Errorf("repaired gitdir %q does not exist: %v", repairedGitdir, err)
	}
}
