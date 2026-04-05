package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/shakestzd/htmlgraph/internal/models"
	"github.com/shakestzd/htmlgraph/internal/workitem"
	"github.com/spf13/cobra"
)

// finalizeResult holds the output of a plan finalize operation.
type finalizeResult struct {
	TrackID          string
	FeatureIDs       []string
	AlreadyFinalized bool
	ExecuteCmd       string // CLI command to start working on the track
}

// planFinalizeCmd creates a cobra command for plan finalize.
func planFinalizeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "finalize <plan-id>",
		Short: "Generate a work item graph from a plan",
		Long: `Read a plan's steps (slices) and generate the work item graph:
a track, features per slice, and edges for dependencies.

Idempotent: re-running on an already-finalized plan is a no-op.

Example:
  htmlgraph plan finalize plan-a1b2c3d4`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			htmlgraphDir, err := findHtmlgraphDir()
			if err != nil {
				return err
			}
			p, err := workitem.Open(htmlgraphDir, "claude-code")
			if err != nil {
				return fmt.Errorf("open project: %w", err)
			}
			defer p.Close()

			result, err := executePlanFinalize(p, htmlgraphDir, args[0])
			if err != nil {
				return err
			}

			if result.AlreadyFinalized {
				fmt.Printf("Plan %s is already finalized", args[0])
				if result.TrackID != "" {
					fmt.Printf(" (track: %s)", result.TrackID)
				}
				fmt.Println()
				if result.ExecuteCmd != "" {
					fmt.Printf("\nExecute:\n  %s\n", result.ExecuteCmd)
				}
				return nil
			}

			fmt.Printf("Created track: %s\n", result.TrackID)
			fmt.Printf("Created %d features\n", len(result.FeatureIDs))
			for _, fid := range result.FeatureIDs {
				fmt.Printf("  %s\n", fid)
			}
			if result.ExecuteCmd != "" {
				fmt.Printf("\nExecute:\n  %s\n", result.ExecuteCmd)
			}
			return nil
		},
	}
}

// executePlanFinalize reads a plan's steps, creates a track and features,
// wires edges, and marks the plan as finalized. Idempotent.
func executePlanFinalize(p *workitem.Project, htmlgraphDir, planID string) (*finalizeResult, error) {
	// Get the plan node.
	planNode, err := p.Plans.Get(planID)
	if err != nil {
		return nil, fmt.Errorf("plan %s not found: %w", planID, err)
	}

	// Idempotent: if already finalized, find existing track + features and return.
	if planNode.Status == "finalized" || planNode.Status == "done" {
		trackID := findTrackForPlan(p, planID)
		featureIDs := findFeaturesForTrack(p, trackID)
		return &finalizeResult{
			TrackID:          trackID,
			FeatureIDs:       featureIDs,
			AlreadyFinalized: true,
			ExecuteCmd:       buildExecuteCmd(trackID),
		}, nil
	}

	// Use plan steps as slices — all steps are treated as approved.
	slices := parsePlanStepsAsSlices(planNode)

	// Create track from the plan title.
	trackNode, err := p.Tracks.Create(planNode.Title,
		workitem.TrackWithStatus("in-progress"),
	)
	if err != nil {
		return nil, fmt.Errorf("create track: %w", err)
	}

	// Create features for each slice.
	var featureIDs []string
	for _, s := range slices {
		opts := []workitem.FeatureOption{
			workitem.FeatWithTrack(trackNode.ID),
		}
		if planNode.Content != "" {
			opts = append(opts, workitem.FeatWithContent(planNode.Content))
		}

		featNode, err := p.Features.Create(s.title, opts...)
		if err != nil {
			return nil, fmt.Errorf("create feature for slice %d: %w", s.num, err)
		}

		featureIDs = append(featureIDs, featNode.ID)

		// Wire bidirectional track <-> feature edges.
		if err := wireTrackEdges(p, featNode.ID, trackNode.ID, s.title); err != nil {
			return nil, fmt.Errorf("wire track edges for %s: %w", featNode.ID, err)
		}
	}

	// Link plan to track: plan implemented_in track.
	edge := models.Edge{
		TargetID:     trackNode.ID,
		Relationship: models.RelImplementedIn,
		Title:        trackNode.ID,
		Since:        time.Now().UTC(),
	}
	_, _ = p.Plans.AddEdge(planNode.ID, edge)

	// Mark plan as finalized.
	if _, err := p.Plans.Complete(planID); err != nil {
		// Best-effort: try updating status directly.
		edit := p.Plans.Edit(planID)
		edit = edit.SetStatus("finalized")
		_ = edit.Save()
	}

	return &finalizeResult{
		TrackID:    trackNode.ID,
		FeatureIDs: featureIDs,
		ExecuteCmd: buildExecuteCmd(trackNode.ID),
	}, nil
}

// planSlice holds metadata for a single slice parsed from the plan.
type planSlice struct {
	num     int
	name    string
	title   string
	depNums []int
}

// wireTrackEdges creates bidirectional part_of/contains edges between a
// feature and its track.
func wireTrackEdges(p *workitem.Project, featureID, trackID, featureTitle string) error {
	now := time.Now().UTC()

	// feature -> track (part_of)
	partOf := models.Edge{
		TargetID:     trackID,
		Relationship: models.RelPartOf,
		Title:        trackID,
		Since:        now,
	}
	if _, err := p.Features.AddEdge(featureID, partOf); err != nil {
		return fmt.Errorf("part_of: %w", err)
	}

	// track -> feature (contains)
	contains := models.Edge{
		TargetID:     featureID,
		Relationship: models.RelContains,
		Title:        featureTitle,
		Since:        now,
	}
	if _, err := p.Tracks.AddEdge(trackID, contains); err != nil {
		return fmt.Errorf("contains: %w", err)
	}

	return nil
}

// buildExecuteCmd returns the CLI command to start working on a finalized track.
func buildExecuteCmd(trackID string) string {
	if trackID == "" {
		return ""
	}
	return "htmlgraph yolo --track " + trackID
}

// findFeaturesForTrack returns feature IDs linked to a track via contains edges.
func findFeaturesForTrack(p *workitem.Project, trackID string) []string {
	if trackID == "" {
		return nil
	}
	node, err := p.Tracks.Get(trackID)
	if err != nil {
		return nil
	}
	var ids []string
	for _, edge := range node.Edges[string(models.RelContains)] {
		if strings.HasPrefix(edge.TargetID, "feat-") {
			ids = append(ids, edge.TargetID)
		}
	}
	return ids
}

// findTrackForPlan searches for an existing track linked to the plan via
// an implemented_in edge. Returns the track ID or empty string.
func findTrackForPlan(p *workitem.Project, planID string) string {
	node, err := p.Plans.Get(planID)
	if err != nil {
		return ""
	}
	for _, edge := range node.Edges[string(models.RelImplementedIn)] {
		if strings.HasPrefix(edge.TargetID, "trk-") {
			return edge.TargetID
		}
	}
	return ""
}
