// Register in main.go: rootCmd.AddCommand(sessionCmd())
package main

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"strings"
	"time"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/models"
	"github.com/spf13/cobra"
)

func sessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage development sessions",
	}
	cmd.AddCommand(sessionListCmd())
	cmd.AddCommand(sessionStartCmd())
	cmd.AddCommand(sessionEndCmd())
	return cmd
}

// sessionListCmd lists sessions from the SQLite DB.
func sessionListCmd() *cobra.Command {
	var activeOnly bool
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sessions",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runSessionList(activeOnly, limit)
		},
	}
	cmd.Flags().BoolVar(&activeOnly, "active", false, "Only show active sessions")
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum number of sessions to show")
	return cmd
}

func runSessionList(activeOnly bool, limit int) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	db, err := openDB(dir)
	if err != nil {
		return err
	}
	defer db.Close()

	sessions, err := dbpkg.ListSessions(db, activeOnly, limit)
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	fmt.Printf("%-16s  %-18s  %-10s  %-22s  %s\n",
		"SESSION", "AGENT", "STATUS", "STARTED", "DURATION")
	fmt.Println(strings.Repeat("-", 85))
	for _, s := range sessions {
		printSessionRow(s)
	}
	fmt.Printf("\n%d session(s)\n", len(sessions))
	return nil
}

func printSessionRow(s *models.Session) {
	id := truncate(s.SessionID, 14)
	agent := truncate(s.AgentAssigned, 18)
	started := s.CreatedAt.Local().Format("2006-01-02 15:04:05")
	duration := sessionDuration(s)
	fmt.Printf("%-16s  %-18s  %-10s  %-22s  %s\n",
		id, agent, s.Status, started, duration)
}

func sessionDuration(s *models.Session) string {
	if s.CompletedAt != nil {
		return fmtDuration(s.CompletedAt.Sub(s.CreatedAt))
	}
	if s.Status == "active" {
		return fmtDuration(time.Since(s.CreatedAt))
	}
	return "-"
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	sec := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, sec)
	}
	return fmt.Sprintf("%dm%02ds", m, sec)
}

// sessionStartCmd creates a new session row.
func sessionStartCmd() *cobra.Command {
	var agent string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new session",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runSessionStart(agent)
		},
	}
	cmd.Flags().StringVar(&agent, "agent", "claude-code", "Agent identifier for this session")
	return cmd
}

func runSessionStart(agent string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	db, err := openDB(dir)
	if err != nil {
		return err
	}
	defer db.Close()

	s := &models.Session{
		SessionID:     generateSessionID(),
		AgentAssigned: agent,
		Status:        "active",
		CreatedAt:     time.Now().UTC(),
	}

	if err := dbpkg.InsertSession(db, s); err != nil {
		return fmt.Errorf("start session: %w", err)
	}
	fmt.Printf("Started session: %s\n", s.SessionID)
	return nil
}

// sessionEndCmd ends a session by ID (or the most recent active session).
func sessionEndCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "end [session-id]",
		Short: "End a session",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			id := ""
			if len(args) > 0 {
				id = args[0]
			}
			return runSessionEnd(id)
		},
	}
}

func runSessionEnd(sessionID string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	db, err := openDB(dir)
	if err != nil {
		return err
	}
	defer db.Close()

	if sessionID == "" {
		sessionID, err = dbpkg.MostRecentActiveSession(db)
		if err != nil {
			return fmt.Errorf("find active session: %w", err)
		}
		if sessionID == "" {
			return fmt.Errorf("no active sessions found\nRun 'htmlgraph session start' to begin tracking, or specify a session ID explicitly.")
		}
	}

	if err := dbpkg.UpdateSessionStatus(db, sessionID, "completed"); err != nil {
		return fmt.Errorf("end session: %w", err)
	}
	fmt.Printf("Ended session: %s\n", sessionID)
	return nil
}

// openDB is a shared helper to open the SQLite DB from the .htmlgraph dir.
func openDB(htmlgraphDir string) (*sql.DB, error) {
	db, err := dbpkg.Open(htmlgraphDir + "/htmlgraph.db")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return db, nil
}

// generateSessionID produces a collision-resistant session ID using crypto/rand.
// Format: sess-{hex8} matching Python/SDK convention.
func generateSessionID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("sess-%x", b)
}
