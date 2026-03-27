package hooks

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/models"
)

// recordSimpleEvent is a shared helper for hook handlers that record a single
// agent_event and always return Continue. It resolves the session and feature
// IDs from the event/database, builds the AgentEvent, and inserts it
// non-fatally. Returns Continue on missing session ID.
func recordSimpleEvent(
	eventType models.EventType,
	toolName, inputSummary, status string,
	event *CloudEvent,
	database *sql.DB,
) (*HookResult, error) {
	sessionID := EnvSessionID(event.SessionID)
	if sessionID == "" {
		return &HookResult{Continue: true}, nil
	}

	featureID := GetActiveFeatureID(database, sessionID)
	now := time.Now().UTC()

	ev := &models.AgentEvent{
		EventID:      uuid.New().String(),
		AgentID:      agentIDFromEnv(),
		EventType:    eventType,
		Timestamp:    now,
		ToolName:     toolName,
		InputSummary: inputSummary,
		SessionID:    sessionID,
		FeatureID:    featureID,
		Status:       status,
		Source:       "hook",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_ = db.InsertEvent(database, ev) // Non-fatal

	return &HookResult{Continue: true}, nil
}

// Stop handles the Stop Claude Code hook event (agent/session stopped).
// Records a checkpoint event so stop events appear in the activity feed.
func Stop(event *CloudEvent, database *sql.DB) (*HookResult, error) {
	return recordSimpleEvent(models.EventEnd, "Stop", "Agent stopped", "recorded", event, database)
}

// PreCompact handles the PreCompact Claude Code hook event.
// Records a checkpoint before conversation context compaction.
func PreCompact(event *CloudEvent, database *sql.DB) (*HookResult, error) {
	return recordSimpleEvent(models.EventCheckPoint, "PreCompact", "Conversation compaction triggered", "recorded", event, database)
}

// TeammateIdle handles the TeammateIdle Claude Code hook event.
// Records a teammate_idle event when a teammate agent goes idle.
func TeammateIdle(event *CloudEvent, database *sql.DB) (*HookResult, error) {
	return recordSimpleEvent(models.EventTeammateIdle, "TeammateIdle", "Teammate agent went idle", "recorded", event, database)
}

// TaskCompleted handles the TaskCompleted Claude Code hook event.
// Records a task_completed event when a delegated task finishes.
func TaskCompleted(event *CloudEvent, database *sql.DB) (*HookResult, error) {
	return recordSimpleEvent(models.EventTaskCompleted, "TaskCompleted", "Delegated task completed", "completed", event, database)
}

// InstructionsLoaded handles the InstructionsLoaded Claude Code hook event.
// Records a checkpoint when CLAUDE.md or other instruction files are loaded.
func InstructionsLoaded(event *CloudEvent, database *sql.DB) (*HookResult, error) {
	return recordSimpleEvent(models.EventCheckPoint, "InstructionsLoaded", "Instruction files loaded (CLAUDE.md etc.)", "recorded", event, database)
}

// PermissionRequest handles the PermissionRequest Claude Code hook event.
// Records a checkpoint when Claude requests a permission prompt.
func PermissionRequest(event *CloudEvent, database *sql.DB) (*HookResult, error) {
	summary := "Permission requested"
	if event.ToolName != "" {
		summary = fmt.Sprintf("Permission requested for tool: %s", event.ToolName)
	}
	return recordSimpleEvent(models.EventCheckPoint, "PermissionRequest", summary, "recorded", event, database)
}

// PostToolUseFailure handles the PostToolUseFailure Claude Code hook event.
// Records a tool crash/exception as an error event with details from ToolResult.
// This handler is kept separate because it extracts error info from ToolResult.
func PostToolUseFailure(event *CloudEvent, database *sql.DB) (*HookResult, error) {
	sessionID := EnvSessionID(event.SessionID)
	if sessionID == "" {
		return &HookResult{Continue: true}, nil
	}

	featureID := GetActiveFeatureID(database, sessionID)
	errorSummary := summariseOutput(event.ToolResult)
	if errorSummary == "" {
		errorSummary = fmt.Sprintf("tool %q crashed or threw exception", event.ToolName)
	}

	now := time.Now().UTC()
	ev := &models.AgentEvent{
		EventID:      uuid.New().String(),
		AgentID:      agentIDFromEnv(),
		EventType:    models.EventError,
		Timestamp:    now,
		ToolName:     event.ToolName,
		InputSummary: fmt.Sprintf("PostToolUseFailure: %s", errorSummary),
		SessionID:    sessionID,
		FeatureID:    featureID,
		Status:       "failed",
		Source:       "hook",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_ = db.InsertEvent(database, ev) // Non-fatal

	return &HookResult{Continue: true}, nil
}
