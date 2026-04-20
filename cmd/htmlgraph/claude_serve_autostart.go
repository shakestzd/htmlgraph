package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	otelreceiver "github.com/shakestzd/htmlgraph/internal/otel/receiver"
)

// ensureServeForOtel checks whether the OTLP HTTP receiver is already
// listening, and spawns a detached `htmlgraph serve` if not. Called from
// launchClaude before exec'ing claude so that OTel signals from the
// child process have somewhere to land.
//
// Gating:
//   - When HTMLGRAPH_OTEL_ENABLED is not set, the launcher hasn't asked
//     for OTel, so nothing to guarantee — return nil immediately.
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
// The spawned process inherits HTMLGRAPH_OTEL_ENABLED so its child's
// serve_child wiring turns on the receiver. Stdout/stderr go to a log
// file under .htmlgraph/logs so the orphaned server doesn't pollute
// the user's terminal.
func ensureServeForOtel(projectDir string) {
	if !isTruthy(os.Getenv("HTMLGRAPH_OTEL_ENABLED")) {
		return
	}
	cfg := otelreceiver.LoadConfigFromEnv("")
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
