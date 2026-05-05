package main

import (
	"database/sql"
	"net/http"
	"strconv"
)

// timelineHandler returns chronological session activity as a JSON array.
// Supports ?session=SESSION_ID and ?limit=N (default 50, max 200).
// Each entry has: { type, timestamp, summary, feature_id, details }.
func timelineHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.URL.Query().Get("session")

		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}

		entries := make([]map[string]any, 0, limit)

		var rows *sql.Rows
		var err error
		if sessionID != "" {
			rows, err = database.Query(`
				SELECT event_type, timestamp,
				       COALESCE(tool_name, ''),
				       COALESCE(input_summary, ''),
				       COALESCE(output_summary, ''),
				       COALESCE(feature_id, ''),
				       COALESCE(agent_id, ''),
				       COALESCE(status, ''),
				       COALESCE(subagent_type, '')
				FROM agent_events
				WHERE session_id = ?
				ORDER BY timestamp DESC
				LIMIT ?`, sessionID, limit)
		} else {
			rows, err = database.Query(`
				SELECT event_type, timestamp,
				       COALESCE(tool_name, ''),
				       COALESCE(input_summary, ''),
				       COALESCE(output_summary, ''),
				       COALESCE(feature_id, ''),
				       COALESCE(agent_id, ''),
				       COALESCE(status, ''),
				       COALESCE(subagent_type, '')
				FROM agent_events
				ORDER BY timestamp DESC
				LIMIT ?`, limit)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var eventType, ts, toolName, inputSum, outputSum, featureID string
			var agentID, status, subagentType string
			if err := rows.Scan(&eventType, &ts, &toolName, &inputSum, &outputSum,
				&featureID, &agentID, &status, &subagentType); err != nil {
				continue
			}
			summary := buildTimelineSummary(eventType, toolName, inputSum, subagentType)
			entries = append(entries, map[string]any{
				"type":       eventType,
				"timestamp":  ts,
				"summary":    summary,
				"feature_id": featureID,
				"details": map[string]string{
					"agent_id":      agentID,
					"tool_name":     toolName,
					"input_summary": inputSum,
					"output":        outputSum,
					"status":        status,
					"subagent_type": subagentType,
				},
			})
		}

		// Append session start/end markers when a specific session is requested.
		if sessionID != "" {
			sessionRows, serr := database.Query(`
				SELECT created_at, COALESCE(completed_at, ''),
				       agent_assigned, COALESCE(status, '')
				FROM sessions
				WHERE session_id = ?
				LIMIT 1`, sessionID)
			if serr == nil {
				defer sessionRows.Close()
				for sessionRows.Next() {
					var createdAt, completedAt, agentAssigned, sessStatus string
					if err := sessionRows.Scan(&createdAt, &completedAt, &agentAssigned, &sessStatus); err != nil {
						continue
					}
					entries = append(entries, map[string]any{
						"type":       "session_start",
						"timestamp":  createdAt,
						"summary":    "Session started by " + agentAssigned,
						"feature_id": "",
						"details": map[string]string{
							"agent_id": agentAssigned,
							"status":   sessStatus,
						},
					})
					if completedAt != "" {
						entries = append(entries, map[string]any{
							"type":       "session_end",
							"timestamp":  completedAt,
							"summary":    "Session completed",
							"feature_id": "",
							"details": map[string]string{
								"agent_id": agentAssigned,
								"status":   sessStatus,
							},
						})
					}
				}
			}
		}

		respondJSON(w, entries)
	}
}

// buildTimelineSummary produces a human-readable one-line summary for a timeline entry.
func buildTimelineSummary(eventType, toolName, inputSum, subagentType string) string {
	switch eventType {
	case "tool_call":
		if toolName != "" {
			return "Tool: " + toolName
		}
		return "Tool call"
	case "tool_result":
		if toolName != "" {
			return "Result: " + toolName
		}
		return "Tool result"
	case "delegation", "task_delegation":
		if subagentType != "" {
			return "Delegated to " + subagentType
		}
		return "Delegation"
	case "completion":
		return "Completed"
	case "start":
		return "Agent started"
	case "end":
		return "Agent ended"
	case "error":
		if inputSum != "" {
			return "Error: " + inputSum
		}
		return "Error"
	case "check_point":
		if inputSum != "" {
			return inputSum
		}
		return "Checkpoint"
	default:
		if inputSum != "" {
			return inputSum
		}
		return eventType
	}
}
