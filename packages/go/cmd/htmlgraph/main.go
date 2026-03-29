// Package main is the entry point for the htmlgraph CLI.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shakestzd/htmlgraph/internal/paths"
	"github.com/spf13/cobra"
)

// version is set at build time via ldflags.
var version = "dev"

// projectDirFlag holds the value of the --project-dir persistent flag.
var projectDirFlag string

func main() {
	rootCmd := &cobra.Command{
		Use:           "htmlgraph",
		Short:         "Local-first observability for AI-assisted development",
		Long:          "HtmlGraph — local-first observability and coordination platform for AI-assisted development.",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	// --project-dir overrides all other project-root detection strategies.
	rootCmd.PersistentFlags().StringVar(
		&projectDirFlag,
		"project-dir",
		"",
		"explicit project root containing .htmlgraph/ (overrides CLAUDE_PROJECT_DIR and CWD walk-up)",
	)

	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(statuslineCmd())
	rootCmd.AddCommand(serveCmd())
	rootCmd.AddCommand(featureCmdWithExtras())
	rootCmd.AddCommand(workitemCmd("spike", "spikes"))
	rootCmd.AddCommand(workitemCmd("bug", "bugs"))
	rootCmd.AddCommand(snapshotCmd())
	rootCmd.AddCommand(hookCmd())
	rootCmd.AddCommand(claudeCmd())
	rootCmd.AddCommand(yoloCmd())
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(trackCmdWithExtras())
	rootCmd.AddCommand(sessionCmd())
	rootCmd.AddCommand(wipCmd())
	rootCmd.AddCommand(analyticsCmd())
	rootCmd.AddCommand(orchestratorCmd())
	rootCmd.AddCommand(installHooksCmd())
	rootCmd.AddCommand(buildCmd())
	rootCmd.AddCommand(setupCLICmd())
	rootCmd.AddCommand(devCmd())
	rootCmd.AddCommand(reportCmd())
	rootCmd.AddCommand(findCmd())
	rootCmd.AddCommand(checkCmd())
	rootCmd.AddCommand(healthCmd())
	rootCmd.AddCommand(budgetCmd())
	rootCmd.AddCommand(specCmd())
	rootCmd.AddCommand(reviewCmd())
	rootCmd.AddCommand(complianceCmd())
	rootCmd.AddCommand(tddCmd())
	rootCmd.AddCommand(ingestCmd())
	rootCmd.AddCommand(linkCmd())
	rootCmd.AddCommand(batchCmd())
	rootCmd.AddCommand(workitemCmd("plan", "plans"))
	rootCmd.AddCommand(backfillCmd())
	rootCmd.AddCommand(reindexCmd())
	rootCmd.AddCommand(helpCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("htmlgraph %s (go)\n", version)
		},
	}
}

// findHtmlgraphDir locates the .htmlgraph directory using the following
// priority order:
//
//  1. --project-dir flag (explicit override, highest priority)
//  2. CLAUDE_PROJECT_DIR env var (set by session-start hook; fixes plugin-dir CWD issue)
//  3. Walk up from os.Getwd() (original behaviour, lowest priority)
func findHtmlgraphDir() (string, error) {
	// 1. Explicit --project-dir flag takes top priority.
	if projectDirFlag != "" {
		candidate := filepath.Join(projectDirFlag, ".htmlgraph")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
		return "", fmt.Errorf("--project-dir %q: no .htmlgraph directory found", projectDirFlag)
	}

	// 2. CLAUDE_PROJECT_DIR env var (set by session-start hook for downstream
	//    invocations; also allows users/CI to override CWD without a flag).
	if d := os.Getenv("CLAUDE_PROJECT_DIR"); d != "" {
		candidate := filepath.Join(d, ".htmlgraph")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
		// Env var was set but points somewhere without .htmlgraph — fall through
		// to CWD walk-up rather than hard-failing, preserving usability.
	}

	// 3. Git worktree detection — resolve linked worktrees to main repo root.
	//    When running inside a git worktree (e.g. .claude/worktrees/feat-xxx/),
	//    git rev-parse --git-common-dir returns the main .git directory, whose
	//    parent is the main repo root. This ensures CLI commands find the main
	//    repo's .htmlgraph/ rather than a worktree-local copy.
	if dir := paths.ResolveViaGitCommonDir(""); dir != "" {
		return filepath.Join(dir, ".htmlgraph"), nil
	}

	// 4. Walk up from the process working directory.
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	for {
		candidate := filepath.Join(dir, ".htmlgraph")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", errors.New("no .htmlgraph directory found (run from within an htmlgraph project)")
}

// truncate shortens s to maxLen characters, appending "…" if cut.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}
