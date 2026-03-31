package main

import (
	"database/sql"
	"net/http"
	"os"
	"strings"
	"time"

	dbpkg "github.com/shakestzd/htmlgraph/packages/go/internal/db"
	"github.com/shakestzd/htmlgraph/packages/go/internal/ingest"
)

// sessionIngestHandler handles POST /api/sessions/{id}/ingest.
// It discovers the JSONL file for the given session ID, parses it,
// and stores messages and tool calls in the database.
// The endpoint is idempotent: it re-ingests only when the file has changed
// since the last sync timestamp (or when no messages exist yet).
func sessionIngestHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract session ID from URL: /api/sessions/{id}/ingest
		sessionID := extractSessionID(r.URL.Path)
		if sessionID == "" {
			http.Error(w, "missing session ID", http.StatusBadRequest)
			return
		}

		msgCount, toolCount, err := ingestSession(database, sessionID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				respondJSON(w, map[string]any{
					"session_id": sessionID,
					"status":     "not_found",
					"messages":   0,
					"tool_calls": 0,
				})
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		respondJSON(w, map[string]any{
			"session_id": sessionID,
			"status":     "ok",
			"messages":   msgCount,
			"tool_calls": toolCount,
		})
	}
}

// extractSessionID pulls the session UUID from a URL path like
// /api/sessions/{id}/ingest. Returns empty string if not found.
func extractSessionID(path string) string {
	// Strip trailing slash
	path = strings.TrimSuffix(path, "/")
	// Expect: /api/sessions/{id}/ingest
	const prefix = "/api/sessions/"
	const suffix = "/ingest"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return ""
	}
	mid := path[len(prefix) : len(path)-len(suffix)]
	if mid == "" {
		return ""
	}
	return mid
}

// ingestSession finds the JSONL file for sessionID, checks whether a
// re-ingest is needed, and stores messages/tool calls. Returns counts of
// newly stored rows. Returns (0, 0, nil) when the session is already
// up-to-date (idempotent skip).
func ingestSession(database *sql.DB, sessionID string) (int, int, error) {
	files, err := ingest.DiscoverSessions("")
	if err != nil {
		return 0, 0, err
	}

	for _, sf := range files {
		if sf.SessionID != sessionID {
			continue
		}

		// Skip re-ingest if file hasn't changed since last sync.
		count, _ := dbpkg.CountMessages(database, sessionID)
		if count > 0 {
			var syncedAt string
			database.QueryRow(
				`SELECT COALESCE(transcript_synced, '') FROM sessions WHERE session_id = ?`,
				sessionID).Scan(&syncedAt)
			if syncedAt != "" {
				if info, statErr := os.Stat(sf.Path); statErr == nil {
					synced, parseErr := time.Parse(time.RFC3339, syncedAt)
					if parseErr == nil && !info.ModTime().After(synced) {
						// Already up-to-date
						return count, 0, nil
					}
				}
			}
			// File changed — clear old messages before re-ingest
			_ = dbpkg.DeleteSessionMessages(database, sessionID)
		}

		result, parseErr := ingest.ParseFile(sf.Path)
		if parseErr != nil {
			return 0, 0, parseErr
		}
		if len(result.Messages) == 0 {
			return 0, 0, nil
		}

		ensureSession(database, sessionID, result)
		msgCount, toolCount := storeParseResult(database, sessionID, "", result)
		_ = dbpkg.UpdateTranscriptSync(database, sessionID, sf.Path)
		return msgCount, toolCount, nil
	}

	return 0, 0, &notFoundError{sessionID}
}

type notFoundError struct{ id string }

func (e *notFoundError) Error() string {
	return "session " + e.id + " not found"
}
