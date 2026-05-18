package main

// bug-74a7bda7 (roborev HIGH follow-up): under the read-only dashboard mux
// topology the child server passes a read-only handle for read routes and a
// SEPARATE writable handle for mutating routes. roborev found that mutating
// HTTP endpoints (plan feedback/finalize/delete/chat, manual session ingest)
// were still wired to the read-only handle and would fail SQLITE_READONLY.
//
// These tests reproduce the exact production wiring (read-only `database` +
// writable `writeDB`) and assert:
//
//   - POST /api/plans/{id}/feedback succeeds (writes land via writeDB), with
//     NO "attempt to write a readonly database" / SQLITE_READONLY error, and
//   - a read endpoint (GET /api/plans/{id}/feedback) still works on the
//     read-only handle and observes the write.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shakestzd/wipnote/internal/db"

	_ "modernc.org/sqlite"
)

// newSplitHandleDBPath builds a schema'd temp-file DB (via the writable
// schema-creating Open, exactly like runServeChild) and returns its path so
// callers can open the production handle pair (read-only mux + dedicated
// writable) against the same file.
func newSplitHandleDBPath(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "split-handle.db")
	w, err := db.Open(dbPath) // create schema + migrations
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	w.Close()
	return dbPath
}

// TestPlanFeedbackPOST_WritesViaWritableHandle_UnderReadOnlyMux drives
// planRouter with the production split-handle wiring and asserts the POST
// feedback path commits through the writable handle while the read-only mux
// handle rejects direct writes.
func TestPlanFeedbackPOST_WritesViaWritableHandle_UnderReadOnlyMux(t *testing.T) {
	dbPath := newSplitHandleDBPath(t)

	roDB, err := db.OpenReadOnly(dbPath)
	if err != nil {
		t.Fatalf("OpenReadOnly: %v", err)
	}
	defer roDB.Close()
	writeDB, err := db.OpenWritable(dbPath)
	if err != nil {
		t.Fatalf("OpenWritable: %v", err)
	}
	defer writeDB.Close()

	// Sanity: the read-only mux handle must reject writes.
	if _, err := roDB.Exec(
		`INSERT INTO plan_feedback (plan_id, section, action, value, question_id)
		 VALUES ('p','design','approve','true','')`,
	); err == nil {
		t.Fatal("read-only handle accepted an INSERT; want SQLITE_READONLY")
	}

	const planID = "plan-ro-topology"
	// Production wiring: read routes on roDB, mutating routes on writeDB.
	router := planRouter(roDB, writeDB, t.TempDir())

	// POST /feedback — mutating route, must succeed via writeDB.
	reqBody, _ := json.Marshal(planFeedbackRequest{
		Section: "design", Action: "approve", Value: "true",
	})
	postReq := httptest.NewRequest(http.MethodPost,
		"/api/plans/"+planID+"/feedback", bytes.NewReader(reqBody))
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	router(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("POST /feedback: got %d, want 200; body: %s",
			postRec.Code, postRec.Body.String())
	}
	if b := postRec.Body.String(); strings.Contains(b, "readonly") ||
		strings.Contains(b, "SQLITE_READONLY") || strings.Contains(b, "read-only") {
		t.Fatalf("POST /feedback leaked a read-only error: %s", b)
	}

	// The write must actually be visible via the read-only handle.
	entries, err := db.GetPlanFeedback(roDB, planID)
	if err != nil {
		t.Fatalf("GetPlanFeedback via read-only handle: %v", err)
	}
	if len(entries) != 1 || entries[0].Section != "design" {
		t.Fatalf("feedback not persisted via writable handle: %+v", entries)
	}

	// GET /feedback — read route on the read-only handle still works.
	getReq := httptest.NewRequest(http.MethodGet,
		"/api/plans/"+planID+"/feedback", nil)
	getRec := httptest.NewRecorder()
	router(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET /feedback on read-only handle: got %d, want 200; body: %s",
			getRec.Code, getRec.Body.String())
	}
}

// TestSessionIngestPOST_WritesViaWritableHandle_UnderReadOnlyMux asserts the
// manual-ingest mutating route uses the writable handle and does not fail
// with SQLITE_READONLY under the split-handle topology. (No JSONL file exists
// for the synthetic session id, so the handler returns the not_found status
// JSON — the point is that it does NOT 500 on a read-only write error and
// does NOT 404, which would mean the route was misrouted.)
func TestSessionIngestPOST_WritesViaWritableHandle_UnderReadOnlyMux(t *testing.T) {
	dbPath := newSplitHandleDBPath(t)
	roDB, err := db.OpenReadOnly(dbPath)
	if err != nil {
		t.Fatalf("OpenReadOnly: %v", err)
	}
	defer roDB.Close()
	writeDB, err := db.OpenWritable(dbPath)
	if err != nil {
		t.Fatalf("OpenWritable: %v", err)
	}
	defer writeDB.Close()

	handler := sessionIngestHandler(roDB, writeDB)
	req := httptest.NewRequest(http.MethodPost,
		"/api/sessions/sess-ro-ingest-1/ingest", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatalf("ingest route 404'd — misrouted (preview stole the route)")
	}
	if b := rec.Body.String(); strings.Contains(b, "readonly") ||
		strings.Contains(b, "SQLITE_READONLY") || strings.Contains(b, "read-only") {
		t.Fatalf("ingest leaked a read-only write error under split topology: %s", b)
	}
}
