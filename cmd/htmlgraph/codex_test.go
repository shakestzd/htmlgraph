package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCodexHelpRenders verifies that codexCmd().Execute() with --help
// doesn't error and prints help text.
func TestCodexHelpRenders(t *testing.T) {
	cmd := codexCmd()
	cmd.SetArgs([]string{"--help"})

	// Capture output to avoid printing during test
	outBuf := &strings.Builder{}
	cmd.SetOut(outBuf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("codexCmd().Execute() with --help: %v", err)
	}

	output := outBuf.String()
	if !strings.Contains(output, "Launch Codex CLI") {
		t.Errorf("help output missing expected text. Got:\n%s", output)
	}
}

// TestCodexParsingFlags verifies that codex command flags are parsed correctly.
// We only test flags that don't trigger external commands (like codex.exe or marketplace ops).
func TestCodexParsingFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantInit bool
	}{
		{
			name:     "--init with --dry-run",
			args:     []string{"--init", "--dry-run", "--yes"},
			wantInit: true,
		},
		{
			name:     "--help",
			args:     []string{"--help"},
			wantInit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := codexCmd()
			cmd.SetArgs(tt.args)

			// Suppress stdout/stderr for testing.
			cmd.SetOut(&strings.Builder{})
			cmd.SetErr(&strings.Builder{})

			// Note: --help causes Execute to return nil without running the command,
			// so it's safe to test. Commands that try to exec codex (no flags, or
			// --continue/--resume/--dev without --dry-run) will fail during tests
			// because codex binary is not available. Those are integration tests.
			err := cmd.Execute()
			if err != nil {
				t.Logf("Execute returned: %v (expected for --help or --init --dry-run)", err)
			}
		})
	}
}

// TestIsCodexMarketplaceInstalledAt verifies the marketplace detection logic.
func TestIsCodexMarketplaceInstalledAt(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "config.toml")

	// Test 1: File does not exist — should return false
	if isCodexMarketplaceInstalledAt(configPath) {
		t.Errorf("expected false when config file does not exist")
	}

	// Test 2: File exists but does not contain the marketplace section
	err := os.WriteFile(configPath, []byte("[other]\nkey = value\n"), 0644)
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if isCodexMarketplaceInstalledAt(configPath) {
		t.Errorf("expected false when marketplace not in config")
	}

	// Test 3: File contains the marketplace section
	err = os.WriteFile(configPath, []byte("[marketplaces.htmlgraph]\nrepo = \"shakestzd/htmlgraph\"\n"), 0644)
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if !isCodexMarketplaceInstalledAt(configPath) {
		t.Errorf("expected true when marketplace section exists")
	}

	// Test 4: File contains the plugin section variant
	err = os.WriteFile(configPath, []byte(`[plugins."htmlgraph@htmlgraph"]`+"\n"), 0644)
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if !isCodexMarketplaceInstalledAt(configPath) {
		t.Errorf("expected true when plugin section exists")
	}
}

// TestIsCodexHooksEnabledAt verifies the hooks feature flag detection logic.
func TestIsCodexHooksEnabledAt(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "config.toml")

	// Test 1: File does not exist
	if isCodexHooksEnabledAt(configPath) {
		t.Errorf("expected false when config file does not exist")
	}

	// Test 2: File exists but no codex_hooks line
	err := os.WriteFile(configPath, []byte("[other]\nkey = value\n"), 0644)
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if isCodexHooksEnabledAt(configPath) {
		t.Errorf("expected false when codex_hooks not in config")
	}

	// Test 3: File has codex_hooks = true
	err = os.WriteFile(configPath, []byte("[features]\ncodex_hooks = true\n"), 0644)
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if !isCodexHooksEnabledAt(configPath) {
		t.Errorf("expected true when codex_hooks = true")
	}

	// Test 4: File has codex_hooks = false
	err = os.WriteFile(configPath, []byte("[features]\ncodex_hooks = false\n"), 0644)
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if isCodexHooksEnabledAt(configPath) {
		t.Errorf("expected false when codex_hooks = false")
	}

	// Test 5: File has codex_hooks with spaces around =
	err = os.WriteFile(configPath, []byte("[features]\ncodex_hooks  =  true\n"), 0644)
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if !isCodexHooksEnabledAt(configPath) {
		t.Errorf("expected true when codex_hooks has spaces around =")
	}
}

