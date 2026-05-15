package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// testTmpComponentRe matches any single path component that looks like a Go
// test scratch directory name. These directories live inside the project tree
// (to avoid /tmp noexec in devcontainers) but must never trigger real git
// mutations during test runs.
var testTmpComponentRe = regexp.MustCompile(`^\.(test-tmp|tmp-gotest)$`)

// isTestTmpPath returns true when absPath contains a component matching
// testTmpComponentRe or begins with the well-known prefixes "tmp/go-tmp/" or
// "tmp/gotmp.". This is a defense-in-depth guard: even if a test
// mis-configures isolation, real git commits are prevented.
func isTestTmpPath(absPath string) bool {
	// Walk every component of the path.
	for p := absPath; ; {
		dir, base := filepath.Split(filepath.Clean(p))
		if testTmpComponentRe.MatchString(base) {
			return true
		}
		// Also catch absolute components like "/tmp/go-tmp/" by checking
		// the full remaining prefix.
		if strings.Contains(absPath, "/tmp/go-tmp/") || strings.Contains(absPath, "/tmp/gotmp.") {
			return true
		}
		if dir == p || dir == "" || dir == "." {
			break
		}
		p = dir
	}
	return false
}

// commitWipnoteArtifact stages and commits the work-item HTML file for the
// given typeName and id to the main git repository that contains wipnoteDir.
//
// typeName is "feature", "bug", or "spike"; the HTML lives at
// .wipnote/<typeName>s/<id>.html relative to the project root.
//
// action is the state-transition verb embedded in the commit message —
// "create", "start", "complete", "reopen", "block" — producing messages like
// "wipnote: start feat-XYZ" or "wipnote: complete feat-XYZ". The action gives
// `git log` a clean per-transition trail for each work item.
//
// Design: `git -C <repoRoot>` anchors the command to the main repository even
// when the caller's shell CWD is inside a linked worktree. The per-worktree
// gitignore installed by excludeWipnoteFromWorktree (internal/worktree) only
// affects the worktree's own `git status` display; explicit paths passed to
// `git add --` bypass the exclude entirely, so the file is staged and committed
// in the main repo regardless of CWD.
//
// Non-fatal contract: if the project is not in a git repo, or if git commit
// fails for any reason (hook rejection, locked index, nothing to commit), the
// function logs to stderr and returns nil. The caller must not make completion
// of the work item depend on the git commit succeeding.
func commitWipnoteArtifact(wipnoteDir, typeName, id, action string) error {
	// Derive the repo root: wipnoteDir is .wipnote/ inside the project root.
	repoRoot := filepath.Dir(wipnoteDir)

	// Defense-in-depth: reject paths that look like Go test scratch
	// directories. These live inside the project tree (see .gitignore comment
	// "Go test scratch space") but must never trigger real git mutations.
	// Silent skip — this is never a caller error, just a mis-configured test.
	absWipnote, err := filepath.Abs(wipnoteDir)
	if err != nil {
		absWipnote = wipnoteDir
	}
	if isTestTmpPath(absWipnote) {
		if os.Getenv("WIPNOTE_DEBUG") == "1" {
			fmt.Fprintf(stderr, "autocommit skipped: path looks like a test temp dir: %s\n", absWipnote)
		}
		return nil
	}

	if !isGitRepo(repoRoot) {
		fmt.Fprintf(stderr, "autocommit skipped: %s is not inside a git repository\n", repoRoot)
		return nil
	}

	subDir := typeName + "s"
	relPath := filepath.Join(".wipnote", subDir, id+".html")
	absPath := filepath.Join(wipnoteDir, subDir, id+".html")

	// Stage the file. Use an explicit path to avoid sweeping unrelated changes.
	addOut, err := exec.Command("git", "-C", repoRoot, "add", "--", absPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("autocommit: git add %s: %s: %w", relPath, strings.TrimSpace(string(addOut)), err)
	}

	// Check whether anything was staged by the add above. Use git diff --cached
	// on the specific file so we don't accidentally commit unrelated staged changes.
	diffOut, err := exec.Command("git", "-C", repoRoot, "diff", "--cached", "--quiet", "--", absPath).CombinedOutput()
	if err == nil {
		// Exit code 0 means no diff — nothing new to commit.
		return nil
	}
	_ = diffOut // non-zero exit is the expected "there is a diff" result

	// Commit only the artifact file — never touch the broader index.
	if action == "" {
		action = "update"
	}
	msg := "wipnote: " + action + " " + id
	commitOut, err := exec.Command(
		"git", "-C", repoRoot, "commit", "-m", msg, "--", absPath,
	).CombinedOutput()
	if err != nil {
		outStr := string(commitOut)
		if strings.Contains(outStr, "nothing to commit") || strings.Contains(outStr, "no changes added") {
			return nil
		}
		fmt.Fprintf(stderr, "autocommit warning: git commit failed for %s (artifact persisted to disk — please commit manually): %s\n",
			id, strings.TrimSpace(outStr))
		return nil
	}

	return nil
}

