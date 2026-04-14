package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

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

// seedTraceFeatureDB creates a minimal in-memory DB with a feature, commit, and file.
func seedTraceFeatureDB(t *testing.T) (*sql.DB, string, string, string) {
	t.Helper()
	database, err := dbpkg.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	featureID := "feat-11223344"
	commitSHA := "aabbccdd1234567"
	filePath := "internal/db/trace_me.go"
	sessionID := "sess-trace-test"

	// Insert track and feature.
	database.Exec(`INSERT INTO tracks (id, type, title, status) VALUES (?, ?, ?, ?)`,
		"trk-trace0001", "track", "Trace Track", "in-progress")
	database.Exec(`INSERT INTO features (id, type, title, status, track_id) VALUES (?, ?, ?, ?, ?)`,
		featureID, "feature", "Trace Feature", "in-progress", "trk-trace0001")

	// Insert a commit linked to the feature.
	database.Exec(`INSERT INTO git_commits (commit_hash, session_id, feature_id, message, timestamp) VALUES (?, ?, ?, ?, ?)`,
		commitSHA, sessionID, featureID, "trace commit msg", time.Now().UTC().Format(time.RFC3339))

	// Insert a feature file.
	dbpkg.UpsertFeatureFile(database, &models.FeatureFile{
		ID:        "ff-trace-001",
		FeatureID: featureID,
		FilePath:  filePath,
		Operation: "edit",
		SessionID: sessionID,
	})

	return database, featureID, commitSHA, filePath
}

func TestTraceRoutesFeatureID(t *testing.T) {
	database, featureID, commitSHA, filePath := seedTraceFeatureDB(t)
	defer database.Close()

	var buf bytes.Buffer
	if err := runTraceFeature(&buf, database, featureID); err != nil {
		t.Fatalf("runTraceFeature: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, commitSHA[:9]) {
		t.Errorf("output should contain commit SHA prefix %q\ngot:\n%s", commitSHA[:9], out)
	}
	if !strings.Contains(out, filePath) {
		t.Errorf("output should contain file path %q\ngot:\n%s", filePath, out)
	}
}

func TestTraceJSONOutput(t *testing.T) {
	database, featureID, commitSHA, filePath := seedTraceFeatureDB(t)
	defer database.Close()

	var buf bytes.Buffer
	if err := runTraceFeatureJSON(&buf, database, featureID); err != nil {
		t.Fatalf("runTraceFeatureJSON: %v", err)
	}

	var result traceFeatureJSON
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("json.Unmarshal: %v\noutput:\n%s", err, buf.String())
	}

	if result.Feature != featureID {
		t.Errorf("JSON feature = %q, want %q", result.Feature, featureID)
	}
	if len(result.Commits) == 0 {
		t.Errorf("JSON commits should be non-empty")
	} else if result.Commits[0] != commitSHA {
		t.Errorf("JSON commits[0] = %q, want %q", result.Commits[0], commitSHA)
	}
	if len(result.Files) == 0 {
		t.Errorf("JSON files should be non-empty")
	} else if result.Files[0] != filePath {
		t.Errorf("JSON files[0] = %q, want %q", result.Files[0], filePath)
	}
}

func TestTraceSHAUnchanged(t *testing.T) {
	database, _, _, _ := seedTraceFeatureDB(t)
	defer database.Close()

	commitSHA := "aabbccdd1234567"

	results, err := dbpkg.TraceCommit(database, commitSHA)
	if err != nil {
		t.Fatalf("TraceCommit: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected TraceCommit to find the seeded commit via SHA path")
	}
	if results[0].CommitHash != commitSHA {
		t.Errorf("CommitHash = %q, want %q", results[0].CommitHash, commitSHA)
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
