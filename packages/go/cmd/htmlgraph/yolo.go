package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/shakestzd/htmlgraph/packages/go/internal/htmlparse"
)

func yoloCmd() *cobra.Command {
	var dev, initMode, continueMode, noWorktree bool
	var permMode, trackID, featureID string

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
			switch {
			case dev:
				return launchYoloDev(trackID, featureID, noWorktree, args)
			case initMode:
				return launchYoloInit(trackID, featureID, args)
			case continueMode:
				return launchYoloContinue(args)
			default:
				return launchYoloDefault(permMode, trackID, featureID, noWorktree, args)
			}
		},
	}

	cmd.Flags().BoolVar(&dev, "dev", false, "Load plugin from local source (development mode)")
	cmd.Flags().BoolVar(&initMode, "init", false, "Initialize .htmlgraph/ then launch in YOLO mode")
	cmd.Flags().BoolVar(&continueMode, "continue", false, "Resume last YOLO session")
	cmd.Flags().BoolVar(&noWorktree, "no-worktree", false, "Skip worktree creation (run in project root)")
	cmd.Flags().StringVar(&permMode, "permission-mode", "bypassPermissions",
		"Permission mode (bypassPermissions, acceptEdits)")
	cmd.Flags().StringVar(&trackID, "track", "", "Track ID to work on (e.g., trk-3719d8f3)")
	cmd.Flags().StringVar(&featureID, "feature", "", "Feature ID to work on (e.g., feat-15c458aa)")
	return cmd
}

// yoloSessionName returns a unique session name for YOLO mode.
func yoloSessionName() string {
	return fmt.Sprintf("yolo-%s", time.Now().UTC().Format("20060102-150405"))
}

// resolveYoloPromptFile returns the path to yolo-prompt.md inside the plugin config dir.
// Returns empty string if not found (prompt injection is best-effort).
func resolveYoloPromptFile(pluginDir string) string {
	if pluginDir == "" {
		return ""
	}
	path := filepath.Join(pluginDir, "config", "yolo-prompt.md")
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}

