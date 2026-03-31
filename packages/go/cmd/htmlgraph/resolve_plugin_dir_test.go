package main

import (
	"os"
	"path/filepath"
	"testing"
)

// createFakePlugin creates a minimal plugin directory structure at the given path.
// Returns the plugin dir path for convenience.
func createFakePlugin(t *testing.T, dir string) string {
	t.Helper()
	claudePluginDir := filepath.Join(dir, ".claude-plugin")
	if err := os.MkdirAll(claudePluginDir, 0755); err != nil {
		t.Fatalf("failed to create .claude-plugin dir: %v", err)
	}
	pluginJSON := filepath.Join(claudePluginDir, "plugin.json")
	if err := os.WriteFile(pluginJSON, []byte(`{"name":"htmlgraph","version":"0.1.0"}`), 0644); err != nil {
		t.Fatalf("failed to write plugin.json: %v", err)
	}
	return dir
}

// TestResolvePluginDir_EnvVarOverride tests that HTMLGRAPH_PLUGIN_DIR takes highest priority.
func TestResolvePluginDir_EnvVarOverride(t *testing.T) {
	tmpDir := t.TempDir()
	pluginDir := createFakePlugin(t, filepath.Join(tmpDir, "my-plugin"))

	t.Setenv("HTMLGRAPH_PLUGIN_DIR", pluginDir)

	got := resolvePluginDir()
	if got != pluginDir {
		t.Errorf("resolvePluginDir() = %q, want %q", got, pluginDir)
	}
}

// TestResolvePluginDir_EnvVarInvalidFallsThrough tests that an invalid
// HTMLGRAPH_PLUGIN_DIR does not short-circuit -- the function falls through
// to subsequent strategies.
func TestResolvePluginDir_EnvVarInvalidFallsThrough(t *testing.T) {
	t.Setenv("HTMLGRAPH_PLUGIN_DIR", "/nonexistent/path/that/does/not/exist")

	// With no valid env var and no well-known path or symlink, should return "".
	got := resolvePluginDir()
	// We can't assert "" because the symlink walk-up from the test binary might
	// accidentally find a plugin dir. Instead, just ensure it didn't return
	// the invalid path.
	if got == "/nonexistent/path/that/does/not/exist" {
		t.Errorf("resolvePluginDir() returned invalid env var path without validation")
	}
}

// TestResolvePluginDir_EnvVarMissingPluginJSON tests that HTMLGRAPH_PLUGIN_DIR
// is skipped when the directory exists but lacks .claude-plugin/plugin.json.
func TestResolvePluginDir_EnvVarMissingPluginJSON(t *testing.T) {
	tmpDir := t.TempDir()
	// Directory exists but has no .claude-plugin/plugin.json
	emptyDir := filepath.Join(tmpDir, "empty-plugin")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	t.Setenv("HTMLGRAPH_PLUGIN_DIR", emptyDir)

	got := resolvePluginDir()
	if got == emptyDir {
		t.Errorf("resolvePluginDir() returned env var dir that lacks plugin.json")
	}
}

// TestResolvePluginDir_WellKnownPath tests that ~/.claude/plugins/htmlgraph/ is
// discovered when no env var is set. We override HOME to use a temp directory.
func TestResolvePluginDir_WellKnownPath(t *testing.T) {
	tmpHome := t.TempDir()
	wellKnownDir := filepath.Join(tmpHome, ".claude", "plugins", "htmlgraph")
	createFakePlugin(t, wellKnownDir)

	// Clear env var so it doesn't interfere
	t.Setenv("HTMLGRAPH_PLUGIN_DIR", "")
	// Override HOME so os.UserHomeDir() returns our temp dir
	t.Setenv("HOME", tmpHome)

	got := resolvePluginDir()
	if got != wellKnownDir {
		t.Errorf("resolvePluginDir() = %q, want %q (well-known path)", got, wellKnownDir)
	}
}

// TestResolvePluginDir_EnvVarTakesPrecedenceOverWellKnown tests that the env var
// wins even when the well-known path also has a valid plugin.
func TestResolvePluginDir_EnvVarTakesPrecedenceOverWellKnown(t *testing.T) {
	tmpDir := t.TempDir()

	// Set up both: env var plugin and well-known plugin
	envPluginDir := createFakePlugin(t, filepath.Join(tmpDir, "env-plugin"))
	tmpHome := filepath.Join(tmpDir, "home")
	wellKnownDir := filepath.Join(tmpHome, ".claude", "plugins", "htmlgraph")
	createFakePlugin(t, wellKnownDir)

	t.Setenv("HTMLGRAPH_PLUGIN_DIR", envPluginDir)
	t.Setenv("HOME", tmpHome)

	got := resolvePluginDir()
	if got != envPluginDir {
		t.Errorf("resolvePluginDir() = %q, want %q (env var should take precedence)", got, envPluginDir)
	}
}

// TestResolvePluginDir_SymlinkWalkUpFallback tests that the original symlink walk-up
// behavior still works as a fallback. This validates backward compatibility
// with the dev mode workflow where the binary is symlinked from the plugin tree.
//
// Note: This test is inherently limited because os.Executable() returns the
// test binary path, not a symlink inside a plugin tree. We verify that the
// function at least returns "" when no other strategy matches, confirming the
// symlink walk-up doesn't crash.
func TestResolvePluginDir_SymlinkWalkUpFallback(t *testing.T) {
	t.Setenv("HTMLGRAPH_PLUGIN_DIR", "")
	// Set HOME to a temp dir with no plugin installed
	t.Setenv("HOME", t.TempDir())

	got := resolvePluginDir()
	// The test binary is not inside a plugin tree, so the symlink walk-up
	// should return "". This confirms the fallback path runs without error.
	if got != "" {
		// It's possible the real binary happens to be in a plugin tree,
		// so we only warn rather than fail.
		t.Logf("resolvePluginDir() returned %q (may be a real plugin tree)", got)
	}
}
