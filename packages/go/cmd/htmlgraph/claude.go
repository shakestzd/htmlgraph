package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// LaunchOpts controls how Claude Code is launched.
type LaunchOpts struct {
	// Mode is written to the launch marker (e.g. "go", "init", "continue", "default").
	Mode string
	// PluginDir, if non-empty, passes --plugin-dir to claude.
	PluginDir string
	// Resume adds --resume to claude args (for --continue mode).
	Resume bool
	// SystemPromptDir is the directory from which to load system-prompt.md.
	SystemPromptDir string
	// SystemPromptFile, if set, reads this file and appends it as system prompt.
	// Takes precedence over SystemPromptDir.
	SystemPromptFile string
	// PermissionMode, if set, passes --permission-mode to claude (e.g. "bypassPermissions").
	PermissionMode string
	// Name, if set, passes --name to claude for session naming.
	Name string
	// ExtraArgs are forwarded to the claude process.
	ExtraArgs []string
	// ProjectRoot is the absolute path to the project root (directory containing .htmlgraph/).
	// When set, Claude Code is started with this as the working directory, and path-sensitive
	// helpers (writeLaunchMarker, stubProjectHooks) anchor their paths here instead of CWD.
	ProjectRoot string
}

func claudeCmd() *cobra.Command {
	var dev, init_, continue_ bool

	cmd := &cobra.Command{
		Use:   "claude",
		Short: "Launch Claude Code with HtmlGraph",
		RunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case dev:
				return launchClaudeDev(args)
			case init_:
				return launchClaudeInit(args)
			case continue_:
				return launchClaudeContinue(args)
			default:
				return launchClaudeDefault(args)
			}
		},
	}
	cmd.Flags().BoolVar(&dev, "dev", false, "Launch with local Go plugin for development")
	cmd.Flags().BoolVar(&init_, "init", false, "Launch with marketplace plugin installation")
	cmd.Flags().BoolVar(&continue_, "continue", false, "Resume last session with marketplace plugin")
	cmd.AddCommand(yoloCmd())
	return cmd
}

func launchClaudeDev(extraArgs []string) error {
	// resolvePluginDir is defined in serve.go; returns "" on failure.
	pluginDir := resolvePluginDir()
	if pluginDir == "" {
		return fmt.Errorf("could not find plugin directory. The binary may not be installed at the expected location (packages/go-plugin/hooks/bin/htmlgraph)")
	}
	// Verify expected plugin structure.
	if _, err := os.Stat(filepath.Join(pluginDir, ".claude-plugin", "plugin.json")); os.IsNotExist(err) {
		return fmt.Errorf("plugin.json not found at %s. The binary may not be installed at the expected location (packages/go-plugin/hooks/bin/htmlgraph)",
			filepath.Join(pluginDir, ".claude-plugin", "plugin.json"))
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "hooks", "bin", "htmlgraph")); os.IsNotExist(err) {
		return fmt.Errorf("Go hooks binary not found at %s\nBuild with: packages/go-plugin/build.sh",
			filepath.Join(pluginDir, "hooks", "bin", "htmlgraph"))
	}

	// Resolve project root so paths are anchored correctly regardless of CWD.
	projectRoot := ""
	if htmlgraphDir, err := findHtmlgraphDir(); err == nil {
		projectRoot = filepath.Dir(htmlgraphDir)
	}

	// Disable marketplace plugin to prevent duplicate hooks.
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

	fmt.Printf("Launching Claude Code with Go plugin\n")
	fmt.Printf("  Plugin: %s\n", pluginDir)
	fmt.Printf("  Hooks: Go binary (near-zero cold start)\n")

	launchErr := launchClaude(LaunchOpts{
		Mode:            "go",
		PluginDir:       pluginDir,
		SystemPromptDir: pluginDir,
		ExtraArgs:       extraArgs,
		ProjectRoot:     projectRoot,
	})
	restoreFn()
	return launchErr
}

func launchClaudeInit(extraArgs []string) error {
	pluginDir := resolvePluginDir()
	projectRoot := ""
	if htmlgraphDir, err := findHtmlgraphDir(); err == nil {
		projectRoot = filepath.Dir(htmlgraphDir)
	}
	ensureHtmlgraphPlugin()
	fmt.Println("Launching Claude Code with marketplace plugin (init mode)...")
	return launchClaude(LaunchOpts{
		Mode:            "init",
		SystemPromptDir: pluginDir,
		ExtraArgs:       extraArgs,
		ProjectRoot:     projectRoot,
	})
}

func launchClaudeContinue(extraArgs []string) error {
	projectRoot := ""
	if htmlgraphDir, err := findHtmlgraphDir(); err == nil {
		projectRoot = filepath.Dir(htmlgraphDir)
	}
	fmt.Println("Resuming last Claude Code session (continue mode)...")
	return launchClaude(LaunchOpts{
		Mode:        "continue",
		Resume:      true,
		ExtraArgs:   extraArgs,
		ProjectRoot: projectRoot,
	})
}

