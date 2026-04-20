package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/shakestzd/htmlgraph/internal/otel/materialize"
)

// otelRollupHandler returns the aggregated per-session OTel rollup.
// Reads otel_session_rollup (populated on SessionEnd) if present,
// otherwise computes the aggregate live from otel_signals. The live
// path lets the dashboard show partial stats for sessions that haven't
// reached SessionEnd yet.
//
// GET /api/otel/rollup?session_id=<id>
//   404 if no OTel signals exist for the session
//   200 JSON body shaped like the rollup struct with snake_case keys
func otelRollupHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			http.Error(w, "session_id required", http.StatusBadRequest)
			return
		}

		// Prefer the materialized row when it exists — it was written
		// inside a SessionEnd transaction so the caller gets a coherent
		// snapshot. Fall back to live aggregation for in-flight sessions.
		row, ok, err := readMaterializedRollup(database, sessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !ok {
			live, err := materialize.Session(database, sessionID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if live == nil {
				http.Error(w, "no OTel data for session", http.StatusNotFound)
				return
			}
			row = rollupJSON{
				SessionID:                live.SessionID,
				Harness:                  live.Harness,
				TotalCostUSD:             live.TotalCostUSD,
				TotalTokensIn:            live.TotalTokensIn,
				TotalTokensOut:           live.TotalTokensOut,
				TotalTokensCacheRead:     live.TotalTokensCacheRead,
				TotalTokensCacheCreation: live.TotalTokensCacheCreation,
				TotalTokensThought:       live.TotalTokensThought,
				TotalTokensTool:          live.TotalTokensTool,
				TotalTokensReasoning:     live.TotalTokensReasoning,
				TotalTurns:               live.TotalTurns,
				TotalToolCalls:           live.TotalToolCalls,
				TotalAPICalls:            live.TotalAPICalls,
				TotalAPIErrors:           live.TotalAPIErrors,
				MaxAttempt:               live.MaxAttempt,
				Live:                     true,
			}
		}
		respondJSON(w, row)
	}
}

// otelPromptsHandler returns per-prompt aggregates so the dashboard's
// event-tree can render cost/token badges per turn.
//
// GET /api/otel/prompts?session_id=<id>
//   200 JSON body: {"prompts": [{...}]}
func otelPromptsHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			http.Error(w, "session_id required", http.StatusBadRequest)
			return
		}
		ps, err := materialize.Prompts(database, sessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out := make([]promptJSON, 0, len(ps))
		for _, p := range ps {
			out = append(out, promptJSON{
				PromptID:            p.PromptID,
				FirstTsMicros:       p.FirstTs,
				DurationMs:          p.DurationMs,
				CostUSD:             p.CostUSD,
				TokensIn:            p.TokensIn,
				TokensOut:           p.TokensOut,
				TokensCacheRead:     p.TokensCacheRead,
				TokensCacheCreation: p.TokensCacheCreation,
				APICalls:            p.APICalls,
				ToolCalls:           p.ToolCalls,
				APIErrors:           p.APIErrors,
			})
		}
		respondJSON(w, map[string]any{"prompts": out})
	}
}

