// Package main — agent_worktree provides git worktree helpers for parallel
// agent workflows. Each agent task gets its own worktree branched from the
// named track branch, keeping agent work isolated until it is ready to merge.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// agentBranchName returns the flat branch name for an agent task.
// Format: agent-<trackBranch>-<taskID>
func agentBranchName(trackBranch, taskID string) string {
	return "agent-" + trackBranch + "-" + taskID
}

// agentWorktreePath returns the worktree path for an agent task.
// Format: <projectDir>/.claude/worktrees/<trackBranch>/agent-<taskID>
func agentWorktreePath(trackBranch, taskID, projectDir string) string {
	return filepath.Join(projectDir, ".claude", "worktrees", trackBranch, "agent-"+taskID)
}

// createAgentWorktree creates (or reuses) a git worktree for the given agent
// task, branching from trackBranch. Returns the worktree path, a cleanup
// function that removes the worktree, and any error.
func createAgentWorktree(trackBranch, taskID, projectDir string) (string, func(), error) {
	worktreePath := agentWorktreePath(trackBranch, taskID, projectDir)
	branchName := agentBranchName(trackBranch, taskID)

	noop := func() {}

	// Verify the track branch exists before creating the worktree.
	checkCmd := exec.Command("git", "-C", projectDir, "rev-parse", "--verify", trackBranch)
	if out, err := checkCmd.CombinedOutput(); err != nil {
		return "", noop, fmt.Errorf("track branch %q not found: %s", trackBranch, out)
	}

	// If worktree path already exists, reuse it.
	if _, err := os.Stat(worktreePath); err == nil {
		cleanup := func() {
			exec.Command("git", "-C", projectDir, "worktree", "remove", "--force", worktreePath).Run() //nolint:errcheck
		}
		return worktreePath, cleanup, nil
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return "", noop, fmt.Errorf("mkdir %s: %w", filepath.Dir(worktreePath), err)
	}

	// Create the worktree with a new branch from trackBranch.
	addCmd := exec.Command("git", "-C", projectDir, "worktree", "add", "-b", branchName, worktreePath, trackBranch)
	if out, err := addCmd.CombinedOutput(); err != nil {
		return "", noop, fmt.Errorf("git worktree add: %s: %w", out, err)
	}

	cleanup := func() {
		exec.Command("git", "-C", projectDir, "worktree", "remove", "--force", worktreePath).Run() //nolint:errcheck
		exec.Command("git", "-C", projectDir, "branch", "-D", branchName).Run()                   //nolint:errcheck
	}

	return worktreePath, cleanup, nil
}

