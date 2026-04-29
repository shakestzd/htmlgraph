package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

// TestGenerateOtelSessionID verifies OTel session ID generation produces
// unique, non-empty strings with the expected format.
func TestGenerateOtelSessionID(t *testing.T) {
	id1 := generateOtelSessionID()
	if id1 == "" {
		t.Fatal("generateOtelSessionID returned empty string")
	}
	id2 := generateOtelSessionID()
	if id2 == "" {
		t.Fatal("generateOtelSessionID returned empty string")
	}
	if id1 == id2 {
		t.Errorf("two calls returned same ID: %q", id1)
	}
	// 12 hex timestamp + 16 hex entropy = 28 chars
	if len(id1) != 28 {
		t.Errorf("session ID length = %d, want 28: %q", len(id1), id1)
	}
}

// TestSpawnCollector_HandshakeAndPort spawns a real otel-collect child,
// asserts the handshake returns a valid port, and verifies the process
// is alive.
func TestSpawnCollector_HandshakeAndPort(t *testing.T) {
	bin := buildOtelCollectTestBinary(t)
	projectDir := mkOtelCollectProject(t)

	port, proc, err := spawnCollector(bin, "test-spawn-hs", projectDir)
	if err != nil {
		t.Fatalf("spawnCollector: %v", err)
	}
	t.Cleanup(func() {
		_ = proc.Kill()
		_, _ = proc.Wait()
	})

	if port <= 0 || port > 65535 {
		t.Errorf("port out of range: %d", port)
	}

	// Process should be alive — kill -0 check (signal 0 probes existence).
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		t.Errorf("collector process not alive: %v", err)
	}
}

// TestSpawnCollector_BindFailure tests that a non-existent binary path
// causes spawnCollector to return an error without leaking a process.
func TestSpawnCollector_BindFailure(t *testing.T) {
	projectDir := mkOtelCollectProject(t)

	port, proc, err := spawnCollector("/nonexistent/binary", "test-bindfail", projectDir)
	if err == nil {
		if proc != nil {
			_ = proc.Kill()
			_, _ = proc.Wait()
		}
		t.Fatal("expected error for non-existent binary, got nil")
	}
	if port != 0 {
		t.Errorf("expected port 0 on error, got %d", port)
	}
	if proc != nil {
		t.Error("expected nil process on error")
	}
}

// TestSpawnCollector_HandshakeTimeout verifies that spawnCollector returns
// an error when the child does not print a handshake line within the timeout.
// We simulate this by spawning a binary that never prints the expected line.
func TestSpawnCollector_HandshakeTimeout(t *testing.T) {
	// Use "sleep" as the binary — it will never print a handshake.
	// spawnCollector should timeout and kill it.
	port, proc, err := spawnCollector("sleep", "test-timeout", t.TempDir())
	if err == nil {
		if proc != nil {
			_ = proc.Kill()
			_, _ = proc.Wait()
		}
		t.Fatal("expected error for non-handshaking binary, got nil")
	}
	if port != 0 {
		t.Errorf("expected port 0 on error, got %d", port)
	}
	if proc != nil {
		t.Error("expected nil process on error")
	}
	if !strings.Contains(err.Error(), "handshake") && !strings.Contains(err.Error(), "timeout") &&
		!strings.Contains(err.Error(), "start") {
		t.Errorf("error should mention handshake/timeout/start, got: %v", err)
	}
}

// TestWriteCollectorPID writes a PID file and reads it back.
func TestWriteCollectorPID(t *testing.T) {
	projectDir := t.TempDir()
	sid := "test-pid-write"
	pid := 42

	writeCollectorPID(projectDir, sid, pid)

	pidPath := filepath.Join(projectDir, ".htmlgraph", "sessions", sid, ".collector-pid")
	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("PID file not found at %s: %v", pidPath, err)
	}

	got, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("PID file content is not a valid integer: %q", string(data))
	}
	if got != pid {
		t.Errorf("PID = %d, want %d", got, pid)
	}
}

// TestWriteCollectorPID_CreatesDirectories verifies that writeCollectorPID
// creates the necessary directory structure.
func TestWriteCollectorPID_CreatesDirectories(t *testing.T) {
	projectDir := t.TempDir()
	sid := "test-pid-dirs"

	writeCollectorPID(projectDir, sid, 1234)

	sessDir := filepath.Join(projectDir, ".htmlgraph", "sessions", sid)
	info, err := os.Stat(sessDir)
	if err != nil {
		t.Fatalf("session dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("session dir is not a directory")
	}
}

// TestSpawnFailLoudStrict verifies that when HTMLGRAPH_OTEL_STRICT=1 and
// collector spawn fails, spawnSessionCollectorTo emits a FATAL line on the
// provided stderr writer and returns wantExit=true.
func TestSpawnFailLoudStrict(t *testing.T) {
	t.Setenv("HTMLGRAPH_OTEL_STRICT", "1")

	var buf bytes.Buffer
	projectDir := t.TempDir()

	overrides, wantExit := spawnSessionCollectorTo(projectDir, "/nonexistent/binary", &buf)

	stderr := buf.String()
	if !strings.Contains(stderr, "htmlgraph: FATAL:") {
		t.Errorf("expected FATAL line on stderr, got: %q", stderr)
	}
	if !wantExit {
		t.Error("expected wantExit=true when HTMLGRAPH_OTEL_STRICT=1 and spawn fails")
	}
	if overrides.CollectorPort != 0 || overrides.SessionID != "" || overrides.Cleanup != nil {
		t.Errorf("expected zero-value overrides on failure, got: %+v", overrides)
	}
}

// TestSpawnQuietByDefault verifies that without HTMLGRAPH_OTEL_STRICT, a
// failed spawn still emits a FATAL line on stderr but returns wantExit=false
// and zero-value overrides (degraded mode).
func TestSpawnQuietByDefault(t *testing.T) {
	t.Setenv("HTMLGRAPH_OTEL_STRICT", "")

	var buf bytes.Buffer
	projectDir := t.TempDir()

	overrides, wantExit := spawnSessionCollectorTo(projectDir, "/nonexistent/binary", &buf)

	stderr := buf.String()
	if !strings.Contains(stderr, "htmlgraph: FATAL:") {
		t.Errorf("expected FATAL line on stderr even without strict mode, got: %q", stderr)
	}
	if wantExit {
		t.Error("expected wantExit=false when HTMLGRAPH_OTEL_STRICT is not set")
	}
	if overrides.CollectorPort != 0 || overrides.SessionID != "" || overrides.Cleanup != nil {
		t.Errorf("expected zero-value overrides on failure, got: %+v", overrides)
	}
}
