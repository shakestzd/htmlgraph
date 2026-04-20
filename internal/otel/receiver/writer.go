package receiver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shakestzd/htmlgraph/internal/otel"
)

// Writer persists UnifiedSignals into the otel_signals table. It owns
// its own *sql.DB with MaxOpenConns=1 so every write serializes through
// one connection — this eliminates SQLITE_BUSY errors under concurrent
// load from the OTLP receiver and hook binaries that share the DB file.
//
// All inserts go through BEGIN IMMEDIATE transactions (one per batch);
// IMMEDIATE acquires the writer lock up front so we don't burn retry
// budget on deferred upgrades. Prepared statements are held for the
// Writer's lifetime.
type Writer struct {
	db         *sql.DB
	insertStmt *sql.Stmt
	sessStmt   *sql.Stmt
	resStmt    *sql.Stmt
}

// NewWriter opens a writer-mode DB handle on dbPath. The handle is
// separate from whatever read pool the caller may already have open:
//
//	readers := db.Open(path)             // existing read pool
//	writer  := receiver.NewWriter(path)  // dedicated single-conn writer
//
// Both are fine because SQLite WAL mode allows concurrent readers with
// a single writer. The caller must Close the writer on shutdown so the
// prepared statements release.
func NewWriter(dbPath string) (*Writer, error) {
	// file: URL parameters configure WAL + busy timeout at open time.
	// These are idempotent with schema.go's ApplyPragmas, which ran
	// already via the read-pool Open.
	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open writer: %w", err)
	}
	// The single-writer constraint is the core of the concurrency
	// design. Do not raise this number without reworking the batching
	// strategy.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxIdleTime(0)

	w := &Writer{db: db}
	if err := w.prepare(); err != nil {
		db.Close()
		return nil, err
	}
	return w, nil
}

