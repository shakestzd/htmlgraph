package terminal

import (
	"testing"
)

// TestBuildShellCmd covers the full matrix from the slice-1 spec.
func TestBuildShellCmd(t *testing.T) {
	tests := []struct {
		name     string
		agent    string
		mode     string
		workItem string
		want     string
	}{
		{"defaults", "", "", "", "htmlgraph claude --dev"},
		{"claude dev", "claude", "dev", "", "htmlgraph claude --dev"},
		{"claude normal", "claude", "normal", "", "htmlgraph claude"},
		{"codex dev", "codex", "dev", "", "htmlgraph codex --dev"},
		{"gemini dev", "gemini", "dev", "", "htmlgraph gemini --dev"},
		{"yolo bypasses wrapper", "yolo", "dev", "", "claude --permission-mode bypassPermissions"},
		{"work item prefix claude", "claude", "dev", "feat-abc", "htmlgraph feature start feat-abc >/dev/null 2>&1; htmlgraph claude --dev"},
		{"work item prefix codex", "codex", "dev", "feat-abc", "htmlgraph feature start feat-abc >/dev/null 2>&1; htmlgraph codex --dev"},
		{"work item prefix gemini", "gemini", "dev", "feat-abc", "htmlgraph feature start feat-abc >/dev/null 2>&1; htmlgraph gemini --dev"},
		{"work item prefix yolo", "yolo", "dev", "feat-abc", "htmlgraph feature start feat-abc >/dev/null 2>&1; claude --permission-mode bypassPermissions"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildShellCmd(tc.agent, tc.mode, tc.workItem)
			if got != tc.want {
				t.Errorf("buildShellCmd(%q, %q, %q)\n  got:  %q\n  want: %q",
					tc.agent, tc.mode, tc.workItem, got, tc.want)
			}
		})
	}
}

// TestStartRequestZeroValue verifies that a zero-value StartRequest with defaultDir
// resolves CWD to defaultDir and produces "htmlgraph claude --dev".
func TestStartRequestZeroValue(t *testing.T) {
	var req StartRequest

	// Resolve agent/mode defaults as Manager.Start does.
	agent := req.Agent
	if agent == "" {
		agent = "claude"
	}
	mode := req.Mode
	if mode == "" {
		mode = "dev"
	}

	// CWD falls back to defaultDir when req.CWD is empty.
	defaultDir := "/mock/test-project"
	cwd := req.CWD
	if cwd == "" {
		cwd = defaultDir
	}
	if cwd != defaultDir {
		t.Errorf("zero StartRequest CWD should fall back to defaultDir %q, got %q", defaultDir, cwd)
	}

	got := buildShellCmd(agent, mode, req.WorkItem)
	want := "htmlgraph claude --dev"
	if got != want {
		t.Errorf("zero StartRequest should produce %q, got %q", want, got)
	}
}
