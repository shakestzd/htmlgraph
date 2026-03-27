package hooks

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/shakestzd/htmlgraph/internal/db"
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
	_ = err // Non-fatal

	// Mark lineage trace complete so tree queries show accurate status.
	_ = db.CompleteLineageTrace(database, sessionID)

	return &HookResult{Continue: true}, nil
}

// SessionResume handles the SessionResume Claude Code hook event.
// It updates the session status back to active and refreshes env vars.
func SessionResume(event *CloudEvent, database *sql.DB, projectDir string) (*HookResult, error) {
	sessionID := EnvSessionID(event.SessionID)
	if sessionID == "" {
		return &HookResult{Continue: true}, nil
	}

	_, _ = database.Exec(`
		UPDATE sessions
		SET status = 'active', completed_at = NULL
		WHERE session_id = ? AND status = 'completed'`,
		sessionID,
	)

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
