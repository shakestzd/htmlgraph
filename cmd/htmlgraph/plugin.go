package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func pluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage the HtmlGraph Claude Code plugin",
	}
	cmd.AddCommand(pluginInstallCmd())
	return cmd
}

func pluginInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install the HtmlGraph plugin for Claude Code",
		Long:  "Installs the HtmlGraph plugin into Claude Code via 'claude plugin install htmlgraph'.",
		RunE: func(cmd *cobra.Command, args []string) error {
			claudePath, err := exec.LookPath("claude")
			if err != nil {
				return fmt.Errorf("'claude' not found on PATH: install Claude Code first (https://claude.ai/code)")
			}

			c := exec.Command(claudePath, "plugin", "install", "htmlgraph")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Run(); err != nil {
				return fmt.Errorf("claude plugin install htmlgraph: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Plugin installed. Run 'htmlgraph claude --dev' to test.")
			return nil
		},
	}
}
