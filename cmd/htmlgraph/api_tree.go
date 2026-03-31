package main

import (
	"database/sql"
	"net/http"
	"strconv"
)

// turnStats holds per-turn aggregate counts.
type turnStats struct {
	ToolCount  int      `json:"tool_count"`
	ErrorCount int      `json:"error_count"`
	Models     []string `json:"models"`
}

// turn groups a UserQuery with its child events and stats.
type turn struct {
	SessionID string           `json:"session_id"`
	UserQuery map[string]any   `json:"user_query"`
	Children  []map[string]any `json:"children"`
	Stats     turnStats        `json:"stats"`
}

// eventColumns is the shared SELECT column list for agent_events.
const eventColumns = `event_id, agent_id, event_type, timestamp, tool_name,
	COALESCE(input_summary, ''), COALESCE(output_summary, ''),
	session_id, COALESCE(feature_id, ''), status,
	COALESCE(parent_event_id, ''), COALESCE(subagent_type, ''),
	COALESCE(model, ''), COALESCE(step_id, '')`

// treeHandler returns hierarchical event data grouped by UserQuery turns.
// GET /api/events/tree?limit=50
func treeHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
				limit = n
			}
		}

		turns := buildEventTree(database, limit)
		respondJSON(w, turns)
	}
}

// buildEventTree fetches UserQuery anchors and recursively builds
// the parent-child tree for each turn.
func buildEventTree(database *sql.DB, limit int) []turn {
	rows, err := database.Query(`
		SELECT `+eventColumns+`
		FROM agent_events
		WHERE tool_name = 'UserQuery'
		ORDER BY timestamp DESC
		LIMIT ?`, limit)
	if err != nil {
		return []turn{}
	}
	defer rows.Close()

	var turns []turn
	for rows.Next() {
		evt := scanEvent(rows)
		if evt == nil {
			continue
		}

		sessionID, _ := evt["session_id"].(string)
		eventID, _ := evt["event_id"].(string)

		children := fetchChildren(database, eventID, sessionID, 1)
		stats := computeStats(children)

		turns = append(turns, turn{
			SessionID: sessionID,
			UserQuery: evt,
			Children:  children,
			Stats:     stats,
		})
	}

	if turns == nil {
		return []turn{}
	}
	return turns
}

// fetchChildren recursively fetches child events up to maxDepth=4 (depth 0-3).
func fetchChildren(database *sql.DB, parentID, sessionID string, depth int) []map[string]any {
	if depth > 3 {
		return nil
	}

	rows, err := database.Query(`
		SELECT `+eventColumns+`
		FROM agent_events
		WHERE parent_event_id = ?
		ORDER BY timestamp DESC`, parentID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var children []map[string]any
	for rows.Next() {
		evt := scanEvent(rows)
		if evt == nil {
			continue
		}

		eventID, _ := evt["event_id"].(string)

		// Recurse for direct children.
		evt["children"] = fetchChildren(database, eventID, sessionID, depth+1)

		children = append(children, evt)
	}

	// Suppress duplicate tool_call/Agent rows when a sibling task_delegation exists.
	hasDelegation := false
	for _, c := range children {
		if et, _ := c["event_type"].(string); et == "task_delegation" {
			hasDelegation = true
			break
		}
	}
	if hasDelegation {
		filtered := children[:0]
		for _, c := range children {
			et, _ := c["event_type"].(string)
			tn, _ := c["tool_name"].(string)
			if et == "tool_call" && tn == "Agent" {
				continue // suppress — task_delegation is the canonical row
			}
			filtered = append(filtered, c)
		}
		children = filtered
	}

	return children
}

// scanEvent reads one row from the standard eventColumns projection.
func scanEvent(rows *sql.Rows) map[string]any {
	var eventID, agentID, eventType, ts, toolName string
	var inputSum, outputSum, sessionID, featureID, status string
	var parentEvtID, subagentType, model, stepID string

	if err := rows.Scan(
		&eventID, &agentID, &eventType, &ts, &toolName,
		&inputSum, &outputSum, &sessionID, &featureID, &status,
		&parentEvtID, &subagentType, &model, &stepID,
	); err != nil {
		return nil
	}

	return map[string]any{
		"event_id":        eventID,
		"agent_id":        agentID,
		"event_type":      eventType,
		"timestamp":       ts,
		"tool_name":       toolName,
		"input_summary":   inputSum,
		"output_summary":  outputSum,
		"session_id":      sessionID,
		"feature_id":      featureID,
		"status":          status,
		"parent_event_id": parentEvtID,
		"subagent_type":   subagentType,
		"tool_use_id":     stepID,
		"model":           model,
	}
}

// computeStats aggregates tool_count, error_count, and distinct models
// from a flat walk of the children tree.
func computeStats(children []map[string]any) turnStats {
	var stats turnStats
	modelSet := make(map[string]bool)
	walkChildren(children, &stats, modelSet)
	for m := range modelSet {
		if m != "" {
			stats.Models = append(stats.Models, m)
		}
	}
	if stats.Models == nil {
		stats.Models = []string{}
	}
	return stats
}

func walkChildren(children []map[string]any, stats *turnStats, models map[string]bool) {
	for _, evt := range children {
		stats.ToolCount++
		evtType, _ := evt["event_type"].(string)
		status, _ := evt["status"].(string)
		if evtType == "error" || status == "failed" {
			stats.ErrorCount++
		}
		if m, ok := evt["model"].(string); ok && m != "" {
			models[m] = true
		}
		if sub, ok := evt["children"].([]map[string]any); ok {
			walkChildren(sub, stats, models)
		}
	}
}

