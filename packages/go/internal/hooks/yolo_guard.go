package hooks

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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

// checkYoloWorkItemGuard blocks Write/Edit tools when no active work item
// exists in YOLO mode. Returns a non-empty reason to block, or "" to allow.
//
// featureID is the session's active_feature_id column (set at session-start).
// hasInProgressItem is true when a feature or bug with status="in-progress"
// exists in the DB — this covers items started mid-session via
// `htmlgraph feature start` / `htmlgraph bug start` which update the features
// table but do NOT update sessions.active_feature_id.
func checkYoloWorkItemGuard(toolName, featureID string, yolo, hasInProgressItem bool) string {
	if !yolo {
		return ""
	}
	switch toolName {
	case "Write", "Edit", "MultiEdit":
	default:
		return ""
	}
	if featureID != "" || hasInProgressItem {
		return ""
	}
	return "YOLO mode requires an active work item before writing code. " +
		"Create or start a feature first: htmlgraph feature create \"title\""
}

// hasAnyInProgressWorkItem returns true when any feature or bug with
// status="in-progress" exists in the features table. Used as a fallback when
// sessions.active_feature_id is empty (e.g. item started mid-session via CLI).
func hasAnyInProgressWorkItem(database *sql.DB) bool {
	var count int
	database.QueryRow(
		`SELECT COUNT(*) FROM features WHERE status = 'in-progress' LIMIT 1`,
	).Scan(&count)
	return count > 0
}

// featureStartPattern matches htmlgraph feature/bug start commands.
var featureStartPattern = regexp.MustCompile(`\bhtmlgraph\s+(feature|bug)\s+start\s+([\w-]+)`)

// checkYoloStepsGuard warns when starting a work item that has no
// implementation steps. Returns a non-empty reason to warn, or "" to allow.
func checkYoloStepsGuard(event *CloudEvent, yolo bool, htmlgraphDir string) string {
	if !yolo || event.ToolName != "Bash" {
		return ""
	}
	cmd, _ := event.ToolInput["command"].(string)
	m := featureStartPattern.FindStringSubmatch(cmd)
	if m == nil {
		return ""
	}
	itemID := m[2]
	stepsCount := countStepsForItem(htmlgraphDir, itemID)
	if stepsCount > 0 {
		return ""
	}
	return fmt.Sprintf(
		"Warning: %s has no implementation steps. "+
			"Add steps first: htmlgraph feature add-step %s \"description\"",
		itemID, itemID)
}

