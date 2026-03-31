package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/shakestzd/htmlgraph/packages/go/internal/models"
)

// InsertEvent writes an agent event row.
func InsertEvent(db *sql.DB, e *models.AgentEvent) error {
	_, err := db.Exec(`
		INSERT INTO agent_events (
			event_id, agent_id, event_type, timestamp, tool_name,
			input_summary, tool_input, output_summary, session_id, feature_id,
			parent_agent_id, parent_event_id, subagent_type,
			cost_tokens, execution_duration_seconds, status,
			model, claude_task_id, source, step_id,
			created_at, updated_at
		) VALUES (?,?,?,?,?, ?,?,?,?,?, ?,?,?,?,?, ?,?,?,?,?, ?,?)`,
		e.EventID, e.AgentID, string(e.EventType),
		e.Timestamp.UTC().Format(time.RFC3339), nullStr(e.ToolName),
		nullStr(e.InputSummary), nullStr(e.ToolInput), nullStr(e.OutputSummary),
		e.SessionID, nullStr(e.FeatureID),
		nullStr(e.ParentAgentID), nullStr(e.ParentEventID),
		nullStr(e.SubagentType),
		e.CostTokens, e.ExecDuration, e.Status,
		nullStr(e.Model), nullStr(e.ClaudeTaskID),
		e.Source, nullStr(e.StepID),
		e.CreatedAt.UTC().Format(time.RFC3339),
		e.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert event %s: %w", e.EventID, err)
	}
	return nil
}

// GetEvent retrieves a single agent event by ID.
func GetEvent(db *sql.DB, eventID string) (*models.AgentEvent, error) {
	row := db.QueryRow(`
		SELECT event_id, agent_id, event_type, timestamp, tool_name,
			input_summary, tool_input, output_summary, session_id, feature_id,
			parent_agent_id, parent_event_id, subagent_type,
			cost_tokens, execution_duration_seconds, status,
			model, source, step_id, created_at, updated_at
		FROM agent_events WHERE event_id = ?`, eventID)

	e := &models.AgentEvent{}
	var (
		tsStr, createdStr, updatedStr                        string
		toolName, inSum, toolInput, outSum, featID           sql.NullString
		parentAgent, parentEvt, subType, model, src, stepID sql.NullString
	)

	err := row.Scan(
		&e.EventID, &e.AgentID, &e.EventType, &tsStr, &toolName,
		&inSum, &toolInput, &outSum, &e.SessionID, &featID,
		&parentAgent, &parentEvt, &subType,
		&e.CostTokens, &e.ExecDuration, &e.Status,
		&model, &src, &stepID, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, fmt.Errorf("get event %s: %w", eventID, err)
	}

	e.Timestamp, _ = time.Parse(time.RFC3339, tsStr)
	e.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	e.ToolName = toolName.String
	e.InputSummary = inSum.String
	e.ToolInput = toolInput.String
	e.OutputSummary = outSum.String
	e.FeatureID = featID.String
	e.ParentAgentID = parentAgent.String
	e.ParentEventID = parentEvt.String
	e.SubagentType = subType.String
	e.Model = model.String
	e.Source = src.String
	e.StepID = stepID.String

	return e, nil
}

// ListEventsBySession returns events for a session ordered by timestamp DESC.
func ListEventsBySession(db *sql.DB, sessionID string, limit int) ([]models.AgentEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.Query(`
		SELECT event_id, agent_id, event_type, timestamp, tool_name,
			session_id, feature_id, status, model
		FROM agent_events
		WHERE session_id = ?
		ORDER BY timestamp DESC
		LIMIT ?`, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("list events for session %s: %w", sessionID, err)
	}
	defer rows.Close()

	var events []models.AgentEvent
	for rows.Next() {
		var e models.AgentEvent
		var tsStr string
		var toolName, featID, model sql.NullString

		if err := rows.Scan(
			&e.EventID, &e.AgentID, &e.EventType, &tsStr, &toolName,
			&e.SessionID, &featID, &e.Status, &model,
		); err != nil {
			return nil, err
		}
		e.Timestamp, _ = time.Parse(time.RFC3339, tsStr)
		e.ToolName = toolName.String
		e.FeatureID = featID.String
		e.Model = model.String
		events = append(events, e)
	}
	return events, rows.Err()
}

