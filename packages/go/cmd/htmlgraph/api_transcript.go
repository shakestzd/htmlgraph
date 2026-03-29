package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
)

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
				"tool_name":           tc.ToolName,
				"category":            tc.Category,
				"tool_use_id":         tc.ToolUseID,
				"input_json":          tc.InputJSON,
				"subagent_session_id": tc.SubagentSessionID,
			})
		}

		result := make([]map[string]any, 0, len(messages))
		for _, m := range messages {
			entry := map[string]any{
				"id":                m.ID,
				"ordinal":           m.Ordinal,
				"role":              m.Role,
				"content":           m.Content,
				"timestamp":         m.Timestamp.Format(time.RFC3339),
				"has_thinking":      m.HasThinking,
				"has_tool_use":      m.HasToolUse,
				"content_length":    m.ContentLength,
				"model":             m.Model,
				"input_tokens":      m.InputTokens,
				"output_tokens":     m.OutputTokens,
				"cache_read_tokens": m.CacheReadTokens,
				"stop_reason":       m.StopReason,
			}
			if tools, ok := toolsByMsg[m.ID]; ok {
				entry["tool_calls"] = tools
			}
			result = append(result, entry)
		}

		respondJSON(w, map[string]any{
			"session_id":    sessionID,
			"message_count": len(messages),
			"tool_count":    len(toolCalls),
			"messages":      result,
		})
	}
}

// subagentEventsHandler returns agent_events whose parent_event_id matches
// the given tool_use_id. Used by the transcript view to show subagent activity
// inline. GET /api/events/subagent?parent_event_id=XXX
func subagentEventsHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parentID := r.URL.Query().Get("parent_event_id")
		if parentID == "" {
			http.Error(w, "parent_event_id required", http.StatusBadRequest)
			return
		}

		rows, err := database.Query(`
			SELECT event_id, agent_id, event_type, timestamp, COALESCE(tool_name, ''),
			       COALESCE(input_summary, ''), COALESCE(output_summary, ''),
			       session_id, COALESCE(status, ''), COALESCE(subagent_type, '')
			FROM agent_events
			WHERE parent_event_id = ?
			ORDER BY timestamp ASC`, parentID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		events := make([]map[string]any, 0)
		for rows.Next() {
			var eid, aid, etype, ts, tool, inputSum, outputSum, sid, status, subType string
			if err := rows.Scan(&eid, &aid, &etype, &ts, &tool,
				&inputSum, &outputSum, &sid, &status, &subType); err != nil {
				continue
			}
			events = append(events, map[string]any{
				"event_id":       eid,
				"agent_id":       aid,
				"event_type":     etype,
				"timestamp":      ts,
				"tool_name":      tool,
				"input_summary":  inputSum,
				"output_summary": outputSum,
				"session_id":     sid,
				"status":         status,
				"subagent_type":  subType,
			})
		}
		if err := rows.Err(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		respondJSON(w, events)
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
