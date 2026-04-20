package receiver_test

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/otel"
	"github.com/shakestzd/htmlgraph/internal/otel/receiver"
)

// newWriter opens a fresh SQLite DB with the OTel schema and returns
// both a writer and a reader handle. The reader is a second *sql.DB
// for assertions (we can't query through the writer's MaxOpenConns=1
// while a transaction is open in a concurrent test).
func newWriter(t *testing.T) (*receiver.Writer, string) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "otel.db")
	readDB, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}
	readDB.Close()
	w, err := receiver.NewWriter(dbPath)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	t.Cleanup(func() { w.Close() })
	return w, dbPath
}

func sigFixture(session, prompt string, overrides ...func(*otel.UnifiedSignal)) otel.UnifiedSignal {
	s := otel.UnifiedSignal{
		Harness:       otel.HarnessClaude,
		SignalID:      "sig-" + session + "-" + prompt,
		Kind:          otel.KindLog,
		CanonicalName: otel.CanonicalAPIRequest,
		NativeName:    "api_request",
		Timestamp:     time.Unix(0, 1735000000000000000),
		SessionID:     session,
		PromptID:      prompt,
		Model:         "claude-haiku-4-5-20251001",
		Tokens: otel.TokenCounts{
			Input: 10, Output: 577, CacheRead: 23276, CacheCreation: 2261,
		},
		CostUSD:    0.00804885,
		CostSource: otel.CostSourceVendor,
		DurationMs: 5835,
		RawAttrs:   map[string]any{"request_id": "req_011"},
	}
	for _, fn := range overrides {
		fn(&s)
	}
	return s
}

func TestWriter_InsertsSignalAndPlaceholderSession(t *testing.T) {
	w, _ := newWriter(t)
	ctx := context.Background()

	res := map[string]any{
		"service.name":    "claude-code",
		"service.version": "2.1.42",
		"terminal.type":   "iTerm.app",
	}
	inserted, err := w.WriteBatch(ctx, otel.HarnessClaude, res,
		[]otel.UnifiedSignal{sigFixture("sess-A", "prompt-1")})
	if err != nil {
		t.Fatalf("WriteBatch: %v", err)
	}
	if inserted != 1 {
		t.Errorf("inserted = %d, want 1", inserted)
	}

	// Session placeholder created by the writer.
	var agent, status string
	if err := w.DB().QueryRow(
		"SELECT agent_assigned, status FROM sessions WHERE session_id='sess-A'",
	).Scan(&agent, &status); err != nil {
		t.Fatalf("lookup session placeholder: %v", err)
	}
	if agent != "claude_code" || status != "active" {
		t.Errorf("placeholder session = (%q, %q)", agent, status)
	}

	// Resource attributes recorded.
	var val string
	if err := w.DB().QueryRow(
		"SELECT value FROM otel_resource_attrs WHERE session_id='sess-A' AND key='terminal.type'",
	).Scan(&val); err != nil {
		t.Fatalf("resource attr lookup: %v", err)
	}
	if val != "iTerm.app" {
		t.Errorf("terminal.type = %q", val)
	}

	// Signal row has the token + cost data preserved.
	var tokensIn, tokensOut int64
	var cost float64
	if err := w.DB().QueryRow(
		"SELECT tokens_in, tokens_out, cost_usd FROM otel_signals WHERE signal_id='sig-sess-A-prompt-1'",
	).Scan(&tokensIn, &tokensOut, &cost); err != nil {
		t.Fatalf("signal lookup: %v", err)
	}
	if tokensIn != 10 || tokensOut != 577 {
		t.Errorf("tokens = (%d, %d)", tokensIn, tokensOut)
	}
	if cost != 0.00804885 {
		t.Errorf("cost = %v", cost)
	}
}

func TestWriter_IdempotentOnDuplicateSignalID(t *testing.T) {
	w, _ := newWriter(t)
	ctx := context.Background()
	sig := sigFixture("sess-B", "prompt-1")
	batch := []otel.UnifiedSignal{sig}

	n1, _ := w.WriteBatch(ctx, otel.HarnessClaude, map[string]any{"service.name": "claude-code"}, batch)
	n2, _ := w.WriteBatch(ctx, otel.HarnessClaude, map[string]any{"service.name": "claude-code"}, batch)
	if n1 != 1 || n2 != 0 {
		t.Errorf("insert counts (%d, %d), want (1, 0)", n1, n2)
	}

	var count int
	if err := w.DB().QueryRow(
		"SELECT COUNT(*) FROM otel_signals WHERE session_id='sess-B'").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("duplicate-insert produced %d rows, want 1", count)
	}
}

func TestWriter_BatchMultipleSessions(t *testing.T) {
	w, _ := newWriter(t)
	ctx := context.Background()

	batch := []otel.UnifiedSignal{
		sigFixture("sess-C", "p1"),
		sigFixture("sess-D", "p1"),
		sigFixture("sess-C", "p2"),
	}
	n, err := w.WriteBatch(ctx, otel.HarnessClaude, map[string]any{"service.name": "claude-code"}, batch)
	if err != nil {
		t.Fatalf("WriteBatch: %v", err)
	}
	if n != 3 {
		t.Errorf("inserted = %d, want 3", n)
	}

	// Exactly one placeholder session per distinct ID.
	var c int
	if err := w.DB().QueryRow("SELECT COUNT(*) FROM sessions WHERE session_id IN ('sess-C','sess-D')").Scan(&c); err != nil {
		t.Fatalf("count: %v", err)
	}
	if c != 2 {
		t.Errorf("session count = %d, want 2", c)
	}
}

// TestWriter_ConcurrentBatches verifies the MaxOpenConns=1 invariant
// prevents SQLITE_BUSY under concurrent writers. Two goroutines each
// insert a batch; both must succeed, and the final row count must be
// the sum without loss.
func TestWriter_ConcurrentBatches(t *testing.T) {
	w, _ := newWriter(t)
	ctx := context.Background()
	res := map[string]any{"service.name": "claude-code"}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for g := 0; g < 2; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			batch := make([]otel.UnifiedSignal, 20)
			for i := range batch {
				batch[i] = sigFixture(
					"sess-G",
					"p-g1",
					func(s *otel.UnifiedSignal) {
						s.SignalID = "g" + string(rune('0'+g)) + "-" + string(rune('0'+i%10)) + "-" + string(rune('a'+i/10))
					},
				)
			}
			if _, err := w.WriteBatch(ctx, otel.HarnessClaude, res, batch); err != nil {
				errs <- err
			}
		}(g)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatalf("concurrent write failed: %v", e)
	}

	var c int
	if err := w.DB().QueryRow("SELECT COUNT(*) FROM otel_signals WHERE session_id='sess-G'").Scan(&c); err != nil {
		t.Fatalf("count: %v", err)
	}
	if c != 40 {
		t.Errorf("concurrent batches produced %d rows, want 40", c)
	}
}

func TestWriter_DropsSignalWithEmptySessionID(t *testing.T) {
	w, _ := newWriter(t)
	ctx := context.Background()
	batch := []otel.UnifiedSignal{
		sigFixture("", "p1"),             // dropped — no session
		sigFixture("sess-F", "p1"),       // kept
	}
	n, err := w.WriteBatch(ctx, otel.HarnessClaude, map[string]any{"service.name": "claude-code"}, batch)
	if err != nil {
		t.Fatalf("WriteBatch: %v", err)
	}
	if n != 1 {
		t.Errorf("inserted = %d, want 1 (empty-session dropped)", n)
	}
}
