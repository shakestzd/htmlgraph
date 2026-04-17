package pluginbuild

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// fixtureManifest is a minimal, self-contained manifest used to exercise both
// adapters without depending on the live packages/plugin-core/manifest.json.
func fixtureManifest() *Manifest {
	return &Manifest{
		Name:        "htmlgraph",
		Version:     "0.0.0-test",
		Description: "test plugin",
		Author:      Author{Name: "Tester", Email: "t@example.com"},
		Homepage:    "https://example.com",
		Repository:  "https://example.com/repo",
		License:     "MIT",
		Category:    "Dev",
		Keywords:    []string{"test"},
		Targets: map[string]Target{
			"claude": {OutDir: "plugin", ManifestPath: ".claude-plugin/plugin.json", HooksPath: "hooks/hooks.json"},
			"codex":  {OutDir: "packages/codex-plugin", ManifestPath: ".codex-plugin/plugin.json", HooksPath: "hooks.json", MCPPath: ".mcp.json"},
		},
		AssetSources: AssetSources{
			Commands: "plugin/commands",
			Agents:   "plugin/agents",
		},
		Hooks: HookMatrix{Events: []HookEvent{
			{Name: "SessionStart", Handler: "session-start", Targets: []string{"claude", "codex"}},
			{Name: "UserPromptSubmit", Handler: "user-prompt", Targets: []string{"claude", "codex"}},
			{Name: "Stop", Handler: "stop", Targets: []string{"claude"}},
			{Name: "TaskStarted", Handler: "task-started", Targets: []string{"codex"}},
			{Name: "SessionStart", Command: "date", Timeout: 2, Targets: []string{"claude"}, Matcher: "resume"},
		}},
	}
}

func TestClaudeAdapterEmitsManifestAndHooks(t *testing.T) {
	repoRoot := t.TempDir()
	seedAssets(t, repoRoot)
	outDir := filepath.Join(repoRoot, "plugin")

	if err := (claudeAdapter{}).Emit(fixtureManifest(), repoRoot, outDir); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	var plug claudePluginJSON
	readJSON(t, filepath.Join(outDir, ".claude-plugin", "plugin.json"), &plug)
	if plug.Name != "htmlgraph" || plug.Version != "0.0.0-test" {
		t.Fatalf("claude manifest wrong: %+v", plug)
	}

	hooksRaw, err := os.ReadFile(filepath.Join(outDir, "hooks", "hooks.json"))
	if err != nil {
		t.Fatalf("read hooks.json: %v", err)
	}
	// Hooks assertions: Claude-only events present, Codex-only absent.
	s := string(hooksRaw)
	for _, want := range []string{`"SessionStart"`, `"Stop"`, `"UserPromptSubmit"`, `"htmlgraph hook session-start"`, `"htmlgraph hook stop"`} {
		if !contains(s, want) {
			t.Errorf("claude hooks missing %q:\n%s", want, s)
		}
	}
	for _, notWant := range []string{`"TaskStarted"`} {
		if contains(s, notWant) {
			t.Errorf("claude hooks should not contain Codex-only %q", notWant)
		}
	}
	// Asset copy
	if _, err := os.Stat(filepath.Join(outDir, "commands", "hello.md")); err != nil {
		t.Errorf("expected copied command: %v", err)
	}
}

func TestCodexAdapterEmitsManifestHooksAndMCP(t *testing.T) {
	repoRoot := t.TempDir()
	seedAssets(t, repoRoot)
	outDir := filepath.Join(repoRoot, "packages", "codex-plugin")

	if err := (codexAdapter{}).Emit(fixtureManifest(), repoRoot, outDir); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	var plug codexPluginJSON
	readJSON(t, filepath.Join(outDir, ".codex-plugin", "plugin.json"), &plug)
	if plug.Interface.DisplayName != "HtmlGraph" {
		t.Errorf("codex interface.displayName: %+v", plug.Interface)
	}
	if plug.Author.Email != "t@example.com" {
		t.Errorf("codex author.email: %+v", plug.Author)
	}

	hooksRaw, err := os.ReadFile(filepath.Join(outDir, "hooks.json"))
	if err != nil {
		t.Fatalf("read hooks.json: %v", err)
	}
	s := string(hooksRaw)
	// Codex-only events present, Claude-only absent.
	for _, want := range []string{`"SessionStart"`, `"UserPromptSubmit"`, `"TaskStarted"`, `"htmlgraph hook task-started"`} {
		if !contains(s, want) {
			t.Errorf("codex hooks missing %q:\n%s", want, s)
		}
	}
	if contains(s, `"Stop"`) {
		t.Errorf("codex hooks should not contain Claude-only Stop event")
	}

	// .mcp.json stub written.
	if _, err := os.Stat(filepath.Join(outDir, ".mcp.json")); err != nil {
		t.Errorf("expected .mcp.json stub: %v", err)
	}
}

func TestLoadAndValidateRejectsBadManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	os.WriteFile(path, []byte(`{"name":"","version":"1","targets":{"x":{"outDir":"a","manifestPath":"b","hooksPath":"c"}}}`), 0o644)
	if _, err := Load(path); err == nil {
		t.Fatal("expected error on empty name")
	}
}

func TestFindManifestWalksUp(t *testing.T) {
	root := t.TempDir()
	corePath := filepath.Join(root, "packages", "plugin-core")
	if err := os.MkdirAll(corePath, 0o755); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(corePath, "manifest.json")
	writeFixtureManifest(t, manifestPath)

	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	found, err := FindManifest(deep)
	if err != nil {
		t.Fatalf("FindManifest: %v", err)
	}
	if found != manifestPath {
		t.Errorf("found=%q want=%q", found, manifestPath)
	}
}

func TestHookEventAppliesTo(t *testing.T) {
	e := HookEvent{Targets: []string{"claude", "codex"}}
	if !e.AppliesTo("claude") || !e.AppliesTo("codex") || e.AppliesTo("gemini") {
		t.Fatalf("AppliesTo mismatch")
	}
}

// --- helpers ---

func seedAssets(t *testing.T, repoRoot string) {
	t.Helper()
	cmdDir := filepath.Join(repoRoot, "plugin", "commands")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cmdDir, "hello.md"), []byte("# hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	agDir := filepath.Join(repoRoot, "plugin", "agents")
	if err := os.MkdirAll(agDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agDir, "x.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeFixtureManifest(t *testing.T, path string) {
	t.Helper()
	data, err := json.MarshalIndent(fixtureManifest(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readJSON(t *testing.T, path string, into any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, into); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
