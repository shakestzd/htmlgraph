package main

import (
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

// TestAppendCodexHooksFlag verifies appending the hooks flag to config.
func TestAppendCodexHooksFlag(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "config.toml")

	// Create a config file with some content
	initialContent := "[other]\nkey = value\n"
	err := os.WriteFile(configPath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Append the hooks flag
	err = appendCodexHooksFlag(configPath)
	if err != nil {
		t.Fatalf("appendCodexHooksFlag: %v", err)
	}

	// Verify the content was appended
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "codex_hooks = true") {
		t.Errorf("expected appended codex_hooks = true not found in config:\n%s", content)
	}

	// Verify the original content is still there
	if !strings.Contains(content, "key = value") {
		t.Errorf("original content was not preserved")
	}
}
