// Package main is the entry point for the htmlgraph CLI.
//
// Subcommands are scaffolded here and will be fleshed out in later waves.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "htmlgraph",
		Short: "Local-first observability for AI-assisted development",
		Long:  "HtmlGraph — local-first observability and coordination platform for AI-assisted development.",
	}

	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(serveCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("htmlgraph %s (go)\n", version)
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show work item status summary",
		RunE: func(_ *cobra.Command, _ []string) error {
			// Placeholder: will be fleshed out in Wave 3 (feat-35d26c62).
			fmt.Println("htmlgraph status — not yet implemented (see feat-35d26c62)")
			return nil
		},
	}
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the dashboard server",
		RunE: func(_ *cobra.Command, _ []string) error {
			// Placeholder: will be fleshed out in Wave 5.
			fmt.Println("htmlgraph serve — not yet implemented")
			return nil
		},
	}
}
