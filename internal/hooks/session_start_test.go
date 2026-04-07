package hooks

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/models"
)

func TestSessionStartStoresProjectDir(t *testing.T) {
	// Set up a temporary project directory with a .htmlgraph subdir.
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".htmlgraph"), 0o755); err != nil {
		t.Fatalf("mkdir .htmlgraph: %v", err)
	}

	// Open an in-memory SQLite database.
	database, err := db.Open(filepath.Join(projectDir, ".htmlgraph", "htmlgraph.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer database.Close()

	sessionID := "test-session-project-dir-001"
	// Set CWD to the temp projectDir so resolveWorktreeParentSession does not
	// accidentally read the real .active-session from the developer's worktree.
	event := &CloudEvent{SessionID: sessionID, CWD: projectDir}

	// Unset env vars that would override session ID or mark this as a subagent.
	t.Setenv("CLAUDE_SESSION_ID", "")
	t.Setenv("HTMLGRAPH_PARENT_SESSION", "")
	t.Setenv("HTMLGRAPH_NESTING_DEPTH", "")
	t.Setenv("CLAUDE_ENV_FILE", "") // prevent writing to a real env file

	_, err = SessionStart(event, database, projectDir)
	if err != nil {
		t.Fatalf("SessionStart: %v", err)
	}

	// Retrieve the session from DB and verify project_dir is stored.
	got, err := db.GetSession(database, sessionID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.ProjectDir != projectDir {
		t.Errorf("project_dir mismatch: got %q, want %q", got.ProjectDir, projectDir)
	}
}

func TestSessionStartActiveSessionContainsProjectDir(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".htmlgraph"), 0o755); err != nil {
		t.Fatalf("mkdir .htmlgraph: %v", err)
	}

	database, err := db.Open(filepath.Join(projectDir, ".htmlgraph", "htmlgraph.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer database.Close()

	sessionID := "test-session-active-file-001"
	// Set CWD to the temp projectDir so resolveWorktreeParentSession does not
	// accidentally read the real .active-session from the developer's worktree.
	event := &CloudEvent{SessionID: sessionID, CWD: projectDir}

	t.Setenv("CLAUDE_SESSION_ID", "")
	t.Setenv("HTMLGRAPH_PARENT_SESSION", "")
	t.Setenv("HTMLGRAPH_NESTING_DEPTH", "")
	t.Setenv("CLAUDE_ENV_FILE", "") // force fallback to .active-session

	_, err = SessionStart(event, database, projectDir)
	if err != nil {
		t.Fatalf("SessionStart: %v", err)
	}

	// .active-session should have been written (CLAUDE_ENV_FILE unset path).
	active := ReadActiveSession(projectDir)
	if active == nil {
		t.Fatal("ReadActiveSession returned nil — .active-session not written")
	}
	if active.ProjectDir != projectDir {
		t.Errorf(".active-session project_dir mismatch: got %q, want %q", active.ProjectDir, projectDir)
	}
}

// TestSessionStartWorktreeParentSessionIDPopulated verifies that when a
// subagent session is started with a known parent session ID (as
// resolveWorktreeParentSession would provide), the new session row gets
// parent_session_id set and is_subagent = true.
//
// We test the upsertSession path directly rather than going through
// resolveWorktreeParentSession (which requires a real git worktree) to keep
// the test hermetic.  The FK constraint previously caused INSERT OR IGNORE to
// silently drop the row when the parent session was absent from the test DB.
func TestSessionStartWorktreeParentSessionIDPopulated(t *testing.T) {
	// Set up the project directory.
	mainDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(mainDir, ".htmlgraph"), 0o755); err != nil {
		t.Fatalf("mkdir .htmlgraph: %v", err)
	}

	database, err := db.Open(filepath.Join(mainDir, ".htmlgraph", "htmlgraph.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer database.Close()

	// Insert the parent (outer YOLO) session so FK constraint is satisfied.
	parentSessionID := "parent-yolo-session-001"
	if err := db.InsertSession(database, &models.Session{
		SessionID:     parentSessionID,
		AgentAssigned: "claude-code",
		Status:        "active",
		CreatedAt:     time.Now().UTC(),
		ProjectDir:    mainDir,
	}); err != nil {
		t.Fatalf("InsertSession parent: %v", err)
	}

	// Write .active-session as the outer YOLO session would have done so that
	// ReadActiveSession can return the parent session ID.
	WriteActiveSession(parentSessionID, mainDir)

	// Verify ReadActiveSession round-trips correctly.
	as := ReadActiveSession(mainDir)
	if as == nil || as.SessionID != parentSessionID {
		t.Fatalf("ReadActiveSession: got %v, want session_id=%q", as, parentSessionID)
	}

	// Simulate what SessionStart does after resolveWorktreeParentSession
	// returns (parentSessionID, true): upsert the subagent session with
	// parent_session_id and is_subagent = true.
	subSessionID := "sub-worktree-session-001"
	if err := upsertSession(database, &models.Session{
		SessionID:       subSessionID,
		AgentAssigned:   "claude-code",
		Status:          "active",
		CreatedAt:       time.Now().UTC(),
		ProjectDir:      mainDir,
		ParentSessionID: parentSessionID,
		IsSubagent:      true,
	}); err != nil {
		t.Fatalf("upsertSession subagent: %v", err)
	}

	got, err := db.GetSession(database, subSessionID)
	if err != nil {
		t.Fatalf("GetSession subagent: %v", err)
	}
	if got.ParentSessionID != parentSessionID {
		t.Errorf("parent_session_id: got %q, want %q", got.ParentSessionID, parentSessionID)
	}
	if !got.IsSubagent {
		t.Error("is_subagent: got false, want true")
	}
}

func TestInsertAndGetSessionProjectDir(t *testing.T) {
	dir := t.TempDir()
	database, err := db.Open(filepath.Join(dir, "htmlgraph.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer database.Close()

	s := &models.Session{
		SessionID:     "sess-proj-dir-test",
		AgentAssigned: "test-agent",
		Status:        "active",
		CreatedAt:     time.Now().UTC(),
		ProjectDir:    "/home/user/myproject",
	}
	if err := db.InsertSession(database, s); err != nil {
		t.Fatalf("InsertSession: %v", err)
	}

	got, err := db.GetSession(database, s.SessionID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.ProjectDir != s.ProjectDir {
		t.Errorf("project_dir round-trip: got %q, want %q", got.ProjectDir, s.ProjectDir)
	}
}
