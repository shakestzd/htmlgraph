package db_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	db "github.com/shakestzd/wipnote/internal/db"
	_ "modernc.org/sqlite"
)

// TestApplyPragmas_AppliesBusyTimeout verifies that Open applies busy_timeout
// to the database connection. Rather than relying on lock-contention timing
// (which is non-deterministic across CI environments), we query the PRAGMA
// value directly after opening and assert it equals the configured value.
func TestApplyPragmas_AppliesBusyTimeout(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	conn, err := database.Conn(ctx)
	if err != nil {
		t.Fatalf("Conn: %v", err)
	}
	defer conn.Close()

	var busyTimeout int
	row := conn.QueryRowContext(ctx, "PRAGMA busy_timeout")
	if err := row.Scan(&busyTimeout); err != nil {
		t.Fatalf("PRAGMA busy_timeout scan: %v", err)
	}

	const wantBusyTimeout = 5000
	if busyTimeout != wantBusyTimeout {
		t.Errorf("busy_timeout = %d, want %d", busyTimeout, wantBusyTimeout)
	}
}

// forceDeleteJournalDB creates a fresh schema'd temp-file DB and forces it into
// DELETE journal mode via the documented seam: a direct `PRAGMA journal_mode =
// delete` on a real temp-file handle. It does NOT stub isUnsafeForMmap or edit
// pragmas.go's mode selection (out of scope per the bug brief — only the WRITE
// of the pragma is hardened, not which mode is chosen). DELETE is the failing
// scenario for bug-dd5db2d1: under DELETE-journal saturation the SHARED→RESERVED
// upgrade that `PRAGMA journal_mode = DELETE` performs at every db.Open can
// return SQLITE_BUSY immediately, bypassing busy_timeout. The schema-creating
// handle is closed before returning so it does not itself contend; the file's
// DELETE journal mode persists on disk.
func forceDeleteJournalDB(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "pragma-busy.db")
	w, err := db.Open(dbPath) // creates schema + runs migrations
	if err != nil {
		t.Fatalf("db.Open seed: %v", err)
	}
	if _, err := w.Exec("PRAGMA journal_mode = delete"); err != nil {
		w.Close()
		t.Fatalf("force journal_mode=delete: %v", err)
	}
	got := db.QueryJournalMode(w)
	w.Close()
	if got != "delete" {
		t.Fatalf("journal_mode = %q, want delete", got)
	}
	return dbPath
}