// otelCostHandler returns grouped cost aggregates. Supports three group
// dimensions matching common dashboard questions:
//
//   GET /api/otel/cost?group_by=model      — cost per model
//   GET /api/otel/cost?group_by=session    — cost per session
//   GET /api/otel/cost?group_by=day        — cost per calendar day (UTC)
//
// Omitting group_by defaults to "model". Invalid values return 400.
func otelCostHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		groupBy := r.URL.Query().Get("group_by")
		if groupBy == "" {
			groupBy = "model"
		}
		var groupCol, groupExpr string
		switch groupBy {
		case "model":
			groupCol = "model"
			groupExpr = "COALESCE(model, 'unknown')"
		case "session":
			groupCol = "session_id"
			groupExpr = "session_id"
		case "day":
			groupCol = "day"
			// ts_micros → UTC YYYY-MM-DD via SQLite's strftime.
			groupExpr = "strftime('%Y-%m-%d', ts_micros / 1000000, 'unixepoch')"
		default:
			http.Error(w, "group_by must be one of: model|session|day", http.StatusBadRequest)
			return
		}

		query := fmt.Sprintf(`
			SELECT %s AS k,
				COALESCE(SUM(cost_usd), 0) AS total_cost,
				COALESCE(SUM(tokens_in), 0) AS tokens_in,
				COALESCE(SUM(tokens_out), 0) AS tokens_out,
				COUNT(*) AS signal_count
			FROM otel_signals
			WHERE canonical = 'api_request' AND kind = 'log'
			GROUP BY k
			ORDER BY total_cost DESC
			LIMIT 200`, groupExpr)

		rows, err := database.Query(query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type bucket struct {
			Key         string  `json:"key"`
			TotalCost   float64 `json:"total_cost_usd"`
			TokensIn    int64   `json:"tokens_in"`
			TokensOut   int64   `json:"tokens_out"`
			SignalCount int64   `json:"signal_count"`
		}
		out := []bucket{}
		for rows.Next() {
			var b bucket
			if err := rows.Scan(&b.Key, &b.TotalCost, &b.TokensIn, &b.TokensOut, &b.SignalCount); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			out = append(out, b)
		}
		respondJSON(w, map[string]any{
			"group_by": groupCol,
			"buckets":  out,
		})
	}
}

// rollupJSON is the wire shape for /api/otel/rollup. Snake-case keys
// for JS callers that conventionally use them over Go's camelCase.
type rollupJSON struct {
	SessionID                string  `json:"session_id"`
	Harness                  string  `json:"harness"`
	TotalCostUSD             float64 `json:"total_cost_usd"`
	TotalTokensIn            int64   `json:"total_tokens_in"`
	TotalTokensOut           int64   `json:"total_tokens_out"`
	TotalTokensCacheRead     int64   `json:"total_tokens_cache_read"`
	TotalTokensCacheCreation int64   `json:"total_tokens_cache_creation"`
	TotalTokensThought       int64   `json:"total_tokens_thought"`
	TotalTokensTool          int64   `json:"total_tokens_tool"`
	TotalTokensReasoning     int64   `json:"total_tokens_reasoning"`
	TotalTurns               int64   `json:"total_turns"`
	TotalToolCalls           int64   `json:"total_tool_calls"`
	TotalAPICalls            int64   `json:"total_api_calls"`
	TotalAPIErrors           int64   `json:"total_api_errors"`
	MaxAttempt               int64   `json:"max_attempt"`
	// Live is true when the response was computed from otel_signals
	// rather than the materialized rollup. The dashboard can use this
	// to show a "session still active" indicator.
	Live bool `json:"live"`
}

type promptJSON struct {
	PromptID            string  `json:"prompt_id"`
	FirstTsMicros       int64   `json:"first_ts_micros"`
	DurationMs          int64   `json:"duration_ms"`
	CostUSD             float64 `json:"cost_usd"`
	TokensIn            int64   `json:"tokens_in"`
	TokensOut           int64   `json:"tokens_out"`
	TokensCacheRead     int64   `json:"tokens_cache_read"`
	TokensCacheCreation int64   `json:"tokens_cache_creation"`
	APICalls            int64   `json:"api_calls"`
	ToolCalls           int64   `json:"tool_calls"`
	APIErrors           int64   `json:"api_errors"`
}

// spanJSON is one row of the /api/otel/spans response. Shapes a
// single OTel span for the client-side tree builder: the client groups
// by trace_id and walks parent_span → span_id to reconstruct the tree.
//
// Details carries a whitelisted subset of the signal's attrs_json so
// the dashboard can render tool-specific content (bash command, file
// path, subagent type, etc.) without pulling the full payload — raw
// API bodies with OTEL_LOG_RAW_API_BODIES=1 can exceed 60 KB per signal.
type spanJSON struct {
	SignalID   string     `json:"signal_id"`
	TraceID    string     `json:"trace_id"`
	SpanID     string     `json:"span_id"`
	ParentSpan string     `json:"parent_span"`
	NativeName string     `json:"native_name"`
	Canonical  string     `json:"canonical"`
	ToolName   string     `json:"tool_name"`
	Model      string     `json:"model"`
	TsMicros   int64      `json:"ts_micros"`
	DurationMs int64      `json:"duration_ms"`
	TokensIn   int64      `json:"tokens_in"`
	TokensOut  int64      `json:"tokens_out"`
	CostUSD    float64    `json:"cost_usd"`
	Decision   string     `json:"decision"`
	Success    *bool      `json:"success,omitempty"`
	Details    spanDetail `json:"details"`
}

