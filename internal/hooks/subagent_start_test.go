package hooks

import (
	"testing"

	"github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/models"
)

// TestSubagentStart_WritesLineageRows is the bug-cb4918d8 regression test for
// the subagent-start write path. Given a CloudEvent with agent_id and
// agent_type populated (the empirically-verified discriminator fields from
// /tmp/htmlgraph-hook-trace.jsonl), the handler must:
//
//  1. Insert a synthetic sessions row keyed by agent_id with
//     parent_session_id = event.SessionID and is_subagent = 1.
//  2. Insert an agent_lineage_trace row with trace_id = agent_id,
//     root_session_id = event.SessionID, and agent_name = event.AgentType.
func TestSubagentStart_WritesLineageRows(t *testing.T) {
	database, projectDir := setupLifecycleDB(t)
	parentSessionID := "parent-session-cb4918d8"
	subagentID := "subagent-abc123"
	agentType := "htmlgraph:haiku-coder"

	// Isolate from the dev environment.
	t.Setenv("HTMLGRAPH_PROJECT_DIR", projectDir)
	t.Setenv("HTMLGRAPH_SESSION_ID", parentSessionID)
	t.Setenv("HTMLGRAPH_AGENT_ID", "claude-code")
	t.Setenv("HTMLGRAPH_PARENT_EVENT", "")

	// Seed the parent session so downstream queries don't trip FK issues.
	if err := db.InsertSession(database, &models.Session{
		SessionID:     parentSessionID,
		AgentAssigned: "claude-code",
		Status:        "active",
	}); err != nil {
		t.Fatalf("InsertSession parent: %v", err)
	}

	event := &CloudEvent{
		SessionID: parentSessionID,
		CWD:       projectDir,
		AgentID:   subagentID,
		AgentType: agentType,
	}
	if _, err := SubagentStart(event, database); err != nil {
		t.Fatalf("SubagentStart: %v", err)
	}

	// Assertion 1: synthetic sessions row keyed by agent_id.
	childSess, err := db.GetSession(database, subagentID)
	if err != nil || childSess == nil {
		t.Fatalf("GetSession subagent: sess=%v err=%v", childSess, err)
	}
	if childSess.ParentSessionID != parentSessionID {
		t.Errorf("parent_session_id: got %q, want %q", childSess.ParentSessionID, parentSessionID)
	}
	if !childSess.IsSubagent {
		t.Error("is_subagent: got false, want true")
	}

	// Assertion 2: lineage trace row.
	trace, err := db.GetLineageBySession(database, subagentID)
	if err != nil {
		t.Fatalf("GetLineageBySession: %v", err)
	}
	if trace == nil {
		t.Fatal("expected lineage trace, got nil")
	}
	if trace.TraceID != subagentID {
		t.Errorf("trace_id: got %q, want %q", trace.TraceID, subagentID)
	}
	if trace.RootSessionID != parentSessionID {
		t.Errorf("root_session_id: got %q, want %q", trace.RootSessionID, parentSessionID)
	}
	if trace.AgentName != agentType {
		t.Errorf("agent_name: got %q, want %q", trace.AgentName, agentType)
	}
	if trace.Status != "active" {
		t.Errorf("status: got %q, want %q", trace.Status, "active")
	}
}

// TestSubagentStart_Idempotent asserts re-delivery of the same start event
// is safe — the INSERT OR IGNORE path on sessions plus a duplicate-PK warn
// on agent_lineage_trace must not fail the hook.
func TestSubagentStart_Idempotent(t *testing.T) {
	database, projectDir := setupLifecycleDB(t)
	parentSessionID := "parent-idempotent"
	subagentID := "subagent-idempotent"

	t.Setenv("HTMLGRAPH_PROJECT_DIR", projectDir)
	t.Setenv("HTMLGRAPH_SESSION_ID", parentSessionID)

	if err := db.InsertSession(database, &models.Session{
		SessionID: parentSessionID, AgentAssigned: "claude-code", Status: "active",
	}); err != nil {
		t.Fatalf("InsertSession: %v", err)
	}

	event := &CloudEvent{
		SessionID: parentSessionID,
		CWD:       projectDir,
		AgentID:   subagentID,
		AgentType: "general-purpose",
	}
	if _, err := SubagentStart(event, database); err != nil {
		t.Fatalf("SubagentStart first: %v", err)
	}
	if _, err := SubagentStart(event, database); err != nil {
		t.Fatalf("SubagentStart re-delivery: %v", err)
	}
}

// TestSubagentStop_ClosesLineage asserts that SubagentStop updates the lineage
// row for the matching agent_id, setting status=completed and completed_at.
func TestSubagentStop_ClosesLineage(t *testing.T) {
	database, projectDir := setupLifecycleDB(t)
	parentSessionID := "parent-stop-test"
	subagentID := "subagent-stop-test"

	t.Setenv("HTMLGRAPH_PROJECT_DIR", projectDir)
	t.Setenv("HTMLGRAPH_SESSION_ID", parentSessionID)

	if err := db.InsertSession(database, &models.Session{
		SessionID: parentSessionID, AgentAssigned: "claude-code", Status: "active",
	}); err != nil {
		t.Fatalf("InsertSession: %v", err)
	}

	startEvent := &CloudEvent{
		SessionID: parentSessionID,
		CWD:       projectDir,
		AgentID:   subagentID,
		AgentType: "general-purpose",
	}
	if _, err := SubagentStart(startEvent, database); err != nil {
		t.Fatalf("SubagentStart: %v", err)
	}

	stopEvent := &CloudEvent{
		SessionID:            parentSessionID,
		CWD:                  projectDir,
		AgentID:              subagentID,
		LastAssistantMessage: "all done",
	}
	if _, err := SubagentStop(stopEvent, database); err != nil {
		t.Fatalf("SubagentStop: %v", err)
	}

	trace, err := db.GetLineageBySession(database, subagentID)
	if err != nil {
		t.Fatalf("GetLineageBySession: %v", err)
	}
	if trace == nil {
		t.Fatal("expected lineage row, got nil")
	}
	if trace.Status != "completed" {
		t.Errorf("status: got %q, want %q", trace.Status, "completed")
	}
	if trace.CompletedAt == nil {
		t.Error("completed_at: got nil, want non-nil")
	}
}

// TestSubagentStop_MissingTraceIsNonFatal asserts a stop event with no
// matching start row does not return an error (log-and-continue semantics).
func TestSubagentStop_MissingTraceIsNonFatal(t *testing.T) {
	database, projectDir := setupLifecycleDB(t)
	parentSessionID := "parent-no-trace"

	t.Setenv("HTMLGRAPH_PROJECT_DIR", projectDir)
	t.Setenv("HTMLGRAPH_SESSION_ID", parentSessionID)

	if err := db.InsertSession(database, &models.Session{
		SessionID: parentSessionID, AgentAssigned: "claude-code", Status: "active",
	}); err != nil {
		t.Fatalf("InsertSession: %v", err)
	}

	stopEvent := &CloudEvent{
		SessionID: parentSessionID,
		CWD:       projectDir,
		AgentID:   "subagent-never-started",
	}
	if _, err := SubagentStop(stopEvent, database); err != nil {
		t.Fatalf("SubagentStop (missing trace) should not error: %v", err)
	}
}