// ListEventsBySessionAsc returns events for a session ordered by timestamp ASC
// including parent_event_id for hierarchy reconstruction.
func ListEventsBySessionAsc(db *sql.DB, sessionID string, limit int) ([]models.AgentEvent, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := db.Query(`
		SELECT event_id, agent_id, event_type, timestamp, tool_name,
			session_id, feature_id, parent_event_id, status, model
		FROM agent_events
		WHERE session_id = ?
		ORDER BY timestamp ASC
		LIMIT ?`, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("list events asc for session %s: %w", sessionID, err)
	}
	defer rows.Close()

	var events []models.AgentEvent
	for rows.Next() {
		var e models.AgentEvent
		var tsStr string
		var toolName, featID, parentEvt, model sql.NullString

		if err := rows.Scan(
			&e.EventID, &e.AgentID, &e.EventType, &tsStr, &toolName,
			&e.SessionID, &featID, &parentEvt, &e.Status, &model,
		); err != nil {
			return nil, err
		}
		e.Timestamp, _ = time.Parse(time.RFC3339, tsStr)
		e.ToolName = toolName.String
		e.FeatureID = featID.String
		e.ParentEventID = parentEvt.String
		e.Model = model.String
		events = append(events, e)
	}
	return events, rows.Err()
}

// MostRecentSession returns the session_id of the latest session (any status),
// or ("", nil) if the table is empty.
func MostRecentSession(db *sql.DB) (string, error) {
	row := db.QueryRow(`
		SELECT session_id FROM sessions
		ORDER BY created_at DESC LIMIT 1`)
	var id string
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("most recent session: %w", err)
	}
	return id, nil
}

