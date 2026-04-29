// Package collector provides the CollectorLifecycle interface and its
// ProcessCollector implementation for spawning, monitoring, and cleaning up
// htmlgraph otel-collect child processes.
//
// Future launchers (Codex, Gemini) call Spawn directly without duplicating
// retry/watchdog/cleanup machinery.
package collector

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// Lifecycle is the minimal interface for spawning a per-session OTel collector.
type Lifecycle interface {
	// Spawn starts an otel-collect process for the given session and returns
	// the port it is listening on plus a cleanup function. The cleanup function
	// stops the watchdog goroutine, SIGTERMs the process (waits up to 3s, then
	// SIGKILLs), and removes the .collector-pid file.
	Spawn(binPath, sessionID, projectDir string) (port int, cleanup func(), err error)
}

// ProcessCollectorOpts configures a ProcessCollector.
type ProcessCollectorOpts struct {
	// Stderr is where warning/info/FATAL lines are written. Defaults to os.Stderr.
	Stderr io.Writer

	// StrictMode is reserved for callers that want Spawn errors to be fatal;
	// the ProcessCollector itself does not call os.Exit — that decision belongs
	// to the caller.
	StrictMode bool

	// SpawnFn overrides the default spawn function. Nil means use DefaultSpawnFn.
	// Primarily for tests.
	SpawnFn func(binPath, sessionID, projectDir string) (int, *os.Process, error)

	// WatchdogIntervalEnv is the env-var name used to override the watchdog
	// poll interval. Empty string defaults to "HTMLGRAPH_OTEL_WATCHDOG_INTERVAL".
	WatchdogIntervalEnv string
}

// ProcessCollector implements Lifecycle by managing a real os.Process.
type ProcessCollector struct {
	opts ProcessCollectorOpts
}

// NewProcessCollector returns a new ProcessCollector configured by opts.
func NewProcessCollector(opts ProcessCollectorOpts) *ProcessCollector {
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.WatchdogIntervalEnv == "" {
		opts.WatchdogIntervalEnv = "HTMLGRAPH_OTEL_WATCHDOG_INTERVAL"
	}
	return &ProcessCollector{opts: opts}
}

// Spawn starts the collector, retries up to 3 times with backoff, writes the
// PID file, starts the watchdog, and returns the port and cleanup func.
// On failure it writes a FATAL line to Stderr and returns a non-nil error.
func (c *ProcessCollector) Spawn(binPath, sessionID, projectDir string) (int, func(), error) {
	spawnFn := c.opts.SpawnFn
	if spawnFn == nil {
		spawnFn = DefaultSpawnFn
	}

	port, proc, attempts, err := RetrySpawn(binPath, sessionID, projectDir, 3, spawnFn, c.opts.Stderr)
	if err != nil {
		fmt.Fprintf(c.opts.Stderr, "htmlgraph: FATAL: collector spawn failed after %d attempts: %v\n", attempts, err)
		return 0, nil, err
	}

	WriteCollectorPID(projectDir, sessionID, proc.Pid)
	stopWatchdog := c.startWatchdog(proc, binPath, sessionID, projectDir)
	baseCleanup := RegisterCleanup(proc, projectDir, sessionID)
	cleanup := func() { stopWatchdog(); baseCleanup() }

	return port, cleanup, nil
}

// DefaultSpawnFn starts an otel-collect child process, waits for its handshake
// line ("htmlgraph-otel-ready port=<N>"), and returns the port and process.
// The child is started in its own process group (Setpgid) so it can be
// independently signalled.
func DefaultSpawnFn(binPath, sessionID, projectDir string) (int, *os.Process, error) {
	cmd := exec.Command(binPath, "otel-collect",
		"--session-id", sessionID,
		"--project-dir", projectDir,
		"--listen", "127.0.0.1:0",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, nil, fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return 0, nil, fmt.Errorf("start otel-collect: %w", err)
	}

	port, err := readHandshake(bufio.NewScanner(stdout))
	if err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return 0, nil, err
	}
	return port, cmd.Process, nil
}

// readHandshake scans stdout for the handshake line within 3s.
func readHandshake(scanner *bufio.Scanner) (int, error) {
	type result struct {
		port int
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			var p int
			if _, err := fmt.Sscanf(line, "htmlgraph-otel-ready port=%d", &p); err == nil {
				ch <- result{port: p}
				return
			}
		}
		ch <- result{err: fmt.Errorf("otel-collect: handshake not found (stdout closed)")}
	}()

	select {
	case r := <-ch:
		return r.port, r.err
	case <-time.After(3 * time.Second):
		return 0, fmt.Errorf("otel-collect: handshake timeout (3s)")
	}
}

