package hooks

import (
	"database/sql"
	"fmt"
	"os"
	"time"
)

// PostToolUse handles the PostToolUse Claude Code hook event.
// It finds the most recent "started" event for this session/tool and marks it completed.
// Note: env vars don't persist between hook processes, so we query the DB instead.
func PostToolUse(event *CloudEvent, database *sql.DB) (*HookResult, error) {
	sessionID := EnvSessionID(event.SessionID)
	if sessionID == "" {
		return &HookResult{Continue: true}, nil
	}

	success := isSuccess(event.ToolResult)
	status := "completed"
	if !success {
		status = "failed"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	outputSummary := summariseOutput(event.ToolResult)

	// Find the most recent "started" event for this session and tool.
	var eventID string
	err := database.QueryRow(`
		SELECT event_id FROM agent_events
		WHERE session_id = ? AND tool_name = ? AND status = 'started'
		ORDER BY timestamp DESC
		LIMIT 1`, sessionID, event.ToolName,
	).Scan(&eventID)

	if err != nil {
		return &HookResult{Continue: true}, nil
	}

	_, _ = database.Exec(`
		UPDATE agent_events
		SET status = ?,
		    output_summary = ?,
		    updated_at = ?
		WHERE event_id = ?`,
		status, outputSummary, now, eventID,
	)

	// Record orchestrator direct-tool usage for analytics.
	// Subagents are excluded — only direct orchestrator use is interesting here.
	isSubagent := event.AgentID != "" && event.AgentID != "claude-code"
	if !isSubagent {
		recordOrchestratorToolUse(database, sessionID, event.ToolName, success)
	}

	result := &HookResult{Continue: true}

	// Quality gate: warn when Write/Edit/MultiEdit produces an oversized file.
	switch event.ToolName {
	case "Write", "Edit", "MultiEdit":
		if filePath := extractFilePath(event.ToolInput); filePath != "" {
			if warnings := CheckFileQuality(filePath); warnings != "" {
				result.AdditionalContext = warnings
			}
		}
	}

	return result, nil
}

// recordOrchestratorToolUse emits a structured log line to stderr when the
// orchestrator uses a delegatable tool directly. This is picked up by
// Claude Code's hook debug output and serves as lightweight analytics
// without requiring a dedicated DB table.
func recordOrchestratorToolUse(_ *sql.DB, sessionID, toolName string, success bool) {
	if _, ok := delegateToolAgents[toolName]; !ok {
		return // only track tools that should be delegated
	}
	status := "completed"
	if !success {
		status = "failed"
	}
	fmt.Fprintf(os.Stderr,
		"[htmlgraph] orchestrator_direct_tool session=%s tool=%s status=%s ts=%s\n",
		sessionID, toolName, status, time.Now().UTC().Format(time.RFC3339),
	)
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
