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
	event := &CloudEvent{SessionID: sessionID}

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
	event := &CloudEvent{SessionID: sessionID}

	t.Setenv("CLAUDE_SESSION_ID", "")
	t.Setenv("HTMLGRAPH_PARENT_SESSION", "")
	t.Setenv("HTMLGRAPH_NESTING_DEPTH", "")
	t.Setenv("CLAUDE_ENV_FILE", "") // force fallback to .active-session

	_, err = SessionStart(event, database, projectDir)
	if err != nil {
		t.Fatalf("SessionStart: %v", err)
	}

	// .active-session should have been written (CLAUDE_ENV_FILE unset path).
	active := readActiveSession(projectDir)
	if active == nil {
		t.Fatal("readActiveSession returned nil — .active-session not written")
	}
	if active.ProjectDir != projectDir {
		t.Errorf(".active-session project_dir mismatch: got %q, want %q", active.ProjectDir, projectDir)
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
