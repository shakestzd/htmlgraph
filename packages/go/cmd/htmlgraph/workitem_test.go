package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shakestzd/htmlgraph/internal/htmlparse"
)

func TestAutoTrackEdgesOnCreate(t *testing.T) {
	tmpDir := t.TempDir()
	hgDir := filepath.Join(tmpDir, ".htmlgraph")
	for _, sub := range []string{"features", "bugs", "spikes", "tracks", "plans", "specs"} {
		if err := os.MkdirAll(filepath.Join(hgDir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	projectDirFlag = tmpDir
	defer func() { projectDirFlag = "" }()

	// Create a track first
	if err := runWiCreate("track", "Test Track", "", "medium", false); err != nil {
		t.Fatalf("create track: %v", err)
	}

	// Find the track ID from disk
	trackFiles, _ := filepath.Glob(filepath.Join(hgDir, "tracks", "trk-*.html"))
	if len(trackFiles) != 1 {
		t.Fatalf("expected 1 track file, got %d", len(trackFiles))
	}
	trackNode, err := htmlparse.ParseFile(trackFiles[0])
	if err != nil {
		t.Fatalf("parse track: %v", err)
	}
	trackID := trackNode.ID

	// Create a feature linked to the track
	if err := runWiCreate("feature", "Tracked Feature", trackID, "high", false); err != nil {
		t.Fatalf("create feature: %v", err)
	}

	// Find the feature
	featFiles, _ := filepath.Glob(filepath.Join(hgDir, "features", "feat-*.html"))
	if len(featFiles) != 1 {
		t.Fatalf("expected 1 feature file, got %d", len(featFiles))
	}
	featNode, err := htmlparse.ParseFile(featFiles[0])
	if err != nil {
		t.Fatalf("parse feature: %v", err)
	}

	// Verify feature has part_of edge to track
	partOfEdges, ok := featNode.Edges["part_of"]
	if !ok || len(partOfEdges) == 0 {
		t.Errorf("feature missing part_of edge; edges = %v", featNode.Edges)
	} else if partOfEdges[0].TargetID != trackID {
		t.Errorf("part_of target = %q, want %q", partOfEdges[0].TargetID, trackID)
	}

	// Re-read the track to check contains edge
	trackNode, err = htmlparse.ParseFile(trackFiles[0])
	if err != nil {
		t.Fatalf("re-parse track: %v", err)
	}
	containsEdges, ok := trackNode.Edges["contains"]
	if !ok || len(containsEdges) == 0 {
		t.Errorf("track missing contains edge; edges = %v", trackNode.Edges)
	} else if containsEdges[0].TargetID != featNode.ID {
		t.Errorf("contains target = %q, want %q", containsEdges[0].TargetID, featNode.ID)
	}
}

func TestAutoTrackEdgesNotCreatedForTrack(t *testing.T) {
	tmpDir := t.TempDir()
	hgDir := filepath.Join(tmpDir, ".htmlgraph")
	for _, sub := range []string{"features", "bugs", "spikes", "tracks", "plans", "specs"} {
		os.MkdirAll(filepath.Join(hgDir, sub), 0o755)
	}

	projectDirFlag = tmpDir
	defer func() { projectDirFlag = "" }()

	// Creating a track should not attempt auto-edges even if trackID is passed
	if err := runWiCreate("track", "Parent Track", "", "medium", false); err != nil {
		t.Fatalf("create track: %v", err)
	}

	trackFiles, _ := filepath.Glob(filepath.Join(hgDir, "tracks", "trk-*.html"))
	if len(trackFiles) != 1 {
		t.Fatalf("expected 1 track file, got %d", len(trackFiles))
	}
	node, _ := htmlparse.ParseFile(trackFiles[0])
	if len(node.Edges) > 0 {
		t.Errorf("track should have no edges, got %v", node.Edges)
	}
}

func TestAutoImplementedInEdgeOnStart(t *testing.T) {
	tmpDir := t.TempDir()
	hgDir := filepath.Join(tmpDir, ".htmlgraph")
	for _, sub := range []string{"features", "bugs", "spikes", "tracks", "plans", "specs"} {
		os.MkdirAll(filepath.Join(hgDir, sub), 0o755)
	}

	projectDirFlag = tmpDir
	defer func() { projectDirFlag = "" }()

	// Set a fake session ID (EnvSessionID reads HTMLGRAPH_SESSION_ID first)
	t.Setenv("HTMLGRAPH_SESSION_ID", "test-session-abc")

	// Create a feature
	if err := runWiCreate("feature", "Impl Feature", "", "high", false); err != nil {
		t.Fatalf("create feature: %v", err)
	}

	// Find the feature ID
	featFiles, _ := filepath.Glob(filepath.Join(hgDir, "features", "feat-*.html"))
	if len(featFiles) != 1 {
		t.Fatalf("expected 1 feature file, got %d", len(featFiles))
	}
	featNode, _ := htmlparse.ParseFile(featFiles[0])
	featID := featNode.ID

	// Start the feature (should create implemented_in edge)
	if err := runWiSetStatus("feature", featID, "in-progress"); err != nil {
		t.Fatalf("start feature: %v", err)
	}

	// Re-read and check for implemented_in edge
	featNode, _ = htmlparse.ParseFile(featFiles[0])
	implEdges, ok := featNode.Edges["implemented_in"]
	if !ok || len(implEdges) == 0 {
		t.Errorf("feature missing implemented_in edge; edges = %v", featNode.Edges)
	} else if implEdges[0].TargetID != "test-session-abc" {
		t.Errorf("implemented_in target = %q, want %q", implEdges[0].TargetID, "test-session-abc")
	}

	// Start again — should be idempotent (no duplicate edge)
	if err := runWiSetStatus("feature", featID, "in-progress"); err != nil {
		t.Fatalf("re-start feature: %v", err)
	}
	featNode, _ = htmlparse.ParseFile(featFiles[0])
	implEdges = featNode.Edges["implemented_in"]
	if len(implEdges) != 1 {
		t.Errorf("expected 1 implemented_in edge after re-start, got %d", len(implEdges))
	}
}

func TestNoImplementedInEdgeWithoutSession(t *testing.T) {
	tmpDir := t.TempDir()
	hgDir := filepath.Join(tmpDir, ".htmlgraph")
	for _, sub := range []string{"features", "bugs", "spikes", "tracks", "plans", "specs"} {
		os.MkdirAll(filepath.Join(hgDir, sub), 0o755)
	}

	projectDirFlag = tmpDir
	defer func() { projectDirFlag = "" }()

	// Use a sentinel value that won't match any real session, then clear it.
	// EnvSessionID checks HTMLGRAPH_SESSION_ID first; "none" forces it to
	// return "none" (which is fine — the edge target will be "none").
	// Instead, we set it empty and chdir to tmpDir so readActiveSession
	// finds no .active-session file.
	t.Setenv("HTMLGRAPH_SESSION_ID", "")
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	if err := runWiCreate("feature", "No Session Feature", "", "low", false); err != nil {
		t.Fatalf("create: %v", err)
	}

	featFiles, _ := filepath.Glob(filepath.Join(hgDir, "features", "feat-*.html"))
	featNode, _ := htmlparse.ParseFile(featFiles[0])

	if err := runWiSetStatus("feature", featNode.ID, "in-progress"); err != nil {
		t.Fatalf("start: %v", err)
	}

	featNode, _ = htmlparse.ParseFile(featFiles[0])
	if len(featNode.Edges["implemented_in"]) > 0 {
		t.Errorf("should not have implemented_in edge without session, got %v", featNode.Edges)
	}
}
