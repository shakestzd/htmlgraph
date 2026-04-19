package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
)

// codexMarketplaceRepo is the GitHub repo that hosts the codex marketplace.
const codexMarketplaceRepo = "shakestzd/htmlgraph"

// codexMarketplaceSparse is the sparse path within the monorepo.
const codexMarketplaceSparse = "packages/codex-marketplace"

// codexConfigPath returns the path to ~/.codex/config.toml.
func codexConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex", "config.toml")
}

// codexMarketplaceSection is the TOML key that indicates our marketplace is registered.
const codexMarketplaceSection = `[marketplaces.htmlgraph]`

// isCodexMarketplaceInstalled returns true if ~/.codex/config.toml contains
// evidence that the htmlgraph marketplace (or plugin) is already registered.
// Supports both the [marketplaces.htmlgraph] and [plugins."htmlgraph@htmlgraph"] forms.
func isCodexMarketplaceInstalled() bool {
	return isCodexMarketplaceInstalledAt(codexConfigPath())
}

// isCodexMarketplaceInstalledAt is the testable core that reads the given path.
func isCodexMarketplaceInstalledAt(configPath string) bool {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}
	content := string(data)
	return strings.Contains(content, "[marketplaces.htmlgraph]") ||
		strings.Contains(content, `[plugins."htmlgraph@htmlgraph"]`)
}

// isCodexHooksEnabled returns true if config.toml already has codex_hooks = true.
func isCodexHooksEnabled() bool {
	return isCodexHooksEnabledAt(codexConfigPath())
}

// isCodexHooksEnabledAt is the testable core.
func isCodexHooksEnabledAt(configPath string) bool {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "codex_hooks") && strings.Contains(trimmed, "=") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 && strings.TrimSpace(parts[1]) == "true" {
				return true
			}
		}
	}
	return false
}

// getCodexMarketplacePathAt parses config.toml and returns the registered htmlgraph
// marketplace path, or empty string if not found.
func getCodexMarketplacePathAt(configPath string) string {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}

	tree := make(map[string]interface{})
	if err := toml.Unmarshal(data, &tree); err != nil {
		return ""
	}

	// Check [marketplaces.htmlgraph]
	if mkts, ok := tree["marketplaces"].(map[string]interface{}); ok {
		if hg, ok := mkts["htmlgraph"].(map[string]interface{}); ok {
			if source, ok := hg["source"].(string); ok {
				return source
			}
			if path, ok := hg["path"].(string); ok {
				return path
			}
		}
	}

	// Check [plugins."htmlgraph@htmlgraph"]
	if plugins, ok := tree["plugins"].(map[string]interface{}); ok {
		if hg, ok := plugins["htmlgraph@htmlgraph"].(map[string]interface{}); ok {
			if source, ok := hg["source"].(string); ok {
				return source
			}
			if path, ok := hg["path"].(string); ok {
				return path
			}
		}
	}

	return ""
}

// removeCodexHtmlgraphRegistrations removes any HtmlGraph marketplace or plugin
// registrations from the given config.toml file. It is idempotent — if the file
// does not exist or contains no htmlgraph entries, it is a no-op.
// Returns (removed bool, error). removed=true indicates at least one entry was deleted.
func removeCodexHtmlgraphRegistrations(configPath string) (bool, error) {
	// Read existing config, if any
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // file doesn't exist; no-op
		}
		return false, fmt.Errorf("reading %s: %w", configPath, err)
	}

	// Parse the TOML tree
	tree := make(map[string]interface{})
	if len(data) > 0 {
		if err := toml.Unmarshal(data, &tree); err != nil {
			return false, fmt.Errorf("parsing %s: %w", configPath, err)
		}
	}

	removed := false

	// Remove from [plugins] — only the exact "htmlgraph@htmlgraph" entry
	if plugins, ok := tree["plugins"].(map[string]interface{}); ok {
		if _, exists := plugins["htmlgraph@htmlgraph"]; exists {
			delete(plugins, "htmlgraph@htmlgraph")
			removed = true
		}
		// If [plugins] is now empty, remove the whole section
		if len(plugins) == 0 {
			delete(tree, "plugins")
		}
	}

	// Remove from [marketplaces] — the "htmlgraph" entry
	if mkts, ok := tree["marketplaces"].(map[string]interface{}); ok {
		if _, exists := mkts["htmlgraph"]; exists {
			delete(mkts, "htmlgraph")
			removed = true
		}
		// If [marketplaces] is now empty, remove the whole section
		if len(mkts) == 0 {
			delete(tree, "marketplaces")
		}
	}

	// If nothing was removed, no need to rewrite the file
	if !removed {
		return false, nil
	}

	// Marshal back to TOML and write
	newData, err := toml.Marshal(tree)
	if err != nil {
		return false, fmt.Errorf("marshaling TOML: %w", err)
	}

	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return false, fmt.Errorf("writing %s: %w", configPath, err)
	}

	return true, nil
}

