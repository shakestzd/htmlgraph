package pluginbuild

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCodexMarketplaceJSONEmitted checks that marketplace.json is written at
// <outDir>/.agents/plugins/marketplace.json with the required fields: non-empty
// name, non-empty plugins[], and source.path starting with "./" (relative to
// the marketplace root, per the Codex docs).
func TestCodexMarketplaceJSONEmitted(t *testing.T) {
	repoRoot := t.TempDir()
	seedAssets(t, repoRoot)
	outDir := filepath.Join(repoRoot, "packages", "codex-marketplace")

	if err := (codexAdapter{}).Emit(fixtureManifest(), repoRoot, outDir); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	mktPath := filepath.Join(outDir, ".agents", "plugins", "marketplace.json")
	data, err := os.ReadFile(mktPath)
	if err != nil {
		t.Fatalf("read marketplace.json at %s: %v", mktPath, err)
	}

	// marketplace.json must NOT exist at the old outDir root location.
	oldPath := filepath.Join(outDir, "marketplace.json")
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("marketplace.json must NOT exist at outDir root %q", oldPath)
	}

	var mkt codexMarketplaceJSON
	if err := json.Unmarshal(data, &mkt); err != nil {
		t.Fatalf("unmarshal marketplace.json: %v", err)
	}

	if mkt.Name == "" {
		t.Error("marketplace.json: name must not be empty")
	}
	if len(mkt.Plugins) == 0 {
		t.Fatal("marketplace.json: plugins[] must not be empty")
	}
	plug := mkt.Plugins[0]
	if plug.Name == "" {
		t.Error("marketplace.json: plugins[0].name must not be empty")
	}
	if !strings.HasPrefix(plug.Source.Path, "./") {
		t.Errorf("marketplace.json: plugins[0].source.path must start with \"./\", got %q", plug.Source.Path)
	}
	if plug.Source.Source != "local" {
		t.Errorf("marketplace.json: plugins[0].source.source must be \"local\", got %q", plug.Source.Source)
	}
	if mkt.Interface.DisplayName == "" {
		t.Error("marketplace.json: interface.displayName must not be empty")
	}
}

// TestCodexMarketplaceSourcePathPointsAtPluginsSubdir verifies that
// marketplace.json's source.path is "./plugins/htmlgraph" — relative to the
// marketplace root (outDir), as required by the Codex docs. The old layout
// used ".agents/plugins/" as the anchor, yielding "./htmlgraph", which Codex
// resolved to a path that never existed.
func TestCodexMarketplaceSourcePathPointsAtPluginsSubdir(t *testing.T) {
	repoRoot := t.TempDir()
	seedAssets(t, repoRoot)
	outDir := filepath.Join(repoRoot, "packages", "codex-marketplace")

	if err := (codexAdapter{}).Emit(fixtureManifest(), repoRoot, outDir); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	mktPath := filepath.Join(outDir, ".agents", "plugins", "marketplace.json")
	data, err := os.ReadFile(mktPath)
	if err != nil {
		t.Fatalf("read marketplace.json: %v", err)
	}

	var mkt codexMarketplaceJSON
	if err := json.Unmarshal(data, &mkt); err != nil {
		t.Fatalf("unmarshal marketplace.json: %v", err)
	}

	if len(mkt.Plugins) == 0 {
		t.Fatal("marketplace.json: plugins[] must not be empty")
	}
	want := "./plugins/htmlgraph"
	if got := mkt.Plugins[0].Source.Path; got != want {
		t.Errorf("marketplace.json: plugins[0].source.path = %q, want %q (must be relative to marketplace root, not .agents/plugins/)", got, want)
	}
}

// TestCodexPluginAtPluginsSubdir verifies that plugin content lives at
// <outDir>/plugins/htmlgraph/, NOT inside .agents/plugins/htmlgraph/. Per the
// Codex docs, source.path is resolved relative to the marketplace root (outDir).
func TestCodexPluginAtPluginsSubdir(t *testing.T) {
	repoRoot := t.TempDir()
	seedAssets(t, repoRoot)
	outDir := filepath.Join(repoRoot, "packages", "codex-marketplace")

	if err := (codexAdapter{}).Emit(fixtureManifest(), repoRoot, outDir); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	// Plugin manifest must exist at new layout path.
	pluginManifest := filepath.Join(outDir, "plugins", "htmlgraph", ".codex-plugin", "plugin.json")
	if _, err := os.Stat(pluginManifest); err != nil {
		t.Fatalf(".codex-plugin/plugin.json not found at expected path %q: %v", pluginManifest, err)
	}

	// Plugin manifest must NOT exist at the old .agents/plugins/htmlgraph/ path.
	oldPath := filepath.Join(outDir, ".agents", "plugins", "htmlgraph", ".codex-plugin", "plugin.json")
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("plugin.json should NOT exist at old .agents/plugins/htmlgraph/ path %q (got stat err=%v)", oldPath, err)
	}

	// Plugin manifest must NOT exist at the flat marketplace root.
	rootPath := filepath.Join(outDir, ".codex-plugin", "plugin.json")
	if _, err := os.Stat(rootPath); !os.IsNotExist(err) {
		t.Errorf("plugin.json should NOT exist at marketplace root %q (got stat err=%v)", rootPath, err)
	}

	// Confirm plugin content is parseable.
	var plug codexPluginJSON
	readJSON(t, pluginManifest, &plug)
	if plug.Name == "" || plug.Version == "" {
		t.Errorf("plugin.json missing name/version: %+v", plug)
	}
}

