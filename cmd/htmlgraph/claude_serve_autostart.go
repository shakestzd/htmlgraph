package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	otelreceiver "github.com/shakestzd/htmlgraph/internal/otel/receiver"
)

// serveLockPath returns the path of the per-project serve lock file.
// The lock file stores the PID of a running `htmlgraph serve` process.
func serveLockPath(projectDir string) string {
	return filepath.Join(projectDir, ".htmlgraph", ".serve.lock")
}

// ensureServeForOtel checks whether the OTLP HTTP receiver is already
// listening, and spawns a detached `htmlgraph serve` if not. Called from
// launchClaude before exec'ing claude so that OTel signals from the
// child process have somewhere to land.
//
// Gating:
//   - When HTMLGRAPH_OTEL_ENABLED is explicitly disabled (0/false/no/off),
//     return immediately — user opted out.
//   - When the configured OTLP port already accepts a TCP connection,
//     assume a receiver is live (either htmlgraph serve or a user-run
//     collector) — return nil. We don't probe further because there's
//     no portable way to tell "ours" from "theirs" without sending a
//     real request, and duplicating a running server would be worse
//     than leaving an external one in charge.
//   - Otherwise spawn `htmlgraph serve` detached, wait up to 3 seconds
//     for it to bind, and log a warning if it never does. Never return
//     an error — a missing receiver is degraded operation, not a fatal
//     launcher failure.
//
// The spawned process inherits the parent env so the serve child's
// receiver wiring picks up HTMLGRAPH_OTEL_* config. Stdout/stderr go to
// a log file under .htmlgraph/logs so the orphaned server doesn't
// pollute the user's terminal.
func ensureServeForOtel(projectDir string) {
	if isExplicitlyDisabled(os.Getenv("HTMLGRAPH_OTEL_ENABLED")) {
		return
	}
	cfg := otelreceiver.LoadConfigFromEnv("", projectDir)
	host := cfg.BindHost
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	port := cfg.HTTPPort
	if port == 0 {
		port = 4318
	}

	if probePort(host, port, 200*time.Millisecond) {
		return // something is already bound — leave it alone
	}

	// Check the lockfile before spawning. If a serve process is already
	// running (lock file contains a live PID), skip the spawn to prevent
	// a second htmlgraph serve from racing to bind port 8080.
	if skipSpawn, stale := checkServeLock(projectDir); skipSpawn {
		debugLog("ensureServeForOtel: skipping spawn, serve already running (lockfile)")
		return
	} else if stale {
		// Stale lockfile (process gone) — remove it so future spawns work.
		_ = os.Remove(serveLockPath(projectDir))
	}

	if err := spawnDetachedServe(projectDir); err != nil {
		fmt.Fprintf(os.Stderr, "htmlgraph: auto-start serve failed: %v\n", err)
		return
	}

	// Poll the port for up to 3 seconds. If it binds within that window,
	// OTel signals from the spawned claude will land correctly. Anything
	// longer suggests the server hit a problem — warn but continue.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for {
		if probePort(host, port, 200*time.Millisecond) {
			fmt.Fprintf(os.Stderr, "htmlgraph: started serve for OTel receiver on %s:%d\n", host, port)
			return
		}
		select {
		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "htmlgraph: serve did not bind %s:%d within 3s; OTel signals may drop until it comes up\n", host, port)
			return
		case <-time.After(150 * time.Millisecond):
		}
	}
}

