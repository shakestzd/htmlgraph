package hooks

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/models"
)

// PostToolUse handles the PostToolUse Claude Code hook event.
// It finds the most recent "started" event for this session/tool and marks it completed.
// Note: env vars don't persist between hook processes, so we query the DB instead.
func PostToolUse(event *CloudEvent, database *sql.DB) (*HookResult, error) {
	ctx := resolveToolUseContext(event, database)
	if ctx == nil {
		return &HookResult{Continue: true}, nil
	}

	success := isSuccess(event.ToolResult)
	status := "completed"
	if !success {
		status = "failed"
	}

	outputSummary := summariseOutput(event.ToolResult)

	// For subagent events, scope the lookup to this specific agent to avoid
	// completing events belonging to a different concurrent agent.
	var (
		eventID string
		err     error
	)
	if ctx.IsSubagent {
		eventID, err = db.FindStartedEventByAgent(database, ctx.SessionID, event.ToolName, ctx.AgentID)
		if err != nil {
			// Fall back to unscoped lookup when no agent-specific event exists.
			eventID, err = db.FindStartedEvent(database, ctx.SessionID, event.ToolName)
		}
	} else {
		eventID, err = db.FindStartedEvent(database, ctx.SessionID, event.ToolName)
	}
	if err != nil {
		return &HookResult{Continue: true}, nil
	}

	_ = db.UpdateEventFields(database, eventID, status, outputSummary)

	// Record orchestrator direct-tool usage for analytics.
	// Subagents are excluded — only direct orchestrator use is interesting here.
	if !ctx.IsSubagent {
		recordOrchestratorToolUse(database, ctx.SessionID, event.ToolName, success)
	}

	// Capture git commits and link to the active work item.
	if event.ToolName == "Bash" {
		if cmd := extractBashCommand(event.ToolInput); looksLikeGitCommit(cmd) {
			if hash, msg := parseGitCommitOutput(summarizeToolOutput(event.ToolResult)); hash != "" {
				commit := &models.GitCommit{
					CommitHash:  hash,
					SessionID:   ctx.SessionID,
					FeatureID:   ctx.FeatureID,
					ToolEventID: eventID,
					Message:     msg,
					Timestamp:   time.Now().UTC(),
				}
				_ = db.InsertGitCommit(database, commit)
			}
		}
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

// gitCommitOutputRe matches the commit line from git commit output, e.g.:
// "[main abc1234] commit message here"
var gitCommitOutputRe = regexp.MustCompile(`\[[\w/\-]+\s+([0-9a-f]{7,40})\]\s+(.*)`)

// looksLikeGitCommit returns true when the bash command appears to be a git commit.
func looksLikeGitCommit(cmd string) bool {
	return strings.Contains(cmd, "git commit") || strings.Contains(cmd, "git-commit")
}

// parseGitCommitOutput extracts the commit hash and message from git's stdout.
// Returns ("", "") when the output does not match the expected format.
func parseGitCommitOutput(output string) (hash, message string) {
	for _, line := range strings.Split(output, "\n") {
		if m := gitCommitOutputRe.FindStringSubmatch(strings.TrimSpace(line)); len(m) == 3 {
			return m[1], strings.TrimSpace(m[2])
		}
	}
	return "", ""
}

// extractBashCommand extracts the "command" field from a Bash tool_input map.
func extractBashCommand(input map[string]any) string {
	if input == nil {
		return ""
	}
	if v, ok := input["command"].(string); ok {
		return v
	}
	return ""
}

// summarizeToolOutput extracts the full output string from a tool result for
// commit parsing (we need more than the 200-char summariseOutput truncation).
func summarizeToolOutput(result map[string]any) string {
	if result == nil {
		return ""
	}
	for _, key := range []string{"output", "content", "result"} {
		if v, ok := result[key].(string); ok {
			return v
		}
	}
	return ""
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
