package harness

// _codexOtelID must match otel.HarnessCodex ("codex").
// Cross-package assertion is in registry_test.go (TestRegistry_IDsMatchOtelConsts).
const _codexOtelID = "codex"

func init() {
	if _codexOtelID != "codex" {
		panic("harness: _codexOtelID mismatch — must equal otel.HarnessCodex")
	}
	Register(&HarnessConfig{
		ID:      _codexOtelID,
		AgentID: "codex",
		// Codex emits two service.name variants: the TypeScript CLI and the Rust rewrite.
		ServiceNames:   []string{"codex-cli", "codex_cli_rs"},
		SessionAttr:    "conversation.id",
		HookEventNames: nil,
		HooksHarness:   HooksCodex,
	})
}
