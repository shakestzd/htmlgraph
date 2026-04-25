package main

import (
	"fmt"
	"path/filepath"

	"github.com/shakestzd/htmlgraph/internal/otel/retention"
	"github.com/spf13/cobra"
)

// sessionRestoreCmd returns a cobra.Command that extracts an archived session
// from .htmlgraph/archive/ back into .htmlgraph/sessions/ so the indexer can
// pick it up on next replay.
func sessionRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore <session-id>",
		Short: "Restore an archived session for re-indexing",
		Long: `Extracts a previously-archived session (.htmlgraph/archive/<yyyy-mm>/<sid>.tar.gz)
back into .htmlgraph/sessions/<sid>/ so the NDJSON indexer picks it up on
its next replay cycle. The session must have been archived by the retention
job (htmlgraph serve runs this automatically every 24h).`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runSessionRestore(args[0])
		},
	}
}

func runSessionRestore(sessionID string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	htmlgraphDir := filepath.Clean(dir)

	if err := retention.ExtractArchive(htmlgraphDir, sessionID); err != nil {
		return fmt.Errorf("restore session %s: %w", sessionID, err)
	}

	fmt.Printf("Restored session %s to .htmlgraph/sessions/%s/\n", sessionID, sessionID)
	fmt.Println("The indexer will pick up events.ndjson on its next replay cycle.")
	return nil
}
