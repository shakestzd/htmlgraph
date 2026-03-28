package hooks

import (
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// isYoloMode checks if the current session is in YOLO mode by reading
// the .htmlgraph/.launch-mode file.
func isYoloMode(htmlgraphDir string) bool {
	data, err := os.ReadFile(filepath.Join(htmlgraphDir, ".launch-mode"))
	if err != nil {
		return false
	}
	return strings.Contains(string(data), `"yolo`)
}

// checkYoloWorkItemGuard blocks Write/Edit tools when no active feature
// exists in YOLO mode. Returns a non-empty reason to block, or "" to allow.
func checkYoloWorkItemGuard(toolName, featureID string, yolo bool) string {
	if !yolo {
		return ""
	}
	switch toolName {
	case "Write", "Edit", "MultiEdit":
	default:
		return ""
	}
	if featureID != "" {
		return ""
	}
	return "YOLO mode requires an active work item before writing code. " +
		"Create or start a feature first: htmlgraph feature create \"title\""
}

// gitCommitPattern matches git commit commands in Bash.
var gitCommitPattern = regexp.MustCompile(`\bgit\s+commit\b`)

// checkYoloCommitGuard blocks git commit when tests haven't run in
// the current session. Returns a non-empty reason to block, or "" to allow.
func checkYoloCommitGuard(event *CloudEvent, yolo, testRan bool) string {
	if !yolo {
		return ""
	}
	if event.ToolName != "Bash" {
		return ""
	}
	cmd, _ := event.ToolInput["command"].(string)
	if !gitCommitPattern.MatchString(cmd) {
		return ""
	}
	if testRan {
		return ""
	}
	return "YOLO mode requires tests to pass before committing. " +
		"Run: go test ./... or uv run pytest"
}

// testPattern matches common test runner commands in Bash input summaries.
var testPattern = regexp.MustCompile(`\bgo test\b|\bpytest\b|\buv run pytest\b|\buv run ruff\b`)

// hasRecentTestRun checks if a test command was executed in this session
// by scanning recent agent_events for Bash commands matching test patterns.
func hasRecentTestRun(database *sql.DB, sessionID string) bool {
	var count int
	database.QueryRow(`
		SELECT COUNT(*) FROM agent_events
		WHERE session_id = ? AND tool_name = 'Bash'
		  AND status = 'completed'
		  AND input_summary REGEXP ?`,
		sessionID, `(go test|pytest|uv run ruff)`,
	).Scan(&count)
	if count > 0 {
		return true
	}
	// Fallback: LIKE-based check for SQLite without REGEXP
	database.QueryRow(`
		SELECT COUNT(*) FROM agent_events
		WHERE session_id = ? AND tool_name = 'Bash'
		  AND (input_summary LIKE '%go test%'
		    OR input_summary LIKE '%pytest%'
		    OR input_summary LIKE '%uv run ruff%')`,
		sessionID,
	).Scan(&count)
	return count > 0
}
