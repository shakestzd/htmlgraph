package main

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/htmlparse"
	"github.com/spf13/cobra"
)

func reindexCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reindex",
		Short: "Sync HTML work items to SQLite index",
		Long: `Reads all HTML work item files from .htmlgraph/features/, .htmlgraph/tracks/,
and .htmlgraph/spikes/ and upserts them into the features SQLite table.
Safe to run multiple times — uses ON CONFLICT upsert.`,
		RunE: runReindex,
	}
}

func runReindex(_ *cobra.Command, _ []string) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	database, err := dbpkg.Open(filepath.Join(htmlgraphDir, "htmlgraph.db"))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	var total, upserted, errCount int
	validIDs := make(map[string]bool)

	// Pass 1: upsert tracks first so features.track_id FK is satisfied.
	trackTotal, trackUpserted, trackErrs := reindexTracks(database, htmlgraphDir, validIDs)
	total += trackTotal
	upserted += trackUpserted
	errCount += trackErrs

	// Pass 2: upsert features, bugs, and spikes (track_id FK now safe).
	for _, dir := range []string{"features", "bugs", "spikes"} {
		t, u, e := reindexFeatureDir(database, htmlgraphDir, dir, validIDs)
		total += t
		upserted += u
		errCount += e
	}

	purged, edgesPurged := purgeStaleEntries(database, validIDs)

	fmt.Printf("Reindexed: %d upserted, %d errors (of %d HTML files)\n",
		upserted, errCount, total)
	if purged > 0 || edgesPurged > 0 {
		fmt.Printf("Purged: %d stale features, %d stale edges\n", purged, edgesPurged)
	}
	return nil
}

// reindexTracks globs both flat (tracks/*.html) and nested (tracks/*/index.html)
// track files and upserts each into the tracks table.
// Returns (total, upserted, errors).
func reindexTracks(database *sql.DB, htmlgraphDir string, validIDs map[string]bool) (int, int, int) {
	patterns := []string{
		filepath.Join(htmlgraphDir, "tracks", "*.html"),
		filepath.Join(htmlgraphDir, "tracks", "*", "index.html"),
	}

	seen := make(map[string]bool)
	var total, upserted, errCount int

	for _, pattern := range patterns {
		files, _ := filepath.Glob(pattern)
		for _, f := range files {
			if seen[f] {
				continue
			}
			seen[f] = true
			total++

			node, parseErr := htmlparse.ParseFile(f)
			if parseErr != nil {
				errCount++
				continue
			}

			createdAt, updatedAt := normalizeTimes(node.CreatedAt, node.UpdatedAt)
			track := &dbpkg.Track{
				ID:        node.ID,
				Type:      "track",
				Title:     node.Title,
				Priority:  string(node.Priority),
				Status:    normalizeStatus(string(node.Status)),
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			}

			if upsertErr := dbpkg.UpsertTrack(database, track); upsertErr != nil {
				errCount++
				continue
			}
			validIDs[node.ID] = true
			upserted++
		}
	}
	return total, upserted, errCount
}

// reindexFeatureDir upserts all HTML files in a single directory into the features table.
// Returns (total, upserted, errors).
func reindexFeatureDir(database *sql.DB, htmlgraphDir, dir string, validIDs map[string]bool) (int, int, int) {
	pattern := filepath.Join(htmlgraphDir, dir, "*.html")
	files, _ := filepath.Glob(pattern)

	var total, upserted, errCount int
	for _, f := range files {
		total++
		node, parseErr := htmlparse.ParseFile(f)
		if parseErr != nil {
			errCount++
			continue
		}

		createdAt, updatedAt := normalizeTimes(node.CreatedAt, node.UpdatedAt)
		desc := node.Content
		if len([]rune(desc)) > 500 {
			desc = string([]rune(desc)[:499]) + "…"
		}

		stepsTotal := len(node.Steps)
		stepsCompleted := 0
		for _, s := range node.Steps {
			if s.Completed {
				stepsCompleted++
			}
		}

		feat := &dbpkg.Feature{
			ID:             node.ID,
			Type:           mapNodeType(node.Type),
			Title:          node.Title,
			Description:    desc,
			Status:         normalizeStatus(string(node.Status)),
			Priority:       string(node.Priority),
			AssignedTo:     node.AgentAssigned,
			TrackID:        node.TrackID,
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
			StepsTotal:     stepsTotal,
			StepsCompleted: stepsCompleted,
		}

		if upsertErr := dbpkg.UpsertFeature(database, feat); upsertErr != nil {
			errCount++
			continue
		}
		validIDs[node.ID] = true
		upserted++
	}
	return total, upserted, errCount
}

