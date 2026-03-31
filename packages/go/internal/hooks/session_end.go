package hooks

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/shakestzd/htmlgraph/packages/go/internal/db"
)

// SessionEnd handles the SessionEnd Claude Code hook event.
// It marks the session as completed and records the end commit.
func SessionEnd(event *CloudEvent, database *sql.DB, projectDir string) (*HookResult, error) {
	sessionID := EnvSessionID(event.SessionID)
	if sessionID == "" {
		return &HookResult{Continue: true}, nil
	}

	endCommit := headCommit(projectDir)
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := database.Exec(`
		UPDATE sessions
		SET status = 'completed',
		    completed_at = ?,
		    end_commit = COALESCE(NULLIF(?, ''), end_commit)
		WHERE session_id = ?`,
		now, endCommit, sessionID,
	)
	if err != nil {
		debugLog(projectDir, "[error] handler=session-end session=%s: update sessions: %v", sessionID[:minLen(sessionID, 8)], err)
	}

	// Store transcript_path and termination reason if provided.
	if event.TranscriptPath != "" || event.Reason != "" {
		_, _ = database.Exec(`
			UPDATE sessions
			SET transcript_path = COALESCE(NULLIF(?, ''), transcript_path),
			    metadata = json_set(COALESCE(metadata, '{}'), '$.end_reason', ?)
			WHERE session_id = ?`,
			event.TranscriptPath, event.Reason, sessionID,
		)
	}

	// Mark lineage trace complete so tree queries show accurate status.
	if err := db.CompleteLineageTrace(database, sessionID); err != nil {
		debugLog(projectDir, "[error] handler=session-end session=%s: complete lineage trace: %v", sessionID[:minLen(sessionID, 8)], err)
	}

	// Release all active claims held by this session.
	if released, err := db.ReleaseAllClaimsForSession(database, sessionID); err != nil {
		debugLog(projectDir, "[error] handler=session-end session=%s: release claims: %v", sessionID[:minLen(sessionID, 8)], err)
	} else if released > 0 {
		debugLog(projectDir, "[htmlgraph] session-end: released %d claims for session %s", released, sessionID[:minLen(sessionID, 8)])
	}

	return &HookResult{Continue: true}, nil
}

// SessionResume handles the SessionResume Claude Code hook event.
// It updates the session status back to active and refreshes env vars.
func SessionResume(event *CloudEvent, database *sql.DB, projectDir string) (*HookResult, error) {
	sessionID := EnvSessionID(event.SessionID)
	if sessionID == "" {
		return &HookResult{Continue: true}, nil
	}

	if _, err := database.Exec(`
		UPDATE sessions
		SET status = 'active', completed_at = NULL
		WHERE session_id = ? AND status = 'completed'`,
		sessionID,
	); err != nil {
		debugLog(projectDir, "[error] handler=session-resume session=%s: update sessions: %v", sessionID[:minLen(sessionID, 8)], err)
	}

	// Re-export env vars so downstream hooks have the session ID.
	writeEnvVars(sessionID, projectDir)

	// Fetch active feature for context message.
	var featID sql.NullString
	_ = database.QueryRow(
		`SELECT active_feature_id FROM sessions WHERE session_id = ?`, sessionID,
	).Scan(&featID)

	msg := fmt.Sprintf("[HtmlGraph] Session %s resumed.", sessionID[:minLen(sessionID, 8)])
	if featID.Valid && featID.String != "" {
		msg += fmt.Sprintf(" Active feature: %s", featID.String)
	}

	return &HookResult{Continue: true, AdditionalContext: msg}, nil
}

func minLen(s string, n int) int {
	if len(s) < n {
		return len(s)
	}
	return n
}
