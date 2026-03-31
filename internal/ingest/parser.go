// Package ingest reads Claude Code JSONL session files and extracts
// structured messages and tool calls for storage in HtmlGraph's database.
package ingest

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/shakestzd/htmlgraph/internal/models"
	"github.com/tidwall/gjson"
)

// ParseResult holds the structured output of parsing a JSONL session file.
type ParseResult struct {
	SessionID string
	Messages  []models.Message
	ToolCalls []models.ToolCall
	Title     string
	Model     string // most-used model
	FileSize  int64
}

// ParseFile reads an entire Claude Code JSONL session file and returns
// structured messages and tool calls. It filters out system meta messages,
// compact summaries, and file-history snapshots.
func ParseFile(path string) (*ParseResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	info, _ := f.Stat()
	result, err := parse(f)
	if err != nil {
		return nil, err
	}
	if info != nil {
		result.FileSize = info.Size()
	}
	return result, nil
}

// ParseFileFrom reads only the bytes starting at offset, for incremental sync.
func ParseFileFrom(path string, offset int64) (*ParseResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to %d: %w", offset, err)
	}

	info, _ := f.Stat()
	result, err := parse(f)
	if err != nil {
		return nil, err
	}
	if info != nil {
		result.FileSize = info.Size()
	}
	return result, nil
}

func parse(r io.Reader) (*ParseResult, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10 MB max line

	result := &ParseResult{}
	ordinal := 0
	modelCounts := map[string]int{}

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		lineType := gjson.Get(line, "type").String()
		sessionID := gjson.Get(line, "sessionId").String()
		if result.SessionID == "" && sessionID != "" {
			result.SessionID = sessionID
		}

		switch lineType {
		case "custom-title":
			result.Title = gjson.Get(line, "customTitle").String()
			continue
		case "file-history-snapshot", "queue-operation", "system":
			continue
		case "user":
			if gjson.Get(line, "isMeta").Bool() || gjson.Get(line, "isCompactSummary").Bool() {
				continue
			}
			msg := parseUserMessage(line, ordinal)
			if msg == nil {
				continue
			}
			msg.SessionID = result.SessionID
			result.Messages = append(result.Messages, *msg)
			ordinal++

		case "assistant":
			msg, tools := parseAssistantMessage(line, ordinal)
			if msg == nil {
				continue
			}
			msg.SessionID = result.SessionID
			result.Messages = append(result.Messages, *msg)

			for i := range tools {
				tools[i].SessionID = result.SessionID
				tools[i].MessageOrdinal = ordinal
			}
			result.ToolCalls = append(result.ToolCalls, tools...)

			if msg.Model != "" {
				modelCounts[msg.Model]++
			}
			ordinal++
		}
	}

	// Determine most-used model.
	maxCount := 0
	for m, c := range modelCounts {
		if c > maxCount {
			result.Model = m
			maxCount = c
		}
	}

	return result, scanner.Err()
}

func parseUserMessage(line string, ordinal int) *models.Message {
	content := extractContent(line)
	if content == "" {
		return nil
	}
	// Filter system messages embedded as user turns.
	if isSystemMessage(content) {
		return nil
	}

	ts := parseTimestamp(line)
	uuid := gjson.Get(line, "uuid").String()
	parentUUID := gjson.Get(line, "parentUuid").String()

	// Check if this is a tool_result (user turn carrying tool output).
	contentArr := gjson.Get(line, "message.content")
	hasToolResult := false
	if contentArr.IsArray() {
		contentArr.ForEach(func(_, v gjson.Result) bool {
			if v.Get("type").String() == "tool_result" {
				hasToolResult = true
				return false
			}
			return true
		})
	}
	// Skip pure tool_result turns — they're captured via ToolCall.ResultContentLength.
	if hasToolResult {
		return nil
	}

	return &models.Message{
		Ordinal:       ordinal,
		Role:          "user",
		Content:       content,
		Timestamp:     ts,
		ContentLength: len(content),
		UUID:          uuid,
		ParentUUID:    parentUUID,
	}
}

func parseAssistantMessage(line string, ordinal int) (*models.Message, []models.ToolCall) {
	ts := parseTimestamp(line)
	uuid := gjson.Get(line, "uuid").String()
	parentUUID := gjson.Get(line, "parentUuid").String()
	model := gjson.Get(line, "message.model").String()
	stopReason := gjson.Get(line, "message.stop_reason").String()

	// Token usage.
	usage := gjson.Get(line, "message.usage")
	inputTokens := int(usage.Get("input_tokens").Int())
	outputTokens := int(usage.Get("output_tokens").Int())
	cacheRead := int(usage.Get("cache_read_input_tokens").Int())

	// Walk content blocks.
	var textParts []string
	var tools []models.ToolCall
	hasThinking := false
	hasToolUse := false

	contentArr := gjson.Get(line, "message.content")
	if contentArr.IsArray() {
		contentArr.ForEach(func(_, block gjson.Result) bool {
			blockType := block.Get("type").String()
			switch blockType {
			case "text":
				textParts = append(textParts, block.Get("text").String())
			case "thinking":
				hasThinking = true
			case "tool_use":
				hasToolUse = true
				toolName := block.Get("name").String()
				tools = append(tools, models.ToolCall{
					ToolName:  toolName,
					Category:  models.ToolCategory(toolName),
					ToolUseID: block.Get("id").String(),
					InputJSON: block.Get("input").Raw,
				})
			}
			return true
		})
	}

	content := strings.Join(textParts, "\n")

	msg := &models.Message{
		Ordinal:         ordinal,
		Role:            "assistant",
		Content:         content,
		Timestamp:       ts,
		HasThinking:     hasThinking,
		HasToolUse:      hasToolUse,
		ContentLength:   len(content),
		Model:           model,
		InputTokens:     inputTokens,
		OutputTokens:    outputTokens,
		CacheReadTokens: cacheRead,
		StopReason:      stopReason,
		UUID:            uuid,
		ParentUUID:      parentUUID,
	}
	return msg, tools
}

func extractContent(line string) string {
	msg := gjson.Get(line, "message.content")
	if msg.Type == gjson.String {
		return msg.String()
	}
	if msg.IsArray() {
		var parts []string
		msg.ForEach(func(_, v gjson.Result) bool {
			if v.Get("type").String() == "text" {
				parts = append(parts, v.Get("text").String())
			}
			return true
		})
		return strings.Join(parts, "\n")
	}
	return ""
}

func parseTimestamp(line string) time.Time {
	ts := gjson.Get(line, "timestamp").String()
	if ts == "" {
		ts = gjson.Get(line, "snapshot.timestamp").String()
	}
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t, _ = time.Parse("2006-01-02T15:04:05.000Z", ts)
	}
	return t
}

func isSystemMessage(content string) bool {
	prefixes := []string{
		"This session is being continued",
		"[Request interrupted",
		"[Session resumed",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(content, p) {
			return true
		}
	}
	return false
}
