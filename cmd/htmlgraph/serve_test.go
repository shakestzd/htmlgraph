package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestBuildSingleProjectMuxRegistersMode verifies that the factory returns a
// non-nil mux that handles /api/mode without requiring DB access. This is
// the cheapest correctness check — it catches regressions where the factory
// is gutted or where runServer stops wiring the mode endpoint.
func TestBuildSingleProjectMuxRegistersMode(t *testing.T) {
	// /api/mode has no DB dependency so nil is safe here.
	mux := buildSingleProjectMux(nil, t.TempDir())
	if mux == nil {
		t.Fatal("buildSingleProjectMux returned nil")
	}

	req := httptest.NewRequest("GET", "/api/mode", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("/api/mode: got %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["mode"] != "single" {
		t.Errorf("mode: got %v, want single", body["mode"])
	}
}

// TestBuildSingleProjectMuxServesDashboard verifies the embedded dashboard
// file server is wired in. GET / should return 200 (or 301 redirect) — not
// 404. This catches the failure mode where the factory registers API routes
// but forgets the root handler.
func TestBuildSingleProjectMuxServesDashboard(t *testing.T) {
	mux := buildSingleProjectMux(nil, t.TempDir())

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Errorf("GET /: got 404, want dashboard response")
	}
}
