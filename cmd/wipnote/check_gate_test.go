package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	dbpkg "github.com/shakestzd/wipnote/internal/db"
	"github.com/shakestzd/wipnote/internal/storage"
)

func setupGateTestProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{
		".wipnote/features",
		".wipnote/bugs",
		".wipnote/spikes",
		"plugin/config",
	} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/gatetest\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "plugin", "config", "quality-gate-flake-allowlist.json"), []byte(`[
  {
    "id": "tmp-noexec",
    "match_all": ["/tmp/", "permission denied"],
    "justification": "Test fixture justification"
  }
]`), 0o644); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
	for _, dir := range []string{".gotmp-exec", ".gocache"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	t.Setenv("GOTMPDIR", filepath.Join(root, ".gotmp-exec"))
	t.Setenv("GOCACHE", filepath.Join(root, ".gocache"))
	return root
}

func openGateTestDB(t *testing.T, projectRoot string) *sql.DB {
	t.Helper()
	dbPath, err := storage.CanonicalDBPath(projectRoot)
	if err != nil {
		t.Fatalf("CanonicalDBPath: %v", err)
	}
	if err := storage.EnsureDBDir(dbPath); err != nil {
		t.Fatalf("EnsureDBDir: %v", err)
	}
	database, err := dbpkg.Open(dbPath)
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	return database
}

func TestRunSessionGate_WritesSessionLocalRecord(t *testing.T) {
	projectRoot := setupGateTestProject(t)
	result, err := runSessionGate(projectRoot, "sess-gate-pass", "", "check", os.Stdout, os.Stderr)
	if err != nil {
		t.Fatalf("runSessionGate: %v", err)
	}
	if !result.Passed {
		t.Fatal("expected passing gate")
	}

	database := openGateTestDB(t, projectRoot)
	defer database.Close()

	record, err := dbpkg.LatestGateRecordForSession(database, "sess-gate-pass")
	if err != nil {
		t.Fatalf("LatestGateRecordForSession: %v", err)
	}
	if record == nil {
		t.Fatal("expected gate record")
	}
	if record.Status != "pass" {
		t.Fatalf("status = %q, want pass", record.Status)
	}
	if record.ProjectType != "go" {
		t.Fatalf("project type = %q, want go", record.ProjectType)
	}
	if !record.SignatureValid() {
		t.Fatal("expected valid signature")
	}
	if got, want := record.Source, "check"; got != want {
		t.Fatalf("source = %q, want %q", got, want)
	}
}

func TestLoadGateAllowlist_RequiresJustification(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, "plugin", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "plugin", "config", "quality-gate-flake-allowlist.json"), []byte(`[
  {
    "id": "tmp-noexec",
    "match_all": ["/tmp/", "permission denied"],
    "justification": ""
  }
]`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := loadGateAllowlist(projectRoot)
	if err == nil {
		t.Fatal("expected missing justification to fail")
	}
	if !strings.Contains(err.Error(), "missing justification") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckCompletionGateRecord_RequiresCurrentSessionRecord(t *testing.T) {
	projectRoot := setupGateTestProject(t)
	database := openGateTestDB(t, projectRoot)
	defer database.Close()

	if _, err := database.Exec(`INSERT OR REPLACE INTO feature_files (id, feature_id, file_path, operation, session_id) VALUES (?, ?, ?, ?, ?)`,
		"ff-1", "feat-gate", "main.go", "write", "sess-prev"); err != nil {
		t.Fatalf("insert feature file: %v", err)
	}

	err := checkCompletionGateRecord(database, projectRoot, "sess-current", "feat-gate")
	if err == nil {
		t.Fatal("expected completion gate refusal without current-session record")
	}
	if !strings.Contains(err.Error(), "wipnote check --gate") {
		t.Fatalf("expected remediation command, got: %v", err)
	}
}

func TestCheckCompletionGateRecord_AcceptsMatchingSessionAfterRecheck(t *testing.T) {
	projectRoot := setupGateTestProject(t)
	database := openGateTestDB(t, projectRoot)
	defer database.Close()

	if _, err := database.Exec(`INSERT OR REPLACE INTO feature_files (id, feature_id, file_path, operation, session_id) VALUES (?, ?, ?, ?, ?)`,
		"ff-2", "feat-gate", "main.go", "write", "sess-gate-ok"); err != nil {
		t.Fatalf("insert feature file: %v", err)
	}

	initial, err := runSessionGate(projectRoot, "sess-gate-ok", "feat-gate", "check", os.Stdout, os.Stderr)
	if err != nil {
		t.Fatalf("initial runSessionGate: %v", err)
	}
	if !initial.Passed || initial.Record == nil {
		t.Fatalf("expected initial passing record, got %+v", initial)
	}

	if err := checkCompletionGateRecord(database, projectRoot, "sess-gate-ok", "feat-gate"); err != nil {
		t.Fatalf("expected matching gate record to pass, got: %v", err)
	}

	count, err := dbpkg.CountGateRecords(database, "sess-gate-ok")
	if err != nil {
		t.Fatalf("CountGateRecords: %v", err)
	}
	if count < 2 {
		t.Fatalf("expected recheck to write a second gate record, got %d", count)
	}
}