// countStepsForItem reads an HTML work item file and counts its steps.
func countStepsForItem(htmlgraphDir, itemID string) int {
	subdirs := []string{"features", "bugs", "spikes", "tracks", "plans", "specs"}
	for _, sub := range subdirs {
		path := filepath.Join(htmlgraphDir, sub, itemID+".html")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		return strings.Count(string(data), "data-step-id=")
	}
	return 0
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

// checkYoloBudgetGuard blocks git commit when the staged diff exceeds
// YOLO hard limits (20 files or 600 lines added).
func checkYoloBudgetGuard(event *CloudEvent, yolo bool) string {
	if !yolo || event.ToolName != "Bash" {
		return ""
	}
	cmd, _ := event.ToolInput["command"].(string)
	if !gitCommitPattern.MatchString(cmd) {
		return ""
	}
	out, err := exec.Command("git", "diff", "--cached", "--numstat").Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var fileCount, totalAdded int
	for _, line := range lines {
		if line == "" {
			continue
		}
		fileCount++
		parts := strings.Fields(line)
		if len(parts) >= 1 && parts[0] != "-" {
			n, _ := strconv.Atoi(parts[0])
			totalAdded += n
		}
	}
	if fileCount > 20 || totalAdded > 600 {
		return fmt.Sprintf(
			"YOLO budget HARD LIMIT: %d files, %d lines (max 20/600). "+
				"Split into sub-features.", fileCount, totalAdded)
	}
	return ""
}

// checkYoloWorktreeGuard blocks Write/Edit on main/master branch in YOLO mode.
func checkYoloWorktreeGuard(toolName, branch string, yolo bool) string {
	if !yolo {
		return ""
	}
	switch toolName {
	case "Write", "Edit", "MultiEdit":
	default:
		return ""
	}
	if branch == "main" || branch == "master" {
		return "YOLO mode requires a feature branch or worktree. " +
			"Create one: git worktree add -b feat-xxx .claude/worktrees/xxx main"
	}
	return ""
}

// checkYoloResearchGuard blocks Write/Edit when no Read/Grep/Glob has
// occurred in the session (research-first principle).
func checkYoloResearchGuard(toolName string, yolo, hasResearch bool) string {
	if !yolo {
		return ""
	}
	switch toolName {
	case "Write", "Edit", "MultiEdit":
	default:
		return ""
	}
	if hasResearch {
		return ""
	}
	return "YOLO mode requires research before writing code. " +
		"Read existing code first: use Read, Grep, or Glob tools."
}

// checkYoloDiffReviewGuard blocks git commit when no git diff has been
// reviewed in this session.
func checkYoloDiffReviewGuard(event *CloudEvent, yolo, diffRan bool) string {
	if !yolo || event.ToolName != "Bash" {
		return ""
	}
	cmd, _ := event.ToolInput["command"].(string)
	if !gitCommitPattern.MatchString(cmd) {
		return ""
	}
	if diffRan {
		return ""
	}
	return "YOLO mode requires a diff review before committing. " +
		"Run: git diff --stat"
}

// checkYoloCodeHealthGuard blocks writes that would create oversized
// Go/Python files (>500 lines) in YOLO mode.
func checkYoloCodeHealthGuard(event *CloudEvent, yolo bool) string {
	if !yolo {
		return ""
	}
	switch event.ToolName {
	case "Write", "Edit", "MultiEdit":
	default:
		return ""
	}
	path, _ := event.ToolInput["file_path"].(string)
	if path == "" {
		path, _ = event.ToolInput["path"].(string)
	}
	if !strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, ".py") {
		return ""
	}
	// Check existing file size — if it's already >500 lines, warn
	data, err := os.ReadFile(path)
	if err != nil {
		return "" // new file, allow
	}
	lines := strings.Count(string(data), "\n")
	if lines > 500 {
		return fmt.Sprintf(
			"YOLO code health: %s has %d lines (limit 500). "+
				"Refactor into smaller modules.", filepath.Base(path), lines)
	}
	return ""
}

// hasRecentResearch checks if Read/Grep/Glob was used in this session.
func hasRecentResearch(database *sql.DB, sessionID string) bool {
	var count int
	database.QueryRow(`
		SELECT COUNT(*) FROM agent_events
		WHERE session_id = ? AND tool_name IN ('Read', 'Grep', 'Glob', 'Agent')
		LIMIT 1`,
		sessionID,
	).Scan(&count)
	return count > 0
}

// getSessionAndParent returns the current session ID plus its parent session
// ID (if any). Worktree subagents inherit context from the outer orchestrator
// session that spawned them.
func getSessionAndParent(database *sql.DB, sessionID string) []string {
	sessionIDs := []string{sessionID}
	var parentID string
	database.QueryRow(
		`SELECT COALESCE(parent_session_id, '') FROM sessions WHERE session_id = ?`,
		sessionID,
	).Scan(&parentID)
	if parentID != "" {
		sessionIDs = append(sessionIDs, parentID)
	}
	return sessionIDs
}

// hasRecentDiffReview checks if git diff was run in this session or its
// parent session. Worktree subagents inherit diff reviews from the outer
// orchestrator session that spawned them.
func hasRecentDiffReview(database *sql.DB, sessionID string) bool {
	for _, sid := range getSessionAndParent(database, sessionID) {
		var count int
		database.QueryRow(`
			SELECT COUNT(*) FROM agent_events
			WHERE session_id = ? AND tool_name = 'Bash'
			  AND (input_summary LIKE '%git diff%'
			    OR input_summary LIKE '%git show%')`,
			sid,
		).Scan(&count)
		if count > 0 {
			return true
		}
	}
	return false
}

