// Package paths provides shared path-resolution utilities for the htmlgraph
// CLI and hook runner.
package paths

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ProjectDirOptions configures the unified project-directory resolver.
// Zero value is safe: all optional fields default to "not set".
type ProjectDirOptions struct {
	// ExplicitDir is the value of the --project-dir CLI flag.
	// When non-empty it is checked first and an error is returned if it
	// does not contain a .htmlgraph directory.
	ExplicitDir string

	// EventCWD is the "cwd" field extracted from a CloudEvent payload.
	// Checked after CLAUDE_PROJECT_DIR and git-common-dir resolution.
	EventCWD string

	// WalkLevels is the maximum number of parent directories to traverse
	// during the CWD walk-up phase.  0 means "no limit" (walk to root).
	WalkLevels int

	// SessionID enables session-scoped hint lookup when set (hook context).
	// CLI callers leave this empty and skip the hint step entirely.
	SessionID string
}

// ResolveProjectDir locates the project root (the directory that contains
// .htmlgraph/) using the following priority order:
//
//  1. opts.ExplicitDir (--project-dir flag) — hard error if set but invalid
//  2. CLAUDE_PROJECT_DIR env var — fall through on miss (not an error)
//  3. HTMLGRAPH_PROJECT_DIR env var — written by SubagentStart for subagents
//     whose EventCWD is a temp dir (e.g. /private/tmp/claude-501/...)
//  4. Session-scoped hint file — written by SubagentStart for worktree
//     subagents whose CLAUDE_ENV_FILE is unset; read via ReadSessionHint()
//  5. ResolveViaGitCommonDir() — worktree → main repo root
//  6. opts.EventCWD — direct .htmlgraph check
//  7. os.Getwd() — direct .htmlgraph check
//  8. Walk-up from opts.EventCWD (limited by WalkLevels when > 0)
//  9. Walk-up from os.Getwd() (unlimited)
//
// Returns the project root directory (not the .htmlgraph subdirectory).
// The only hard-error case is when ExplicitDir is set but no .htmlgraph
// can be found there.  All other failures fall back gracefully.
func ResolveProjectDir(opts ProjectDirOptions) (string, error) {
	// 1. Explicit flag — highest priority, hard-fail on miss.
	if opts.ExplicitDir != "" {
		if _, err := os.Stat(filepath.Join(opts.ExplicitDir, ".htmlgraph")); err == nil {
			return opts.ExplicitDir, nil
		}
		return "", fmt.Errorf("--project-dir %q: no .htmlgraph directory found", opts.ExplicitDir)
	}

	// 2. CLAUDE_PROJECT_DIR env var — fall through on miss.
	if d := os.Getenv("CLAUDE_PROJECT_DIR"); d != "" {
		if _, err := os.Stat(filepath.Join(d, ".htmlgraph")); err == nil {
			return d, nil
		}
	}

	// 3. HTMLGRAPH_PROJECT_DIR env var — written by SubagentStart so that
	// subagent hook invocations can find the real project when EventCWD is a
	// temp directory (e.g. /private/tmp/claude-501/...).
	if d := os.Getenv("HTMLGRAPH_PROJECT_DIR"); d != "" {
		if _, err := os.Stat(filepath.Join(d, ".htmlgraph")); err == nil {
			return d, nil
		}
	}

	// 4. Session-scoped hint file — written by SubagentStart for worktree
	// subagents whose CLAUDE_ENV_FILE is unset. Only consulted when SessionID
	// is provided (hook context). CLI callers don't set SessionID and skip this.
	if opts.SessionID != "" {
		if d := ReadSessionHint(opts.SessionID); d != "" {
			if _, err := os.Stat(filepath.Join(d, ".htmlgraph")); err == nil {
				return d, nil
			}
		}
	}

	// 5. Git worktree detection — resolve linked worktrees to main repo root.
	startDir := opts.EventCWD
	if startDir == "" {
		startDir, _ = os.Getwd()
	}
	if dir := ResolveViaGitCommonDir(startDir); dir != "" {
		return dir, nil
	}

	// 6. EventCWD direct check.
	if opts.EventCWD != "" {
		if _, err := os.Stat(filepath.Join(opts.EventCWD, ".htmlgraph")); err == nil {
			return opts.EventCWD, nil
		}
	}

	// 7. Process CWD direct check.
	if wd, err := os.Getwd(); err == nil {
		if _, err := os.Stat(filepath.Join(wd, ".htmlgraph")); err == nil {
			return wd, nil
		}
	}

	// 8. Walk-up from EventCWD (limited when WalkLevels > 0).
	if opts.EventCWD != "" {
		if found := walkUpForHtmlgraph(opts.EventCWD, opts.WalkLevels); found != "" {
			return found, nil
		}
	}

	// 9. Walk-up from process CWD (unlimited).
	if wd, err := os.Getwd(); err == nil {
		if found := walkUpForHtmlgraph(wd, 0); found != "" {
			return found, nil
		}
	}

	// Fallback: return best-effort directory without error (mirrors prior hook
	// behaviour where ResolveProjectDir never returned an empty string).
	if opts.EventCWD != "" {
		return opts.EventCWD, nil
	}
	if wd, err := os.Getwd(); err == nil {
		return wd, nil
	}
	return "", errors.New("no .htmlgraph directory found (run from within an htmlgraph project)")
}

// walkUpForHtmlgraph traverses parent directories looking for .htmlgraph/.
// maxLevels == 0 means walk all the way to the filesystem root.
func walkUpForHtmlgraph(start string, maxLevels int) string {
	dir := start
	for i := 0; maxLevels == 0 || i < maxLevels; i++ {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
		if _, err := os.Stat(filepath.Join(dir, ".htmlgraph")); err == nil {
			return dir
		}
	}
	return ""
}

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

// SessionHintPath returns the path to the session-scoped project dir hint.
func SessionHintPath(sessionID string) string {
	return filepath.Join(os.TempDir(), "htmlgraph-session-"+sessionID+".projectdir")
}

// ReadSessionHint reads the project directory from a session-scoped hint file.
// Returns "" when the file does not exist or cannot be read.
func ReadSessionHint(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	b, err := os.ReadFile(SessionHintPath(sessionID))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// WriteSessionHint writes the project directory to a session-scoped hint file.
func WriteSessionHint(sessionID, projectDir string) {
	if sessionID == "" || projectDir == "" {
		return
	}
	_ = os.WriteFile(SessionHintPath(sessionID), []byte(projectDir), 0o644)
}

// CleanupSessionHint removes the session-scoped hint file.
func CleanupSessionHint(sessionID string) {
	if sessionID == "" {
		return
	}
	_ = os.Remove(SessionHintPath(sessionID))
}

// CleanupGlobalHint removes the legacy global hint file if it exists.
// Called once at startup to clean up stale state from older versions.
func CleanupGlobalHint() {
	_ = os.Remove(filepath.Join(os.TempDir(), "htmlgraph-project-dir.hint"))
}
