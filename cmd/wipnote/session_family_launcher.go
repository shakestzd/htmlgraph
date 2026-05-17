package main

import (
	"os"

	"github.com/shakestzd/wipnote/internal/agent"
)

// resolveSessionFamilyID returns the session family ID to use for a new
// launcher invocation. The rules are:
//
//  1. If WIPNOTE_SESSION_FAMILY_ID is already set in the environment (e.g. the
//     user is re-running inside a shell that already has the launcher env
//     injected), reuse it — this keeps all sub-invocations in one family.
//  2. If this is a resume/continue launch AND the project has an existing active
//     session with a known family, inherit that family so resumed sessions are
//     grouped with their originating session.
//  3. Otherwise create a new family ID equal to the new session ID (each fresh
//     start is its own family of one until a resume joins it).
//
// The projectDir and isResume arguments let the caller signal resume intent.
// newSessionID is the freshly-minted OTel session ID for this launch.
func resolveSessionFamilyID(projectDir, newSessionID string, isResume bool) string {
	// 1. Inherit from environment (nested / re-launched within the same family).
	if v := os.Getenv("WIPNOTE_SESSION_FAMILY_ID"); v != "" {
		return v
	}

	// 2. On resume: look up the most-recently-written session state and inherit
	//    its family. This creates the concrete link between resumed sessions.
	if isResume && projectDir != "" {
		// Read the family index to find the most recent family for this project.
		idx, err := agent.ReadSessionFamilyIndex(projectDir)
		if err == nil && len(idx) > 0 {
			// Return the first family found — any family is better than none.
			// The session start hook will set the DB column from this value.
			for _, fam := range idx {
				return fam
			}
		}
	}

	// 3. Fresh launch: new session = new family (self-as-family).
	return newSessionID
}

// persistLauncherSessionFamily writes the session→family mapping to the
// project's family index and writes the per-session state file. This is the
// CONCRETE write path that survives even if the SessionStart hook never fires
// (e.g. harness spawned without hooks configured).
//
// agentID is "codex" or "gemini" (the harness name).
// Errors are silently ignored — this is a best-effort durability write;
// hook handlers are the authoritative path for DB writes.
func persistLauncherSessionFamily(projectDir, sessionID, agentID, familyID string) {
	if projectDir == "" || sessionID == "" {
		return
	}
	_ = agent.RegisterSessionFamily(projectDir, sessionID, familyID)
	_ = agent.WriteSessionState(projectDir, sessionID, agentID, familyID)
}