// UpsertEvent performs an INSERT OR REPLACE for idempotent event writes.
// This is useful when a hook may fire multiple times for the same logical event.
func UpsertEvent(db *sql.DB, e *models.AgentEvent) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO agent_events (
			event_id, agent_id, event_type, timestamp, tool_name,
			input_summary, tool_input, output_summary, session_id, feature_id,
			parent_agent_id, parent_event_id, subagent_type,
			cost_tokens, execution_duration_seconds, status,
			model, claude_task_id, source, step_id,
			created_at, updated_at
		) VALUES (?,?,?,?,?, ?,?,?,?,?, ?,?,?,?,?, ?,?,?,?,?, ?,?)`,
		e.EventID, e.AgentID, string(e.EventType),
		e.Timestamp.UTC().Format(time.RFC3339), nullStr(e.ToolName),
		nullStr(e.InputSummary), nullStr(e.ToolInput), nullStr(e.OutputSummary),
		e.SessionID, nullStr(e.FeatureID),
		nullStr(e.ParentAgentID), nullStr(e.ParentEventID),
		nullStr(e.SubagentType),
		e.CostTokens, e.ExecDuration, e.Status,
		nullStr(e.Model), nullStr(e.ClaudeTaskID),
		e.Source, nullStr(e.StepID),
		e.CreatedAt.UTC().Format(time.RFC3339),
		e.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("upsert event %s: %w", e.EventID, err)
	}
	return nil
}

// UpdateEventFields performs a partial UPDATE on an event, setting status,
// output_summary, and updated_at.
func UpdateEventFields(db *sql.DB, eventID, status, outputSummary string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`
		UPDATE agent_events
		SET status = ?, output_summary = ?, updated_at = ?
		WHERE event_id = ?`,
		status, nullStr(outputSummary), now, eventID,
	)
	return err
}

// UpdateEventStatus sets only the status and updated_at on an event.
func UpdateEventStatus(db *sql.DB, eventID, status string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`
		UPDATE agent_events SET status = ?, updated_at = ?
		WHERE event_id = ?`,
		status, now, eventID,
	)
	return err
}

// FindStartedEvent returns the event_id of the most recent started event
// matching tool_name in the session. Returns ("", sql.ErrNoRows) when not found.
func FindStartedEvent(db *sql.DB, sessionID, toolName string) (string, error) {
	var eventID string
	err := db.QueryRow(`
		SELECT event_id FROM agent_events
		WHERE session_id = ? AND tool_name = ? AND status = 'started'
		ORDER BY timestamp DESC
		LIMIT 1`, sessionID, toolName,
	).Scan(&eventID)
	return eventID, err
}

// FindStartedEventByAgent returns the event_id of the most recent started event
// matching tool_name and agent_id. Returns ("", sql.ErrNoRows) when not found.
func FindStartedEventByAgent(db *sql.DB, sessionID, toolName, agentID string) (string, error) {
	var eventID string
	err := db.QueryRow(`
		SELECT event_id FROM agent_events
		WHERE session_id = ? AND tool_name = ? AND agent_id = ? AND status = 'started'
		ORDER BY timestamp DESC
		LIMIT 1`, sessionID, toolName, agentID,
	).Scan(&eventID)
	return eventID, err
}

// FindStartedDelegation returns the event_id of the most recent started
// task_delegation in the session. Returns ("", sql.ErrNoRows) when not found.
func FindStartedDelegation(db *sql.DB, sessionID string) (string, error) {
	var eventID string
	err := db.QueryRow(`
		SELECT event_id FROM agent_events
		WHERE session_id = ?
		  AND event_type IN ('task_delegation', 'delegation')
		  AND status = 'started'
		ORDER BY timestamp DESC
		LIMIT 1`, sessionID,
	).Scan(&eventID)
	return eventID, err
}

// FindDelegationByAgent returns the most recent delegation event for the agent
// (any status). Returns ("", sql.ErrNoRows) when not found.
func FindDelegationByAgent(db *sql.DB, sessionID, agentID string) (string, error) {
	var eventID string
	err := db.QueryRow(`
		SELECT event_id FROM agent_events
		WHERE session_id = ?
		  AND event_type IN ('task_delegation', 'delegation')
		  AND agent_id = ?
		ORDER BY timestamp DESC
		LIMIT 1`, sessionID, agentID,
	).Scan(&eventID)
	return eventID, err
}

// FindStartedDelegationByAgent returns the most recent started delegation for
// the agent. Returns ("", sql.ErrNoRows) when not found.
func FindStartedDelegationByAgent(db *sql.DB, sessionID, agentID string) (string, error) {
	var eventID string
	err := db.QueryRow(`
		SELECT event_id FROM agent_events
		WHERE session_id = ?
		  AND event_type IN ('task_delegation', 'delegation')
		  AND agent_id = ?
		  AND status = 'started'
		ORDER BY timestamp DESC
		LIMIT 1`, sessionID, agentID,
	).Scan(&eventID)
	return eventID, err
}

// LatestEventByTool returns the event_id of the most recent event for the given
// session and tool_name, regardless of status. Returns ("", sql.ErrNoRows) when not found.
func LatestEventByTool(db *sql.DB, sessionID, toolName string) (string, error) {
	var eventID string
	err := db.QueryRow(`
		SELECT event_id FROM agent_events
		WHERE session_id = ? AND tool_name = ?
		ORDER BY timestamp DESC
		LIMIT 1`, sessionID, toolName,
	).Scan(&eventID)
	return eventID, err
}

// CountRecentDuplicates returns the count of events matching tool_name and
// input_summary within the last windowSeconds. Used for dedup checks.
func CountRecentDuplicates(db *sql.DB, sessionID, toolName, inputSummary string, windowSeconds int) (int, error) {
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM agent_events
		 WHERE session_id = ? AND tool_name = ? AND input_summary = ?
		   AND timestamp > datetime('now', ? || ' seconds')`,
		sessionID, toolName, inputSummary, fmt.Sprintf("-%d", windowSeconds),
	).Scan(&count)
	return count, err
}
