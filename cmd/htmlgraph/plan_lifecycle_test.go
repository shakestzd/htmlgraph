package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shakestzd/htmlgraph/internal/workitem"
)

// TestPlanFirstLifecycle exercises the complete plan-first pipeline:
// create → add-slice → critique → approve → finalize → verify work items.
func TestPlanFirstLifecycle(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"plans", "features", "tracks"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Step 1: Create plan from topic.
	planID, err := createPlanFromTopic(dir, "Auth Middleware Rewrite", "Rewrite auth for compliance requirements")
	if err != nil {
		t.Fatalf("createPlanFromTopic: %v", err)
	}
	planPath := filepath.Join(dir, "plans", planID+".html")
	if _, err := os.Stat(planPath); err != nil {
		t.Fatalf("plan file not created: %v", err)
	}

	// Step 2: Add slices.
	for _, title := range []string{"Error handling layer", "Token validation", "Migration script"} {
		if err := addSliceToPlan(dir, planID, title); err != nil {
			t.Fatalf("addSliceToPlan(%s): %v", title, err)
		}
	}

	// Verify slices exist.
	data, _ := os.ReadFile(planPath)
	html := string(data)
	for i := 1; i <= 3; i++ {
		marker := `data-slice="` + string(rune('0'+i)) + `"`
		if !strings.Contains(html, marker) {
			t.Errorf("plan missing slice %d", i)
		}
	}

	// Step 3: Critique.
	critique, err := extractCritiqueData(dir, planID)
	if err != nil {
		t.Fatalf("extractCritiqueData: %v", err)
	}
	if !critique.CritiqueWarranted {
		t.Error("3 slices should warrant critique")
	}
	if critique.Complexity != "medium" {
		t.Errorf("complexity = %q, want medium", critique.Complexity)
	}
	if len(critique.Slices) != 3 {
		t.Errorf("critique slices = %d, want 3", len(critique.Slices))
	}

	// Step 4: Approve all slices (simulate user clicking approve checkboxes).
	data, _ = os.ReadFile(planPath)
	html = strings.ReplaceAll(string(data), `data-action="approve"`, `data-action="approve" checked`)
	if err := os.WriteFile(planPath, []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}

	// Step 5: Finalize — creates track + features.
	p, err := workitem.Open(dir, "test-agent")
	if err != nil {
		t.Fatalf("workitem.Open: %v", err)
	}
	defer p.Close()

	// Register plan in the collection so finalize can link it.
	_, _ = p.Plans.Create("Auth Middleware Rewrite")

	result, err := executePlanFinalize(p, dir, planID)
	if err != nil {
		t.Fatalf("executePlanFinalize: %v", err)
	}

	// Verify track created.
	if result.TrackID == "" {
		t.Fatal("no track created")
	}
	if !strings.HasPrefix(result.TrackID, "trk-") {
		t.Errorf("track ID %q doesn't have trk- prefix", result.TrackID)
	}

	// Verify 3 features created (one per approved slice).
	if len(result.FeatureIDs) != 3 {
		t.Errorf("features = %d, want 3", len(result.FeatureIDs))
	}
	for _, fid := range result.FeatureIDs {
		if !strings.HasPrefix(fid, "feat-") {
			t.Errorf("feature ID %q doesn't have feat- prefix", fid)
		}
		// Verify feature file exists.
		featPath := filepath.Join(dir, "features", fid+".html")
		if _, err := os.Stat(featPath); err != nil {
			t.Errorf("feature file missing: %s", featPath)
		}
	}

	// Verify track file exists.
	trackPath := filepath.Join(dir, "tracks", result.TrackID+".html")
	if _, err := os.Stat(trackPath); err != nil {
		t.Errorf("track file missing: %s", trackPath)
	}

	// Verify plan status is now finalized.
	data, _ = os.ReadFile(planPath)
	if !strings.Contains(string(data), `data-status="finalized"`) {
		t.Error("plan not marked as finalized")
	}

	// Step 6: Idempotent re-finalize.
	result2, err := executePlanFinalize(p, dir, planID)
	if err != nil {
		t.Fatalf("re-finalize: %v", err)
	}
	if !result2.AlreadyFinalized {
		t.Error("re-finalize should report AlreadyFinalized=true")
	}
	if result2.TrackID != result.TrackID {
		t.Errorf("re-finalize track %q != original %q", result2.TrackID, result.TrackID)
	}
}

// TestPlanLifecycle_NoApprovedSlices verifies finalize with no approvals
// creates a track but zero features.
func TestPlanLifecycle_NoApprovedSlices(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"plans", "features", "tracks"} {
		os.MkdirAll(filepath.Join(dir, sub), 0o755)
	}

	planID, err := createPlanFromTopic(dir, "Empty Plan", "nothing approved")
	if err != nil {
		t.Fatal(err)
	}
	if err := addSliceToPlan(dir, planID, "Unapproved Slice"); err != nil {
		t.Fatal(err)
	}

	p, err := workitem.Open(dir, "test-agent")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	result, err := executePlanFinalize(p, dir, planID)
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}
	if result.TrackID == "" {
		t.Error("track should be created even with no approved slices")
	}
	if len(result.FeatureIDs) != 0 {
		t.Errorf("features = %d, want 0 (no slices approved)", len(result.FeatureIDs))
	}
}
