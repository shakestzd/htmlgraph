package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
)

// generateTitle calls `claude -p --model haiku` once to produce a short title.
// Sets title to the result on success, or "--" on failure, so it never retries.
func generateTitle(database *sql.DB, sessionID string) {
	// Skip if title already set (any non-empty value).
	var existing sql.NullString
	database.QueryRow(`SELECT title FROM sessions WHERE session_id = ?`, sessionID).Scan(&existing)
	if existing.Valid && existing.String != "" {
		return
	}

	// Gather first 3 user messages as context.
	msgs, err := dbpkg.ListMessages(database, sessionID, 20)
	if err != nil || len(msgs) == 0 {
		return
	}

	var userMsgs []string
	for _, m := range msgs {
		if m.Role != "user" {
			continue
		}
		text := m.Content
		if len(text) > 200 {
			text = text[:200]
		}
		userMsgs = append(userMsgs, text)
		if len(userMsgs) >= 3 {
			break
		}
	}
	if len(userMsgs) == 0 {
		return
	}

	prompt := fmt.Sprintf(
		`[htmlgraph-titler] Generate a concise 4-8 word title for this AI coding session. Return ONLY the title, no quotes, no punctuation at the end.

User messages:
%s`, strings.Join(userMsgs, "\n---\n"))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude", "-p", "--model", "haiku", prompt)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		log.Printf("titler: failed for %s: %v (will not retry)", truncate(sessionID, 12), err)
		// Mark as attempted so we never retry.
		database.Exec(`UPDATE sessions SET title = '--' WHERE session_id = ?`, sessionID)
		return
	}

	title := strings.TrimSpace(out.String())
	if title == "" || len(title) > 100 {
		database.Exec(`UPDATE sessions SET title = '--' WHERE session_id = ?`, sessionID)
		return
	}

	database.Exec(`UPDATE sessions SET title = ? WHERE session_id = ?`, title, sessionID)
	log.Printf("titler: %s → %q", truncate(sessionID, 12), title)
}
