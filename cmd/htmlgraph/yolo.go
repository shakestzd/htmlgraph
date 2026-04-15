package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/shakestzd/htmlgraph/internal/htmlparse"
	"github.com/shakestzd/htmlgraph/internal/workitem"
)

func yoloCmd() *cobra.Command {
	var dev, initMode, continueMode, noWorktree, tmux bool
	var permMode, trackID, featureID, resumeID string

	cmd := &cobra.Command{
		Use:   "yolo",
		Short: "Launch Claude Code in autonomous YOLO mode with development guardrails",
		Long: `Launches Claude Code with bypassPermissions and enforced quality guardrails.

YOLO mode removes permission prompts but enforces code quality at every step:
  - Mandatory TDD workflow (tests before implementation)
  - Quality gate checks before every commit
  - Budget limits to keep features focused
  - Worktree-per-feature isolation

Each session is auto-named with a timestamp for easy identification.

Requires --track or --feature to identify the work item for attribution.
Without either flag, launches in planning mode to help you create one first.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Tmux wrap must happen before any side-effecting work.
			// When --tmux is set and we are not already inside tmux, this
			// replaces the current process with: tmux new-session -A -s htmlgraph-yolo -- <argv without --tmux>
			// and never returns. If tmux is missing, an error is returned.
			// If we are already inside tmux (TMUX env set), this is a no-op.
			_ = tmux // flag is consumed via os.Args inspection in maybeTmuxWrap
			if err := maybeTmuxWrap("htmlgraph-yolo"); err != nil {
				return err
			}
			switch {
			case dev:
				return launchYoloDev(trackID, featureID, noWorktree, resumeID, args)
			case initMode:
				return launchYoloInit(trackID, featureID, resumeID, args)
			case continueMode:
				return launchYoloContinue(args)
			default:
				return launchYoloDefault(permMode, trackID, featureID, noWorktree, resumeID, args)
			}
		},
	}

	cmd.Flags().BoolVar(&dev, "dev", false, "Load plugin from local source (development mode)")
	cmd.Flags().BoolVar(&initMode, "init", false, "Initialize .htmlgraph/ then launch in YOLO mode")
	cmd.Flags().BoolVar(&continueMode, "continue", false, "Resume last YOLO session")
	cmd.Flags().BoolVar(&noWorktree, "no-worktree", false, "Skip worktree creation (run in project root)")
	cmd.Flags().BoolVar(&tmux, "tmux", false, "Wrap yolo in a tmux session named 'htmlgraph-yolo' (survives disconnects; reattaches on re-run)")
	cmd.Flags().StringVar(&permMode, "permission-mode", "bypassPermissions",
		"Permission mode (bypassPermissions, acceptEdits)")
	cmd.Flags().StringVar(&trackID, "track", "", "Track ID to work on (e.g., trk-3719d8f3)")
	cmd.Flags().StringVar(&featureID, "feature", "", "Feature ID to work on (e.g., feat-15c458aa)")
	cmd.Flags().StringVar(&resumeID, "resume", "", "Resume a specific Claude Code session by ID")
	return cmd
}

// yoloSessionName returns a unique session name for YOLO mode.
func yoloSessionName() string {
	return fmt.Sprintf("yolo-%s", time.Now().UTC().Format("20060102-150405"))
}

// validateWorkItem checks that a track or feature HTML file exists in .htmlgraph/.
// Returns the validated ID and item type, or an error.
func validateWorkItem(trackID, featureID, projectRoot string) (id, kind string, err error) {
	htmlgraphDir := filepath.Join(projectRoot, ".htmlgraph")
	switch {
	case trackID != "":
		htmlFile := filepath.Join(htmlgraphDir, "tracks", trackID+".html")
		if _, statErr := os.Stat(htmlFile); os.IsNotExist(statErr) {
			return "", "", workitem.ErrNotFound("track", trackID)
		}
		return trackID, "track", nil
	case featureID != "":
		htmlFile := filepath.Join(htmlgraphDir, "features", featureID+".html")
		if _, statErr := os.Stat(htmlFile); os.IsNotExist(statErr) {
			return "", "", workitem.ErrNotFound("feature", featureID)
		}
		return featureID, "feature", nil
	default:
		return "", "", nil
	}
}

// excludeHtmlgraphFromWorktree adds .htmlgraph/ to the worktree's local git exclude file.
// In git worktrees, .git is a file (not a directory) containing "gitdir: <path>".
// The actual git metadata is at the gitdir path, so the exclude file is at gitdir/info/exclude.
// Best-effort: errors are printed but do not abort.
func excludeHtmlgraphFromWorktree(worktreePath string) {
	gitFile := filepath.Join(worktreePath, ".git")
	content, err := os.ReadFile(gitFile)
	if err != nil {
		fmt.Printf("  Warning: could not read .git file for exclude setup: %v\n", err)
		return
	}

	// Parse the gitdir from the .git file
	gitdirLine := strings.TrimSpace(string(content))
	gitdir := strings.TrimPrefix(gitdirLine, "gitdir: ")
	if gitdir == gitdirLine {
		// No gitdir prefix found — not a worktree
		return
	}

	excludePath := filepath.Join(gitdir, "info", "exclude")

	// Ensure info/ directory exists
	if err := os.MkdirAll(filepath.Dir(excludePath), 0755); err != nil {
		fmt.Printf("  Warning: could not create exclude directory: %v\n", err)
		return
	}

	// Append .htmlgraph/ to the exclude file
	f, err := os.OpenFile(excludePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("  Warning: could not open exclude file: %v\n", err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString("\n.htmlgraph/\n"); err != nil {
		fmt.Printf("  Warning: could not write to exclude file: %v\n", err)
	}
}

// createFeatureWorktree creates a git worktree at .claude/worktrees/<featureID> on branch
// yolo-<featureID>. If the worktree path already exists it is reused. Returns the worktree
// path and a cleanup function that removes the worktree on error.
func createFeatureWorktree(featureID, projectRoot string) (string, func(), error) {
	worktreePath := filepath.Join(projectRoot, ".claude", "worktrees", featureID)
	branchName := "yolo-" + featureID
	noop := func() {}

	// If path already exists, reuse it — the worktree was created in a prior run.
	if _, err := os.Stat(worktreePath); err == nil {
		fmt.Printf("  Worktree: %s (reusing existing)\n", worktreePath)
		return worktreePath, noop, nil
	}

	// Ensure the parent directory exists.
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return "", noop, fmt.Errorf("could not create worktrees directory: %w", err)
	}

	cmd := exec.Command("git", "-C", projectRoot, "worktree", "add", worktreePath, "-b", branchName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", noop, fmt.Errorf("git worktree add failed: %w\n%s", err, out)
	}

	fmt.Printf("  Worktree: %s (branch: %s)\n", worktreePath, branchName)

	// Exclude .htmlgraph/ from git status — all CLI ops route to main via HTMLGRAPH_PROJECT_DIR.
	excludeHtmlgraphFromWorktree(worktreePath)

	// Reindex the worktree SQLite so it reflects current HTML state.
	reindexWorktree(worktreePath)

	cleanup := func() {
		removeCmd := exec.Command("git", "-C", projectRoot, "worktree", "remove", "--force", worktreePath)
		removeCmd.Run() //nolint:errcheck
	}
	return worktreePath, cleanup, nil
}

// createTrackWorktree creates a git worktree at .claude/worktrees/<trackID> on branch
// trk-<trackID>. If the worktree path already exists it is reused. Returns the worktree
// path and a cleanup function that removes the worktree on error.
func createTrackWorktree(trackID, projectRoot string) (string, func(), error) {
	worktreePath := filepath.Join(projectRoot, ".claude", "worktrees", trackID)
	branchName := trackID // Track worktrees use branch name trk-abc123, not yolo-trk-abc123
	noop := func() {}

	// If path already exists, reuse it — the worktree was created in a prior run.
	if _, err := os.Stat(worktreePath); err == nil {
		fmt.Printf("  Worktree: %s (reusing existing)\n", worktreePath)
		return worktreePath, noop, nil
	}

	// Ensure the parent directory exists.
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return "", noop, fmt.Errorf("could not create worktrees directory: %w", err)
	}

	cmd := exec.Command("git", "-C", projectRoot, "worktree", "add", worktreePath, "-b", branchName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", noop, fmt.Errorf("git worktree add failed: %w\n%s", err, out)
	}

	fmt.Printf("  Worktree: %s (branch: %s)\n", worktreePath, branchName)

	// Exclude .htmlgraph/ from git status — all CLI ops route to main via HTMLGRAPH_PROJECT_DIR.
	excludeHtmlgraphFromWorktree(worktreePath)

	// Reindex the worktree SQLite so it reflects current HTML state.
	reindexWorktree(worktreePath)

	cleanup := func() {
		removeCmd := exec.Command("git", "-C", projectRoot, "worktree", "remove", "--force", worktreePath)
		removeCmd.Run() //nolint:errcheck
	}
	return worktreePath, cleanup, nil
}

// reindexWorktree runs `htmlgraph reindex` in the given worktree directory so
// the worktree's SQLite cache is current before Claude launches. Best-effort:
// failures are printed but do not abort worktree setup.
func reindexWorktree(worktreeDir string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	exe, err := os.Executable()
	if err != nil {
		fmt.Printf("  Warning: could not determine executable path for reindex: %v\n", err)
		return
	}
	reindexCmd := exec.CommandContext(ctx, exe, "reindex")
	reindexCmd.Dir = worktreeDir
	if err := reindexCmd.Run(); err != nil {
		fmt.Printf("  Warning: reindex in worktree failed: %v\n", err)
	}
}

// createAgentWorktree creates a git worktree branching from the track branch
// (not main). The branch is named agent-{trackID}-{taskName} (flat name due to Git's
// ref hierarchy constraints) and the worktree is placed at .claude/worktrees/{trackID}/agent-{taskName}.
//
// The track branch must exist. If the worktree path already exists, it is reused.
// Returns the worktree path and a cleanup function.
func createAgentWorktree(trackID, taskName, projectRoot string) (string, func(), error) {
	agentBranch := "agent-" + trackID + "-" + taskName
	worktreePath := filepath.Join(projectRoot, ".claude", "worktrees", trackID, "agent-"+taskName)
	noop := func() {}

	// If path already exists, reuse.
	if _, err := os.Stat(worktreePath); err == nil {
		fmt.Printf("  Agent worktree: %s (reusing existing)\n", worktreePath)
		return worktreePath, noop, nil
	}

	// Verify track branch exists.
	if err := exec.Command("git", "-C", projectRoot, "rev-parse", "--verify", trackID).Run(); err != nil {
		return "", noop, fmt.Errorf("track branch %s not found: create track worktree first with htmlgraph yolo --track %s", trackID, trackID)
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return "", noop, fmt.Errorf("could not create agent worktrees directory: %w", err)
	}

	// Create worktree branching from track branch.
	cmd := exec.Command("git", "-C", projectRoot, "worktree", "add", worktreePath, "-b", agentBranch, trackID)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", noop, fmt.Errorf("git worktree add failed: %w\n%s", err, out)
	}

	fmt.Printf("  Agent worktree: %s (branch: %s, from: %s)\n", worktreePath, agentBranch, trackID)

	cleanup := func() {
		removeCmd := exec.Command("git", "-C", projectRoot, "worktree", "remove", "--force", worktreePath)
		removeCmd.Run() //nolint:errcheck
	}
	return worktreePath, cleanup, nil
}

// mergeAgentToTrack merges an agent branch back into its parent track branch
// and removes the agent worktree. This is the cleanup step after agent completion.
func mergeAgentToTrack(trackID, taskName, projectRoot string) error {
	agentBranch := "agent-" + trackID + "-" + taskName
	worktreePath := filepath.Join(projectRoot, ".claude", "worktrees", trackID, "agent-"+taskName)

	// Checkout track branch first
	checkoutCmd := exec.Command("git", "-C", projectRoot, "checkout", trackID)
	if out, err := checkoutCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("checkout track branch failed: %w\n%s", err, out)
	}

	// Merge agent branch into track branch
	mergeCmd := exec.Command("git", "-C", projectRoot, "merge", "--no-ff", agentBranch,
		"-m", fmt.Sprintf("feat: merge agent-%s into %s", taskName, trackID))

	if out, err := mergeCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("merge agent to track failed: %w\n%s", err, out)
	}

	// Remove the agent worktree
	removeCmd := exec.Command("git", "-C", projectRoot, "worktree", "remove", "--force", worktreePath)
	removeCmd.Run() //nolint:errcheck

	// Delete the agent branch
	deleteCmd := exec.Command("git", "-C", projectRoot, "branch", "-d", agentBranch)
	deleteCmd.Run() //nolint:errcheck

	return nil
}

// resolveTrackForFeature reads a feature HTML file and returns its data-track-id attribute.
// If the feature file doesn't exist or has no track ID, returns empty string.
func resolveTrackForFeature(featureID, projectRoot string) string {
	featureFile := filepath.Join(projectRoot, ".htmlgraph", "features", featureID+".html")
	node, err := htmlparse.ParseFile(featureFile)
	if err != nil {
		// File not found or parse error — gracefully return empty
		return ""
	}
	return node.TrackID
}

// buildWorkItemPromptPrefix returns the work item header to prepend to the yolo prompt.
func buildWorkItemPromptPrefix(id, _ string) string {
	return strings.Join([]string{
		"## Active Work Item",
		fmt.Sprintf("You are working on: %s", id),
		"All work in this session must be attributed to this item.",
		"",
	}, "\n")
}

// buildYoloSystemPrompt prepends the work item header to the embedded yolo prompt.
func buildYoloSystemPrompt(id, kind string) string {
	var sb strings.Builder
	if id != "" {
		sb.WriteString(buildWorkItemPromptPrefix(id, kind))
	}
	sb.WriteString(yoloPromptContent)
	return sb.String()
}

// launchYoloPlanningMode launches Claude in planning mode (no bypass permissions)
// when no --track or --feature is provided. Prints guidance before launching.
func launchYoloPlanningMode(projectRoot string, extraArgs []string) error {
	fmt.Println("No --track or --feature specified.")
	fmt.Println("Launching in planning mode to help you create a track or feature first.")
	fmt.Println("Once you have a track/feature, restart with:")
	fmt.Println("  htmlgraph yolo --track <track-id>")
	fmt.Println("  htmlgraph yolo --feature <feature-id>")
	fmt.Println()
	return launchClaude(LaunchOpts{
		Mode:               "yolo-planning",
		InjectSystemPrompt: true,
		ExtraArgs:          extraArgs,
		ProjectRoot:        projectRoot,
	})
}

func launchYoloDefault(permMode, trackID, featureID string, noWorktree bool, resumeID string, extraArgs []string) error {
	projectRoot := ""
	if htmlgraphDir, err := findHtmlgraphDir(); err == nil {
		projectRoot = filepath.Dir(htmlgraphDir)
	}

	ensurePluginOnLaunch()

	// No work item provided — fall back to planning mode.
	if trackID == "" && featureID == "" {
		return launchYoloPlanningMode(projectRoot, extraArgs)
	}

	// Validate the provided work item exists.
	id, kind, err := validateWorkItem(trackID, featureID, projectRoot)
	if err != nil {
		return err
	}

	// Create a worktree for isolation (skip for --no-worktree).
	workDir := projectRoot
	if !noWorktree && projectRoot != "" {
		// If --track is provided, create a track worktree
		if trackID != "" {
			worktreePath, cleanup, wtErr := createTrackWorktree(trackID, projectRoot)
			if wtErr != nil {
				return wtErr
			}
			_ = cleanup // only used on error; worktree persists for the session
			workDir = worktreePath
		} else if featureID != "" {
			// If --feature is provided, check if it has a parent track
			resolvedTrackID := resolveTrackForFeature(featureID, projectRoot)
			if resolvedTrackID != "" {
				// Feature has a parent track — use the track worktree
				worktreePath, cleanup, wtErr := createTrackWorktree(resolvedTrackID, projectRoot)
				if wtErr != nil {
					return wtErr
				}
				_ = cleanup // only used on error; worktree persists for the session
				workDir = worktreePath
			} else {
				// Feature has no parent track — use the feature worktree
				worktreePath, cleanup, wtErr := createFeatureWorktree(featureID, projectRoot)
				if wtErr != nil {
					return wtErr
				}
				_ = cleanup // only used on error; worktree persists for the session
				workDir = worktreePath
			}
		}
	}

	sessionName := yoloSessionName()
	yoloPrompt := buildYoloSystemPrompt(id, kind)

	fmt.Printf("Launching Claude Code in YOLO mode (%s)...\n", permMode)
	fmt.Printf("  Session: %s\n", sessionName)
	fmt.Printf("  Work item: %s\n", id)

	// Write the combined prompt to a temp file so launchClaude can pass it via
	// --append-system-prompt without needing a new field.
	tmpFile, err := os.CreateTemp("", "yolo-prompt-*.md")
	if err != nil {
		return fmt.Errorf("could not create temp prompt file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(yoloPrompt); err != nil {
		return fmt.Errorf("could not write temp prompt file: %w", err)
	}
	tmpFile.Close()

	return launchClaude(LaunchOpts{
		Mode:             "yolo",
		ResumeID:         resumeID,
		SystemPromptFile: tmpFile.Name(),
		PermissionMode:   permMode,
		Name:             sessionName,
		ExtraArgs:        extraArgs,
		ProjectRoot:      workDir,
		HtmlgraphRoot:    projectRoot,
	})
}

func launchYoloDev(trackID, featureID string, noWorktree bool, resumeID string, extraArgs []string) error {
	// Dev mode resolves the plugin from local source, NOT the marketplace.
	pluginDir := resolveProjectPluginDir()
	if pluginDir == "" {
		return fmt.Errorf("could not find plugin/ directory relative to project root. Run from the project directory containing .htmlgraph/ and plugin/")
	}
	if _, err := os.Stat(filepath.Join(pluginDir, ".claude-plugin", "plugin.json")); os.IsNotExist(err) {
		return fmt.Errorf("plugin.json not found at %s",
			filepath.Join(pluginDir, ".claude-plugin", "plugin.json"))
	}
	if _, err := exec.LookPath("htmlgraph"); err != nil {
		return fmt.Errorf("htmlgraph binary not found on PATH\nBuild with: htmlgraph build (or plugin/build.sh)")
	}

	projectRoot := ""
	if htmlgraphDir, err := findHtmlgraphDir(); err == nil {
		projectRoot = filepath.Dir(htmlgraphDir)
	}

	// No work item provided — fall back to planning mode.
	if trackID == "" && featureID == "" {
		return launchYoloPlanningMode(projectRoot, extraArgs)
	}

	// Validate the provided work item exists.
	id, kind, err := validateWorkItem(trackID, featureID, projectRoot)
	if err != nil {
		return err
	}

	// Create a worktree for isolation (skip for --no-worktree).
	workDir := projectRoot
	if !noWorktree && projectRoot != "" {
		// If --track is provided, create a track worktree
		if trackID != "" {
			worktreePath, cleanup, wtErr := createTrackWorktree(trackID, projectRoot)
			if wtErr != nil {
				return wtErr
			}
			_ = cleanup // only used on error; worktree persists for the session
			workDir = worktreePath
		} else if featureID != "" {
			// If --feature is provided, check if it has a parent track
			resolvedTrackID := resolveTrackForFeature(featureID, projectRoot)
			if resolvedTrackID != "" {
				// Feature has a parent track — use the track worktree
				worktreePath, cleanup, wtErr := createTrackWorktree(resolvedTrackID, projectRoot)
				if wtErr != nil {
					return wtErr
				}
				_ = cleanup // only used on error; worktree persists for the session
				workDir = worktreePath
			} else {
				// Feature has no parent track — use the feature worktree
				worktreePath, cleanup, wtErr := createFeatureWorktree(featureID, projectRoot)
				if wtErr != nil {
					return wtErr
				}
				_ = cleanup // only used on error; worktree persists for the session
				workDir = worktreePath
			}
		}
	}

	// Nuke marketplace plugin so it can't shadow the --plugin-dir agents/skills.
	removeMarketplaceHtmlgraph()

	sessionName := yoloSessionName()
	yoloPrompt := buildYoloSystemPrompt(id, kind)

	fmt.Printf("Launching Claude Code in YOLO dev mode...\n")
	fmt.Printf("  Plugin: %s\n", pluginDir)
	fmt.Printf("  Session: %s\n", sessionName)
	fmt.Printf("  Work item: %s\n", id)

	tmpFile, err := os.CreateTemp("", "yolo-prompt-*.md")
	if err != nil {
		return fmt.Errorf("could not create temp prompt file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(yoloPrompt); err != nil {
		return fmt.Errorf("could not write temp prompt file: %w", err)
	}
	tmpFile.Close()

	return launchClaude(LaunchOpts{
		Mode:             "yolo-dev",
		PluginDir:        pluginDir,
		ResumeID:         resumeID,
		SystemPromptFile: tmpFile.Name(),
		PermissionMode:   "bypassPermissions",
		Name:             sessionName,
		ExtraArgs:        extraArgs,
		ProjectRoot:      workDir,
		HtmlgraphRoot:    projectRoot,
	})
}

func launchYoloInit(trackID, featureID string, resumeID string, extraArgs []string) error {
	// Initialize .htmlgraph/ first.
	if err := runInit(nil, nil); err != nil {
		return fmt.Errorf("init failed: %w", err)
	}
	fmt.Println()
	return launchYoloDefault("bypassPermissions", trackID, featureID, false, resumeID, extraArgs)
}

func launchYoloContinue(extraArgs []string) error {
	projectRoot := ""
	if htmlgraphDir, err := findHtmlgraphDir(); err == nil {
		projectRoot = filepath.Dir(htmlgraphDir)
	}

	ensurePluginOnLaunch()
	fmt.Println("Resuming last YOLO session...")

	return launchClaude(LaunchOpts{
		Mode:           "yolo-continue",
		Resume:         true,
		PermissionMode: "bypassPermissions",
		ExtraArgs:      extraArgs,
		ProjectRoot:    projectRoot,
	})
}
