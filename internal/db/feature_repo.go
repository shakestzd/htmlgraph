package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Feature is a lightweight row struct for the features table.
// The full Node model lives in internal/models; this is for DB CRUD only.
type Feature struct {
	ID             string
	Type           string
	Title          string
	Description    string
	Status         string
	Priority       string
	AssignedTo     string
	TrackID        string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	StepsTotal     int
	StepsCompleted int
}

// InsertFeature creates a new feature row.
func InsertFeature(db *sql.DB, f *Feature) error {
	_, err := db.Exec(`
		INSERT INTO features (id, type, title, description, status, priority,
			assigned_to, track_id, created_at, updated_at,
			steps_total, steps_completed)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, f.Type, f.Title, nullStr(f.Description),
		f.Status, f.Priority,
		nullStr(f.AssignedTo), nullStr(f.TrackID),
		f.CreatedAt.UTC().Format(time.RFC3339),
		f.UpdatedAt.UTC().Format(time.RFC3339),
		f.StepsTotal, f.StepsCompleted,
	)
	if err != nil {
		return fmt.Errorf("insert feature %s: %w", f.ID, err)
	}
	return nil
}

// GetFeature retrieves a feature by ID.
func GetFeature(db *sql.DB, id string) (*Feature, error) {
	row := db.QueryRow(`
		SELECT id, type, title, description, status, priority,
			assigned_to, track_id, created_at, updated_at,
			steps_total, steps_completed
		FROM features WHERE id = ?`, id)

	f := &Feature{}
	var desc, assignedTo, trackID sql.NullString
	var createdStr, updatedStr string

	err := row.Scan(
		&f.ID, &f.Type, &f.Title, &desc, &f.Status, &f.Priority,
		&assignedTo, &trackID, &createdStr, &updatedStr,
		&f.StepsTotal, &f.StepsCompleted,
	)
	if err != nil {
		return nil, fmt.Errorf("get feature %s: %w", id, err)
	}

	f.Description = desc.String
	f.AssignedTo = assignedTo.String
	f.TrackID = trackID.String
	f.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)

	return f, nil
}

// UpsertFeature inserts or updates a feature row.
// On conflict by id, all mutable fields are updated.
func UpsertFeature(database *sql.DB, f *Feature) error {
	_, err := database.Exec(`
		INSERT INTO features (id, type, title, description, status, priority,
			assigned_to, track_id, created_at, updated_at,
			steps_total, steps_completed)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			status = excluded.status,
			priority = excluded.priority,
			assigned_to = excluded.assigned_to,
			track_id = excluded.track_id,
			updated_at = excluded.updated_at,
			steps_total = excluded.steps_total,
			steps_completed = excluded.steps_completed`,
		f.ID, f.Type, f.Title, nullStr(f.Description),
		f.Status, f.Priority,
		nullStr(f.AssignedTo), nullStr(f.TrackID),
		f.CreatedAt.UTC().Format(time.RFC3339),
		f.UpdatedAt.UTC().Format(time.RFC3339),
		f.StepsTotal, f.StepsCompleted,
	)
	if err != nil {
		return fmt.Errorf("upsert feature %s: %w", f.ID, err)
	}
	return nil
}

// UpdateFeatureStatus updates a feature's status (and updated_at).
func UpdateFeatureStatus(db *sql.DB, id, status string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`
		UPDATE features SET status = ?, updated_at = ? WHERE id = ?`,
		status, now, id,
	)
	return err
}

// UpdateFeatureSteps updates the steps_total and steps_completed counters
// for a feature in the SQLite read index. HTML is canonical; this is best-effort.
func UpdateFeatureSteps(db *sql.DB, featureID string, total, completed int) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`
		UPDATE features SET steps_total = ?, steps_completed = ?, updated_at = ?
		WHERE id = ?`,
		total, completed, now, featureID,
	)
	return err
}

// ListFeaturesByStatus returns features matching the given status,
// ordered by priority DESC, created_at DESC.
func ListFeaturesByStatus(db *sql.DB, status string, limit int) ([]Feature, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.Query(`
		SELECT id, type, title, status, priority, track_id,
			created_at, updated_at, steps_total, steps_completed
		FROM features
		WHERE status = ?
		ORDER BY
			CASE priority
				WHEN 'critical' THEN 0
				WHEN 'high' THEN 1
				WHEN 'medium' THEN 2
				WHEN 'low' THEN 3
			END,
			created_at DESC
		LIMIT ?`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var features []Feature
	for rows.Next() {
		var f Feature
		var trackID sql.NullString
		var createdStr, updatedStr string

		if err := rows.Scan(
			&f.ID, &f.Type, &f.Title, &f.Status, &f.Priority, &trackID,
			&createdStr, &updatedStr, &f.StepsTotal, &f.StepsCompleted,
		); err != nil {
			return nil, err
		}
		f.TrackID = trackID.String
		f.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		features = append(features, f)
	}
	return features, rows.Err()
}
