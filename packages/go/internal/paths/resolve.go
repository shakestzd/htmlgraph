// Package paths provides shared path-resolution utilities for the htmlgraph
// CLI and hook runner.
package paths

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolveViaGitCommonDir detects when dir is inside a git linked worktree and
// returns the main repository root (i.e. the parent of the shared .git dir).
//
// When running in a linked worktree, `git rev-parse --git-common-dir` returns
// something like `../../.git` or an absolute path — its parent directory is the
// main repo root.  When running in the main worktree the command returns the
// literal string `.git`, which means we are NOT in a linked worktree; in that
// case the function returns "" so the caller falls through to its normal
// walk-up logic.
//
// The function also verifies that the resolved main repo root contains a
// `.htmlgraph/` directory before returning it, so callers can use the return
// value directly as a project root without a second stat.
//
// If dir is empty the function uses os.Getwd().
// All errors are silently ignored; on any failure the function returns "".
func ResolveViaGitCommonDir(dir string) string {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil || dir == "" {
			return ""
		}
	}

	cmd := exec.Command("git", "-C", dir, "rev-parse", "--git-common-dir")
	out, err := cmd.Output()
	if err != nil {
		return "" // not a git repo, or git not installed
	}

	gitCommonDir := strings.TrimSpace(string(out))
	if gitCommonDir == "" || gitCommonDir == ".git" {
		// ".git" means we are in the main worktree, not a linked worktree.
		// Let the caller's normal walk-up handle it.
		return ""
	}

	// For linked worktrees the path may be relative (e.g. "../../.git").
	if !filepath.IsAbs(gitCommonDir) {
		gitCommonDir = filepath.Join(dir, gitCommonDir)
	}
	gitCommonDir = filepath.Clean(gitCommonDir)

	mainRepoRoot := filepath.Dir(gitCommonDir)

	candidate := filepath.Join(mainRepoRoot, ".htmlgraph")
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return mainRepoRoot
	}
	return ""
}

// GetGitRemoteURL returns the remote origin URL for the given directory by
// running `git -C <dir> remote get-url origin`.  It returns an empty string
// on any error (not a git repo, no origin remote, git not installed, etc.).
// If dir is empty, the function returns "" immediately.
func GetGitRemoteURL(dir string) string {
	if dir == "" {
		return ""
	}
	cmd := exec.Command("git", "-C", dir, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
