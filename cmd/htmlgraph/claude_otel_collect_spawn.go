package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/shakestzd/htmlgraph/internal/otel/collector"
)

// generateOtelSessionID produces a hex session ID from a Unix-millisecond
// timestamp (12 hex digits) and 8 random bytes (16 hex digits), giving
// 28 hex characters total. Lexicographically sortable by creation time.
// Distinct from generateSessionID (sess-{hex8}) which is used for
// non-OTel session tracking.
func generateOtelSessionID() string {
	ts := time.Now().UnixMilli()
	var entropy [8]byte
	_, _ = rand.Read(entropy[:]) // crypto/rand never errors on supported platforms
	return fmt.Sprintf("%012x%016x", ts, entropy)
}

// otelEnvOverrides holds optional overrides for OTel env vars set by
// the launcher. Zero-value fields mean "use the default derivation".
type otelEnvOverrides struct {
	CollectorPort int
	SessionID     string
	Cleanup       func() // called on launcher exit to SIGTERM the collector
}

// spawnCollectorFn is the package-level spawn function used by
// retrySpawnCollector. Tests may replace it to inject a fake.
var spawnCollectorFn = collector.DefaultSpawnFn

// spawnCollector starts an otel-collect child process, waits for its
// handshake line ("htmlgraph-otel-ready port=<N>"), and returns the
// port and process. Delegates to collector.DefaultSpawnFn.
//
// binPath is the path to the htmlgraph binary to invoke. In production
// callers should pass the result of os.Executable(); tests pass a
// pre-built test binary.
func spawnCollector(binPath, sessionID, projectDir string) (int, *os.Process, error) {
	return collector.DefaultSpawnFn(binPath, sessionID, projectDir)
}

// retrySpawnCollector attempts to spawn the collector up to maxAttempts times.
// Backoff delays between attempts are: 100ms, 300ms, 700ms (indices 0, 1, 2).
// spawnFn overrides the package-level spawnCollectorFn when non-nil (for tests).
// After each non-final failure a warning line is written to warnW.
// Returns the port, process, number of attempts made, and any final error.
func retrySpawnCollector(binPath, sessionID, projectDir string, maxAttempts int, spawnFn func(string, string, string) (int, *os.Process, error), warnW io.Writer) (int, *os.Process, int, error) {
	if spawnFn == nil {
		spawnFn = spawnCollectorFn
	}
	return collector.RetrySpawn(binPath, sessionID, projectDir, maxAttempts, spawnFn, warnW)
}

// watchdogInterval returns the polling interval for the collector watchdog.
// HTMLGRAPH_OTEL_WATCHDOG_INTERVAL overrides the default of 15s.
func watchdogInterval() time.Duration {
	return collector.WatchdogInterval("HTMLGRAPH_OTEL_WATCHDOG_INTERVAL")
}

// startCollectorWatchdog launches a goroutine that polls the collector process
// every watchdogInterval(). On process death it calls retrySpawnCollector and
// updates the current process. Returns a stop func that terminates the goroutine.
func startCollectorWatchdog(initialProc *os.Process, binPath, sessionID, projectDir string, warnW io.Writer) func() {
	return collector.StartWatchdog(initialProc, binPath, sessionID, projectDir, warnW, spawnCollectorFn, "HTMLGRAPH_OTEL_WATCHDOG_INTERVAL")
}

// registerCollectorCleanup returns a cleanup function that sends SIGTERM,
// waits up to 3s, then SIGKILLs, and removes the .collector-pid file.
func registerCollectorCleanup(proc *os.Process, projectDir, sessionID string) func() {
	return collector.RegisterCleanup(proc, projectDir, sessionID)
}

// removeCollectorPID removes the .collector-pid file for a session.
// Best-effort: missing file or unreadable directory is not an error.
func removeCollectorPID(projectDir, sessionID string) {
	collector.RemoveCollectorPID(projectDir, sessionID)
}

// writeCollectorPID writes the collector PID to the session directory.
// Best-effort: errors are silently ignored.
func writeCollectorPID(projectDir, sessionID string, pid int) {
	collector.WriteCollectorPID(projectDir, sessionID, pid)
}

// spawnSessionCollectorTo is the testable core of collector spawning.
// It generates a session ID, spawns the collector at binPath (with up to 3
// retry attempts using exponential backoff), and returns overrides and a
// wantExit flag. On spawn failure it always writes a FATAL line to errW;
// wantExit is true only when HTMLGRAPH_OTEL_STRICT=1.
func spawnSessionCollectorTo(projectDir, binPath string, errW io.Writer) (otelEnvOverrides, bool) {
	sessionID := generateOtelSessionID()

	port, proc, attempts, err := retrySpawnCollector(binPath, sessionID, projectDir, 3, nil, errW)
	if err != nil {
		fmt.Fprintf(errW, "htmlgraph: FATAL: collector spawn failed after %d attempts: %v\n", attempts, err)
		wantExit := os.Getenv("HTMLGRAPH_OTEL_STRICT") == "1"
		return otelEnvOverrides{}, wantExit
	}

	writeCollectorPID(projectDir, sessionID, proc.Pid)
	stopWatchdog := startCollectorWatchdog(proc, binPath, sessionID, projectDir, errW)
	baseCleanup := registerCollectorCleanup(proc, projectDir, sessionID)
	cleanup := func() { stopWatchdog(); baseCleanup() }

	return otelEnvOverrides{
		CollectorPort: port,
		SessionID:     sessionID,
		Cleanup:       cleanup,
	}, false
}

// spawnSessionCollector generates a session ID, spawns a per-session
// collector, writes the PID file, and returns a cleanup function.
// On spawn failure emits a FATAL line to stderr; exits non-zero when
// HTMLGRAPH_OTEL_STRICT=1. Silent-fail is preserved when the binary
// path cannot be resolved (soft precondition).
func spawnSessionCollector(projectDir string) otelEnvOverrides {
	binPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "htmlgraph: warning: per-session collector skipped: %v\n", err)
		return otelEnvOverrides{}
	}

	overrides, wantExit := spawnSessionCollectorTo(projectDir, binPath, os.Stderr)
	if wantExit {
		os.Exit(1)
	}
	return overrides
}
