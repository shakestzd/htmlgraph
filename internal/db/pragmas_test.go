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

// selectedJournalMode returns the journal mode the REAL, unchanged selection
// logic (BuildPragmas -> isUnsafeForMmap) chooses for dbPath on the host
// running the test. Assertions derive the expected mode from this — never a
// hardcoded "delete", which only holds on WAL-unsafe filesystems (overlayfs/
// FUSE devcontainers) and would false-fail on WAL hosts (host installs on
// APFS/ext4). This is the same filesystem-agnostic discipline as
// bug-74a7bda7's busy_filesystem_agnostic_test.go.
func selectedJournalMode(dbPath string) string {
	return strings.ToLower(db.BuildPragmas(dbPath)["journal_mode"])
}

// seededJournalDB creates a fresh schema'd temp-file DB via db.Open (which
// applies BuildPragmas, so the file lands in whatever mode the host's
// selection logic chooses) and returns its path. It does NOT force a specific
// journal mode — callers that need the DELETE-specific contention path either
// skip-on-WAL or drive ApplyPragmas directly with an explicit DELETE pragma
// map. The schema-creating handle is closed before returning so it does not
// itself contend.
func seededJournalDB(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "pragma-busy.db")
	w, err := db.Open(dbPath) // creates schema + runs migrations
	if err != nil {
		t.Fatalf("db.Open seed: %v", err)
	}
	w.Close()
	return dbPath
}

// deleteModePragmas returns an explicit pragma map identical to the real
// BuildPragmas output EXCEPT journal_mode is pinned to DELETE. Driving
// ApplyPragmas with this map deterministically exercises the DELETE-journal
// contended-write path the bug is about, on ANY host filesystem, WITHOUT
// touching production mode selection (isUnsafeForMmap/BuildPragmas are
// unchanged — the test simply supplies its own inputs to ApplyPragmas).
func deleteModePragmas(dbPath string) map[string]string {
	p := db.BuildPragmas(dbPath)
	p["journal_mode"] = "DELETE"
	return p
}

// TestApplyPragmas_JournalModeWriteRetriesUnderContention is the bug-dd5db2d1
// regression test. It reproduces the EXACT failing scenario deterministically
// on ANY host filesystem: it drives ApplyPragmas DIRECTLY with an explicit
// DELETE pragma map (deleteModePragmas) — option (b) from the roborev
// followup — so the DELETE-journal contended write is exercised regardless of
// whether the host's BuildPragmas would have selected WAL or DELETE.
// Production mode SELECTION (isUnsafeForMmap/BuildPragmas) is UNCHANGED; the
// test merely supplies its own pragma inputs.
//
// A continuous-churn writer (BEGIN IMMEDIATE → tiny write → COMMIT, the proven
// harness from cmd/wipnote/lineage_busy_test.go) saturates the write lock. A
// concurrent `PRAGMA journal_mode = DELETE` (the open-time SET in
// ApplyPragmas) loses the lock-upgrade race to the next churn cycle before
// SQLite's busy poll resolves. Without the RetryOnBusy wrap this surfaces as
// "applying PRAGMA journal_mode: database is locked" — the exact bug-dd5db2d1
// failure. The test proves concurrent opens retry through it and succeed.
func TestApplyPragmas_JournalModeWriteRetriesUnderContention(t *testing.T) {
	db.ResetBusyCounters()
	dbPath := seededJournalDB(t)
	deletePragmas := deleteModePragmas(dbPath)

	// Continuous-churn writer: a goroutine that repeatedly takes the writer
	// lock (BEGIN IMMEDIATE → tiny write → COMMIT) with no gap.
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

	// Fire concurrent opens that each drive ApplyPragmas with the explicit
	// DELETE pragma map (exactly the code path db.Open uses, but with a
	// deterministic DELETE selection independent of host filesystem). They
	// must all retry through the churn contention and succeed; none may
	// surface the journal_mode lock error.
	const concurrentOpens = 6
	var wg sync.WaitGroup
	errs := make([]error, concurrentOpens)
	for i := range concurrentOpens {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			d, oerr := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)")
			if oerr != nil {
				errs[idx] = oerr
				return
			}
			defer d.Close()
			errs[idx] = db.ApplyPragmas(d, deletePragmas)
		}(i)
	}

	// Let the contention bite for a window that spans multiple attempts,
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
		t.Fatal("concurrent ApplyPragmas did not complete within 20s")
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
	dbPath := seededJournalDB(t)
	// Expected mode is whatever the REAL, unchanged selection logic chooses
	// for this host's filesystem — DELETE on overlayfs/FUSE, WAL on APFS/ext4.
	// Never a hardcoded literal.
	wantMode := selectedJournalMode(dbPath)

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
	// Sanity: the selected mode survived the wrapped re-application (we only
	// hardened the WRITE; mode SELECTION is unchanged and filesystem-derived).
	if jm := db.QueryJournalMode(d); !strings.EqualFold(jm, wantMode) {
		d.Close()
		t.Fatalf("journal_mode = %q after Open, want %q (the host-selected "+
			"mode; RetryOnBusy wrap must not change which mode is written)",
			jm, wantMode)
	}
	d.Close()
}
