// Package main is the entry point for the htmlgraph CLI.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shakestzd/htmlgraph/internal/agent"
	"github.com/shakestzd/htmlgraph/internal/paths"
	versionpkg "github.com/shakestzd/htmlgraph/internal/version"
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

	// Lazy session registration: every CLI command self-heals attribution
	// chains by detecting the agent and ensuring a session row exists.
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		// Skip commands that must work without .htmlgraph/.
		switch cmd.Name() {
		case "version", "help", "init", "build", "install-hooks", "setup", "setup-cli":
			return nil
		}
		// Skip hook subtree — hooks manage their own session lifecycle.
		for p := cmd; p != nil; p = p.Parent() {
			if p.Name() == "hook" {
				return nil
			}
		}
		// Degrade gracefully: commands must not fail because session
		// registration is unavailable.
		hgDir, err := findHtmlgraphDir()
		if err != nil {
			return nil
		}
		database, err := openDB(hgDir)
		if err != nil {
			return nil
		}
		defer database.Close()
		_, _ = agent.EnsureSession(database, filepath.Dir(hgDir))
		return nil
	}

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
	rootCmd.AddCommand(setupCmd())
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
	rootCmd.AddCommand(planCmdWithExtras())
	rootCmd.AddCommand(backfillCmd())
	rootCmd.AddCommand(reindexCmd())
	rootCmd.AddCommand(recommendCmd())
	rootCmd.AddCommand(ciCmd())
	rootCmd.AddCommand(helpCmd())
	rootCmd.AddCommand(claimCmd())
	rootCmd.AddCommand(agentInitCmd())
	rootCmd.AddCommand(pluginCmd())
	rootCmd.AddCommand(purgeSpikesCmd())
	rootCmd.AddCommand(traceCmd())
	rootCmd.AddCommand(graphCmd())
	rootCmd.AddCommand(queryCmd())

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
			if latest, newer, _ := versionpkg.CheckForUpdate(version); newer {
				fmt.Printf("Update available: v%s → run `htmlgraph build` or check https://github.com/shakestzd/htmlgraph/releases\n", latest)
			}
		},
	}
}

// findHtmlgraphDir locates the .htmlgraph directory by delegating to the
// shared paths.ResolveProjectDir resolver (--project-dir flag → CLAUDE_PROJECT_DIR
// env → git worktree detection → CWD walk-up) and appending ".htmlgraph".
func findHtmlgraphDir() (string, error) {
	paths.CleanupGlobalHint() // Remove stale global hint from older versions
	root, err := paths.ResolveProjectDir(paths.ProjectDirOptions{
		ExplicitDir: projectDirFlag,
	})
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".htmlgraph"), nil
}

// truncate shortens s to maxLen characters, appending "…" if cut.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}