// spanDetail holds the whitelisted attributes extracted from attrs_json
// that the dashboard needs to render rich span rows. Anything not in
// this struct stays in the SQLite attrs_json column for drill-through
// via a future "span detail" view.
type spanDetail struct {
	FullCommand   string `json:"full_command,omitempty"`    // Bash: exact command executed
	BashCommand   string `json:"bash_command,omitempty"`    // Bash: un-shelled command
	Description   string `json:"description,omitempty"`     // Bash: human description
	FilePath      string `json:"file_path,omitempty"`       // Read/Edit/Write: target path
	Offset        int64  `json:"offset,omitempty"`          // Read: 1-based start line
	Limit         int64  `json:"limit,omitempty"`           // Read: line count
	URL           string `json:"url,omitempty"`             // WebFetch/WebSearch
	Pattern       string `json:"pattern,omitempty"`         // Grep/Glob
	SkillName     string `json:"skill_name,omitempty"`      // Skill tool
	SubagentType  string `json:"subagent_type,omitempty"`   // Agent/Task delegation target
	MCPServerName string `json:"mcp_server_name,omitempty"` // MCP tool
	MCPToolName   string `json:"mcp_tool_name,omitempty"`   // MCP tool
	DecisionSrc   string `json:"decision_source,omitempty"` // tool.blocked_on_user
	Speed         string `json:"speed,omitempty"`           // llm_request: fast|normal
	RequestID     string `json:"request_id,omitempty"`      // llm_request: Anthropic request ID
	Attempt       int64  `json:"attempt,omitempty"`         // llm_request: retry number
}

// otelSpansHandler returns every span persisted for the given session,
// ordered by timestamp. Clients build the tree by grouping on trace_id
// and linking parent_span → span_id. Typical payload is small (~100
// spans for a busy session); no pagination.
//
// GET /api/otel/spans?session_id=<id>
//   200 { "spans": [...] } — empty array if none exist
//   400 when session_id is missing
func otelSpansHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			http.Error(w, "session_id required", http.StatusBadRequest)
			return
		}
		rows, err := database.Query(`
			SELECT signal_id,
				COALESCE(trace_id, ''), COALESCE(span_id, ''), COALESCE(parent_span, ''),
				native, canonical,
				COALESCE(tool_name, ''), COALESCE(model, ''),
				ts_micros,
				COALESCE(duration_ms, 0),
				COALESCE(tokens_in, 0), COALESCE(tokens_out, 0),
				COALESCE(cost_usd, 0),
				COALESCE(decision, ''),
				success,
				COALESCE(attrs_json, '{}')
			FROM otel_signals
			WHERE session_id = ? AND kind = 'span'
			ORDER BY ts_micros ASC`, sessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		out := []spanJSON{}
		for rows.Next() {
			var s spanJSON
			var successVal sql.NullInt64
			var attrsRaw string
			if err := rows.Scan(
				&s.SignalID, &s.TraceID, &s.SpanID, &s.ParentSpan,
				&s.NativeName, &s.Canonical, &s.ToolName, &s.Model,
				&s.TsMicros, &s.DurationMs,
				&s.TokensIn, &s.TokensOut, &s.CostUSD,
				&s.Decision, &successVal, &attrsRaw,
			); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if successVal.Valid {
				b := successVal.Int64 == 1
				s.Success = &b
			}
			s.Details = extractSpanDetails(attrsRaw)
			out = append(out, s)
		}

		// Second pass: enrich tool spans with context from their matching
		// tool_result log. Tool input details (Read offset/limit, Agent
		// subagent_type, Edit old_string length, etc.) live on the log
		// side only — the span carries a thinner attr set. Matching is
		// by (tool_name, ordinality) within the session; tool spans and
		// tool_result logs are emitted 1:1 per tool call, so this gives
		// deterministic pairing without fuzzy timestamp matching.
		enrichToolSpansFromLogs(database, sessionID, out)

		respondJSON(w, map[string]any{"spans": out})
	}
}

