package pluginbuild

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGeminiAdapterEmitsHooksFromFixture exercises the Gemini hooks sub-emitter
// against the fixture manifest. It asserts that:
//   - `hooks/hooks.json` is written with the SessionStart event and its mapped
//     `htmlgraph hook session-start` command.
//   - Codex-only events (TaskStarted) do not leak into the Gemini output.
//   - Claude-only matcher variants (SessionStart + `session-resume` / matcher
//     "resume") do not leak into the Gemini output.
func TestGeminiAdapterEmitsHooksFromFixture(t *testing.T) {
	repoRoot := t.TempDir()
	seedAssets(t, repoRoot)
	outDir := filepath.Join(repoRoot, "packages", "gemini-extension")
	seedAssets(t, repoRoot)
	// Phase 1's sub-emitter copies repo-root GEMINI.md; seed a placeholder so
	// the full Emit chain doesn't fail before Phase 3's hook assertion runs.
	if err := os.WriteFile(filepath.Join(repoRoot, "GEMINI.md"), []byte("# ctx\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := fixtureManifest()
	// Tag the SessionStart/UserPromptSubmit/Stop fixture events for Gemini to
	// mirror the live manifest without requiring fixtureManifest() edits.
	for i := range m.Hooks.Events {
		e := &m.Hooks.Events[i]
		switch {
		case e.Name == "SessionStart" && e.Handler == "session-start":
			e.Targets = append(e.Targets, "gemini")
		case e.Name == "UserPromptSubmit" && e.Handler == "user-prompt":
			e.Targets = append(e.Targets, "gemini")
		case e.Name == "Stop" && e.Handler == "stop":
			e.Targets = append(e.Targets, "gemini")
		}
	}

	if err := (geminiAdapter{}).Emit(m, repoRoot, outDir); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	hooksRaw, err := os.ReadFile(filepath.Join(outDir, "hooks", "hooks.json"))
	if err != nil {
		t.Fatalf("read hooks.json: %v", err)
	}
	s := string(hooksRaw)

	for _, want := range []string{`"SessionStart"`, `"htmlgraph hook session-start"`} {
		if !strings.Contains(s, want) {
			t.Errorf("gemini hooks missing %q:\n%s", want, s)
		}
	}
	// Codex-only event must not leak through.
	if strings.Contains(s, `"TaskStarted"`) {
		t.Errorf("gemini hooks should not contain Codex-only TaskStarted:\n%s", s)
	}
	// Claude-only matcher variant (resume) must not leak through — the fixture's
	// matcher:"resume" entry is claude-only and must be filtered out.
	if strings.Contains(s, `"resume"`) {
		t.Errorf("gemini hooks should not contain Claude-only matcher %q:\n%s", "resume", s)
	}
}

// TestGeminiParityFromLiveManifest loads the real manifest and confirms the
// five conservative events Phase 3 tagged for Gemini appear in the emitted
// hooks.json. This guards against manifest drift: if anyone drops "gemini"
// from a targets list, this test fails loudly.
func TestGeminiParityFromLiveManifest(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	manifestPath, err := FindManifest(cwd)
	if err != nil {
		t.Fatalf("FindManifest: %v", err)
	}
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(manifestPath)))

	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	outDir := t.TempDir()
	if err := (geminiAdapter{}).Emit(m, repoRoot, outDir); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	hooksBytes, err := os.ReadFile(filepath.Join(outDir, "hooks", "hooks.json"))
	if err != nil {
		t.Fatalf("read hooks.json: %v", err)
	}
	hooks := string(hooksBytes)

	for _, want := range []string{
		`"SessionStart"`,
		`"UserPromptSubmit"`,
		`"PreToolUse"`,
		`"PostToolUse"`,
		`"Stop"`,
	} {
		if !strings.Contains(hooks, want) {
			t.Errorf("gemini hooks missing %s", want)
		}
	}
	// Codex-only and Claude-only variants must not appear in the Gemini output.
	for _, notWant := range []string{
		`"TaskStarted"`,
		`"TurnAborted"`,
		`"TaskComplete"`,
		`"ExitPlanMode"`,
	} {
		if strings.Contains(hooks, notWant) {
			t.Errorf("gemini hooks contains disallowed %s", notWant)
		}
	}
}
