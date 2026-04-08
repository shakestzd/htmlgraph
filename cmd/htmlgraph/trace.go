package main

import (
	"fmt"
	"path/filepath"
	"strings"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/spf13/cobra"
)

func traceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "trace <commit-sha | file-path>",
		Short: "Trace a commit or file back to its work items",
		Long: `Takes a commit SHA or file path and returns attribution:

  trace <commit-sha>  — session, feature, and track for a commit
  trace <file-path>   — all features that touched the file, with tracks

Examples:
  htmlgraph trace abc1234
  htmlgraph trace internal/db/schema.go`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runTrace(args[0])
		},
	}
}

// looksLikeFilePath returns true when the argument looks like a file path
// rather than a commit SHA. File paths contain "/" or "." (except lone hex).
func looksLikeFilePath(arg string) bool {
	if strings.Contains(arg, "/") {
		return true
	}
	if strings.Contains(arg, ".") {
		return true
	}
	return false
}

func runTrace(arg string) error {
	if looksLikeFilePath(arg) {
		return runTraceFile(arg)
	}
	return runTraceCommit(arg)
}

func runTraceCommit(sha string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	database, err := dbpkg.Open(filepath.Join(dir, "htmlgraph.db"))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	commits, err := dbpkg.TraceCommit(database, sha)
	if err != nil {
		return err
	}
	if len(commits) == 0 {
		return fmt.Errorf("commit %s not found in git_commits table\nRun 'htmlgraph ingest commits' to import git history", sha)
	}

	sep := strings.Repeat("─", 60)
	fmt.Println(sep)
	fmt.Printf("  Trace: %s\n", truncate(sha, 10))
	fmt.Println(sep)

	for _, c := range commits {
		fmt.Printf("  Commit    %s\n", truncate(c.CommitHash, 10))
		if c.Message != "" {
			fmt.Printf("  Message   %s\n", truncate(c.Message, 55))
		}
		fmt.Printf("  Session   %s\n", c.SessionID)
		if c.FeatureID != "" {
			fmt.Printf("  Feature   %s\n", c.FeatureID)
		}
		if c.TrackID != "" {
			fmt.Printf("  Track     %s\n", c.TrackID)
		}
		if len(commits) > 1 {
			fmt.Println()
		}
	}
	return nil
}

func runTraceFile(filePath string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	database, err := dbpkg.Open(filepath.Join(dir, "htmlgraph.db"))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	results, err := dbpkg.TraceFile(database, filePath)
	if err != nil {
		return err
	}

	sep := strings.Repeat("─", 60)
	fmt.Println(sep)
	fmt.Printf("  Trace: %s\n", filePath)
	fmt.Println(sep)

	if len(results) == 0 {
		fmt.Println("  No features found for this file.")
		fmt.Println("  Run 'htmlgraph reindex' to rebuild file attribution.")
		return nil
	}

	// Collect unique tracks.
	tracks := make(map[string]bool)
	for _, r := range results {
		if r.TrackID != "" {
			tracks[r.TrackID] = true
		}
	}

	fmt.Printf("\n  Features (%d):\n", len(results))
	for _, r := range results {
		status := r.Status
		if status == "" {
			status = "unknown"
		}
		fmt.Printf("    %s  [%s]  %s\n", r.FeatureID, status, truncate(r.Title, 40))
		if r.TrackID != "" {
			fmt.Printf("      Track: %s\n", r.TrackID)
		}
		fmt.Printf("      Op: %s  Last seen: %s\n", r.Operation, truncate(r.LastSeen, 19))
	}

	if len(tracks) > 0 {
		fmt.Printf("\n  Tracks (%d):\n", len(tracks))
		for trackID := range tracks {
			fmt.Printf("    %s\n", trackID)
		}
	}

	// Show the most likely owner.
	if owner := dbpkg.ResolveFileOwner(database, filePath); owner != nil {
		fmt.Printf("\n  Owner: %s", owner.FeatureID)
		if owner.Title != "" {
			fmt.Printf("  %s", truncate(owner.Title, 40))
		}
		fmt.Println()
	}

	return nil
}
