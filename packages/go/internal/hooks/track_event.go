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

// TrackEvent handles generic Claude Code hook events that should be recorded
// as agent_events without blocking (e.g. InstructionsLoaded, PreCompact).
func TrackEvent(toolName string, event *CloudEvent, database *sql.DB) (*HookResult, error) {
	sessionID := EnvSessionID(event.SessionID)
	if sessionID == "" {
		return &HookResult{Continue: true}, nil
	}

	featureID := GetActiveFeatureID(database, sessionID)

	ev := &models.AgentEvent{
		EventID:      uuid.New().String(),
		AgentID:      agentIDFromEnv(),
		EventType:    models.EventCheckPoint,
		Timestamp:    time.Now().UTC(),
		ToolName:     toolName,
		InputSummary: fmt.Sprintf("%s event recorded", toolName),
		SessionID:    sessionID,
		FeatureID:    featureID,
		Status:       "recorded",
		Source:       "hook",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := db.InsertEvent(database, ev); err != nil {
		fmt.Fprintf(os.Stderr, "htmlgraph track-event %s: db error: %v\n", toolName, err)
	}

	return &HookResult{Continue: true}, nil
}

// UpdateEventStatus updates the status field of an agent_event by ID.
func UpdateEventStatus(database *sql.DB, eventID, status string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := database.Exec(`
		UPDATE agent_events SET status = ?, updated_at = ? WHERE event_id = ?`,
		status, now, eventID,
	)
	return err
}

// ensure db is used
var _ = db.InsertEvent

// ensure sql.DB and models are reachable
var _ *models.AgentEvent
