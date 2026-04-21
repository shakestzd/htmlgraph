package hooks

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/otel/materialize"
	"github.com/shakestzd/htmlgraph/internal/paths"
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

	// Finalize session HTML file (non-critical, errors silently logged).
	var evtCount int
	_ = database.QueryRow(`SELECT COUNT(*) FROM agent_events WHERE session_id = ?`, sessionID).Scan(&evtCount)
	FinalizeSessionHTML(projectDir, sessionID, now, "completed", evtCount)

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

	// Populate features_worked_on from distinct feature_ids in agent_events.
	if feats, fErr := db.DistinctFeatureIDs(database, sessionID); fErr == nil && len(feats) > 0 {
		if featsJSON, jErr := json.Marshal(feats); jErr == nil {
			database.Exec(`UPDATE sessions SET features_worked_on = ? WHERE session_id = ?`,
				string(featsJSON), sessionID)
		}
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

	// Clean up the session-scoped project dir hint file now that this session is ending.
	paths.CleanupSessionHint(sessionID)

	// Backfill any user prompts missed by the live UserPromptSubmit hook path.
	// transcript_path may come from the current event or from the sessions table
	// (written by SessionStart or Stop). Non-fatal: errors are logged only.
	backfillTranscriptPath := event.TranscriptPath
	if backfillTranscriptPath == "" {
		var storedPath sql.NullString
		_ = database.QueryRow(`SELECT transcript_path FROM sessions WHERE session_id = ?`, sessionID).Scan(&storedPath)
		if storedPath.Valid {
			backfillTranscriptPath = storedPath.String
		}
	}
	if backfillTranscriptPath != "" {
		if n, err := backfillMissedUserPrompts(database, projectDir, sessionID, backfillTranscriptPath); err != nil {
			debugLog(projectDir, "[user-prompt-backfill] session-end: %v", err)
		} else if n > 0 {
			debugLog(projectDir, "[user-prompt-backfill] session-end: %d prompts recovered (session=%s)", n, sessionID[:minLen(sessionID, 8)])
		}
	}

	// Materialize OTel rollup (no-op if no signals received for this session).
	// Non-fatal: errors are logged but do not block SessionEnd completion.
	if err := materialize.Materialize(database, projectDir, sessionID); err != nil {
		debugLog(projectDir, "[error] handler=session-end session=%s: materialize otel: %v", sessionID[:minLen(sessionID, 8)], err)
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
