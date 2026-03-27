package hooks

import "fmt"

// delegateToolAgents maps tool names that the orchestrator should delegate
// to the recommended subagent type.
var delegateToolAgents = map[string]string{
	"Read":  "htmlgraph:researcher",
	"Grep":  "htmlgraph:researcher",
	"Glob":  "htmlgraph:researcher",
	"Edit":  "htmlgraph:sonnet-coder",
	"Write": "htmlgraph:sonnet-coder",
}

// buildOrchestratorContext returns advisory text for the orchestrator when:
//   - no work item is active (attribution warning), or
//   - a tool is used directly that should be delegated (delegation reminder).
//
// Both checks are advisory only — they never block execution.
func buildOrchestratorContext(toolName, featureID string) string {
	var ctx string

	if featureID == "" {
		ctx += "\n[Attribution] No active work item. Use `htmlgraph feature start <id>` to attribute your work."
	}

	if agent, ok := delegateToolAgents[toolName]; ok {
		ctx += fmt.Sprintf(
			"\n[Orchestrator] Direct %s usage detected. Consider delegating to %s.",
			toolName, agent,
		)
	}

	return ctx
}