// validateWorkItem checks that a track or feature HTML file exists in .htmlgraph/.
// Returns the validated ID and item type, or an error.
func validateWorkItem(trackID, featureID, projectRoot string) (id, kind string, err error) {
	htmlgraphDir := filepath.Join(projectRoot, ".htmlgraph")
	switch {
	case trackID != "":
		htmlFile := filepath.Join(htmlgraphDir, "tracks", trackID+".html")
		if _, statErr := os.Stat(htmlFile); os.IsNotExist(statErr) {
			return "", "", fmt.Errorf("track %s not found in .htmlgraph/", trackID)
		}
		return trackID, "track", nil
	case featureID != "":
		htmlFile := filepath.Join(htmlgraphDir, "features", featureID+".html")
		if _, statErr := os.Stat(htmlFile); os.IsNotExist(statErr) {
			return "", "", fmt.Errorf("feature %s not found in .htmlgraph/", featureID)
		}
		return featureID, "feature", nil
	default:
		return "", "", nil
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

	cleanup := func() {
		removeCmd := exec.Command("git", "-C", projectRoot, "worktree", "remove", "--force", worktreePath)
		removeCmd.Run() //nolint:errcheck
	}
	return worktreePath, cleanup, nil
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
func buildWorkItemPromptPrefix(id, kind string) string {
	return strings.Join([]string{
		"## Active Work Item",
		fmt.Sprintf("You are working on: %s", id),
		"All work in this session must be attributed to this item.",
		"",
	}, "\n")
}

// buildYoloSystemPrompt reads the yolo prompt file and prepends the work item header.
// If promptFile is empty or unreadable, falls back to the header alone.
func buildYoloSystemPrompt(promptFile, id, kind string) string {
	var sb strings.Builder
	if id != "" {
		sb.WriteString(buildWorkItemPromptPrefix(id, kind))
	}
	if promptFile != "" {
		if data, err := os.ReadFile(promptFile); err == nil {
			sb.Write(data)
		}
	}
	return sb.String()
}

// launchYoloPlanningMode launches Claude in planning mode (no bypass permissions)
// when no --track or --feature is provided. Prints guidance before launching.
func launchYoloPlanningMode(pluginDir, projectRoot string, extraArgs []string) error {
	fmt.Println("No --track or --feature specified.")
	fmt.Println("Launching in planning mode to help you create a track or feature first.")
	fmt.Println("Once you have a track/feature, restart with:")
	fmt.Println("  htmlgraph yolo --track <track-id>")
	fmt.Println("  htmlgraph yolo --feature <feature-id>")
	fmt.Println()
	return launchClaude(LaunchOpts{
		Mode:            "yolo-planning",
		SystemPromptDir: pluginDir,
		ExtraArgs:       extraArgs,
		ProjectRoot:     projectRoot,
	})
}

func launchYoloDefault(permMode, trackID, featureID string, noWorktree bool, extraArgs []string) error {
	pluginDir := resolvePluginDir()
	projectRoot := ""
	if htmlgraphDir, err := findHtmlgraphDir(); err == nil {
		projectRoot = filepath.Dir(htmlgraphDir)
	}

	// No work item provided — fall back to planning mode.
	if trackID == "" && featureID == "" {
		return launchYoloPlanningMode(pluginDir, projectRoot, extraArgs)
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
	yoloPromptFile := resolveYoloPromptFile(pluginDir)
	systemPromptContent := buildYoloSystemPrompt(yoloPromptFile, id, kind)

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
	if _, err := tmpFile.WriteString(systemPromptContent); err != nil {
		return fmt.Errorf("could not write temp prompt file: %w", err)
	}
	tmpFile.Close()

	return launchClaude(LaunchOpts{
		Mode:             "yolo",
		PluginDir:        pluginDir,
		SystemPromptFile: tmpFile.Name(),
		PermissionMode:   permMode,
		Name:             sessionName,
		ExtraArgs:        extraArgs,
		ProjectRoot:      workDir,
	})
}

func launchYoloDev(trackID, featureID string, noWorktree bool, extraArgs []string) error {
	pluginDir := resolvePluginDir()
	if pluginDir == "" {
		return fmt.Errorf("could not find plugin directory. The binary may not be installed at the expected location (packages/go-plugin/hooks/bin/htmlgraph)")
	}
	if _, err := os.Stat(filepath.Join(pluginDir, ".claude-plugin", "plugin.json")); os.IsNotExist(err) {
		return fmt.Errorf("plugin.json not found at %s. The binary may not be installed at the expected location (packages/go-plugin/hooks/bin/htmlgraph)",
			filepath.Join(pluginDir, ".claude-plugin", "plugin.json"))
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "hooks", "bin", "htmlgraph")); os.IsNotExist(err) {
		return fmt.Errorf("Go hooks binary not found at %s\nBuild with: packages/go-plugin/build.sh",
			filepath.Join(pluginDir, "hooks", "bin", "htmlgraph"))
	}

	projectRoot := ""
	if htmlgraphDir, err := findHtmlgraphDir(); err == nil {
		projectRoot = filepath.Dir(htmlgraphDir)
	}

	// No work item provided — fall back to planning mode.
	if trackID == "" && featureID == "" {
		return launchYoloPlanningMode(pluginDir, projectRoot, extraArgs)
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

	fmt.Println("Disabling marketplace htmlgraph plugin...")
	for _, scope := range []string{"htmlgraph@htmlgraph", "htmlgraph@local-marketplace"} {
		exec.Command("claude", "plugin", "disable", scope).Run() //nolint:errcheck
	}

	restoreFn := stubProjectHooks(projectRoot)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		restoreFn()
		os.Exit(0)
	}()

	sessionName := yoloSessionName()
	yoloPromptFile := resolveYoloPromptFile(pluginDir)
	systemPromptContent := buildYoloSystemPrompt(yoloPromptFile, id, kind)

	fmt.Printf("Launching Claude Code in YOLO dev mode...\n")
	fmt.Printf("  Plugin: %s\n", pluginDir)
	fmt.Printf("  Session: %s\n", sessionName)
	fmt.Printf("  Work item: %s\n", id)

	tmpFile, err := os.CreateTemp("", "yolo-prompt-*.md")
	if err != nil {
		restoreFn()
		return fmt.Errorf("could not create temp prompt file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(systemPromptContent); err != nil {
		restoreFn()
		return fmt.Errorf("could not write temp prompt file: %w", err)
	}
	tmpFile.Close()

	launchErr := launchClaude(LaunchOpts{
		Mode:             "yolo-dev",
		PluginDir:        pluginDir,
		SystemPromptFile: tmpFile.Name(),
		PermissionMode:   "bypassPermissions",
		Name:             sessionName,
		ExtraArgs:        extraArgs,
		ProjectRoot:      workDir,
	})
	restoreFn()
	return launchErr
}

func launchYoloInit(trackID, featureID string, extraArgs []string) error {
	// Initialize .htmlgraph/ first.
	if err := runInit(nil, nil); err != nil {
		return fmt.Errorf("init failed: %w", err)
	}
	fmt.Println()
	return launchYoloDefault("bypassPermissions", trackID, featureID, false, extraArgs)
}

func launchYoloContinue(extraArgs []string) error {
	projectRoot := ""
	if htmlgraphDir, err := findHtmlgraphDir(); err == nil {
		projectRoot = filepath.Dir(htmlgraphDir)
	}

	fmt.Println("Resuming last YOLO session...")

	return launchClaude(LaunchOpts{
		Mode:           "yolo-continue",
		Resume:         true,
		PermissionMode: "bypassPermissions",
		ExtraArgs:      extraArgs,
		ProjectRoot:    projectRoot,
	})
}
