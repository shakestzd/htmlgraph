package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// migrationObserver is an optional hook called by Open each time a migration
// step fires. It is used exclusively by tests to assert that OpenWritable does
// NOT trigger migrations. Production code should leave it nil.
var migrationObserver func(name string)

// SetMigrationObserver installs a migration observer for testing. Pass nil to
// remove a previously installed observer. This function is not concurrency-safe
// — install observers only at test setup before any Open call.
func SetMigrationObserver(fn func(name string)) {
	migrationObserver = fn
}

// notifyMigration calls the migration observer if one is installed.
func notifyMigration(name string) {
	if migrationObserver != nil {
		migrationObserver(name)
	}
}

// OpenReadOnly opens an existing wipnote SQLite database in read-only mode.
// It applies connection-level pragmas (busy_timeout, cache_size, etc.) but
// does NOT run any DDL, migrations, or normalisation writes.
//
// SQLite enforces read-only access at the engine level via mode=ro in the DSN;
// any attempt to execute DDL or DML will return SQLITE_READONLY.
//
// IMPORTANT — prompt close contract: read-only paths MUST close all sql.Rows,
// sql.Stmt, and sql.Tx values promptly after use. In DELETE journal mode a
// reader that holds an open shared-lock blocks the single writer from acquiring
// the reserved lock. Failure to close promptly can cause SQLITE_BUSY on the
// writer side. In WAL mode readers and writers do not block each other, but the
// prompt-close discipline must still be observed for portability.
//
// Returns an error if the database file does not exist (mode=ro never creates a
// new file).
func OpenReadOnly(dbPath string) (*sql.DB, error) {
	// Fail fast if the file doesn't exist — mode=ro should not create files.
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("OpenReadOnly: database file not found: %w", err)
	}

	// Build a read-only DSN.
	// mode=ro: SQLite engine-level read-only enforcement (no DDL, no DML).
	// _pragma=busy_timeout(5000): wait up to 5s for shared-lock acquisition.
	dsn := buildReadOnlyDSN(dbPath)
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("OpenReadOnly: sql.Open: %w", err)
	}

	// Apply read-compatible pragmas best-effort (read-only connections reject
	// write-requiring pragmas like journal_mode SET and foreign_keys SET).
	pragmas := buildReadOnlyPragmas()
	if err := applyReadOnlyPragmas(database, pragmas); err != nil {
		database.Close()
		return nil, fmt.Errorf("OpenReadOnly: apply pragmas: %w", err)
	}

	// Verify read access with a lightweight query.
	if _, err := database.Exec("SELECT 1"); err != nil {
		database.Close()
		return nil, fmt.Errorf("OpenReadOnly: smoke check failed: %w", err)
	}

	return database, nil
}

// OpenWritable opens an existing wipnote SQLite database for reading and
// writing. It applies all connection-level pragmas needed for normal operation
// but does NOT run schema creation (CreateAllTables / CreateAllIndexes) or any
// migration hooks.
//
// Use this mode when the schema is already known to be current (e.g. the
// database was previously initialised by Open) and you only need to read/write
// data rows. Callers that need schema creation or migrations must use Open
// instead.
//
// IMPORTANT — prompt close contract: same as OpenReadOnly. In DELETE journal
// mode, open transactions can block the writer. Close rows and transactions
// promptly.
//
// APPROVED CALLERS — every first-party Go callsite that opens a writable
// SQLite handle is enumerated in cmd/wipnote/sqlite_write_boundary_test.go
// (variable approvedWriteSites). Adding a new caller without updating that
// inventory fails the boundary test. Hook / indexer / OTLP-receiver paths
// MUST route writes through the slice-6 writer service (feat-f3bcbcef);
// do not add new direct OpenWritable callers in those locations.
func OpenWritable(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("OpenWritable: creating db directory: %w", err)
	}

	// Use the same busy_timeout DSN embedding as Open to prevent SQLITE_BUSY
	// on the very first connection before pragmas have been applied.
	isInMemory := strings.Contains(dbPath, ":memory:")
	dsn := dbPath
	if !isInMemory {
		dsn = dsn + "?_pragma=busy_timeout(5000)"
	}

	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("OpenWritable: sql.Open: %w", err)
	}

	// Apply all connection pragmas (same as Open) but do NOT call CreateAllTables,
	// CreateAllIndexes, or any migration helpers.
	if err := ApplyPragmas(database, BuildPragmas(dbPath)); err != nil {
		database.Close()
		return nil, fmt.Errorf("OpenWritable: applying pragmas: %w", err)
	}

	return database, nil
}

// OpenReadOnlyMigrated guarantees the database at dbPath exists and is at the
// current schema version, then returns a read-only handle for the actual
// query work. It mirrors the serve_child.go topology (writable Open FIRST so
// schema/migrations are applied — mode=ro never creates a file and never
// migrates — THEN a separate read-only handle for the long read path).
//
// bug-7dbaf552 / roborev followup: read-only CLI surfaces (`wipnote query`,
// `wipnote lineage`) were switched to OpenReadOnly for contention safety, but
// that dropped the migrate-on-open guarantee that the prior writable open
// provided — a fresh or schema-behind workspace would fail before the read
// even ran. This helper restores BOTH guarantees: Open here is the Fix-1
// RetryOnBusy-wrapped migration path, so the brief bootstrap open is itself
// resilient to a transient SQLITE_BUSY; the bootstrap handle is closed
// immediately so it never holds the writer lock during the (potentially long)
// read path that follows on the returned read-only handle.
func OpenReadOnlyMigrated(dbPath string) (*sql.DB, error) {
	boot, err := Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("bootstrap (schema/migrations): %w", err)
	}
	if cerr := boot.Close(); cerr != nil {
		return nil, fmt.Errorf("close bootstrap handle: %w", cerr)
	}
	return OpenReadOnly(dbPath)
}

// buildReadOnlyDSN builds a URI DSN for SQLite read-only access.
func buildReadOnlyDSN(dbPath string) string {
	if strings.Contains(dbPath, ":memory:") {
		// In-memory databases cannot use file URI mode; keep as-is.
		return dbPath
	}
	// Use file: URI with mode=ro. The _pragma parameter applies busy_timeout
	// at first connection open so the first shared-lock acquisition is protected.
	return "file:" + dbPath + "?mode=ro&_pragma=busy_timeout(5000)"
}

// buildReadOnlyPragmas returns the pragma subset appropriate for a read-only
// connection. journal_mode SET and foreign_keys are excluded — read-only
// connections cannot change those and the attempt returns SQLITE_READONLY.
func buildReadOnlyPragmas() map[string]string {
	return map[string]string{
		"cache_size": "-64000",
		"temp_store": "MEMORY",
	}
}

// applyReadOnlyPragmas sets a restricted set of pragmas on a read-only
// connection. All pragmas are applied best-effort; failures are silently
// ignored because read-only connections may reject certain PRAGMA writes.
func applyReadOnlyPragmas(database *sql.DB, pragmas map[string]string) error {
	for pragma, value := range pragmas {
		if _, err := database.Exec(fmt.Sprintf("PRAGMA %s = %s", pragma, value)); err != nil {
			// Best-effort: skip. Read-only connections may reject PRAGMA writes.
			_ = err
		}
	}
	return nil
}