// ensureCodexHooksEnabled parses the config.toml file, merges codex_hooks = true
// into the [features] table (creating the section if absent), and writes it back.
// This is idempotent: if codex_hooks = true is already set, it's a no-op after
// re-serialization.
func ensureCodexHooksEnabled(configPath string) error {
	// Read existing config, if any
	data, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", configPath, err)
	}

	// Parse or create the TOML tree
	tree := make(map[string]interface{})
	if err == nil && len(data) > 0 {
		if err := toml.Unmarshal(data, &tree); err != nil {
			return fmt.Errorf("parsing %s: %w", configPath, err)
		}
	}

	// Ensure [features] table exists and set codex_hooks = true
	features, ok := tree["features"].(map[string]interface{})
	if !ok {
		features = make(map[string]interface{})
		tree["features"] = features
	}
	features["codex_hooks"] = true

	// Marshal back to TOML and write
	newData, err := toml.Marshal(tree)
	if err != nil {
		return fmt.Errorf("marshaling TOML: %w", err)
	}

	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", configPath, err)
	}

	return nil
}

// promptYesNo asks the user a yes/no question and returns true if they answer y/Y/yes.
// If yes is true (--yes flag), the function returns true without prompting.
func promptYesNo(question string, yes bool) bool {
	if yes {
		return true
	}
	fmt.Print(question + " [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(strings.ToLower(line))
	return answer == "y" || answer == "yes"
}

// copyDir recursively copies src directory to dst.
// If dst exists, it is removed first (idempotent).
// Uses filepath.Walk + os.MkdirAll + io.Copy for portability.
func copyDir(src, dst string) error {
	// Remove destination if it exists
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("remove existing destination: %w", err)
	}

	// Walk the source directory
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Compute relative path
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Destination path
		dstPath := filepath.Join(dst, rel)

		if info.IsDir() {
			// Create directory
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		// Create parent directories if needed
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return err
		}

		return os.Chmod(dstPath, info.Mode())
	})
}

// parseCodexMarketplaceJSON parses the marketplace.json file at
// <marketplaceRoot>/.agents/plugins/marketplace.json and returns:
// - marketplace name (e.g., "htmlgraph")
// - plugin name (e.g., "htmlgraph")
// - plugin source subpath (e.g., "htmlgraph", corresponding to "./htmlgraph" in the JSON)
func parseCodexMarketplaceJSON(marketplaceRoot string) (mktName, pluginName, pluginSourceSubpath string, err error) {
	jsonPath := filepath.Join(marketplaceRoot, ".agents", "plugins", "marketplace.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return "", "", "", fmt.Errorf("read marketplace.json at %s: %w", jsonPath, err)
	}

	var mkt struct {
		Name    string `json:"name"`
		Plugins []struct {
			Name   string `json:"name"`
			Source struct {
				Path string `json:"path"`
			} `json:"source"`
		} `json:"plugins"`
	}

	if err := json.Unmarshal(data, &mkt); err != nil {
		return "", "", "", fmt.Errorf("parse marketplace.json: %w", err)
	}

	if len(mkt.Plugins) == 0 {
		return "", "", "", fmt.Errorf("marketplace.json has no plugins defined")
	}

	mktName = mkt.Name
	pluginName = mkt.Plugins[0].Name
	// Remove leading "./" if present
	pluginSourceSubpath = strings.TrimPrefix(mkt.Plugins[0].Source.Path, "./")

	return mktName, pluginName, pluginSourceSubpath, nil
}

