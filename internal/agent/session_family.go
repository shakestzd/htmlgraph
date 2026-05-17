package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// sessionFamilyIndex is the JSON structure stored in
// .wipnote/session-families.json. It maps session_id -> session_family_id for
// all active/recent sessions in a project.
//
// This replaces the last-writer-wins .active-session for projects running
// parallel root sessions: each entry coexists without clobbering the others.
type sessionFamilyIndex struct {
	Families map[string]string `json:"families"` // session_id -> family_id
}

// sessionFamilyMu serializes writes to session-families.json within a process.
// Cross-process safety relies on the write-tmp-rename atomic update pattern.
var sessionFamilyMu sync.Mutex

// familyIndexPath returns the path to the project's session family index file.
func familyIndexPath(projectDir string) string {
	return filepath.Join(projectDir, ".wipnote", "session-families.json")
}

// RegisterSessionFamily records sessionID -> familyID in the project's family
// index. Multiple sessions may share the same familyID (resumed continuations).
// The write is atomic (temp+rename) so concurrent processes cannot corrupt it.
func RegisterSessionFamily(projectDir, sessionID, familyID string) error {
	sessionFamilyMu.Lock()
	defer sessionFamilyMu.Unlock()

	idx := readFamilyIndexLocked(projectDir)
	if idx.Families == nil {
		idx.Families = make(map[string]string)
	}
	idx.Families[sessionID] = familyID
	return writeFamilyIndexLocked(projectDir, idx)
}

// ReadSessionFamilyIndex reads the project's family index. Returns an empty
// index (not an error) when the file does not exist.
func ReadSessionFamilyIndex(projectDir string) (map[string]string, error) {
	idx := readFamilyIndexLocked(projectDir)
	if idx.Families == nil {
		return map[string]string{}, nil
	}
	return idx.Families, nil
}

// readFamilyIndexLocked reads the family index (no lock acquired — caller holds lock or reads once).
func readFamilyIndexLocked(projectDir string) sessionFamilyIndex {
	path := familyIndexPath(projectDir)
	b, err := os.ReadFile(path)
	if err != nil {
		return sessionFamilyIndex{}
	}
	var idx sessionFamilyIndex
	if err := json.Unmarshal(b, &idx); err != nil {
		return sessionFamilyIndex{}
	}
	return idx
}

// writeFamilyIndexLocked atomically writes the family index.
func writeFamilyIndexLocked(projectDir string, idx sessionFamilyIndex) error {
	b, err := json.Marshal(idx)
	if err != nil {
		return err
	}
	dir := filepath.Join(projectDir, ".wipnote")
	tmp, err := os.CreateTemp(dir, ".session-families.tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	_ = os.Chmod(tmpPath, 0o644)
	return os.Rename(tmpPath, familyIndexPath(projectDir))
}

// SessionStateFile is the per-session state stored in
// .wipnote/sessions/<session_id>/state.json. It contains the harness-neutral
// session identity and family linkage, allowing each session to have its own
// fallback without the last-writer-wins collisions of .active-session.
type SessionStateFile struct {
	SessionID       string `json:"session_id"`
	AgentID         string `json:"agent_id"`
	SessionFamilyID string `json:"session_family_id"`
	Timestamp       int64  `json:"timestamp"`
}

// sessionStatePath returns the per-session state file path.
func sessionStatePath(projectDir, sessionID string) string {
	return filepath.Join(projectDir, ".wipnote", "sessions", sessionID, "state.json")
}

// WriteSessionState writes per-session state to the session-scoped directory.
// Unlike .active-session, each session has its own file so parallel sessions
// cannot overwrite each other.
func WriteSessionState(projectDir, sessionID, agentID, familyID string) error {
	if projectDir == "" || sessionID == "" {
		return nil
	}
	sessDir := filepath.Join(projectDir, ".wipnote", "sessions", sessionID)
	if err := os.MkdirAll(sessDir, 0o755); err != nil {
		return err
	}
	state := SessionStateFile{
		SessionID:       sessionID,
		AgentID:         agentID,
		SessionFamilyID: familyID,
		Timestamp:       time.Now().Unix(),
	}
	b, err := json.Marshal(state)
	if err != nil {
		return err
	}
	path := sessionStatePath(projectDir, sessionID)
	tmp, err := os.CreateTemp(sessDir, ".state.tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	_ = os.Chmod(tmpPath, 0o644)
	return os.Rename(tmpPath, path)
}

// ReadSessionState reads the per-session state for the given session. Returns
// (nil, nil) when the file does not exist (session predates per-session state).
func ReadSessionState(projectDir, sessionID string) (*SessionStateFile, error) {
	path := sessionStatePath(projectDir, sessionID)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var state SessionStateFile
	if err := json.Unmarshal(b, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// ResolvePrimarySessionID returns the session ID for a project using the
// per-session state file when present (preferred), falling back to the legacy
// .active-session file for backward compatibility.
//
// Callers should prefer per-session state when a concrete sessionID is known;
// this function is for callers that need to discover the "current" session
// without an explicit ID (e.g. CLI commands running outside a hook).
func ResolvePrimarySessionID(projectDir, sessionID string) string {
	// Per-session state is primary when present.
	if sessionID != "" {
		if st, err := ReadSessionState(projectDir, sessionID); err == nil && st != nil {
			return st.SessionID
		}
	}
	// Fall back to legacy .active-session (backward compat).
	return readActiveSessionID(projectDir)
}