// enrichToolSpansFromLogs fetches tool_result logs for the session and
// merges the nested tool_input attrs into the matching tool span's
// Details. Mutates the out slice in place. Failures (DB error, bad JSON)
// are logged at debug-level elsewhere; one missing enrichment shouldn't
// poison the whole endpoint.
func enrichToolSpansFromLogs(database *sql.DB, sessionID string, out []spanJSON) {
	rows, err := database.Query(`
		SELECT COALESCE(tool_name, ''), COALESCE(attrs_json, '{}')
		FROM otel_signals
		WHERE session_id = ? AND kind = 'log' AND canonical = 'tool_result'
		ORDER BY ts_micros ASC`, sessionID)
	if err != nil {
		return
	}
	defer rows.Close()

	// Group log attrs by tool_name, in emit order.
	logsByTool := map[string][]string{}
	for rows.Next() {
		var tool, attrs string
		if err := rows.Scan(&tool, &attrs); err != nil {
			continue
		}
		if tool == "" {
			continue
		}
		logsByTool[tool] = append(logsByTool[tool], attrs)
	}

	// Per tool, walk the spans in order and pair with logs in order.
	// Eligible spans: any span that carries a tool_name and is a logical
	// tool invocation. Claude's adapter canonicalizes ordinary tool spans
	// as "tool_result" and Agent/Task subagent tool spans as
	// "subagent_invocation" — both need enrichment from their matching
	// tool_result log. Infrastructure spans (interaction, llm_request,
	// tool.execution, tool.blocked_on_user) have no corresponding log.
	spanIdxByTool := map[string]int{}
	for i := range out {
		s := &out[i]
		if s.ToolName == "" {
			continue
		}
		if s.Canonical != "tool_result" && s.Canonical != "subagent_invocation" {
			continue
		}
		logs := logsByTool[s.ToolName]
		idx := spanIdxByTool[s.ToolName]
		spanIdxByTool[s.ToolName] = idx + 1
		if idx >= len(logs) {
			continue
		}
		mergeLogIntoSpanDetails(&s.Details, logs[idx])
	}
}

// mergeLogIntoSpanDetails parses a tool_result log's attrs_json and
// merges fields the span doesn't already have. Specifically pulls the
// nested tool_input string and extracts keys like offset, limit,
// subagent_type, description.
func mergeLogIntoSpanDetails(d *spanDetail, logAttrsRaw string) {
	var logAttrs map[string]any
	if err := json.Unmarshal([]byte(logAttrsRaw), &logAttrs); err != nil {
		return
	}
	// tool_input is a JSON-encoded string inside attrs; parse it.
	toolInputStr, ok := logAttrs["tool_input"].(string)
	if !ok || toolInputStr == "" {
		return
	}
	var ti map[string]any
	if err := json.Unmarshal([]byte(toolInputStr), &ti); err != nil {
		return
	}
	// Fill in only fields the span didn't already populate. Later adapter
	// versions may surface these on the span directly — prefer span-side.
	if d.SubagentType == "" {
		if s, _ := ti["subagent_type"].(string); s != "" {
			d.SubagentType = s
		}
	}
	if d.Description == "" {
		if s, _ := ti["description"].(string); s != "" {
			d.Description = s
		}
	}
	if d.FilePath == "" {
		if s, _ := ti["file_path"].(string); s != "" {
			d.FilePath = s
		}
	}
	if d.Offset == 0 {
		d.Offset = pullInt(ti, "offset")
	}
	if d.Limit == 0 {
		d.Limit = pullInt(ti, "limit")
	}
	if d.Pattern == "" {
		if s, _ := ti["pattern"].(string); s != "" {
			d.Pattern = s
		}
	}
	if d.URL == "" {
		if s, _ := ti["url"].(string); s != "" {
			d.URL = s
		}
	}
}

