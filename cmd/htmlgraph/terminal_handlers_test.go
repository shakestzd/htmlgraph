package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shakestzd/htmlgraph/internal/terminal"
)

// mockTerminalStarter is a test double that records Start calls without spawning ttyd.
type mockTerminalStarter struct {
	lastReq    terminal.StartRequest
	lastDir    string
	returnPort int
	returnPid  int
	returnErr  error
}

func (m *mockTerminalStarter) Start(req terminal.StartRequest, defaultDir string) (int, int, error) {
	m.lastDir = defaultDir
	m.lastReq = req
	return m.returnPort, m.returnPid, m.returnErr
}

func (m *mockTerminalStarter) Stop(pid int) error {
	return nil
}

// TestTerminalStartHandler_EmptyBody verifies that POST {} returns 200 and spawns
// the default claude --dev session in the server's projectDir (back-compat).
func TestTerminalStartHandler_EmptyBody(t *testing.T) {
	mock := &mockTerminalStarter{returnPort: 9999, returnPid: 1234}
	handler := handleTerminalStart("/srv/project", mock)

	req := httptest.NewRequest(http.MethodPost, "/api/terminal/start", bytes.NewBufferString("{}"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200 — body: %s", rec.Code, rec.Body)
	}

	// Empty body should default to agent=claude, mode=dev.
	if mock.lastReq.Agent != "" {
		t.Errorf("expected Agent to be empty (handler passes through; terminal applies default), got %q", mock.lastReq.Agent)
	}
	if mock.lastReq.Mode != "" {
		t.Errorf("expected Mode to be empty (handler passes through; terminal applies default), got %q", mock.lastReq.Mode)
	}
	if mock.lastDir != "/srv/project" {
		t.Errorf("expected projectDir /srv/project, got %q", mock.lastDir)
	}

	var resp terminalStartResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Port != 9999 {
		t.Errorf("port: got %d, want 9999", resp.Port)
	}
	if resp.Pid != 1234 {
		t.Errorf("pid: got %d, want 1234", resp.Pid)
	}
	// Empty body → handler echoes back defaults.
	if resp.Agent != "claude" {
		t.Errorf("agent in response: got %q, want claude", resp.Agent)
	}
	if resp.Mode != "dev" {
		t.Errorf("mode in response: got %q, want dev", resp.Mode)
	}
}

// TestTerminalStartHandler_CustomAgent verifies that custom agent/mode/cwd/work_item
// fields are decoded and passed through to the manager correctly.
func TestTerminalStartHandler_CustomAgent(t *testing.T) {
	mock := &mockTerminalStarter{returnPort: 8888, returnPid: 5678}
	handler := handleTerminalStart("/srv/project", mock)

	body := `{"agent":"codex","mode":"dev","cwd":"/mock/test","work_item":"feat-abc"}`
	req := httptest.NewRequest(http.MethodPost, "/api/terminal/start", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200 — body: %s", rec.Code, rec.Body)
	}

	if mock.lastReq.Agent != "codex" {
		t.Errorf("agent: got %q, want codex", mock.lastReq.Agent)
	}
	if mock.lastReq.Mode != "dev" {
		t.Errorf("mode: got %q, want dev", mock.lastReq.Mode)
	}
	if mock.lastReq.CWD != "/mock/test" {
		t.Errorf("cwd: got %q, want /mock/test", mock.lastReq.CWD)
	}
	if mock.lastReq.WorkItem != "feat-abc" {
		t.Errorf("work_item: got %q, want feat-abc", mock.lastReq.WorkItem)
	}

	var resp terminalStartResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.WorkItem != "feat-abc" {
		t.Errorf("work_item in response: got %q, want feat-abc", resp.WorkItem)
	}
	if resp.Agent != "codex" {
		t.Errorf("agent in response: got %q, want codex", resp.Agent)
	}
}

// TestTerminalStartHandler_MethodNotAllowed verifies that GET returns 405.
func TestTerminalStartHandler_MethodNotAllowed(t *testing.T) {
	mock := &mockTerminalStarter{}
	handler := handleTerminalStart("/srv/project", mock)

	req := httptest.NewRequest(http.MethodGet, "/api/terminal/start", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want 405", rec.Code)
	}
}
