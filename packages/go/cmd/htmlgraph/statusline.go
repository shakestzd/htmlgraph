package main

import (
	"fmt"

	"github.com/shakestzd/htmlgraph/internal/workitem"
	"github.com/spf13/cobra"
)

func statuslineCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "statusline",
		Short: "Print the active work item for Claude Code status line",
		RunE:  runStatusline,
	}
}

func runStatusline(_ *cobra.Command, _ []string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return nil
	}

	p, err := workitem.Open(dir, "claude-code")
	if err != nil {
		return nil
	}
	defer p.Close()

	// Check bugs first (higher priority), then features
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
