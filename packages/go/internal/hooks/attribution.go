package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// traceparentEntry is the JSON structure written to the temp queue so
// subagent sessions can claim their parent linkage at session-start time.
type traceparentEntry struct {
	TraceID      string  `json:"trace_id"`
	ParentSpanID string  `json:"parent_span_id"`
	Timestamp    float64 `json:"timestamp"`
	Claimed      bool    `json:"claimed"`
}

// writeTraceparent writes a traceparent entry to the temp queue directory.
// Mirrors the Python write_traceparent_queue() helper in pretooluse.py.
func writeTraceparent(parentSessionID, parentEventID string) {
	queueDir := filepath.Join(os.TempDir(), "htmlgraph-traceparent")
	if err := os.MkdirAll(queueDir, 0o755); err != nil {
		return
	}

	entry := traceparentEntry{
		TraceID:      parentSessionID,
		ParentSpanID: parentEventID,
		Timestamp:    float64(time.Now().UnixNano()) / 1e9,
		Claimed:      false,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	filename := fmt.Sprintf("tp-%s.json", uuid.New().String()[:8])
	path := filepath.Join(queueDir, filename)
	_ = os.WriteFile(path, data, 0o644)
}

// claimTraceparent reads and claims the most recent unclaimed traceparent
// from the temp queue. Returns nil if nothing is available or entries are stale.
// Mirrors claim_traceparent() in session-start.py.
func claimTraceparent() *traceparentEntry {
	queueDir := filepath.Join(os.TempDir(), "htmlgraph-traceparent")
	entries, err := filepath.Glob(filepath.Join(queueDir, "tp-*.json"))
	if err != nil || len(entries) == 0 {
		return nil
	}

	now := float64(time.Now().UnixNano()) / 1e9
	var best *traceparentEntry
	var bestPath string

	for _, path := range entries {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var entry traceparentEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}
		age := now - entry.Timestamp
		if entry.Claimed || age > 30 {
			if age > 300 {
				// Clean up stale entries older than 5 minutes.
				_ = os.Remove(path)
			}
			continue
		}
		// Prefer the most recent unclaimed entry.
		if best == nil || entry.Timestamp > best.Timestamp {
			best = &entry
			bestPath = path
		}
	}

	if best == nil || bestPath == "" {
		return nil
	}

	// Claim it atomically by rewriting with claimed=true.
	best.Claimed = true
	if data, err := json.Marshal(best); err == nil {
		_ = os.WriteFile(bestPath, data, 0o644)
	}
	return best
}

// ApplyTraceparent reads a traceparent from the queue and exports env vars
// for parent session / parent event linkage. Called during session-start.
func ApplyTraceparent() (parentSession, parentEvent string) {
	tp := claimTraceparent()
	if tp == nil {
		return "", ""
	}
	if tp.TraceID != "" {
		os.Setenv("HTMLGRAPH_PARENT_SESSION", tp.TraceID)
	}
	if tp.ParentSpanID != "" {
		os.Setenv("HTMLGRAPH_PARENT_EVENT", tp.ParentSpanID)
	}
	return tp.TraceID, tp.ParentSpanID
}
