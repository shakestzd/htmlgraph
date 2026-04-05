package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/models"
	"github.com/shakestzd/htmlgraph/internal/planyaml"
	"github.com/shakestzd/htmlgraph/internal/workitem"
	"github.com/spf13/cobra"
)

// planFinalizeYAMLCmd creates track + features from approved slices in a YAML plan.
func planFinalizeYAMLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "finalize-yaml <plan-id>",
		Short: "Create track and features from approved YAML plan slices",
		Long: `Read a YAML plan + SQLite feedback, create a track and features for
approved slices, wire dependency edges. Updates YAML status to finalized.

Example:
  htmlgraph plan finalize-yaml plan-a1b2c3d4`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runFinalizeYAML(args[0])
		},
	}
}

func runFinalizeYAML(planID string) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	planPath := filepath.Join(htmlgraphDir, "plans", planID+".yaml")
	plan, err := planyaml.Load(planPath)
	if err != nil {
		return fmt.Errorf("load plan: %w", err)
	}

	// Read approvals from SQLite.
	dbPath := filepath.Join(htmlgraphDir, "htmlgraph.db")
	db, err := dbpkg.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	approvals := map[string]bool{}
	rows, err := db.Query(
		"SELECT section, value FROM plan_feedback WHERE plan_id = ? AND action = 'approve'",
		planID,
	)
	if err != nil {
		return fmt.Errorf("query approvals: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var section, value string
		rows.Scan(&section, &value)
		approvals[section] = strings.EqualFold(value, "true")
	}

	// Open project for work item creation.
	p, err := workitem.Open(htmlgraphDir, agentForClaim())
	if err != nil {
		return fmt.Errorf("open project: %w", err)
	}
	defer p.Close()

	// Create track from plan title.
	track, err := p.Tracks.Create(plan.Meta.Title)
	if err != nil {
		return fmt.Errorf("create track: %w", err)
	}
	fmt.Printf("Created track: %s  %s\n", track.ID, track.Title)

	// Create features for approved slices.
	numToFeatID := map[int]string{}
	for _, s := range plan.Slices {
		approved := approvals[fmt.Sprintf("slice-%d", s.Num)]
		if !approved {
			fmt.Printf("  Skipped slice %d: %s (not approved)\n", s.Num, s.Title)
			continue
		}
		feat, err := p.Features.Create(s.Title,
			workitem.FeatWithTrack(track.ID),
			workitem.FeatWithContent(s.What),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error creating feature for slice %d: %v\n", s.Num, err)
			continue
		}
		numToFeatID[s.Num] = feat.ID
		fmt.Printf("  Created feature: %s  %s\n", feat.ID, feat.Title)
	}

	// Wire blocked_by edges from slice deps.
	for _, s := range plan.Slices {
		featID, ok := numToFeatID[s.Num]
		if !ok {
			continue
		}
		for _, depNum := range s.Deps {
			depFeatID, ok := numToFeatID[depNum]
			if !ok {
				continue
			}
			p.Features.AddEdge(featID, models.Edge{
				TargetID:     depFeatID,
				Relationship: "blocked_by",
			})
		}
	}

	// Update YAML status.
	plan.Meta.Status = "finalized"
	plan.Meta.TrackID = track.ID
	for i := range plan.Slices {
		plan.Slices[i].Approved = approvals[fmt.Sprintf("slice-%d", plan.Slices[i].Num)]
	}
	if err := planyaml.Save(planPath, plan); err != nil {
		return fmt.Errorf("save plan: %w", err)
	}

	fmt.Printf("\nFinalized: %s\n", planID)
	fmt.Printf("Track: %s  %s\n", track.ID, track.Title)
	fmt.Printf("Features: %d created, %d skipped\n", len(numToFeatID), len(plan.Slices)-len(numToFeatID))
	if cmd := buildExecuteCmd(track.ID); cmd != "" {
		fmt.Printf("\nExecute:\n  %s\n", cmd)
	}
	return nil
}