func (w *Writer) prepare() error {
	var err error
	w.insertStmt, err = w.db.Prepare(`
		INSERT OR IGNORE INTO otel_signals (
			signal_id, harness, session_id, prompt_id,
			trace_id, span_id, parent_span,
			kind, canonical, native, ts_micros,
			tool_name, tool_use_id, model, decision, decision_source,
			tokens_in, tokens_out, tokens_cache_read, tokens_cache_creation,
			tokens_thought, tokens_tool, tokens_reasoning,
			cost_usd, cost_source,
			duration_ms, success, error_msg, attempt, status_code,
			attrs_json, feature_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	// Session placeholder upsert: if the OTLP receiver sees a session_id
	// we haven't created via the hooks path, we create a minimal row so
	// the FK resolves. If SessionStart later fires for the same id, it
	// upgrades agent_assigned from the placeholder. Status stays 'active'.
	w.sessStmt, err = w.db.Prepare(`
		INSERT OR IGNORE INTO sessions (session_id, agent_assigned, status)
		VALUES (?, ?, 'active')`)
	if err != nil {
		return fmt.Errorf("prepare session upsert: %w", err)
	}
	// Resource attribute upsert: per (session_id, key), replace on conflict.
	// OTel resource attrs repeat on every batch; we want the latest value.
	w.resStmt, err = w.db.Prepare(`
		INSERT INTO otel_resource_attrs (session_id, harness, key, value, observed_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(session_id, key) DO UPDATE SET
			value = excluded.value,
			observed_at = excluded.observed_at`)
	if err != nil {
		return fmt.Errorf("prepare resource upsert: %w", err)
	}
	return nil
}

// Close releases prepared statements and the underlying connection.
func (w *Writer) Close() error {
	if w.insertStmt != nil {
		w.insertStmt.Close()
	}
	if w.sessStmt != nil {
		w.sessStmt.Close()
	}
	if w.resStmt != nil {
		w.resStmt.Close()
	}
	return w.db.Close()
}

// WriteBatch persists one OTLP request's worth of signals plus the
// resource attributes that produced them. The whole batch runs in one
// BEGIN IMMEDIATE transaction — either every signal lands or none do.
//
// session_ids are deduplicated inside the transaction so we only issue
// one sessions placeholder upsert per distinct session in the batch.
//
// Returns the number of rows actually inserted (excludes idempotent
// rejections on duplicate signal_id). Callers log the rejection count
// separately for observability.
func (w *Writer) WriteBatch(
	ctx context.Context,
	harness otel.Harness,
	resourceAttrs map[string]any,
	signals []otel.UnifiedSignal,
) (inserted int, err error) {
	if len(signals) == 0 {
		return 0, nil
	}

	tx, err := w.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Upgrade the tx to an IMMEDIATE writer lock so we don't race
	// another connection. modernc.org/sqlite supports this via a
	// second Exec before the real work starts.
	if _, err = tx.ExecContext(ctx, "SELECT 1"); err != nil {
		return 0, fmt.Errorf("warm tx: %w", err)
	}

	// Track sessions we've already upserted this batch so we don't
	// fire a redundant INSERT per signal.
	seen := map[string]bool{}
	// Per-session cache of active work item (feature/bug/spike claimed
	// by the session's root agent). Populated lazily on first signal
	// for each session so we issue at most one SELECT per distinct
	// session per batch, regardless of signal count.
	featureByID := map[string]string{}
	resObservedAt := time.Now().UnixMicro()

	insertStmt := tx.Stmt(w.insertStmt)
	sessStmt := tx.Stmt(w.sessStmt)
	resStmt := tx.Stmt(w.resStmt)

	for i := range signals {
		s := &signals[i]
		if s.SessionID == "" {
			// Drop signals without a session. OTel emissions always
			// carry session.id either on the resource or the signal;
			// a missing one means the adapter couldn't normalize.
			continue
		}
		if !seen[s.SessionID] {
			agent := string(harness)
			if _, err = sessStmt.ExecContext(ctx, s.SessionID, agent); err != nil {
				return inserted, fmt.Errorf("sessions upsert: %w", err)
			}
			// Persist the resource attributes snapshot for this session.
			for k, v := range resourceAttrs {
				if sv, ok := valueString(v); ok {
					if _, err = resStmt.ExecContext(ctx, s.SessionID, string(harness), k, sv, resObservedAt); err != nil {
						return inserted, fmt.Errorf("resource attr upsert: %w", err)
					}
				}
			}
			seen[s.SessionID] = true
		}

		attrsJSON, jerr := json.Marshal(s.RawAttrs)
		if jerr != nil {
			attrsJSON = []byte(`{}`)
		}

		var successVal sql.NullInt64
		if s.Success != nil {
			successVal.Valid = true
			if *s.Success {
				successVal.Int64 = 1
			}
		}

		// Look up the session's active work item on first encounter,
		// then reuse the cached value. Uses the __root__ sentinel since
		// OTel signals don't carry an agent_id — subagent-level
		// attribution is the planned follow-up (feat-82e11bbb).
		featureID, cached := featureByID[s.SessionID]
		if !cached {
			var fid sql.NullString
			_ = tx.QueryRowContext(ctx,
				`SELECT work_item_id FROM active_work_items WHERE session_id = ? AND agent_id = ?`,
				s.SessionID, "__root__",
			).Scan(&fid)
			featureID = fid.String
			featureByID[s.SessionID] = featureID
		}

		res, execErr := insertStmt.ExecContext(ctx,
			s.SignalID, string(s.Harness), s.SessionID, nullStr(s.PromptID),
			nullStr(s.TraceID), nullStr(s.SpanID), nullStr(s.ParentSpan),
			string(s.Kind), s.CanonicalName, s.NativeName, s.Timestamp.UnixMicro(),
			nullStr(s.ToolName), nullStr(s.ToolUseID), nullStr(s.Model),
			nullStr(s.Decision), nullStr(s.DecisionSource),
			nullInt64(s.Tokens.Input), nullInt64(s.Tokens.Output),
			nullInt64(s.Tokens.CacheRead), nullInt64(s.Tokens.CacheCreation),
			nullInt64(s.Tokens.Thought), nullInt64(s.Tokens.Tool), nullInt64(s.Tokens.Reasoning),
			nullFloat(s.CostUSD), nullStr(string(s.CostSource)),
			nullInt64(s.DurationMs), successVal, nullStr(s.ErrorMsg),
			nullInt(s.Attempt), nullInt(s.StatusCode),
			string(attrsJSON), nullStr(featureID),
		)
		if execErr != nil {
			return inserted, fmt.Errorf("insert signal %s: %w", s.SignalID, execErr)
		}
		if n, err := res.RowsAffected(); err == nil {
			inserted += int(n)
		}
	}

	if err = tx.Commit(); err != nil {
		return inserted, fmt.Errorf("commit: %w", err)
	}
	return inserted, nil
}

// DB returns the underlying handle. Tests use this to assert row counts
// without opening a second connection (which would contend for the
// MaxOpenConns=1 writer lock).
func (w *Writer) DB() *sql.DB { return w.db }

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
func nullInt64(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}
func nullInt(v int) any {
	if v == 0 {
		return nil
	}
	return v
}
func nullFloat(v float64) any {
	if v == 0 {
		return nil
	}
	return v
}

// valueString converts a resource-attribute AnyValue (already flattened
// to map[string]any by the decoder) into a string suitable for the
// otel_resource_attrs.value column. Non-scalar values are JSON-encoded.
func valueString(v any) (string, bool) {
	if v == nil {
		return "", false
	}
	switch x := v.(type) {
	case string:
		return x, true
	case bool:
		if x {
			return "true", true
		}
		return "false", true
	case int64:
		return fmt.Sprintf("%d", x), true
	case int:
		return fmt.Sprintf("%d", x), true
	case float64:
		return fmt.Sprintf("%g", x), true
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", false
		}
		return string(b), true
	}
}
