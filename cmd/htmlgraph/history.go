package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// historyEntry holds a single git log record for a work-item file.
type historyEntry struct {
	SHA     string `json:"sha"`
	ISOTime string `json:"iso_time"`
	Author  string `json:"author"`
	Subject string `json:"subject"`
}

// newHistoryCmd returns the cobra command for `htmlgraph history <id>`.
func newHistoryCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "history <id>",
		Short: "Show the git commit history for a work-item file",
		Long: `Resolves a work-item ID to its HTML (or YAML) file and prints
the git log for that file, most-recent commit first.

Supported prefixes: feat-, bug-, spk-, plan-, trk-

Examples:
  htmlgraph history feat-2a43f5f8
  htmlgraph history plan-3b0d5133 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runHistory(args[0], jsonOut)
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON array of log entries")
	return cmd
}

// runHistory is the top-level handler: resolves the path, runs git log, and
// renders the result.
func runHistory(id string, jsonOut bool) error {
	hgDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	path, err := resolveHistoryPath(hgDir, id)
	if err != nil {
		return err
	}

	// Resolve the git toplevel from the directory that OWNS the discovered
	// .htmlgraph/ checkout — not from process cwd. Using cwd would target a
	// nested repo/submodule if the command were run inside one. `git -C <dir>`
	// pins resolution to the checkout that actually owns the work-item files.
	repoRoot, err := gitToplevel(filepath.Dir(hgDir))
	if err != nil {
		// Fallback: assume hgDir's parent is the toplevel (flat repo case).
		repoRoot = filepath.Dir(hgDir)
	}

	entries, err := runHistoryLog(repoRoot, path)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Fprintf(os.Stderr, "No commits found for %s\nIs this file tracked by git?\n", id)
		return nil
	}

	if jsonOut {
		return renderHistoryJSON(entries)
	}
	return renderHistoryTable(id, entries)
}

// resolveHistoryPath maps a work-item ID to its file path under hgDir.
// It checks the primary location first, then falls back to archives.
// Returns an error if neither exists.
func resolveHistoryPath(hgDir, id string) (string, error) {
	sub, ext := subDirAndExt(id)
	if sub == "" {
		return "", fmt.Errorf("unknown work-item prefix for %q (expected feat-, bug-, spk-, plan-, or trk-)", id)
	}

	primary := filepath.Join(hgDir, sub, id+ext)
	if _, err := os.Stat(primary); err == nil {
		return primary, nil
	}

	// Fallback: archives directory (flat, may have been renamed).
	archivePath := filepath.Join(hgDir, "archives", id+ext)
	if _, err := os.Stat(archivePath); err == nil {
		return archivePath, nil
	}

	return "", fmt.Errorf("work item %q not found in .htmlgraph/%s/ or .htmlgraph/archives/", id, sub)
}

// subDirAndExt returns the subdirectory name and file extension for a given
// work-item ID based on its prefix.
func subDirAndExt(id string) (string, string) {
	switch {
	case strings.HasPrefix(id, "feat-"):
		return "features", ".html"
	case strings.HasPrefix(id, "bug-"):
		return "bugs", ".html"
	case strings.HasPrefix(id, "spk-"):
		return "spikes", ".html"
	case strings.HasPrefix(id, "plan-"):
		return "plans", ".yaml"
	case strings.HasPrefix(id, "trk-"):
		return "tracks", ".html"
	default:
		return "", ""
	}
}

// gitToplevel returns the absolute path to the worktree that owns `dir` by
// invoking `git -C <dir> rev-parse --show-toplevel`. Pinning to `dir` makes
// the lookup independent of process cwd so a nested submodule or the user's
// shell location can't redirect `git log` to the wrong repository.
func gitToplevel(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("git -C %s rev-parse --show-toplevel: %w", dir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// runHistoryLog shells out to git log with --follow to handle renames and
// returns a slice of historyEntry values, newest first.
func runHistoryLog(repoRoot, filePath string) ([]historyEntry, error) {
	// %H = full SHA, %ai = author date ISO 8601, %an = author name, %s = subject
	cmd := exec.Command(
		"git", "log",
		"--follow",
		"--pretty=format:%H\t%ai\t%an\t%s",
		"--",
		filePath,
	)
	cmd.Dir = repoRoot

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}

	lines := strings.Split(raw, "\n")
	entries := make([]historyEntry, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		entries = append(entries, historyEntry{
			SHA:     parts[0],
			ISOTime: parts[1],
			Author:  parts[2],
			Subject: parts[3],
		})
	}
	return entries, nil
}

// renderHistoryTable pretty-prints entries as aligned columns to stdout.
func renderHistoryTable(id string, entries []historyEntry) error {
	sep := strings.Repeat("─", 72)
	fmt.Println(sep)
	fmt.Printf("  History: %s  (%d commits)\n", id, len(entries))
	fmt.Println(sep)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  DATE\tAUTHOR\tSHA\tSUBJECT")
	for _, e := range entries {
		date := e.ISOTime
		if len(date) >= 19 {
			date = date[:19] // trim timezone
		}
		sha := e.SHA
		if len(sha) > 8 {
			sha = sha[:8]
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
			date,
			truncate(e.Author, 20),
			sha,
			truncate(e.Subject, 50),
		)
	}
	return w.Flush()
}

// renderHistoryJSON marshals entries to indented JSON on stdout.
func renderHistoryJSON(entries []historyEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
