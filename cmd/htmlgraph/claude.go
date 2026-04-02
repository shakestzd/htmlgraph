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

// devModeBackup records the state we swapped out during dev mode so we can
// restore it after the session (or after a crash on next startup).
type devModeBackup struct {
	InstallPath    string `json:"installPath"`
	BackupPath     string `json:"backupPath"`
	WasEnabled     bool   `json:"wasEnabled"`
	PluginKey      string `json:"pluginKey"`
	HadInstallPath bool   `json:"hadInstallPath"`
}

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
	// helpers (writeLaunchMarker, etc.) anchor their paths here instead of CWD.
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
		return fmt.Errorf("could not find plugin directory. The binary may not be installed at the expected location (plugin/hooks/bin/htmlgraph)")
	}
	// Verify expected plugin structure.
	if _, err := os.Stat(filepath.Join(pluginDir, ".claude-plugin", "plugin.json")); os.IsNotExist(err) {
		return fmt.Errorf("plugin.json not found at %s. The binary may not be installed at the expected location (plugin/hooks/bin/htmlgraph)",
			filepath.Join(pluginDir, ".claude-plugin", "plugin.json"))
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "hooks", "bin", "htmlgraph")); os.IsNotExist(err) {
		return fmt.Errorf("Go hooks binary not found at %s\nBuild with: plugin/build.sh",
			filepath.Join(pluginDir, "hooks", "bin", "htmlgraph"))
	}

	// Resolve project root so paths are anchored correctly regardless of CWD.
	projectRoot := ""
	if htmlgraphDir, err := findHtmlgraphDir(); err == nil {
		projectRoot = filepath.Dir(htmlgraphDir)
	}

	// Clean up any leftover dev-mode state from a previous crash.
	cleanupStaleDev(projectRoot)

	// Set up marketplace symlink so plugin hooks fire natively.
	restoreFn, err := setupDevModeSymlink(pluginDir, projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not set up plugin symlink (%v); falling back to --plugin-dir\n", err)
		// Fallback: use --plugin-dir (hooks won't fire, but at least agents/skills work).
		return launchClaudeDevFallback(pluginDir, projectRoot, extraArgs)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		restoreFn()
		os.Exit(0)
	}()

	fmt.Printf("Launching Claude Code with Go plugin (marketplace symlink mode)\n")
	fmt.Printf("  Plugin source: %s\n", pluginDir)
	fmt.Printf("  Hooks: firing natively via installed plugin\n")

	launchErr := launchClaude(LaunchOpts{
		Mode:            "go",
		SystemPromptDir: pluginDir,
		ExtraArgs:       extraArgs,
		ProjectRoot:     projectRoot,
	})
	restoreFn()
	return launchErr
}

// launchClaudeDevFallback uses the legacy --plugin-dir approach when symlink setup fails.
func launchClaudeDevFallback(pluginDir, projectRoot string, extraArgs []string) error {
	// Disable marketplace plugin to prevent duplicate hooks.
	fmt.Println("Disabling marketplace htmlgraph plugin (fallback mode)...")
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

	fmt.Printf("  Hooks: WARNING — plugin hooks will NOT fire in --plugin-dir mode\n")

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

// installedPluginsJSON is the path to the Claude Code installed plugins registry.
func installedPluginsJSONPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "plugins", "installed_plugins.json")
}

// claudeSettingsJSONPath is the path to the Claude Code user settings file.
func claudeSettingsJSONPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

// devModeBackupPath returns the path to the dev-mode backup state file.
func devModeBackupPath(projectRoot string) string {
	base := ".htmlgraph"
	if projectRoot != "" {
		base = filepath.Join(projectRoot, ".htmlgraph")
	}
	return filepath.Join(base, ".dev-mode-backup")
}

