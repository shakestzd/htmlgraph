package main

import (
	"os/exec"
	"testing"
)

// TestRunHarnessWithCleanup_NormalExit asserts that cleanup runs and the
// function returns nil when the child exits successfully.
func TestRunHarnessWithCleanup_NormalExit(t *testing.T) {
	cleanupCalled := false
	cleanup := func() { cleanupCalled = true }

	c := exec.Command("/bin/sh", "-c", "exit 0")
	if err := runHarnessWithCleanup(c, cleanup); err != nil {
		t.Errorf("expected nil error on success, got: %v", err)
	}
	if !cleanupCalled {
		t.Error("cleanup was not invoked on normal exit")
	}
}

// TestRunHarnessWithCleanup_NilCleanup asserts that a nil cleanup is
// tolerated — the helper should still run the child and report success.
func TestRunHarnessWithCleanup_NilCleanup(t *testing.T) {
	c := exec.Command("/bin/sh", "-c", "exit 0")
	if err := runHarnessWithCleanup(c, nil); err != nil {
		t.Errorf("expected nil error on success with nil cleanup, got: %v", err)
	}
}

// TestRunHarnessWithCleanup_StartFailure asserts that cleanup runs and an
// error is returned when the child fails to start (binary not found).
func TestRunHarnessWithCleanup_StartFailure(t *testing.T) {
	cleanupCalled := false
	cleanup := func() { cleanupCalled = true }

	c := exec.Command("/this/binary/does/not/exist/htmlgraph-test")
	err := runHarnessWithCleanup(c, cleanup)
	if err == nil {
		t.Error("expected error on start failure, got nil")
	}
	if !cleanupCalled {
		t.Error("cleanup was not invoked when c.Start failed")
	}
}
