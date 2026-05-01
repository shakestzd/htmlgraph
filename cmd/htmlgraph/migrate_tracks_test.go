package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/migrate"
	"github.com/shakestzd/htmlgraph/internal/models"
	"github.com/shakestzd/htmlgraph/internal/workitem"
)

// migrateTracksTestEnv builds a temp project directory and seeds:
//   - tracks: trk-old, trk-yolo, trk-plan
//   - features: featClear (yolo-dominant, currently on trk-old),
//     featAmbig (split between two tracks),
//     featOrphan (no feature_files),
//     featStable (already on its dominant track).
//
// Returns the .htmlgraph directory path and a project root for the test.
func migrateTracksTestEnv(t *testing.T) (hgDir, rulesPath string) {
	t.Helper()
	root := t.TempDir()
	hgDir = filepath.Join(root, ".htmlgraph")
	if err := os.MkdirAll(hgDir, 0o755); err != nil {
		t.Fatalf("mkdir .htmlgraph: %v", err)
	}

	// Force the DB to live inside the project for test isolation.
	dbPath := filepath.Join(hgDir, "htmlgraph.db")
	t.Setenv("HTMLGRAPH_DB_PATH", dbPath)

	// Open the project — workitem.Open will use HTMLGRAPH_DB_PATH and create
	// the DB file. We also use it to write canonical HTML so feature update
	// works end-to-end.
	p, err := workitem.Open(hgDir, "test-agent")
	if err != nil {
		t.Fatalf("workitem.Open: %v", err)
	}
	t.Cleanup(func() { p.Close() })

	tOld, err := p.Tracks.Create("Old Track")
	if err != nil {
		t.Fatal(err)
	}
	tYolo, err := p.Tracks.Create("Yolo Track")
	if err != nil {
		t.Fatal(err)
	}
	tPlan, err := p.Tracks.Create("Plan Track")
	if err != nil {
		t.Fatal(err)
	}

	featClear, err := p.Features.Create("Clear yolo feature", workitem.FeatWithTrack(tOld.ID))
	if err != nil {
		t.Fatal(err)
	}
	featAmbig, err := p.Features.Create("Ambiguous feature", workitem.FeatWithTrack(tOld.ID))
	if err != nil {
		t.Fatal(err)
	}
	featOrphan, err := p.Features.Create("Orphan feature", workitem.FeatWithTrack(tOld.ID))
	if err != nil {
		t.Fatal(err)
	}
	featStable, err := p.Features.Create("Already correct", workitem.FeatWithTrack(tYolo.ID))
	if err != nil {
		t.Fatal(err)
	}

	// Seed feature_files in the SQLite read index. We use the same DB the
	// project uses (env var override), so direct UpsertFeatureFile is fine.
	now := time.Now().UTC()
	_ = now
	upsert := func(featureID, path string) {
		if err := db.UpsertFeatureFile(p.DB, &models.FeatureFile{
			ID: "ff-" + featureID + "-" + path, FeatureID: featureID, FilePath: path, Operation: "edit",
		}); err != nil {
			t.Fatalf("upsert feature_file: %v", err)
		}
	}

	// featClear: 4 yolo files, 0 plan
	for _, f := range []string{"cmd/htmlgraph/yolo.go", "cmd/htmlgraph/tmux.go", "cmd/htmlgraph/budget.go", "internal/worktree/manager.go"} {
		upsert(featClear.ID, f)
	}
	// featAmbig: 2 plan, 2 yolo (50/50 split)
	for _, f := range []string{"cmd/htmlgraph/plan_create.go", "cmd/htmlgraph/plan_show.go", "cmd/htmlgraph/yolo.go", "cmd/htmlgraph/tmux.go"} {
		upsert(featAmbig.ID, f)
	}
	// featOrphan: zero feature_files
	_ = featOrphan
	// featStable: 3 yolo files; current track is already trk-yolo.
	for _, f := range []string{"cmd/htmlgraph/yolo.go", "cmd/htmlgraph/tmux.go", "cmd/htmlgraph/launch_run.go"} {
		upsert(featStable.ID, f)
	}

	// Save IDs for later assertions.
	t.Setenv("MTT_FEAT_CLEAR", featClear.ID)
	t.Setenv("MTT_FEAT_AMBIG", featAmbig.ID)
	t.Setenv("MTT_FEAT_ORPHAN", featOrphan.ID)
	t.Setenv("MTT_FEAT_STABLE", featStable.ID)
	t.Setenv("MTT_TRK_OLD", tOld.ID)
	t.Setenv("MTT_TRK_YOLO", tYolo.ID)
	t.Setenv("MTT_TRK_PLAN", tPlan.ID)

	// Write a rules file using the actual track IDs we just created.
	rulesPath = filepath.Join(root, "rules.yaml")
	rulesYAML := "rules:\n" +
		"  - { glob: \"cmd/htmlgraph/yolo.go\",        track_id: \"" + tYolo.ID + "\", priority: 110 }\n" +
		"  - { glob: \"cmd/htmlgraph/tmux.go\",        track_id: \"" + tYolo.ID + "\", priority: 110 }\n" +
		"  - { glob: \"cmd/htmlgraph/budget.go\",      track_id: \"" + tYolo.ID + "\", priority: 110 }\n" +
		"  - { glob: \"cmd/htmlgraph/launch_run.go\",  track_id: \"" + tYolo.ID + "\", priority: 110 }\n" +
		"  - { glob: \"internal/worktree/**\",         track_id: \"" + tYolo.ID + "\", priority: 100 }\n" +
		"  - { glob: \"cmd/htmlgraph/plan_*.go\",      track_id: \"" + tPlan.ID + "\", priority: 100 }\n"
	if err := os.WriteFile(rulesPath, []byte(rulesYAML), 0o644); err != nil {
		t.Fatalf("write rules: %v", err)
	}

	return hgDir, rulesPath
}

