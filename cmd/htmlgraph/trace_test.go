package main

import (
	"testing"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/models"
)

func TestLooksLikeFilePath(t *testing.T) {
	tests := []struct {
		arg  string
		want bool
	}{
		{"internal/db/schema.go", true},
		{"cmd/htmlgraph/main.go", true},
		{"file.go", true},
		{"./relative/path", true},
		{"abc1234", false},
		{"45da73fa", false},
		{"deadbeef", false},
	}
	for _, tt := range tests {
		if got := looksLikeFilePath(tt.arg); got != tt.want {
			t.Errorf("looksLikeFilePath(%q) = %v, want %v", tt.arg, got, tt.want)
		}
	}
}

func TestTraceFile(t *testing.T) {
	database, err := dbpkg.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	// Seed a track and feature.
	_, err = database.Exec(`INSERT INTO tracks (id, type, title, status) VALUES (?, ?, ?, ?)`,
		"trk-bbbb2222", "track", "Test track", "in-progress")
	if err != nil {
		t.Fatalf("insert track: %v", err)
	}
	_, err = database.Exec(`INSERT INTO features (id, type, title, status, track_id) VALUES (?, ?, ?, ?, ?)`,
		"feat-aaaa1111", "feature", "Test feature", "in-progress", "trk-bbbb2222")
	if err != nil {
		t.Fatalf("insert feature: %v", err)
	}

	// Seed a feature_files row.
	ff := &models.FeatureFile{
		ID:        "feat-aaaa1111-test",
		FeatureID: "feat-aaaa1111",
		FilePath:  "internal/db/schema.go",
		Operation: "edit",
		SessionID: "sess-test",
	}
	if err := dbpkg.UpsertFeatureFile(database, ff); err != nil {
		t.Fatalf("upsert feature file: %v", err)
	}

	results, err := dbpkg.TraceFile(database, "internal/db/schema.go")
	if err != nil {
		t.Fatalf("TraceFile: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.FeatureID != "feat-aaaa1111" {
		t.Errorf("FeatureID = %q, want feat-aaaa1111", r.FeatureID)
	}
	if r.Title != "Test feature" {
		t.Errorf("Title = %q, want 'Test feature'", r.Title)
	}
	if r.TrackID != "trk-bbbb2222" {
		t.Errorf("TrackID = %q, want trk-bbbb2222", r.TrackID)
	}
	if r.Operation != "edit" {
		t.Errorf("Operation = %q, want 'edit'", r.Operation)
	}
}

func TestTraceFile_NoResults(t *testing.T) {
	database, err := dbpkg.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	results, err := dbpkg.TraceFile(database, "nonexistent/file.go")
	if err != nil {
		t.Fatalf("TraceFile: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestTraceFile_MultipleFeatures(t *testing.T) {
	database, err := dbpkg.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	// Seed track and two features.
	database.Exec(`INSERT INTO tracks (id, type, title, status) VALUES (?, ?, ?, ?)`,
		"trk-xxxx1111", "track", "Test track", "in-progress")
	database.Exec(`INSERT INTO features (id, type, title, status, track_id) VALUES (?, ?, ?, ?, ?)`,
		"feat-aaaa1111", "feature", "First feature", "done", "trk-xxxx1111")
	database.Exec(`INSERT INTO features (id, type, title, status, track_id) VALUES (?, ?, ?, ?, ?)`,
		"feat-bbbb2222", "feature", "Second feature", "in-progress", "trk-xxxx1111")

	// Both touch the same file.
	dbpkg.UpsertFeatureFile(database, &models.FeatureFile{
		ID: "ff1", FeatureID: "feat-aaaa1111", FilePath: "shared/file.go", Operation: "write",
	})
	dbpkg.UpsertFeatureFile(database, &models.FeatureFile{
		ID: "ff2", FeatureID: "feat-bbbb2222", FilePath: "shared/file.go", Operation: "edit",
	})

	results, err := dbpkg.TraceFile(database, "shared/file.go")
	if err != nil {
		t.Fatalf("TraceFile: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}