// extractSpanDetails pulls the whitelisted attributes out of attrs_json.
// Unrecognized keys are ignored so the payload stays small; callers can
// drill into the raw JSON later if we add a detail-view endpoint.
//
// Returns a zero-value spanDetail on any JSON error — one bad row should
// not poison the whole endpoint.
func extractSpanDetails(attrsRaw string) spanDetail {
	var d spanDetail
	if attrsRaw == "" || attrsRaw == "{}" {
		return d
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(attrsRaw), &raw); err != nil {
		return d
	}
	pull := func(k string) string {
		if v, ok := raw[k]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	d.FullCommand = pull("full_command")
	d.BashCommand = pull("bash_command")
	d.Description = pull("description")
	d.FilePath = pull("file_path")
	d.URL = pull("url")
	d.Pattern = pull("pattern")
	d.SkillName = pull("skill_name")
	d.SubagentType = pull("subagent_type")
	d.MCPServerName = pull("mcp_server_name")
	d.MCPToolName = pull("mcp_tool_name")
	d.DecisionSrc = pull("source")
	d.Speed = pull("speed")
	d.RequestID = pull("request_id")
	// Numeric fields that may arrive as int (OTLP/gRPC binary) or as
	// string (OTLP/HTTP JSON). Best-effort parse in both cases.
	d.Attempt = pullInt(raw, "attempt")
	d.Offset = pullInt(raw, "offset")
	d.Limit = pullInt(raw, "limit")
	return d
}

// pullInt extracts a numeric attr, accepting int / float / digit-string.
// Returns 0 when missing or unparseable — consistent with other
// "not reported" conventions in this file.
func pullInt(raw map[string]any, key string) int64 {
	v, ok := raw[key]
	if !ok {
		return 0
	}
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case int:
		return int64(x)
	case string:
		var n int64
		for i := 0; i < len(x); i++ {
			if x[i] < '0' || x[i] > '9' {
				return 0
			}
			n = n*10 + int64(x[i]-'0')
		}
		return n
	}
	return 0
}


// readMaterializedRollup fetches the row from otel_session_rollup.
// Returns (zero, false, nil) when no row exists, so the caller can
// fall back to a live aggregation.
func readMaterializedRollup(database *sql.DB, sessionID string) (rollupJSON, bool, error) {
	var r rollupJSON
	err := database.QueryRow(`
		SELECT
			session_id, harness, total_cost_usd,
			COALESCE(total_tokens_in, 0), COALESCE(total_tokens_out, 0),
			COALESCE(total_tokens_cache_read, 0), COALESCE(total_tokens_cache_creation, 0),
			COALESCE(total_tokens_thought, 0), COALESCE(total_tokens_tool, 0),
			COALESCE(total_tokens_reasoning, 0),
			COALESCE(total_turns, 0), COALESCE(total_tool_calls, 0),
			COALESCE(total_api_calls, 0), COALESCE(total_api_errors, 0),
			COALESCE(max_attempt, 0)
		FROM otel_session_rollup
		WHERE session_id = ?`, sessionID,
	).Scan(
		&r.SessionID, &r.Harness, &r.TotalCostUSD,
		&r.TotalTokensIn, &r.TotalTokensOut,
		&r.TotalTokensCacheRead, &r.TotalTokensCacheCreation,
		&r.TotalTokensThought, &r.TotalTokensTool,
		&r.TotalTokensReasoning,
		&r.TotalTurns, &r.TotalToolCalls,
		&r.TotalAPICalls, &r.TotalAPIErrors,
		&r.MaxAttempt,
	)
	if err == sql.ErrNoRows {
		return rollupJSON{}, false, nil
	}
	if err != nil {
		return rollupJSON{}, false, err
	}
	return r, true, nil
}
