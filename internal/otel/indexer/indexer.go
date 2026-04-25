package indexer

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/shakestzd/htmlgraph/internal/otel"
	"github.com/shakestzd/htmlgraph/internal/otel/sink"
)

const pollInterval = 500 * time.Millisecond

// FileInfo holds per-file health metrics for the /api/indexer/status endpoint.
type FileInfo struct {
	LastOffset    int64     `json:"last_offset"`
	CurrentSize   int64     `json:"current_size"`
	LagBytes      int64     `json:"lag_bytes"`
	LastError     string    `json:"last_error"`
	LastIndexedAt time.Time `json:"last_indexed_at"`
}

// Indexer polls .htmlgraph/sessions/*/events.ndjson files for new appends,
// parses each line into a UnifiedSignal, and applies them to SQLite via snk.
type Indexer struct {
	htmlgraphDir string
	snk          sink.SignalSink

	mu     sync.RWMutex
	status map[string]FileInfo
}

// New constructs an Indexer rooted at htmlgraphDir.
// htmlgraphDir is the .htmlgraph/ directory (e.g. /path/to/project/.htmlgraph).
func New(htmlgraphDir string, snk sink.SignalSink) *Indexer {
	return &Indexer{
		htmlgraphDir: htmlgraphDir,
		snk:          snk,
		status:       make(map[string]FileInfo),
	}
}

// Start runs the poll loop until ctx is cancelled. Intended to be called as a goroutine.
func (idx *Indexer) Start(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			idx.runOnce(ctx)
		}
	}
}

// Status returns a snapshot of per-session file health.
func (idx *Indexer) Status() map[string]FileInfo {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	out := make(map[string]FileInfo, len(idx.status))
	for k, v := range idx.status {
		out[k] = v
	}
	return out
}

// runOnce discovers all sessions and processes any new data.
func (idx *Indexer) runOnce(ctx context.Context) {
	sessions, err := idx.discoverSessions()
	if err != nil {
		log.Printf("indexer: discover sessions: %v", err)
		return
	}
	for _, sid := range sessions {
		if ctx.Err() != nil {
			return
		}
		if err := idx.processSession(ctx, sid); err != nil {
			log.Printf("indexer: session %s: %v", sid, err)
			idx.recordError(sid, err)
		}
	}
}

// discoverSessions returns session IDs that have an events.ndjson file.
func (idx *Indexer) discoverSessions() ([]string, error) {
	sessionsDir := filepath.Join(idx.htmlgraphDir, "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var sessions []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		ndjson := filepath.Join(sessionsDir, e.Name(), "events.ndjson")
		if _, err := os.Stat(ndjson); err == nil {
			sessions = append(sessions, e.Name())
		}
	}
	return sessions, nil
}

// processSession tails events.ndjson for sessionID from the last checkpoint,
// parses each line, and applies the batch to snk. On success, writes a new checkpoint.
func (idx *Indexer) processSession(ctx context.Context, sessionID string) error {
	sessDir := filepath.Join(idx.htmlgraphDir, "sessions", sessionID)
	ndjsonPath := filepath.Join(sessDir, "events.ndjson")
	checkpointPath := filepath.Join(sessDir, ".index-offset")

	offset, err := readCheckpoint(checkpointPath)
	if err != nil {
		return err
	}

	info, err := os.Stat(ndjsonPath)
	if err != nil {
		return err
	}
	currentSize := info.Size()
	idx.updateSize(sessionID, offset, currentSize)

	if currentSize <= offset {
		return nil // no new data
	}

	signals, newOffset, err := idx.readNewSignals(ndjsonPath, offset)
	if err != nil {
		return err
	}
	if len(signals) == 0 {
		return writeCheckpoint(checkpointPath, newOffset)
	}

	if err := idx.snk.WriteBatch(ctx, otel.HarnessClaude, nil, signals); err != nil {
		return err
	}

	if err := writeCheckpoint(checkpointPath, newOffset); err != nil {
		return err
	}

	idx.recordSuccess(sessionID, newOffset, currentSize)
	return nil
}

// readNewSignals opens ndjsonPath, seeks to offset, reads and parses lines.
// Returns the parsed signals and the new file offset after the last processed line.
func (idx *Indexer) readNewSignals(ndjsonPath string, offset int64) ([]otel.UnifiedSignal, int64, error) {
	f, err := os.Open(ndjsonPath)
	if err != nil {
		return nil, offset, err
	}
	defer f.Close()

	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return nil, offset, err
		}
	}

	var signals []otel.UnifiedSignal
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	committedOffset := offset

	for scanner.Scan() {
		lineBytes := scanner.Bytes()
		lineLen := int64(len(lineBytes)) + 1

		if len(lineBytes) == 0 {
			committedOffset += lineLen
			continue
		}

		sig, err := parseLine(lineBytes)
		if err != nil {
			log.Printf("indexer: skip malformed line in %s at offset ~%d: %v",
				ndjsonPath, committedOffset+lineLen, err)
			committedOffset += lineLen
			continue
		}
		if sig == nil {
			committedOffset += lineLen
			continue
		}
		signals = append(signals, *sig)
		committedOffset += lineLen
	}
	if err := scanner.Err(); err != nil {
		if err == bufio.ErrTooLong {
			log.Printf("indexer: line exceeds 4MB in %s at offset ~%d — skipping",
				ndjsonPath, committedOffset)
			return signals, committedOffset, nil
		}
		return signals, committedOffset, err
	}
	return signals, committedOffset, nil
}

// updateSize records the current file size without touching LastIndexedAt.
func (idx *Indexer) updateSize(sessionID string, offset, currentSize int64) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	fi := idx.status[sessionID]
	fi.LastOffset = offset
	fi.CurrentSize = currentSize
	fi.LagBytes = currentSize - offset
	idx.status[sessionID] = fi
}

// recordSuccess updates the status snapshot after a successful batch.
func (idx *Indexer) recordSuccess(sessionID string, newOffset, currentSize int64) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.status[sessionID] = FileInfo{
		LastOffset:    newOffset,
		CurrentSize:   currentSize,
		LagBytes:      currentSize - newOffset,
		LastIndexedAt: time.Now().UTC(),
	}
}

// recordError updates the last_error field in the status snapshot.
func (idx *Indexer) recordError(sessionID string, err error) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	fi := idx.status[sessionID]
	fi.LastError = err.Error()
	idx.status[sessionID] = fi
}
