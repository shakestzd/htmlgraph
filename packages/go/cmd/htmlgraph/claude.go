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

func claudeCmd() *cobra.Command {
	var dev bool

	cmd := &cobra.Command{
		Use:   "claude",
		Short: "Launch Claude Code with HtmlGraph Go plugin",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !dev {
				return fmt.Errorf("--dev flag required (only dev mode is supported)")
			}
			return launchClaudeDev(args)
		},
	}
	cmd.Flags().BoolVar(&dev, "dev", false, "Launch in dev mode with Go binary hooks")
	return cmd
}

func launchClaudeDev(extraArgs []string) error {
	// Step 1: Uninstall marketplace plugin (prevent conflicts with Go plugin)
	fmt.Println("Disabling marketplace htmlgraph plugin...")
	uninstall := exec.Command("claude", "plugin", "uninstall", "htmlgraph@htmlgraph")
	uninstall.Run() // ignore errors — may not be installed

	// Step 2: Resolve Go plugin directory
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable path: %w", err)
	}
	binDir := filepath.Dir(binPath)
	// Binary is at packages/go-plugin/hooks/bin/htmlgraph-hooks
	// Plugin root is packages/go-plugin/ (two levels up from bin/)
	pluginDir, err := filepath.Abs(filepath.Join(binDir, "..", ".."))
	if err != nil {
		return fmt.Errorf("resolving plugin dir: %w", err)
	}

	// Verify plugin structure
	pluginJSON := filepath.Join(pluginDir, ".claude-plugin", "plugin.json")
	if _, err := os.Stat(pluginJSON); os.IsNotExist(err) {
		return fmt.Errorf("plugin.json not found at %s\nAre you running from the project root?", pluginJSON)
	}

	// Verify Go binary in hooks
	hooksBinary := filepath.Join(pluginDir, "hooks", "bin", "htmlgraph-hooks")
	if _, err := os.Stat(hooksBinary); os.IsNotExist(err) {
		return fmt.Errorf("Go hooks binary not found at %s\nBuild with: packages/go-plugin/build.sh", hooksBinary)
	}

	// Step 3: Stub out .claude/hooks/hooks.json to prevent Python hook merging
	restoreFn := stubProjectHooks()

	// Setup cleanup on signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		restoreFn()
		os.Exit(0)
	}()

	// Step 4: Load system prompt from Go plugin config
	systemPrompt := loadSystemPrompt(pluginDir)

	// Step 5: Write launch marker for hooks to detect Go mode
	writeLaunchMarker("go")

	// Step 6: Build claude command args
	claudeArgs := []string{"--plugin-dir", pluginDir}
	if systemPrompt != "" {
		claudeArgs = append(claudeArgs, "--append-system-prompt", systemPrompt)
	}
	claudeArgs = append(claudeArgs, extraArgs...)

	fmt.Printf("Launching Claude Code with Go plugin\n")
	fmt.Printf("  Plugin: %s\n", pluginDir)
	fmt.Printf("  Hooks: Go binary (near-zero cold start)\n")

	claudePath, err := exec.LookPath("claude")
	if err != nil {
		restoreFn()
		return fmt.Errorf("claude not found in PATH: %w", err)
	}

	// Run claude as child process so we can restore hooks.json after it exits
	c := exec.Command(claudePath, claudeArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	err = c.Run()
	restoreFn()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}

// stubProjectHooks replaces .claude/hooks/hooks.json with an empty stub
// to prevent Python hooks from firing alongside Go hooks.
// Returns a restore function that must be called on exit.
func stubProjectHooks() func() {
	projectHooks := ".claude/hooks/hooks.json"
	backupPath := ".claude/hooks/hooks.json.go-backup"

	original, err := os.ReadFile(projectHooks)
	if err != nil {
		// No existing hooks.json — nothing to stub
		return func() {}
	}

	// Write backup before stubbing
	if err := os.WriteFile(backupPath, original, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not backup hooks.json: %v\n", err)
		return func() {}
	}

	// Write empty stub to prevent Python hook merging
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

// loadSystemPrompt reads the Go-centric system prompt from the plugin config directory.
func loadSystemPrompt(pluginDir string) string {
	promptPath := filepath.Join(pluginDir, "config", "system-prompt.md")
	data, err := os.ReadFile(promptPath)
	if err != nil {
		return ""
	}
	return string(data)
}

// writeLaunchMarker writes .htmlgraph/.launch-mode for hooks to detect the launch mode.
func writeLaunchMarker(mode string) {
	marker := map[string]any{
		"mode":      mode,
		"pid":       os.Getpid(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(marker)
	if err != nil {
		return
	}
	os.MkdirAll(".htmlgraph", 0755) //nolint:errcheck
	os.WriteFile(".htmlgraph/.launch-mode", data, 0644) //nolint:errcheck
}