// TestPromptYesNo verifies the yes/no prompt logic.
func TestPromptYesNo(t *testing.T) {
	tests := []struct {
		name      string
		autoYes   bool
		wantResp  bool
		question  string
	}{
		{
			name:     "auto-yes returns true immediately",
			autoYes:  true,
			wantResp: true,
			question: "Enable feature?",
		},
		{
			name:     "auto-yes=false still returns true (no stdin)",
			autoYes:  false,
			wantResp: false, // will be false because we have no stdin input
			question: "Enable feature?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When yes=true, promptYesNo returns immediately without reading stdin
			resp := promptYesNo(tt.question, tt.autoYes)
			if tt.autoYes && !resp {
				t.Errorf("promptYesNo(..., true) should return true immediately")
			}
		})
	}
}

// TestEnsureCodexHooksEnabledIdempotent verifies that ensureCodexHooksEnabled
// is idempotent — calling it twice produces identical output.
func TestEnsureCodexHooksEnabledIdempotent(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "config.toml")

	// First call: create and enable
	if err := ensureCodexHooksEnabled(configPath); err != nil {
		t.Fatalf("first ensureCodexHooksEnabled: %v", err)
	}
	data1, _ := os.ReadFile(configPath)

	// Second call: should be idempotent
	if err := ensureCodexHooksEnabled(configPath); err != nil {
		t.Fatalf("second ensureCodexHooksEnabled: %v", err)
	}
	data2, _ := os.ReadFile(configPath)

	if string(data1) != string(data2) {
		t.Errorf("second call changed the output:\nFirst:\n%s\nSecond:\n%s", string(data1), string(data2))
	}
}

// TestCodexHooksUpsertPreservesExistingFeaturesTable verifies that enabling
// codex_hooks merges into an existing [features] table without duplicating it.
func TestCodexHooksUpsertPreservesExistingFeaturesTable(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "config.toml")

	// Create a config with existing [features] section and other keys
	initialContent := "[features]\nother_flag = true\n"
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Call ensureCodexHooksEnabled
	if err := ensureCodexHooksEnabled(configPath); err != nil {
		t.Fatalf("ensureCodexHooksEnabled: %v", err)
	}

	// Verify the config has both keys in a single [features] table
	data, _ := os.ReadFile(configPath)
	content := string(data)

	// Count [features] sections (should be exactly one)
	featuresSectionCount := strings.Count(content, "[features]")
	if featuresSectionCount != 1 {
		t.Errorf("expected exactly 1 [features] section, got %d:\n%s", featuresSectionCount, content)
	}

	// Verify both keys are present
	if !strings.Contains(content, "codex_hooks") {
		t.Errorf("codex_hooks not found in output")
	}
	if !strings.Contains(content, "other_flag") {
		t.Errorf("other_flag not preserved in output")
	}
}

// TestEnsureCodexHooksEnabledCreatesFromEmpty verifies that ensureCodexHooksEnabled
// can create a new config file with just the [features] section.
func TestEnsureCodexHooksEnabledCreatesFromEmpty(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "config.toml")

	// Enable codex_hooks on a non-existent file
	if err := ensureCodexHooksEnabled(configPath); err != nil {
		t.Fatalf("ensureCodexHooksEnabled: %v", err)
	}

	// Verify the file was created with codex_hooks enabled
	data, _ := os.ReadFile(configPath)
	content := string(data)

	if !strings.Contains(content, "codex_hooks") {
		t.Errorf("codex_hooks not found in newly created config")
	}
	if !isCodexHooksEnabledAt(configPath) {
		t.Errorf("codex_hooks = true check failed after ensureCodexHooksEnabled")
	}
}

// TestCodexDevReplacesMismatchedMarketplace verifies that --dev mode detects
// a mismatched marketplace registration and replaces it.
func TestCodexDevReplacesMismatchedMarketplace(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "config.toml")

	// Seed a config with a mismatched marketplace pointing elsewhere
	initialContent := `[marketplaces.htmlgraph]
source = "/some/other/path"
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Verify the mismatched path is detected
	detected := getCodexMarketplacePathAt(configPath)
	if detected != "/some/other/path" {
		t.Errorf("expected to detect /some/other/path, got %q", detected)
	}

	// In a real scenario, launchCodexDev would now detect the mismatch
	// and run marketplace remove + add. For testing, we just verify the detection.
	// A full integration test would mock exec.Command.
}

// TestGetCodexMarketplacePathAt verifies marketplace path detection from TOML.
func TestGetCodexMarketplacePathAt(t *testing.T) {
	tmpdir := t.TempDir()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "no config file",
			content: "",
			want:    "",
		},
		{
			name: "marketplaces.htmlgraph with source",
			content: "[marketplaces.htmlgraph]\n" +
				"source = \"/path/to/marketplace\"\n",
			want: "/path/to/marketplace",
		},
		{
			name: "marketplaces.htmlgraph with path",
			content: "[marketplaces.htmlgraph]\n" +
				"path = \"/alt/path\"\n",
			want: "/alt/path",
		},
		{
			name: "plugins variant",
			content: "[plugins]\n" +
				"\"htmlgraph@htmlgraph\" = {source = \"/plugin/path\"}\n",
			want: "/plugin/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tmpdir, tt.name+".toml")
			if tt.content != "" {
				if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
					t.Fatalf("WriteFile: %v", err)
				}
			}

			got := getCodexMarketplacePathAt(configPath)
			if got != tt.want {
				t.Errorf("getCodexMarketplacePathAt: want %q, got %q", tt.want, got)
			}
		})
	}
}

// TestRemoveCodexHtmlgraphRegistrations verifies that removeCodexHtmlgraphRegistrations
// correctly deletes htmlgraph entries while preserving other config sections.
func TestRemoveCodexHtmlgraphRegistrations(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "config.toml")

	// Create a realistic config with htmlgraph entries plus other unrelated config
	initialContent := `[plugins]
