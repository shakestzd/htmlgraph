package db_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/shakestzd/wipnote/internal/db"
)

// migrationCallRecorder wraps db.MigrationHook and records every call.
type migrationCallRecorder struct {
	mu    sync.Mutex
	calls []string
}

func (r *migrationCallRecorder) Record(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, name)
}

func (r *migrationCallRecorder) Calls() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.calls))
	copy(out, r.calls)
	return out
}

// TestOpenReadOnly_NoDDL verifies that OpenReadOnly returns a *sql.DB that:
//   - Opens successfully when the database file already exists.
//   - Rejects DDL statements (CREATE TABLE, ALTER TABLE, DROP TABLE) — SQLite's
//     mode=ro enforces this at the engine level.
//   - Allows read queries (SELECT).
func TestOpenReadOnly_NoDDL(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Seed: create a valid wipnote DB first so read-only open has something to open.
	seed, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("seed Open: %v", err)
	}
	seed.Close()

	// Now open read-only.
	rodb, err := db.OpenReadOnly(dbPath)
	if err != nil {
		t.Fatalf("OpenReadOnly: %v", err)
	}
	defer rodb.Close()

	// READ must succeed.
	rows, err := rodb.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		t.Fatalf("SELECT on read-only db: %v", err)
	}
	rows.Close()

	// DDL must be rejected.
	// Note: DROP TABLE IF EXISTS on a non-existent table is a no-op in SQLite's
	// read-only mode, so we test DROP on an existing table (sessions) instead.
	ddlCases := []string{
		`CREATE TABLE IF NOT EXISTS ddl_test_probe (id TEXT PRIMARY KEY)`,
		`ALTER TABLE sessions ADD COLUMN zzz_probe TEXT`,
		`DROP TABLE sessions`,
	}
	for _, stmt := range ddlCases {
		_, execErr := rodb.Exec(stmt)
		if execErr == nil {
			t.Errorf("expected DDL to fail on read-only DB; stmt: %.60s", stmt)
		} else {
			// Verify the error is a read-only rejection (not an unrelated error).
			msg := strings.ToLower(execErr.Error())
			if !strings.Contains(msg, "readonly") && !strings.Contains(msg, "read-only") &&
				!strings.Contains(msg, "read only") && !strings.Contains(msg, "attempt to write") &&
				!strings.Contains(msg, "sqlite_readonly") {
				t.Errorf("DDL error unexpected type for stmt %.60s: %v", stmt, execErr)
			}
		}
	}
}

// TestOpenReadOnly_NonExistentFile verifies that OpenReadOnly returns an error
// when the database file does not exist (mode=ro must not create the file).
func TestOpenReadOnly_NonExistentFile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "nonexistent.db")

	_, err := db.OpenReadOnly(dbPath)
	if err == nil {
		t.Fatal("expected OpenReadOnly to fail on non-existent file; got nil error")
	}

	// Confirm the file was NOT created.
	if _, statErr := os.Stat(dbPath); statErr == nil {
		t.Error("OpenReadOnly must not create the database file in read-only mode")
	}
}

// TestOpenWritable_NoMigrations verifies that OpenWritable applies connection
// pragmas but does NOT invoke any migration hooks (no schema creation or alter
// table migrations).
func TestOpenWritable_NoMigrations(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Seed: create a valid wipnote DB first.
	seed, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("seed Open: %v", err)
	}
	seed.Close()

	// Track what migration hooks fire.
	recorder := &migrationCallRecorder{}
	db.SetMigrationObserver(recorder.Record)
	defer db.SetMigrationObserver(nil)

	writable, err := db.OpenWritable(dbPath)
	if err != nil {
		t.Fatalf("OpenWritable: %v", err)
	}
	defer writable.Close()

	// No migration calls should have been made.
	calls := recorder.Calls()
	if len(calls) != 0 {
		t.Errorf("OpenWritable called migration hooks: %v", calls)
	}

	// Normal read/write must work.
	_, err = writable.Exec(`INSERT OR IGNORE INTO metadata (key, value) VALUES ('test_key', 'test_val')`)
	if err != nil {
		t.Errorf("INSERT on OpenWritable db: %v", err)
	}

	var val string
	row := writable.QueryRow(`SELECT value FROM metadata WHERE key = 'test_key'`)
	if err := row.Scan(&val); err != nil {
		t.Errorf("SELECT after INSERT on OpenWritable db: %v", err)
	}
	if val != "test_val" {
		t.Errorf("SELECT value = %q, want %q", val, "test_val")
	}
}

// TestOpenMigrated_RunsMigrations verifies that Open (the migrated writable mode)
// does invoke migration hooks.
func TestOpenMigrated_RunsMigrations(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	recorder := &migrationCallRecorder{}
	db.SetMigrationObserver(recorder.Record)
	defer db.SetMigrationObserver(nil)

	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Open (migrated): %v", err)
	}
	defer database.Close()

	// Migration hooks should have fired.
	calls := recorder.Calls()
	if len(calls) == 0 {
		t.Error("Open (migrated) expected to invoke migration hooks, got none")
	}
}

// TestOpenReadOnly_PromptClose verifies that read-only paths close rows
// promptly and do not hold long-lived read transactions that block writers.
func TestOpenReadOnly_PromptClose(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	seed, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("seed Open: %v", err)
	}
	seed.Close()

	rodb, err := db.OpenReadOnly(dbPath)
	if err != nil {
		t.Fatalf("OpenReadOnly: %v", err)
	}
	defer rodb.Close()

	// Open rows but close them promptly — writer must not be blocked.
	rows, err := rodb.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	// Consume and close rows promptly.
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			t.Fatalf("Scan: %v", err)
		}
	}
	rows.Close()

	// Now a writer should be able to open and write without blocking.
	writer, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Open (writer) after read: %v", err)
	}
	defer writer.Close()

	_, err = writer.Exec(`INSERT OR IGNORE INTO metadata (key, value) VALUES ('prompt_close_test', '1')`)
	if err != nil {
		t.Errorf("writer INSERT after reader closed rows: %v", err)
	}
}
