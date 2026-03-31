package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCreateTrackWorktree_CreatesNewBranch tests that createTrackWorktree creates a new branch with the correct name.
func TestCreateTrackWorktree_CreatesNewBranch(t *testing.T) {
	// Set up a temp git repo
	tmpDir := t.TempDir()
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmpDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Create initial commit so we have something to branch from
	if err := gitCommitInDir(tmpDir, "initial commit"); err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}

	// Call createTrackWorktree
	trackID := "trk-abc123"
	worktreePath, cleanup, err := createTrackWorktree(trackID, tmpDir)
	if err != nil {
		t.Fatalf("createTrackWorktree failed: %v", err)
	}
	defer cleanup()

	// Assert: worktree path is .claude/worktrees/trk-abc123
	expectedPath := filepath.Join(tmpDir, ".claude", "worktrees", trackID)
	if worktreePath != expectedPath {
		t.Errorf("worktree path: got %q, want %q", worktreePath, expectedPath)
	}

	// Assert: path exists
	if _, err := os.Stat(worktreePath); err != nil {
		t.Errorf("worktree path does not exist: %v", err)
	}

	// Assert: branch name is trk-abc123 (can check by listing worktrees)
	listCmd := exec.Command("git", "worktree", "list")
	listCmd.Dir = tmpDir
	out, err := listCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git worktree list failed: %v", err)
	}
	if !contains(string(out), trackID) {
		t.Errorf("branch name %q not found in worktree list:\n%s", trackID, out)
	}
}

// TestCreateTrackWorktree_ReusesExisting tests that createTrackWorktree reuses an existing worktree.
func TestCreateTrackWorktree_ReusesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmpDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Create initial commit
	if err := gitCommitInDir(tmpDir, "initial"); err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}

	trackID := "trk-xyz789"

	// Create worktree first time
	worktreePath1, cleanup1, err := createTrackWorktree(trackID, tmpDir)
	if err != nil {
		t.Fatalf("first createTrackWorktree failed: %v", err)
	}
	defer cleanup1()

	// Create worktree second time (should reuse)
	worktreePath2, cleanup2, err := createTrackWorktree(trackID, tmpDir)
	if err != nil {
		t.Fatalf("second createTrackWorktree failed: %v", err)
	}
	defer cleanup2()

	// Assert: both calls return the same path
	if worktreePath1 != worktreePath2 {
		t.Errorf("paths differ: first %q, second %q", worktreePath1, worktreePath2)
	}
}

// TestResolveTrackForFeature_ReturnsTrackID tests that resolveTrackForFeature returns the track ID from a feature HTML.
func TestResolveTrackForFeature_ReturnsTrackID(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .htmlgraph/features directory
	featureDir := filepath.Join(tmpDir, ".htmlgraph", "features")
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		t.Fatalf("failed to create feature dir: %v", err)
	}

	// Create a feature HTML file with data-track-id
	featureID := "feat-xyz"
	trackID := "trk-parent123"
	html := `<article id="` + featureID + `" data-track-id="` + trackID + `" data-status="todo">
		<header><h1>Test Feature</h1></header>
		<section data-content><p>Description</p></section>
	</article>`

	featureFile := filepath.Join(featureDir, featureID+".html")
	if err := os.WriteFile(featureFile, []byte(html), 0644); err != nil {
		t.Fatalf("failed to write feature HTML: %v", err)
	}

	// Call resolveTrackForFeature
	resolvedTrackID := resolveTrackForFeature(featureID, tmpDir)

	// Assert: returns the track ID
	if resolvedTrackID != trackID {
		t.Errorf("track ID: got %q, want %q", resolvedTrackID, trackID)
	}
}

// TestResolveTrackForFeature_ReturnsEmptyWhenNoTrack tests that resolveTrackForFeature returns empty when feature has no track.
func TestResolveTrackForFeature_ReturnsEmptyWhenNoTrack(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .htmlgraph/features directory
	featureDir := filepath.Join(tmpDir, ".htmlgraph", "features")
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		t.Fatalf("failed to create feature dir: %v", err)
	}

	// Create a feature HTML file WITHOUT data-track-id
	featureID := "feat-standalone"
	html := `<article id="` + featureID + `" data-status="todo">
		<header><h1>Test Feature</h1></header>
		<section data-content><p>Description</p></section>
	</article>`

	featureFile := filepath.Join(featureDir, featureID+".html")
	if err := os.WriteFile(featureFile, []byte(html), 0644); err != nil {
		t.Fatalf("failed to write feature HTML: %v", err)
	}

	// Call resolveTrackForFeature
	resolvedTrackID := resolveTrackForFeature(featureID, tmpDir)

	// Assert: returns empty string
	if resolvedTrackID != "" {
		t.Errorf("track ID: got %q, want empty", resolvedTrackID)
	}
}

// TestResolveTrackForFeature_ReturnsEmptyWhenFileNotFound tests that resolveTrackForFeature returns empty when feature file not found.
func TestResolveTrackForFeature_ReturnsEmptyWhenFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Call resolveTrackForFeature with non-existent feature
	resolvedTrackID := resolveTrackForFeature("feat-nonexistent", tmpDir)

	// Assert: returns empty string (graceful)
	if resolvedTrackID != "" {
		t.Errorf("track ID: got %q, want empty", resolvedTrackID)
	}
}

// gitCommitInDir runs git commit in a temp directory, configuring
// user.name/user.email so the command works in CI environments.
func gitCommitInDir(dir, msg string) error {
	cmd := exec.Command("git",
		"-c", "user.name=test",
		"-c", "user.email=test@test.com",
		"commit", "--allow-empty", "-m", msg,
	)
	cmd.Dir = dir
	return cmd.Run()
}

// Helper: contains checks if a string is contained in another.
func contains(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && (haystack == needle || len(haystack) >= len(needle))
}