func TestMigrateTracksDryRunOutput(t *testing.T) {
	hgDir, rulesPath := migrateTracksTestEnv(t)

	var buf bytes.Buffer
	opts := migrateTracksOpts{
		rulesPath: rulesPath,
		dryRun:    true,
		types:     "features",
		threshold: 0.6,
		format:    "text",
	}
	if err := runMigrateTracks(context.Background(), hgDir, opts, &buf); err != nil {
		t.Fatalf("runMigrateTracks: %v", err)
	}

	out := buf.String()
	// Headline summary line should be present.
	if !strings.Contains(out, "features classified:") {
		t.Errorf("missing summary line:\n%s", out)
	}
	// Confident move should be there.
	if !strings.Contains(out, "confident") {
		t.Errorf("expected at least one 'confident' decision:\n%s", out)
	}
	// Ambiguous label should appear (50/50 case is below 0.6 threshold).
	if !strings.Contains(strings.ToLower(out), "ambiguous") {
		t.Errorf("expected an ambiguous decision in output:\n%s", out)
	}

	// Dry-run must NOT have written a manifest.
	matches, _ := filepath.Glob(filepath.Join(hgDir, "migrations", "track-backfill-*.json"))
	if len(matches) != 0 {
		t.Errorf("dry-run should not write manifests, found: %v", matches)
	}
}

