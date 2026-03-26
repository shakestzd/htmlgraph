package hooks

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/models"
)

// UserPrompt handles the UserPromptSubmit Claude Code hook event.
// It inserts a UserQuery agent_event and returns CIGS attribution guidance.
func UserPrompt(event *CloudEvent, database *sql.DB) (*HookResult, error) {
	sessionID := EnvSessionID(event.SessionID)
	if sessionID == "" || event.Prompt == "" {
		return &HookResult{Continue: true}, nil
	}

	featureID := GetActiveFeatureID(database, sessionID)

	promptSummary := event.Prompt
	if len(promptSummary) > 300 {
		promptSummary = promptSummary[:300] + "…"
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

	if err := db.InsertEvent(database, ev); err != nil {
		fmt.Fprintf(os.Stderr, "htmlgraph user-prompt: db error: %v\n", err)
	}

	// Update session last_user_query fields.
	updateLastQuery(database, sessionID, event.Prompt)

	guidance := buildAttributionGuidance(database, sessionID, featureID)

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

func activeFeatureOrNone(id string) string {
	if id == "" {
		return "none"
	}
	return id
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
