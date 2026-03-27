package hooks

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/models"
)

// UserPrompt handles the UserPromptSubmit Claude Code hook event.
// It inserts a UserQuery agent_event, classifies the prompt intent,
// and returns combined CIGS attribution + classification guidance.
func UserPrompt(event *CloudEvent, database *sql.DB) (*HookResult, error) {
	sessionID := EnvSessionID(event.SessionID)
	if sessionID == "" || event.Prompt == "" {
		return &HookResult{Continue: true}, nil
	}

	featureID := GetActiveFeatureID(database, sessionID)

	promptSummary := sanitizePrompt(event.Prompt)
	if promptSummary == "" {
		return &HookResult{Continue: true}, nil
	}
	if len(promptSummary) > 300 {
		promptSummary = promptSummary[:300] + "…"
	}

	// Dedup: skip if identical UserQuery was recorded in last 5 seconds
	var recentCount int
	_ = database.QueryRow(
		`SELECT COUNT(*) FROM agent_events WHERE session_id = ? AND tool_name = 'UserQuery' AND input_summary = ? AND timestamp > datetime('now', '-5 seconds')`,
		sessionID, promptSummary,
	).Scan(&recentCount)
	if recentCount > 0 {
		return &HookResult{Continue: true}, nil
	}

	ev := &models.AgentEvent{
		EventID:      uuid.New().String(),
		AgentID:      agentIDFromEnv(),
		EventType:    models.EventToolCall,
		Timestamp:    time.Now().UTC(),
		ToolName:     "UserQuery",
		InputSummary: promptSummary,
		SessionID:    sessionID,
		FeatureID:    featureID,
		Status:       "recorded",
		Source:       "hook",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	_ = db.InsertEvent(database, ev) // Non-fatal

	// Update session last_user_query fields.
	updateLastQuery(database, sessionID, event.Prompt)

	// Classify the prompt intent for CIGS guidance.
	intent := ClassifyPrompt(event.Prompt)

	// Look up active work item type for intent-specific directives.
	activeWorkType := getActiveWorkItemType(database, featureID)

	// Build attribution block (open work items listing).
	attributionBlock := buildAttributionGuidance(database, sessionID, featureID)

	// Combine classification guidance with attribution.
	guidance := GenerateGuidance(intent, featureID, activeWorkType, attributionBlock)

	result := &HookResult{}
	if guidance != "" {
		result.AdditionalContext = guidance
	} else {
		result.Continue = true
	}
	return result, nil
}

// updateLastQuery refreshes last_user_query_at and last_user_query on the session.
func updateLastQuery(database *sql.DB, sessionID, prompt string) {
	summary := prompt
	if len(summary) > 200 {
		summary = summary[:200] + "…"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = database.Exec(`
		UPDATE sessions
		SET last_user_query_at = ?,
		    last_user_query = ?
		WHERE session_id = ?`,
		now, summary, sessionID,
	)
}

// buildAttributionGuidance returns a compact CIGS attribution block listing
// open work items so Claude can call sdk.features.start() for the right item.
func buildAttributionGuidance(database *sql.DB, sessionID, activeFeatureID string) string {
	open := listOpenWorkItems(database)
	if len(open) == 0 {
		return ""
	}

	lines := []string{
		"## Work Item Attribution (CIGS)",
		"",
		"**ACTIVE**: " + activeFeatureOrNone(activeFeatureID),
		"",
		"**Open work items** — call `sdk.features.start(\"id\")` for the item matching this task:",
	}
	for _, item := range open {
		marker := "  "
		if item.id == activeFeatureID {
			marker = "* "
		}
		lines = append(lines, fmt.Sprintf("%s`%s` — %s [%s]", marker, item.id, item.title, item.status))
	}
	return joinLines(lines)
}

type workItemRow struct {
	id     string
	title  string
	status string
	itype  string
}

// listOpenWorkItems returns in-progress and todo features/bugs/spikes.
func listOpenWorkItems(database *sql.DB) []workItemRow {
	rows, err := database.Query(`
		SELECT id, title, status, type
		FROM features
		WHERE status IN ('in-progress', 'todo', 'active')
		ORDER BY
			CASE status WHEN 'in-progress' THEN 0 ELSE 1 END,
			CASE type WHEN 'feature' THEN 0 WHEN 'bug' THEN 1 ELSE 2 END,
			created_at DESC
		LIMIT 10`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var items []workItemRow
	for rows.Next() {
		var r workItemRow
		if err := rows.Scan(&r.id, &r.title, &r.status, &r.itype); err == nil {
			items = append(items, r)
		}
	}
	return items
}

// getActiveWorkItemType returns the type ("feature", "bug", "spike") of the
// active work item, or "" if no active item or lookup fails.
func getActiveWorkItemType(database *sql.DB, featureID string) string {
	if featureID == "" {
		return ""
	}
	var itemType sql.NullString
	_ = database.QueryRow(
		`SELECT type FROM features WHERE id = ?`, featureID,
	).Scan(&itemType)
	return itemType.String
}

func activeFeatureOrNone(id string) string {
	if id == "" {
		return "none"
	}
	return id
}

// sanitizePrompt strips XML notification/reminder blocks from prompt text.
func sanitizePrompt(s string) string {
	for _, tag := range []string{"task-notification", "system-reminder", "command-message", "local-command-caveat"} {
		open := "<" + tag + ">"
		close := "</" + tag + ">"
		for {
			i := strings.Index(s, open)
			if i == -1 {
				break
			}
			j := strings.Index(s[i:], close)
			if j == -1 {
				s = s[:i]
				break
			}
			s = s[:i] + s[i+j+len(close):]
		}
	}
	// Strip lines that are just notification artifacts
	var cleaned []string
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "Full transcript available at:") {
			continue
		}
		if strings.HasPrefix(trimmed, "Read the output file to retrieve") {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func joinLines(lines []string) string {
	result := ""
	for i, l := range lines {
		result += l
		if i < len(lines)-1 {
			result += "\n"
		}
	}
	return result
}