// parseCodexPluginVersion reads the plugin version from the plugin's
// .codex-plugin/plugin.json manifest at the given marketplace root + subpath.
func parseCodexPluginVersion(marketplaceRoot, pluginSourceSubpath string) (string, error) {
	jsonPath := filepath.Join(marketplaceRoot, pluginSourceSubpath, ".codex-plugin", "plugin.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return "", fmt.Errorf("read plugin.json at %s: %w", jsonPath, err)
	}

	var manifest struct {
		Version string `json:"version"`
	}

	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", fmt.Errorf("parse plugin.json: %w", err)
	}

	if manifest.Version == "" {
		return "", fmt.Errorf("plugin.json has no version field")
	}

	return manifest.Version, nil
}

// installCodexPluginToCache copies the plugin source tree to Codex's expected
// cache layout so the plugin is immediately usable without the user manually
// invoking /plugin → install in the TUI.
//
// Cache path layout per Codex docs:
//
//	~/.codex/plugins/cache/<marketplaceName>/<pluginName>/<version>/
//
// For example:
//
//	~/.codex/plugins/cache/htmlgraph/htmlgraph/0.55.5/
func installCodexPluginToCache(marketplaceRoot, marketplaceName, pluginName, pluginSourceSubpath string) error {
	// Get the plugin version
	version, err := parseCodexPluginVersion(marketplaceRoot, pluginSourceSubpath)
	if err != nil {
		return fmt.Errorf("determine plugin version: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home directory: %w", err)
	}

	src := filepath.Join(marketplaceRoot, pluginSourceSubpath)
	dst := filepath.Join(home, ".codex", "plugins", "cache", marketplaceName, pluginName, version)

	// Idempotent: remove dst if it exists, then copy fresh
	if err := copyDir(src, dst); err != nil {
		return fmt.Errorf("copy plugin to cache at %s: %w", dst, err)
	}

	return nil
}