func TestMigrateTracksWriteCreatesManifest(t *testing.T) {
	hgDir, rulesPath := migrateTracksTestEnv(t)
	featClear := os.Getenv("MTT_FEAT_CLEAR")
	trkYolo := os.Getenv("MTT_TRK_YOLO")

	var buf bytes.Buffer
	opts := migrateTracksOpts{
		rulesPath: rulesPath,
		write:     true,
		types:     "features",
		threshold: 0.6,
		format:    "text",
	}
	if err := runMigrateTracks(context.Background(), hgDir, opts, &buf); err != nil {
		t.Fatalf("runMigrateTracks: %v", err)
	}

	// Manifest written?
	matches, _ := filepath.Glob(filepath.Join(hgDir, "migrations", "track-backfill-*.json"))
	if len(matches) != 1 {
		t.Fatalf("expected 1 manifest file, got %d: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest struct {
		Decisions []migrate.Decision `json:"decisions"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	// Confirm featClear was moved.
	moved := false
	for _, d := range manifest.Decisions {
		if d.FeatureID == featClear && d.ProposedTrack == trkYolo {
			moved = true
			break
		}
	}
	if !moved {
		t.Fatalf("featClear (%s) → %s not recorded in manifest:\n%s", featClear, trkYolo, data)
	}

	// Confirm the canonical store reflects the new track for featClear.
	p, err := workitem.Open(hgDir, "test-agent")
	if err != nil {
		t.Fatalf("workitem.Open after write: %v", err)
	}
	defer p.Close()
	feat, err := p.Features.Get(featClear)
	if err != nil {
		t.Fatalf("Features.Get: %v", err)
	}
	if feat.TrackID != trkYolo {
		t.Errorf("featClear track_id = %q, want %q", feat.TrackID, trkYolo)
	}
}

func TestMigrateTracksWriteRefusesOverwrite(t *testing.T) {
	hgDir, rulesPath := migrateTracksTestEnv(t)

	// Pre-create a manifest so the second write call collides.
	migDir := filepath.Join(hgDir, "migrations")
	if err := os.MkdirAll(migDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(migDir, "track-backfill-1234567890.json")
	if err := os.WriteFile(existing, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	opts := migrateTracksOpts{
		rulesPath: rulesPath,
		write:     true,
		types:     "features",
		threshold: 0.6,
		format:    "text",
		// force is FALSE
	}
	err := runMigrateTracks(context.Background(), hgDir, opts, &buf)
	if err == nil {
		t.Fatalf("expected error on existing manifest without --force")
	}
	if !strings.Contains(err.Error(), "manifest") {
		t.Errorf("error = %q, want a manifest-collision message", err.Error())
	}

	// With --force it should succeed.
	opts.force = true
	buf.Reset()
	if err := runMigrateTracks(context.Background(), hgDir, opts, &buf); err != nil {
		t.Fatalf("runMigrateTracks --force: %v", err)
	}
}

func TestMigrateTracksJSONFormat(t *testing.T) {
	hgDir, rulesPath := migrateTracksTestEnv(t)

	var buf bytes.Buffer
	opts := migrateTracksOpts{
		rulesPath: rulesPath,
		dryRun:    true,
		types:     "features",
		threshold: 0.6,
		format:    "json",
	}
	if err := runMigrateTracks(context.Background(), hgDir, opts, &buf); err != nil {
		t.Fatalf("runMigrateTracks json: %v", err)
	}
	var ds []migrate.Decision
	if err := json.Unmarshal(buf.Bytes(), &ds); err != nil {
		t.Fatalf("not valid JSON Decision array: %v\n%s", err, buf.String())
	}
	if len(ds) == 0 {
		t.Errorf("expected non-empty decisions array")
	}
}

func TestMigrateTracksAmbiguousSkippedInWrite(t *testing.T) {
	hgDir, rulesPath := migrateTracksTestEnv(t)
	featAmbig := os.Getenv("MTT_FEAT_AMBIG")
	trkOld := os.Getenv("MTT_TRK_OLD")

	var buf bytes.Buffer
	opts := migrateTracksOpts{
		rulesPath: rulesPath,
		write:     true,
		types:     "features",
		threshold: 0.6,
		format:    "text",
	}
	if err := runMigrateTracks(context.Background(), hgDir, opts, &buf); err != nil {
		t.Fatalf("runMigrateTracks: %v", err)
	}

	p, err := workitem.Open(hgDir, "test-agent")
	if err != nil {
		t.Fatalf("workitem.Open: %v", err)
	}
	defer p.Close()
	feat, err := p.Features.Get(featAmbig)
	if err != nil {
		t.Fatalf("Features.Get: %v", err)
	}
	if feat.TrackID != trkOld {
		t.Errorf("ambiguous feature should not have moved; track_id = %q, want %q", feat.TrackID, trkOld)
	}
}

func TestMigrateTracksTypesValidation(t *testing.T) {
	hgDir, rulesPath := migrateTracksTestEnv(t)

	cases := []struct {
		types string
		ok    bool
	}{
		{"features", true},
		{"bugs", true},
		{"features,bugs", true},
		{"sessions", false}, // unknown type
		{"", false},
	}
	for _, c := range cases {
		opts := migrateTracksOpts{
			rulesPath: rulesPath,
			dryRun:    true,
			types:     c.types,
			threshold: 0.6,
			format:    "text",
		}
		var buf bytes.Buffer
		err := runMigrateTracks(context.Background(), hgDir, opts, &buf)
		if c.ok && err != nil {
			t.Errorf("types=%q: unexpected error %v", c.types, err)
		}
		if !c.ok && err == nil {
			t.Errorf("types=%q: expected error, got none", c.types)
		}
	}
}
