package main

import (
	"fmt"
	"path/filepath"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/hooks"
	"github.com/shakestzd/htmlgraph/internal/workitem"
	"github.com/spf13/cobra"
)

func statuslineCmd() *cobra.Command {
	var sessionID string

	cmd := &cobra.Command{
		Use:   "statusline",
		Short: "Print the active work item for Claude Code status line",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatusline(sessionID)
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "Session ID to scope the active work item lookup")
	return cmd
}

func runStatusline(sessionID string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return nil
	}

	// If a session ID is provided, look up the session's active_feature_id from SQLite.
	if sessionID != "" {
		return statuslineFromSession(dir, sessionID)
	}

	// Fallback: scan HTML files for any in-progress item.
	return statuslineFromHTML(dir)
}

func statuslineFromSession(dir, sessionID string) error {
	database, err := dbpkg.Open(filepath.Join(dir, "htmlgraph.db"))
	if err != nil {
		return nil
	}
	defer database.Close()

	featureID := hooks.GetActiveFeatureID(database, sessionID)
	if featureID == "" {
		return nil
	}

	// Look up the title from the HTML file.
	p, err := workitem.Open(dir, "claude-code")
	if err != nil {
		// We have the ID but can't get title — still show the ID.
		fmt.Println(featureID)
		return nil
	}
	defer p.Close()

	for _, typeName := range []string{"bug", "feature", "spike"} {
		col := collectionFor(p, typeName)
		node, err := col.Get(featureID)
		if err == nil && node != nil {
			fmt.Printf("%s: %s\n", node.ID, truncate(node.Title, 25))
			return nil
		}
	}

	// ID exists in DB but no matching HTML — show ID alone.
	fmt.Println(featureID)
	return nil
}

func statuslineFromHTML(dir string) error {
	p, err := workitem.Open(dir, "claude-code")
	if err != nil {
		return nil
	}
	defer p.Close()

	for _, typeName := range []string{"bug", "feature"} {
		col := collectionFor(p, typeName)
		nodes, err := col.List()
		if err != nil {
			continue
		}
		for _, n := range nodes {
			if n.Status == "in-progress" {
				fmt.Printf("%s: %s\n", n.ID, truncate(n.Title, 25))
				return nil
			}
		}
	}

	return nil
}