// currentBranchIn returns the git branch for the given directory.
func currentBranchIn(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// branchForFilePath returns the git branch for the worktree that owns filePath.
// When the file lives in a linked worktree (e.g. .claude/worktrees/yolo-feat-xxx),
// this returns that worktree's branch rather than the main repo's branch.
// Falls back to cwdBranch when filePath is empty or not under git control.
func branchForFilePath(filePath, cwdBranch string) string {
	if filePath == "" {
		return cwdBranch
	}
	dir := filepath.Dir(filePath)
	branch := currentBranchIn(dir)
	if branch == "" {
		return cwdBranch
	}
	return branch
}

// testPattern matches common test runner commands in Bash input summaries.
var testPattern = regexp.MustCompile(`\bgo test\b|\bpytest\b|\buv run pytest\b|\buv run ruff\b`)

// hasRecentTestRun checks if a test command was executed in this session
// or its parent session by scanning recent agent_events for Bash commands
// matching test patterns. Worktree subagents inherit test runs from the
// outer orchestrator session that spawned them.
func hasRecentTestRun(database *sql.DB, sessionID string) bool {
	for _, sid := range getSessionAndParent(database, sessionID) {
		var count int
		database.QueryRow(`
			SELECT COUNT(*) FROM agent_events
			WHERE session_id = ? AND tool_name = 'Bash'
			  AND (input_summary LIKE '%go test%'
			    OR input_summary LIKE '%go build%'
			    OR input_summary LIKE '%pytest%'
			    OR input_summary LIKE '%uv run ruff%')`,
			sid,
		).Scan(&count)
		if count > 0 {
			return true
		}
	}
	return false
}

// checkYoloUIValidationGuard blocks git commit when UI files were modified in
// the session but no screenshot or visual validation was performed.
// Returns a non-empty reason to block, or "" to allow.
func checkYoloUIValidationGuard(event *CloudEvent, yolo bool, database *sql.DB, sessionID string) string {
	if !yolo || event.ToolName != "Bash" {
		return ""
	}
	cmd, _ := event.ToolInput["command"].(string)
	if !gitCommitPattern.MatchString(cmd) {
		return ""
	}

	// Check if any UI files were modified in this session.
	var uiFileCount int
	database.QueryRow(`
		SELECT COUNT(*) FROM agent_events
		WHERE session_id = ? AND tool_name IN ('Write', 'Edit', 'MultiEdit')
		  AND (input_summary LIKE '%.html%' OR input_summary LIKE '%.css%'
		    OR input_summary LIKE '%.js%'  OR input_summary LIKE '%.ts%'
		    OR input_summary LIKE '%.tsx%' OR input_summary LIKE '%.vue%'
		    OR input_summary LIKE '%.svelte%')
		  AND status = 'completed'`,
		sessionID,
	).Scan(&uiFileCount)

	if uiFileCount == 0 {
		return "" // no UI files touched
	}

	// Check for screenshot / UI validation in session (+ parent).
	for _, sid := range getSessionAndParent(database, sessionID) {
		var validationCount int
		database.QueryRow(`
			SELECT COUNT(*) FROM agent_events
			WHERE session_id = ?
			  AND (tool_name LIKE '%screenshot%'
			    OR tool_name LIKE '%take_screenshot%'
			    OR input_summary LIKE '%ui-review%'
			    OR input_summary LIKE '%htmlgraph:ui-review%')`,
			sid,
		).Scan(&validationCount)
		if validationCount > 0 {
			return ""
		}
	}

	return "UI files were modified but no visual validation was performed. " +
		"Take a screenshot or run /htmlgraph:ui-review before committing."
}
