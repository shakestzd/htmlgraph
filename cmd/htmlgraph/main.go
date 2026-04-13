// Package main is the entry point for the htmlgraph CLI.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shakestzd/htmlgraph/internal/agent"
	"github.com/shakestzd/htmlgraph/internal/paths"
	"github.com/shakestzd/htmlgraph/internal/registry"
	versionpkg "github.com/shakestzd/htmlgraph/internal/version"
	"github.com/spf13/cobra"
)

// version is set at build time via ldflags.
var version = "dev"

// projectDirFlag holds the value of the --project-dir persistent flag.
var projectDirFlag string

// getGitRemoteURLFn is a package-level indirection for paths.GetGitRemoteURL
// so tests can stub it and count invocations. Production code calls the real
// implementation.
var getGitRemoteURLFn = paths.GetGitRemoteURL

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

	// Lazy session registration + passive project registration: every CLI
	// command self-heals attribution chains and upserts the current project
	// into the cross-project registry.
	rootCmd.PersistentPreRunE = persistentPreRunE

	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(statuslineCmd())
	rootCmd.AddCommand(serveCmd())
	rootCmd.AddCommand(serveChildCmd())
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
	rootCmd.AddCommand(sweepCmd())
	rootCmd.AddCommand(cleanupCmd())
	rootCmd.AddCommand(migrateCmd())
	rootCmd.AddCommand(recommendCmd())
	rootCmd.AddCommand(ciCmd())
	rootCmd.AddCommand(helpCmd())
	rootCmd.AddCommand(claimCmd())
	rootCmd.AddCommand(agentInitCmd())
	rootCmd.AddCommand(pluginCmd())
	rootCmd.AddCommand(purgeSpikesCmd())
	rootCmd.AddCommand(projectsCmd())
	rootCmd.AddCommand(traceCmd())
	rootCmd.AddCommand(graphCmd())
	rootCmd.AddCommand(queryCmd())
	rootCmd.AddCommand(upgradeCmd())

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

// persistentPreRunE is attached to rootCmd and runs before every command. It
// performs two side-effects: (1) ensures a session row exists for the current
// agent attribution chain, and (2) upserts the current project into the
// cross-project registry at ~/.local/share/htmlgraph/projects.json. Both
// operations degrade gracefully — registration failures never block a CLI
// command from running.
func persistentPreRunE(cmd *cobra.Command, _ []string) error {
	// Skip commands that must work without .htmlgraph/.
	switch cmd.Name() {
	case "version", "help", "init", "build", "install-hooks", "setup", "setup-cli", "projects", "upgrade", "update":
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
	projectDir := filepath.Dir(hgDir)
	if database, dberr := openDB(hgDir); dberr == nil {
		_, _ = agent.EnsureSession(database, projectDir)
		database.Close()
	}
	// Registry upsert — silent, cached git remote lookup.
	if reg, regErr := registry.Load(registry.DefaultPath()); regErr == nil {
		var cachedRemote string
		for _, e := range reg.List() {
			if filepath.Clean(e.ProjectDir) == filepath.Clean(projectDir) {
				cachedRemote = e.GitRemoteURL
				break
			}
		}
		remoteURL := cachedRemote
		if remoteURL == "" {
			remoteURL = getGitRemoteURLFn(projectDir)
		}
		reg.Upsert(projectDir, filepath.Base(projectDir), remoteURL)
		// Opportunistic worktree cleanup: registry entries created by
		// older binaries (before findHtmlgraphDir started resolving
		// linked worktrees to their main repo) persist as duplicate
		// project cards in the doorway. Drop any entry whose path is
		// inside a linked worktree of a registered main repo.
		reg.DropLinkedWorktrees(paths.ResolveViaGitCommonDir)
		_ = reg.Save()
	}
	return nil
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

// printProjectHeaderIfDifferent prints a one-line "Project: <path>" header
// to stdout when the resolved project root differs from the current working
// directory. Project-level mutation commands (migrate, sweep, ingest) call
// this before touching data so the user can tell at a glance when env-var
// resolution (HTMLGRAPH_PROJECT_DIR / CLAUDE_PROJECT_DIR) or worktree
// detection is pointing them at a different project than the one they're
// sitting in. No-op when the user is already in the resolved project —
// keeps normal usage silent.
func printProjectHeaderIfDifferent(htmlgraphDir string) {
	projectRoot := filepath.Dir(htmlgraphDir)
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	// Resolve symlinks on both sides so /var/... and /private/var/... compare
	// equal on macOS and worktrees that traverse symlinked paths don't
	// trigger a false-positive "outside project" header.
	resolvedProject := resolveForCompare(projectRoot)
	resolvedWD := resolveForCompare(wd)
	if resolvedWD == resolvedProject {
		return
	}
	// Silent when CWD is inside the project (worktrees, subdirs).
	if rel, relErr := filepath.Rel(resolvedProject, resolvedWD); relErr == nil &&
		!strings.HasPrefix(rel, "..") && rel != "." {
		return
	}
	fmt.Fprintf(os.Stderr, "Project: %s  (CWD: %s — use --project-dir to override)\n",
		projectRoot, wd)
}

// resolveForCompare returns the absolute, symlink-resolved, cleaned path for
// directory comparison. Falls back to the absolute path when symlink
// resolution fails (e.g. the path does not exist).
func resolveForCompare(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(resolved)
	}
	return filepath.Clean(abs)
}

// truncate shortens s to maxLen characters, appending "…" if cut.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}