// setupDevModeSymlink replaces the installed htmlgraph plugin directory with a
// symlink pointing to the local plugin source, enables the plugin in settings,
// and returns a cleanup function that restores the original state.
func setupDevModeSymlink(pluginDir, projectRoot string) (func(), error) {
	const pluginKey = "htmlgraph@htmlgraph"

	// Read installed_plugins.json to find the current installPath.
	pluginsFile := installedPluginsJSONPath()
	pluginsData, err := os.ReadFile(pluginsFile)
	if err != nil {
		return nil, fmt.Errorf("reading installed_plugins.json: %w", err)
	}

	// The file has a top-level "plugins" key.
	var outer struct {
		Version int                        `json:"version"`
		Plugins map[string]json.RawMessage `json:"plugins"`
	}
	if err := json.Unmarshal(pluginsData, &outer); err != nil {
		return nil, fmt.Errorf("parsing installed_plugins.json structure: %w", err)
	}

	type pluginEntry struct {
		Scope       string `json:"scope"`
		InstallPath string `json:"installPath"`
		Version     string `json:"version"`
	}

	var installPath string
	hadInstallPath := false

	if raw, ok := outer.Plugins[pluginKey]; ok {
		var entries []pluginEntry
		if err := json.Unmarshal(raw, &entries); err == nil && len(entries) > 0 {
			installPath = entries[0].InstallPath
			hadInstallPath = installPath != ""
		}
	}

	if !hadInstallPath {
		return nil, fmt.Errorf("htmlgraph plugin not installed (no entry for %q in installed_plugins.json); install with: claude plugin install htmlgraph@htmlgraph", pluginKey)
	}

	// Determine backup path for the real install directory.
	backupPath := installPath + ".dev-bak"

	// Save backup state to disk so we can recover from a crash.
	backup := devModeBackup{
		InstallPath:    installPath,
		BackupPath:     backupPath,
		PluginKey:      pluginKey,
		HadInstallPath: hadInstallPath,
	}

	// Read current enabled state from settings so we can restore it.
	wasEnabled := false
	if settingsData, err := os.ReadFile(claudeSettingsJSONPath()); err == nil {
		var settings map[string]json.RawMessage
		if json.Unmarshal(settingsData, &settings) == nil {
			if epRaw, ok := settings["enabledPlugins"]; ok {
				var ep map[string]bool
				if json.Unmarshal(epRaw, &ep) == nil {
					wasEnabled = ep[pluginKey]
				}
			}
		}
	}
	backup.WasEnabled = wasEnabled

	// Write backup state file.
	backupStateFile := devModeBackupPath(projectRoot)
	if data, err := json.MarshalIndent(backup, "", "  "); err == nil {
		os.WriteFile(backupStateFile, data, 0644) //nolint:errcheck
	}

	// Perform the swap: back up real directory, then symlink source in its place.
	if err := swapToSymlink(installPath, backupPath, pluginDir); err != nil {
		os.Remove(backupStateFile) //nolint:errcheck
		return nil, err
	}

	// Enable the plugin in settings.json.
	if err := setPluginEnabled(pluginKey, true); err != nil {
		// Non-fatal: warn and continue — user may have it enabled already.
		fmt.Fprintf(os.Stderr, "warning: could not enable plugin in settings.json: %v\n", err)
	}

	fmt.Printf("  Symlinked: %s -> %s\n", installPath, pluginDir)

	restoreFn := func() {
		restoreFromSymlink(installPath, backupPath, pluginKey, wasEnabled, backupStateFile)
	}
	return restoreFn, nil
}

// swapToSymlink backs up installPath (if it exists and is not already a symlink)
// then creates a symlink at installPath pointing to pluginDir.
func swapToSymlink(installPath, backupPath, pluginDir string) error {
	info, statErr := os.Lstat(installPath)

	if statErr == nil {
		// Path exists.
		if info.Mode()&os.ModeSymlink != 0 {
			// Already a symlink (leftover from previous crash) — remove it.
			if err := os.Remove(installPath); err != nil {
				return fmt.Errorf("removing stale symlink at %s: %w", installPath, err)
			}
		} else {
			// Real directory — back it up.
			if err := os.Rename(installPath, backupPath); err != nil {
				return fmt.Errorf("backing up %s to %s: %w", installPath, backupPath, err)
			}
		}
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("stat %s: %w", installPath, statErr)
	} else {
		// installPath doesn't exist — ensure parent directory exists.
		if err := os.MkdirAll(filepath.Dir(installPath), 0755); err != nil {
			return fmt.Errorf("creating parent directory for %s: %w", installPath, err)
		}
	}

	if err := os.Symlink(pluginDir, installPath); err != nil {
		// Attempt to restore backup before returning the error.
		if _, backupExists := os.Stat(backupPath); backupExists == nil {
			os.Rename(backupPath, installPath) //nolint:errcheck
		}
		return fmt.Errorf("creating symlink %s -> %s: %w", installPath, pluginDir, err)
	}
	return nil
}

