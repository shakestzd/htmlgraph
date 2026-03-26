package hooks

import (
	"database/sql"
	"fmt"
	"os"
	"time"
)

// PostToolUse handles the PostToolUse Claude Code hook event.
// It updates the agent_event row written by PreToolUse with the tool result.
func PostToolUse(event *CloudEvent, database *sql.DB) (*HookResult, error) {
	eventID := os.Getenv("HTMLGRAPH_CURRENT_EVENT_ID")
	if eventID == "" {
		// PreToolUse didn't fire or was skipped — nothing to update.
		return &HookResult{Continue: true}, nil
	}

	success := isSuccess(event.ToolResult)
	status := "completed"
	if !success {
		status = "failed"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	outputSummary := summariseOutput(event.ToolResult)

	_, err := database.Exec(`
		UPDATE agent_events
		SET status = ?,
		    output_summary = ?,
		    updated_at = ?
		WHERE event_id = ?`,
		status, outputSummary, now, eventID,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "htmlgraph posttooluse: db error: %v\n", err)
	}

	// Clear env var so the next tool gets a fresh slot.
	os.Unsetenv("HTMLGRAPH_CURRENT_EVENT_ID")

	return &HookResult{Continue: true}, nil
}

// isSuccess returns false when the tool result contains an explicit error flag.
func isSuccess(result map[string]any) bool {
	if result == nil {
		return true
	}
	if v, ok := result["is_error"].(bool); ok && v {
		return false
	}
	return true
}

// summariseOutput extracts a short string from the tool result map.
func summariseOutput(result map[string]any) string {
	if result == nil {
		return ""
	}
	for _, key := range []string{"output", "content", "result", "error"} {
		if v, ok := result[key].(string); ok && v != "" {
			if len(v) > 200 {
				v = v[:200] + "…"
			}
			return v
		}
	}
	return ""
}

// ensure sql is referenced (used indirectly via nullable helpers).
var _ *sql.DB
