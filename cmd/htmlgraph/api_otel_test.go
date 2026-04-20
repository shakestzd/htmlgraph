package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/shakestzd/htmlgraph/internal/db"
)

// seedOtelSignals creates a session and two prompts' worth of
// api_request + tool_result + api_error signals mirroring the
// empirical Claude fixtures used in the materializer tests.
func seedOtelSignals(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "api-otel.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })

	sessionID := "sess-api-1"
	database.Exec(`INSERT INTO sessions (session_id, agent_assigned) VALUES (?, ?)`, sessionID, "claude-code")

	insert := func(id, prompt, canonical, kind string, ts int64, tokIn, tokOut, cr, cc int64, cost float64, dur int64, attempt int) {
		t.Helper()
		_, err := database.Exec(`
			INSERT INTO otel_signals (
				signal_id, harness, session_id, prompt_id, kind, canonical, native,
				ts_micros, model, tokens_in, tokens_out,
				tokens_cache_read, tokens_cache_creation,
				cost_usd, cost_source, duration_ms, attempt, attrs_json
			) VALUES (?, 'claude_code', ?, ?, ?, ?, 'claude_code.'||?, ?, 'claude-haiku-4-5', ?, ?, ?, ?, ?, 'vendor', ?, ?, '{}')`,
			id, sessionID, prompt, kind, canonical, canonical,
			ts, tokIn, tokOut, cr, cc, cost, dur, attempt)
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	insert("s1", "prompt-A", "api_request", "log", 1, 10, 577, 23276, 2261, 0.00804885, 5835, 1)
	insert("s2", "prompt-A", "tool_result", "log", 2, 0, 0, 0, 0, 0, 100, 0)
	insert("s3", "prompt-B", "api_request", "log", 3, 3, 87, 0, 16623, 0.02121675, 1635, 1)
	insert("s4", "prompt-B", "api_error", "log", 4, 0, 0, 0, 0, 0, 30000, 11)
	return database
}

func TestOtelRollupHandler_LivePath(t *testing.T) {
	database := seedOtelSignals(t)

	req := httptest.NewRequest(http.MethodGet, "/api/otel/rollup?session_id=sess-api-1", nil)
	rec := httptest.NewRecorder()
	otelRollupHandler(database).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var got rollupJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.SessionID != "sess-api-1" {
		t.Errorf("SessionID = %q", got.SessionID)
	}
	if !got.Live {
		t.Errorf("Live = false, want true (no materialized row)")
	}
	wantCost := 0.00804885 + 0.02121675
	if got.TotalCostUSD < wantCost-1e-9 || got.TotalCostUSD > wantCost+1e-9 {
		t.Errorf("TotalCostUSD = %v, want %v", got.TotalCostUSD, wantCost)
	}
	if got.TotalAPIErrors != 1 {
		t.Errorf("TotalAPIErrors = %d", got.TotalAPIErrors)
	}
	if got.MaxAttempt != 11 {
		t.Errorf("MaxAttempt = %d", got.MaxAttempt)
	}
}

func TestOtelRollupHandler_MaterializedPath(t *testing.T) {
	database := seedOtelSignals(t)
	// Pre-write a materialized row so the handler hits the fast path.
	_, err := database.Exec(`
		INSERT INTO otel_session_rollup (
			session_id, harness, total_cost_usd,
			total_tokens_in, total_tokens_out,
			total_tokens_cache_read, total_tokens_cache_creation,
			total_tokens_thought, total_tokens_tool, total_tokens_reasoning,
			total_turns, total_tool_calls, total_api_calls, total_api_errors,
			max_attempt, materialized_at
		) VALUES ('sess-api-1', 'claude_code', 0.0325479,
			18, 664, 23276, 18884, 0, 0, 0,
			2, 1, 2, 1,
			11, 0)`)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/otel/rollup?session_id=sess-api-1", nil)
	rec := httptest.NewRecorder()
	otelRollupHandler(database).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got rollupJSON
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Live {
		t.Error("Live = true, want false (materialized row present)")
	}
	if got.TotalTokensIn != 18 {
		t.Errorf("TotalTokensIn = %d, want 18 (materialized value)", got.TotalTokensIn)
	}
}

func TestOtelRollupHandler_404ForMissingSession(t *testing.T) {
	database := seedOtelSignals(t)
	req := httptest.NewRequest(http.MethodGet, "/api/otel/rollup?session_id=nonexistent", nil)
	rec := httptest.NewRecorder()
	otelRollupHandler(database).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestOtelRollupHandler_400ForMissingParam(t *testing.T) {
	database := seedOtelSignals(t)
	req := httptest.NewRequest(http.MethodGet, "/api/otel/rollup", nil)
	rec := httptest.NewRecorder()
	otelRollupHandler(database).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestOtelPromptsHandler(t *testing.T) {
	database := seedOtelSignals(t)
	req := httptest.NewRequest(http.MethodGet, "/api/otel/prompts?session_id=sess-api-1", nil)
	rec := httptest.NewRecorder()
	otelPromptsHandler(database).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		Prompts []promptJSON `json:"prompts"`
	}
	json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Prompts) != 2 {
		t.Fatalf("got %d prompts, want 2", len(body.Prompts))
	}
	if body.Prompts[0].PromptID != "prompt-A" {
		t.Errorf("first prompt = %q, want prompt-A", body.Prompts[0].PromptID)
	}
	// prompt-B has the api_error; it should surface on the breakdown.
	if body.Prompts[1].APIErrors != 1 {
		t.Errorf("prompt-B APIErrors = %d, want 1", body.Prompts[1].APIErrors)
	}
}

func TestOtelCostHandler_GroupByModel(t *testing.T) {
	database := seedOtelSignals(t)
	req := httptest.NewRequest(http.MethodGet, "/api/otel/cost?group_by=model", nil)
	rec := httptest.NewRecorder()
	otelCostHandler(database).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		GroupBy string `json:"group_by"`
		Buckets []struct {
			Key       string  `json:"key"`
			TotalCost float64 `json:"total_cost_usd"`
		} `json:"buckets"`
	}
	json.Unmarshal(rec.Body.Bytes(), &body)
	if body.GroupBy != "model" {
		t.Errorf("GroupBy = %q", body.GroupBy)
	}
	if len(body.Buckets) != 1 {
		t.Fatalf("got %d buckets, want 1", len(body.Buckets))
	}
	if body.Buckets[0].Key != "claude-haiku-4-5" {
		t.Errorf("Key = %q", body.Buckets[0].Key)
	}
}

func TestOtelCostHandler_GroupBySession(t *testing.T) {
	database := seedOtelSignals(t)
	req := httptest.NewRequest(http.MethodGet, "/api/otel/cost?group_by=session", nil)
	rec := httptest.NewRecorder()
	otelCostHandler(database).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOtelCostHandler_BadGroupBy(t *testing.T) {
	database := seedOtelSignals(t)
	req := httptest.NewRequest(http.MethodGet, "/api/otel/cost?group_by=bogus", nil)
	rec := httptest.NewRecorder()
	otelCostHandler(database).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestOtelHandlers_RejectNonGet(t *testing.T) {
	database := seedOtelSignals(t)
	for _, h := range []http.HandlerFunc{
		otelRollupHandler(database),
		otelPromptsHandler(database),
		otelCostHandler(database),
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/otel/x?session_id=sess-api-1", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST status = %d, want 405", rec.Code)
		}
	}
}
