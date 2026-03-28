package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/htmlparse"
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
			SELECT s.session_id, s.agent_assigned, s.status, s.created_at,
			       COALESCE(s.completed_at, ''), s.total_events,
			       COALESCE(s.active_feature_id, ''), COALESCE(s.model, ''),
			       COALESCE(s.title, '') AS title,
			       COALESCE((SELECT SUBSTR(m.content, 1, 120)
			                 FROM messages m
			                 WHERE m.session_id = s.session_id AND m.role = 'user'
			                 ORDER BY m.ordinal LIMIT 1), '') AS first_msg,
			       COALESCE((SELECT COUNT(*) FROM messages m2
			                 WHERE m2.session_id = s.session_id), 0) AS msg_count
			FROM sessions s
			WHERE s.total_events > 0
			   OR EXISTS (SELECT 1 FROM messages m WHERE m.session_id = s.session_id)
			   OR s.status = 'active'
			ORDER BY s.created_at DESC
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
			var featureID, model, title, firstMsg string
			var msgCount int
			if err := rows.Scan(&sid, &agent, &status, &created, &completed,
				&totalEvents, &featureID, &model, &title, &firstMsg, &msgCount); err != nil {
				continue
			}
			sessions = append(sessions, map[string]any{
				"session_id":    sid,
				"agent":         agent,
				"status":        status,
				"created_at":    created,
				"completed_at":  completed,
				"total_events":  totalEvents,
				"feature_id":    featureID,
				"model":         model,
				"title":         title,
				"first_message": firstMsg,
				"message_count": msgCount,
			})
		}

		respondJSON(w, sessions)
	}
}

// featuresHandler returns up to 50 features, in-progress first.
// Falls back to scanning HTML files when SQLite features table is empty.
func featuresHandler(database *sql.DB, projectDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		features := featuresFromDB(database)
		if len(features) == 0 {
			features = featuresFromHTML(projectDir)
		}
		respondJSON(w, features)
	}
}

func featuresFromDB(database *sql.DB) []map[string]any {
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
		return nil
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
	return features
}

// featuresFromHTML scans .htmlgraph/features/*.html, .htmlgraph/bugs/*.html,
// .htmlgraph/spikes/*.html, .htmlgraph/tracks/*.html and parses each file.
func featuresFromHTML(projectDir string) []map[string]any {
	features := make([]map[string]any, 0, 100)
	for _, subdir := range []string{"features", "bugs", "spikes", "tracks"} {
		pattern := filepath.Join(projectDir, subdir, "*.html")
		files, _ := filepath.Glob(pattern)
		for _, f := range files {
			node, err := htmlparse.ParseFile(f)
			if err != nil || node == nil {
				continue
			}
			completed := 0
			for _, s := range node.Steps {
				if s.Completed {
					completed++
				}
			}
			features = append(features, map[string]any{
				"id":              node.ID,
				"type":            node.Type,
				"title":           node.Title,
				"status":          string(node.Status),
				"priority":        string(node.Priority),
				"track_id":        node.TrackID,
				"created_at":      node.CreatedAt.Format(time.RFC3339),
				"steps_total":     len(node.Steps),
				"steps_completed": completed,
			})
		}
	}
	return features
}

// statsHandler returns a summary of counts from the database.
// Falls back to HTML files for feature counts when SQLite features table is empty.
func statsHandler(database *sql.DB, projectDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var total, inProgress, done, todo int
		var activeSessions, totalEvents int

		database.QueryRow(`SELECT COUNT(*) FROM features`).Scan(&total)

		// If SQLite has no features, count from HTML files
		if total == 0 {
			items := featuresFromHTML(projectDir)
			total = len(items)
			for _, item := range items {
				switch item["status"] {
				case "in-progress":
					inProgress++
				case "done":
					done++
				case "todo":
					todo++
				}
			}
		} else {
			database.QueryRow(`SELECT COUNT(*) FROM features WHERE status='in-progress'`).Scan(&inProgress)
			database.QueryRow(`SELECT COUNT(*) FROM features WHERE status='done'`).Scan(&done)
			database.QueryRow(`SELECT COUNT(*) FROM features WHERE status='todo'`).Scan(&todo)
		}

		database.QueryRow(`SELECT COUNT(*) FROM sessions WHERE status='active'`).Scan(&activeSessions)
		database.QueryRow(`SELECT COUNT(*) FROM agent_events`).Scan(&totalEvents)

		// Read launch mode and timestamp
		launchMode := ""
		launchTimestamp := ""
		if data, err := os.ReadFile(filepath.Join(projectDir, ".launch-mode")); err == nil {
			content := string(data)
			if strings.Contains(content, `"yolo`) {
				launchMode = "yolo"
			}
			// Extract timestamp: {"mode":"yolo-dev","pid":123,"timestamp":"2026-..."}
			if idx := strings.Index(content, `"timestamp":"`); idx >= 0 {
				rest := content[idx+13:]
				if end := strings.Index(rest, `"`); end >= 0 {
					launchTimestamp = rest[:end]
				}
			}
		}

		respondJSON(w, map[string]any{
			"features_total":       total,
			"features_in_progress": inProgress,
			"features_done":        done,
			"features_todo":        todo,
			"active_sessions":      activeSessions,
			"total_events":         totalEvents,
			"launch_mode":          launchMode,
			"launch_timestamp":     launchTimestamp,
		})
	}
}