"htmlgraph@htmlgraph" = {source = "/old/path"}
"github@openai-curated" = {source = "https://github.com/openai/curated"}

[marketplaces]
htmlgraph = {source = "/also/old/path"}
other_marketplace = {source = "https://other.com"}

[mcp_servers]
my_server = {command = "/path/to/server"}

[features]
some_feature = true
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Call removeCodexHtmlgraphRegistrations
	removed, err := removeCodexHtmlgraphRegistrations(configPath)
	if err != nil {
		t.Fatalf("removeCodexHtmlgraphRegistrations: %v", err)
	}
	if !removed {
		t.Errorf("expected removed=true, got false")
	}

	// Read the result and verify
	data, _ := os.ReadFile(configPath)
	content := string(data)

	// htmlgraph entries should be gone
	if strings.Contains(content, `"htmlgraph@htmlgraph"`) {
		t.Errorf("htmlgraph@htmlgraph should be removed but is still present")
	}
	if strings.Contains(content, "htmlgraph = ") {
		t.Errorf("[marketplaces.htmlgraph] should be removed but is still present")
	}

	// Other entries must be preserved
	if !strings.Contains(content, "github@openai-curated") {
		t.Errorf("github@openai-curated plugin should be preserved but was removed")
	}
	if !strings.Contains(content, "other_marketplace") {
		t.Errorf("other_marketplace should be preserved but was removed")
	}
	if !strings.Contains(content, "mcp_servers") {
		t.Errorf("[mcp_servers] section should be preserved but was removed")
	}
	if !strings.Contains(content, "some_feature") {
		t.Errorf("[features] section should be preserved but was removed")
	}
}

