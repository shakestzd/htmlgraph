package main

import (
	"fmt"
	"os"
	"time"

	"github.com/shakestzd/htmlgraph/internal/hooks"
	"github.com/shakestzd/htmlgraph/internal/models"
	"github.com/shakestzd/htmlgraph/internal/workitem"
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

// warnMissingFields prints warnings for missing recommended fields per type.
// Returns an error if required fields are missing for features and bugs.
func warnMissingFields(typeName string, o *wiCreateOpts) error {
	// Track: warn about no --track (spikes and tracks exempt).
	if o.trackID == "" && typeName != "track" && typeName != "spike" {
		fmt.Fprintf(os.Stderr, "Warning: no track specified. Use --track <trk-id> to link this %s to an initiative.\nRun 'htmlgraph track list' to see existing tracks.\n", typeName)
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
	// spike: no requirements. track/plan: steps are usually added later.
	}
	return nil
}