// checkUncommittedSourceCompleteGate refuses completion when tracked files
// outside .wipnote/ have uncommitted changes. Completion auto-commits only the
// work-item artifact, so allowing dirty source by default makes the "done"
// signal stronger than the durable implementation state.
func checkUncommittedSourceCompleteGate(wipnoteDir, id string, allowDirty bool) error {
	repoRoot := filepath.Dir(wipnoteDir)
	if !isGitRepo(repoRoot) {
		return nil
	}

	files, err := dirtyTrackedSourceFiles(repoRoot)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	if allowDirty {
		fmt.Fprintf(os.Stderr, "allow-dirty warning: completing %s with uncommitted source changes:\n", id)
		for _, f := range files {
			fmt.Fprintf(os.Stderr, "  %s\n", f)
		}
		return nil
	}

	return fmt.Errorf(
		"refusing to complete %s with uncommitted source changes outside .wipnote/:\n%s\n\nCommit the implementation first, for example:\n  git add %s && git commit -m %q\n\nTo bypass intentionally, rerun with --allow-dirty",
		id,
		formatPathList(files),
		strings.Join(shellQuotePaths(files), " "),
		id+": commit implementation",
	)
}

func dirtyTrackedSourceFiles(repoRoot string) ([]string, error) {
	out, err := exec.Command(
		"git", "-C", repoRoot, "status", "--porcelain=v1", "-z",
		"--untracked-files=no",
	).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git status: %s: %w", strings.TrimSpace(string(out)), err)
	}

	var files []string
	for _, entry := range parsePorcelainZ(out) {
		if isWipnotePath(entry.Path) {
			continue
		}
		files = append(files, entry.Path)
	}
	sort.Strings(files)
	return files, nil
}

type porcelainEntry struct {
	Status string
	Path   string
}

func parsePorcelainZ(out []byte) []porcelainEntry {
	var entries []porcelainEntry
	for len(out) > 0 {
		nul := bytes.IndexByte(out, 0)
		if nul < 0 {
			break
		}
		record := string(out[:nul])
		out = out[nul+1:]
		if len(record) < 4 {
			continue
		}
		status := record[:2]
		path := filepath.ToSlash(record[3:])
		entries = append(entries, porcelainEntry{Status: status, Path: path})

		// In porcelain v1 -z, renamed/copied entries carry the destination in
		// the first record and the source path in the following NUL record.
		if strings.ContainsAny(status, "RC") {
			nul = bytes.IndexByte(out, 0)
			if nul < 0 {
				break
			}
			out = out[nul+1:]
		}
	}
	return entries
}

func isWipnotePath(path string) bool {
	path = filepath.ToSlash(path)
	return path == ".wipnote" || strings.HasPrefix(path, ".wipnote/")
}

func formatPathList(paths []string) string {
	var b strings.Builder
	for _, path := range paths {
		fmt.Fprintf(&b, "  %s\n", path)
	}
	return strings.TrimRight(b.String(), "\n")
}

func shellQuotePaths(paths []string) []string {
	quoted := make([]string, 0, len(paths))
	for _, path := range paths {
		quoted = append(quoted, shellQuote(path))
	}
	return quoted
}

func shellQuote(path string) string {
	if path != "" && strings.IndexFunc(path, func(r rune) bool {
		return !(r == '/' || r == '.' || r == '-' || r == '_' ||
			(r >= '0' && r <= '9') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= 'a' && r <= 'z'))
	}) < 0 {
		return path
	}
	return "'" + strings.ReplaceAll(path, "'", "'\"'\"'") + "'"
}

// shouldAutocommitWorkitemArtifact returns true when commitWipnoteArtifact
// should run for the given typeName. Plans are excluded because they have
// their own atomic YAML+HTML commit path (commitPlanChange in plan_yaml_cmds.go);
// auto-committing only the rendered HTML would leave the authoritative YAML
// out of sync (roborev #1662). Future work-item types must opt in explicitly.
func shouldAutocommitWorkitemArtifact(typeName string) bool {
	switch typeName {
	case "feature", "bug", "spike":
		return true
	default:
		return false
	}
}

// actionFromStatus maps a wipnote work-item status value to the verb used in
// the auto-commit message ("wipnote: <action> <id>"). "in-progress" becomes
// "start" since that's the human-readable transition verb; "done" becomes
// "complete". Other statuses pass through verbatim, used as-is for messages
// like "wipnote: blocked <id>" or "wipnote: todo <id>" (rare resets).
func actionFromStatus(status string) string {
	switch status {
	case "in-progress":
		return "start"
	case "done":
		return "complete"
	default:
		return status
	}
}
