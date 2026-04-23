// Package worktree provides helpers to create and reuse git worktrees for
// HtmlGraph work items (features, tracks, and agent tasks).
//
// All three public functions are idempotent: calling them on an already-existing
// worktree returns the existing path without error. Progress messages are written
// to the io.Writer passed by the caller; pass io.Discard to suppress all output.
package worktree

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/shakestzd/htmlgraph/internal/htmlparse"
)

// skipReindexEnv is an explicit override for the reindex subprocess. When
// set to any non-empty value, reindexWorktree becomes a no-op. Useful for
// benchmarks or adhoc scripts; tests rely on the auto-detection below
// instead.
const skipReindexEnv = "HTMLGRAPH_WORKTREE_SKIP_REINDEX"

// isGoTestBinary reports whether the current process is a go-test binary,
// detected by the conventional ".test" suffix that go test produces.
// Centralizes the test-mode skip so any package using this worktree code
// from within `go test` automatically skips the reindex subprocess — no
// per-test-helper setup required. Keeps the "testing" import out of
// production code.
func isGoTestBinary() bool {
	return strings.HasSuffix(os.Args[0], ".test") ||
		strings.Contains(filepath.Base(os.Args[0]), ".test.")
}

// EnsureForFeature ensures a git worktree exists for the given feature and returns its path.
// When the feature belongs to a parent track, the track worktree is created/reused instead.
// Progress is written to w; pass io.Discard to suppress output.
func EnsureForFeature(featureID, repoRoot string, w io.Writer) (string, error) {
	// If the feature has a parent track, delegate to the track worktree.
	trackID := resolveTrackForFeature(featureID, repoRoot)
	if trackID != "" {
		return EnsureForTrack(trackID, repoRoot, w)
	}

	worktreePath := filepath.Join(repoRoot, ".claude", "worktrees", featureID)
	branchName := "yolo-" + featureID

	// Reuse existing worktree.
	if _, err := os.Stat(worktreePath); err == nil {
		fmt.Fprintf(w, "  Worktree: %s (reusing existing)\n", worktreePath)
		return worktreePath, nil
	}

	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return "", fmt.Errorf("could not create worktrees directory: %w", err)
	}

	cmd := exec.Command("git", "-C", repoRoot, "worktree", "add", worktreePath, "-b", branchName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git worktree add failed: %w\n%s", err, out)
	}

	fmt.Fprintf(w, "  Worktree: %s (branch: %s)\n", worktreePath, branchName)
	excludeHtmlgraphFromWorktree(worktreePath, w)
	reindexWorktree(worktreePath, w)

	return worktreePath, nil
}

// EnsureForTrack ensures a git worktree exists for the given track and returns its path.
// Progress is written to w; pass io.Discard to suppress output.
func EnsureForTrack(trackID, repoRoot string, w io.Writer) (string, error) {
	worktreePath := filepath.Join(repoRoot, ".claude", "worktrees", trackID)
	branchName := trackID // Track worktrees use the track ID as the branch name.

	// Reuse existing worktree.
	if _, err := os.Stat(worktreePath); err == nil {
		fmt.Fprintf(w, "  Worktree: %s (reusing existing)\n", worktreePath)
		return worktreePath, nil
	}

	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return "", fmt.Errorf("could not create worktrees directory: %w", err)
	}

	cmd := exec.Command("git", "-C", repoRoot, "worktree", "add", worktreePath, "-b", branchName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git worktree add failed: %w\n%s", err, out)
	}

	fmt.Fprintf(w, "  Worktree: %s (branch: %s)\n", worktreePath, branchName)
	excludeHtmlgraphFromWorktree(worktreePath, w)
	reindexWorktree(worktreePath, w)

	return worktreePath, nil
}