// codexCmd returns the cobra command for `htmlgraph codex`.
func codexCmd() *cobra.Command {
	var init_, continue_, dev, cleanup, dryRun, yes bool
	var resumeID string

	cmd := &cobra.Command{
		Use:   "codex",
		Short: "Launch Codex CLI with HtmlGraph context",
		Long: `Launch Codex CLI with HtmlGraph observability context.

Modes:
  htmlgraph codex                   Launch Codex interactively with HtmlGraph env.
  htmlgraph codex --init            Install the HtmlGraph Codex marketplace (idempotent).
  htmlgraph codex --continue        Resume the last Codex session (codex resume --last).
  htmlgraph codex --resume <id>     Resume a specific Codex session by ID.
  htmlgraph codex --dev             Register local packages/codex-marketplace/ and launch.

Session IDs come from ~/.codex/session_index.jsonl.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case init_:
				return runCodexInit(yes, dryRun)
			case dev:
				return launchCodexDev(resumeID, cleanup, dryRun, args)
			case continue_:
				return launchCodexContinue(resumeID, args)
			default:
				return launchCodexDefault(resumeID, args)
			}
		},
	}

	cmd.Flags().BoolVar(&init_, "init", false, "Install the HtmlGraph Codex marketplace plugin (idempotent)")
	cmd.Flags().BoolVar(&continue_, "continue", false, "Resume the last Codex session")
	cmd.Flags().BoolVar(&dev, "dev", false, "Register local packages/codex-marketplace/ and launch Codex")
	cmd.Flags().BoolVar(&cleanup, "cleanup", false, "With --dev: unregister the local marketplace on exit")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would happen without executing")
	cmd.Flags().BoolVar(&yes, "yes", false, "Answer yes to all prompts (non-interactive)")
	cmd.Flags().StringVar(&resumeID, "resume", "", "Resume a specific Codex session by ID")

	return cmd
}

// runCodexInit installs the HtmlGraph Codex marketplace plugin, idempotently.
// Corresponds to: htmlgraph codex --init
// Phase 1: Install / verify marketplace (idempotent).
// Phase 2: Check codex_hooks — prompt user if not set.
func runCodexInit(yes, dryRun bool) error {
	configPath := codexConfigPath()

	// Phase 1: Install or verify marketplace.
	marketplaceInstalled := isCodexMarketplaceInstalledAt(configPath)
	if !marketplaceInstalled {
		addArgs := []string{
			"marketplace", "add",
			codexMarketplaceRepo,
			"--sparse", codexMarketplaceSparse,
		}
		fmt.Printf("Installing HtmlGraph Codex marketplace...\n")
		fmt.Printf("  repo: %s  sparse: %s\n", codexMarketplaceRepo, codexMarketplaceSparse)

		if dryRun {
			fmt.Printf("[dry-run] codex %s\n", strings.Join(addArgs, " "))
		} else {
			if out, err := exec.Command("codex", addArgs...).CombinedOutput(); err != nil {
				return fmt.Errorf("codex marketplace add failed: %w\n%s", err, strings.TrimSpace(string(out)))
			}
			fmt.Println("HtmlGraph Codex marketplace installed.")
		}
	} else {
		fmt.Println("HtmlGraph Codex marketplace is already installed.")
	}

	// Phase 1b: Install plugin to cache.
	// This only works if we have a project root (for --init from the repo).
	// If running --init from outside the repo, we assume the sparse checkout has
	// already populated the marketplace source, but we won't install to cache
	// (that requires knowing the source path structure).
	if projectRoot, err := resolveProjectRoot(); err == nil {
		localMarketplace := filepath.Join(projectRoot, "packages", "codex-marketplace")
		if _, statErr := os.Stat(localMarketplace); statErr == nil {
			// We have the marketplace source locally; install to cache
			mktName, plgName, plgSub, parseErr := parseCodexMarketplaceJSON(localMarketplace)
			if parseErr != nil {
				fmt.Fprintf(os.Stderr, "warning: could not parse marketplace.json: %v\n", parseErr)
			} else {
				if dryRun {
					fmt.Printf("[dry-run] would install plugin to ~/.codex/plugins/cache/%s/%s/<version>/\n", mktName, plgName)
				} else {
					if installErr := installCodexPluginToCache(localMarketplace, mktName, plgName, plgSub); installErr != nil {
						fmt.Fprintf(os.Stderr, "warning: could not install plugin to cache: %v\n", installErr)
					} else {
						fmt.Printf("Plugin installed to ~/.codex/plugins/cache/%s/%s/<version>/\n", mktName, plgName)
					}
				}
			}
		}
	}

	// Phase 2: Check and optionally enable codex_hooks feature flag.
	// This runs on every --init so partial setups can be repaired.
	if !isCodexHooksEnabledAt(configPath) {
		if promptYesNo("Enable the codex_hooks feature flag in ~/.codex/config.toml?", yes) {
			if dryRun {
				fmt.Println("[dry-run] would enable codex_hooks = true in ~/.codex/config.toml")
			} else {
				if err := ensureCodexHooksEnabled(configPath); err != nil {
					fmt.Fprintf(os.Stderr, "warning: could not enable codex_hooks: %v\n", err)
				} else {
					fmt.Println("codex_hooks feature flag enabled.")
				}
			}
		}
	} else {
		fmt.Println("codex_hooks feature flag is already enabled.")
	}

	fmt.Println()
	fmt.Println("Setup complete. Run: htmlgraph codex")
	return nil
}

// launchCodexDefault launches Codex interactively with HtmlGraph env injection.
// Corresponds to: htmlgraph codex
func launchCodexDefault(resumeID string, extraArgs []string) error {
	projectRoot, _ := resolveProjectRoot()
	fmt.Println("Launching Codex CLI with HtmlGraph context...")
	return execCodex(codexLaunchOpts{
		ResumeID:    resumeID,
		ExtraArgs:   extraArgs,
		ProjectRoot: projectRoot,
	})
}

// launchCodexContinue resumes the last Codex session.
// Corresponds to: htmlgraph codex --continue
func launchCodexContinue(resumeID string, extraArgs []string) error {
	projectRoot, _ := resolveProjectRoot()
	fmt.Println("Resuming last Codex session...")
	return execCodex(codexLaunchOpts{
		ResumeLast:  resumeID == "", // only pass --last when no specific ID
		ResumeID:    resumeID,
		ExtraArgs:   extraArgs,
		ProjectRoot: projectRoot,
	})
}

// launchCodexDev registers the local packages/codex-marketplace/ and launches Codex.
// Corresponds to: htmlgraph codex --dev [--cleanup]
// If a mismatched marketplace is already registered (e.g., from a prior --init),
// it is removed and replaced with the local path.
func launchCodexDev(resumeID string, cleanup, dryRun bool, extraArgs []string) error {
	// Resolve the local marketplace path relative to the project root.
	localMarketplace, err := resolveLocalCodexMarketplace()
	if err != nil {
		return err
	}

	fmt.Printf("Launching Codex CLI in dev mode...\n")
	fmt.Printf("  Local marketplace: %s\n", localMarketplace)

	// Ensure the local marketplace is registered (replace mismatched registrations).
	configPath := codexConfigPath()
	registeredPath := getCodexMarketplacePathAt(configPath)

	// Convert to absolute paths for comparison
	localAbs, _ := filepath.Abs(localMarketplace)
	registeredAbs, _ := filepath.Abs(registeredPath)

	if registeredAbs != "" && registeredAbs != localAbs {
		// Mismatched registration: remove the old one via direct TOML editing
		oldPathDisplay := registeredPath
		if oldPathDisplay == "" {
			oldPathDisplay = "(unknown previous path)"
		}
		fmt.Printf("Replacing mismatched marketplace registration (%s)\n", oldPathDisplay)
		if dryRun {
			fmt.Printf("[dry-run] would remove HtmlGraph registrations from %s\n", configPath)
		} else {
			removed, rmErr := removeCodexHtmlgraphRegistrations(configPath)
			if rmErr != nil {
				return fmt.Errorf("removing mismatched marketplace from %s: %w", configPath, rmErr)
			}
			if removed {
				fmt.Println("Mismatched registration removed from config.toml.")
			}
		}
		registeredPath = "" // Force re-add
	}

	// Add the local marketplace if not already registered at the correct path
	if registeredAbs != localAbs {
		addArgs := []string{"marketplace", "add", localMarketplace}
		if dryRun {
			fmt.Printf("[dry-run] codex %s\n", strings.Join(addArgs, " "))
		} else {
			if out, err := exec.Command("codex", addArgs...).CombinedOutput(); err != nil {
				return fmt.Errorf("registering local marketplace failed: %w\n%s", err, strings.TrimSpace(string(out)))
			}
			fmt.Println("Local marketplace registered.")
		}
	} else {
		fmt.Println("Local marketplace already registered — proceeding.")
	}

	// Install plugin to cache (so it's available without manual /plugin install)
	if !dryRun {
		mktName, plgName, plgSub, parseErr := parseCodexMarketplaceJSON(localMarketplace)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not parse marketplace.json: %v\n", parseErr)
		} else {
			if installErr := installCodexPluginToCache(localMarketplace, mktName, plgName, plgSub); installErr != nil {
				fmt.Fprintf(os.Stderr, "warning: could not install plugin to cache: %v\n", installErr)
			} else {
				fmt.Printf("Plugin installed to ~/.codex/plugins/cache/%s/%s/<version>/\n", mktName, plgName)
			}
		}
	} else {
		fmt.Printf("[dry-run] would install plugin to ~/.codex/plugins/cache/<marketplace>/<plugin>/<version>/\n")
	}

	projectRoot, _ := resolveProjectRoot()

	if dryRun {
		fmt.Printf("[dry-run] would exec: codex (resume=%q) in %s\n", resumeID, projectRoot)
		return nil
	}

	err = execCodex(codexLaunchOpts{
		ResumeID:    resumeID,
		ExtraArgs:   extraArgs,
		ProjectRoot: projectRoot,
	})

	// --cleanup: unregister the local marketplace after session ends.
	if cleanup && !dryRun {
		fmt.Println("Cleaning up local marketplace registration...")
		removed, rmErr := removeCodexHtmlgraphRegistrations(configPath)
		if rmErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not remove marketplace registration: %v\n", rmErr)
		} else if !removed {
			fmt.Println("No HtmlGraph registrations found to clean up.")
		}
	}

	return err
}

// resolveLocalCodexMarketplace returns the absolute path to packages/codex-marketplace/
// by walking up from CWD to find the project root (directory containing .htmlgraph/).
// Returns an error if no project root is found or the marketplace directory is missing.
func resolveLocalCodexMarketplace() (string, error) {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return "", fmt.Errorf("could not find project root (.htmlgraph/ directory not found)\n" +
			"Run from the HtmlGraph project directory, or use htmlgraph codex --init for the marketplace version")
	}
	projectRoot := filepath.Dir(htmlgraphDir)
	marketplacePath := filepath.Join(projectRoot, "packages", "codex-marketplace")
	if _, statErr := os.Stat(marketplacePath); os.IsNotExist(statErr) {
		return "", fmt.Errorf("packages/codex-marketplace/ not found at %s\n"+
			"Run from the HtmlGraph repo root, or use htmlgraph codex --init for the marketplace version",
			marketplacePath)
	}
	abs, err := filepath.Abs(marketplacePath)
	if err != nil {
		return "", fmt.Errorf("resolving absolute path for %s: %w", marketplacePath, err)
	}
	return abs, nil
}

// codexLaunchOpts controls how Codex is launched.
type codexLaunchOpts struct {
	// ResumeLast, when true, passes "resume --last" to codex.
	ResumeLast bool
	// ResumeID, if non-empty, passes "resume <id>" to codex.
	// Takes precedence over ResumeLast.
	ResumeID string
	// ExtraArgs are forwarded to the codex process.
	ExtraArgs []string
	// ProjectRoot is the absolute path to the project root.
	// When set, Codex is started with this as the working directory, and
	// HTMLGRAPH_PROJECT_DIR env var is injected.
	ProjectRoot string
}

// execCodex builds the codex argv and execs it, replacing the current process.
// Returns only on exec error.
func execCodex(opts codexLaunchOpts) error {
	codexPath, err := exec.LookPath("codex")
	if err != nil {
		return fmt.Errorf("codex not found in PATH: %w\nInstall Codex CLI first: https://github.com/openai/codex", err)
	}

	var codexArgs []string

	// Determine if we're resuming.
	if opts.ResumeID != "" {
		codexArgs = append(codexArgs, "resume", opts.ResumeID)
	} else if opts.ResumeLast {
		codexArgs = append(codexArgs, "resume", "--last")
	}

	codexArgs = append(codexArgs, opts.ExtraArgs...)

	c := exec.Command(codexPath, codexArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	// Inject HTMLGRAPH_PROJECT_DIR so htmlgraph CLI and hooks resolve to the
	// correct project root regardless of CWD.
	if opts.ProjectRoot != "" {
		c.Env = append(os.Environ(), "HTMLGRAPH_PROJECT_DIR="+opts.ProjectRoot)
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