// restoreFromSymlink removes the dev-mode symlink and restores the backup.
func restoreFromSymlink(installPath, backupPath, pluginKey string, wasEnabled bool, backupStateFile string) {
	// Remove the symlink.
	if err := os.Remove(installPath); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "warning: could not remove dev symlink %s: %v\n", installPath, err)
	}

	// Restore backup if it exists.
	if _, err := os.Stat(backupPath); err == nil {
		if err := os.Rename(backupPath, installPath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not restore %s from %s: %v\n", installPath, backupPath, err)
		}
	}

	// Restore enabled state in settings.json.
	if err := setPluginEnabled(pluginKey, wasEnabled); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not restore plugin enabled state: %v\n", err)
	}

	// Remove the backup state file.
	os.Remove(backupStateFile) //nolint:errcheck

	fmt.Println("Dev mode cleanup complete.")
}

// setPluginEnabled sets enabledPlugins[key] = enabled in ~/.claude/settings.json.
func setPluginEnabled(key string, enabled bool) error {
	settingsPath := claudeSettingsJSONPath()
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("reading settings.json: %w", err)
	}

	var settings map[string]json.RawMessage
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("parsing settings.json: %w", err)
	}

	var ep map[string]bool
	if epRaw, ok := settings["enabledPlugins"]; ok {
		if err := json.Unmarshal(epRaw, &ep); err != nil {
			ep = make(map[string]bool)
		}
	} else {
		ep = make(map[string]bool)
	}

	ep[key] = enabled

	epBytes, err := json.Marshal(ep)
	if err != nil {
		return fmt.Errorf("marshalling enabledPlugins: %w", err)
	}
	settings["enabledPlugins"] = json.RawMessage(epBytes)

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling settings.json: %w", err)
	}
	if err := os.WriteFile(settingsPath, out, 0644); err != nil {
		return fmt.Errorf("writing settings.json: %w", err)
	}
	return nil
}

// cleanupStaleDev checks for a leftover .dev-mode-backup file from a previous
// crash and restores the original plugin state if one is found.
func cleanupStaleDev(projectRoot string) {
	backupStateFile := devModeBackupPath(projectRoot)
	data, err := os.ReadFile(backupStateFile)
	if err != nil {
		return // No stale backup — nothing to do.
	}

	var backup devModeBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not parse stale dev-mode backup state: %v\n", err)
		return
	}

	fmt.Println("Found stale dev-mode state from previous crash — restoring...")
	restoreFromSymlink(backup.InstallPath, backup.BackupPath, backup.PluginKey, backup.WasEnabled, backupStateFile)
}

func launchClaudeInit(extraArgs []string) error {
	pluginDir := resolvePluginDir()
	projectRoot, _ := resolveProjectRoot()
	cleanupStaleDev(projectRoot)
	ensurePluginOnLaunch()
	fmt.Println("Launching Claude Code with marketplace plugin (init mode)...")
	return launchClaude(LaunchOpts{
		Mode:            "init",
		SystemPromptDir: pluginDir,
		ExtraArgs:       extraArgs,
		ProjectRoot:     projectRoot,
	})
}

func launchClaudeContinue(extraArgs []string) error {
	projectRoot, _ := resolveProjectRoot()
	cleanupStaleDev(projectRoot)
	ensurePluginOnLaunch()
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
	projectRoot, _ := resolveProjectRoot()
	cleanupStaleDev(projectRoot)
	ensurePluginOnLaunch()
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
	exec.Command("claude", "plugin", "marketplace", "add",
		htmlgraphMarketplaceRepo).Run() //nolint:errcheck

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
