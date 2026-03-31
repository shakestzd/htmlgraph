package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shakestzd/htmlgraph/internal/models"
)

// activeStatusList returns the active claim statuses as a SQL IN-list literal.
// Example: "'proposed','claimed','in_progress','blocked','handoff_pending'"
func activeStatusList() string {
	quoted := make([]string, len(models.ActiveClaimStatuses))
	for i, s := range models.ActiveClaimStatuses {
		quoted[i] = "'" + string(s) + "'"
	}
	return strings.Join(quoted, ",")
}

// ClaimItem creates a new claim for a work item. It first reaps expired claims,
// then attempts an atomic INSERT. If the work item already has an active claim
// by another session, returns an error describing the conflict.
func ClaimItem(db *sql.DB, claim *models.Claim, leaseDuration time.Duration) error {
	if _, err := ReapExpiredClaims(db); err != nil {
		return fmt.Errorf("reap before claim: %w", err)
	}

	now := time.Now().UTC()
	claim.LeasedAt = now
	claim.LeaseExpiresAt = now.Add(leaseDuration)
	claim.LastHeartbeatAt = now
	claim.CreatedAt = now
	claim.UpdatedAt = now

	if claim.Status == "" {
		claim.Status = models.ClaimProposed
	}
	if claim.OwnerAgent == "" {
		claim.OwnerAgent = "claude-code"
	}

	// Ensure FK-referenced rows exist. HTML is canonical; SQLite is a read
	// index that may not have the row yet (e.g. workitem tests, CLI-only usage).
	ensureFeatureRow(db, claim.WorkItemID)
	ensureSessionRow(db, claim.OwnerSessionID, claim.OwnerAgent)

	activeList := activeStatusList()
	query := fmt.Sprintf(`
		INSERT INTO claims (
			claim_id, work_item_id, track_id, owner_session_id, owner_agent,
			status, intended_output, write_scope,
			leased_at, lease_expires_at, last_heartbeat_at,
			dependencies, progress_notes, blocker_reason,
			created_at, updated_at
		)
		SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		WHERE NOT EXISTS (
			SELECT 1 FROM claims
			WHERE work_item_id = ?
			  AND status IN (%s)
		)`, activeList)

	result, err := db.Exec(query,
		claim.ClaimID, claim.WorkItemID, nullStr(claim.TrackID),
		claim.OwnerSessionID, claim.OwnerAgent,
		string(claim.Status), nullStr(claim.IntendedOutput),
		nullJSON(claim.WriteScope),
		now.Format(time.RFC3339), claim.LeaseExpiresAt.Format(time.RFC3339),
		now.Format(time.RFC3339),
		nullJSON(claim.Dependencies), nullStr(claim.ProgressNotes),
		nullStr(claim.BlockerReason),
		now.Format(time.RFC3339), now.Format(time.RFC3339),
		claim.WorkItemID,
	)
	if err != nil {
		return fmt.Errorf("insert claim %s: %w", claim.ClaimID, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("claim rows affected: %w", err)
	}
	if rows == 0 {
		existing, qErr := GetActiveClaim(db, claim.WorkItemID)
		if qErr != nil {
			return fmt.Errorf("work item %s already has an active claim (lookup failed: %w)", claim.WorkItemID, qErr)
		}
		return fmt.Errorf("work item %s already claimed by session %s (claim %s, status %s)",
			claim.WorkItemID, existing.OwnerSessionID, existing.ClaimID, existing.Status)
	}
	return nil
}

// HeartbeatClaim renews the lease on an active claim owned by sessionID.
// Updates last_heartbeat_at and extends lease_expires_at by leaseDuration.
func HeartbeatClaim(db *sql.DB, claimID, sessionID string, leaseDuration time.Duration) error {
	now := time.Now().UTC()
	newExpiry := now.Add(leaseDuration)
	activeList := activeStatusList()

	query := fmt.Sprintf(`
		UPDATE claims
		SET last_heartbeat_at = ?, lease_expires_at = ?, updated_at = ?
		WHERE claim_id = ?
		  AND owner_session_id = ?
		  AND status IN (%s)`, activeList)

	result, err := db.Exec(query,
		now.Format(time.RFC3339), newExpiry.Format(time.RFC3339),
		now.Format(time.RFC3339), claimID, sessionID,
	)
	if err != nil {
		return fmt.Errorf("heartbeat claim %s: %w", claimID, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("claim %s not found, not owned by session %s, or not active", claimID, sessionID)
	}
	return nil
}

// HeartbeatClaimByWorkItem renews the lease on the active claim for a work item.
// This is the main hook entry point — hooks know the work item ID, not the claim ID.
func HeartbeatClaimByWorkItem(db *sql.DB, workItemID, sessionID string, leaseDuration time.Duration) error {
	now := time.Now().UTC()
	newExpiry := now.Add(leaseDuration)
	activeList := activeStatusList()

	query := fmt.Sprintf(`
		UPDATE claims
		SET last_heartbeat_at = ?, lease_expires_at = ?, updated_at = ?
		WHERE work_item_id = ?
		  AND owner_session_id = ?
		  AND status IN (%s)`, activeList)

	result, err := db.Exec(query,
		now.Format(time.RFC3339), newExpiry.Format(time.RFC3339),
		now.Format(time.RFC3339), workItemID, sessionID,
	)
	if err != nil {
		return fmt.Errorf("heartbeat claim for work item %s: %w", workItemID, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no active claim for work item %s owned by session %s", workItemID, sessionID)
	}
	return nil
}

// TransitionClaim moves a claim to a new status, enforcing the state machine.
// Returns error if the transition is invalid per ValidClaimTransitions.
func TransitionClaim(db *sql.DB, claimID string, toStatus models.ClaimStatus) error {
	claim, err := GetClaim(db, claimID)
	if err != nil {
		return fmt.Errorf("get claim for transition: %w", err)
	}
	if !claim.CanTransitionTo(toStatus) {
		return fmt.Errorf("invalid transition %s -> %s for claim %s", claim.Status, toStatus, claimID)
	}
	now := time.Now().UTC()
	_, err = db.Exec(`
		UPDATE claims SET status = ?, updated_at = ? WHERE claim_id = ?`,
		string(toStatus), now.Format(time.RFC3339), claimID,
	)
	if err != nil {
		return fmt.Errorf("transition claim %s: %w", claimID, err)
	}
	return nil
}

// ReleaseClaim sets a claim to completed or abandoned.
// terminalStatus must be ClaimCompleted or ClaimAbandoned.
func ReleaseClaim(db *sql.DB, claimID, sessionID string, terminalStatus models.ClaimStatus) error {
	if terminalStatus != models.ClaimCompleted && terminalStatus != models.ClaimAbandoned {
		return fmt.Errorf("terminalStatus must be completed or abandoned, got %s", terminalStatus)
	}
	now := time.Now().UTC()
	result, err := db.Exec(`
		UPDATE claims SET status = ?, updated_at = ?
		WHERE claim_id = ? AND owner_session_id = ?`,
		string(terminalStatus), now.Format(time.RFC3339), claimID, sessionID,
	)
	if err != nil {
		return fmt.Errorf("release claim %s: %w", claimID, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("claim %s not found or not owned by session %s", claimID, sessionID)
	}
	return nil
}

// ReleaseAllClaimsForSession marks all active claims for a session as abandoned.
// Called on session end. Returns the number of claims released.
func ReleaseAllClaimsForSession(db *sql.DB, sessionID string) (int, error) {
	now := time.Now().UTC()
	activeList := activeStatusList()

	query := fmt.Sprintf(`
		UPDATE claims SET status = 'abandoned', updated_at = ?
		WHERE owner_session_id = ?
		  AND status IN (%s)`, activeList)

	result, err := db.Exec(query, now.Format(time.RFC3339), sessionID)
	if err != nil {
		return 0, fmt.Errorf("release all claims for session %s: %w", sessionID, err)
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// ReapExpiredClaims transitions all expired-lease active claims to ClaimExpired.
// Returns the number of claims reaped.
func ReapExpiredClaims(db *sql.DB) (int, error) {
	now := time.Now().UTC()
	activeList := activeStatusList()

	query := fmt.Sprintf(`
		UPDATE claims SET status = 'expired', updated_at = ?
		WHERE lease_expires_at < ?
		  AND status IN (%s)`, activeList)

	result, err := db.Exec(query, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("reap expired claims: %w", err)
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// GetActiveClaim returns the active claim for a work item, or nil if none.
func GetActiveClaim(db *sql.DB, workItemID string) (*models.Claim, error) {
	activeList := activeStatusList()
	query := fmt.Sprintf(`
		SELECT claim_id, work_item_id, track_id, owner_session_id, owner_agent,
		       status, intended_output, write_scope,
		       leased_at, lease_expires_at, last_heartbeat_at,
		       dependencies, progress_notes, blocker_reason,
		       created_at, updated_at
		FROM claims
		WHERE work_item_id = ? AND status IN (%s)
		LIMIT 1`, activeList)

	row := db.QueryRow(query, workItemID)
	c, err := scanClaim(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active claim for %s: %w", workItemID, err)
	}
	return c, nil
}

// GetClaim returns a claim by ID.
func GetClaim(db *sql.DB, claimID string) (*models.Claim, error) {
	row := db.QueryRow(`
		SELECT claim_id, work_item_id, track_id, owner_session_id, owner_agent,
		       status, intended_output, write_scope,
		       leased_at, lease_expires_at, last_heartbeat_at,
		       dependencies, progress_notes, blocker_reason,
		       created_at, updated_at
		FROM claims WHERE claim_id = ?`, claimID)

	c, err := scanClaim(row)
	if err != nil {
		return nil, fmt.Errorf("get claim %s: %w", claimID, err)
	}
	return c, nil
}

// ListActiveClaimsBySession returns all active claims for a session.
func ListActiveClaimsBySession(db *sql.DB, sessionID string) ([]models.Claim, error) {
	activeList := activeStatusList()
	query := fmt.Sprintf(`
		SELECT claim_id, work_item_id, track_id, owner_session_id, owner_agent,
		       status, intended_output, write_scope,
		       leased_at, lease_expires_at, last_heartbeat_at,
		       dependencies, progress_notes, blocker_reason,
		       created_at, updated_at
		FROM claims
		WHERE owner_session_id = ? AND status IN (%s)
		ORDER BY created_at DESC`, activeList)

	return queryClaims(db, query, sessionID)
}

// ListClaims returns claims matching the given filters.
// If sessionID is empty, returns all. If statusFilter is empty, returns all statuses.
func ListClaims(db *sql.DB, sessionID, statusFilter string, limit int) ([]models.Claim, error) {
	if limit <= 0 {
		limit = 100
	}

	var conditions []string
	var args []any

	if sessionID != "" {
		conditions = append(conditions, "owner_session_id = ?")
		args = append(args, sessionID)
	}
	if statusFilter != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, statusFilter)
	}

	query := `
		SELECT claim_id, work_item_id, track_id, owner_session_id, owner_agent,
		       status, intended_output, write_scope,
		       leased_at, lease_expires_at, last_heartbeat_at,
		       dependencies, progress_notes, blocker_reason,
		       created_at, updated_at
		FROM claims`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	return queryClaims(db, query, args...)
}

// scanClaim scans a single claim row from a *sql.Row.
func scanClaim(row *sql.Row) (*models.Claim, error) {
	c := &models.Claim{}
	var (
		trackID, intendedOutput, progressNotes, blockerReason sql.NullString
		writeScope, dependencies                              sql.NullString
		leasedStr, expiresStr, heartbeatStr                   string
		createdStr, updatedStr                                string
	)
	err := row.Scan(
		&c.ClaimID, &c.WorkItemID, &trackID, &c.OwnerSessionID, &c.OwnerAgent,
		&c.Status, &intendedOutput, &writeScope,
		&leasedStr, &expiresStr, &heartbeatStr,
		&dependencies, &progressNotes, &blockerReason,
		&createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}
	c.TrackID = trackID.String
	c.IntendedOutput = intendedOutput.String
	c.ProgressNotes = progressNotes.String
	c.BlockerReason = blockerReason.String
	if writeScope.Valid && writeScope.String != "" {
		c.WriteScope = json.RawMessage(writeScope.String)
	}
	if dependencies.Valid && dependencies.String != "" {
		c.Dependencies = json.RawMessage(dependencies.String)
	}
	c.LeasedAt, _ = time.Parse(time.RFC3339, leasedStr)
	c.LeaseExpiresAt, _ = time.Parse(time.RFC3339, expiresStr)
	c.LastHeartbeatAt, _ = time.Parse(time.RFC3339, heartbeatStr)
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return c, nil
}

// scanClaimRow scans a claim from a *sql.Rows (multi-row cursor).
func scanClaimRow(rows *sql.Rows) (models.Claim, error) {
	c := models.Claim{}
	var (
		trackID, intendedOutput, progressNotes, blockerReason sql.NullString
		writeScope, dependencies                              sql.NullString
		leasedStr, expiresStr, heartbeatStr                   string
		createdStr, updatedStr                                string
	)
	err := rows.Scan(
		&c.ClaimID, &c.WorkItemID, &trackID, &c.OwnerSessionID, &c.OwnerAgent,
		&c.Status, &intendedOutput, &writeScope,
		&leasedStr, &expiresStr, &heartbeatStr,
		&dependencies, &progressNotes, &blockerReason,
		&createdStr, &updatedStr,
	)
	if err != nil {
		return c, err
	}
	c.TrackID = trackID.String
	c.IntendedOutput = intendedOutput.String
	c.ProgressNotes = progressNotes.String
	c.BlockerReason = blockerReason.String
	if writeScope.Valid && writeScope.String != "" {
		c.WriteScope = json.RawMessage(writeScope.String)
	}
	if dependencies.Valid && dependencies.String != "" {
		c.Dependencies = json.RawMessage(dependencies.String)
	}
	c.LeasedAt, _ = time.Parse(time.RFC3339, leasedStr)
	c.LeaseExpiresAt, _ = time.Parse(time.RFC3339, expiresStr)
	c.LastHeartbeatAt, _ = time.Parse(time.RFC3339, heartbeatStr)
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return c, nil
}

// queryClaims executes a query and returns a slice of Claim.
func queryClaims(db *sql.DB, query string, args ...any) ([]models.Claim, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query claims: %w", err)
	}
	defer rows.Close()

	var claims []models.Claim
	for rows.Next() {
		c, err := scanClaimRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan claim row: %w", err)
		}
		claims = append(claims, c)
	}
	return claims, rows.Err()
}

// nullJSON returns sql.NullString for a JSON field — empty if nil or zero-length.
func nullJSON(raw json.RawMessage) sql.NullString {
	if len(raw) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{String: string(raw), Valid: true}
}

// ensureFeatureRow creates a placeholder feature row if it doesn't exist.
// This handles the case where a feature is created via HTML but not yet indexed
// into the database, or when tests create features without database indexing.
// Best-effort: errors are logged but not returned, since HTML is canonical.
func ensureFeatureRow(db *sql.DB, featureID string) {
	now := time.Now().UTC()
	_, _ = db.Exec(`
		INSERT OR IGNORE INTO features (id, type, title, status, priority, created_at, updated_at)
		VALUES (?, 'feature', '', 'todo', 'medium', ?, ?)`,
		featureID, now.Format(time.RFC3339), now.Format(time.RFC3339))
}

// ensureSessionRow creates a placeholder session row if it doesn't exist.
// This handles the case where a session is referenced before it's been created,
// or when tests create claims without proper session setup.
// Best-effort: errors are logged but not returned.
func ensureSessionRow(db *sql.DB, sessionID, agent string) {
	now := time.Now().UTC()
	_, _ = db.Exec(`
		INSERT OR IGNORE INTO sessions (session_id, agent, created_at, ended_at)
		VALUES (?, ?, ?, ?)`,
		sessionID, agent, now.Format(time.RFC3339), nil)
}
