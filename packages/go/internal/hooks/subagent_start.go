package hooks

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/models"
)

// SubagentStart handles the SubagentStart Claude Code hook event.
// It records a task_delegation agent_event and writes a traceparent entry
// to the temp queue so the spawned subagent can claim parent linkage.
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

	ev := &models.AgentEvent{
		EventID:      eventID,
		AgentID:      event.AgentID,
		EventType:    models.EventDelegation,
		Timestamp:    time.Now().UTC(),
		ToolName:     "Task",
		InputSummary: fmt.Sprintf("Subagent started: type=%s id=%s", agentType, event.AgentID),
		SessionID:    sessionID,
		FeatureID:    featureID,
		SubagentType: agentType,
		Status:       "started",
		Source:       "hook",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := db.InsertEvent(database, ev); err != nil {
		fmt.Fprintf(os.Stderr, "htmlgraph subagent-start: db error: %v\n", err)
	}

	// Write traceparent so the subagent's session-start can claim it.
	writeTraceparent(sessionID, eventID)

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
		  AND event_type = 'task_delegation'
		  AND status = 'started'
		ORDER BY timestamp DESC
		LIMIT 1`, sessionID,
	).Scan(&eventID)

	if err != nil {
		if err != sql.ErrNoRows {
			fmt.Fprintf(os.Stderr, "htmlgraph subagent-stop: query error: %v\n", err)
		}
		return &HookResult{Continue: true}, nil
	}

	_, err = database.Exec(`
		UPDATE agent_events
		SET status = 'completed', updated_at = ?
		WHERE event_id = ?`,
		now, eventID,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "htmlgraph subagent-stop: update error: %v\n", err)
	}

	return &HookResult{Continue: true}, nil
}
