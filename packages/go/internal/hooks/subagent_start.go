package hooks

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/models"
)

// SubagentStart handles the SubagentStart Claude Code hook event.
// It records a task_delegation agent_event, links it to the current UserQuery,
// and writes env vars so the subagent's hooks know their parent and identity.
func SubagentStart(event *CloudEvent, database *sql.DB) (*HookResult, error) {
	sessionID := EnvSessionID(event.SessionID)
	if sessionID == "" {
		return &HookResult{Continue: true}, nil
	}

	featureID := GetActiveFeatureID(database, sessionID)
	eventID := uuid.New().String()
	agentType := event.AgentType
	if agentType == "" {
		agentType = "general-purpose"
	}

	// Link delegation to the most recent UserQuery in this session
	var parentEventID string
	_ = database.QueryRow(
		`SELECT event_id FROM agent_events WHERE session_id = ? AND tool_name = 'UserQuery' ORDER BY timestamp DESC LIMIT 1`,
		sessionID,
	).Scan(&parentEventID)

	ev := &models.AgentEvent{
		EventID:       eventID,
		AgentID:       event.AgentID,
		EventType:     models.EventTaskDelegation,
		Timestamp:     time.Now().UTC(),
		ToolName:      "Task",
		InputSummary:  fmt.Sprintf("Subagent started: type=%s id=%s", agentType, event.AgentID),
		SessionID:     sessionID,
		FeatureID:     featureID,
		ParentEventID: parentEventID,
		SubagentType:  agentType,
		Status:        "started",
		Source:        "hook",
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	_ = db.InsertEvent(database, ev) // Non-fatal

	// Write traceparent so the subagent's session-start can claim it.
	writeTraceparent(sessionID, eventID)

	// Write env vars so subagent hooks know their parent and identity.
	writeSubagentEnvVars(eventID, agentType)

	return &HookResult{Continue: true}, nil
}

// SubagentStop handles the SubagentStop Claude Code hook event.
// It marks the most recent started task_delegation event as completed.
func SubagentStop(event *CloudEvent, database *sql.DB) (*HookResult, error) {
	sessionID := EnvSessionID(event.SessionID)
	if sessionID == "" {
		return &HookResult{Continue: true}, nil
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Find the most recent task_delegation with status='started' in this session.
	var eventID string
	err := database.QueryRow(`
		SELECT event_id FROM agent_events
		WHERE session_id = ?
		  AND event_type IN ('task_delegation', 'delegation')
		  AND status = 'started'
		ORDER BY timestamp DESC
		LIMIT 1`, sessionID,
	).Scan(&eventID)

	if err != nil {
		return &HookResult{Continue: true}, nil
	}

	_, _ = database.Exec(`
		UPDATE agent_events
		SET status = 'completed', updated_at = ?
		WHERE event_id = ?`,
		now, eventID,
	)

	return &HookResult{Continue: true}, nil
}
