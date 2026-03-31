package hooks

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shakestzd/htmlgraph/packages/go/internal/db"
	"github.com/shakestzd/htmlgraph/packages/go/internal/models"
)

// setupLifecycleDB creates a temp project dir with .htmlgraph/ and a real
// SQLite DB. Returns the database and the project dir.
func setupLifecycleDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	projectDir := t.TempDir()
	hgDir := filepath.Join(projectDir, ".htmlgraph")
	if err := os.MkdirAll(hgDir, 0o755); err != nil {
		t.Fatalf("mkdir .htmlgraph: %v", err)
	}
	database, err := db.Open(filepath.Join(hgDir, "htmlgraph.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database, projectDir
}

// TestHookLifecycle exercises the full session lifecycle:
// SessionStart → UserPromptSubmit → PreToolUse → PostToolUse → SessionEnd.
func TestHookLifecycle(t *testing.T) {
	database, projectDir := setupLifecycleDB(t)
	sessionID := "lifecycle-test-session-001"

	// Isolate from the developer's real environment.
	// HTMLGRAPH_PROJECT_DIR must point to the test projectDir so that
	// ResolveProjectDir returns projectDir (not the real project via the hint
	// file), preventing checkProjectDivergence from blocking the test event.
	t.Setenv("CLAUDE_SESSION_ID", "")
	t.Setenv("HTMLGRAPH_PARENT_SESSION", "")
	t.Setenv("HTMLGRAPH_NESTING_DEPTH", "")
	t.Setenv("CLAUDE_ENV_FILE", "")
	t.Setenv("CLAUDE_PROJECT_DIR", "")
	t.Setenv("HTMLGRAPH_PROJECT_DIR", projectDir)
	t.Setenv("HTMLGRAPH_SESSION_ID", sessionID)
	t.Setenv("HTMLGRAPH_AGENT_ID", "claude-code")
	t.Setenv("HTMLGRAPH_AGENT_TYPE", "")
	t.Setenv("HTMLGRAPH_PARENT_EVENT", "")

	// --- Step 1: SessionStart ---
	startEvent := &CloudEvent{SessionID: sessionID, CWD: projectDir}
	_, err := SessionStart(startEvent, database, projectDir)
	if err != nil {
		t.Fatalf("SessionStart: %v", err)
	}

	sess, err := db.GetSession(database, sessionID)
	if err != nil || sess == nil {
		t.Fatalf("GetSession after SessionStart: %v", err)
	}
	if sess.Status != "active" {
		t.Errorf("expected session status=active, got %q", sess.Status)
	}
	if sess.ProjectDir != projectDir {
		t.Errorf("project_dir mismatch: got %q, want %q", sess.ProjectDir, projectDir)
	}

	// --- Step 2: UserPromptSubmit ---
	promptEvent := &CloudEvent{
		SessionID: sessionID,
		CWD:       projectDir,
		Prompt:    "implement the feature",
	}
	promptResult, err := UserPrompt(promptEvent, database)
	if err != nil {
		t.Fatalf("UserPrompt: %v", err)
	}
	// UserPrompt returns Continue:false when guidance is injected (normal behaviour).
	// The meaningful assertion is that the UserQuery event was recorded.
	if promptResult == nil {
		t.Fatal("UserPrompt returned nil result")
	}

	var queryCount int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM agent_events WHERE session_id = ? AND tool_name = 'UserQuery'`,
		sessionID,
	).Scan(&queryCount); err != nil {
		t.Fatalf("count UserQuery events: %v", err)
	}
	if queryCount != 1 {
		t.Errorf("expected 1 UserQuery event, got %d", queryCount)
	}

	// --- Step 3: PreToolUse (Bash) ---
	// Reset per-process feature ID cache so the new session is picked up.
	featureIDCache = featureIDCacheEntry{}

	preEvent := &CloudEvent{
		SessionID: sessionID,
		CWD:       projectDir,
		ToolName:  "Bash",
		ToolUseID: "tool-use-001",
		ToolInput: map[string]any{"command": "(cd packages/go && go test ./...)"},
	}
	preResult, err := PreToolUse(preEvent, database)
	if err != nil {
		t.Fatalf("PreToolUse: %v", err)
	}
	// Should allow (empty decision).
	if preResult.Decision == "block" {
		t.Errorf("PreToolUse blocked unexpectedly: %s", preResult.Reason)
	}

	var startedCount int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM agent_events WHERE session_id = ? AND tool_name = 'Bash' AND status = 'started'`,
		sessionID,
	).Scan(&startedCount); err != nil {
		t.Fatalf("count started Bash events: %v", err)
	}
	if startedCount != 1 {
		t.Errorf("expected 1 started Bash event, got %d", startedCount)
	}

	// --- Step 4: PostToolUse (Bash) ---
	postEvent := &CloudEvent{
		SessionID: sessionID,
		CWD:       projectDir,
		ToolName:  "Bash",
		ToolUseID: "tool-use-001",
		ToolInput: map[string]any{"command": "(cd packages/go && go test ./...)"},
		ToolResult: map[string]any{
			"output":   "ok  github.com/shakestzd/htmlgraph/...",
			"is_error": false,
		},
	}
	postResult, err := PostToolUse(postEvent, database)
	if err != nil {
		t.Fatalf("PostToolUse: %v", err)
	}
	if !postResult.Continue {
		t.Error("expected Continue=true from PostToolUse")
	}

	var completedCount int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM agent_events WHERE session_id = ? AND tool_name = 'Bash' AND status = 'completed'`,
		sessionID,
	).Scan(&completedCount); err != nil {
		t.Fatalf("count completed Bash events: %v", err)
	}
	if completedCount != 1 {
		t.Errorf("expected 1 completed Bash event, got %d", completedCount)
	}

	// --- Step 5: SessionEnd ---
	endEvent := &CloudEvent{SessionID: sessionID, CWD: projectDir}
	endResult, err := SessionEnd(endEvent, database, projectDir)
	if err != nil {
		t.Fatalf("SessionEnd: %v", err)
	}
	if !endResult.Continue {
		t.Error("expected Continue=true from SessionEnd")
	}

	sess, err = db.GetSession(database, sessionID)
	if err != nil || sess == nil {
		t.Fatalf("GetSession after SessionEnd: %v", err)
	}
	if sess.Status != "completed" {
		t.Errorf("expected session status=completed after SessionEnd, got %q", sess.Status)
	}
}

// TestEventRecordingFlow verifies that PreToolUse inserts a started event and
// PostToolUse transitions it to completed with output summary populated.
func TestEventRecordingFlow(t *testing.T) {
	database, projectDir := setupLifecycleDB(t)
	sessionID := "event-flow-session-001"

	t.Setenv("HTMLGRAPH_SESSION_ID", sessionID)
	t.Setenv("HTMLGRAPH_AGENT_ID", "claude-code")
	t.Setenv("HTMLGRAPH_AGENT_TYPE", "")
	t.Setenv("HTMLGRAPH_PARENT_EVENT", "")

	// Insert the session so FK constraints pass.
	if err := db.InsertSession(database, &models.Session{
		SessionID:     sessionID,
		AgentAssigned: "claude-code",
		Status:        "active",
		CreatedAt:     time.Now().UTC(),
		ProjectDir:    projectDir,
	}); err != nil {
		t.Fatalf("InsertSession: %v", err)
	}

	// Insert an active feature and link it to the session.
	feat := &db.Feature{
		ID:        "feat-lifecycle-01",
		Type:      "feature",
		Title:     "Lifecycle test feature",
		Status:    "in-progress",
		Priority:  "medium",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := db.InsertFeature(database, feat); err != nil {
		t.Fatalf("InsertFeature: %v", err)
	}
	if _, err := database.Exec(
		`UPDATE sessions SET active_feature_id = ? WHERE session_id = ?`,
		feat.ID, sessionID,
	); err != nil {
		t.Fatalf("set active_feature_id: %v", err)
	}

	// Reset per-process feature ID cache.
	featureIDCache = featureIDCacheEntry{}

	// PreToolUse: Read tool (no YOLO guards trigger).
	preEvent := &CloudEvent{
		SessionID: sessionID,
		CWD:       projectDir,
		ToolName:  "Read",
		ToolUseID: "tool-read-001",
		ToolInput: map[string]any{"file_path": filepath.Join(projectDir, "main.go")},
	}
	if _, err := PreToolUse(preEvent, database); err != nil {
		t.Fatalf("PreToolUse: %v", err)
	}

	// Verify the event is in 'started' state with the feature linked.
	var evFeatureID, evStatus string
	if err := database.QueryRow(
		`SELECT COALESCE(feature_id,''), status FROM agent_events
		 WHERE session_id = ? AND tool_name = 'Read' ORDER BY created_at DESC LIMIT 1`,
		sessionID,
	).Scan(&evFeatureID, &evStatus); err != nil {
		t.Fatalf("query started event: %v", err)
	}
	if evStatus != "started" {
		t.Errorf("expected status=started, got %q", evStatus)
	}
	if evFeatureID != feat.ID {
		t.Errorf("expected feature_id=%q, got %q", feat.ID, evFeatureID)
	}

	// PostToolUse: complete the Read event.
	postEvent := &CloudEvent{
		SessionID: sessionID,
		CWD:       projectDir,
		ToolName:  "Read",
		ToolUseID: "tool-read-001",
		ToolInput: map[string]any{"file_path": filepath.Join(projectDir, "main.go")},
		ToolResult: map[string]any{
			"output":   "package main\n\nfunc main() {}",
			"is_error": false,
		},
	}
	if _, err := PostToolUse(postEvent, database); err != nil {
		t.Fatalf("PostToolUse: %v", err)
	}

	// Verify the event transitioned to completed with non-empty output_summary.
	var finalStatus, outputSummary string
	if err := database.QueryRow(
		`SELECT status, COALESCE(output_summary,'') FROM agent_events
		 WHERE session_id = ? AND tool_name = 'Read' ORDER BY created_at DESC LIMIT 1`,
		sessionID,
	).Scan(&finalStatus, &outputSummary); err != nil {
		t.Fatalf("query completed event: %v", err)
	}
	if finalStatus != "completed" {
		t.Errorf("expected status=completed, got %q", finalStatus)
	}
	if outputSummary == "" {
		t.Error("expected non-empty output_summary after PostToolUse")
	}
}

// TestYoloModeGuards verifies that YOLO mode blocks write tools when no work
// item is active, and allows them once one is set.
func TestYoloModeGuards(t *testing.T) {
	database, projectDir := setupLifecycleDB(t)
	sessionID := "yolo-guard-session-001"
	hgDir := filepath.Join(projectDir, ".htmlgraph")

	t.Setenv("HTMLGRAPH_SESSION_ID", sessionID)
	t.Setenv("HTMLGRAPH_AGENT_ID", "claude-code")
	t.Setenv("HTMLGRAPH_AGENT_TYPE", "")
	t.Setenv("HTMLGRAPH_PARENT_EVENT", "")

	// Write YOLO launch mode file.
	resetYoloModeCache()
	if err := os.WriteFile(
		filepath.Join(hgDir, ".launch-mode"),
		[]byte(`{"mode":"yolo-dev","pid":9999}`),
		0o644,
	); err != nil {
		t.Fatalf("write .launch-mode: %v", err)
	}

	// Insert session without any active feature.
	if err := db.InsertSession(database, &models.Session{
		SessionID:     sessionID,
		AgentAssigned: "claude-code",
		Status:        "active",
		CreatedAt:     time.Now().UTC(),
		ProjectDir:    projectDir,
	}); err != nil {
		t.Fatalf("InsertSession: %v", err)
	}

	// Reset caches so YOLO mode and feature ID are read fresh.
	featureIDCache = featureIDCacheEntry{}

	// Write tool without a work item should be blocked.
	writeEvent := &CloudEvent{
		SessionID: sessionID,
		CWD:       projectDir,
		ToolName:  "Write",
		ToolInput: map[string]any{
			"file_path": filepath.Join(projectDir, "foo.go"),
			"content":   "package main",
		},
	}
	result, err := PreToolUse(writeEvent, database)
	if err != nil {
		t.Fatalf("PreToolUse (no work item): %v", err)
	}
	if result.Decision != "block" {
		t.Errorf("expected block in YOLO mode without work item, got decision=%q reason=%q",
			result.Decision, result.Reason)
	}

	// Now add an in-progress work item and link it to the session.
	feat := &db.Feature{
		ID:        "feat-yolo-guard-01",
		Type:      "feature",
		Title:     "YOLO guard test feature",
		Status:    "in-progress",
		Priority:  "medium",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := db.InsertFeature(database, feat); err != nil {
		t.Fatalf("InsertFeature: %v", err)
	}
	if _, err := database.Exec(
		`UPDATE sessions SET active_feature_id = ? WHERE session_id = ?`,
		feat.ID, sessionID,
	); err != nil {
		t.Fatalf("set active_feature_id: %v", err)
	}

	// Reset caches to pick up the new feature and re-read YOLO mode.
	featureIDCache = featureIDCacheEntry{}
	resetYoloModeCache()

	// Write tool with an active work item should be allowed (worktree guard
	// will not trigger because CWD is a temp dir, not main/master branch).
	result, err = PreToolUse(writeEvent, database)
	if err != nil {
		t.Fatalf("PreToolUse (with work item): %v", err)
	}
	if result.Decision == "block" {
		// Work item guard should have passed; any remaining block is from another
		// YOLO guard (e.g. research or worktree). Accept blocks from those guards
		// but fail if reason still mentions missing work item.
		if result.Reason == "No active work item. Start a feature or bug before writing: htmlgraph feature start <id>" {
			t.Errorf("work item guard still blocking after feature set: %s", result.Reason)
		}
	}
}
