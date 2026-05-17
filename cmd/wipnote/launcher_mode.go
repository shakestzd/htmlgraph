package main

import (
	"fmt"
	"os"

	"github.com/shakestzd/wipnote/internal/launcher/mode"
)

// LauncherModeResult is the computed mode object exposed to preflight paths.
// Callers can log or inspect it without changing launcher behavior.
type LauncherModeResult = mode.LauncherMode

// computeLauncherMode returns a LauncherMode for the given launcher invocation.
// worktreePath should be non-empty when running in an isolated git worktree,
// devPlugin when launched with --dev (in-tree plugin source), and
// generatedPort when a harness-generated tree is active.
//
// This is the non-behavior-changing wiring point for all launchers.
// Future slices will act on the returned value; this slice only computes and
// optionally logs it.
func computeLauncherMode(worktreePath string, devPlugin, generatedPort bool) LauncherModeResult {
	m := mode.Compute(worktreePath, false, devPlugin, generatedPort)
	if os.Getenv("WIPNOTE_DEBUG") != "" {
		fmt.Fprintf(os.Stderr,
			"wipnote [debug]: mode runtime=%s execution=%s plugin=%s dashboard=%s:%d\n",
			m.Runtime, m.Execution, m.Plugin, m.DashboardHost, m.DashboardPort,
		)
	}
	return m
}
