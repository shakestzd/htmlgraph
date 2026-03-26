// Package eventlog provides an append-only JSONL event logger.
//
// Each session gets its own file: .htmlgraph/events/<session_id>.jsonl
// Records are appended one JSON object per line, matching the Python
// JsonlEventLog from event_log.py.
package eventlog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/shakestzd/htmlgraph/internal/models"
)

// Logger is a thread-safe, append-only JSONL event writer.
type Logger struct {
	dir string
	mu  sync.Mutex
}

// New creates a Logger that writes to the given events directory.
// The directory is created if it does not exist.
func New(eventsDir string) (*Logger, error) {
	if err := os.MkdirAll(eventsDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating events dir: %w", err)
	}
	return &Logger{dir: eventsDir}, nil
}

// pathFor returns the JSONL file path for a given session.
func (l *Logger) pathFor(sessionID string) string {
	return filepath.Join(l.dir, sessionID+".jsonl")
}

// Append serializes the record as a single JSON line and appends it to the
// session's JSONL file. Duplicate event_ids (checked against the file tail)
// are silently skipped, matching the Python dedup behaviour.
func (l *Logger) Append(record *models.EventRecord) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	path := l.pathFor(record.SessionID)

	// Best-effort dedup: check if event_id already exists in file tail.
	if isDuplicate(path, record.EventID) {
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	data = append(data, '\n')
	_, err = f.Write(data)
	return err
}

// ReadSession reads all event records from a session's JSONL file.
// Returns an empty slice (not an error) if the file does not exist.
func (l *Logger) ReadSession(sessionID string) ([]models.EventRecord, error) {
	path := l.pathFor(sessionID)

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var records []models.EventRecord
	scanner := bufio.NewScanner(f)
	// Allow lines up to 1 MB (events with large payloads).
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec models.EventRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			// Skip malformed lines (matches Python's lenient behaviour).
			continue
		}
		records = append(records, rec)
	}
	return records, scanner.Err()
}

// isDuplicate scans the tail of the file for a matching event_id.
// Returns false if the file does not exist or on any I/O error.
func isDuplicate(path, eventID string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.Size() == 0 {
		return false
	}

	// Read last 64 KB — same window as the Python implementation.
	tailSize := int64(64 * 1024)
	if info.Size() < tailSize {
		tailSize = info.Size()
	}
	if _, err := f.Seek(-tailSize, 2); err != nil {
		return false
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	count := 0
	for scanner.Scan() {
		count++
		// Only check last 250 lines, matching Python.
		if count > 250 {
			break
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		// Quick string check before full parse.
		if !containsBytes(line, []byte(eventID)) {
			continue
		}
		var partial struct {
			EventID string `json:"event_id"`
		}
		if json.Unmarshal(line, &partial) == nil && partial.EventID == eventID {
			return true
		}
	}
	return false
}

// containsBytes is a simple byte-slice substring check.
func containsBytes(haystack, needle []byte) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) && bytesContains(haystack, needle)
}

func bytesContains(haystack, needle []byte) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
