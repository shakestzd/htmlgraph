package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

// runHarnessWithCleanup runs the harness child process under a signal
// handler that intercepts SIGINT and SIGTERM, ensuring cleanup runs
// before the launcher exits. This is required because the standard
// runtime path — `defer cleanup()` plus `os.Exit(child.ExitCode())` on
// non-zero return — bypasses cleanup in three cases:
//
//  1. Ctrl-C in the terminal: the signal hits the parent before c.Wait
//     returns; without a handler, Go's default behavior aborts the
//     process and skips defers.
//  2. `kill -INT <launcher-pid>` from outside the terminal: same
//     mechanism, no foreground-group propagation.
//  3. Existing logic on `*exec.ExitError`: was already fixed in
//     dc185548, but only covers the child's exit code path, not signal
//     interruption of the parent.
//
// Pattern:
//   - Install signal.Notify so the runtime no longer aborts on
//     SIGINT/SIGTERM.
//   - Start the child and Wait in a goroutine.
//   - select on sigCh and waitCh: on signal, forward it to the child
//     and wait for child exit; on normal exit, just proceed.
//   - Always call cleanup (idempotent via sync.Once anyway).
//   - On signal path, re-raise the same signal after reset so the
//     parent's exit code reflects normal signal-exit semantics
//     (128+signum on POSIX).
//   - On non-zero child exit code, os.Exit with that code (cleanup
//     already ran above).
//
// cleanup may be nil — if so, no cleanup is invoked but signal
// handling still runs.
func runHarnessWithCleanup(c *exec.Cmd, cleanup func()) error {
	var once sync.Once
	callCleanup := func() {
		once.Do(func() {
			if cleanup != nil {
				cleanup()
			}
		})
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	if err := c.Start(); err != nil {
		callCleanup()
		return fmt.Errorf("start harness: %w", err)
	}

	waitCh := make(chan error, 1)
	go func() { waitCh <- c.Wait() }()

	var sigReceived os.Signal
	select {
	case sigReceived = <-sigCh:
		// Forward the signal to the child so it exits gracefully.
		// In a normal terminal Ctrl-C, the child already received
		// SIGINT from the foreground group; the redundant Signal call
		// is harmless. For external `kill -TERM` against the launcher
		// only, this is the explicit forward.
		if c.Process != nil {
			_ = c.Process.Signal(sigReceived)
		}
		<-waitCh // child reaps
	case <-waitCh:
		// Child exited on its own; nothing to forward.
	}

	callCleanup()

	if sigReceived != nil {
		// Reset to default handler and re-raise so the parent exits
		// with conventional signal semantics (128+signum).
		signal.Reset(syscall.SIGINT, syscall.SIGTERM)
		if sysSig, ok := sigReceived.(syscall.Signal); ok {
			_ = syscall.Kill(os.Getpid(), sysSig)
		}
		// If the re-raise didn't terminate (rare), fall through.
		return nil
	}

	if c.ProcessState != nil && !c.ProcessState.Success() {
		os.Exit(c.ProcessState.ExitCode())
	}
	return nil
}
