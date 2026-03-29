package hooks

import (
	"database/sql"
	"fmt"
	"path/filepath"
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

	outputSummary := summariseToolOutput(event.ToolName, event.ToolInput, event.ToolResult, success)

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
	// Orchestrator analytics removed — stderr caused "hook error" in Claude Code UI.
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

	// YOLO advisory: after a successful git commit, remind agent to mark feature done.
	if advisory := checkYoloFeatureCompleteAdvisory(event, ctx, database); advisory != "" {
		if result.AdditionalContext != "" {
			result.AdditionalContext += "\n" + advisory
		} else {
			result.AdditionalContext = advisory
		}
	}

	return result, nil
}

// checkYoloFeatureCompleteAdvisory returns an advisory string when a successful
// git commit is made in YOLO mode and the active feature is still in-progress.
func checkYoloFeatureCompleteAdvisory(event *CloudEvent, ctx *toolUseContext, database *sql.DB) string {
	if event.ToolName != "Bash" {
		return ""
	}
	if !isSuccess(event.ToolResult) {
		return ""
	}
	if cmd := extractBashCommand(event.ToolInput); !looksLikeGitCommit(cmd) {
		return ""
	}
	projectDir := ResolveProjectDir(event.CWD)
	if !isYoloMode(filepath.Join(projectDir, ".htmlgraph")) {
		return ""
	}
	featureID := ctx.FeatureID
	if featureID == "" {
		return ""
	}
	var status string
	_ = database.QueryRow(`SELECT status FROM features WHERE id = ?`, featureID).Scan(&status)
	if status != "in-progress" {
		return ""
	}
	return fmt.Sprintf("Mark the feature complete: htmlgraph feature complete %s", featureID)
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

// summariseToolOutput builds a tool-specific structured output summary that
// captures key metadata (file path, success, content length) rather than raw
// output text. Falls back to summariseOutput for unrecognised tools.
func summariseToolOutput(toolName string, input map[string]any, result map[string]any, success bool) string {
	switch toolName {
	case "Read":
		return summariseReadOutput(input, result, success)
	case "Write":
		return summariseWriteOutput(input, success)
	case "Edit", "MultiEdit":
		return summariseEditOutput(input, success)
	case "Glob":
		return summariseGlobOutput(result, success)
	case "Grep":
		return summariseGrepOutput(result, success)
	default:
		return summariseOutput(result)
	}
}

func summariseReadOutput(input, result map[string]any, success bool) string {
	filePath := extractFilePath(input)
	if filePath == "" {
		filePath = "unknown"
	}
	if !success {
		return fmt.Sprintf("%s (error)", filePath)
	}
	// Count lines in content to report size.
	content := ""
	for _, key := range []string{"output", "content", "result"} {
		if v, ok := result[key].(string); ok {
			content = v
			break
		}
	}
	lines := countLines(content)
	return fmt.Sprintf("%s (ok, %d lines)", filePath, lines)
}

func summariseWriteOutput(input map[string]any, success bool) string {
	filePath := extractFilePath(input)
	if filePath == "" {
		filePath = "unknown"
	}
	if !success {
		return fmt.Sprintf("%s (error)", filePath)
	}
	return fmt.Sprintf("%s (written)", filePath)
}

func summariseEditOutput(input map[string]any, success bool) string {
	filePath := extractFilePath(input)
	if filePath == "" {
		filePath = "unknown"
	}
	if !success {
		return fmt.Sprintf("%s (error)", filePath)
	}
	return fmt.Sprintf("%s (edited)", filePath)
}

func summariseGlobOutput(result map[string]any, success bool) string {
	if !success {
		return "glob (error)"
	}
	content := ""
	for _, key := range []string{"output", "content", "result"} {
		if v, ok := result[key].(string); ok {
			content = v
			break
		}
	}
	n := countLines(content)
	return fmt.Sprintf("%d files matched", n)
}

func summariseGrepOutput(result map[string]any, success bool) string {
	if !success {
		return "grep (error)"
	}
	content := ""
	for _, key := range []string{"output", "content", "result"} {
		if v, ok := result[key].(string); ok {
			content = v
			break
		}
	}
	n := countLines(content)
	return fmt.Sprintf("%d matches", n)
}

// countLines returns the number of non-empty lines in s.
func countLines(s string) int {
	if s == "" {
		return 0
	}
	n := 1
	for i := range s {
		if s[i] == '\n' {
			n++
		}
	}
	// Don't count trailing newline as an extra line.
	if len(s) > 0 && s[len(s)-1] == '\n' {
		n--
	}
	return n
}
