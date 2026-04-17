package pluginbuild

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToGeminiCommandTOMLWrapsBody(t *testing.T) {
	got := toGeminiCommandTOML("# hello\nbody")
	if !strings.Contains(got, `prompt = """`) {
		t.Errorf("missing triple-quote prompt opener:\n%s", got)
	}
	if !strings.Contains(got, "body") {
		t.Errorf("missing body content:\n%s", got)
	}
	if !strings.HasSuffix(got, "\"\"\"\n") {
		t.Errorf("missing triple-quote close:\n%s", got)
	}
}

func TestToGeminiCommandTOMLEscapesTripleQuote(t *testing.T) {
	// A literal """ in the body would prematurely terminate the TOML string.
	// The helper must break it so the resulting TOML parses with exactly one
	// prompt value. We verify by counting unescaped triple-quote runs — there
	// should be exactly two (opener and closer) for the wrapper.
	body := "before\n\"\"\"inside\"\"\"\nafter"
	got := toGeminiCommandTOML(body)
	if strings.Contains(got, "\n\"\"\"inside") {
		t.Errorf("raw triple-quote survived, would break TOML parse:\n%s", got)
	}
	// The escaped form appears as ""\" — assert its presence.
	if !strings.Contains(got, `""\"`) {
		t.Errorf("expected escaped form \"\"\\\" in output:\n%s", got)
	}
	// The body content (without the bare triple-quotes) must still be there.
	if !strings.Contains(got, "before") || !strings.Contains(got, "inside") || !strings.Contains(got, "after") {
		t.Errorf("body content lost during escape:\n%s", got)
	}
}

func TestGeminiAdapterEmitsCommandsTOML(t *testing.T) {
	repoRoot := t.TempDir()
	cmdDir := filepath.Join(repoRoot, "plugin", "commands")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cmdDir, "hello.md"), []byte("# hello\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(repoRoot, "packages", "gemini-extension")
	if err := (geminiAdapter{}).Emit(fixtureManifest(), repoRoot, outDir); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	tomlPath := filepath.Join(outDir, "commands", "htmlgraph", "hello.toml")
	data, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatalf("read emitted toml: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `prompt = """`) {
		t.Errorf("emitted toml missing prompt opener:\n%s", s)
	}
	if !strings.Contains(s, "# hello") || !strings.Contains(s, "body") {
		t.Errorf("emitted toml missing markdown body:\n%s", s)
	}
}

// TestGeminiAdapterCommandParity asserts that every .md in plugin/commands/
// produces exactly one .toml under commands/<namespace>/ in the emitted tree.
// This guards against silent drops (e.g. filter bugs) when new commands are
// added to plugin/commands/.
func TestGeminiAdapterCommandParity(t *testing.T) {
	manifestPath, err := FindManifest(".")
	if err != nil {
		t.Skipf("no live manifest (pre-integration test): %v", err)
	}
	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(manifestPath)))

	srcDir := filepath.Join(repoRoot, m.AssetSources.Commands)
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		t.Fatalf("read live commands dir: %v", err)
	}
	var wantCount int
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			wantCount++
		}
	}
	if wantCount == 0 {
		t.Fatalf("expected plugin/commands/*.md to exist, found 0")
	}

	outDir := t.TempDir()
	target := m.Targets["gemini"]
	if err := emitGeminiCommands(m, repoRoot, outDir, target); err != nil {
		t.Fatalf("emitGeminiCommands: %v", err)
	}

	dstDir := filepath.Join(outDir, "commands", target.CommandNamespace)
	out, err := os.ReadDir(dstDir)
	if err != nil {
		t.Fatalf("read emitted toml dir: %v", err)
	}
	var gotCount int
	for _, e := range out {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".toml") {
			gotCount++
		}
	}
	if gotCount != wantCount {
		t.Errorf("parity mismatch: %d .md in source, %d .toml emitted", wantCount, gotCount)
	}
}