// EnsureForAgent ensures a git worktree exists for the given agent task and returns its path.
// The worktree branches from the track branch and is placed at
// .claude/worktrees/<trackID>/agent-<taskName>.
// Progress is written to w; pass io.Discard to suppress output.
func EnsureForAgent(trackID, taskName, repoRoot string, w io.Writer) (string, error) {
	agentBranch := "agent-" + trackID + "-" + taskName
	worktreePath := filepath.Join(repoRoot, ".claude", "worktrees", trackID, "agent-"+taskName)

	// Reuse existing worktree.
	if _, err := os.Stat(worktreePath); err == nil {
		fmt.Fprintf(w, "  Agent worktree: %s (reusing existing)\n", worktreePath)
		return worktreePath, nil
	}

	// Track branch must exist before creating an agent branch from it.
	if err := exec.Command("git", "-C", repoRoot, "rev-parse", "--verify", trackID).Run(); err != nil {
		return "", fmt.Errorf("track branch %s not found: create track worktree first with htmlgraph yolo --track %s", trackID, trackID)
	}

	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return "", fmt.Errorf("could not create agent worktrees directory: %w", err)
	}

	cmd := exec.Command("git", "-C", repoRoot, "worktree", "add", worktreePath, "-b", agentBranch, trackID)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git worktree add failed: %w\n%s", err, out)
	}

	fmt.Fprintf(w, "  Agent worktree: %s (branch: %s, from: %s)\n", worktreePath, agentBranch, trackID)
	return worktreePath, nil
}

// resolveTrackForFeature reads a feature HTML file and returns its data-track-id attribute.
// If the feature file doesn't exist or has no track ID, returns empty string.
func resolveTrackForFeature(featureID, projectRoot string) string {
	featureFile := filepath.Join(projectRoot, ".htmlgraph", "features", featureID+".html")
	node, err := htmlparse.ParseFile(featureFile)
	if err != nil {
		// File not found or parse error — gracefully return empty.
		return ""
	}
	return node.TrackID
}

// excludeHtmlgraphFromWorktree adds .htmlgraph/ to the worktree's local git exclude file.
// Best-effort: errors are written to w but do not abort.
func excludeHtmlgraphFromWorktree(worktreePath string, w io.Writer) {
	gitFile := filepath.Join(worktreePath, ".git")
	content, err := os.ReadFile(gitFile)
	if err != nil {
		fmt.Fprintf(w, "  Warning: could not read .git file for exclude setup: %v\n", err)
		return
	}

	gitdirLine := strings.TrimSpace(string(content))
	gitdir := strings.TrimPrefix(gitdirLine, "gitdir: ")
	if gitdir == gitdirLine {
		return // Not a worktree — no gitdir prefix found.
	}

	excludePath := filepath.Join(gitdir, "info", "exclude")
	if err := os.MkdirAll(filepath.Dir(excludePath), 0755); err != nil {
		fmt.Fprintf(w, "  Warning: could not create exclude directory: %v\n", err)
		return
	}

	f, err := os.OpenFile(excludePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(w, "  Warning: could not open exclude file: %v\n", err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString("\n.htmlgraph/\n"); err != nil {
		fmt.Fprintf(w, "  Warning: could not write to exclude file: %v\n", err)
	}
}

// reindexWorktree runs `htmlgraph reindex` in the given worktree directory so
// the worktree's SQLite cache is current before Claude launches. Best-effort:
// failures are written to w but do not abort.
//
// Auto-skipped when:
//   - the process is a go-test binary (any test file in any package),
//   - or HTMLGRAPH_WORKTREE_SKIP_REINDEX is explicitly set.
//
// Under `go test`, os.Executable() returns the test binary path; invoking
// it with `reindex` would recursively re-run tests. The auto-detection
// covers every consumer test — not just internal/worktree — without
// requiring per-test-helper env plumbing.
func reindexWorktree(worktreeDir string, w io.Writer) {
	if isGoTestBinary() || os.Getenv(skipReindexEnv) != "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(w, "  Warning: could not determine executable path for reindex: %v\n", err)
		return
	}
	reindexCmd := exec.CommandContext(ctx, exe, "reindex")
	reindexCmd.Dir = worktreeDir
	if err := reindexCmd.Run(); err != nil {
		fmt.Fprintf(w, "  Warning: reindex in worktree failed: %v\n", err)
	}
}
