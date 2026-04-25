// Package ndjson provides a SignalSink that appends unified OTel signals as
// newline-delimited JSON (one line per signal) to a per-session events.ndjson
// file. No DB connection is opened. Placeholder/upgrade logic is intentionally
// absent — the NDJSON→SQLite indexer (slice 5) handles that on replay.
//
// File layout: .htmlgraph/sessions/<session_id>/events.ndjson
//
// Each line is a JSON object with all UnifiedSignal fields plus:
//   - "kind"    — signal kind ("span", "metric", "log")
//   - "ts"      — timestamp in RFC3339Nano
//   - "harness" — harness name
//
// Every write acquires syscall.Flock(LOCK_EX) before appending and releases
// it afterward, matching the pattern in session_html.go:147 and materialize.go:241.
package ndjson

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/shakestzd/htmlgraph/internal/otel"
	"github.com/shakestzd/htmlgraph/internal/otel/sink"
)

// Sink appends signals to a per-session NDJSON file.
type Sink struct {
	path string
	mu   sync.Mutex // guards concurrent WriteBatch calls within one process
}

// New constructs a Sink for the given project directory and session ID.
// The events.ndjson file is created lazily on first write; the session
// directory must already exist.
func New(projectDir, sessionID string) (*Sink, error) {
	path := filepath.Join(projectDir, ".htmlgraph", "sessions", sessionID, "events.ndjson")
	return &Sink{path: path}, nil
}

// WriteBatch appends one JSON line per signal to events.ndjson.
// An exclusive flock is held for the duration of the write so concurrent
// processes (e.g. a collector child and the indexer) don't interleave lines.
// Empty batches are a no-op.
func (s *Sink) WriteBatch(_ context.Context, harness otel.Harness, resourceAttrs map[string]any, signals []otel.UnifiedSignal) error {
	if len(signals) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("ndjson open %s: %w", s.path, err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("ndjson flock %s: %w", s.path, err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN) //nolint:errcheck

	for i := range signals {
		line, err := marshalLine(harness, resourceAttrs, &signals[i])
		if err != nil {
			return fmt.Errorf("ndjson marshal signal %s: %w", signals[i].SignalID, err)
		}
		line = append(line, '\n')
		if _, err := f.Write(line); err != nil {
			return fmt.Errorf("ndjson write signal %s: %w", signals[i].SignalID, err)
		}
	}
	return nil
}

// Close is a no-op for NDJSON — the file is opened and closed per write.
// Satisfies the sink.SignalSink interface.
func (s *Sink) Close() error { return nil }

// Ensure Sink implements SignalSink at compile time.
var _ sink.SignalSink = (*Sink)(nil)

// signalLine is the on-disk JSON representation of a single signal.
// Top-level fields carry the most-queried attributes; RawAttrs holds everything else.
type signalLine struct {
	Kind      string         `json:"kind"`
	Harness   string         `json:"harness"`
	TS        string         `json:"ts"`
	SignalID  string         `json:"signal_id"`
	SessionID string         `json:"session_id"`
	PromptID  string         `json:"prompt_id,omitempty"`

	CanonicalName string `json:"canonical,omitempty"`
	NativeName    string `json:"native,omitempty"`

	TraceID    string `json:"trace_id,omitempty"`
	SpanID     string `json:"span_id,omitempty"`
	ParentSpan string `json:"parent_span,omitempty"`

	ToolName       string `json:"tool_name,omitempty"`
	ToolUseID      string `json:"tool_use_id,omitempty"`
	Model          string `json:"model,omitempty"`
	Decision       string `json:"decision,omitempty"`
	DecisionSource string `json:"decision_source,omitempty"`

	TokensInput         int64 `json:"tokens_input,omitempty"`
	TokensOutput        int64 `json:"tokens_output,omitempty"`
	TokensCacheRead     int64 `json:"tokens_cache_read,omitempty"`
	TokensCacheCreation int64 `json:"tokens_cache_creation,omitempty"`
	TokensThought       int64 `json:"tokens_thought,omitempty"`
	TokensTool          int64 `json:"tokens_tool,omitempty"`
	TokensReasoning     int64 `json:"tokens_reasoning,omitempty"`

	CostUSD    float64 `json:"cost_usd,omitempty"`
	CostSource string  `json:"cost_source,omitempty"`

	DurationMs int64   `json:"duration_ms,omitempty"`
	Success    *bool   `json:"success,omitempty"`
	ErrorMsg   string  `json:"error_msg,omitempty"`
	Attempt    int     `json:"attempt,omitempty"`
	StatusCode int     `json:"status_code,omitempty"`

	ResourceAttrs map[string]any `json:"resource_attrs,omitempty"`
	Attrs         map[string]any `json:"attrs,omitempty"`
}

// marshalLine converts a UnifiedSignal into a JSON byte slice for NDJSON output.
func marshalLine(harness otel.Harness, resourceAttrs map[string]any, s *otel.UnifiedSignal) ([]byte, error) {
	line := signalLine{
		Kind:                string(s.Kind),
		Harness:             string(harness),
		TS:                  s.Timestamp.UTC().Format(time.RFC3339Nano),
		SignalID:            s.SignalID,
		SessionID:           s.SessionID,
		PromptID:            s.PromptID,
		CanonicalName:       s.CanonicalName,
		NativeName:          s.NativeName,
		TraceID:             s.TraceID,
		SpanID:              s.SpanID,
		ParentSpan:          s.ParentSpan,
		ToolName:            s.ToolName,
		ToolUseID:           s.ToolUseID,
		Model:               s.Model,
		Decision:            s.Decision,
		DecisionSource:      s.DecisionSource,
		TokensInput:         s.Tokens.Input,
		TokensOutput:        s.Tokens.Output,
		TokensCacheRead:     s.Tokens.CacheRead,
		TokensCacheCreation: s.Tokens.CacheCreation,
		TokensThought:       s.Tokens.Thought,
		TokensTool:          s.Tokens.Tool,
		TokensReasoning:     s.Tokens.Reasoning,
		CostUSD:             s.CostUSD,
		CostSource:          string(s.CostSource),
		DurationMs:          s.DurationMs,
		Success:             s.Success,
		ErrorMsg:            s.ErrorMsg,
		Attempt:             s.Attempt,
		StatusCode:          s.StatusCode,
		ResourceAttrs:       resourceAttrs,
		Attrs:               s.RawAttrs,
	}
	return json.Marshal(line)
}
