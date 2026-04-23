package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/shakestzd/htmlgraph/internal/terminal"
)

// terminalStarter is the interface used by handleTerminalStart.
// Defined as an interface to allow mocking in tests.
type terminalStarter interface {
	Start(req terminal.StartRequest, defaultDir string) (port, pid int, err error)
	Stop(pid int) error
}

// terminalMgr is the package-level manager for ttyd sidecar processes.
// It is initialised once and shared across all requests.
var terminalMgr terminalStarter = terminal.NewManager()

// terminalStartRequest is the JSON body for POST /api/terminal/start.
// All fields are optional; zero values fall back to MVP defaults.
type terminalStartRequest struct {
	Agent    string `json:"agent"`
	Mode     string `json:"mode"`
	CWD      string `json:"cwd"`
	WorkItem string `json:"work_item"`
}

// terminalStartResponse is the JSON body returned on success.
type terminalStartResponse struct {
	Port     int    `json:"port"`
	Pid      int    `json:"pid"`
	URL      string `json:"url"`
	Agent    string `json:"agent,omitempty"`
	Mode     string `json:"mode,omitempty"`
	WorkItem string `json:"work_item,omitempty"`
}

// terminalStopRequest is the JSON body for POST /api/terminal/stop.
type terminalStopRequest struct {
	Pid int `json:"pid"`
}

// handleTerminalStart handles POST /api/terminal/start.
// It spawns a ttyd sidecar on a free port and returns the access URL.
// The starter parameter allows injection of a mock for testing.
func handleTerminalStart(projectDir string, starter ...terminalStarter) http.HandlerFunc {
	mgr := terminalMgr
	if len(starter) > 0 && starter[0] != nil {
		mgr = starter[0]
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req terminalStartRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		startReq := terminal.StartRequest{
			Agent:    req.Agent,
			Mode:     req.Mode,
			CWD:      req.CWD,
			WorkItem: req.WorkItem,
		}
		port, pid, err := mgr.Start(startReq, projectDir)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		// Echo back effective agent/mode so callers can verify what launched.
		agent := req.Agent
		if agent == "" {
			agent = "claude"
		}
		mode := req.Mode
		if mode == "" {
			mode = "dev"
		}

		respondJSON(w, terminalStartResponse{
			Port:     port,
			Pid:      pid,
			URL:      fmt.Sprintf("http://127.0.0.1:%d", port),
			Agent:    agent,
			Mode:     mode,
			WorkItem: req.WorkItem,
		})
	}
}

// handleTerminalStop handles POST /api/terminal/stop.
// It signals the ttyd process identified by pid to terminate.
func handleTerminalStop(starter ...terminalStarter) http.HandlerFunc {
	mgr := terminalMgr
	if len(starter) > 0 && starter[0] != nil {
		mgr = starter[0]
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req terminalStopRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.Pid == 0 {
			http.Error(w, "pid required", http.StatusBadRequest)
			return
		}

		if err := mgr.Stop(req.Pid); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		respondJSON(w, map[string]bool{"ok": true})
	}
}
