package db

import (
	"database/sql"
	"fmt"
)

// CreateOtelTables creates the OpenTelemetry ingestion tables. It is
// called from Open after CreateAllTables so the otel_signals foreign
// key to sessions(session_id) resolves. All statements are idempotent.
//
// Schema overview:
//
//   otel_signals         — one row per OTLP metric point, log record, or span
//   otel_resource_attrs  — per-session resource attribute snapshot (service.version, terminal.type, ...)
//   otel_session_rollup  — materialized totals written on SessionEnd
//
// Design notes:
//   - signal_id is the idempotency key. Receivers compute it as a hash
//     of (resource, scope, name, timestamp, sorted attributes) so OTLP
//     retries don't double-count. INSERT OR IGNORE on conflict.
//   - session_id is normalized across harnesses (Claude session.id, Codex
//     conversation_id, Gemini session.id).
//   - prompt_id is Claude's native prompt.id, or a synthesized ID for
//     Codex (codex:{conversation_id}:{turn_counter}) and any future
//     harness without a native per-turn correlator.
//   - tokens_* columns cover every dimension any harness emits; unused
//     dimensions are NULL, not 0, so aggregate queries can distinguish
//     "zero reported" from "not applicable".
//   - cost_source records how cost_usd was derived: "vendor" when the
//     harness reported it natively (Claude), "derived" when we computed
//     it from tokens × pricing (Codex, Gemini), or "unknown" when we
//     lacked pricing data for the model.
func CreateOtelTables(db *sql.DB) error {
	stmts := []string{
		// otel_signals: one row per OTLP metric/log/span signal.
		`CREATE TABLE IF NOT EXISTS otel_signals (
			signal_id             TEXT PRIMARY KEY,
			harness               TEXT NOT NULL,
			session_id            TEXT NOT NULL,
			prompt_id             TEXT,
			trace_id              TEXT,
			span_id               TEXT,
			parent_span           TEXT,
			kind                  TEXT NOT NULL CHECK(kind IN ('metric','log','span')),
			canonical             TEXT NOT NULL,
			native                TEXT NOT NULL,
			ts_micros             INTEGER NOT NULL,
			tool_name             TEXT,
			tool_use_id           TEXT,
			model                 TEXT,
			decision              TEXT,
			decision_source       TEXT,
			tokens_in             INTEGER,
			tokens_out            INTEGER,
			tokens_cache_read     INTEGER,
			tokens_cache_creation INTEGER,
			tokens_thought        INTEGER,
			tokens_tool           INTEGER,
			tokens_reasoning      INTEGER,
			cost_usd              REAL,
			cost_source           TEXT CHECK(cost_source IS NULL OR cost_source IN ('vendor','derived','unknown')),
			duration_ms           INTEGER,
			success               INTEGER,
			error_msg             TEXT,
			attempt               INTEGER,
			status_code           INTEGER,
			attrs_json            TEXT NOT NULL,
			created_at            INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000000),
			FOREIGN KEY (session_id) REFERENCES sessions(session_id) ON DELETE CASCADE ON UPDATE CASCADE
		)`,

		// otel_resource_attrs: one row per (session_id, key).
		// Resource attributes repeat on every OTLP batch; we snapshot them
		// once per session so queries can filter by terminal.type, host.arch,
		// service.version, etc. without scanning otel_signals.
		`CREATE TABLE IF NOT EXISTS otel_resource_attrs (
			session_id TEXT NOT NULL,
			harness    TEXT NOT NULL,
			key        TEXT NOT NULL,
			value      TEXT NOT NULL,
			observed_at INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000000),
			PRIMARY KEY (session_id, key),
			FOREIGN KEY (session_id) REFERENCES sessions(session_id) ON DELETE CASCADE ON UPDATE CASCADE
		)`,

		// otel_session_rollup: aggregated totals, materialized on SessionEnd.
		// The dashboard reads this table for cheap per-session cost/token
		// summaries instead of scanning otel_signals. Rebuilt idempotently
		// from otel_signals, so destroying and recomputing is always safe.
		`CREATE TABLE IF NOT EXISTS otel_session_rollup (
			session_id                   TEXT PRIMARY KEY,
			harness                      TEXT NOT NULL,
			total_cost_usd               REAL,
			total_tokens_in              INTEGER,
			total_tokens_out             INTEGER,
			total_tokens_cache_read      INTEGER,
			total_tokens_cache_creation  INTEGER,
			total_tokens_thought         INTEGER,
			total_tokens_tool            INTEGER,
			total_tokens_reasoning       INTEGER,
			total_turns                  INTEGER,
			total_tool_calls             INTEGER,
			total_api_calls              INTEGER,
			total_api_errors             INTEGER,
			max_attempt                  INTEGER,
			materialized_at              INTEGER NOT NULL,
			FOREIGN KEY (session_id) REFERENCES sessions(session_id) ON DELETE CASCADE ON UPDATE CASCADE
		)`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("exec OTel DDL: %w\nSQL: %.160s", err, stmt)
		}
	}

	// Idempotent migration: feature_id column added after initial schema
	// so existing DBs pick it up on the next `htmlgraph serve`. Duplicate
	// column errors are expected on re-runs and are silently swallowed,
	// matching the convention used elsewhere in internal/db/schema.go.
	if _, err := db.Exec(`ALTER TABLE otel_signals ADD COLUMN feature_id TEXT`); err != nil {
		// Ignore "duplicate column" errors — the column is already there.
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_otel_feature_ts ON otel_signals(feature_id, ts_micros) WHERE feature_id IS NOT NULL`); err != nil {
		// Index creation is non-critical; continue.
	}
	return nil
}

// CreateOtelIndexes creates performance indexes for the OTel tables.
// Mirrors the CreateAllIndexes pattern — non-fatal on individual failures
// so a partially-migrated DB can still serve traffic.
func CreateOtelIndexes(db *sql.DB) error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_otel_session_ts   ON otel_signals(session_id, ts_micros)",
		"CREATE INDEX IF NOT EXISTS idx_otel_prompt       ON otel_signals(prompt_id)",
		"CREATE INDEX IF NOT EXISTS idx_otel_canonical_ts ON otel_signals(canonical, ts_micros DESC)",
		"CREATE INDEX IF NOT EXISTS idx_otel_trace        ON otel_signals(trace_id)",
		"CREATE INDEX IF NOT EXISTS idx_otel_parent_span  ON otel_signals(parent_span)",
		"CREATE INDEX IF NOT EXISTS idx_otel_tool         ON otel_signals(session_id, tool_name, ts_micros)",
		"CREATE INDEX IF NOT EXISTS idx_otel_harness      ON otel_signals(harness, ts_micros DESC)",
		"CREATE INDEX IF NOT EXISTS idx_otel_model_ts     ON otel_signals(model, ts_micros) WHERE model IS NOT NULL",
	}
	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			_ = err // non-fatal, matches CreateAllIndexes convention
		}
	}
	return nil
}
