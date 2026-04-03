package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/hooks"
	"github.com/shakestzd/htmlgraph/internal/models"
	"github.com/shakestzd/htmlgraph/internal/workitem"
	"github.com/shakestzd/htmlgraph/internal/workowners"
)

// detectActiveFeature returns the active feature ID from the session DB, or "".
func detectActiveFeature(p *workitem.Project, htmlgraphDir string) string {
	if p.DB == nil {
		return ""
	}
	sessionID := hooks.EnvSessionID("")
	if sessionID == "" {
		return ""
	}
	return hooks.GetActiveFeatureID(p.DB, sessionID)
}

// autoCausedByEdge creates a caused_by edge from a bug to the active feature.
func autoCausedByEdge(p *workitem.Project, bugID, featureID string) {
	edge := models.Edge{
		TargetID:     featureID,
		Relationship: models.RelCausedBy,
		Title:        featureID,
		Since:        time.Now().UTC(),
	}
	_, _ = p.Bugs.AddEdge(bugID, edge)
}

// autoImplementedInEdge creates an implemented_in edge from a work item to
// a session. Idempotent: skips if edge already exists. Non-fatal on error.
func autoImplementedInEdge(col *workitem.Collection, itemID, sessionID string) {
	node, err := col.Get(itemID)
	if err != nil {
		return
	}
	// Check for existing implemented_in edge to this session.
	for _, e := range node.Edges[string(models.RelImplementedIn)] {
		if e.TargetID == sessionID {
			return // already linked
		}
	}
	edge := models.Edge{
		TargetID:     sessionID,
		Relationship: models.RelImplementedIn,
		Title:        "session " + sessionID,
		Since:        time.Now().UTC(),
	}
	_, _ = col.AddEdge(itemID, edge)
}

// autoTrackEdges creates bidirectional part_of/contains edges between a work
// item and its track. Errors are non-fatal (warn-not-block).
func autoTrackEdges(p *workitem.Project, itemID, typeName, trackID, itemTitle string) error {
	now := time.Now().UTC()

	// item → track (part_of)
	col := collectionFor(p, typeName)
	partOf := models.Edge{
		TargetID:     trackID,
		Relationship: models.RelPartOf,
		Title:        trackID,
		Since:        now,
	}
	if _, err := col.AddEdge(itemID, partOf); err != nil {
		return fmt.Errorf("part_of: %w", err)
	}

	// track → item (contains)
	contains := models.Edge{
		TargetID:     itemID,
		Relationship: models.RelContains,
		Title:        itemTitle,
		Since:        now,
	}
	if _, err := p.Tracks.AddEdge(trackID, contains); err != nil {
		return fmt.Errorf("contains: %w", err)
	}

	return nil
}

// warnMissingFields validates required and recommended fields per work item type.
// Features and bugs REQUIRE --track and --description. Spikes, tracks, plans,
// and specs are exempt from the track requirement.
func warnMissingFields(typeName string, o *wiCreateOpts) error {
	// Features and bugs require a track to link to an initiative.
	if o.trackID == "" && (typeName == "feature" || typeName == "bug") {
		msg := fmt.Sprintf("%s requires --track <trk-id> to link to an initiative.\nRun 'htmlgraph track list' to see existing tracks.\nTo create a new track: htmlgraph track create \"Track Title\"", typeName)

		// For bugs with --files, try to suggest the track via file ownership.
		if typeName == "bug" && o.files != "" {
			if suggestion := suggestTrackFromFiles(o.files); suggestion != "" {
				msg += "\n\nSuggested: " + suggestion
			}
		}

		return fmt.Errorf("%s", msg)
	}

	switch typeName {
	case "bug", "feature":
		if o.description == "" {
			return fmt.Errorf("%s requires --description (captures context for future sessions)\nExample: htmlgraph %s create \"title\" --description \"root cause and context\"", typeName, typeName)
		}
	case "spec":
		if o.description == "" {
			fmt.Fprintf(os.Stderr, "Warning: spec created without --description.\n")
		}
	}
	return nil
}

// suggestTrackFromFiles resolves file ownership for the first affected file.
// Checks WORKOWNERS static map first, then falls back to DB heuristic.
// Returns a suggestion string like "--track trk-abc (owns cmd/foo.go)".
func suggestTrackFromFiles(files string) string {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return ""
	}

	// Check WORKOWNERS static map first.
	wf, _ := workowners.Parse(filepath.Join(dir, "WORKOWNERS"))
	for _, f := range strings.Split(files, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if ownerID := wf.Resolve(f); ownerID != "" {
			return fmt.Sprintf("--track %s (WORKOWNERS: %s)", ownerID, f)
		}
	}

	// Fall back to DB heuristic.
	database := openTrackDB(dir)
	if database == nil {
		return ""
	}
	defer database.Close()

	for _, f := range strings.Split(files, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		owner := dbpkg.ResolveFileOwner(database, f)
		if owner == nil {
			continue
		}
		if owner.TrackID != "" {
			return fmt.Sprintf("--track %s (%s owns %s)", owner.TrackID, owner.Title, f)
		}
		return fmt.Sprintf("feature %s (%s) owns %s — find its track with: htmlgraph feature show %s",
			owner.FeatureID, owner.Title, f, owner.FeatureID)
	}
	return ""
}
