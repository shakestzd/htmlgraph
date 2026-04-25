package ndjson_test

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shakestzd/htmlgraph/internal/otel"
	"github.com/shakestzd/htmlgraph/internal/otel/sink/ndjson"
)

func makeSignal(kind otel.Kind, id, sessionID string) otel.UnifiedSignal {
	return otel.UnifiedSignal{
		Harness:       otel.HarnessClaude,
		SignalID:      id,
		Kind:          kind,
		CanonicalName: "test_event",
		NativeName:    "test.event",
		Timestamp:     time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC),
		SessionID:     sessionID,
		RawAttrs:      map[string]any{"key": "val"},
	}
}

func TestNDJSONSink_OneLinePerSignal(t *testing.T) {
	dir := t.TempDir()
	sid := "ndjson-test-sess"
	sessDir := filepath.Join(dir, ".htmlgraph", "sessions", sid)
	if err := os.MkdirAll(sessDir, 0o755); err != nil {
		t.Fatal(err)
	}

	s, err := ndjson.New(dir, sid)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()

	signals := []otel.UnifiedSignal{
		makeSignal(otel.KindSpan, "id-span", sid),
		makeSignal(otel.KindMetric, "id-metric", sid),
		makeSignal(otel.KindLog, "id-log", sid),
	}
	if err := s.WriteBatch(context.Background(), otel.HarnessClaude, nil, signals); err != nil {
		t.Fatalf("WriteBatch: %v", err)
	}

	f, err := os.Open(filepath.Join(sessDir, "events.ndjson"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	var rows []map[string]any
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var m map[string]any
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			t.Errorf("bad JSON: %v | line: %s", err, sc.Text())
			continue
		}
		rows = append(rows, m)
	}
	if sc.Err() != nil {
		t.Fatal(sc.Err())
	}

	if len(rows) != 3 {
		t.Fatalf("want 3 lines, got %d", len(rows))
	}

	wantKinds := []string{"span", "metric", "log"}
	for i, row := range rows {
		k, _ := row["kind"].(string)
		if k != wantKinds[i] {
			t.Errorf("row %d: want kind=%q got %q", i, wantKinds[i], k)
		}
		if _, ok := row["ts"]; !ok {
			t.Errorf("row %d: missing ts", i)
		}
		if _, ok := row["harness"]; !ok {
			t.Errorf("row %d: missing harness", i)
		}
		if _, ok := row["signal_id"]; !ok {
			t.Errorf("row %d: missing signal_id", i)
		}
	}
}

func TestNDJSONSink_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	sid := "json-valid-sess"
	sessDir := filepath.Join(dir, ".htmlgraph", "sessions", sid)
	os.MkdirAll(sessDir, 0o755)

	s, _ := ndjson.New(dir, sid)
	defer s.Close()

	signals := []otel.UnifiedSignal{makeSignal(otel.KindSpan, "id-1", sid)}
	s.WriteBatch(context.Background(), otel.HarnessClaude, map[string]any{"env": "test"}, signals)

	data, err := os.ReadFile(filepath.Join(sessDir, "events.ndjson"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data[:len(data)-1], &m); err != nil { // strip trailing newline
		t.Fatalf("not valid JSON: %v\nline: %s", err, data)
	}
}

func TestNDJSONSink_EmptyBatchIsNoOp(t *testing.T) {
	dir := t.TempDir()
	sid := "empty-batch-sess"
	sessDir := filepath.Join(dir, ".htmlgraph", "sessions", sid)
	os.MkdirAll(sessDir, 0o755)

	s, _ := ndjson.New(dir, sid)
	defer s.Close()

	if err := s.WriteBatch(context.Background(), otel.HarnessClaude, nil, nil); err != nil {
		t.Fatalf("empty WriteBatch returned error: %v", err)
	}

	// File should not exist since nothing was written.
	ndjsonPath := filepath.Join(sessDir, "events.ndjson")
	if _, err := os.Stat(ndjsonPath); !os.IsNotExist(err) {
		// File may exist if New creates it eagerly — only check if no lines.
		f, _ := os.Open(ndjsonPath)
		if f != nil {
			sc := bufio.NewScanner(f)
			lineCount := 0
			for sc.Scan() {
				lineCount++
			}
			f.Close()
			if lineCount != 0 {
				t.Errorf("empty batch: expected 0 lines, got %d", lineCount)
			}
		}
	}
}

func TestNDJSONSink_CloseIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	sid := "close-idem-sess"
	sessDir := filepath.Join(dir, ".htmlgraph", "sessions", sid)
	os.MkdirAll(sessDir, 0o755)

	s, err := ndjson.New(dir, sid)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Errorf("first Close: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}
