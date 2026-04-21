package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/shakestzd/htmlgraph/internal/terminal"
)

// terminalMgr is the package-level manager for ttyd sidecar processes.
// It is initialised once and shared across all requests.
var terminalMgr = terminal.NewManager()

// terminalStartRequest is the JSON body for POST /api/terminal/start.
type terminalStartRequest struct {
	WorkItem string `json:"work_item"`
}

// terminalStartResponse is the JSON body returned on success.
type terminalStartResponse struct {
	Port int    `json:"port"`
	Pid  int    `json:"pid"`
	URL  string `json:"url"`
}

// terminalStopRequest is the JSON body for POST /api/terminal/stop.
type terminalStopRequest struct {
	Pid int `json:"pid"`
}

// handleTerminalStart handles POST /api/terminal/start.
// It spawns a ttyd sidecar on a free port and returns the access URL.
func handleTerminalStart(projectDir string) http.HandlerFunc {
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

		port, pid, err := terminalMgr.Start(projectDir, req.WorkItem)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		respondJSON(w, terminalStartResponse{
			Port: port,
			Pid:  pid,
			URL:  fmt.Sprintf("http://127.0.0.1:%d", port),
		})
	}
}

// handleTerminalStop handles POST /api/terminal/stop.
// It signals the ttyd process identified by pid to terminate.
func handleTerminalStop() http.HandlerFunc {
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

		if err := terminalMgr.Stop(req.Pid); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		respondJSON(w, map[string]bool{"ok": true})
	}
}
