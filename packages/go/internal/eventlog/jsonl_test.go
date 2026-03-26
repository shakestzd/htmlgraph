package eventlog_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shakestzd/htmlgraph/internal/eventlog"
	"github.com/shakestzd/htmlgraph/internal/models"
)

func TestAppendAndRead(t *testing.T) {
	dir := t.TempDir()

	logger, err := eventlog.New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)

	records := []models.EventRecord{
		{
			EventID:   "evt-001",
			Timestamp: now,
			SessionID: "sess-test",
			Agent:     "claude",
			Tool:      "Bash",
			Summary:   "ran git status",
			Success:   true,
			FilePaths: []string{},
		},
		{
			EventID:   "evt-002",
			Timestamp: now.Add(time.Second),
			SessionID: "sess-test",
			Agent:     "claude",
			Tool:      "Read",
			Summary:   "read main.go",
			Success:   true,
			FeatureID: "feat-abc",
			FilePaths: []string{"main.go"},
		},
	}

	for _, rec := range records {
		if err := logger.Append(&rec); err != nil {
			t.Fatalf("Append %s: %v", rec.EventID, err)
		}
	}

	// Verify file exists.
	path := filepath.Join(dir, "sess-test.jsonl")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("JSONL file was not created")
	}

	// Read back.
	got, err := logger.ReadSession("sess-test")
	if err != nil {
		t.Fatalf("ReadSession: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("record count: got %d, want 2", len(got))
	}

	if got[0].EventID != "evt-001" {
		t.Errorf("first event: got %q, want %q", got[0].EventID, "evt-001")
	}
	if got[1].FeatureID != "feat-abc" {
		t.Errorf("second event feature_id: got %q, want %q", got[1].FeatureID, "feat-abc")
	}
	if got[1].Tool != "Read" {
		t.Errorf("second event tool: got %q, want %q", got[1].Tool, "Read")
	}
}

func TestDedupe(t *testing.T) {
	dir := t.TempDir()

	logger, err := eventlog.New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := &models.EventRecord{
		EventID:   "evt-dup",
		Timestamp: time.Now().UTC(),
		SessionID: "sess-dup",
		Agent:     "claude",
		Tool:      "Bash",
		Summary:   "test",
		Success:   true,
		FilePaths: []string{},
	}

	// Append twice.
	if err := logger.Append(rec); err != nil {
		t.Fatalf("first append: %v", err)
	}
	if err := logger.Append(rec); err != nil {
		t.Fatalf("second append: %v", err)
	}

	got, err := logger.ReadSession("sess-dup")
	if err != nil {
		t.Fatalf("ReadSession: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("dedup failed: got %d records, want 1", len(got))
	}
}

func TestReadNonexistentSession(t *testing.T) {
	dir := t.TempDir()

	logger, err := eventlog.New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	got, err := logger.ReadSession("nonexistent")
	if err != nil {
		t.Fatalf("ReadSession error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %d records", len(got))
	}
}
