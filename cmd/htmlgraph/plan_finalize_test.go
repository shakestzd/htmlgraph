package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shakestzd/htmlgraph/internal/workitem"
)

func setupFinalizeProject(t *testing.T) (*workitem.Project, string) {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"plans", "features", "tracks"} {
		os.MkdirAll(filepath.Join(dir, sub), 0o755)
	}
	p, err := workitem.Open(dir, "test-agent")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Close() })
	return p, dir
}

func TestPlanFinalize_CreatesTrackAndFeatures(t *testing.T) {
	p, dir := setupFinalizeProject(t)

	// Create a plan with 2 slices (steps).
	node, err := p.Plans.Create("Test Plan")
	if err != nil {
		t.Fatal(err)
	}
	edit := p.Plans.Edit(node.ID)
	edit = edit.AddStep("Error handling")
	edit = edit.AddStep("Token validation")
	if err := edit.Save(); err != nil {
		t.Fatal(err)
	}

	result, err := executePlanFinalize(p, dir, node.ID)
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}

	if result.TrackID == "" {
		t.Fatal("no track created")
	}
	if !strings.HasPrefix(result.TrackID, "trk-") {
		t.Errorf("track ID %q missing trk- prefix", result.TrackID)
	}
	if len(result.FeatureIDs) != 2 {
		t.Errorf("features = %d, want 2", len(result.FeatureIDs))
	}
	for _, fid := range result.FeatureIDs {
		if !strings.HasPrefix(fid, "feat-") {
			t.Errorf("feature ID %q missing feat- prefix", fid)
		}
	}
}

func TestPlanFinalize_Idempotent(t *testing.T) {
	p, dir := setupFinalizeProject(t)

	node, err := p.Plans.Create("Idempotent Plan")
	if err != nil {
		t.Fatal(err)
	}
	edit := p.Plans.Edit(node.ID)
	edit = edit.AddStep("Slice A")
	if err := edit.Save(); err != nil {
		t.Fatal(err)
	}

	// First finalize.
	result1, err := executePlanFinalize(p, dir, node.ID)
	if err != nil {
		t.Fatalf("first finalize: %v", err)
	}

	// Second finalize should detect already-finalized.
	result2, err := executePlanFinalize(p, dir, node.ID)
	if err != nil {
		t.Fatalf("second finalize: %v", err)
	}

	if !result2.AlreadyFinalized {
		t.Error("second finalize should report AlreadyFinalized=true")
	}
	if result2.TrackID != result1.TrackID {
		t.Errorf("tracks differ: %q vs %q", result2.TrackID, result1.TrackID)
	}
	if len(result2.FeatureIDs) != len(result1.FeatureIDs) {
		t.Errorf("feature counts differ: %d vs %d", len(result2.FeatureIDs), len(result1.FeatureIDs))
	}
}

func TestPlanFinalize_EmptyPlan(t *testing.T) {
	p, dir := setupFinalizeProject(t)

	node, err := p.Plans.Create("Empty Plan")
	if err != nil {
		t.Fatal(err)
	}

	result, err := executePlanFinalize(p, dir, node.ID)
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}

	if result.TrackID == "" {
		t.Error("track should be created even with no slices")
	}
	if len(result.FeatureIDs) != 0 {
		t.Errorf("features = %d, want 0", len(result.FeatureIDs))
	}
}