// TestCodexAdapterRemovesStaleFilesMarketplace verifies that stale-file cleanup
// works correctly for the new marketplace layout. The owned subtree is narrowed
// to plugins/htmlgraph/; leftovers under htmlgraph/ must be removed, but the
// marketplace.json under .agents/plugins/ must survive the clean and be
// regenerated by Emit.
func TestCodexAdapterRemovesStaleFilesMarketplace(t *testing.T) {
	repoRoot := t.TempDir()
	seedAssets(t, repoRoot)
	outDir := filepath.Join(repoRoot, "packages", "codex-marketplace")

	// Seed a stale file deep inside the plugins/htmlgraph/ owned subtree.
	staleFile := filepath.Join(outDir, "plugins", "htmlgraph", "commands", "stale-cmd.md")
	if err := os.MkdirAll(filepath.Dir(staleFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staleFile, []byte("stale content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Seed a stale directory (simulating a renamed agent).
	staleDir := filepath.Join(outDir, "plugins", "htmlgraph", "agents", "old-agent")
	if err := os.MkdirAll(staleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staleDir, "README.md"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Seed a hand-maintained file at the marketplace root to verify it survives rebuild.
	handMaintained := filepath.Join(outDir, "README.md")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(handMaintained, []byte("hand-maintained readme"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := (codexAdapter{}).Emit(fixtureManifest(), repoRoot, outDir); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	// Stale command file must be gone (plugins/htmlgraph/ subtree was fully cleaned).
	if _, err := os.Stat(staleFile); !os.IsNotExist(err) {
		t.Errorf("expected stale command file to be removed; stat err=%v", err)
	}
	// Stale agent dir must be gone.
	if _, err := os.Stat(staleDir); !os.IsNotExist(err) {
		t.Errorf("expected stale agent dir to be removed; stat err=%v", err)
	}

	// Hand-maintained README must STILL EXIST (not in owned subtrees).
	if _, err := os.Stat(handMaintained); err != nil {
		t.Errorf("expected hand-maintained README to survive rebuild; stat err=%v", err)
	}

	// marketplace.json must exist after Emit (regenerated inside .agents/plugins/).
	mktPath := filepath.Join(outDir, ".agents", "plugins", "marketplace.json")
	if _, err := os.Stat(mktPath); err != nil {
		t.Errorf("marketplace.json must exist after Emit at %s: %v", mktPath, err)
	}
}

// TestCodexMarketplaceSourcePathForNestedSubdir verifies that source.path is
// computed correctly when pluginSubdir is nested deeper than the default
// plugins/htmlgraph. The path should be relative to the marketplace root (outDir)
// and use forward slashes.
func TestCodexMarketplaceSourcePathForNestedSubdir(t *testing.T) {
	repoRoot := t.TempDir()
	seedAssets(t, repoRoot)
	outDir := filepath.Join(repoRoot, "packages", "codex-marketplace")

	// Create a manifest with a nested pluginSubdir.
	m := fixtureManifest()
	// Can't assign to map value directly, so create new targets map.
	codexTarget := m.Targets["codex"]
	codexTarget.PluginSubdir = "plugins/nested/htmlgraph"
	m.Targets["codex"] = codexTarget

	if err := (codexAdapter{}).Emit(m, repoRoot, outDir); err != nil {
		t.Fatalf("Emit with nested subdir: %v", err)
	}

	// Read marketplace.json and verify source.path reflects the nested path.
	mktPath := filepath.Join(outDir, ".agents", "plugins", "marketplace.json")
	data, err := os.ReadFile(mktPath)
	if err != nil {
		t.Fatalf("read marketplace.json: %v", err)
	}

	var mkt codexMarketplaceJSON
	if err := json.Unmarshal(data, &mkt); err != nil {
		t.Fatalf("unmarshal marketplace.json: %v", err)
	}

	if len(mkt.Plugins) == 0 {
		t.Fatal("marketplace.json: plugins[] must not be empty")
	}
	plug := mkt.Plugins[0]
	expected := "./plugins/nested/htmlgraph"
	if plug.Source.Path != expected {
		t.Errorf("plugins[0].source.path = %q, want %q", plug.Source.Path, expected)
	}

	// Verify the plugin tree actually exists at the nested location.
	pluginManifest := filepath.Join(outDir, "plugins", "nested", "htmlgraph", ".codex-plugin", "plugin.json")
	if _, err := os.Stat(pluginManifest); err != nil {
		t.Errorf("expected plugin manifest at nested path %q: %v", pluginManifest, err)
	}
}

// TestCodexPluginJSONDeclaresSkillsPath verifies that the generated plugin.json
// includes the "skills": "./skills/" field, which tells Codex's TUI where to find
// SKILL.md files for mention autocomplete (e.g., $htmlgraph or $skill-name).
func TestCodexPluginJSONDeclaresSkillsPath(t *testing.T) {
	repoRoot := t.TempDir()
	seedAssets(t, repoRoot)
	outDir := filepath.Join(repoRoot, "packages", "codex-marketplace")

	if err := (codexAdapter{}).Emit(fixtureManifest(), repoRoot, outDir); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	pluginManifest := filepath.Join(outDir, "plugins", "htmlgraph", ".codex-plugin", "plugin.json")
	var plug codexPluginJSON
	readJSON(t, pluginManifest, &plug)

	if plug.Skills != "./skills/" {
		t.Errorf("plugin.json skills field = %q, want %q", plug.Skills, "./skills/")
	}
}
