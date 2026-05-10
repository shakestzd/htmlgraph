package harness_test

import (
	"sort"
	"testing"

	"github.com/shakestzd/wipnote/internal/harness"
	"github.com/shakestzd/wipnote/internal/hooks"
	"github.com/shakestzd/wipnote/internal/otel"
)

// TestRegistry_All_HasThree verifies that All() returns exactly three entries
// with the expected harness IDs.
func TestRegistry_All_HasThree(t *testing.T) {
	all := harness.All()
	if len(all) != 3 {
		t.Fatalf("All() returned %d entries, want 3", len(all))
	}

	ids := make([]string, len(all))
	for i, cfg := range all {
		ids[i] = cfg.ID
	}
	sort.Strings(ids)

	want := []string{"claude_code", "codex", "gemini_cli"}
	for i, id := range want {
		if ids[i] != id {
			t.Errorf("All() IDs[%d] = %q, want %q", i, ids[i], id)
		}
	}
}

// TestRegistry_Get_ByID checks lookup by canonical harness ID.
func TestRegistry_Get_ByID(t *testing.T) {
	cfg := harness.Get("codex")
	if cfg == nil {
		t.Fatal("Get(\"codex\") returned nil")
	}
	if cfg.AgentID != "codex" {
		t.Errorf("Get(\"codex\").AgentID = %q, want \"codex\"", cfg.AgentID)
	}

	if got := harness.Get("nonexistent"); got != nil {
		t.Errorf("Get(\"nonexistent\") = %v, want nil", got)
	}
}

// TestRegistry_GetByAgentID checks lookup by WIPNOTE_AGENT_ID value.
func TestRegistry_GetByAgentID(t *testing.T) {
	cfg := harness.GetByAgentID("gemini")
	if cfg == nil {
		t.Fatal("GetByAgentID(\"gemini\") returned nil")
	}
	if cfg.ID != "gemini_cli" {
		t.Errorf("GetByAgentID(\"gemini\").ID = %q, want \"gemini_cli\"", cfg.ID)
	}

	if got := harness.GetByAgentID("nonexistent"); got != nil {
		t.Errorf("GetByAgentID(\"nonexistent\") = %v, want nil", got)
	}
}

// TestRegistry_GetByHooksHarness checks lookup by HooksHarness enum value.
func TestRegistry_GetByHooksHarness(t *testing.T) {
	cfg := harness.GetByHooksHarness(harness.HooksGemini)
	if cfg == nil {
		t.Fatal("GetByHooksHarness(HooksGemini) returned nil")
	}
	if cfg.ID != "gemini_cli" {
		t.Errorf("GetByHooksHarness(HooksGemini).ID = %q, want \"gemini_cli\"", cfg.ID)
	}
}

// TestRegistry_GeminiEventNames verifies the Gemini hook event names are populated.
func TestRegistry_GeminiEventNames(t *testing.T) {
	cfg := harness.Get("gemini_cli")
	if cfg == nil {
		t.Fatal("Get(\"gemini_cli\") returned nil")
	}
	if len(cfg.HookEventNames) == 0 {
		t.Fatal("Get(\"gemini_cli\").HookEventNames is empty")
	}

	found := false
	for _, name := range cfg.HookEventNames {
		if name == "BeforeAgent" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Get(\"gemini_cli\").HookEventNames does not contain \"BeforeAgent\"; got %v", cfg.HookEventNames)
	}
}

// TestRegistry_IDsMatchOtelConsts bridges the harness package and otel package
// via tests (harness itself does NOT import otel).
func TestRegistry_IDsMatchOtelConsts(t *testing.T) {
	tests := []struct {
		otelConst otel.Harness
		harnessID string
	}{
		{otel.HarnessClaude, "claude_code"},
		{otel.HarnessCodex, "codex"},
		{otel.HarnessGemini, "gemini_cli"},
	}

	for _, tt := range tests {
		if string(tt.otelConst) != tt.harnessID {
			t.Errorf("otel.%s = %q, want %q", tt.harnessID, string(tt.otelConst), tt.harnessID)
		}
		cfg := harness.Get(tt.harnessID)
		if cfg == nil {
			t.Errorf("harness.Get(%q) returned nil", tt.harnessID)
			continue
		}
		if cfg.ID != string(tt.otelConst) {
			t.Errorf("harness.Get(%q).ID = %q, want %q (otel const)", tt.harnessID, cfg.ID, string(tt.otelConst))
		}
	}
}

// TestRegistry_HooksHarnessMatchesHooksConst verifies iota ordering alignment
// between internal/harness and internal/hooks.
func TestRegistry_HooksHarnessMatchesHooksConst(t *testing.T) {
	tests := []struct {
		harnessVal harness.HooksHarness
		hooksVal   hooks.Harness
		name       string
	}{
		{harness.HooksClaude, hooks.HarnessClaude, "Claude"},
		{harness.HooksCodex, hooks.HarnessCodex, "Codex"},
		{harness.HooksGemini, hooks.HarnessGemini, "Gemini"},
	}

	for _, tt := range tests {
		if int(tt.harnessVal) != int(tt.hooksVal) {
			t.Errorf("%s: harness.Hooks%s=%d != hooks.Harness%s=%d",
				tt.name, tt.name, int(tt.harnessVal), tt.name, int(tt.hooksVal))
		}
	}
}
