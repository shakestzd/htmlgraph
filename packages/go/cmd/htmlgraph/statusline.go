package main

import (
	"database/sql"
	"fmt"
	"path/filepath"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/hooks"
	"github.com/shakestzd/htmlgraph/internal/workitem"
	"github.com/spf13/cobra"
)

func statuslineCmd() *cobra.Command {
	var sessionID string

	cmd := &cobra.Command{
		Use:   "statusline",
		Short: "Print the active work item for Claude Code status line",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatusline(sessionID)
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "Session ID to scope the active work item lookup")
	return cmd
}

func runStatusline(sessionID string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return nil
	}

	// If a session ID is provided, look up the session's active_feature_id from SQLite.
	if sessionID != "" {
		return statuslineFromSession(dir, sessionID)
	}

	// Fallback: scan HTML files for any in-progress item.
	return statuslineFromHTML(dir)
}

func statuslineFromSession(dir, sessionID string) error {
	database, err := dbpkg.Open(filepath.Join(dir, "htmlgraph.db"))
	if err != nil {
		return nil
	}
	defer database.Close()

	featureID := hooks.GetActiveFeatureID(database, sessionID)
	if featureID == "" {
		return nil
	}

	// Look up the title from the HTML file.
	p, err := workitem.Open(dir, "claude-code")
	if err != nil {
		// We have the ID but can't get title — still show the ID.
		fmt.Println(featureID)
		return nil
	}
	defer p.Close()

	// Find the feature node.
	var featureType string
	var featureTitle string
	for _, typeName := range []string{"bug", "feature", "spike"} {
		col := collectionFor(p, typeName)
		node, err := col.Get(featureID)
		if err == nil && node != nil {
			if node.Status == "done" || node.Status == "completed" {
				return nil // Feature was completed — don't show it
			}
			featureType = typeName
			featureTitle = node.Title
			break
		}
	}
	if featureTitle == "" {
		return nil
	}

	// Check if feature belongs to a track.
	trackLine := resolveTrackContext(database, featureID)

	if trackLine != "" {
		fmt.Printf("%s → %s %s\n", trackLine, iconFor(featureType), truncate(featureTitle, 25))
	} else {
		fmt.Printf("%s %s\n", iconFor(featureType), truncate(featureTitle, 30))
	}
	return nil
}

// resolveTrackContext returns a formatted track summary if the feature belongs to a track.
// Format: "track_icon Track Title [done/total]"
// Returns empty string if no track.
func resolveTrackContext(database *sql.DB, featureID string) string {
	// Check track_id in SQLite first (fast path).
	var trackID sql.NullString
	database.QueryRow("SELECT track_id FROM features WHERE id = ?", featureID).Scan(&trackID) //nolint:errcheck

	if !trackID.Valid || trackID.String == "" {
		// Check graph_edges for part_of relationship.
		database.QueryRow(`
			SELECT to_node_id FROM graph_edges
			WHERE from_node_id = ? AND relationship_type = 'part_of'
			AND to_node_id LIKE 'trk-%'
			LIMIT 1`, featureID).Scan(&trackID) //nolint:errcheck
	}

	if !trackID.Valid || trackID.String == "" {
		return ""
	}

	// Get track title.
	var trackTitle sql.NullString
	database.QueryRow("SELECT title FROM tracks WHERE id = ?", trackID.String).Scan(&trackTitle) //nolint:errcheck

	// Count done/total features via track_id column.
	var total, done int
	database.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END), 0)
		FROM features WHERE track_id = ?`, trackID.String).Scan(&total, &done) //nolint:errcheck

	// Also count features linked only via contains edges (not in track_id column).
	var edgeTotal, edgeDone int
	database.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(CASE WHEN f.status = 'done' THEN 1 ELSE 0 END), 0)
		FROM graph_edges ge
		JOIN features f ON ge.to_node_id = f.id
		WHERE ge.from_node_id = ? AND ge.relationship_type = 'contains'
		AND (f.track_id IS NULL OR f.track_id != ?)`,
		trackID.String, trackID.String).Scan(&edgeTotal, &edgeDone) //nolint:errcheck

	total += edgeTotal
	done += edgeDone

	title := trackID.String
	if trackTitle.Valid && trackTitle.String != "" {
		title = truncate(trackTitle.String, 25)
	}

	return fmt.Sprintf("%s %s [%d/%d]", iconFor("track"), title, done, total)
}

func statuslineFromHTML(dir string) error {
	p, err := workitem.Open(dir, "claude-code")
	if err != nil {
		return nil
	}
	defer p.Close()

	for _, typeName := range []string{"bug", "feature"} {
		col := collectionFor(p, typeName)
		nodes, err := col.List()
		if err != nil {
			continue
		}
		for _, n := range nodes {
			if n.Status == "in-progress" {
				fmt.Printf("%s %s\n", iconFor(typeName), truncate(n.Title, 30))
				return nil
			}
		}
	}

	return nil
}

func iconFor(typeName string) string {
	switch typeName {
	case "bug":
		return "\uf188" //  bug
	case "feature":
		return "\uf0eb" //  lightbulb
	case "spike":
		return "\uf0e7" //  bolt
	case "track":
		return "\uf018" //  road
	default:
		return "\uf111" //  circle
	}
}

func inferType(id string) string {
	if len(id) < 4 {
		return "feature"
	}
	switch id[:4] {
	case "bug-":
		return "bug"
	case "feat":
		return "feature"
	case "spk-", "spik":
		return "spike"
	case "trk-":
		return "track"
	default:
		return "feature"
	}
}
