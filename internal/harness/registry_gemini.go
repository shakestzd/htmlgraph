package harness

// _geminiOtelID must match otel.HarnessGemini ("gemini_cli").
// Cross-package assertion is in registry_test.go (TestRegistry_IDsMatchOtelConsts).
const _geminiOtelID = "gemini_cli"

func init() {
	if _geminiOtelID != "gemini_cli" {
		panic("harness: _geminiOtelID mismatch — must equal otel.HarnessGemini")
	}
	Register(&HarnessConfig{
		ID:           _geminiOtelID,
		AgentID:      "gemini",
		ServiceNames: []string{"gemini-cli"},
		SessionAttr:  "session.id",
		// Gemini-native hook_event_name values used by detectHarnessWithEnv for
		// payload-only discrimination when WIPNOTE_AGENT_ID is not set.
		HookEventNames: []string{
			"BeforeAgent",
			"AfterAgent",
			"AfterModel",
			"BeforeTool",
			"AfterTool",
		},
		HooksHarness: HooksGemini,
	})
}
