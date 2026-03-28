package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/ingest"
	"github.com/shakestzd/htmlgraph/internal/paths"
	"github.com/spf13/cobra"
)

func ingestCmd() *cobra.Command {
	var (
		sessionID string
		project   string
		all       bool
		force     bool
	)

	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Ingest Claude Code session transcripts from JSONL files",
		Long: `Reads Claude Code session JSONL files from ~/.claude/projects/ and
stores structured messages and tool calls in the HtmlGraph database.

By default, discovers sessions for the current project. Use --all to
ingest all projects, or --session to target a specific session.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runIngest(sessionID, project, all, force)
		},
	}

	cmd.Flags().StringVar(&sessionID, "session", "", "ingest a specific session ID")
	cmd.Flags().StringVar(&project, "project", "", "filter by project name (substring match)")
	cmd.Flags().BoolVar(&all, "all", false, "ingest all discovered sessions")
	cmd.Flags().BoolVar(&force, "force", false, "re-ingest even if already synced")

	return cmd
}

func runIngest(sessionID, project string, all, force bool) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	database, err := dbpkg.Open(filepath.Join(htmlgraphDir, "htmlgraph.db"))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	// Single session mode: find the file by scanning all projects.
	if sessionID != "" {
		return ingestBySessionID(database, sessionID, force)
	}

	// Resolve the git remote URL for the current project to use as a filter.
	// When --all is set or --project is explicitly provided, skip the remote filter.
	var gitRemote string
	if project == "" && !all {
		gitRemote = paths.GetGitRemoteURL(filepath.Dir(htmlgraphDir))
	}

	files, err := ingest.DiscoverSessions(project)
	if err != nil {
		return fmt.Errorf("discover sessions: %w", err)
	}

	// Apply git remote filter when we have a resolved remote and no explicit
	// project name or --all flag was provided.
	if gitRemote != "" {
		files = ingest.FilterByGitRemote(files, gitRemote)
	}

	if len(files) == 0 {
		fmt.Println("No session files found.")
		return nil
	}

	fmt.Printf("Found %d session files", len(files))
	switch {
	case gitRemote != "":
		fmt.Printf(" (git remote filter: %q)", gitRemote)
	case project != "":
		fmt.Printf(" (project filter: %q)", project)
	}
	fmt.Println()

	var ingested, skipped, errCount int
	for _, sf := range files {
		if !force {
			count, _ := dbpkg.CountMessages(database, sf.SessionID)
			if count > 0 {
				skipped++
				continue
			}
		}

		n, toolN, err := ingestFile(database, sf, force)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  skip %s: %v\n", truncate(sf.SessionID, 12), err)
			errCount++
			continue
		}
		if n == 0 {
			skipped++
			continue
		}

		fmt.Printf("  %-14s %3d msgs  %3d tools  (%s)\n",
			truncate(sf.SessionID, 14), n, toolN, sf.Project)
		ingested++
	}

	fmt.Printf("\nDone: %d ingested, %d skipped, %d errors\n", ingested, skipped, errCount)
	return nil
}

func ingestBySessionID(database *sql.DB, sessionID string, force bool) error {
	files, err := ingest.DiscoverSessions("")
	if err != nil {
		return err
	}
	for _, sf := range files {
		if sf.SessionID == sessionID {
			n, toolN, err := ingestFile(database, sf, force)
			if err != nil {
				return err
			}
			fmt.Printf("Ingested %d messages, %d tool calls from %s\n", n, toolN, sf.Path)
			return nil
		}
	}
	return fmt.Errorf("session %s not found in ~/.claude/projects/", sessionID)
}

func ingestFile(database *sql.DB, sf ingest.SessionFile, force bool) (int, int, error) {
	result, err := ingest.ParseFile(sf.Path)
	if err != nil {
		return 0, 0, err
	}
	if len(result.Messages) == 0 {
		return 0, 0, nil
	}

	if force {
		_ = dbpkg.DeleteSessionMessages(database, sf.SessionID)
	}

	ensureSession(database, sf.SessionID, result)
	msgCount, toolCount := storeParseResult(database, sf.SessionID, result)
	_ = dbpkg.UpdateTranscriptSync(database, sf.SessionID, sf.Path)

	return msgCount, toolCount, nil
}

func storeParseResult(database *sql.DB, sessionID string, result *ingest.ParseResult) (int, int) {
	var msgCount, toolCount int

	// Map ordinal → message DB ID for linking tool calls.
	msgIDs := map[int]int64{}

	for _, m := range result.Messages {
		m.SessionID = sessionID
		id, err := dbpkg.InsertMessage(database, &m)
		if err != nil {
			fmt.Fprintf(os.Stderr, "    warn: msg ord %d: %v\n", m.Ordinal, err)
			continue
		}
		msgIDs[m.Ordinal] = id
		msgCount++
	}

	// Fetch the session's active_feature_id to tag each tool call.
	var activeFeatureID string
	database.QueryRow(
		`SELECT COALESCE(active_feature_id, '') FROM sessions WHERE session_id = ?`,
		sessionID,
	).Scan(&activeFeatureID)

	for _, tc := range result.ToolCalls {
		tc.SessionID = sessionID
		if mid, ok := msgIDs[tc.MessageOrdinal]; ok {
			tc.MessageID = int(mid)
		}
		if activeFeatureID != "" {
			tc.FeatureID = activeFeatureID
		}
		if err := dbpkg.InsertToolCall(database, &tc); err != nil {
			fmt.Fprintf(os.Stderr, "    warn: tool %s: %v\n", tc.ToolName, err)
			continue
		}
		toolCount++
	}

	// Update session model if we detected one.
	if result.Model != "" {
		database.Exec(`UPDATE sessions SET model = ? WHERE session_id = ? AND (model IS NULL OR model = '')`,
			result.Model, sessionID)
	}

	return msgCount, toolCount
}

// ensureSession creates a session row if one doesn't already exist.
// This handles sessions discovered from JSONL that predate hook installation.
func ensureSession(database *sql.DB, sessionID string, result *ingest.ParseResult) {
	var exists int
	database.QueryRow(`SELECT COUNT(*) FROM sessions WHERE session_id = ?`, sessionID).Scan(&exists)
	if exists > 0 {
		return
	}

	// Create a minimal session from transcript metadata.
	ts := ""
	if len(result.Messages) > 0 {
		ts = result.Messages[0].Timestamp.UTC().Format("2006-01-02T15:04:05Z")
	}

	database.Exec(`
		INSERT INTO sessions (session_id, agent_assigned, created_at, status, model)
		VALUES (?, 'claude-code', COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP), 'completed', ?)`,
		sessionID, ts, nullStrVal(result.Model),
	)
}

func nullStrVal(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
