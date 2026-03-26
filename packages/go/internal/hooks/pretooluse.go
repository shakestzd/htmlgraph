package hooks

import (
	"database/sql"
	"encoding/json"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/models"
)

// PreToolUse handles the PreToolUse Claude Code hook event.
// It inserts a tool_call agent_event row and allows the tool to proceed.
func PreToolUse(event *CloudEvent, database *sql.DB) (*HookResult, error) {
	sessionID := EnvSessionID(event.SessionID)
	if sessionID == "" {
		return &HookResult{Decision: "allow"}, nil
	}

	// Guard: never intercept writes to .htmlgraph/ — mirror of
	// pretooluse-htmlgraph-guard.py to prevent accidental DB corruption.
	if isHtmlGraphWrite(event) {
		return &HookResult{
			Decision: "block",
			Reason:   ".htmlgraph/ is managed by HtmlGraph SDK. Use SDK methods instead.",
		}, nil
	}

	featureID := GetActiveFeatureID(database, sessionID)
	parentEventID := os.Getenv("HTMLGRAPH_PARENT_EVENT")

	// Multi-method parent resolution (matches Python event_tracker.py):
	// 1. Env var HTMLGRAPH_PARENT_EVENT (set by SubagentStart for subagent processes)
	// 2. Most recent started task_delegation in this session (for subagent tool calls)
	// 3. Most recent UserQuery in this session (for top-level tool calls)
	if parentEventID == "" {
		_ = database.QueryRow(
			`SELECT event_id FROM agent_events WHERE session_id = ? AND event_type IN ('task_delegation', 'delegation') AND status = 'started' ORDER BY timestamp DESC LIMIT 1`,
			sessionID,
		).Scan(&parentEventID)
	}
	if parentEventID == "" {
		_ = database.QueryRow(
			`SELECT event_id FROM agent_events WHERE session_id = ? AND tool_name = 'UserQuery' ORDER BY timestamp DESC LIMIT 1`,
			sessionID,
		).Scan(&parentEventID)
	}

	inputSummary := summariseInput(event.ToolName, event.ToolInput)

	ev := &models.AgentEvent{
		EventID:       uuid.New().String(),
		AgentID:       agentIDFromEnv(),
		EventType:     models.EventToolCall,
		Timestamp:     time.Now().UTC(),
		ToolName:      event.ToolName,
		InputSummary:  inputSummary,
		SessionID:     sessionID,
		FeatureID:     featureID,
		ParentEventID: parentEventID,
		Status:        "started",
		Source:        "hook",
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	// Ignore insert errors (FK violations are expected before session-start runs).
	_ = db.InsertEvent(database, ev)

	// Export event ID so posttooluse can link the result.
	os.Setenv("HTMLGRAPH_CURRENT_EVENT_ID", ev.EventID)

	return &HookResult{Decision: "allow"}, nil
}

// isHtmlGraphWrite returns true for file-write tools targeting .htmlgraph/.
func isHtmlGraphWrite(event *CloudEvent) bool {
	switch event.ToolName {
	case "Write", "Edit", "MultiEdit":
	default:
		return false
	}
	path, _ := event.ToolInput["path"].(string)
	if path == "" {
		path, _ = event.ToolInput["file_path"].(string)
	}
	return containsHtmlgraphDir(path)
}

func containsHtmlgraphDir(path string) bool {
	for i := range path {
		if path[i] == '.' && i+11 <= len(path) && path[i:i+11] == ".htmlgraph/" {
			return true
		}
	}
	return path == ".htmlgraph"
}

// summariseInput builds a short human-readable summary of tool input.
func summariseInput(toolName string, input map[string]any) string {
	if input == nil {
		return toolName
	}
	// For file tools, use the path.
	for _, key := range []string{"path", "file_path", "command", "query", "prompt"} {
		if v, ok := input[key].(string); ok && v != "" {
			if len(v) > 120 {
				v = v[:120] + "…"
			}
			return v
		}
	}
	// Fallback: compact JSON of first 200 chars.
	b, _ := json.Marshal(input)
	s := string(b)
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}

// agentIDFromEnv returns the current agent ID.
func agentIDFromEnv() string {
	if v := os.Getenv("HTMLGRAPH_AGENT_ID"); v != "" {
		return v
	}
	return "claude-code"
}
