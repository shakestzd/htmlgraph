package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupAgentGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Init git repo
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s", args, out)
		}
	}

	// Create initial commit
	f, _ := os.Create(filepath.Join(dir, "README.md"))
	f.WriteString("# Test")
	f.Close()
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "initial").Run()

	return dir
}

func TestCreateAgentWorktree_BranchesFromTrackBranch(t *testing.T) {
	dir := setupAgentGitRepo(t)

	// Create a track branch first (not via worktree, just as a branch)
	exec.Command("git", "-C", dir, "branch", "track-abc").Run()

	// Now create agent worktree branching from track-abc
	path, agentCleanup, err := createAgentWorktree("track-abc", "task1", dir)
	if err != nil {
		t.Fatalf("createAgentWorktree: %v", err)
	}
	defer agentCleanup()

	// Verify path
	expected := filepath.Join(dir, ".claude", "worktrees", "track-abc", "agent-task1")
	if path != expected {
		t.Errorf("expected path %s, got %s", expected, path)
	}

	// Verify branch name uses flat naming scheme
	out, _ := exec.Command("git", "-C", path, "rev-parse", "--abbrev-ref", "HEAD").Output()
	branch := strings.TrimSpace(string(out))
	if branch != "agent-track-abc-task1" {
		t.Errorf("expected branch agent-track-abc-task1, got %s", branch)
	}
}

func TestCreateAgentWorktree_PathNamingConvention(t *testing.T) {
	dir := setupAgentGitRepo(t)

	// Create a track branch
	exec.Command("git", "-C", dir, "branch", "mytrack").Run()

	path, cleanup, err := createAgentWorktree("mytrack", "my-task", dir)
	if err != nil {
		t.Fatalf("createAgentWorktree: %v", err)
	}
	defer cleanup()

	// Check the worktree path follows naming convention
	if !strings.Contains(path, "mytrack") || !strings.Contains(path, "agent-my-task") {
		t.Errorf("path doesn't follow naming convention: %s", path)
	}
}

func TestCreateAgentWorktree_FailsWithoutTrackBranch(t *testing.T) {
	dir := setupAgentGitRepo(t)

	// Don't create track branch — agent should fail
	_, _, err := createAgentWorktree("nonexistent-track", "task1", dir)
	if err == nil {
		t.Error("expected error when track branch doesn't exist")
	}
}

func TestCreateAgentWorktree_ReusesExistingPath(t *testing.T) {
	dir := setupAgentGitRepo(t)

	// Create a track branch
	exec.Command("git", "-C", dir, "branch", "track-reuse").Run()

	// Create agent worktree
	path1, cleanup1, err := createAgentWorktree("track-reuse", "task1", dir)
	if err != nil {
		t.Fatalf("first createAgentWorktree: %v", err)
	}
	defer cleanup1()

	// Try to create the same agent worktree again
	path2, cleanup2, err := createAgentWorktree("track-reuse", "task1", dir)
	if err != nil {
		t.Fatalf("second createAgentWorktree: %v", err)
	}
	defer cleanup2()

	if path1 != path2 {
		t.Errorf("reused worktree paths don't match: %s vs %s", path1, path2)
	}
}

func TestMergeAgentToTrack(t *testing.T) {
	dir := setupAgentGitRepo(t)

	// Create a track branch in the project root (not a worktree)
	exec.Command("git", "-C", dir, "branch", "merge-track").Run()

	// Create agent worktree
	agentPath, agentCleanup, err := createAgentWorktree("merge-track", "task1", dir)
	if err != nil {
		t.Fatalf("createAgentWorktree: %v", err)
	}
	defer agentCleanup()

	// Make a change in the agent worktree
	testFile := filepath.Join(agentPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Commit the change in agent worktree
	exec.Command("git", "-C", agentPath, "add", ".").Run()
	out, err := exec.Command("git", "-C", agentPath, "commit", "-m", "agent change").CombinedOutput()
	if err != nil {
		t.Fatalf("agent commit failed: %s", out)
	}

	// Merge agent to track
	if err := mergeAgentToTrack("merge-track", "task1", dir); err != nil {
		t.Fatalf("mergeAgentToTrack: %v", err)
	}

	// Verify the merge happened by checking the track branch has the new commit
	out, err = exec.Command("git", "-C", dir, "log", "merge-track", "--oneline", "-n", "2").CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v", err)
	}
	logOutput := string(out)
	if !strings.Contains(logOutput, "agent-task1") || !strings.Contains(logOutput, "merge-track") {
		t.Errorf("merge commit not found in track branch log:\n%s", logOutput)
	}
}
