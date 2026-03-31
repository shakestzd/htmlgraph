package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/shakestzd/htmlgraph/packages/go/internal/models"
)

// UpsertFeatureFile inserts a feature_file row or updates last_seen on conflict.
// The UNIQUE constraint is (feature_id, file_path), so re-touching the same file
// within the same feature just refreshes the timestamp and operation.
func UpsertFeatureFile(db *sql.DB, ff *models.FeatureFile) error {
	_, err := db.Exec(`
		INSERT INTO feature_files
			(id, feature_id, file_path, operation, session_id,
			 first_seen, last_seen, created_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(feature_id, file_path) DO UPDATE SET
			last_seen  = CURRENT_TIMESTAMP,
			operation  = excluded.operation,
			session_id = excluded.session_id`,
		ff.ID, ff.FeatureID, ff.FilePath, ff.Operation, nullStr(ff.SessionID),
	)
	if err != nil {
		return fmt.Errorf("upsert feature_file %s/%s: %w", ff.FeatureID, ff.FilePath, err)
	}
	return nil
}

// ListFilesByFeature returns all file paths recorded for a feature.
func ListFilesByFeature(db *sql.DB, featureID string) ([]models.FeatureFile, error) {
	rows, err := db.Query(`
		SELECT id, feature_id, file_path, operation,
		       COALESCE(session_id, ''),
		       first_seen, last_seen, created_at
		FROM feature_files
		WHERE feature_id = ?
		ORDER BY last_seen DESC`, featureID)
	if err != nil {
		return nil, fmt.Errorf("list files for feature %s: %w", featureID, err)
	}
	defer rows.Close()
	return scanFeatureFiles(rows)
}

// ListFeaturesByFile returns all features that have touched a given file path.
func ListFeaturesByFile(db *sql.DB, filePath string) ([]models.FeatureFile, error) {
	rows, err := db.Query(`
		SELECT id, feature_id, file_path, operation,
		       COALESCE(session_id, ''),
		       first_seen, last_seen, created_at
		FROM feature_files
		WHERE file_path = ?
		ORDER BY last_seen DESC`, filePath)
	if err != nil {
		return nil, fmt.Errorf("list features for file %s: %w", filePath, err)
	}
	defer rows.Close()
	return scanFeatureFiles(rows)
}

// RelatedFeature summarises another feature that shares files with a given one.
type RelatedFeature struct {
	FeatureID   string   `json:"feature_id"`
	Title       string   `json:"title"`
	SharedCount int      `json:"shared_count"`
	SharedFiles []string `json:"shared_files"`
}

// FindRelatedFeatures returns features that share at least one file with
// featureID, ordered by shared file count descending.
func FindRelatedFeatures(db *sql.DB, featureID string) ([]RelatedFeature, error) {
	// Step 1: find related feature IDs and their shared counts.
	rows, err := db.Query(`
		SELECT ff2.feature_id, COUNT(DISTINCT ff2.file_path) AS shared_count
		FROM feature_files ff1
		JOIN feature_files ff2 ON ff1.file_path = ff2.file_path
		WHERE ff1.feature_id = ? AND ff2.feature_id != ?
		GROUP BY ff2.feature_id
		ORDER BY shared_count DESC`, featureID, featureID)
	if err != nil {
		return nil, fmt.Errorf("find related features for %s: %w", featureID, err)
	}
	defer rows.Close()

	var related []RelatedFeature
	for rows.Next() {
		var r RelatedFeature
		if err := rows.Scan(&r.FeatureID, &r.SharedCount); err != nil {
			continue
		}
		related = append(related, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Step 2: populate shared file paths and title for each related feature.
	for i := range related {
		rid := related[i].FeatureID

		// Shared file paths.
		fileRows, ferr := db.Query(`
			SELECT ff2.file_path
			FROM feature_files ff1
			JOIN feature_files ff2 ON ff1.file_path = ff2.file_path
			WHERE ff1.feature_id = ? AND ff2.feature_id = ?
			GROUP BY ff2.file_path
			ORDER BY ff2.file_path`, featureID, rid)
		if ferr == nil {
			for fileRows.Next() {
				var fp string
				if fileRows.Scan(&fp) == nil {
					related[i].SharedFiles = append(related[i].SharedFiles, fp)
				}
			}
			fileRows.Close()
		}

		// Title from the features table (empty when not yet indexed).
		var title string
		_ = db.QueryRow(`SELECT COALESCE(title, '') FROM features WHERE id = ?`, rid).Scan(&title)
		related[i].Title = title
	}

	return related, nil
}

// scanFeatureFiles reads rows into a slice of FeatureFile.
func scanFeatureFiles(rows *sql.Rows) ([]models.FeatureFile, error) {
	var out []models.FeatureFile
	for rows.Next() {
		var ff models.FeatureFile
		var firstSeen, lastSeen, createdAt string
		if err := rows.Scan(
			&ff.ID, &ff.FeatureID, &ff.FilePath, &ff.Operation,
			&ff.SessionID, &firstSeen, &lastSeen, &createdAt,
		); err != nil {
			continue
		}
		ff.FirstSeen, _ = time.Parse("2006-01-02 15:04:05", firstSeen)
		ff.LastSeen, _ = time.Parse("2006-01-02 15:04:05", lastSeen)
		ff.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		out = append(out, ff)
	}
	return out, rows.Err()
}