func launchClaudeDefault(extraArgs []string) error {
	pluginDir := resolvePluginDir()
	projectRoot := ""
	if htmlgraphDir, err := findHtmlgraphDir(); err == nil {
		projectRoot = filepath.Dir(htmlgraphDir)
	}
	fmt.Println("Launching Claude Code (default mode)...")
	return launchClaude(LaunchOpts{
		Mode:            "default",
		SystemPromptDir: pluginDir,
		ExtraArgs:       extraArgs,
		ProjectRoot:     projectRoot,
	})
}

const htmlgraphMarketplaceRepo = "shakestzd/htmlgraph"

// ensureHtmlgraphPlugin registers the htmlgraph marketplace (if needed) and
// installs or updates the plugin.
func ensureHtmlgraphPlugin() {
	// Step 1: Register marketplace if not already known.
	fmt.Println("Registering htmlgraph marketplace...")
	exec.Command("claude", "plugin", "marketplace", "add", "htmlgraph",
		"--github", htmlgraphMarketplaceRepo).Run() //nolint:errcheck

	// Step 2: Try install, fall back to update.
	fmt.Println("Installing/updating htmlgraph plugin...")
	if out, err := exec.Command("claude", "plugin", "install", "htmlgraph@htmlgraph").CombinedOutput(); err != nil {
		if out2, err2 := exec.Command("claude", "plugin", "update", "htmlgraph").CombinedOutput(); err2 != nil {
			fmt.Fprintf(os.Stderr, "warning: plugin install: %s\nwarning: plugin update: %s\n", out, out2)
		}
	}
}

// launchClaude is the shared launcher used by all modes.
func launchClaude(opts LaunchOpts) error {
	writeLaunchMarker(opts.Mode, opts.ProjectRoot)

	// SystemPromptFile takes precedence over SystemPromptDir.
	var systemPrompt string
	if opts.SystemPromptFile != "" {
		if data, err := os.ReadFile(opts.SystemPromptFile); err == nil {
			systemPrompt = string(data)
		}
	} else {
		systemPrompt = loadSystemPrompt(opts.SystemPromptDir)
	}

	var claudeArgs []string
	if opts.Resume {
		claudeArgs = append(claudeArgs, "--resume")
	}
	if opts.PluginDir != "" {
		claudeArgs = append(claudeArgs, "--plugin-dir", opts.PluginDir)
	}
	if opts.PermissionMode != "" {
		claudeArgs = append(claudeArgs, "--permission-mode", opts.PermissionMode)
	}
	if opts.Name != "" {
		claudeArgs = append(claudeArgs, "--name", opts.Name)
	}
	if systemPrompt != "" {
		claudeArgs = append(claudeArgs, "--append-system-prompt", systemPrompt)
	}
	claudeArgs = append(claudeArgs, opts.ExtraArgs...)

	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found in PATH: %w", err)
	}

	c := exec.Command(claudePath, claudeArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	// Set working directory to project root so Claude starts in the right place,
	// even if this command is run from a subdirectory like packages/go.
	if opts.ProjectRoot != "" {
		c.Dir = opts.ProjectRoot
	}

	if err := c.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}

// stubProjectHooks replaces .claude/hooks/hooks.json with an empty stub
// to prevent Python hooks from firing alongside Go hooks.
// projectRoot, if non-empty, anchors the paths; otherwise CWD is used.
// Returns a restore function that must be called on exit.
func stubProjectHooks(projectRoot string) func() {
	projectHooks := ".claude/hooks/hooks.json"
	backupPath := ".claude/hooks/hooks.json.go-backup"
	if projectRoot != "" {
		projectHooks = filepath.Join(projectRoot, ".claude/hooks/hooks.json")
		backupPath = filepath.Join(projectRoot, ".claude/hooks/hooks.json.go-backup")
	}

	original, err := os.ReadFile(projectHooks)
	if err != nil {
		return func() {}
	}

	if err := os.WriteFile(backupPath, original, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not backup hooks.json: %v\n", err)
		return func() {}
	}

	if err := os.WriteFile(projectHooks, []byte(`{"hooks": {}}`), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not stub hooks.json: %v\n", err)
		return func() {}
	}

	return func() {
		if data, err := os.ReadFile(backupPath); err == nil {
			os.WriteFile(projectHooks, data, 0644) //nolint:errcheck
			os.Remove(backupPath)                   //nolint:errcheck
		}
	}
}

// loadSystemPrompt reads the system prompt from the plugin config directory.
func loadSystemPrompt(pluginDir string) string {
	if pluginDir == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(pluginDir, "config", "system-prompt.md"))
	if err != nil {
		return ""
	}
	return string(data)
}

// writeLaunchMarker writes .htmlgraph/.launch-mode for hooks to detect the launch mode.
// projectRoot must be non-empty; if it is empty the write is skipped to avoid
// polluting whatever directory the user happens to be in.
func writeLaunchMarker(mode, projectRoot string) {
	if projectRoot == "" {
		return // No project root — skip rather than polluting CWD
	}
	marker := map[string]any{
		"mode":      mode,
		"pid":       os.Getpid(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(marker)
	if err != nil {
		return
	}
	dir := filepath.Join(projectRoot, ".htmlgraph")
	os.MkdirAll(dir, 0755)                                       //nolint:errcheck
	os.WriteFile(filepath.Join(dir, ".launch-mode"), data, 0644) //nolint:errcheck
}