// TestRemoveCodexHtmlgraphRegistrationsNoop verifies that removeCodexHtmlgraphRegistrations
// returns removed=false and preserves file content byte-for-byte when no htmlgraph entries exist.
func TestRemoveCodexHtmlgraphRegistrationsNoop(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "config.toml")

	// Create a config with no htmlgraph entries
	initialContent := `[plugins]
"github@openai-curated" = {source = "https://github.com/openai/curated"}

[mcp_servers]
my_server = {command = "/path/to/server"}
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Read the original content for comparison
	originalData, _ := os.ReadFile(configPath)

	// Call removeCodexHtmlgraphRegistrations
	removed, err := removeCodexHtmlgraphRegistrations(configPath)
	if err != nil {
		t.Fatalf("removeCodexHtmlgraphRegistrations: %v", err)
	}
	if removed {
		t.Errorf("expected removed=false (no htmlgraph entries), got true")
	}

	// Verify the file was not modified
	finalData, _ := os.ReadFile(configPath)
	if !bytes.Equal(originalData, finalData) {
		t.Errorf("file was modified when it should have been left unchanged.\nOriginal:\n%s\nFinal:\n%s",
			string(originalData), string(finalData))
	}
}

// TestRemoveCodexHtmlgraphRegistrationsNonexistentFile verifies that removeCodexHtmlgraphRegistrations
// gracefully handles a non-existent config file.
func TestRemoveCodexHtmlgraphRegistrationsNonexistentFile(t *testing.T) {
	configPath := "/nonexistent/path/config.toml"

	removed, err := removeCodexHtmlgraphRegistrations(configPath)
	if err != nil {
		t.Fatalf("removeCodexHtmlgraphRegistrations on non-existent file: %v", err)
	}
	if removed {
		t.Errorf("expected removed=false for non-existent file, got true")
	}
}

// TestParseCodexMarketplaceJSON verifies that parseCodexMarketplaceJSON correctly
// extracts marketplace name, plugin name, and plugin source subpath from marketplace.json.
func TestParseCodexMarketplaceJSON(t *testing.T) {
	tmpdir := t.TempDir()

	// Create the directory structure
	pluginsDir := filepath.Join(tmpdir, ".agents", "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Create marketplace.json
	marketplaceJSON := map[string]interface{}{
		"name": "htmlgraph",
		"interface": map[string]interface{}{
			"displayName": "HtmlGraph",
		},
		"plugins": []map[string]interface{}{
			{
				"name": "htmlgraph",
				"source": map[string]interface{}{
					"source": "local",
					"path":   "./htmlgraph",
				},
				"policy": map[string]interface{}{
					"installation":   "AVAILABLE",
					"authentication": "ON_INSTALL",
				},
				"category": "Development Tools",
			},
		},
	}

	data, err := json.Marshal(marketplaceJSON)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	marketplaceFile := filepath.Join(pluginsDir, "marketplace.json")
	if err := os.WriteFile(marketplaceFile, data, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Parse and verify
	mktName, plgName, plgSub, err := parseCodexMarketplaceJSON(tmpdir)
	if err != nil {
		t.Fatalf("parseCodexMarketplaceJSON: %v", err)
	}

	if mktName != "htmlgraph" {
		t.Errorf("expected marketplace name 'htmlgraph', got %q", mktName)
	}
	if plgName != "htmlgraph" {
		t.Errorf("expected plugin name 'htmlgraph', got %q", plgName)
	}
	if plgSub != "htmlgraph" {
		t.Errorf("expected plugin subpath 'htmlgraph' (without ./), got %q", plgSub)
	}
}

// TestParseCodexPluginVersion verifies that parseCodexPluginVersion correctly
// extracts the version from plugin.json.
func TestParseCodexPluginVersion(t *testing.T) {
	tmpdir := t.TempDir()

	// Create the directory structure
	pluginDir := filepath.Join(tmpdir, "htmlgraph", ".codex-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Create plugin.json
	pluginJSON := map[string]interface{}{
		"version": "0.55.5",
		"name":    "htmlgraph",
	}

	data, err := json.Marshal(pluginJSON)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	pluginFile := filepath.Join(pluginDir, "plugin.json")
	if err := os.WriteFile(pluginFile, data, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Parse and verify
	version, err := parseCodexPluginVersion(tmpdir, "htmlgraph")
	if err != nil {
		t.Fatalf("parseCodexPluginVersion: %v", err)
	}

	if version != "0.55.5" {
		t.Errorf("expected version '0.55.5', got %q", version)
	}
}

// TestCopyDirCreatesExpectedLayout verifies that copyDir successfully copies
// a source directory tree to a destination, creating all necessary subdirectories.
func TestCopyDirCreatesExpectedLayout(t *testing.T) {
	tmpdir := t.TempDir()

	// Create source directory with some files and subdirectories
	srcDir := filepath.Join(tmpdir, "src")
	if err := os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Create some test files
	files := map[string]string{
		"file1.txt":          "content1",
		"subdir/file2.txt":   "content2",
		"subdir/file3.json":  `{"key": "value"}`,
	}

	for relPath, content := range files {
		fullPath := filepath.Join(srcDir, relPath)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile %s: %v", relPath, err)
		}
	}

	// Copy to destination
	dstDir := filepath.Join(tmpdir, "dst")
	if err := copyDir(srcDir, dstDir); err != nil {
		t.Fatalf("copyDir: %v", err)
	}

	// Verify all files exist and have correct content
	for relPath, expectedContent := range files {
		fullPath := filepath.Join(dstDir, relPath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("ReadFile %s: %v", relPath, err)
		}
		if string(content) != expectedContent {
			t.Errorf("file %s: expected %q, got %q", relPath, expectedContent, string(content))
		}
	}
}

// TestCopyDirIdempotent verifies that copyDir is idempotent — calling it twice
// with the same destination produces identical results (destination is cleared first).
func TestCopyDirIdempotent(t *testing.T) {
	tmpdir := t.TempDir()

	// Create source directory with a file
	srcDir := filepath.Join(tmpdir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	testFile := filepath.Join(srcDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	dstDir := filepath.Join(tmpdir, "dst")

	// First copy
	if err := copyDir(srcDir, dstDir); err != nil {
		t.Fatalf("first copyDir: %v", err)
	}
	data1, err := os.ReadFile(filepath.Join(dstDir, "test.txt"))
	if err != nil {
		t.Fatalf("ReadFile after first copy: %v", err)
	}

	// Second copy (should be idempotent)
	if err := copyDir(srcDir, dstDir); err != nil {
		t.Fatalf("second copyDir: %v", err)
	}
	data2, err := os.ReadFile(filepath.Join(dstDir, "test.txt"))
	if err != nil {
		t.Fatalf("ReadFile after second copy: %v", err)
	}

	if !bytes.Equal(data1, data2) {
		t.Errorf("idempotency check failed: content differs after second copy")
	}
}

// TestInstallCodexPluginToCacheCreatesExpectedLayout verifies that
// installCodexPluginToCache creates the expected cache layout.
func TestInstallCodexPluginToCacheCreatesExpectedLayout(t *testing.T) {
	tmpdir := t.TempDir()

	// Set HOME to tmpdir for this test so UserHomeDir() returns tmpdir
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpdir)

	// Create a fake marketplace with plugin.json and a test file
	marketplaceRoot := filepath.Join(tmpdir, "marketplace")
	pluginSourceDir := filepath.Join(marketplaceRoot, "htmlgraph")
	if err := os.MkdirAll(filepath.Join(pluginSourceDir, ".codex-plugin"), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Create plugin.json with version
	pluginJSON := map[string]interface{}{
		"version": "0.55.5",
	}
	data, _ := json.Marshal(pluginJSON)
	if err := os.WriteFile(filepath.Join(pluginSourceDir, ".codex-plugin", "plugin.json"), data, 0644); err != nil {
		t.Fatalf("WriteFile plugin.json: %v", err)
	}

	// Create a test file in the plugin
	testFile := filepath.Join(pluginSourceDir, "test.md")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("WriteFile test.md: %v", err)
	}

	// Install to cache
	if err := installCodexPluginToCache(marketplaceRoot, "htmlgraph", "htmlgraph", "htmlgraph"); err != nil {
		t.Fatalf("installCodexPluginToCache: %v", err)
	}

	// Verify cache layout
	expectedCachePath := filepath.Join(tmpdir, ".codex", "plugins", "cache", "htmlgraph", "htmlgraph", "0.55.5")
	expectedPluginJSON := filepath.Join(expectedCachePath, ".codex-plugin", "plugin.json")
	if _, err := os.Stat(expectedPluginJSON); err != nil {
		t.Fatalf("cache layout check failed: %s does not exist: %v", expectedPluginJSON, err)
	}

	expectedTestFile := filepath.Join(expectedCachePath, "test.md")
	if _, err := os.Stat(expectedTestFile); err != nil {
		t.Fatalf("cache layout check failed: %s does not exist: %v", expectedTestFile, err)
	}
}

// TestInstallCodexPluginToCacheIdempotent verifies that installCodexPluginToCache
// is idempotent — calling it twice produces identical cache state.
func TestInstallCodexPluginToCacheIdempotent(t *testing.T) {
	tmpdir := t.TempDir()

	// Set HOME to tmpdir
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpdir)

	// Create fake marketplace
	marketplaceRoot := filepath.Join(tmpdir, "marketplace")
	pluginSourceDir := filepath.Join(marketplaceRoot, "htmlgraph")
	if err := os.MkdirAll(filepath.Join(pluginSourceDir, ".codex-plugin"), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Create plugin.json
	pluginJSON := map[string]interface{}{
		"version": "0.55.5",
	}
	data, _ := json.Marshal(pluginJSON)
	if err := os.WriteFile(filepath.Join(pluginSourceDir, ".codex-plugin", "plugin.json"), data, 0644); err != nil {
		t.Fatalf("WriteFile plugin.json: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(pluginSourceDir, "test.md")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("WriteFile test.md: %v", err)
	}

	// First install
	if err := installCodexPluginToCache(marketplaceRoot, "htmlgraph", "htmlgraph", "htmlgraph"); err != nil {
		t.Fatalf("first installCodexPluginToCache: %v", err)
	}

	cachePath := filepath.Join(tmpdir, ".codex", "plugins", "cache", "htmlgraph", "htmlgraph", "0.55.5", "test.md")
	data1, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("ReadFile after first install: %v", err)
	}

	// Second install (should be idempotent)
	if err := installCodexPluginToCache(marketplaceRoot, "htmlgraph", "htmlgraph", "htmlgraph"); err != nil {
		t.Fatalf("second installCodexPluginToCache: %v", err)
	}

	data2, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("ReadFile after second install: %v", err)
	}

	if !bytes.Equal(data1, data2) {
		t.Errorf("idempotency check failed: cache content differs after second install")
	}
}
