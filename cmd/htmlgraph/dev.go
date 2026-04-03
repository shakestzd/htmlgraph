package main

import (
	"github.com/spf13/cobra"
)

// devCmd is a shortcut for "htmlgraph claude --dev".
func devCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dev",
		Short: "Launch Claude Code in dev mode (shortcut for 'claude --dev')",
		Long:  "Launch Claude Code with the HtmlGraph Go plugin in dev mode.\nEquivalent to running: htmlgraph claude --dev",
		RunE: func(cmd *cobra.Command, args []string) error {
			return launchClaudeDev(args, false)
		},
	}
}
