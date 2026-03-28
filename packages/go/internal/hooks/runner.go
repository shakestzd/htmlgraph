// Package hooks implements Claude Code hook handlers for HtmlGraph.
//
// Each handler reads a CloudEvent JSON payload from stdin and writes a
// HookResult JSON to stdout. The Go binary replaces the Python hook scripts,
// eliminating the ~500ms uv cold-start per invocation.
package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

// CloudEvent is the JSON payload Claude Code sends to every hook via stdin.
// Only the fields HtmlGraph actually uses are decoded; the rest are ignored.
type CloudEvent struct {
	// Top-level fields common to all hook types
	SessionID string `json:"session_id"`
	CWD       string `json:"cwd"`

	// UserPromptSubmit
	Prompt string `json:"prompt"`

	// PreToolUse / PostToolUse
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`
	ToolUseID string         `json:"tool_use_id"`

	// PostToolUse result
	ToolResult map[string]any `json:"tool_result"`

	// SubagentStart / SubagentStop
	AgentID   string `json:"agent_id"`
	AgentType string `json:"agent_type"`

	// Stop / SubagentStop
	StopReason           string `json:"stop_reason"`
	LastAssistantMessage string `json:"last_assistant_message"`
}

// HookResult is the JSON written to stdout to control Claude Code behaviour.
// Fields are omitted when empty to keep the payload minimal.
type HookResult struct {
	Continue          bool   `json:"continue,omitempty"`
	Decision          string `json:"decision,omitempty"`           // "allow" | "deny" | "block"
	Reason            string `json:"reason,omitempty"`
	Message           string `json:"message,omitempty"`            // shown on stderr
	AdditionalContext string `json:"additionalContext,omitempty"`  // injected into conversation
}

// ReadInput reads and parses a CloudEvent from stdin.
func ReadInput() (*CloudEvent, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	if len(data) == 0 {
		return &CloudEvent{}, nil
	}
	var ev CloudEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return nil, fmt.Errorf("parsing CloudEvent: %w", err)
	}
	return &ev, nil
}

// WriteResult encodes result as JSON to stdout.
func WriteResult(result *HookResult) error {
	return json.NewEncoder(os.Stdout).Encode(result)
}

// Allow writes a simple allow decision and returns.
func Allow() error {
	return WriteResult(&HookResult{Decision: "allow"})
}

// Continue writes a continue:true response (used by non-blocking hooks).
func Continue() error {
	return WriteResult(&HookResult{Continue: true})
}

// Empty writes an empty JSON object (hook has no opinion).
func Empty() error {
	_, err := fmt.Fprintln(os.Stdout, "{}")
	return err
}

// ResolveProjectDir finds the project directory containing .htmlgraph/.
// Mirrors Python bootstrap.resolve_project_dir().
func ResolveProjectDir(cwd string) string {
	// 1. Explicit env var (set by session-start for downstream hooks)
	if d := os.Getenv("CLAUDE_PROJECT_DIR"); d != "" {
		return d
	}
	// 2. CWD from the CloudEvent
	if cwd != "" {
		if _, err := os.Stat(filepath.Join(cwd, ".htmlgraph")); err == nil {
			return cwd
		}
	}
	// 3. Process working directory
	if wd, err := os.Getwd(); err == nil {
		if _, err := os.Stat(filepath.Join(wd, ".htmlgraph")); err == nil {
			return wd
		}
	}
	// 4. Walk up from cwd
	if cwd != "" {
		dir := cwd
		for i := 0; i < 10; i++ {
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
			if _, err := os.Stat(filepath.Join(dir, ".htmlgraph")); err == nil {
				return dir
			}
		}
	}
	if cwd != "" {
		return cwd
	}
	wd, _ := os.Getwd()
	return wd
}

// IsHtmlGraphProject returns true when the project directory has a .htmlgraph/ dir.
func IsHtmlGraphProject(projectDir string) bool {
	_, err := os.Stat(filepath.Join(projectDir, ".htmlgraph"))
	return err == nil
}

// DBPath returns the canonical SQLite path for the given project directory.
func DBPath(projectDir string) string {
	return filepath.Join(projectDir, ".htmlgraph", "htmlgraph.db")
}

// NormaliseSessionID extracts a UUID from a path-style session_id that Claude
// Code sometimes provides for subagent sessions, e.g.:
//
//	/private/tmp/claude-501/-Users-shakes-.../550e8400-e29b-41d4-a716-446655440000
//
// If no UUID is found the original string is returned unchanged.
func NormaliseSessionID(raw string) string {
	if raw == "" || !containsSlash(raw) {
		return raw
	}
	re := regexp.MustCompile(
		`([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})`,
	)
	if m := re.FindString(raw); m != "" {
		return m
	}
	return raw
}

func containsSlash(s string) bool {
	for _, c := range s {
		if c == '/' {
			return true
		}
	}
	return false
}

// EnvSessionID returns the current session ID, preferring the env var set by
// session-start over whatever Claude Code placed in the CloudEvent.
func EnvSessionID(eventSessionID string) string {
	if v := os.Getenv("HTMLGRAPH_SESSION_ID"); v != "" {
		return v
	}
	return NormaliseSessionID(eventSessionID)
}