// probePort returns true when host:port accepts a TCP connection within
// the given timeout. Used both to detect an existing receiver and to
// wait for a freshly-spawned one to come up.
func probePort(host string, port int, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// otelNoticeMarkerPath returns the path of the one-time notice marker file.
func otelNoticeMarkerPath(projectDir string) string {
	return filepath.Join(projectDir, ".htmlgraph", ".otel-notice-shown")
}

// MaybeShowOtelNotice prints a one-time notice to STDERR on first launch
// explaining that HtmlGraph captures Claude Code telemetry via OTel.
// Subsequent launches are silent (a marker file records that the notice
// has been shown). Safe to call when .htmlgraph/ doesn't exist — it
// simply returns without creating the directory or printing anything.
func MaybeShowOtelNotice(projectDir string) {
	if projectDir == "" {
		return
	}
	// Respect explicit opt-out — no need to explain what we're not doing.
	if isExplicitlyDisabled(os.Getenv("HTMLGRAPH_OTEL_ENABLED")) {
		return
	}
	// Only print when .htmlgraph/ already exists — don't create it just
	// to write the marker.
	htmlgraphDir := filepath.Join(projectDir, ".htmlgraph")
	if _, err := os.Stat(htmlgraphDir); os.IsNotExist(err) {
		return
	}
	markerPath := otelNoticeMarkerPath(projectDir)
	if _, err := os.Stat(markerPath); err == nil {
		return // notice already shown on a previous launch
	}

	cfg := otelreceiver.LoadConfigFromEnv("", projectDir)
	host := cfg.BindHost
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	port := cfg.HTTPPort
	if port == 0 {
		port = 4318
	}
	endpoint := "http://" + host + ":" + strconv.Itoa(port)

	notice := strings.Join([]string{
		"",
		"  htmlgraph: OTel telemetry is on (first-launch notice)",
		"  -------------------------------------------------------",
		"  HtmlGraph auto-captures Claude Code activity via OpenTelemetry:",
		"    tool calls, prompts, costs, token usage, and latencies.",
		"",
		"  Data stays 100% local — exported to " + endpoint + ",",
		"  stored in .htmlgraph/htmlgraph.db.",
		"",
		"  Powers: activity feed · per-turn cost badges · span timeline",
		"  Opt out: set HTMLGRAPH_OTEL_ENABLED=0 before launching.",
		"",
	}, "\n")
	fmt.Fprint(os.Stderr, notice)

	// Write marker so the notice doesn't repeat. Ignore errors — if the
	// write fails, re-showing the notice next launch is acceptable.
	_ = os.WriteFile(markerPath, []byte("shown\n"), 0o644)
}

// checkServeLock reads the per-project serve lock file and checks whether
// the PID it contains refers to a live process.
//
// Returns (skipSpawn=true, stale=false) when the lock exists and is alive.
// Returns (skipSpawn=false, stale=true) when the lock exists but the PID is dead.
// Returns (skipSpawn=false, stale=false) when the lock does not exist.
func checkServeLock(projectDir string) (skipSpawn, stale bool) {
	data, err := os.ReadFile(serveLockPath(projectDir))
	if err != nil {
		return false, false // no lock file
	}
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		return false, true // malformed — treat as stale
	}
	// kill -0 checks process existence without sending a signal.
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false, true // can't find process — stale
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return false, true // process not alive — stale
	}
	return true, false // process alive — skip spawn
}

// writeServeLock writes the current process PID to the per-project serve
// lock file. Called by `htmlgraph serve` on startup so concurrent launchers
// can detect a live serve process and skip spawning a duplicate.
// The write is best-effort — errors are silently ignored because a missing
// lock file causes a harmless duplicate-spawn attempt (which then fails to
// bind and exits cleanly).
func writeServeLock(projectDir string) {
	lockPath := serveLockPath(projectDir)
	_ = os.WriteFile(lockPath, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o644)
}

// removeServeLock removes the per-project serve lock file on graceful
// shutdown so subsequent launcher invocations don't see a stale lock.
func removeServeLock(projectDir string) {
	_ = os.Remove(serveLockPath(projectDir))
}

// debugLog writes a message to stderr only when HTMLGRAPH_DEBUG is set.
// Used for low-level operational tracing that should not appear in normal output.
func debugLog(msg string) {
	if os.Getenv("HTMLGRAPH_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "htmlgraph [debug]: %s\n", msg)
	}
}

// spawnDetachedServe starts `htmlgraph serve` in a new process group so
// it survives the launcher's exit and keeps serving the dashboard (and
// the OTel receiver) after claude terminates. Output redirects to
// .htmlgraph/logs/serve-auto.log.
//
// Uses os.Executable() for the binary path so the spawned server is
// the SAME version as the launcher — prevents version skew when the
// user has multiple htmlgraph builds on PATH (dev vs released).
func spawnDetachedServe(projectDir string) error {
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve self path: %w", err)
	}
	logDir := filepath.Join(projectDir, ".htmlgraph", "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}
	logPath := filepath.Join(logDir, "serve-auto.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open log %s: %w", logPath, err)
	}

	cmd := exec.Command(binPath, "serve")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	cmd.Dir = projectDir
	// Detach from the launcher's process group so the server survives
	// our exit. macOS and Linux both accept Setpgid via SysProcAttr.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// Inherit env including HTMLGRAPH_OTEL_ENABLED so the child's
	// serve_child turns the receiver on.
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("spawn serve: %w", err)
	}
	// Don't Wait — let it run. Log file close happens implicitly at
	// process exit; our close is fine to defer since the child has its
	// own fd after Start().
	_ = logFile.Close()
	return nil
}
