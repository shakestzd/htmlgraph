package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/shakestzd/htmlgraph/internal/db"
)

// setupPlanTestDB creates an in-memory DB with plan_feedback schema and inserts
// a test plan feature row. Returns the DB and plan ID.
func setupPlanTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	planID := "plan-route-test"
	_, err = database.Exec(
		`INSERT INTO features (id, type, title, status) VALUES (?, 'plan', 'Route Test Plan', 'in-progress')`,
		planID,
	)
	if err != nil {
		t.Fatalf("insert plan: %v", err)
	}
	return database, planID
}

// writeTempPlanHTML creates a temporary .htmlgraph/plans directory with a
// minimal plan HTML file. Returns the htmlgraphDir.
func writeTempPlanHTML(t *testing.T, planID string) string {
	t.Helper()
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		t.Fatalf("mkdir plans: %v", err)
	}
	html := `<!DOCTYPE html><html><body>` +
		`<article id="` + planID + `" data-type="plan" data-status="draft">` +
		`<header><h1>Test Plan</h1></header>` +
		`</article></body></html>`
	path := filepath.Join(plansDir, planID+".html")
	if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
		t.Fatalf("write plan html: %v", err)
	}
	return dir
}

// ---- extractPlanID ----------------------------------------------------------

func TestExtractPlanID(t *testing.T) {
	cases := []struct {
		path    string
		suffix  string
		want    string
		wantErr bool
	}{
		{"/api/plans/plan-abc/status", "/status", "plan-abc", false},
		{"/api/plans/plan-xyz/feedback", "/feedback", "plan-xyz", false},
		{"/api/plans/plan-123/finalize", "/finalize", "plan-123", false},
		{"/api/plans//status", "/status", "", true},
		{"/api/plans/plan-a/b/status", "/status", "", true},
		{"/other/path/status", "/status", "", true},
	}
	for _, tc := range cases {
		got, err := extractPlanID(tc.path, tc.suffix)
		if tc.wantErr {
			if err == nil {
				t.Errorf("extractPlanID(%q, %q): expected error, got %q", tc.path, tc.suffix, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("extractPlanID(%q, %q): unexpected error: %v", tc.path, tc.suffix, err)
			continue
		}
		if got != tc.want {
			t.Errorf("extractPlanID(%q, %q) = %q, want %q", tc.path, tc.suffix, got, tc.want)
		}
	}
}

// ---- planFileHandler --------------------------------------------------------

func TestPlanFileHandler_Serves(t *testing.T) {
	planID := "plan-file-test"
	htmlgraphDir := writeTempPlanHTML(t, planID)

	handler := planFileHandler(htmlgraphDir)
	req := httptest.NewRequest(http.MethodGet, "/plans/"+planID+".html", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct == "" {
		t.Error("expected non-empty Content-Type")
	}
}

func TestPlanFileHandler_NotFound(t *testing.T) {
	htmlgraphDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(htmlgraphDir, "plans"), 0o755); err != nil {
		t.Fatal(err)
	}
	handler := planFileHandler(htmlgraphDir)
	req := httptest.NewRequest(http.MethodGet, "/plans/plan-missing.html", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

func TestPlanFileHandler_RejectsPathTraversal(t *testing.T) {
	htmlgraphDir := t.TempDir()
	handler := planFileHandler(htmlgraphDir)
	req := httptest.NewRequest(http.MethodGet, "/plans/../secret.html", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	// http.NewRequest cleans the path, so we get a 404 (no file) not 400.
	// Acceptable: the traversal attempt is blocked either way.
	if w.Code == http.StatusOK {
		t.Error("expected non-200 for path traversal attempt")
	}
}

// ---- planStatusHandler ------------------------------------------------------

func TestPlanStatusHandler_OK(t *testing.T) {
	database, planID := setupPlanTestDB(t)
	htmlgraphDir := writeTempPlanHTML(t, planID)

	if err := db.StorePlanFeedback(database, planID, "design", "approve", "true", ""); err != nil {
		t.Fatalf("store feedback: %v", err)
	}

	handler := planStatusHandler(database, htmlgraphDir)
	req := httptest.NewRequest(http.MethodGet, "/api/plans/"+planID+"/status", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var resp planStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.PlanID != planID {
		t.Errorf("plan_id: got %q, want %q", resp.PlanID, planID)
	}
	if resp.Status != "draft" {
		t.Errorf("status: got %q, want draft", resp.Status)
	}
	if resp.ApprovedCount != 1 {
		t.Errorf("approved_count: got %d, want 1", resp.ApprovedCount)
	}
}

func TestPlanStatusHandler_PlanNotFound(t *testing.T) {
	database, _ := setupPlanTestDB(t)
	htmlgraphDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(htmlgraphDir, "plans"), 0o755); err != nil {
		t.Fatal(err)
	}
	handler := planStatusHandler(database, htmlgraphDir)
	req := httptest.NewRequest(http.MethodGet, "/api/plans/plan-missing/status", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

// ---- planFeedbackSubmitHandler ----------------------------------------------

func TestPlanFeedbackSubmitHandler_StoresFeedback(t *testing.T) {
	database, planID := setupPlanTestDB(t)
	handler := planFeedbackSubmitHandler(database)

	body, _ := json.Marshal(planFeedbackRequest{
		Section: "design",
		Action:  "approve",
		Value:   "true",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/plans/"+planID+"/feedback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200; body: %s", w.Code, w.Body.String())
	}

	entries, err := db.GetPlanFeedback(database, planID)
	if err != nil {
		t.Fatalf("get feedback: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("feedback count: got %d, want 1", len(entries))
	}
	if entries[0].Section != "design" || entries[0].Value != "true" {
		t.Errorf("unexpected entry: %+v", entries[0])
	}
}

func TestPlanFeedbackSubmitHandler_MissingFields(t *testing.T) {
	database, planID := setupPlanTestDB(t)
	handler := planFeedbackSubmitHandler(database)

	body, _ := json.Marshal(map[string]string{"section": "design"}) // missing action
	req := httptest.NewRequest(http.MethodPost, "/api/plans/"+planID+"/feedback", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

// ---- planFinalizeHandler ----------------------------------------------------

func TestPlanFinalizeHandler_NotApproved(t *testing.T) {
	database, planID := setupPlanTestDB(t)
	htmlgraphDir := writeTempPlanHTML(t, planID)

	handler := planFinalizeHandler(database, htmlgraphDir)
	req := httptest.NewRequest(http.MethodPost, "/api/plans/"+planID+"/finalize", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400; body: %s", w.Code, w.Body.String())
	}
}

func TestPlanFinalizeHandler_Success(t *testing.T) {
	database, planID := setupPlanTestDB(t)
	htmlgraphDir := writeTempPlanHTML(t, planID)

	for _, section := range []string{"design", "outline"} {
		if err := db.StorePlanFeedback(database, planID, section, "approve", "true", ""); err != nil {
			t.Fatalf("store feedback: %v", err)
		}
	}

	handler := planFinalizeHandler(database, htmlgraphDir)
	req := httptest.NewRequest(http.MethodPost, "/api/plans/"+planID+"/finalize", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "finalized" {
		t.Errorf("status field: got %v, want finalized", resp["status"])
	}

	// Verify HTML file was updated on disk.
	htmlStatus, err := parsePlanHTMLStatus(filepath.Join(htmlgraphDir, "plans", planID+".html"))
	if err != nil {
		t.Fatalf("parsePlanHTMLStatus: %v", err)
	}
	if htmlStatus != "finalized" {
		t.Errorf("HTML data-status: got %q, want finalized", htmlStatus)
	}
}

// ---- planFeedbackReadHandler ------------------------------------------------

func TestPlanFeedbackReadHandler_StructuredResponse(t *testing.T) {
	database, planID := setupPlanTestDB(t)

	if err := db.StorePlanFeedback(database, planID, "design", "approve", "true", ""); err != nil {
		t.Fatalf("store approve: %v", err)
	}
	if err := db.StorePlanFeedback(database, planID, "design", "comment", "looks good", ""); err != nil {
		t.Fatalf("store comment: %v", err)
	}
	if err := db.StorePlanFeedback(database, planID, "outline", "answer", "async", "delivery-mode"); err != nil {
		t.Fatalf("store answer: %v", err)
	}

	handler := planFeedbackReadHandler(database)
	req := httptest.NewRequest(http.MethodGet, "/api/plans/"+planID+"/feedback", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var resp planFeedbackResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.PlanID != planID {
		t.Errorf("plan_id: got %q, want %q", resp.PlanID, planID)
	}
	design, ok := resp.Sections["design"]
	if !ok {
		t.Fatal("missing 'design' section in response")
	}
	if !design.Approved {
		t.Error("design.approved: expected true")
	}
	if design.Comment != "looks good" {
		t.Errorf("design.comment: got %q, want 'looks good'", design.Comment)
	}
	if resp.Questions["delivery-mode"] != "async" {
		t.Errorf("questions[delivery-mode]: got %q, want async", resp.Questions["delivery-mode"])
	}
}

// ---- buildFeedbackResponse --------------------------------------------------

func TestBuildFeedbackResponse_AllApproved(t *testing.T) {
	entries := []db.PlanFeedback{
		{PlanID: "p1", Section: "a", Action: "approve", Value: "true"},
		{PlanID: "p1", Section: "b", Action: "approve", Value: "true"},
	}
	resp := buildFeedbackResponse("p1", entries)
	if resp.Status != "approved" {
		t.Errorf("status: got %q, want approved", resp.Status)
	}
}

func TestBuildFeedbackResponse_NotAllApproved(t *testing.T) {
	entries := []db.PlanFeedback{
		{PlanID: "p1", Section: "a", Action: "approve", Value: "true"},
		{PlanID: "p1", Section: "b", Action: "approve", Value: "false"},
	}
	resp := buildFeedbackResponse("p1", entries)
	if resp.Status != "draft" {
		t.Errorf("status: got %q, want draft", resp.Status)
	}
}
