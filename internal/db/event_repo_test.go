package db_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/models"
)

// setupTestDB opens an in-memory database with schema and a test session.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := db.Open("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// Insert a session so FK constraints pass.
	now := time.Now().UTC()
	sess := &models.Session{
		SessionID:     "sess-test",
		AgentAssigned: "claude-code",
		CreatedAt:     now,
		Status:        "active",
	}
	if err := db.InsertSession(database, sess); err != nil {
		t.Fatalf("InsertSession: %v", err)
	}
	return database
}

func TestUpsertEvent(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	now := time.Now().UTC()
	ev := &models.AgentEvent{
		EventID:   "evt-upsert-1",
		AgentID:   "claude-code",
		EventType: models.EventToolCall,
		Timestamp: now,
		ToolName:  "Bash",
		SessionID: "sess-test",
		Status:    "started",
		Source:    "hook",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// First insert.
	if err := db.UpsertEvent(database, ev); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Verify it exists.
	got, err := db.GetEvent(database, "evt-upsert-1")
	if err != nil {
		t.Fatalf("GetEvent after first upsert: %v", err)
	}
	if got.Status != "started" {
		t.Errorf("status: got %q, want %q", got.Status, "started")
	}

	// Upsert with updated status (should replace).
	ev.Status = "completed"
	ev.OutputSummary = "done"
	if err := db.UpsertEvent(database, ev); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	got, err = db.GetEvent(database, "evt-upsert-1")
	if err != nil {
		t.Fatalf("GetEvent after second upsert: %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("status after upsert: got %q, want %q", got.Status, "completed")
	}
	if got.OutputSummary != "done" {
		t.Errorf("output_summary: got %q, want %q", got.OutputSummary, "done")
	}
}

func TestUpdateEventFields(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	now := time.Now().UTC()
	ev := &models.AgentEvent{
		EventID:   "evt-update-1",
		AgentID:   "claude-code",
		EventType: models.EventToolCall,
		Timestamp: now,
		ToolName:  "Read",
		SessionID: "sess-test",
		Status:    "started",
		Source:    "hook",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.InsertEvent(database, ev); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	if err := db.UpdateEventFields(database, "evt-update-1", "completed", "read main.go"); err != nil {
		t.Fatalf("UpdateEventFields: %v", err)
	}

	got, err := db.GetEvent(database, "evt-update-1")
	if err != nil {
		t.Fatalf("GetEvent: %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("status: got %q, want %q", got.Status, "completed")
	}
	if got.OutputSummary != "read main.go" {
		t.Errorf("output_summary: got %q, want %q", got.OutputSummary, "read main.go")
	}
}

func TestUpdateEventStatus(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	now := time.Now().UTC()
	ev := &models.AgentEvent{
		EventID:   "evt-status-1",
		AgentID:   "claude-code",
		EventType: models.EventToolCall,
		Timestamp: now,
		ToolName:  "Grep",
		SessionID: "sess-test",
		Status:    "started",
		Source:    "hook",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.InsertEvent(database, ev); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	if err := db.UpdateEventStatus(database, "evt-status-1", "failed"); err != nil {
		t.Fatalf("UpdateEventStatus: %v", err)
	}

	got, err := db.GetEvent(database, "evt-status-1")
	if err != nil {
		t.Fatalf("GetEvent: %v", err)
	}
	if got.Status != "failed" {
		t.Errorf("status: got %q, want %q", got.Status, "failed")
	}
}

func TestFindStartedEvent(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	now := time.Now().UTC()

	// Insert a started Bash event.
	ev1 := &models.AgentEvent{
		EventID:   "evt-find-1",
		AgentID:   "claude-code",
		EventType: models.EventToolCall,
		Timestamp: now,
		ToolName:  "Bash",
		SessionID: "sess-test",
		Status:    "started",
		Source:    "hook",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.InsertEvent(database, ev1); err != nil {
		t.Fatalf("InsertEvent ev1: %v", err)
	}

	// Insert a completed Bash event (should not be found).
	ev2 := &models.AgentEvent{
		EventID:   "evt-find-2",
		AgentID:   "claude-code",
		EventType: models.EventToolCall,
		Timestamp: now.Add(time.Second),
		ToolName:  "Bash",
		SessionID: "sess-test",
		Status:    "completed",
		Source:    "hook",
		CreatedAt: now.Add(time.Second),
		UpdatedAt: now.Add(time.Second),
	}
	if err := db.InsertEvent(database, ev2); err != nil {
		t.Fatalf("InsertEvent ev2: %v", err)
	}

	id, err := db.FindStartedEvent(database, "sess-test", "Bash")
	if err != nil {
		t.Fatalf("FindStartedEvent: %v", err)
	}
	if id != "evt-find-1" {
		t.Errorf("got %q, want %q", id, "evt-find-1")
	}

	// No started Read events -> ErrNoRows.
	_, err = db.FindStartedEvent(database, "sess-test", "Read")
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows for Read, got %v", err)
	}
}

func TestFindStartedEventByAgent(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	now := time.Now().UTC()

	// A started Bash event from agent-aaa.
	ev1 := &models.AgentEvent{
		EventID:   "evt-fseba-1",
		AgentID:   "agent-aaa",
		EventType: models.EventToolCall,
		Timestamp: now,
		ToolName:  "Bash",
		SessionID: "sess-test",
		Status:    "started",
		Source:    "hook",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.InsertEvent(database, ev1); err != nil {
		t.Fatalf("InsertEvent ev1: %v", err)
	}

	// A started Bash event from agent-bbb (different agent).
	ev2 := &models.AgentEvent{
		EventID:   "evt-fseba-2",
		AgentID:   "agent-bbb",
		EventType: models.EventToolCall,
		Timestamp: now.Add(time.Second),
		ToolName:  "Bash",
		SessionID: "sess-test",
		Status:    "started",
		Source:    "hook",
		CreatedAt: now.Add(time.Second),
		UpdatedAt: now.Add(time.Second),
	}
	if err := db.InsertEvent(database, ev2); err != nil {
		t.Fatalf("InsertEvent ev2: %v", err)
	}

	// A completed Bash event from agent-aaa (should not match).
	ev3 := &models.AgentEvent{
		EventID:   "evt-fseba-3",
		AgentID:   "agent-aaa",
		EventType: models.EventToolCall,
		Timestamp: now.Add(2 * time.Second),
		ToolName:  "Bash",
		SessionID: "sess-test",
		Status:    "completed",
		Source:    "hook",
		CreatedAt: now.Add(2 * time.Second),
		UpdatedAt: now.Add(2 * time.Second),
	}
	if err := db.InsertEvent(database, ev3); err != nil {
		t.Fatalf("InsertEvent ev3: %v", err)
	}

	// Should find agent-aaa's started event.
	id, err := db.FindStartedEventByAgent(database, "sess-test", "Bash", "agent-aaa")
	if err != nil {
		t.Fatalf("FindStartedEventByAgent agent-aaa: %v", err)
	}
	if id != "evt-fseba-1" {
		t.Errorf("agent-aaa: got %q, want %q", id, "evt-fseba-1")
	}

	// Should find agent-bbb's started event independently.
	id, err = db.FindStartedEventByAgent(database, "sess-test", "Bash", "agent-bbb")
	if err != nil {
		t.Fatalf("FindStartedEventByAgent agent-bbb: %v", err)
	}
	if id != "evt-fseba-2" {
		t.Errorf("agent-bbb: got %q, want %q", id, "evt-fseba-2")
	}

	// Unknown agent -> ErrNoRows.
	_, err = db.FindStartedEventByAgent(database, "sess-test", "Bash", "agent-unknown")
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows for unknown agent, got %v", err)
	}

	// No started Read events for agent-aaa -> ErrNoRows.
	_, err = db.FindStartedEventByAgent(database, "sess-test", "Read", "agent-aaa")
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows for Read/agent-aaa, got %v", err)
	}
}

func TestFindStartedDelegation(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	now := time.Now().UTC()
	ev := &models.AgentEvent{
		EventID:   "evt-deleg-1",
		AgentID:   "subagent-abc",
		EventType: models.EventTaskDelegation,
		Timestamp: now,
		ToolName:  "Task",
		SessionID: "sess-test",
		Status:    "started",
		Source:    "hook",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.InsertEvent(database, ev); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	id, err := db.FindStartedDelegation(database, "sess-test")
	if err != nil {
		t.Fatalf("FindStartedDelegation: %v", err)
	}
	if id != "evt-deleg-1" {
		t.Errorf("got %q, want %q", id, "evt-deleg-1")
	}
}

func TestFindStartedDelegationByAgent(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	now := time.Now().UTC()

	// A started delegation for agent-abc.
	ev1 := &models.AgentEvent{
		EventID:   "evt-sdba-1",
		AgentID:   "agent-abc",
		EventType: models.EventTaskDelegation,
		Timestamp: now,
		ToolName:  "Task",
		SessionID: "sess-test",
		Status:    "started",
		Source:    "hook",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.InsertEvent(database, ev1); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	// A completed delegation for agent-abc (should not match).
	ev2 := &models.AgentEvent{
		EventID:   "evt-sdba-2",
		AgentID:   "agent-abc",
		EventType: models.EventTaskDelegation,
		Timestamp: now.Add(time.Second),
		ToolName:  "Task",
		SessionID: "sess-test",
		Status:    "completed",
		Source:    "hook",
		CreatedAt: now.Add(time.Second),
		UpdatedAt: now.Add(time.Second),
	}
	if err := db.InsertEvent(database, ev2); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	id, err := db.FindStartedDelegationByAgent(database, "sess-test", "agent-abc")
	if err != nil {
		t.Fatalf("FindStartedDelegationByAgent: %v", err)
	}
	if id != "evt-sdba-1" {
		t.Errorf("got %q, want %q", id, "evt-sdba-1")
	}

	// Different agent -> ErrNoRows.
	_, err = db.FindStartedDelegationByAgent(database, "sess-test", "other-agent")
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows for other-agent, got %v", err)
	}
}

func TestFindDelegationByAgent(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	now := time.Now().UTC()
	ev := &models.AgentEvent{
		EventID:   "evt-deleg-agent-1",
		AgentID:   "agent-xyz",
		EventType: models.EventTaskDelegation,
		Timestamp: now,
		ToolName:  "Task",
		SessionID: "sess-test",
		Status:    "started",
		Source:    "hook",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.InsertEvent(database, ev); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	id, err := db.FindDelegationByAgent(database, "sess-test", "agent-xyz")
	if err != nil {
		t.Fatalf("FindDelegationByAgent: %v", err)
	}
	if id != "evt-deleg-agent-1" {
		t.Errorf("got %q, want %q", id, "evt-deleg-agent-1")
	}

	// Wrong agent -> ErrNoRows.
	_, err = db.FindDelegationByAgent(database, "sess-test", "other-agent")
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows for other-agent, got %v", err)
	}
}

func TestLatestEventByTool(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	now := time.Now().UTC()

	// Insert two UserQuery events.
	for i, id := range []string{"evt-uq-1", "evt-uq-2"} {
		ev := &models.AgentEvent{
			EventID:      id,
			AgentID:      "claude-code",
			EventType:    models.EventToolCall,
			Timestamp:    now.Add(time.Duration(i) * time.Second),
			ToolName:     "UserQuery",
			InputSummary: "prompt " + id,
			SessionID:    "sess-test",
			Status:       "recorded",
			Source:       "hook",
			CreatedAt:    now.Add(time.Duration(i) * time.Second),
			UpdatedAt:    now.Add(time.Duration(i) * time.Second),
		}
		if err := db.InsertEvent(database, ev); err != nil {
			t.Fatalf("InsertEvent %s: %v", id, err)
		}
	}

	// Should return the latest one.
	id, err := db.LatestEventByTool(database, "sess-test", "UserQuery")
	if err != nil {
		t.Fatalf("LatestEventByTool: %v", err)
	}
	if id != "evt-uq-2" {
		t.Errorf("got %q, want %q", id, "evt-uq-2")
	}

	// No Bash events -> ErrNoRows.
	_, err = db.LatestEventByTool(database, "sess-test", "Bash")
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows for Bash, got %v", err)
	}
}

func TestCountRecentDuplicates(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	now := time.Now().UTC()
	ev := &models.AgentEvent{
		EventID:      "evt-dup-1",
		AgentID:      "claude-code",
		EventType:    models.EventToolCall,
		Timestamp:    now,
		ToolName:     "UserQuery",
		InputSummary: "hello world",
		SessionID:    "sess-test",
		Status:       "recorded",
		Source:       "hook",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.InsertEvent(database, ev); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	// Count within 5 seconds should find 1.
	count, err := db.CountRecentDuplicates(database, "sess-test", "UserQuery", "hello world", 5)
	if err != nil {
		t.Fatalf("CountRecentDuplicates: %v", err)
	}
	if count != 1 {
		t.Errorf("count: got %d, want 1", count)
	}

	// Different summary -> 0.
	count, err = db.CountRecentDuplicates(database, "sess-test", "UserQuery", "different", 5)
	if err != nil {
		t.Fatalf("CountRecentDuplicates (different): %v", err)
	}
	if count != 0 {
		t.Errorf("count for different: got %d, want 0", count)
	}

	// Different session -> 0.
	count, err = db.CountRecentDuplicates(database, "other-session", "UserQuery", "hello world", 5)
	if err != nil {
		t.Fatalf("CountRecentDuplicates (other session): %v", err)
	}
	if count != 0 {
		t.Errorf("count for other session: got %d, want 0", count)
	}
}