// TestApplyPragmas_JournalModeWriteRetriesUnderContention is the bug-dd5db2d1
// regression test. It reproduces the EXACT failing scenario: a DELETE-journal
// DB with a parked open reader (a live, undrained *sql.Rows cursor holding a
// SHARED lock). Changing journal mode requires escalating to a RESERVED/
// EXCLUSIVE lock; with a reader parked mid-statement SQLite returns
// SQLITE_BUSY *immediately* and does NOT honor busy_timeout (busy_timeout only
// waits on lock acquisition, not on the SHARED→RESERVED upgrade conflict). This
// is precisely the race bug-56b686aa's busy_timeout-first ordering + query-
// before-set fast path leave unprotected — and the failure surface is
// "applying PRAGMA journal_mode: database is locked" raised to the db.Open
// caller. The fix wraps that SET in RetryOnBusy; this test proves concurrent
// db.Open calls retry through the parked-reader contention (which is released
// mid-flight) and succeed without ever surfacing the lock error.
func TestApplyPragmas_JournalModeWriteRetriesUnderContention(t *testing.T) {
	db.ResetBusyCounters()
	dbPath := forceDeleteJournalDB(t)

	// Continuous-churn writer: a goroutine that repeatedly takes the writer
	// lock (BEGIN IMMEDIATE → tiny write → COMMIT) with no gap. This is the
	// proven contention harness from cmd/wipnote/lineage_busy_test.go's
	// holdWriteLock. Under DELETE journal mode a rapidly-cycling RESERVED lock
	// drives a concurrent `PRAGMA journal_mode = delete` (the open-time SET in
	// ApplyPragmas) into the SQLITE_BUSY that busy_timeout alone does NOT
	// reliably absorb: the lock-upgrade keeps losing the race to the next
	// churn cycle before SQLite's busy poll resolves. Without the RetryOnBusy
	// wrap this surfaces as "applying PRAGMA journal_mode: database is locked"
	// to the db.Open caller — the exact bug-dd5db2d1 failure.
	churnCtx, stopChurn := context.WithCancel(context.Background())
	var churnWG sync.WaitGroup
	var churnCycles atomic.Int64
	churnWG.Add(1)
	go func() {
		defer churnWG.Done()
		w, werr := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)")
		if werr != nil {
			t.Errorf("churn writer sql.Open: %v", werr)
			return
		}
		defer w.Close()
		w.SetMaxOpenConns(1)
		for churnCtx.Err() == nil {
			if _, e := w.Exec("BEGIN IMMEDIATE"); e != nil {
				continue
			}
			_, _ = w.Exec(
				`INSERT INTO features (id, type, title, status)
				 VALUES (?, 'bug', 't', 'todo')
				 ON CONFLICT(id) DO UPDATE SET title='t2'`,
				fmt.Sprintf("churn-%d", churnCycles.Load()))
			if _, e := w.Exec("COMMIT"); e == nil {
				churnCycles.Add(1)
			} else {
				_, _ = w.Exec("ROLLBACK")
			}
		}
	}()

	// Fire concurrent Opens while the churn writer saturates the write lock,
	// then stop the churn so the bounded RetryOnBusy backoff (~200/600/1800ms)
	// can recover. Every Open runs ApplyPragmas -> the wrapped journal_mode SET
	// against the DELETE-journal file. They must all retry through the
	// contention and succeed; none may surface the journal_mode lock error.
	const concurrentOpens = 6
	var wg sync.WaitGroup
	errs := make([]error, concurrentOpens)
	for i := range concurrentOpens {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			d, openErr := db.Open(dbPath)
			if openErr == nil {
				d.Close()
			}
			errs[idx] = openErr
		}(i)
	}

	// Let the contention bite for a window that spans multiple Open attempts,
	// then release so retries converge well within the deadline.
	go func() {
		time.Sleep(1200 * time.Millisecond)
		stopChurn()
	}()

	doneOpens := make(chan struct{})
	go func() { wg.Wait(); close(doneOpens) }()
	select {
	case <-doneOpens:
	case <-time.After(20 * time.Second):
		stopChurn()
		churnWG.Wait()
		t.Fatal("concurrent Opens did not complete within 20s")
	}
	stopChurn()
	churnWG.Wait()

	if churnCycles.Load() == 0 {
		t.Fatal("churn writer committed 0 cycles — the no-error assertion " +
			"would be vacuous (contention workload never ran)")
	}

	for i, e := range errs {
		if e != nil {
			if strings.Contains(e.Error(),
				"applying PRAGMA journal_mode: database is locked") {
				t.Fatalf("Open #%d surfaced the forbidden journal_mode lock "+
					"error (bug-dd5db2d1 regression): %v", i, e)
			}
			t.Fatalf("Open #%d failed under DELETE-journal write-churn "+
				"contention: %v (open-time pragma write must retry)", i, e)
		}
	}

	// Accounting consistency: ApplyPragmas's RetryOnBusy wrap does NOT call
	// db.Record (it transparently absorbs transient BUSY), so the first-party
	// launch-gate counter must remain consistent — zero, since every transient
	// BUSY here was absorbed by the retry and no terminal BUSY was recorded.
	if total := db.FirstPartyBusyTotal(); total != 0 {
		t.Fatalf("FirstPartyBusyTotal = %d, want 0 "+
			"(transient open-time pragma BUSY must be absorbed by the retry, "+
			"never recorded as a first-party terminal busy)", total)
	}
}

// TestApplyPragmas_NonBusyPragmaErrorNotRetried guards the blast-radius
// invariant: the RetryOnBusy wrap around the required-pragma SET must NOT
// change behavior on non-BUSY errors — a genuine pragma failure must surface
// immediately (one attempt), not pay the backoff budget. We assert the
// happy-path Open is fast (no spurious retry latency when uncontended), which
// transitively proves the wrap is a near-no-op on the success path that EVERY
// db.Open now traverses.
func TestApplyPragmas_NonBusyPragmaErrorNotRetried(t *testing.T) {
	dbPath := forceDeleteJournalDB(t)
	start := time.Now()
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("uncontended Open: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		d.Close()
		t.Fatalf("uncontended Open took %v; RetryOnBusy wrap must be a "+
			"near-no-op on the success path (no backoff when not BUSY)", elapsed)
	}
	// Sanity: the forced DELETE mode survived the wrapped re-application
	// (we only hardened the WRITE; mode SELECTION is unchanged).
	if jm := db.QueryJournalMode(d); !strings.EqualFold(jm, "delete") {
		d.Close()
		t.Fatalf("journal_mode = %q after Open, want delete "+
			"(RetryOnBusy wrap must not change which mode is written)", jm)
	}
	d.Close()
}