// RetrySpawn attempts to spawn the collector up to maxAttempts times.
// Backoff delays between attempts: 100ms, 300ms, 700ms.
// Writes a warning line to warnW after each non-final failure.
// Returns port, process, number of attempts, and any final error.
func RetrySpawn(
	binPath, sessionID, projectDir string,
	maxAttempts int,
	spawnFn func(string, string, string) (int, *os.Process, error),
	warnW io.Writer,
) (int, *os.Process, int, error) {
	backoff := []time.Duration{100 * time.Millisecond, 300 * time.Millisecond, 700 * time.Millisecond}
	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		port, proc, err := spawnFn(binPath, sessionID, projectDir)
		if err == nil {
			return port, proc, i + 1, nil
		}
		lastErr = err
		if i < maxAttempts-1 {
			fmt.Fprintf(warnW, "htmlgraph: warning: collector spawn attempt %d/%d failed: %v\n", i+1, maxAttempts, err)
			if i < len(backoff) {
				time.Sleep(backoff[i])
			}
		}
	}
	return 0, nil, maxAttempts, lastErr
}

// WatchdogInterval returns the polling interval for the collector watchdog.
// The env var name is configurable.
func WatchdogInterval(envKey string) time.Duration {
	if v := os.Getenv(envKey); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 15 * time.Second
}

// StartWatchdog launches a goroutine that polls the collector process every
// WatchdogInterval(envKey). On process death it calls RetrySpawn and updates
// the current process. Returns a stop func that terminates the goroutine.
func StartWatchdog(
	initialProc *os.Process,
	binPath, sessionID, projectDir string,
	warnW io.Writer,
	spawnFn func(string, string, string) (int, *os.Process, error),
	envKey string,
) func() {
	done := make(chan struct{})

	go func() {
		ticker := time.NewTicker(WatchdogInterval(envKey))
		defer ticker.Stop()
		currentProc := initialProc

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := currentProc.Signal(syscall.Signal(0)); err == nil {
					continue // process still alive
				}
				fmt.Fprintf(warnW, "htmlgraph: warning: collector died (pid=%d), respawning...\n", currentProc.Pid)
				port, newProc, _, spawnErr := RetrySpawn(binPath, sessionID, projectDir, 3, spawnFn, warnW)
				if spawnErr != nil {
					fmt.Fprintf(warnW, "htmlgraph: FATAL: collector respawn failed: %v\n", spawnErr)
					return
				}
				WriteCollectorPID(projectDir, sessionID, newProc.Pid)
				fmt.Fprintf(warnW, "htmlgraph: info: collector respawned (pid=%d port=%d)\n", newProc.Pid, port)
				currentProc = newProc
			}
		}
	}()

	return func() { close(done) }
}

// startWatchdog is the method form of StartWatchdog, using opts from the receiver.
func (c *ProcessCollector) startWatchdog(initialProc *os.Process, binPath, sessionID, projectDir string) func() {
	spawnFn := c.opts.SpawnFn
	if spawnFn == nil {
		spawnFn = DefaultSpawnFn
	}
	return StartWatchdog(initialProc, binPath, sessionID, projectDir, c.opts.Stderr, spawnFn, c.opts.WatchdogIntervalEnv)
}

// RegisterCleanup returns a cleanup function that SIGTERMs the process, waits
// up to 3s, then SIGKILLs, and removes the .collector-pid file.
func RegisterCleanup(proc *os.Process, projectDir, sessionID string) func() {
	go func() { _, _ = proc.Wait() }()

	return func() {
		_ = proc.Signal(syscall.SIGTERM)
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if err := proc.Signal(syscall.Signal(0)); err != nil {
				RemoveCollectorPID(projectDir, sessionID)
				return // process exited
			}
			time.Sleep(100 * time.Millisecond)
		}
		_ = proc.Kill()
		RemoveCollectorPID(projectDir, sessionID)
	}
}

// RemoveCollectorPID removes the .collector-pid file for a session.
// Best-effort: missing file or unreadable directory is not an error.
func RemoveCollectorPID(projectDir, sessionID string) {
	pidPath := filepath.Join(projectDir, ".htmlgraph", "sessions", sessionID, ".collector-pid")
	_ = os.Remove(pidPath)
}

// WriteCollectorPID writes the collector PID to the session directory.
// Best-effort: errors are silently ignored.
func WriteCollectorPID(projectDir, sessionID string, pid int) {
	sessDir := filepath.Join(projectDir, ".htmlgraph", "sessions", sessionID)
	_ = os.MkdirAll(sessDir, 0o755)
	pidPath := filepath.Join(sessDir, ".collector-pid")
	_ = os.WriteFile(pidPath, []byte(strconv.Itoa(pid)+"\n"), 0o644)
}
