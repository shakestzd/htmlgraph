package hooks

import (
	"bufio"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// transcriptRecord is the minimal shape of one JSONL line in the Claude Code
// transcript file. Only the fields we need are decoded; unknown fields ignored.
type transcriptRecord struct {
	Type      string `json:"type"`
	UUID      string `json:"uuid"`
	ParentUUID string `json:"parentUuid"`
	SessionID string `json:"sessionId"`
	RequestID string `json:"requestId"`
	Timestamp string `json:"timestamp"`
	IsSidechain bool  `json:"isSidechain"`
	Message   struct {
		Role       string `json:"role"`
		StopReason string `json:"stop_reason"`
		Content    []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
}

// extractAssistantText returns the concatenated text from an assistant record's
// content array. Returns empty string when there are no text blocks.
func extractAssistantText(rec *transcriptRecord) string {
	if rec == nil {
		return ""
	}
	var sb strings.Builder
	for _, c := range rec.Message.Content {
		if c.Type == "text" {
			sb.WriteString(c.Text)
		}
	}
	return sb.String()
}

// readLastAssistantRecord scans the transcript JSONL from the END and returns
// the most recent non-sidechain assistant record that has at least one non-empty
// text block. Returns nil when none is found (missing file, sidechain-only,
// thinking-only, or malformed JSONL). Reads line-by-line — never loads the
// whole file into memory at once, but does need to buffer all lines to scan
// in reverse order.
func readLastAssistantRecord(transcriptPath string) (*transcriptRecord, error) {
	f, err := os.Open(transcriptPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // missing file is not an error
		}
		return nil, fmt.Errorf("open transcript: %w", err)
	}
	defer f.Close()

	// Collect all lines in memory so we can walk backwards.
	// Transcripts rarely exceed a few MB so this is safe.
	var lines []string
	scanner := bufio.NewScanner(f)
	// Increase buffer for very long lines (large prompts in transcript).
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan transcript: %w", err)
	}

	// Walk backwards to find the most recent qualifying record.
	for i := len(lines) - 1; i >= 0; i-- {
		var rec transcriptRecord
		if err := json.Unmarshal([]byte(lines[i]), &rec); err != nil {
			continue // malformed line — skip
		}
		if rec.Type != "assistant" {
			continue
		}
		if rec.IsSidechain {
			continue
		}
		if rec.Message.Role != "assistant" {
			continue
		}
		if extractAssistantText(&rec) == "" {
			continue // thinking-only or empty content
		}
		return &rec, nil
	}
	return nil, nil
}

// assistantTextSignalID returns a deterministic signal_id for an assistant_text
// signal keyed on the record's UUID. Using the UUID ensures idempotency on Stop
// hook retries while remaining unique per assistant turn.
func assistantTextSignalID(uuid string) string {
	h := sha256.New()
	h.Write([]byte("assistant_text:" + uuid))
	return fmt.Sprintf("%x", h.Sum(nil))[:32]
}

// insertAssistantTextSignal writes an assistant_text otel_signals row derived
// from the last assistant record in the transcript file. It is called by the
// Stop hook handler. Non-fatal: errors are logged to debug.log only.
//
// Schema contract:
//
//	kind          = 'log'
//	canonical     = 'assistant_text'
//	span_id       = transcript record's UUID (assistant turn identity)
//	parent_span   = transcript record's parentUuid (links to user prompt UUID)
//	attrs_json    = {"text": "...", "stop_reason": "...", "request_id": "...", "sidechain": false}
func insertAssistantTextSignal(
	database *sql.DB,
	projectDir string,
	sessionID string,
	transcriptPath string,
) {
	if transcriptPath == "" {
		debugLog(projectDir, "[assistant-text] no transcript_path in Stop payload, skipping")
		return
	}

	rec, err := readLastAssistantRecord(transcriptPath)
	if err != nil {
		debugLog(projectDir, "[assistant-text] read transcript: %v", err)
		return
	}
	if rec == nil {
		// Transcript missing, sidechain-only, or no text turns yet.
		return
	}

	text := extractAssistantText(rec)
	if text == "" {
		return
	}

	// Parse the record timestamp; fall back to now on parse failure.
	var tsMicros int64
	if rec.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339Nano, rec.Timestamp); err == nil {
			tsMicros = t.UnixMicro()
		}
	}
	if tsMicros == 0 {
		tsMicros = time.Now().UnixMicro()
	}

	signalID := assistantTextSignalID(rec.UUID)

	attrsMap := map[string]any{
		"text":        text,
		"stop_reason": rec.Message.StopReason,
		"request_id":  rec.RequestID,
		"sidechain":   false,
	}
	if rec.Message.StopReason != "" && rec.Message.StopReason != "end_turn" {
		attrsMap["interrupted"] = true
	}
	attrsJSON, err := json.Marshal(attrsMap)
	if err != nil {
		debugLog(projectDir, "[assistant-text] marshal attrs: %v", err)
		return
	}

	// Look up active feature for attribution.
	var featureID sql.NullString
	_ = database.QueryRow(
		`SELECT work_item_id FROM active_work_items WHERE session_id = ? AND agent_id = ?`,
		sessionID, "__root__",
	).Scan(&featureID)

	// INSERT OR IGNORE ensures idempotency on hook retries — if the same
	// Stop hook fires twice for the same session, the second insert is a no-op.
	_, dbErr := database.Exec(`
		INSERT OR IGNORE INTO otel_signals (
			signal_id, harness, session_id,
			span_id, parent_span,
			kind, canonical, native, ts_micros,
			attrs_json, feature_id
		) VALUES (?, 'claude', ?, ?, ?, 'log', 'assistant_text', 'assistant_turn', ?, ?, ?)`,
		signalID, sessionID,
		nullableStr(rec.UUID), nullableStr(rec.ParentUUID),
		tsMicros,
		string(attrsJSON),
		featureID,
	)
	if dbErr != nil {
		debugLog(projectDir, "[assistant-text] insert signal: %v", dbErr)
	}
}

