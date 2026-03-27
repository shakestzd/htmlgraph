package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/shakestzd/htmlgraph/internal/models"
)

// InsertEvent writes an agent event row.
func InsertEvent(db *sql.DB, e *models.AgentEvent) error {
	_, err := db.Exec(`
		INSERT INTO agent_events (
			event_id, agent_id, event_type, timestamp, tool_name,
			input_summary, output_summary, session_id, feature_id,
			parent_agent_id, parent_event_id, subagent_type,
			cost_tokens, execution_duration_seconds, status,
			model, claude_task_id, source, step_id,
			created_at, updated_at
		) VALUES (?,?,?,?,?, ?,?,?,?,?, ?,?,?,?,?, ?,?,?,?, ?,?)`,
		e.EventID, e.AgentID, string(e.EventType),
		e.Timestamp.UTC().Format(time.RFC3339), nullStr(e.ToolName),
		nullStr(e.InputSummary), nullStr(e.OutputSummary),
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
			input_summary, output_summary, session_id, feature_id,
			parent_agent_id, parent_event_id, subagent_type,
			cost_tokens, execution_duration_seconds, status,
			model, source, step_id, created_at, updated_at
		FROM agent_events WHERE event_id = ?`, eventID)

	e := &models.AgentEvent{}
	var (
		tsStr, createdStr, updatedStr                       string
		toolName, inSum, outSum, featID                     sql.NullString
		parentAgent, parentEvt, subType, model, src, stepID sql.NullString
	)

	err := row.Scan(
		&e.EventID, &e.AgentID, &e.EventType, &tsStr, &toolName,
		&inSum, &outSum, &e.SessionID, &featID,
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
