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

	dirs := []string{"features", "bugs", "tracks", "spikes"}
	var total, upserted, errCount int
	validIDs := make(map[string]bool)

	for _, dir := range dirs {
		pattern := filepath.Join(htmlgraphDir, dir, "*.html")
		files, _ := filepath.Glob(pattern)

		for _, f := range files {
			total++
			node, parseErr := htmlparse.ParseFile(f)
			if parseErr != nil {
				errCount++
				continue
			}

			stepsTotal := len(node.Steps)
			stepsCompleted := 0
			for _, s := range node.Steps {
				if s.Completed {
					stepsCompleted++
				}
			}

			desc := node.Content
			if len([]rune(desc)) > 500 {
				desc = string([]rune(desc)[:499]) + "…"
			}

			createdAt := node.CreatedAt
			if createdAt.IsZero() {
				createdAt = time.Now()
			}
			updatedAt := node.UpdatedAt
			if updatedAt.IsZero() {
				updatedAt = createdAt
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
	}

	purged, edgesPurged := purgeStaleEntries(database, validIDs)

	fmt.Printf("Reindexed: %d upserted, %d errors (of %d HTML files)\n",
		upserted, errCount, total)
	if purged > 0 || edgesPurged > 0 {
		fmt.Printf("Purged: %d stale features, %d stale edges\n", purged, edgesPurged)
	}
	return nil
}

// purgeStaleEntries removes features and graph_edges whose IDs are no longer
// backed by an HTML file. Returns counts of purged features and edges.
func purgeStaleEntries(database *sql.DB, validIDs map[string]bool) (int, int) {
	staleFeatureIDs := collectStaleIDs(database, "SELECT id FROM features", validIDs)
	purged := deleteByIDs(database, "DELETE FROM features WHERE id = ?", staleFeatureIDs)

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