// timelineHandler returns chronological session activity as a JSON array.
// Supports ?session=SESSION_ID (required) and ?limit=N (default 50, max 200).
// Each entry has: { type, timestamp, summary, feature_id, details }.
// Sources: agent_events (tool calls, delegations) and sessions (start/end markers).
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

		// Pull agent_events for this session, or all recent events if no session given.
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

// transcriptHandler returns messages and tool calls for a session.
// Requires ?session=SESSION_ID. Supports ?limit=N (default 500).
func transcriptHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.URL.Query().Get("session")
		if sessionID == "" {
			http.Error(w, "session parameter required", http.StatusBadRequest)
			return
		}

		limit := 500
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 2000 {
				limit = n
			}
		}

		messages, err := dbpkg.ListMessages(database, sessionID, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		toolCalls, err := dbpkg.ListToolCalls(database, sessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Group tool calls by message ID for easy frontend consumption.
		toolsByMsg := map[int][]map[string]any{}
		for _, tc := range toolCalls {
			toolsByMsg[tc.MessageID] = append(toolsByMsg[tc.MessageID], map[string]any{
				"tool_name":   tc.ToolName,
				"category":    tc.Category,
				"tool_use_id": tc.ToolUseID,
				"input_json":  tc.InputJSON,
			})
		}

		result := make([]map[string]any, 0, len(messages))
		for _, m := range messages {
			entry := map[string]any{
				"id":               m.ID,
				"ordinal":          m.Ordinal,
				"role":             m.Role,
				"content":          m.Content,
				"timestamp":        m.Timestamp.Format(time.RFC3339),
				"has_thinking":     m.HasThinking,
				"has_tool_use":     m.HasToolUse,
				"content_length":   m.ContentLength,
				"model":            m.Model,
				"input_tokens":     m.InputTokens,
				"output_tokens":    m.OutputTokens,
				"cache_read_tokens": m.CacheReadTokens,
				"stop_reason":      m.StopReason,
			}
			if tools, ok := toolsByMsg[m.ID]; ok {
				entry["tool_calls"] = tools
			}
			result = append(result, entry)
		}

		respondJSON(w, map[string]any{
			"session_id":   sessionID,
			"message_count": len(messages),
			"tool_count":   len(toolCalls),
			"messages":     result,
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
				// Force WAL checkpoint so we see writes from hook processes.
				database.Exec("PRAGMA wal_checkpoint(PASSIVE)")

				rows, err := database.Query(`
					SELECT rowid, event_id, agent_id, event_type, timestamp,
					       tool_name, COALESCE(input_summary, ''),
					       COALESCE(output_summary, ''), session_id,
					       COALESCE(feature_id, ''), status
					FROM agent_events
					WHERE rowid > ?
					ORDER BY rowid ASC
					LIMIT 20`, lastRowID)
				if err != nil {
					continue
				}

				for rows.Next() {
					var rowid int64
					var eid, aid, etype, ts, tool, inputSum, outputSum, sid, fid, status string
					if err := rows.Scan(&rowid, &eid, &aid, &etype, &ts,
						&tool, &inputSum, &outputSum, &sid, &fid, &status); err != nil {
						continue
					}
					payload, _ := json.Marshal(map[string]string{
						"event_id":       eid,
						"agent_id":       aid,
						"event_type":     etype,
						"timestamp":      ts,
						"tool_name":      tool,
						"input_summary":  inputSum,
						"output_summary": outputSum,
						"session_id":     sid,
						"feature_id":     fid,
						"status":         status,
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