// normalizeTimes returns sensible defaults for zero-value timestamps.
func normalizeTimes(createdAt, updatedAt time.Time) (time.Time, time.Time) {
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	return createdAt, updatedAt
}

// purgeStaleEntries removes features, tracks, and graph_edges whose IDs are no
// longer backed by an HTML file. Returns counts of purged features+tracks and edges.
func purgeStaleEntries(database *sql.DB, validIDs map[string]bool) (int, int) {
	staleFeatureIDs := collectStaleIDs(database, "SELECT id FROM features", validIDs)
	purged := deleteByIDs(database, "DELETE FROM features WHERE id = ?", staleFeatureIDs)

	// Purge stale tracks (HTML files deleted from .htmlgraph/tracks/).
	staleTrackIDs := collectStaleIDs(database, "SELECT id FROM tracks", validIDs)
	purged += deleteByIDs(database, "DELETE FROM tracks WHERE id = ?", staleTrackIDs)

	// Purge edges that reference deleted node IDs (either endpoint).
	staleEdgeIDs := collectStaleEdgeIDs(database, validIDs)
	edgesPurged := deleteByIDs(database, "DELETE FROM graph_edges WHERE edge_id = ?", staleEdgeIDs)

	return purged, edgesPurged
}

// collectStaleIDs queries all IDs from a single-column SELECT and returns those
// not present in validIDs.
func collectStaleIDs(database *sql.DB, query string, validIDs map[string]bool) []string {
	rows, err := database.Query(query)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var stale []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil && !validIDs[id] {
			stale = append(stale, id)
		}
	}
	return stale
}

// collectStaleEdgeIDs returns edge_ids where either endpoint (from_node_id or
// to_node_id) refers to a node no longer backed by an HTML file.
func collectStaleEdgeIDs(database *sql.DB, validIDs map[string]bool) []string {
	rows, err := database.Query("SELECT edge_id, from_node_id, to_node_id FROM graph_edges")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var stale []string
	for rows.Next() {
		var edgeID, fromID, toID string
		if rows.Scan(&edgeID, &fromID, &toID) == nil {
			if !validIDs[fromID] || !validIDs[toID] {
				stale = append(stale, edgeID)
			}
		}
	}
	return stale
}

// deleteByIDs executes a parameterised DELETE for each ID and returns the count
// of successful deletions.
func deleteByIDs(database *sql.DB, query string, ids []string) int {
	count := 0
	for _, id := range ids {
		if _, err := database.Exec(query, id); err == nil {
			count++
		}
	}
	return count
}

// normalizeStatus maps HTML statuses to the features table CHECK constraint values.
// features table allows: todo, in-progress, blocked, done, active, ended, stale
func normalizeStatus(status string) string {
	switch status {
	case "todo", "in-progress", "blocked", "done", "active", "ended", "stale":
		return status
	case "completed":
		return "done"
	case "in_progress":
		return "in-progress"
	case "archived", "cancelled":
		return "ended"
	case "pending", "identified":
		return "todo"
	default:
		return "todo"
	}
}

// mapNodeType converts HTML node types to the features table CHECK constraint values.
// features table allows: feature, bug, spike, chore, epic, task
func mapNodeType(nodeType string) string {
	switch nodeType {
	case "feature":
		return "feature"
	case "bug":
		return "bug"
	case "spike":
		return "spike"
	case "track":
		return "epic"
	case "chore":
		return "chore"
	case "plan", "spec":
		return "task"
	default:
		return "feature"
	}
}
