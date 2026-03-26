package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// respondJSON encodes v as JSON and writes it with status 200.
func respondJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "encoding response", http.StatusInternalServerError)
	}
}

// initialStatsHandler returns the top-level counts the dashboard header uses.
// Matches /api/initial-stats that dashboard.html's loadInitialStats() calls.
func initialStatsHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var totalEvents, totalSessions int
		database.QueryRow(`SELECT COUNT(*) FROM agent_events`).Scan(&totalEvents)
		database.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&totalSessions)

		// Collect distinct agent IDs for the client-side agent set.
		rows, err := database.Query(
			`SELECT DISTINCT agent_id FROM agent_events ORDER BY agent_id`)
		agents := []string{}
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var a string
				if rows.Scan(&a) == nil {
					agents = append(agents, a)
				}
			}
		}

		respondJSON(w, map[string]any{
			"total_events":   totalEvents,
			"total_sessions": totalSessions,
			"agents":         agents,
		})
	}
}

// recentEventsHandler returns events ordered by timestamp DESC.
// Supports ?limit=N (default 50, max 200).
func recentEventsHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}

		rows, err := database.Query(`
			SELECT event_id, agent_id, event_type, timestamp, tool_name,
			       COALESCE(input_summary, ''), COALESCE(output_summary, ''),
			       session_id, COALESCE(feature_id, ''),
			       COALESCE(parent_event_id, ''), status
			FROM agent_events
			ORDER BY timestamp DESC
			LIMIT ?`, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		events := make([]map[string]any, 0, limit)
		for rows.Next() {
			var eventID, agentID, eventType, ts, toolName string
			var inputSum, outputSum, sessionID, featureID, parentEvtID, status string
			if err := rows.Scan(&eventID, &agentID, &eventType, &ts, &toolName,
				&inputSum, &outputSum, &sessionID, &featureID, &parentEvtID, &status); err != nil {
				continue
			}
			events = append(events, map[string]any{
				"event_id":        eventID,
				"agent_id":        agentID,
				"event_type":      eventType,
				"timestamp":       ts,
				"tool_name":       toolName,
				"input_summary":   inputSum,
				"output_summary":  outputSum,
				"session_id":      sessionID,
				"feature_id":      featureID,
				"parent_event_id": parentEvtID,
				"status":          status,
			})
		}

		respondJSON(w, events)
	}
}

// sessionsHandler returns the 20 most recent sessions.
func sessionsHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := database.Query(`
			SELECT session_id, agent_assigned, status, created_at,
			       COALESCE(completed_at, ''), total_events,
			       COALESCE(active_feature_id, ''), COALESCE(model, '')
			FROM sessions
			ORDER BY created_at DESC
			LIMIT 20`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		sessions := make([]map[string]any, 0, 20)
		for rows.Next() {
			var sid, agent, status, created, completed string
			var totalEvents int
			var featureID, model string
			if err := rows.Scan(&sid, &agent, &status, &created, &completed,
				&totalEvents, &featureID, &model); err != nil {
				continue
			}
			sessions = append(sessions, map[string]any{
				"session_id":   sid,
				"agent":        agent,
				"status":       status,
				"created_at":   created,
				"completed_at": completed,
				"total_events": totalEvents,
				"feature_id":   featureID,
				"model":        model,
			})
		}

		respondJSON(w, sessions)
	}
}

// featuresHandler returns up to 50 features, in-progress first.
func featuresHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := database.Query(`
			SELECT id, type, title, status, priority,
			       COALESCE(track_id, ''), created_at,
			       steps_total, steps_completed
			FROM features
			ORDER BY
			    CASE status WHEN 'in-progress' THEN 0 WHEN 'todo' THEN 1 ELSE 2 END,
			    created_at DESC
			LIMIT 50`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		features := make([]map[string]any, 0, 50)
		for rows.Next() {
			var id, ftype, title, status, priority, trackID, created string
			var stepsTotal, stepsCompleted int
			if err := rows.Scan(&id, &ftype, &title, &status, &priority, &trackID,
				&created, &stepsTotal, &stepsCompleted); err != nil {
				continue
			}
			features = append(features, map[string]any{
				"id":              id,
				"type":            ftype,
				"title":           title,
				"status":          status,
				"priority":        priority,
				"track_id":        trackID,
				"created_at":      created,
				"steps_total":     stepsTotal,
				"steps_completed": stepsCompleted,
			})
		}

		respondJSON(w, features)
	}
}

// statsHandler returns a summary of counts from the database.
func statsHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var total, inProgress, done, todo int
		var activeSessions, totalEvents int

		database.QueryRow(`SELECT COUNT(*) FROM features`).Scan(&total)
		database.QueryRow(`SELECT COUNT(*) FROM features WHERE status='in-progress'`).Scan(&inProgress)
		database.QueryRow(`SELECT COUNT(*) FROM features WHERE status='done'`).Scan(&done)
		database.QueryRow(`SELECT COUNT(*) FROM features WHERE status='todo'`).Scan(&todo)
		database.QueryRow(`SELECT COUNT(*) FROM sessions WHERE status='active'`).Scan(&activeSessions)
		database.QueryRow(`SELECT COUNT(*) FROM agent_events`).Scan(&totalEvents)

		respondJSON(w, map[string]any{
			"features_total":       total,
			"features_in_progress": inProgress,
			"features_done":        done,
			"features_todo":        todo,
			"active_sessions":      activeSessions,
			"total_events":         totalEvents,
		})
	}
}

// sseHandler streams new agent_events rows as Server-Sent Events.
// Polls SQLite every 2 s for rows with a rowid greater than last seen.
func sseHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Track the highest rowid seen so far.
		var lastRowID int64
		database.QueryRow(
			`SELECT COALESCE(MAX(rowid), 0) FROM agent_events`).Scan(&lastRowID)

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				rows, err := database.Query(`
					SELECT rowid, event_id, agent_id, event_type, timestamp,
					       tool_name, COALESCE(output_summary, ''), session_id,
					       COALESCE(feature_id, '')
					FROM agent_events
					WHERE rowid > ?
					ORDER BY rowid ASC
					LIMIT 20`, lastRowID)
				if err != nil {
					continue
				}

				for rows.Next() {
					var rowid int64
					var eid, aid, etype, ts, tool, summary, sid, fid string
					if err := rows.Scan(&rowid, &eid, &aid, &etype, &ts,
						&tool, &summary, &sid, &fid); err != nil {
						continue
					}
					payload, _ := json.Marshal(map[string]string{
						"event_id":   eid,
						"agent_id":   aid,
						"event_type": etype,
						"timestamp":  ts,
						"tool_name":  tool,
						"summary":    summary,
						"session_id": sid,
						"feature_id": fid,
					})
					fmt.Fprintf(w, "data: %s\n\n", payload)
					lastRowID = rowid
				}
				rows.Close()
				flusher.Flush()
			}
		}
	}
}
