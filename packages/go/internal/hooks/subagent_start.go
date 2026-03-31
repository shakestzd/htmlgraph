package hooks

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shakestzd/htmlgraph/packages/go/internal/db"
	"github.com/shakestzd/htmlgraph/packages/go/internal/models"
)

// SubagentStart handles the SubagentStart Claude Code hook event.
// It records a task_delegation agent_event, links it to the current UserQuery,
// and writes env vars so the subagent's hooks know their parent and identity.
func SubagentStart(event *CloudEvent, database *sql.DB) (*HookResult, error) {
	sessionID := EnvSessionID(event.SessionID)
	if sessionID == "" {
		return &HookResult{Continue: true}, nil
	}

	projectDir := ResolveProjectDir(event.CWD)
	featureID := cachedGetActiveFeatureID(database, sessionID)
	eventID := uuid.New().String()
	agentType := event.AgentType
	if agentType == "" {
		agentType = "general-purpose"
	}

	// Link delegation to the most recent UserQuery in this session.
	parentEventID, _ := db.LatestEventByTool(database, sessionID, "UserQuery")

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

	if err := db.InsertEvent(database, ev); err != nil {
		debugLog(projectDir, "[error] handler=subagent-start session=%s: insert event: %v", sessionID[:minSessionLen(sessionID)], err)
	}

	// Write traceparent so the subagent's session-start can claim it.
	writeTraceparent(sessionID, eventID)

	// Write env vars so subagent hooks know their parent and identity.
	writeSubagentEnvVars(eventID, event.AgentID, agentType, projectDir)

	return &HookResult{Continue: true}, nil
}

// SubagentStop handles the SubagentStop Claude Code hook event.
// It marks the task_delegation for this specific agent as completed and
// stores the last assistant message as the output summary.
func SubagentStop(event *CloudEvent, database *sql.DB) (*HookResult, error) {
	sessionID := EnvSessionID(event.SessionID)
	if sessionID == "" {
		return &HookResult{Continue: true}, nil
	}

	outputSummary := event.LastAssistantMessage
	if len(outputSummary) > outputSummaryMaxLen {
		outputSummary = outputSummary[:outputSummaryMaxLen] + "…"
	}

	// Prefer agent_id-scoped lookup to avoid matching the wrong delegation
	// in concurrent multi-agent scenarios.
	var eventID string
	if event.AgentID != "" {
		eventID, _ = db.FindStartedDelegationByAgent(database, sessionID, event.AgentID)
	}

	// Fallback: most recent started delegation in this session.
	if eventID == "" {
		var err error
		eventID, err = db.FindStartedDelegation(database, sessionID)
		if err != nil {
			return &HookResult{Continue: true}, nil
		}
	}

	if err := db.UpdateEventFields(database, eventID, "completed", outputSummary); err != nil {
		projectDir := ResolveProjectDir(event.CWD)
		debugLog(projectDir, "[error] handler=subagent-stop session=%s: update event fields: %v", sessionID[:minSessionLen(sessionID)], err)
	}

	return &HookResult{Continue: true}, nil
}
